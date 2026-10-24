#!/bin/bash
# Incident Response Helper Script
#
# Usage: ./incident-response.sh [incident-id|create]
# Example: ./incident-response.sh 12345
# Example: ./incident-response.sh create
#
# Helps with rapid incident response by gathering diagnostic data.

ACTION="${1:-}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

show_usage() {
    echo "Usage: $0 [incident-id|create]"
    echo ""
    echo "Commands:"
    echo "  create              Create a new incident"
    echo "  <incident-id>       Gather diagnostics for existing incident"
    echo "  list                List active incidents"
    echo ""
    echo "Examples:"
    echo "  $0 create"
    echo "  $0 12345"
    echo "  $0 list"
}

list_active_incidents() {
    echo -e "${BLUE}Active Incidents:${NC}"
    echo ""

    INCIDENTS=$(dd incidents list --status active --json 2>/dev/null)

    if [ -z "$INCIDENTS" ]; then
        echo -e "${RED}Error: Could not fetch incidents${NC}"
        exit 1
    fi

    COUNT=$(echo "$INCIDENTS" | jq '. | length')

    if [ "$COUNT" -eq 0 ]; then
        echo -e "${GREEN}No active incidents ✅${NC}"
        exit 0
    fi

    echo "$INCIDENTS" | jq -r '.[] | "\(.id) | [\(.severity)] \(.title) | \(.status) | Created: \(.created)"'
    echo ""
    echo "Total: $COUNT incident(s)"
}

create_incident() {
    echo -e "${BLUE}Creating New Incident${NC}"
    echo "=========================================="
    echo ""

    # Prompt for title
    read -p "Incident title: " TITLE
    if [ -z "$TITLE" ]; then
        echo -e "${RED}Error: Title is required${NC}"
        exit 1
    fi

    # Prompt for severity
    echo ""
    echo "Severity levels:"
    echo "  1 - SEV-1 (Critical - Service down)"
    echo "  2 - SEV-2 (High - Significant impact)"
    echo "  3 - SEV-3 (Medium - Minor impact)"
    echo "  4 - SEV-4 (Low - Minimal impact)"
    echo "  5 - SEV-5 (Informational)"
    read -p "Severity (1-5): " SEV_NUM

    case $SEV_NUM in
        1) SEVERITY="SEV-1" ;;
        2) SEVERITY="SEV-2" ;;
        3) SEVERITY="SEV-3" ;;
        4) SEVERITY="SEV-4" ;;
        5) SEVERITY="SEV-5" ;;
        *)
            echo -e "${RED}Invalid severity. Using SEV-3${NC}"
            SEVERITY="SEV-3"
            ;;
    esac

    # Prompt for service
    read -p "Service name (optional): " SERVICE

    # Prompt for customer impact
    read -p "Customer impact description: " IMPACT

    echo ""
    echo "Creating incident..."

    # Build create command
    CMD="dd incidents create --title \"$TITLE\" --severity $SEVERITY"
    [ -n "$SERVICE" ] && CMD="$CMD --service $SERVICE"
    [ -n "$IMPACT" ] && CMD="$CMD --customer-impact \"$IMPACT\""

    # Create incident
    if eval "$CMD" > /tmp/incident-create.json 2>&1; then
        INCIDENT_ID=$(jq -r '.id' /tmp/incident-create.json 2>/dev/null)
        echo -e "${GREEN}✅ Incident created: $INCIDENT_ID${NC}"
        echo ""

        # Automatically gather diagnostics
        echo "Gathering diagnostic data..."
        gather_diagnostics "$INCIDENT_ID" "$SERVICE"
    else
        echo -e "${RED}Error: Failed to create incident${NC}"
        cat /tmp/incident-create.json
        exit 1
    fi
}

gather_diagnostics() {
    local INCIDENT_ID=$1
    local SERVICE=$2

    echo ""
    echo -e "${BLUE}Gathering Diagnostic Data${NC}"
    echo "=========================================="
    echo "Incident ID: $INCIDENT_ID"
    echo "Service: ${SERVICE:-auto-detect}"
    echo ""

    OUTPUT_DIR="incident-$INCIDENT_ID-$(date +%Y%m%d-%H%M%S)"
    mkdir -p "$OUTPUT_DIR"

    echo "Output directory: $OUTPUT_DIR"
    echo ""

    # 1. Service Health
    echo -e "${BLUE}[1/8]${NC} Checking service health..."
    if [ -n "$SERVICE" ]; then
        dd health "$SERVICE" --json > "$OUTPUT_DIR/health.json" 2>&1
    else
        dd health --json > "$OUTPUT_DIR/health.json" 2>&1
    fi
    [ $? -eq 0 ] && echo "  ✓ Saved to health.json" || echo "  ✗ Failed"

    # 2. Recent Error Traces
    echo -e "${BLUE}[2/8]${NC} Fetching recent error traces (last 30 minutes)..."
    if [ -n "$SERVICE" ]; then
        dd apm "$SERVICE" --status error --from 30m --json > "$OUTPUT_DIR/error-traces.json" 2>&1
    else
        dd apm --status error --from 30m --json > "$OUTPUT_DIR/error-traces.json" 2>&1
    fi
    [ $? -eq 0 ] && echo "  ✓ Saved to error-traces.json" || echo "  ✗ Failed"

    # 3. Error Logs
    echo -e "${BLUE}[3/8]${NC} Fetching error logs (last 30 minutes)..."
    if [ -n "$SERVICE" ]; then
        dd logs --service "$SERVICE" --status error --from 30m --limit 200 --json > "$OUTPUT_DIR/error-logs.json" 2>&1
    else
        dd logs --status error --from 30m --limit 200 --json > "$OUTPUT_DIR/error-logs.json" 2>&1
    fi
    [ $? -eq 0 ] && echo "  ✓ Saved to error-logs.json" || echo "  ✗ Failed"

    # 4. Critical Logs
    echo -e "${BLUE}[4/8]${NC} Fetching critical logs (last hour)..."
    if [ -n "$SERVICE" ]; then
        dd logs --service "$SERVICE" --query "status:critical" --from 1h --limit 100 --json > "$OUTPUT_DIR/critical-logs.json" 2>&1
    else
        dd logs --query "status:critical" --from 1h --limit 100 --json > "$OUTPUT_DIR/critical-logs.json" 2>&1
    fi
    [ $? -eq 0 ] && echo "  ✓ Saved to critical-logs.json" || echo "  ✗ Failed"

    # 5. Watchdog Anomalies
    echo -e "${BLUE}[5/8]${NC} Checking Watchdog for anomalies..."
    if [ -n "$SERVICE" ]; then
        dd watchdog --service "$SERVICE" --from 1h --json > "$OUTPUT_DIR/watchdog-anomalies.json" 2>&1
    else
        dd watchdog --from 1h --json > "$OUTPUT_DIR/watchdog-anomalies.json" 2>&1
    fi
    [ $? -eq 0 ] && echo "  ✓ Saved to watchdog-anomalies.json" || echo "  ✗ Failed"

    # 6. Monitor Status
    echo -e "${BLUE}[6/8]${NC} Checking monitor alerts..."
    dd monitors --status alert --json > "$OUTPUT_DIR/monitor-alerts.json" 2>&1
    [ $? -eq 0 ] && echo "  ✓ Saved to monitor-alerts.json" || echo "  ✗ Failed"

    # 7. Recent Deployments
    echo -e "${BLUE}[7/8]${NC} Checking recent deployments (last 24 hours)..."
    if [ -n "$SERVICE" ]; then
        dd cicd "$SERVICE" --from 24h --json > "$OUTPUT_DIR/recent-deployments.json" 2>&1
    else
        dd cicd --from 24h --json > "$OUTPUT_DIR/recent-deployments.json" 2>&1
    fi
    [ $? -eq 0 ] && echo "  ✓ Saved to recent-deployments.json" || echo "  ✗ Failed"

    # 8. Incident Details
    echo -e "${BLUE}[8/8]${NC} Fetching incident details..."
    dd incidents list --json 2>&1 | jq ".[] | select(.id == \"$INCIDENT_ID\")" > "$OUTPUT_DIR/incident-details.json" 2>&1
    [ $? -eq 0 ] && echo "  ✓ Saved to incident-details.json" || echo "  ✗ Failed"

    echo ""

    # Generate summary report
    echo "Generating incident summary..."

    cat > "$OUTPUT_DIR/INCIDENT-SUMMARY.md" << EOF
# Incident Response - $INCIDENT_ID

**Generated:** $(date '+%Y-%m-%d %H:%M:%S')
**Incident ID:** $INCIDENT_ID
**Service:** ${SERVICE:-auto-detect}

---

## Incident Details

EOF

    # Add incident details
    if [ -f "$OUTPUT_DIR/incident-details.json" ]; then
        TITLE=$(jq -r '.title // "N/A"' "$OUTPUT_DIR/incident-details.json")
        SEVERITY=$(jq -r '.severity // "N/A"' "$OUTPUT_DIR/incident-details.json")
        STATUS=$(jq -r '.status // "N/A"' "$OUTPUT_DIR/incident-details.json")
        CREATED=$(jq -r '.created // "N/A"' "$OUTPUT_DIR/incident-details.json")

        cat >> "$OUTPUT_DIR/INCIDENT-SUMMARY.md" << EOF
- **Title:** $TITLE
- **Severity:** $SEVERITY
- **Status:** $STATUS
- **Created:** $CREATED

---

## Health Status

EOF
    fi

    # Add health summary
    if [ -f "$OUTPUT_DIR/health.json" ]; then
        HEALTH_STATUS=$(jq -r '.status // "N/A"' "$OUTPUT_DIR/health.json")
        ERROR_RATE=$(jq -r '.error_rate // "N/A"' "$OUTPUT_DIR/health.json")
        LATENCY_P95=$(jq -r '.latency.p95 // "N/A"' "$OUTPUT_DIR/health.json")

        cat >> "$OUTPUT_DIR/INCIDENT-SUMMARY.md" << EOF
- **Status:** $HEALTH_STATUS
- **Error Rate:** ${ERROR_RATE}%
- **P95 Latency:** ${LATENCY_P95}ms

---

## Recent Errors

EOF

        # Top error messages
        if [ -f "$OUTPUT_DIR/error-logs.json" ]; then
            echo "**Top Error Messages:**" >> "$OUTPUT_DIR/INCIDENT-SUMMARY.md"
            echo "" >> "$OUTPUT_DIR/INCIDENT-SUMMARY.md"
            jq -r '.[].message' "$OUTPUT_DIR/error-logs.json" | sort | uniq -c | sort -rn | head -5 | while read count msg; do
                echo "- [$count] $msg" >> "$OUTPUT_DIR/INCIDENT-SUMMARY.md"
            done
            echo "" >> "$OUTPUT_DIR/INCIDENT-SUMMARY.md"
        fi
    fi

    cat >> "$OUTPUT_DIR/INCIDENT-SUMMARY.md" << EOF
---

## Collected Data

- \`health.json\` - Current service health
- \`error-traces.json\` - Recent error traces (30m)
- \`error-logs.json\` - Error logs (30m)
- \`critical-logs.json\` - Critical logs (1h)
- \`watchdog-anomalies.json\` - Watchdog alerts
- \`monitor-alerts.json\` - Firing monitors
- \`recent-deployments.json\` - Recent deployments (24h)
- \`incident-details.json\` - Incident metadata

---

## Next Steps

1. Review error logs for patterns
2. Check Watchdog anomalies for root cause hints
3. Correlate with recent deployments
4. Update incident timeline with findings
5. Communicate with stakeholders

**Datadog Links:**
- [Incident Details](https://app.datadoghq.com/incidents/$INCIDENT_ID)
$([ -n "$SERVICE" ] && echo "- [APM Service](https://app.datadoghq.com/apm/services/$SERVICE)")
- [Logs Explorer](https://app.datadoghq.com/logs)
EOF

    echo -e "${GREEN}✅ Diagnostics collected in: $OUTPUT_DIR/${NC}"
    echo ""
    echo "Summary report: $OUTPUT_DIR/INCIDENT-SUMMARY.md"
    echo ""
    echo "Review the data and update the incident:"
    echo "  dd incidents update $INCIDENT_ID --status investigating"
    echo ""
}

# Main logic
if [ -z "$ACTION" ]; then
    show_usage
    exit 1
fi

case "$ACTION" in
    create)
        create_incident
        ;;
    list)
        list_active_incidents
        ;;
    help|--help|-h)
        show_usage
        exit 0
        ;;
    *)
        # Assume it's an incident ID
        if [[ "$ACTION" =~ ^[0-9]+$ ]]; then
            gather_diagnostics "$ACTION" ""
        else
            echo -e "${RED}Error: Invalid command or incident ID${NC}"
            echo ""
            show_usage
            exit 1
        fi
        ;;
esac
