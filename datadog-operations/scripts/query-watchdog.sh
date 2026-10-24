#!/bin/bash
set -e

# Source monitoring library
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/datadog-monitoring.sh" 2>/dev/null || true
init_monitoring "$(basename "$0")"
trap 'finalize_monitoring $?' EXIT INT TERM

# Query Datadog Watchdog API for Anomaly Detection
# Analyzes automated anomaly detection for services and resources

# Check required environment variables
if [ -z "$DD_API_KEY" ] || [ -z "$DD_APP_KEY" ]; then
    echo "[ERROR] DD_API_KEY and DD_APP_KEY must be set" >&2
    exit 1
fi

DD_SITE=${DD_SITE:-datadoghq.com}

# Parse arguments
SERVICE=""
DURATION="24h"
TYPE=""

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
        --type)
            TYPE="$2"
            shift 2
            ;;
        *)
            echo "[ERROR] Unknown argument: $1" >&2
            echo "[INFO] Usage: $0 [--service SERVICE] [--duration DURATION] [--type TYPE]" >&2
            echo "[INFO] Types: latency, error_rate, traffic, all" >&2
            exit 1
            ;;
    esac
done

# Convert duration to time expression
case $DURATION in
    1h)
        FROM_TIME="now-1h"
        ;;
    24h)
        FROM_TIME="now-24h"
        ;;
    7d)
        FROM_TIME="now-7d"
        ;;
    30d)
        FROM_TIME="now-30d"
        ;;
    *)
        echo "[ERROR] Invalid duration. Use: 1h, 24h, 7d, or 30d" >&2
        exit 1
        ;;
esac

TO_TIME="now"

# Build Watchdog query
QUERY="source:watchdog"

if [ -n "$SERVICE" ]; then
    QUERY="$QUERY service:$SERVICE"
fi

# Add anomaly type filters
case $TYPE in
    latency)
        QUERY="$QUERY (latency OR p99 OR response_time)"
        ;;
    error_rate)
        QUERY="$QUERY (error OR error_rate OR errors)"
        ;;
    traffic)
        QUERY="$QUERY (hits OR traffic OR request_rate OR throughput)"
        ;;
    all|"")
        # No additional filter - include all anomaly types
        ;;
    *)
        echo "[ERROR] Invalid type. Use: latency, error_rate, traffic, or all" >&2
        exit 1
        ;;
esac

echo "[INFO] Querying Watchdog anomalies..." >&2
echo "[INFO] Service: ${SERVICE:-all}" >&2
echo "[INFO] Duration: $DURATION" >&2
echo "[INFO] Anomaly type: ${TYPE:-all}" >&2
echo "" >&2

# Query Events API v2 for Watchdog events
REQUEST_BODY=$(cat <<EOF
{
  "filter": {
    "query": "${QUERY}",
    "from": "${FROM_TIME}",
    "to": "${TO_TIME}"
  },
  "page": {
    "limit": 100
  },
  "sort": "timestamp"
}
EOF
)

start_operation "api_call"
RESPONSE=$(curl -s -X POST "https://api.${DD_SITE}/api/v2/events/search" \
  -H "DD-API-KEY: ${DD_API_KEY}" \
  -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
  -H "Content-Type: application/json" \
  -d "$REQUEST_BODY")

# Check for API errors
if echo "$RESPONSE" | jq -e '.errors' > /dev/null 2>&1; then
    end_operation "error" "error_type:api_error"
    echo "[ERROR] API error:" >&2
    echo "$RESPONSE" | jq '.errors' >&2
    exit 1
fi

end_operation "ok"

# Parse and analyze anomalies
TOTAL_ANOMALIES=$(echo "$RESPONSE" | jq '.data | length')

echo "[INFO] Found $TOTAL_ANOMALIES Watchdog anomalies" >&2
echo "" >&2

if [ "$TOTAL_ANOMALIES" -eq 0 ]; then
    echo "[INFO] No Watchdog anomalies found for the specified criteria" >&2
    send_metric "datadog.skill.watchdog.anomalies.count" 0 "status:ok"
    cat <<EOF
{
  "status": "ok",
  "total_anomalies": 0,
  "duration": "$DURATION",
  "query": "$QUERY",
  "anomalies": [],
  "summary": {
    "latency_spikes": 0,
    "error_rate_increases": 0,
    "traffic_drops": 0,
    "other": 0
  },
  "affected_services": {}
}
EOF
    exit 0
fi

# Categorize anomalies by type
LATENCY=$(echo "$RESPONSE" | jq '[.data[] | select(.attributes.attributes.title // .attributes.attributes.aggregation_key // "" | test("latency|p99|response.?time"; "i"))] | length')
ERROR_RATE=$(echo "$RESPONSE" | jq '[.data[] | select(.attributes.attributes.title // .attributes.attributes.aggregation_key // "" | test("error|failure"; "i"))] | length')
TRAFFIC=$(echo "$RESPONSE" | jq '[.data[] | select(.attributes.attributes.title // .attributes.attributes.aggregation_key // "" | test("hits|traffic|request|throughput|drop"; "i"))] | length')
OTHER=$((TOTAL_ANOMALIES - LATENCY - ERROR_RATE - TRAFFIC))

echo "[SUMMARY] Anomaly type breakdown:" >&2
echo "  Latency spikes: $LATENCY" >&2
echo "  Error rate increases: $ERROR_RATE" >&2
echo "  Traffic drops: $TRAFFIC" >&2
echo "  Other: $OTHER" >&2
echo "" >&2

# Extract affected services
echo "[INFO] Affected services:" >&2
echo "$RESPONSE" | jq -r '.data[].attributes.tags[]? // empty' | \
  grep '^service:' | \
  sort | uniq -c | sort -rn | head -10 >&2 || echo "  No service tags found" >&2
echo "" >&2

# Determine overall status
if [ "$ERROR_RATE" -gt 0 ]; then
    STATUS="critical"
elif [ "$LATENCY" -gt 3 ] || [ "$TRAFFIC" -gt 3 ]; then
    STATUS="warning"
else
    STATUS="ok"
fi

# Send metrics
send_metric "datadog.skill.watchdog.anomalies.count" "$TOTAL_ANOMALIES" "status:ok"
send_metric "datadog.skill.watchdog.latency_spikes" "$LATENCY" "status:ok"
send_metric "datadog.skill.watchdog.error_rate_increases" "$ERROR_RATE" "status:ok"

# Output structured JSON
cat <<EOF
{
  "status": "$STATUS",
  "total_anomalies": $TOTAL_ANOMALIES,
  "duration": "$DURATION",
  "query": "$QUERY",
  "summary": {
    "latency_spikes": $LATENCY,
    "error_rate_increases": $ERROR_RATE,
    "traffic_drops": $TRAFFIC,
    "other": $OTHER
  },
  "affected_services": $(echo "$RESPONSE" | jq '[.data[].attributes.tags[]? // empty | select(startswith("service:"))] | group_by(.) | map({(.[0]): length}) | add // {}'),
  "anomalies": $(echo "$RESPONSE" | jq '[.data[] | {
    id: .id,
    type: .type,
    timestamp: .attributes.timestamp,
    title: .attributes.attributes.title,
    message: (.attributes.attributes.message // "" | split("\n")[0]),
    tags: .attributes.tags,
    priority: .attributes.attributes.priority,
    service: ([.attributes.tags[]? // empty | select(startswith("service:")) | sub("service:"; "")] | first),
    resource: ([.attributes.tags[]? // empty | select(startswith("resource_name:")) | sub("resource_name:"; "")] | first),
    anomaly_category: (
      if (.attributes.attributes.title // "" | test("latency|p99|response.?time"; "i")) then "latency_spike"
      elif (.attributes.attributes.title // "" | test("error|failure"; "i")) then "error_rate_increase"
      elif (.attributes.attributes.title // "" | test("hits|traffic|request|throughput|drop"; "i")) then "traffic_drop"
      else "other"
      end
    ),
    severity: (
      if (.attributes.attributes.priority == "normal") then "medium"
      elif (.attributes.attributes.priority == "low") then "low"
      else "high"
      end
    )
  }]')
}
EOF
