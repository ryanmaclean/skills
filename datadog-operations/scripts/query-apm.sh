#!/bin/bash
set -e

# Source monitoring library
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/datadog-monitoring.sh" 2>/dev/null || true
init_monitoring "$(basename "$0")"
trap 'finalize_monitoring $?' EXIT INT TERM

# Query Datadog APM for performance analysis
# Find slow endpoints, errors, and performance bottlenecks

if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    cat << 'EOF'
Query Datadog APM

Usage:
  query-apm.sh --service SERVICE [--duration DURATION] [--status STATUS] [--limit LIMIT]

Arguments:
  --service SERVICE   Service name to query (required)
  --duration DURATION Time range: 1h, 24h, 7d (default: 1h)
  --status STATUS     Filter by status: error, ok, all (default: all)
  --limit LIMIT       Max endpoints to return (default: 20)

Examples:
  # Query service performance
  query-apm.sh --service payment-api --duration 1h

  # Find slow endpoints
  query-apm.sh --service payment-api --duration 24h --limit 10

  # Get error traces
  query-apm.sh --service payment-api --status error

Environment variables:
  DD_API_KEY   - Datadog API key (required)
  DD_APP_KEY   - Datadog application key (required)
  DD_SITE      - Datadog site (default: datadoghq.com)
EOF
    exit 0
fi

# Check required environment variables
if [ -z "$DD_API_KEY" ] || [ -z "$DD_APP_KEY" ]; then
    echo "[ERROR] DD_API_KEY and DD_APP_KEY must be set" >&2
    exit 1
fi

DD_SITE=${DD_SITE:-datadoghq.com}

# Parse arguments
SERVICE=""
DURATION="1h"
STATUS=""
LIMIT=20

while [[ $# -gt 0 ]]; do
    case $1 in
        --service)
            SERVICE="$2"
            shift 2
            ;;
        --duration)
            DURATION="$2"
            shift 2
            ;;
        --status)
            STATUS="$2"
            shift 2
            ;;
        --limit)
            LIMIT="$2"
            shift 2
            ;;
        *)
            echo "[ERROR] Unknown argument: $1" >&2
            exit 1
            ;;
    esac
done

if [ -z "$SERVICE" ]; then
    echo "[ERROR] --service is required" >&2
    exit 1
fi

# Convert duration to nanoseconds (Datadog APM uses nanoseconds)
case $DURATION in
    1h)
        FROM_NS=$(($(date +%s) - 3600))000000000
        ;;
    24h)
        FROM_NS=$(($(date +%s) - 86400))000000000
        ;;
    7d)
        FROM_NS=$(($(date +%s) - 604800))000000000
        ;;
    *)
        echo "[ERROR] Invalid duration. Use: 1h, 24h, or 7d" >&2
        exit 1
        ;;
esac

TO_NS=$(date +%s)000000000

# Build query
QUERY="service:$SERVICE"
if [ -n "$STATUS" ]; then
    QUERY="$QUERY status:$STATUS"
fi

echo "[INFO] Querying APM traces..." >&2
echo "[INFO] Service: $SERVICE" >&2
echo "[INFO] Duration: $DURATION" >&2
echo "[INFO] Status filter: ${STATUS:-all}" >&2
echo "" >&2

# Query APM API (v2 format with data.attributes wrapper)
start_operation "api_call"
RESPONSE=$(curl -s -X POST "https://api.${DD_SITE}/api/v2/spans/analytics/aggregate" \
  -H "DD-API-KEY: ${DD_API_KEY}" \
  -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
  -H "Content-Type: application/json" \
  -d "{
    \"data\": {
      \"type\": \"aggregate_request\",
      \"attributes\": {
        \"filter\": {
          \"from\": \"now-${DURATION}\",
          \"to\": \"now\",
          \"query\": \"${QUERY}\"
        },
        \"compute\": [
          {\"aggregation\": \"count\"},
          {\"aggregation\": \"avg\", \"metric\": \"@duration\"},
          {\"aggregation\": \"max\", \"metric\": \"@duration\"},
          {\"aggregation\": \"min\", \"metric\": \"@duration\"}
        ],
        \"group_by\": [
          {\"facet\": \"resource_name\", \"limit\": ${LIMIT}}
        ]
      }
    }
  }")

# Check for API errors
if echo "$RESPONSE" | jq -e '.errors' > /dev/null 2>&1; then
    end_operation "error" "error_type:api_error"
    echo "[ERROR] API error:" >&2
    echo "$RESPONSE" | jq '.errors' >&2
    exit 1
fi

end_operation "ok"

# Check if data exists
if ! echo "$RESPONSE" | jq -e '.data' > /dev/null 2>&1 || [ "$(echo "$RESPONSE" | jq '.data | length')" -eq 0 ]; then
    echo "[WARN] No trace data found" >&2
    send_metric "datadog.skill.apm.endpoints.count" 0 "status:no_data" "service:$SERVICE"
    cat <<EOF
{
  "status": "no_data",
  "service": "$SERVICE",
  "duration": "$DURATION",
  "endpoints": []
}
EOF
    exit 0
fi

# Parse results (v2 API format: .data[].attributes.compute and .data[].attributes.by)
# c0=count, c1=avg duration, c2=max duration, c3=min duration (all in nanoseconds)
ENDPOINTS=$(echo "$RESPONSE" | jq -c '.data[] | {
  resource_name: .attributes.by.resource_name,
  request_count: .attributes.compute.c0,
  avg_ms: ((.attributes.compute.c1 // 0) / 1000000 | floor),
  max_ms: ((.attributes.compute.c2 // 0) / 1000000 | floor),
  min_ms: ((.attributes.compute.c3 // 0) / 1000000 | floor)
}' | jq -s '.')

TOTAL_ENDPOINTS=$(echo "$ENDPOINTS" | jq 'length')

echo "[INFO] Found $TOTAL_ENDPOINTS endpoints" >&2
echo "" >&2

# Show top slow endpoints
echo "[SUMMARY] Top 5 slowest endpoints (by avg latency):" >&2
echo "$ENDPOINTS" | jq -r '.[:5] | .[] | "\(.resource_name): avg=\(.avg_ms)ms, max=\(.max_ms)ms, requests=\(.request_count)"' >&2
echo "" >&2

# Calculate statistics
TOTAL_REQUESTS=$(echo "$ENDPOINTS" | jq '[.[].request_count] | add // 0')
AVG_LATENCY=$(echo "$ENDPOINTS" | jq '[.[].avg_ms] | add / length | floor')

echo "[STATS] Total requests: $TOTAL_REQUESTS" >&2
echo "[STATS] Average latency: ${AVG_LATENCY}ms" >&2
echo "" >&2

# Identify problem endpoints (avg > 500ms)
SLOW_ENDPOINTS=$(echo "$ENDPOINTS" | jq '[.[] | select(.avg_ms > 500)] | length')
if [ "$SLOW_ENDPOINTS" -gt 0 ]; then
    echo "[WARN] Found $SLOW_ENDPOINTS endpoints with avg latency > 500ms" >&2
fi

# Send metrics
send_metric "datadog.skill.apm.endpoints.count" "$TOTAL_ENDPOINTS" "status:ok" "service:$SERVICE"
send_metric "datadog.skill.apm.requests.count" "$TOTAL_REQUESTS" "status:ok" "service:$SERVICE"
send_metric "datadog.skill.apm.slow_endpoints.count" "$SLOW_ENDPOINTS" "status:ok" "service:$SERVICE"

# Output structured JSON
cat <<EOF
{
  "status": "ok",
  "service": "$SERVICE",
  "duration": "$DURATION",
  "summary": {
    "total_endpoints": $TOTAL_ENDPOINTS,
    "total_requests": $TOTAL_REQUESTS,
    "avg_latency_ms": $AVG_LATENCY,
    "slow_endpoints_count": $SLOW_ENDPOINTS
  },
  "endpoints": $ENDPOINTS
}
EOF
