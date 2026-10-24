#!/bin/bash
set -e

# Query Datadog Database Monitoring
# Analyze database performance and slow queries

if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    cat << 'EOF'
Query Datadog Database Monitoring

Usage:
  query-database.sh --host HOST [--duration DURATION]

Examples:
  # Query database performance
  query-database.sh --host postgres-prod --duration 1h

  # Get slow queries
  query-database.sh --host mysql-01 --duration 24h

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
HOST=""
DURATION="1h"

while [[ $# -gt 0 ]]; do
    case $1 in
        --host)
            HOST="$2"
            shift 2
            ;;
        --duration)
            DURATION="$2"
            shift 2
            ;;
        *)
            echo "[ERROR] Unknown argument: $1" >&2
            exit 1
            ;;
    esac
done

if [ -z "$HOST" ]; then
    echo "[ERROR] --host is required" >&2
    exit 1
fi

# Convert duration to timestamps
case $DURATION in
    1h)
        FROM_TS=$(($(date +%s) - 3600))
        ;;
    24h)
        FROM_TS=$(($(date +%s) - 86400))
        ;;
    7d)
        FROM_TS=$(($(date +%s) - 604800))
        ;;
    *)
        echo "[ERROR] Invalid duration. Use: 1h, 24h, or 7d" >&2
        exit 1
        ;;
esac

TO_TS=$(date +%s)

echo "[INFO] Querying database monitoring..." >&2
echo "[INFO] Host: $HOST" >&2
echo "[INFO] Duration: $DURATION" >&2
echo "" >&2

# Query database metrics via metrics API
QUERY="avg:postgresql.connections.count{host:${HOST}}"

RESPONSE=$(curl -s -X GET "https://api.${DD_SITE}/api/v1/query" \
    -H "DD-API-KEY: ${DD_API_KEY}" \
    -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
    -G \
    --data-urlencode "query=${QUERY}" \
    --data-urlencode "from=${FROM_TS}" \
    --data-urlencode "to=${TO_TS}")

# Check for errors
if echo "$RESPONSE" | jq -e '.errors' > /dev/null 2>&1; then
    echo "[ERROR] API error:" >&2
    echo "$RESPONSE" | jq '.errors' >&2
    exit 1
fi

# Query slow queries via APM
SLOW_QUERIES=$(curl -s -X POST "https://api.${DD_SITE}/api/v2/spans/analytics/aggregate" \
    -H "DD-API-KEY: ${DD_API_KEY}" \
    -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
    -H "Content-Type: application/json" \
    -d "{
      \"filter\": {
        \"from\": \"$((FROM_TS * 1000000000))\",
        \"to\": \"$((TO_TS * 1000000000))\",
        \"query\": \"resource_type:sql host:${HOST}\"
      },
      \"compute\": [
        {\"aggregation\": \"count\", \"metric\": \"*\"},
        {\"aggregation\": \"pc95\", \"metric\": \"duration\"},
        {\"aggregation\": \"avg\", \"metric\": \"duration\"}
      ],
      \"group_by\": [
        {\"facet\": \"resource_name\", \"limit\": 20, \"sort\": {\"order\": \"desc\", \"aggregation\": \"pc95\", \"metric\": \"duration\"}}
      ]
    }")

# Check for slow query errors
if echo "$SLOW_QUERIES" | jq -e '.errors' > /dev/null 2>&1; then
    echo "[WARN] Could not fetch slow queries" >&2
    SLOW_QUERIES='{"data":{"buckets":[]}}'
fi

QUERY_COUNT=$(echo "$SLOW_QUERIES" | jq '.data.buckets | length')

echo "[INFO] Found $QUERY_COUNT query patterns" >&2
echo "" >&2

# Analyze slow queries
if [ "$QUERY_COUNT" -gt 0 ]; then
    SLOWEST_P95=$(echo "$SLOW_QUERIES" | jq '[.data.buckets[].computes.c1 // 0] | max // 0 / 1000000 | floor')
    AVG_DURATION=$(echo "$SLOW_QUERIES" | jq '[.data.buckets[].computes.c2 // 0] | add / length // 0 / 1000000 | floor')

    echo "[SUMMARY] Database performance:" >&2
    echo "  Slowest query P95: ${SLOWEST_P95}ms" >&2
    echo "  Average duration: ${AVG_DURATION}ms" >&2
    echo "" >&2
fi

# Output structured JSON
cat <<EOF
{
  "status": "ok",
  "host": "$HOST",
  "duration": "$DURATION",
  "summary": {
    "query_patterns": $QUERY_COUNT,
    "slowest_p95_ms": $(echo "$SLOW_QUERIES" | jq '[.data.buckets[].computes.c1 // 0] | max // 0 / 1000000 | floor'),
    "avg_duration_ms": $(echo "$SLOW_QUERIES" | jq '[.data.buckets[].computes.c2 // 0] | add / length // 0 / 1000000 | floor')
  },
  "slow_queries": $(echo "$SLOW_QUERIES" | jq '[.data.buckets[] | {
    query: .by.resource_name,
    count: .computes.c0,
    p95_ms: ((.computes.c1 // 0) / 1000000 | floor),
    avg_ms: ((.computes.c2 // 0) / 1000000 | floor)
  }]'),
  "metrics": {
    "connections": $(echo "$RESPONSE" | jq '.series[0].pointlist[-1][1] // 0 | floor')
  }
}
EOF
