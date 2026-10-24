#!/bin/bash
# Continuous Health Monitoring Script
#
# Usage: ./health-monitor.sh [service-name] [interval-seconds]
# Example: ./health-monitor.sh api-service 60
#
# This script continuously monitors service health and alerts on issues.

SERVICE="${1:-}"
INTERVAL="${2:-60}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Alert thresholds
ERROR_RATE_THRESHOLD=5.0
LATENCY_P95_THRESHOLD=500
CRITICAL_LOGS_THRESHOLD=10

echo "=========================================="
echo "Datadog Health Monitor"
echo "=========================================="
echo "Service: ${SERVICE:-auto-detect}"
echo "Interval: ${INTERVAL}s"
echo "Started: $(date)"
echo "Press Ctrl+C to stop"
echo "=========================================="
echo ""

# Track previous state
PREV_STATUS="unknown"
PREV_ERROR_RATE=0
ALERT_COUNT=0

while true; do
    TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')
    echo -e "${BLUE}[$TIMESTAMP]${NC} Checking health..."

    # Get health status
    if [ -n "$SERVICE" ]; then
        HEALTH_DATA=$(dd health "$SERVICE" --json 2>/dev/null)
    else
        HEALTH_DATA=$(dd health --json 2>/dev/null)
    fi

    if [ -z "$HEALTH_DATA" ]; then
        echo -e "${RED}Error: Could not fetch health data. Check credentials.${NC}"
        sleep "$INTERVAL"
        continue
    fi

    # Parse health data
    STATUS=$(echo "$HEALTH_DATA" | jq -r '.status' 2>/dev/null || echo "unknown")
    ERROR_RATE=$(echo "$HEALTH_DATA" | jq -r '.error_rate' 2>/dev/null || echo "0")
    LATENCY_P50=$(echo "$HEALTH_DATA" | jq -r '.latency.p50' 2>/dev/null || echo "0")
    LATENCY_P95=$(echo "$HEALTH_DATA" | jq -r '.latency.p95' 2>/dev/null || echo "0")
    REQUEST_RATE=$(echo "$HEALTH_DATA" | jq -r '.request_rate' 2>/dev/null || echo "0")

    # Display current metrics
    echo "  Status: $STATUS"
    echo "  Error Rate: ${ERROR_RATE}%"
    echo "  Latency P50: ${LATENCY_P50}ms"
    echo "  Latency P95: ${LATENCY_P95}ms"
    echo "  Request Rate: ${REQUEST_RATE} req/s"

    # Check for status change
    if [ "$STATUS" != "$PREV_STATUS" ] && [ "$PREV_STATUS" != "unknown" ]; then
        if [ "$STATUS" = "healthy" ]; then
            echo -e "${GREEN}✅ Service recovered to healthy state${NC}"
            ALERT_COUNT=0
        elif [ "$STATUS" = "degraded" ]; then
            echo -e "${YELLOW}⚠️  Service degraded!${NC}"
            ((ALERT_COUNT++))
        elif [ "$STATUS" = "unhealthy" ]; then
            echo -e "${RED}🚨 Service unhealthy!${NC}"
            ((ALERT_COUNT++))
        fi
    fi

    # Check error rate threshold
    ERROR_RATE_FLOAT=$(echo "$ERROR_RATE" | awk '{print $1}')
    if (( $(echo "$ERROR_RATE_FLOAT > $ERROR_RATE_THRESHOLD" | bc -l) )); then
        echo -e "${RED}🚨 High error rate: ${ERROR_RATE}% (threshold: ${ERROR_RATE_THRESHOLD}%)${NC}"

        # Show recent errors
        echo "  Recent errors:"
        if [ -n "$SERVICE" ]; then
            dd logs --service "$SERVICE" --status error --from 5m --limit 5 2>/dev/null | jq -r '.[] | "    [\(.timestamp)] \(.message)"' 2>/dev/null || echo "    Could not fetch error logs"
        else
            dd logs --status error --from 5m --limit 5 2>/dev/null | jq -r '.[] | "    [\(.timestamp)] \(.message)"' 2>/dev/null || echo "    Could not fetch error logs"
        fi
    fi

    # Check latency threshold
    LATENCY_P95_INT=$(echo "$LATENCY_P95" | awk '{print int($1)}')
    if [ "$LATENCY_P95_INT" -gt "$LATENCY_P95_THRESHOLD" ]; then
        echo -e "${YELLOW}⚠️  High latency (P95): ${LATENCY_P95}ms (threshold: ${LATENCY_P95_THRESHOLD}ms)${NC}"
    fi

    # Check for anomalies
    ANOMALIES=$(dd watchdog --json 2>/dev/null | jq '. | length' 2>/dev/null || echo "0")
    if [ "$ANOMALIES" -gt 0 ]; then
        echo -e "${YELLOW}⚠️  Watchdog detected $ANOMALIES anomal(ies)${NC}"
    fi

    # Check for active incidents
    INCIDENTS=$(dd incidents list --status active --json 2>/dev/null | jq '. | length' 2>/dev/null || echo "0")
    if [ "$INCIDENTS" -gt 0 ]; then
        echo -e "${RED}🚨 $INCIDENTS active incident(s)${NC}"
    fi

    # Alert escalation
    if [ $ALERT_COUNT -ge 3 ]; then
        echo -e "${RED}🚨🚨🚨 PERSISTENT ISSUES DETECTED (${ALERT_COUNT} consecutive alerts)${NC}"
        echo "Consider creating an incident or investigating immediately."
    fi

    # Update previous state
    PREV_STATUS="$STATUS"
    PREV_ERROR_RATE="$ERROR_RATE"

    echo ""
    sleep "$INTERVAL"
done
