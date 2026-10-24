#!/bin/bash
set -e

# Calculate Error Budget
# Determines how many errors can occur before SLO breach

if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    cat << 'EOF'
Calculate Error Budget

Calculates remaining error budget and time-to-breach estimates for SLOs.

Usage:
  calculate-error-budget.sh --service SERVICE [--slo-target TARGET]

Arguments:
  --service SERVICE    Service name to check
  --slo-target TARGET  SLO target percentage (default: 99.9)

Examples:
  # Calculate error budget for service
  calculate-error-budget.sh --service payment-api

  # Calculate with custom SLO target
  calculate-error-budget.sh --service payment-api --slo-target 99.5

Output:
  - Current error rate
  - SLO target
  - Error budget remaining (%)
  - Allowed errors remaining (count)
  - Time to budget exhaustion (estimated)
  - Burn rate

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
SLO_TARGET="99.9"

while [[ $# -gt 0 ]]; do
    case $1 in
        --service)
            SERVICE="$2"
            shift 2
            ;;
        --slo-target)
            SLO_TARGET="$2"
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

echo "[INFO] Calculating error budget for: $SERVICE" >&2
echo "[INFO] SLO target: ${SLO_TARGET}%" >&2
echo "" >&2

# Get current error rate from APM (last 30 days)
FROM_TS=$(($(date +%s) - 2592000))
TO_TS=$(date +%s)
FROM_NS=${FROM_TS}000000000
TO_NS=${TO_TS}000000000

REQUEST_BODY=$(cat <<EOF
{
  "data": {
    "type": "aggregate_request",
    "attributes": {
      "filter": {
        "from": "now-30d",
        "to": "now",
        "query": "service:${SERVICE}"
      },
      "compute": [
        {"aggregation": "count"},
        {"aggregation": "count", "metric": "@error"}
      ],
      "group_by": []
    }
  }
}
EOF
)

RESPONSE=$(curl -s -X POST "https://api.${DD_SITE}/api/v2/spans/analytics/aggregate" \
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

# Parse response (v2 API format: .data is array)
if ! echo "$RESPONSE" | jq -e '.data[0]' > /dev/null 2>&1; then
    echo "[ERROR] No data found for service: $SERVICE" >&2
    cat <<EOF
{
  "status": "no_data",
  "service": "$SERVICE",
  "message": "No APM data found for this service"
}
EOF
    exit 0
fi

TOTAL_REQUESTS=$(echo "$RESPONSE" | jq '.data[0].attributes.compute.c0 // 0')
ERROR_COUNT=$(echo "$RESPONSE" | jq '.data[0].attributes.compute.c1 // 0')

if [ "$TOTAL_REQUESTS" -eq 0 ]; then
    echo "[ERROR] No requests found for service: $SERVICE" >&2
    exit 1
fi

# Calculate current error rate
CURRENT_ERROR_RATE=$(echo "scale=4; ($ERROR_COUNT * 100) / $TOTAL_REQUESTS" | bc)
CURRENT_SUCCESS_RATE=$(echo "scale=4; 100 - $CURRENT_ERROR_RATE" | bc)

echo "[INFO] Total requests (30d): $TOTAL_REQUESTS" >&2
echo "[INFO] Errors: $ERROR_COUNT" >&2
echo "[INFO] Current success rate: ${CURRENT_SUCCESS_RATE}%" >&2
echo "" >&2

# Calculate error budget
ERROR_BUDGET_PERCENT=$(echo "scale=4; 100 - $SLO_TARGET" | bc)
ALLOWED_ERRORS=$(echo "scale=0; ($TOTAL_REQUESTS * $ERROR_BUDGET_PERCENT) / 100" | bc)
ERRORS_CONSUMED=$(echo "scale=0; $ERROR_COUNT / 1" | bc)
REMAINING_ERRORS=$(echo "scale=0; $ALLOWED_ERRORS - $ERRORS_CONSUMED" | bc)

# Calculate budget remaining percentage
if [ "$ALLOWED_ERRORS" -eq 0 ]; then
    BUDGET_REMAINING_PERCENT="0"
else
    BUDGET_REMAINING_PERCENT=$(echo "scale=2; ($REMAINING_ERRORS * 100) / $ALLOWED_ERRORS" | bc)
fi

# Calculate burn rate (errors per day)
BURN_RATE=$(echo "scale=2; $ERRORS_CONSUMED / 30" | bc)

# Estimate time to exhaustion
if [ "$BURN_RATE" = "0" ] || [ "$BURN_RATE" = "0.00" ]; then
    DAYS_TO_EXHAUSTION="infinite"
    STATUS="healthy"
else
    DAYS_TO_EXHAUSTION=$(echo "scale=1; $REMAINING_ERRORS / $BURN_RATE" | bc)

    # Determine status
    if (( $(echo "$DAYS_TO_EXHAUSTION < 7" | bc -l) )); then
        STATUS="critical"
    elif (( $(echo "$DAYS_TO_EXHAUSTION < 14" | bc -l) )); then
        STATUS="warning"
    else
        STATUS="healthy"
    fi
fi

# Check if budget is exhausted
if (( $(echo "$REMAINING_ERRORS < 0" | bc -l) )); then
    STATUS="exhausted"
    BUDGET_REMAINING_PERCENT="0"
    DAYS_TO_EXHAUSTION="0"
fi

echo "[SUMMARY] Error Budget Analysis:" >&2
echo "  Status: $STATUS" >&2
echo "  Allowed errors (30d): $ALLOWED_ERRORS" >&2
echo "  Errors consumed: $ERRORS_CONSUMED" >&2
echo "  Remaining errors: $REMAINING_ERRORS" >&2
echo "  Budget remaining: ${BUDGET_REMAINING_PERCENT}%" >&2
echo "  Burn rate: ${BURN_RATE} errors/day" >&2
echo "  Days to exhaustion: $DAYS_TO_EXHAUSTION" >&2
echo "" >&2

# Generate recommendations
RECOMMENDATIONS=()

if [ "$STATUS" = "exhausted" ]; then
    RECOMMENDATIONS+=('{"priority": "critical", "action": "Error budget exhausted - SLO breach. Implement immediate remediation."}')
elif [ "$STATUS" = "critical" ]; then
    RECOMMENDATIONS+=('{"priority": "critical", "action": "Error budget critically low - less than 7 days remaining. Freeze non-critical deployments."}')
elif [ "$STATUS" = "warning" ]; then
    RECOMMENDATIONS+=('{"priority": "high", "action": "Error budget below 50% - review error patterns and improve reliability."}')
fi

if (( $(echo "$BURN_RATE > $(echo "scale=2; $ALLOWED_ERRORS / 30 * 1.5" | bc)" | bc -l) )); then
    RECOMMENDATIONS+=('{"priority": "high", "action": "Burn rate 50% above normal - investigate recent changes."}')
fi

# Output structured JSON
cat <<EOF
{
  "status": "$STATUS",
  "service": "$SERVICE",
  "slo_target": $SLO_TARGET,
  "period_days": 30,
  "current_metrics": {
    "total_requests": $TOTAL_REQUESTS,
    "error_count": $ERROR_COUNT,
    "success_rate": $CURRENT_SUCCESS_RATE,
    "error_rate": $CURRENT_ERROR_RATE
  },
  "error_budget": {
    "allowed_errors": $ALLOWED_ERRORS,
    "consumed_errors": $ERRORS_CONSUMED,
    "remaining_errors": $REMAINING_ERRORS,
    "budget_remaining_percent": $BUDGET_REMAINING_PERCENT
  },
  "burn_rate": {
    "errors_per_day": $BURN_RATE,
    "days_to_exhaustion": "$DAYS_TO_EXHAUSTION",
    "acceptable_rate": $(echo "scale=2; $ALLOWED_ERRORS / 30" | bc)
  },
  "recommendations": [
    $(IFS=,; echo "${RECOMMENDATIONS[*]}")
  ]
}
EOF
