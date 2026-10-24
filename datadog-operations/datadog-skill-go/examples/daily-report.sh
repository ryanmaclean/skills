#!/bin/bash
# Daily Health Report Generator
#
# Usage: ./daily-report.sh [service-name] [output-file]
# Example: ./daily-report.sh api-service daily-report.md
#
# Generates a comprehensive daily health report in Markdown format.

SERVICE="${1:-}"
OUTPUT="${2:-daily-report-$(date +%Y-%m-%d).md}"

# Colors for terminal output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}Generating daily health report...${NC}"
echo "Service: ${SERVICE:-auto-detect}"
echo "Output: $OUTPUT"
echo ""

# Start report
cat > "$OUTPUT" << EOF
# Daily Health Report

**Generated:** $(date '+%Y-%m-%d %H:%M:%S')
**Service:** ${SERVICE:-auto-detect}

---

## Executive Summary

EOF

# Get overall health status
echo -e "${BLUE}Checking service health...${NC}"
if [ -n "$SERVICE" ]; then
    HEALTH_JSON=$(dd health "$SERVICE" --json 2>/dev/null)
else
    HEALTH_JSON=$(dd health --json 2>/dev/null)
fi

if [ -n "$HEALTH_JSON" ]; then
    STATUS=$(echo "$HEALTH_JSON" | jq -r '.status')
    ERROR_RATE=$(echo "$HEALTH_JSON" | jq -r '.error_rate')
    REQUEST_RATE=$(echo "$HEALTH_JSON" | jq -r '.request_rate')
    LATENCY_P95=$(echo "$HEALTH_JSON" | jq -r '.latency.p95')

    cat >> "$OUTPUT" << EOF
**Overall Status:** $STATUS
**Error Rate:** ${ERROR_RATE}%
**Request Rate:** ${REQUEST_RATE} req/s
**Latency (P95):** ${LATENCY_P95}ms

EOF
else
    cat >> "$OUTPUT" << EOF
**Overall Status:** Unable to fetch health data

EOF
fi

# Service Health Details
cat >> "$OUTPUT" << EOF
---

## Service Health

### APM Metrics (Last 24 Hours)

EOF

echo -e "${BLUE}Fetching APM metrics...${NC}"
if [ -n "$SERVICE" ]; then
    APM_JSON=$(dd apm "$SERVICE" --from 24h --json 2>/dev/null)
else
    APM_JSON=$(dd apm --from 24h --json 2>/dev/null)
fi

if [ -n "$APM_JSON" ]; then
    cat >> "$OUTPUT" << EOF
- **Total Requests:** $(echo "$APM_JSON" | jq -r '.total_requests // "N/A"')
- **Request Rate:** $(echo "$APM_JSON" | jq -r '.request_rate // "N/A"') req/s
- **Error Rate:** $(echo "$APM_JSON" | jq -r '.error_rate // "N/A"')%
- **Average Latency:** $(echo "$APM_JSON" | jq -r '.latency.avg // "N/A"')ms
- **P50 Latency:** $(echo "$APM_JSON" | jq -r '.latency.p50 // "N/A"')ms
- **P95 Latency:** $(echo "$APM_JSON" | jq -r '.latency.p95 // "N/A"')ms
- **P99 Latency:** $(echo "$APM_JSON" | jq -r '.latency.p99 // "N/A"')ms

EOF
else
    echo "- APM data unavailable" >> "$OUTPUT"
    echo "" >> "$OUTPUT"
fi

# Error Analysis
cat >> "$OUTPUT" << EOF
### Error Analysis

EOF

echo -e "${BLUE}Analyzing errors...${NC}"
if [ -n "$SERVICE" ]; then
    ERROR_LOGS=$(dd logs --service "$SERVICE" --status error --from 24h --limit 100 --json 2>/dev/null)
else
    ERROR_LOGS=$(dd logs --status error --from 24h --limit 100 --json 2>/dev/null)
fi

if [ -n "$ERROR_LOGS" ]; then
    ERROR_COUNT=$(echo "$ERROR_LOGS" | jq '. | length')
    cat >> "$OUTPUT" << EOF
**Total Errors (24h):** $ERROR_COUNT

**Top Error Messages:**

EOF

    echo "$ERROR_LOGS" | jq -r '.[] | .message' | sort | uniq -c | sort -rn | head -5 | while read count msg; do
        echo "- [$count] $msg" >> "$OUTPUT"
    done
    echo "" >> "$OUTPUT"
else
    echo "No error data available" >> "$OUTPUT"
    echo "" >> "$OUTPUT"
fi

# Incidents
cat >> "$OUTPUT" << EOF
---

## Incidents

### Active Incidents

EOF

echo -e "${BLUE}Checking incidents...${NC}"
ACTIVE_INCIDENTS=$(dd incidents list --status active --json 2>/dev/null)

if [ -n "$ACTIVE_INCIDENTS" ]; then
    ACTIVE_COUNT=$(echo "$ACTIVE_INCIDENTS" | jq '. | length')
    if [ "$ACTIVE_COUNT" -gt 0 ]; then
        echo "**Count:** $ACTIVE_COUNT" >> "$OUTPUT"
        echo "" >> "$OUTPUT"
        echo "$ACTIVE_INCIDENTS" | jq -r '.[] | "- [\(.severity)] \(.title) (Status: \(.status))"' >> "$OUTPUT"
    else
        echo "No active incidents ✅" >> "$OUTPUT"
    fi
else
    echo "Unable to fetch incident data" >> "$OUTPUT"
fi
echo "" >> "$OUTPUT"

# Resolved incidents from yesterday
cat >> "$OUTPUT" << EOF
### Recently Resolved Incidents (24h)

EOF

RESOLVED_INCIDENTS=$(dd incidents list --status resolved --from 24h --json 2>/dev/null)
if [ -n "$RESOLVED_INCIDENTS" ]; then
    RESOLVED_COUNT=$(echo "$RESOLVED_INCIDENTS" | jq '. | length')
    if [ "$RESOLVED_COUNT" -gt 0 ]; then
        echo "**Count:** $RESOLVED_COUNT" >> "$OUTPUT"
        echo "" >> "$OUTPUT"
        echo "$RESOLVED_INCIDENTS" | jq -r '.[] | "- [\(.severity)] \(.title)"' >> "$OUTPUT"
    else
        echo "No incidents resolved in last 24h" >> "$OUTPUT"
    fi
else
    echo "Unable to fetch resolved incidents" >> "$OUTPUT"
fi
echo "" >> "$OUTPUT"

# SLO Status
cat >> "$OUTPUT" << EOF
---

## SLO Compliance

EOF

echo -e "${BLUE}Checking SLO compliance...${NC}"
SLO_JSON=$(dd slos --json 2>/dev/null)

if [ -n "$SLO_JSON" ]; then
    cat >> "$OUTPUT" << EOF
**Overall Compliance:** $(echo "$SLO_JSON" | jq -r '.overall_compliance // "N/A"')%

**Individual SLOs:**

EOF

    echo "$SLO_JSON" | jq -r '.slos[]? | "- **\(.name)**: \(.compliance)% (\(.status))"' >> "$OUTPUT" 2>/dev/null || echo "- No SLO data available" >> "$OUTPUT"
else
    echo "SLO data unavailable" >> "$OUTPUT"
fi
echo "" >> "$OUTPUT"

# Monitor Status
cat >> "$OUTPUT" << EOF
---

## Monitor Status

EOF

echo -e "${BLUE}Checking monitors...${NC}"
ALERTING_MONITORS=$(dd monitors --status alert --json 2>/dev/null)

if [ -n "$ALERTING_MONITORS" ]; then
    ALERT_COUNT=$(echo "$ALERTING_MONITORS" | jq '. | length')
    cat >> "$OUTPUT" << EOF
**Currently Alerting:** $ALERT_COUNT

EOF

    if [ "$ALERT_COUNT" -gt 0 ]; then
        echo "$ALERTING_MONITORS" | jq -r '.[] | "- \(.name) (\(.severity))"' >> "$OUTPUT"
    else
        echo "No monitors currently alerting ✅" >> "$OUTPUT"
    fi
else
    echo "Unable to fetch monitor data" >> "$OUTPUT"
fi
echo "" >> "$OUTPUT"

# Watchdog Anomalies
cat >> "$OUTPUT" << EOF
---

## Watchdog Anomalies (24h)

EOF

echo -e "${BLUE}Checking Watchdog...${NC}"
if [ -n "$SERVICE" ]; then
    WATCHDOG_JSON=$(dd watchdog --service "$SERVICE" --from 24h --json 2>/dev/null)
else
    WATCHDOG_JSON=$(dd watchdog --from 24h --json 2>/dev/null)
fi

if [ -n "$WATCHDOG_JSON" ]; then
    ANOMALY_COUNT=$(echo "$WATCHDOG_JSON" | jq '. | length')
    cat >> "$OUTPUT" << EOF
**Anomalies Detected:** $ANOMALY_COUNT

EOF

    if [ "$ANOMALY_COUNT" -gt 0 ]; then
        echo "$WATCHDOG_JSON" | jq -r '.[] | "- \(.type): \(.description)"' >> "$OUTPUT" 2>/dev/null || echo "- Anomaly details unavailable" >> "$OUTPUT"
    else
        echo "No anomalies detected ✅" >> "$OUTPUT"
    fi
else
    echo "Watchdog data unavailable" >> "$OUTPUT"
fi
echo "" >> "$OUTPUT"

# Deployment Activity
cat >> "$OUTPUT" << EOF
---

## Deployment Activity (7 days)

EOF

echo -e "${BLUE}Checking deployment activity...${NC}"
if [ -n "$SERVICE" ]; then
    CICD_JSON=$(dd cicd "$SERVICE" --from 7d --json 2>/dev/null)
else
    CICD_JSON=$(dd cicd --from 7d --json 2>/dev/null)
fi

if [ -n "$CICD_JSON" ]; then
    DEPLOY_COUNT=$(echo "$CICD_JSON" | jq '.total_deployments // 0')
    SUCCESS_RATE=$(echo "$CICD_JSON" | jq '.success_rate // 0')

    cat >> "$OUTPUT" << EOF
**Total Deployments:** $DEPLOY_COUNT
**Success Rate:** ${SUCCESS_RATE}%

EOF
else
    echo "Deployment data unavailable" >> "$OUTPUT"
    echo "" >> "$OUTPUT"
fi

# Cost Analysis
cat >> "$OUTPUT" << EOF
---

## Cost Analysis (30 days)

EOF

echo -e "${BLUE}Fetching cost data...${NC}"
COST_JSON=$(dd cost --from 30d --json 2>/dev/null)

if [ -n "$COST_JSON" ]; then
    cat >> "$OUTPUT" << EOF
**Total Cost:** \$$(echo "$COST_JSON" | jq -r '.total_cost // "N/A"')
**Daily Average:** \$$(echo "$COST_JSON" | jq -r '.daily_average // "N/A"')

**Cost by Service:**

EOF

    echo "$COST_JSON" | jq -r '.by_service[]? | "- \(.name): $\(.cost)"' >> "$OUTPUT" 2>/dev/null || echo "- Cost breakdown unavailable" >> "$OUTPUT"
else
    echo "Cost data unavailable" >> "$OUTPUT"
fi
echo "" >> "$OUTPUT"

# Recommendations
cat >> "$OUTPUT" << EOF
---

## Recommendations

EOF

echo -e "${BLUE}Generating recommendations...${NC}"

# Check health status for recommendations
if [ "$STATUS" = "degraded" ] || [ "$STATUS" = "unhealthy" ]; then
    echo "- 🔴 **Critical:** Service health is $STATUS - immediate investigation required" >> "$OUTPUT"
fi

# Check error rate
ERROR_RATE_FLOAT=$(echo "$ERROR_RATE" | awk '{print $1}' 2>/dev/null || echo "0")
if (( $(echo "$ERROR_RATE_FLOAT > 5.0" | bc -l 2>/dev/null || echo "0") )); then
    echo "- 🟡 **Action Needed:** Error rate is elevated (${ERROR_RATE}%) - review error logs" >> "$OUTPUT"
fi

# Check latency
LATENCY_P95_INT=$(echo "$LATENCY_P95" | awk '{print int($1)}' 2>/dev/null || echo "0")
if [ "$LATENCY_P95_INT" -gt 1000 ]; then
    echo "- 🟡 **Performance:** P95 latency is high (${LATENCY_P95}ms) - investigate slow queries" >> "$OUTPUT"
fi

# Check active incidents
if [ "$ACTIVE_COUNT" -gt 0 ]; then
    echo "- 🔴 **Incidents:** $ACTIVE_COUNT active incident(s) - ensure proper resolution" >> "$OUTPUT"
fi

# Check monitors
if [ "$ALERT_COUNT" -gt 3 ]; then
    echo "- 🟡 **Monitoring:** $ALERT_COUNT monitors alerting - review alert thresholds" >> "$OUTPUT"
fi

# If everything is good
if [ "$STATUS" = "healthy" ] && [ "$ERROR_RATE_FLOAT" = "0" ] && [ "$ACTIVE_COUNT" -eq 0 ] && [ "$ALERT_COUNT" -eq 0 ]; then
    echo "- ✅ **All Good:** Service is healthy with no issues detected" >> "$OUTPUT"
fi

echo "" >> "$OUTPUT"

# Footer
cat >> "$OUTPUT" << EOF
---

## Report Metadata

- **Generated By:** Datadog CLI
- **Report Date:** $(date '+%Y-%m-%d %H:%M:%S')
- **Service:** ${SERVICE:-auto-detect}
- **CLI Version:** $(dd version 2>/dev/null | grep -o 'v[0-9.]*' || echo "unknown")

**View in Datadog:**
$([ -n "$SERVICE" ] && echo "- [APM Service](https://app.datadoghq.com/apm/services/$SERVICE)" || echo "- APM Service: N/A")
- [Logs Explorer](https://app.datadoghq.com/logs)
- [Infrastructure](https://app.datadoghq.com/infrastructure)

---

*This report is automatically generated. For real-time monitoring, visit [Datadog](https://app.datadoghq.com).*
EOF

echo ""
echo -e "${GREEN}✅ Report generated: $OUTPUT${NC}"
echo ""
echo "Preview:"
head -20 "$OUTPUT"
echo "..."
