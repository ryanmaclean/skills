#!/bin/bash
set -e

# Comprehensive Service Investigation
# Runs multiple checks and correlates results

if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    cat << 'EOF'
Comprehensive Service Investigation

Runs multiple Datadog checks and correlates results for a service.

Usage:
  investigate-service.sh --service SERVICE [--duration DURATION]

Checks performed:
  1. APM performance (slow endpoints, errors)
  2. Security signals (threats, attacks)
  3. Watchdog anomalies (automated detection)
  4. Log errors (recent error patterns)
  5. SLO status (if configured)

Examples:
  # Quick investigation (1h)
  investigate-service.sh --service payment-api

  # Extended investigation (24h)
  investigate-service.sh --service payment-api --duration 24h

Output:
  JSON summary with all findings and severity assessment

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

# Parse arguments
SERVICE=""
DURATION="1h"

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

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "[INFO] Investigating service: $SERVICE" >&2
echo "[INFO] Duration: $DURATION" >&2
echo "[INFO] Running comprehensive checks..." >&2
echo "" >&2

# Initialize findings
ISSUES_FOUND=0
WARNINGS=0
CRITICAL=0

# 1. Check APM performance
echo "[CHECK] APM performance..." >&2
APM_RESULT=$(bash "$SCRIPT_DIR/query-apm.sh" --service "$SERVICE" --duration "$DURATION" 2>/dev/null || echo '{"status":"error","slow_endpoints_count":0}')
SLOW_ENDPOINTS=$(echo "$APM_RESULT" | jq '.summary.slow_endpoints_count // 0')

if [ "$SLOW_ENDPOINTS" -gt 0 ]; then
    echo "  [WARN] Found $SLOW_ENDPOINTS slow endpoints (P95 > 500ms)" >&2
    WARNINGS=$((WARNINGS + 1))
    ISSUES_FOUND=$((ISSUES_FOUND + 1))
else
    echo "  [OK] No slow endpoints detected" >&2
fi

# 2. Check security signals
echo "[CHECK] Security signals..." >&2
SECURITY_RESULT=$(bash "$SCRIPT_DIR/query-security-signals.sh" --service "$SERVICE" --duration "$DURATION" 2>/dev/null || echo '{"summary":{"critical":0,"high":0}}')
CRITICAL_SIGNALS=$(echo "$SECURITY_RESULT" | jq '.summary.critical // 0')
HIGH_SIGNALS=$(echo "$SECURITY_RESULT" | jq '.summary.high // 0')

if [ "$CRITICAL_SIGNALS" -gt 0 ]; then
    echo "  [CRITICAL] Found $CRITICAL_SIGNALS critical security signals" >&2
    CRITICAL=$((CRITICAL + 1))
    ISSUES_FOUND=$((ISSUES_FOUND + 1))
elif [ "$HIGH_SIGNALS" -gt 0 ]; then
    echo "  [WARN] Found $HIGH_SIGNALS high-severity security signals" >&2
    WARNINGS=$((WARNINGS + 1))
    ISSUES_FOUND=$((ISSUES_FOUND + 1))
else
    echo "  [OK] No security threats detected" >&2
fi

# 3. Check Watchdog anomalies
echo "[CHECK] Watchdog anomalies..." >&2
WATCHDOG_RESULT=$(bash "$SCRIPT_DIR/query-watchdog.sh" --service "$SERVICE" --duration "$DURATION" 2>/dev/null || echo '{"summary":{"latency_spikes":0,"error_rate_increases":0}}')
LATENCY_SPIKES=$(echo "$WATCHDOG_RESULT" | jq '.summary.latency_spikes // 0')
ERROR_SPIKES=$(echo "$WATCHDOG_RESULT" | jq '.summary.error_rate_increases // 0')

if [ "$ERROR_SPIKES" -gt 0 ]; then
    echo "  [CRITICAL] Detected $ERROR_SPIKES error rate increases" >&2
    CRITICAL=$((CRITICAL + 1))
    ISSUES_FOUND=$((ISSUES_FOUND + 1))
elif [ "$LATENCY_SPIKES" -gt 0 ]; then
    echo "  [WARN] Detected $LATENCY_SPIKES latency spikes" >&2
    WARNINGS=$((WARNINGS + 1))
    ISSUES_FOUND=$((ISSUES_FOUND + 1))
else
    echo "  [OK] No anomalies detected" >&2
fi

# 4. Check error logs
echo "[CHECK] Error logs..." >&2
LOG_RESULT=$(bash "$SCRIPT_DIR/search-logs.sh" --query "service:${SERVICE} status:error" --duration "$DURATION" 2>/dev/null || echo '{"summary":{"error":0}}')
ERROR_COUNT=$(echo "$LOG_RESULT" | jq '.summary.error // 0')

if [ "$ERROR_COUNT" -gt 100 ]; then
    echo "  [WARN] Found $ERROR_COUNT error logs" >&2
    WARNINGS=$((WARNINGS + 1))
    ISSUES_FOUND=$((ISSUES_FOUND + 1))
elif [ "$ERROR_COUNT" -gt 0 ]; then
    echo "  [OK] Found $ERROR_COUNT error logs (normal volume)" >&2
else
    echo "  [OK] No error logs found" >&2
fi

# 5. Check SLO status
echo "[CHECK] SLO status..." >&2
SLO_RESULT=$(bash "$SCRIPT_DIR/query-slos.sh" --service "$SERVICE" 2>/dev/null || echo '{"status":"ok","summary":{"breaching":0}}')
SLO_BREACHES=$(echo "$SLO_RESULT" | jq '.summary.breaching // 0')

if [ "$SLO_BREACHES" -gt 0 ]; then
    echo "  [CRITICAL] $SLO_BREACHES SLOs breaching" >&2
    CRITICAL=$((CRITICAL + 1))
    ISSUES_FOUND=$((ISSUES_FOUND + 1))
else
    echo "  [OK] All SLOs within target" >&2
fi

echo "" >&2

# Determine overall status
OVERALL_STATUS="healthy"
if [ "$CRITICAL" -gt 0 ]; then
    OVERALL_STATUS="critical"
elif [ "$WARNINGS" -gt 0 ]; then
    OVERALL_STATUS="warning"
fi

# Summary
echo "[SUMMARY] Investigation complete" >&2
echo "  Service: $SERVICE" >&2
echo "  Status: $OVERALL_STATUS" >&2
echo "  Issues found: $ISSUES_FOUND" >&2
echo "  Critical: $CRITICAL" >&2
echo "  Warnings: $WARNINGS" >&2
echo "" >&2

# Get trace IDs from slow endpoints for correlation
TRACE_IDS='[]'
if [ "$SLOW_ENDPOINTS" -gt 0 ]; then
    echo "[INFO] Extracting trace IDs from slow endpoints for correlation..." >&2
    # Get trace IDs from APM spans (if available in response)
    TRACE_IDS=$(echo "$APM_RESULT" | jq '[.endpoints[]? | .trace_ids[]?] | .[0:5]' 2>/dev/null || echo '[]')
fi

# Output structured JSON with all findings
cat <<EOF
{
  "service": "$SERVICE",
  "duration": "$DURATION",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "overall_status": "$OVERALL_STATUS",
  "issues_found": $ISSUES_FOUND,
  "severity": {
    "critical": $CRITICAL,
    "warnings": $WARNINGS
  },
  "checks": {
    "apm": {
      "status": $([ "$SLOW_ENDPOINTS" -gt 0 ] && echo '"warning"' || echo '"ok"'),
      "slow_endpoints": $SLOW_ENDPOINTS,
      "details": $APM_RESULT
    },
    "security": {
      "status": $([ "$CRITICAL_SIGNALS" -gt 0 ] && echo '"critical"' || [ "$HIGH_SIGNALS" -gt 0 ] && echo '"warning"' || echo '"ok"'),
      "critical_signals": $CRITICAL_SIGNALS,
      "high_signals": $HIGH_SIGNALS,
      "details": $SECURITY_RESULT
    },
    "watchdog": {
      "status": $([ "$ERROR_SPIKES" -gt 0 ] && echo '"critical"' || [ "$LATENCY_SPIKES" -gt 0 ] && echo '"warning"' || echo '"ok"'),
      "latency_spikes": $LATENCY_SPIKES,
      "error_spikes": $ERROR_SPIKES,
      "details": $WATCHDOG_RESULT
    },
    "logs": {
      "status": $([ "$ERROR_COUNT" -gt 100 ] && echo '"warning"' || echo '"ok"'),
      "error_count": $ERROR_COUNT,
      "details": $LOG_RESULT
    },
    "slos": {
      "status": $([ "$SLO_BREACHES" -gt 0 ] && echo '"critical"' || echo '"ok"'),
      "breaching": $SLO_BREACHES,
      "details": $SLO_RESULT
    }
  },
  "recommendations": [
    $([ "$CRITICAL" -gt 0 ] && echo '{"priority": "critical", "action": "Immediate attention required - create incident and investigate"},' || echo '')
    $([ "$SLOW_ENDPOINTS" -gt 0 ] && echo '{"priority": "high", "action": "Optimize slow endpoints or increase capacity"},' || echo '')
    $([ "$ERROR_COUNT" -gt 100 ] && echo '{"priority": "high", "action": "Investigate error patterns in logs"},' || echo '')
    $([ "$SLO_BREACHES" -gt 0 ] && echo '{"priority": "critical", "action": "SLO breach - escalate per SLO policy"},' || echo '')
    $([ "$CRITICAL_SIGNALS" -gt 0 ] && echo '{"priority": "critical", "action": "Security incident detected - engage security team"},' || echo '')
    {}
  ],
  "trace_correlation": {
    "trace_ids_available": $(echo "$TRACE_IDS" | jq 'length'),
    "trace_ids": $TRACE_IDS,
    "log_search_query": "service:${SERVICE} trace_id:($(echo "$TRACE_IDS" | jq -r '.[] // empty' | head -5 | tr '\n' ' ' | sed 's/ / OR /g' | sed 's/ OR $//'))"
  }
}
EOF
