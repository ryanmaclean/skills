#!/bin/bash
set -e

# Query Datadog Real User Monitoring (RUM)
# Analyze frontend performance and user experience

if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    cat << 'EOF'
Query Datadog RUM (Real User Monitoring)

Usage:
  query-rum.sh --application APP_ID [--duration DURATION]

Examples:
  # Query RUM data for application
  query-rum.sh --application abc-123-def --duration 1h

  # Get page load performance
  query-rum.sh --application abc-123-def --duration 24h

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
APPLICATION=""
DURATION="1h"

while [[ $# -gt 0 ]]; do
    case $1 in
        --application)
            APPLICATION="$2"
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

if [ -z "$APPLICATION" ]; then
    echo "[ERROR] --application is required" >&2
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
FROM_MS=$((FROM_TS * 1000))
TO_MS=$((TO_TS * 1000))

echo "[INFO] Querying RUM data..." >&2
echo "[INFO] Application: $APPLICATION" >&2
echo "[INFO] Duration: $DURATION" >&2
echo "" >&2

# Query RUM events
REQUEST_BODY=$(cat <<EOF
{
  "filter": {
    "from": "${FROM_MS}",
    "to": "${TO_MS}",
    "query": "service:${APPLICATION}"
  },
  "page": {
    "limit": 100
  },
  "sort": "timestamp"
}
EOF
)

RESPONSE=$(curl -s -X POST "https://api.${DD_SITE}/api/v2/rum/events/search" \
    -H "DD-API-KEY: ${DD_API_KEY}" \
    -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
    -H "Content-Type: application/json" \
    -d "$REQUEST_BODY")

# Check for errors
if echo "$RESPONSE" | jq -e '.errors' > /dev/null 2>&1; then
    echo "[ERROR] API error:" >&2
    echo "$RESPONSE" | jq '.errors' >&2
    exit 1
fi

TOTAL_EVENTS=$(echo "$RESPONSE" | jq '.data | length')

echo "[INFO] Found $TOTAL_EVENTS RUM events" >&2
echo "" >&2

if [ "$TOTAL_EVENTS" -eq 0 ]; then
    echo "[INFO] No RUM data found for application: $APPLICATION" >&2
    cat <<EOF
{
  "status": "no_data",
  "application": "$APPLICATION",
  "duration": "$DURATION",
  "message": "No RUM events found. Ensure RUM is configured for this application."
}
EOF
    exit 0
fi

# Analyze page load times
PAGE_LOADS=$(echo "$RESPONSE" | jq '[.data[] | select(.attributes.type == "view")] | length')
ERRORS=$(echo "$RESPONSE" | jq '[.data[] | select(.attributes.type == "error")] | length')
RESOURCES=$(echo "$RESPONSE" | jq '[.data[] | select(.attributes.type == "resource")] | length')

echo "[SUMMARY] Event breakdown:" >&2
echo "  Page loads: $PAGE_LOADS" >&2
echo "  Errors: $ERRORS" >&2
echo "  Resources: $RESOURCES" >&2
echo "" >&2

# Calculate performance metrics
if [ "$PAGE_LOADS" -gt 0 ]; then
    AVG_LOAD_TIME=$(echo "$RESPONSE" | jq '[.data[] | select(.attributes.type == "view") | .attributes.view.loading_time // 0] | add / length | floor')
    P95_LOAD_TIME=$(echo "$RESPONSE" | jq '[.data[] | select(.attributes.type == "view") | .attributes.view.loading_time // 0] | sort | .[length * 95 / 100 | floor]')

    echo "[SUMMARY] Page load performance:" >&2
    echo "  Average load time: ${AVG_LOAD_TIME}ms" >&2
    echo "  P95 load time: ${P95_LOAD_TIME}ms" >&2
    echo "" >&2
fi

# Determine status
STATUS="ok"
if [ "$ERRORS" -gt 10 ]; then
    STATUS="warning"
fi
if [ "$ERRORS" -gt 50 ]; then
    STATUS="critical"
fi

# Output structured JSON
cat <<EOF
{
  "status": "$STATUS",
  "application": "$APPLICATION",
  "duration": "$DURATION",
  "total_events": $TOTAL_EVENTS,
  "summary": {
    "page_loads": $PAGE_LOADS,
    "errors": $ERRORS,
    "resources": $RESOURCES,
    "avg_load_time_ms": $(echo "$RESPONSE" | jq '[.data[] | select(.attributes.type == "view") | .attributes.view.loading_time // 0] | add / length // 0 | floor'),
    "p95_load_time_ms": $(echo "$RESPONSE" | jq '[.data[] | select(.attributes.type == "view") | .attributes.view.loading_time // 0] | sort | .[length * 95 / 100 | floor] // 0'),
    "error_rate": $(echo "scale=2; $ERRORS * 100 / $TOTAL_EVENTS" | bc)
  },
  "top_pages": $(echo "$RESPONSE" | jq '[.data[] | select(.attributes.type == "view") | {
    url: .attributes.view.url,
    loading_time_ms: (.attributes.view.loading_time // 0 | floor)
  }] | group_by(.url) | map({
    url: .[0].url,
    count: length,
    avg_load_time_ms: ([.[].loading_time_ms] | add / length | floor)
  }) | sort_by(-.count) | .[0:10]'),
  "errors": $(echo "$RESPONSE" | jq '[.data[] | select(.attributes.type == "error") | {
    message: .attributes.error.message,
    type: .attributes.error.type,
    source: .attributes.error.source
  }] | .[0:10]')
}
EOF
