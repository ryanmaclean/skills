#!/bin/bash
set -e

# Source monitoring library
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/datadog-monitoring.sh" 2>/dev/null || true
init_monitoring "$(basename "$0")"
trap 'finalize_monitoring $?' EXIT INT TERM

# Query Datadog Security Monitoring Signals
# Analyzes security events and attack attempts

# Check required environment variables
if [ -z "$DD_API_KEY" ] || [ -z "$DD_APP_KEY" ]; then
    echo "[ERROR] DD_API_KEY and DD_APP_KEY must be set" >&2
    exit 1
fi

DD_SITE=${DD_SITE:-datadoghq.com}

# Parse arguments
SERVICE=""
DURATION="24h"
SEVERITY=""

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
        --severity)
            SEVERITY="$2"
            shift 2
            ;;
        *)
            echo "[ERROR] Unknown argument: $1" >&2
            exit 1
            ;;
    esac
done

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
QUERY=""
if [ -n "$SERVICE" ]; then
    QUERY="service:$SERVICE"
fi

if [ -n "$SEVERITY" ]; then
    if [ -n "$QUERY" ]; then
        QUERY="$QUERY AND severity:$SEVERITY"
    else
        QUERY="severity:$SEVERITY"
    fi
fi

echo "[INFO] Querying security signals..." >&2
echo "[INFO] Service: ${SERVICE:-all}" >&2
echo "[INFO] Duration: $DURATION" >&2
echo "[INFO] Severity: ${SEVERITY:-all}" >&2
echo "" >&2

# Query security signals API
start_operation "api_call"
RESPONSE=$(curl -s -X GET "https://api.${DD_SITE}/api/v2/security_monitoring/signals" \
  -H "DD-API-KEY: ${DD_API_KEY}" \
  -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
  -G \
  --data-urlencode "filter[query]=${QUERY}" \
  --data-urlencode "filter[from]=${FROM_MS}" \
  --data-urlencode "filter[to]=${TO_MS}" \
  --data-urlencode "page[limit]=100")

# Check for API errors
if echo "$RESPONSE" | jq -e '.errors' > /dev/null 2>&1; then
    end_operation "error" "error_type:api_error"
    echo "[ERROR] API error:" >&2
    echo "$RESPONSE" | jq '.errors' >&2
    exit 1
fi

end_operation "ok"

# Parse and analyze signals
TOTAL_SIGNALS=$(echo "$RESPONSE" | jq '.data | length')

echo "[INFO] Found $TOTAL_SIGNALS security signals" >&2
echo "" >&2

if [ "$TOTAL_SIGNALS" -eq 0 ]; then
    echo "No security signals found for the specified criteria" >&2
    send_metric "datadog.skill.security.signals.count" 0 "status:ok"
    cat <<EOF
{
  "status": "ok",
  "total_signals": 0,
  "signals": [],
  "summary": {
    "critical": 0,
    "high": 0,
    "medium": 0,
    "low": 0,
    "info": 0
  },
  "attack_types": {}
}
EOF
    exit 0
fi

# Count by severity
CRITICAL=$(echo "$RESPONSE" | jq '[.data[] | select(.attributes.severity == "critical")] | length')
HIGH=$(echo "$RESPONSE" | jq '[.data[] | select(.attributes.severity == "high")] | length')
MEDIUM=$(echo "$RESPONSE" | jq '[.data[] | select(.attributes.severity == "medium")] | length')
LOW=$(echo "$RESPONSE" | jq '[.data[] | select(.attributes.severity == "low")] | length')
INFO=$(echo "$RESPONSE" | jq '[.data[] | select(.attributes.severity == "info")] | length')

echo "[SUMMARY] Severity breakdown:" >&2
echo "  Critical: $CRITICAL" >&2
echo "  High: $HIGH" >&2
echo "  Medium: $MEDIUM" >&2
echo "  Low: $LOW" >&2
echo "  Info: $INFO" >&2
echo "" >&2

# Group by attack type/rule name
echo "[INFO] Top attack types:" >&2
echo "$RESPONSE" | jq -r '.data[].attributes.tags[]' | \
  grep -E '^(attack_type|rule_name):' | \
  sort | uniq -c | sort -rn | head -10 >&2
echo "" >&2

# Get affected services
echo "[INFO] Affected services:" >&2
echo "$RESPONSE" | jq -r '.data[].attributes.tags[]' | \
  grep '^service:' | \
  sort | uniq -c | sort -rn | head -10 >&2
echo "" >&2

# Send metrics
send_metric "datadog.skill.security.signals.count" "$TOTAL_SIGNALS" "status:ok"
send_metric "datadog.skill.security.signals.critical" "$CRITICAL" "status:ok"
send_metric "datadog.skill.security.signals.high" "$HIGH" "status:ok"

# Output structured JSON
cat <<EOF
{
  "status": "$([ $CRITICAL -gt 0 ] && echo "critical" || [ $HIGH -gt 0 ] && echo "warning" || echo "ok")",
  "total_signals": $TOTAL_SIGNALS,
  "duration": "$DURATION",
  "query": "${QUERY}",
  "summary": {
    "critical": $CRITICAL,
    "high": $HIGH,
    "medium": $MEDIUM,
    "low": $LOW,
    "info": $INFO
  },
  "attack_types": $(echo "$RESPONSE" | jq '[.data[].attributes.tags[] | select(startswith("attack_type:"))] | group_by(.) | map({(.[0]): length}) | add // {}'),
  "affected_services": $(echo "$RESPONSE" | jq '[.data[].attributes.tags[] | select(startswith("service:"))] | group_by(.) | map({(.[0]): length}) | add // {}'),
  "recent_signals": $(echo "$RESPONSE" | jq '[.data[0:5] | .[] | {
    id: .id,
    severity: .attributes.severity,
    timestamp: .attributes.timestamp,
    rule_name: (.attributes.tags[] | select(startswith("rule_name:")) | sub("rule_name:"; "")),
    service: (.attributes.tags[] | select(startswith("service:")) | sub("service:"; ""))
  }]')
}
EOF
