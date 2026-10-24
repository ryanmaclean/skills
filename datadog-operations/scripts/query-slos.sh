#!/bin/bash
set -e

# Source monitoring library
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/datadog-monitoring.sh" 2>/dev/null || true
init_monitoring "$(basename "$0")"
trap 'finalize_monitoring $?' EXIT INT TERM

# Query Datadog SLOs (Service Level Objectives)
# Check SLO status and error budgets

if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    cat << 'EOF'
Query Datadog SLOs

Usage:
  query-slos.sh [--service SERVICE] [--tag TAG]

Examples:
  # List all SLOs
  query-slos.sh

  # List SLOs for specific service
  query-slos.sh --service payment-api

  # List SLOs with tag
  query-slos.sh --tag team:backend

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
TAG=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --service)
            SERVICE="$2"
            shift 2
            ;;
        --tag)
            TAG="$2"
            shift 2
            ;;
        *)
            echo "[ERROR] Unknown argument: $1" >&2
            exit 1
            ;;
    esac
done

echo "[INFO] Querying SLOs..." >&2
if [ -n "$SERVICE" ]; then
    echo "[INFO] Service filter: $SERVICE" >&2
fi
if [ -n "$TAG" ]; then
    echo "[INFO] Tag filter: $TAG" >&2
fi
echo "" >&2

# Build query parameters
PARAMS=""
if [ -n "$TAG" ]; then
    PARAMS="?tags=${TAG}"
fi

# Query SLOs
start_operation "api_call"
RESPONSE=$(curl -s -X GET "https://api.${DD_SITE}/api/v1/slo${PARAMS}" \
    -H "DD-API-KEY: ${DD_API_KEY}" \
    -H "DD-APPLICATION-KEY: ${DD_APP_KEY}")

# Check for errors
if echo "$RESPONSE" | jq -e '.errors' > /dev/null 2>&1; then
    end_operation "error" "error_type:api_error"
    echo "[ERROR] API error:" >&2
    echo "$RESPONSE" | jq '.errors' >&2
    exit 1
fi

end_operation "ok"

# Filter by service if specified
if [ -n "$SERVICE" ]; then
    RESPONSE=$(echo "$RESPONSE" | jq --arg service "$SERVICE" '{
      data: [.data[] | select(.tags[]? | contains("service:\($service)"))]
    }')
fi

TOTAL=$(echo "$RESPONSE" | jq '.data | length')

echo "[INFO] Found $TOTAL SLOs" >&2
echo "" >&2

if [ "$TOTAL" -eq 0 ]; then
    echo "[INFO] No SLOs found for the specified criteria" >&2
    send_metric "datadog.skill.slos.count" 0 "status:ok"
    cat <<EOF
{
  "status": "ok",
  "total_slos": 0,
  "slos": []
}
EOF
    exit 0
fi

# Calculate status breakdown
BREACHING=$(echo "$RESPONSE" | jq '[.data[] | select(.slo_value < .target_threshold)] | length')
WARNING=$(echo "$RESPONSE" | jq '[.data[] | select(.slo_value >= .target_threshold and .slo_value < .warning_threshold)] | length')
OK=$(echo "$RESPONSE" | jq '[.data[] | select(.slo_value >= .warning_threshold)] | length')

echo "[SUMMARY] SLO status:" >&2
echo "  Breaching: $BREACHING" >&2
echo "  Warning: $WARNING" >&2
echo "  OK: $OK" >&2
echo "" >&2

# Calculate error budget status
BUDGET_EXHAUSTED=$(echo "$RESPONSE" | jq '[.data[] | select(.error_budget_remaining <= 0)] | length')
BUDGET_LOW=$(echo "$RESPONSE" | jq '[.data[] | select(.error_budget_remaining > 0 and .error_budget_remaining < 20)] | length')

if [ "$BUDGET_EXHAUSTED" -gt 0 ]; then
    echo "[WARN] $BUDGET_EXHAUSTED SLOs have exhausted error budget" >&2
fi

if [ "$BUDGET_LOW" -gt 0 ]; then
    echo "[WARN] $BUDGET_LOW SLOs have low error budget (<20%)" >&2
fi

echo "" >&2

# Determine overall status
if [ "$BREACHING" -gt 0 ]; then
    OVERALL_STATUS="breaching"
elif [ "$WARNING" -gt 0 ]; then
    OVERALL_STATUS="warning"
else
    OVERALL_STATUS="ok"
fi

# Send metrics
send_metric "datadog.skill.slos.count" "$TOTAL" "status:ok"
send_metric "datadog.skill.slos.breaching" "$BREACHING" "status:ok"
send_metric "datadog.skill.slos.budget_exhausted" "$BUDGET_EXHAUSTED" "status:ok"

# Output structured JSON
cat <<EOF
{
  "status": "$OVERALL_STATUS",
  "total_slos": $TOTAL,
  "summary": {
    "breaching": $BREACHING,
    "warning": $WARNING,
    "ok": $OK,
    "budget_exhausted": $BUDGET_EXHAUSTED,
    "budget_low": $BUDGET_LOW
  },
  "slos": $(echo "$RESPONSE" | jq '[.data[] | {
    id: .id,
    name: .name,
    type: .type,
    current_value: (.slo_value // 0 | . * 100 | floor / 100),
    target: (.target_threshold // 0 | . * 100 | floor / 100),
    warning: (.warning_threshold // 0 | . * 100 | floor / 100),
    error_budget_remaining: (.error_budget_remaining // 0 | floor),
    timeframe: .timeframe,
    tags: .tags,
    status: (
      if (.slo_value // 0) < (.target_threshold // 0) then "breaching"
      elif (.slo_value // 0) < (.warning_threshold // 0) then "warning"
      else "ok"
      end
    ),
    budget_status: (
      if (.error_budget_remaining // 0) <= 0 then "exhausted"
      elif (.error_budget_remaining // 0) < 20 then "low"
      else "healthy"
      end
    )
  }]')
}
EOF
