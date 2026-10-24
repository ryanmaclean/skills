#!/bin/bash
set -e

# Source monitoring library
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/datadog-monitoring.sh" 2>/dev/null || true
init_monitoring "$(basename "$0")"
trap 'finalize_monitoring $?' EXIT INT TERM

# Search and Analyze Datadog Logs
# Searches logs for error patterns and provides frequency analysis

show_usage() {
    cat <<EOF
Usage: $0 [OPTIONS]

Search and analyze Datadog logs for error patterns

OPTIONS:
    --query QUERY      Log search query (Datadog query syntax)
    --service SERVICE  Filter by service name
    --duration TIME    Time range: 1h, 24h, 7d, 30d (default: 24h)
    --status STATUS    Filter by status: error, warn, info
    --limit NUMBER     Number of results (default: 100, max: 1000)
    --help            Show this help message

EXAMPLES:
    # Search for errors in a specific service
    $0 --service "my-api" --status error --duration 1h

    # Search for specific error patterns
    $0 --query "Exception OR timeout" --duration 24h

    # Combine filters
    $0 --query "database" --service "backend" --status error --limit 50

ENVIRONMENT VARIABLES:
    DD_API_KEY    Datadog API key (required)
    DD_APP_KEY    Datadog Application key (required)
    DD_SITE       Datadog site (default: datadoghq.com)
EOF
    exit 0
}

# Parse arguments (check for help first)
QUERY=""
SERVICE=""
DURATION="24h"
STATUS=""
LIMIT=100

while [[ $# -gt 0 ]]; do
    case $1 in
        --query)
            QUERY="$2"
            shift 2
            ;;
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
        --help)
            show_usage
            ;;
        *)
            echo "[ERROR] Unknown argument: $1" >&2
            echo "Use --help for usage information" >&2
            exit 1
            ;;
    esac
done

# Check required environment variables
if [ -z "$DD_API_KEY" ] || [ -z "$DD_APP_KEY" ]; then
    echo "[ERROR] DD_API_KEY and DD_APP_KEY must be set" >&2
    exit 1
fi

DD_SITE=${DD_SITE:-datadoghq.com}

# Validate limit
if [ "$LIMIT" -gt 1000 ]; then
    echo "[ERROR] Limit cannot exceed 1000" >&2
    exit 1
fi

# Convert duration to milliseconds
case $DURATION in
    1h)
        FROM_MS=$(($(date +%s) - 3600))000
        ;;
    24h)
        FROM_MS=$(($(date +%s) - 86400))000
        ;;
    7d)
        FROM_MS=$(($(date +%s) - 604800))000
        ;;
    30d)
        FROM_MS=$(($(date +%s) - 2592000))000
        ;;
    *)
        echo "[ERROR] Invalid duration. Use: 1h, 24h, 7d, or 30d" >&2
        exit 1
        ;;
esac

TO_MS=$(date +%s)000

# Build query
SEARCH_QUERY=""

if [ -n "$SERVICE" ]; then
    SEARCH_QUERY="service:$SERVICE"
fi

if [ -n "$STATUS" ]; then
    if [ -n "$SEARCH_QUERY" ]; then
        SEARCH_QUERY="$SEARCH_QUERY AND status:$STATUS"
    else
        SEARCH_QUERY="status:$STATUS"
    fi
fi

if [ -n "$QUERY" ]; then
    if [ -n "$SEARCH_QUERY" ]; then
        SEARCH_QUERY="$SEARCH_QUERY AND ($QUERY)"
    else
        SEARCH_QUERY="$QUERY"
    fi
fi

# If no query specified, search for errors by default
if [ -z "$SEARCH_QUERY" ]; then
    SEARCH_QUERY="status:error"
fi

echo "[INFO] Searching Datadog logs..." >&2
echo "[INFO] Query: $SEARCH_QUERY" >&2
echo "[INFO] Duration: $DURATION" >&2
echo "[INFO] Limit: $LIMIT" >&2
echo "" >&2

# Build JSON request body
REQUEST_BODY=$(cat <<EOF
{
  "filter": {
    "query": "$SEARCH_QUERY",
    "from": "$FROM_MS",
    "to": "$TO_MS"
  },
  "page": {
    "limit": $LIMIT
  },
  "sort": "timestamp"
}
EOF
)

# Query logs API
start_operation "api_call"
RESPONSE=$(curl -s -X POST "https://api.${DD_SITE}/api/v2/logs/events/search" \
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

# Parse response
TOTAL_LOGS=$(echo "$RESPONSE" | jq '.data | length')

echo "[INFO] Found $TOTAL_LOGS log entries" >&2
echo "" >&2

if [ "$TOTAL_LOGS" -eq 0 ]; then
    echo "No logs found for the specified criteria" >&2
    send_metric "datadog.skill.logs.count" 0 "status:no_data"
    cat <<EOF
{
  "status": "ok",
  "total_logs": 0,
  "query": "$SEARCH_QUERY",
  "duration": "$DURATION",
  "logs": [],
  "error_patterns": [],
  "summary": {
    "error": 0,
    "warn": 0,
    "info": 0
  },
  "services": {},
  "hosts": {}
}
EOF
    exit 0
fi

# Count by status
ERROR_COUNT=$(echo "$RESPONSE" | jq '[.data[] | select(.attributes.status == "error")] | length')
WARN_COUNT=$(echo "$RESPONSE" | jq '[.data[] | select(.attributes.status == "warn")] | length')
INFO_COUNT=$(echo "$RESPONSE" | jq '[.data[] | select(.attributes.status == "info")] | length')

echo "[SUMMARY] Status breakdown:" >&2
echo "  Errors: $ERROR_COUNT" >&2
echo "  Warnings: $WARN_COUNT" >&2
echo "  Info: $INFO_COUNT" >&2
echo "" >&2

# Extract error messages and group by frequency
echo "[INFO] Top error messages:" >&2
ERROR_MESSAGES=$(echo "$RESPONSE" | jq -r '.data[] | select(.attributes.status == "error") | .attributes.message // .attributes.attributes.message // "No message"' 2>/dev/null | head -100)

if [ -n "$ERROR_MESSAGES" ]; then
    echo "$ERROR_MESSAGES" | \
        head -c 10000 | \
        awk '{print substr($0, 1, 100)}' | \
        sort | uniq -c | sort -rn | head -10 >&2
    echo "" >&2
else
    echo "  No error messages found" >&2
    echo "" >&2
fi

# Get service breakdown
echo "[INFO] Services with logs:" >&2
echo "$RESPONSE" | jq -r '.data[].attributes.service // "unknown"' | \
    sort | uniq -c | sort -rn | head -10 >&2
echo "" >&2

# Get host breakdown
echo "[INFO] Hosts generating logs:" >&2
echo "$RESPONSE" | jq -r '.data[].attributes.host // "unknown"' | \
    sort | uniq -c | sort -rn | head -10 >&2
echo "" >&2

# Extract trace IDs if available
TRACE_COUNT=$(echo "$RESPONSE" | jq '[.data[] | select(.attributes.attributes.dd.trace_id != null)] | length')
if [ "$TRACE_COUNT" -gt 0 ]; then
    echo "[INFO] Found $TRACE_COUNT logs with trace IDs" >&2
    echo "" >&2
fi

# Group error patterns by message similarity
ERROR_PATTERNS=$(echo "$RESPONSE" | jq -r '
[.data[] | select(.attributes.status == "error") | {
  message: (.attributes.message // .attributes.attributes.message // "No message"),
  service: (.attributes.service // "unknown"),
  host: (.attributes.host // "unknown")
}] |
group_by(.message) |
map({
  message: (.[0].message | tostring | .[0:200]),
  count: length,
  services: ([.[].service] | unique),
  hosts: ([.[].host] | unique)
}) |
sort_by(-.count) |
.[0:10]
')

# Determine overall status
if [ "$ERROR_COUNT" -gt 0 ]; then
    OVERALL_STATUS="error"
elif [ "$WARN_COUNT" -gt 0 ]; then
    OVERALL_STATUS="warning"
else
    OVERALL_STATUS="ok"
fi

# Send metrics
send_metric "datadog.skill.logs.count" "$TOTAL_LOGS" "status:ok"
send_metric "datadog.skill.logs.errors" "$ERROR_COUNT" "status:ok"
send_metric "datadog.skill.logs.warnings" "$WARN_COUNT" "status:ok"

# Output structured JSON
cat <<EOF
{
  "status": "$OVERALL_STATUS",
  "total_logs": $TOTAL_LOGS,
  "query": "$SEARCH_QUERY",
  "duration": "$DURATION",
  "timestamp_range": {
    "from": "$FROM_MS",
    "to": "$TO_MS"
  },
  "summary": {
    "error": $ERROR_COUNT,
    "warn": $WARN_COUNT,
    "info": $INFO_COUNT
  },
  "error_patterns": $ERROR_PATTERNS,
  "services": $(echo "$RESPONSE" | jq '[.data[].attributes.service // "unknown"] | group_by(.) | map({(.[0]): length}) | add // {}'),
  "hosts": $(echo "$RESPONSE" | jq '[.data[].attributes.host // "unknown"] | group_by(.) | map({(.[0]): length}) | add // {}'),
  "trace_ids_count": $TRACE_COUNT,
  "recent_logs": $(echo "$RESPONSE" | jq '[.data[0:10] | .[] | {
    timestamp: .attributes.timestamp,
    status: .attributes.status,
    service: (.attributes.service // "unknown"),
    host: (.attributes.host // "unknown"),
    message: ((.attributes.message // .attributes.attributes.message // "No message") | tostring | .[0:500]),
    trace_id: (.attributes.attributes.dd.trace_id // null),
    span_id: (.attributes.attributes.dd.span_id // null),
    container_id: (.attributes.attributes.container_id // null)
  }]')
}
EOF
