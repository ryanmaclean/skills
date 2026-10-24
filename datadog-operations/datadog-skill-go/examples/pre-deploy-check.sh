#!/bin/bash
# Pre-Deployment Safety Check Script
#
# Usage: ./pre-deploy-check.sh [service-name] [environment]
# Example: ./pre-deploy-check.sh api-service production
#
# This script performs comprehensive safety checks before deploying a service.
# Exit codes:
#   0 - Safe to deploy
#   1 - Unsafe to deploy (critical issues)
#   2 - Warning (non-critical issues)

set -e

SERVICE="${1:-}"
ENVIRONMENT="${2:-production}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Counters
CRITICAL_ISSUES=0
WARNINGS=0

echo "=========================================="
echo "Pre-Deployment Safety Check"
echo "=========================================="
echo "Service: ${SERVICE:-auto-detect}"
echo "Environment: ${ENVIRONMENT}"
echo "Time: $(date)"
echo "=========================================="
echo ""

# Function to print colored status
print_status() {
    local status=$1
    local message=$2
    case $status in
        "PASS")
            echo -e "${GREEN}✅ PASS${NC}: $message"
            ;;
        "FAIL")
            echo -e "${RED}❌ FAIL${NC}: $message"
            ((CRITICAL_ISSUES++))
            ;;
        "WARN")
            echo -e "${YELLOW}⚠️  WARN${NC}: $message"
            ((WARNINGS++))
            ;;
        "INFO")
            echo -e "ℹ️  INFO: $message"
            ;;
    esac
}

# Check 1: Service Health
echo "1. Checking service health..."
if [ -n "$SERVICE" ]; then
    if dd health "$SERVICE" --json > /dev/null 2>&1; then
        HEALTH_STATUS=$(dd health "$SERVICE" --json | jq -r '.status' 2>/dev/null || echo "unknown")
        if [ "$HEALTH_STATUS" = "healthy" ]; then
            print_status "PASS" "Service is healthy"
        elif [ "$HEALTH_STATUS" = "degraded" ]; then
            print_status "WARN" "Service is degraded"
        else
            print_status "FAIL" "Service is unhealthy"
        fi
    else
        print_status "WARN" "Could not determine service health"
    fi
else
    if dd health --json > /dev/null 2>&1; then
        print_status "PASS" "Auto-detected service is healthy"
    else
        print_status "WARN" "Service health check failed - check credentials"
    fi
fi
echo ""

# Check 2: Active Incidents
echo "2. Checking for active incidents..."
ACTIVE_INCIDENTS=$(dd incidents list --status active --json 2>/dev/null | jq '. | length' 2>/dev/null || echo "0")
if [ "$ACTIVE_INCIDENTS" -eq 0 ]; then
    print_status "PASS" "No active incidents"
elif [ "$ACTIVE_INCIDENTS" -lt 3 ]; then
    print_status "WARN" "$ACTIVE_INCIDENTS active incident(s) found"
else
    print_status "FAIL" "$ACTIVE_INCIDENTS active incidents - too many to deploy safely"
fi
echo ""

# Check 3: Error Rate
echo "3. Checking error rate (last 30 minutes)..."
if [ -n "$SERVICE" ]; then
    ERROR_RATE=$(dd apm "$SERVICE" --status error --from 30m --json 2>/dev/null | jq -r '.error_rate' 2>/dev/null || echo "0")
else
    ERROR_RATE=$(dd apm --status error --from 30m --json 2>/dev/null | jq -r '.error_rate' 2>/dev/null || echo "0")
fi

# Convert error rate to comparison (handle both integer and float)
ERROR_RATE_INT=$(echo "$ERROR_RATE" | awk '{print int($1)}')

if [ "$ERROR_RATE_INT" -lt 1 ]; then
    print_status "PASS" "Error rate is low ($ERROR_RATE%)"
elif [ "$ERROR_RATE_INT" -lt 5 ]; then
    print_status "WARN" "Error rate is elevated ($ERROR_RATE%)"
else
    print_status "FAIL" "Error rate is too high ($ERROR_RATE%)"
fi
echo ""

# Check 4: Recent Critical Logs
echo "4. Checking for recent critical logs (last 15 minutes)..."
if [ -n "$SERVICE" ]; then
    CRITICAL_LOGS=$(dd logs --service "$SERVICE" --status error --from 15m --json 2>/dev/null | jq '. | length' 2>/dev/null || echo "0")
else
    CRITICAL_LOGS=$(dd logs --status error --from 15m --json 2>/dev/null | jq '. | length' 2>/dev/null || echo "0")
fi

if [ "$CRITICAL_LOGS" -eq 0 ]; then
    print_status "PASS" "No critical logs in last 15 minutes"
elif [ "$CRITICAL_LOGS" -lt 10 ]; then
    print_status "WARN" "$CRITICAL_LOGS critical log(s) found"
else
    print_status "FAIL" "$CRITICAL_LOGS critical logs - investigate before deploying"
fi
echo ""

# Check 5: Monitor Alerts
echo "5. Checking for firing monitors..."
FIRING_MONITORS=$(dd monitors --status alert --json 2>/dev/null | jq '. | length' 2>/dev/null || echo "0")
if [ "$FIRING_MONITORS" -eq 0 ]; then
    print_status "PASS" "No monitors in alert state"
elif [ "$FIRING_MONITORS" -lt 3 ]; then
    print_status "WARN" "$FIRING_MONITORS monitor(s) alerting"
else
    print_status "FAIL" "$FIRING_MONITORS monitors alerting - too many issues"
fi
echo ""

# Check 6: SLO Compliance (if available)
echo "6. Checking SLO compliance..."
if dd slos --json > /dev/null 2>&1; then
    SLO_COMPLIANCE=$(dd slos --json 2>/dev/null | jq -r '.overall_compliance' 2>/dev/null || echo "100")
    SLO_COMPLIANCE_INT=$(echo "$SLO_COMPLIANCE" | awk '{print int($1)}')

    if [ "$SLO_COMPLIANCE_INT" -ge 99 ]; then
        print_status "PASS" "SLO compliance is good ($SLO_COMPLIANCE%)"
    elif [ "$SLO_COMPLIANCE_INT" -ge 95 ]; then
        print_status "WARN" "SLO compliance is marginal ($SLO_COMPLIANCE%)"
    else
        print_status "FAIL" "SLO compliance is poor ($SLO_COMPLIANCE%)"
    fi
else
    print_status "INFO" "SLO data not available"
fi
echo ""

# Check 7: Deployment Safety (CLI-specific check)
echo "7. Running deployment safety check..."
if [ -n "$SERVICE" ]; then
    DEPLOY_CMD="dd deploy $SERVICE --environment $ENVIRONMENT"
else
    DEPLOY_CMD="dd deploy --environment $ENVIRONMENT"
fi

if $DEPLOY_CMD > /dev/null 2>&1; then
    print_status "PASS" "Deployment safety check passed"
else
    EXIT_CODE=$?
    if [ $EXIT_CODE -eq 2 ]; then
        print_status "WARN" "Deployment has warnings"
    else
        print_status "FAIL" "Deployment safety check failed"
    fi
fi
echo ""

# Summary
echo "=========================================="
echo "Summary"
echo "=========================================="
echo "Critical Issues: $CRITICAL_ISSUES"
echo "Warnings: $WARNINGS"
echo ""

# Final decision
if [ $CRITICAL_ISSUES -gt 0 ]; then
    echo -e "${RED}❌ UNSAFE TO DEPLOY${NC}"
    echo "Critical issues must be resolved before deploying."
    exit 1
elif [ $WARNINGS -gt 0 ]; then
    echo -e "${YELLOW}⚠️  DEPLOY WITH CAUTION${NC}"
    echo "Warnings detected. Proceed carefully and monitor closely."
    exit 2
else
    echo -e "${GREEN}✅ SAFE TO DEPLOY${NC}"
    echo "All checks passed. Deployment can proceed."
    exit 0
fi
