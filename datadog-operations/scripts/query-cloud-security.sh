#!/bin/bash
# Query Datadog Cloud Security Posture Management (CSPM)
# Analyze cloud security misconfigurations and compliance

set -e

# Configuration
DD_SITE=${DD_SITE:-datadoghq.com}
API_BASE="https://api.${DD_SITE}"

# Default values
DURATION="24h"
SEVERITY="high"
STATUS="open"
RULE_ID=""
LIMIT=100

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --duration)
            DURATION="$2"
            shift 2
            ;;
        --severity)
            SEVERITY="$2"
            shift 2
            ;;
        --status)
            STATUS="$2"
            shift 2
            ;;
        --rule-id)
            RULE_ID="$2"
            shift 2
            ;;
        --limit)
            LIMIT="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1" >&2
            exit 1
            ;;
    esac
done

# Validate credentials
if [ -z "$DD_API_KEY" ] || [ -z "$DD_APP_KEY" ]; then
    echo '{"status":"error","message":"DD_API_KEY and DD_APP_KEY must be set"}' >&2
    exit 1
fi

# Convert duration to timestamps
calculate_time_range() {
    local duration=$1
    local now=$(date +%s)
    local seconds=86400

    if [[ $duration =~ ^([0-9]+)([mhd])$ ]]; then
        local value="${BASH_REMATCH[1]}"
        local unit="${BASH_REMATCH[2]}"

        case $unit in
            m) seconds=$((value * 60)) ;;
            h) seconds=$((value * 3600)) ;;
            d) seconds=$((value * 86400)) ;;
        esac
    fi

    local from=$((now - seconds))
    echo "$from $now"
}

read FROM TO <<< $(calculate_time_range "$DURATION")

# Build query filter
build_query() {
    local query="@type:security_signal"

    if [ -n "$SEVERITY" ]; then
        query="$query @severity:${SEVERITY}"
    fi

    if [ -n "$STATUS" ]; then
        query="$query @workflow.triage.state:${STATUS}"
    fi

    if [ -n "$RULE_ID" ]; then
        query="$query @rule.id:${RULE_ID}"
    fi

    echo "$query"
}

QUERY=$(build_query)

# Query CSPM findings
query_cspm() {
    response=$(curl -s -G \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        --data-urlencode "filter[query]=${QUERY}" \
        --data-urlencode "filter[from]=${FROM}" \
        --data-urlencode "filter[to]=${TO}" \
        --data-urlencode "page[limit]=${LIMIT}" \
        "${API_BASE}/api/v2/security_monitoring/signals")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    # Check if data exists
    if ! echo "$response" | jq -e '.data' > /dev/null 2>&1 || [ "$(echo "$response" | jq '.data | length')" -eq 0 ]; then
        echo "{\"status\":\"no_data\",\"query\":\"$QUERY\",\"message\":\"No CSPM findings\"}"
        exit 0
    fi

    # Parse CSPM findings
    echo "$response" | jq \
        --arg duration "$DURATION" \
        --arg query "$QUERY" \
        '{
            status: "success",
            duration: $duration,
            query: $query,
            cspm_summary: {
                total_findings: (.data | length),
                critical: ([.data[] | select(.attributes.severity == "critical")] | length),
                high: ([.data[] | select(.attributes.severity == "high")] | length),
                medium: ([.data[] | select(.attributes.severity == "medium")] | length),
                low: ([.data[] | select(.attributes.severity == "low")] | length),
                open: ([.data[] | select(.attributes.workflow.triage.state == "open")] | length),
                under_review: ([.data[] | select(.attributes.workflow.triage.state == "under_review")] | length)
            },
            critical_findings: [
                .data[]
                | select(.attributes.severity == "critical" or .attributes.severity == "high")
                | {
                    signal_id: .id,
                    rule_name: .attributes.rule.name,
                    severity: .attributes.severity,
                    status: .attributes.workflow.triage.state,
                    resource: .attributes.custom.resource,
                    cloud_provider: .attributes.custom.cloud_provider,
                    account_id: .attributes.custom.account_id,
                    region: .attributes.custom.region,
                    timestamp: .attributes.timestamp,
                    message: .attributes.message
                }
            ] | sort_by(.timestamp) | reverse | .[0:20],
            by_rule: (
                [.data[] | .attributes.rule.name]
                | group_by(.)
                | map({
                    rule: .[0],
                    count: length
                })
                | sort_by(-.count)
            ),
            by_cloud_provider: (
                [.data[] | .attributes.custom.cloud_provider]
                | group_by(.)
                | map({
                    provider: .[0],
                    count: length
                })
            ),
            by_severity: (
                [.data[] | .attributes.severity]
                | group_by(.)
                | map({
                    severity: .[0],
                    count: length
                })
            ),
            recommendations: [
                (if ([.data[] | select(.attributes.severity == "critical")] | length) > 0
                    then "Address \([.data[] | select(.attributes.severity == "critical")] | length) critical security findings immediately"
                    else null end),
                (if ([.data[] | select(.attributes.workflow.triage.state == "open")] | length) > 10
                    then "High number of open findings - prioritize triage and remediation"
                    else null end),
                "Review compliance framework mappings for affected resources",
                "Implement automated remediation for common misconfigurations"
            ] | map(select(. != null))
        }'
}

# Query CSPM findings
query_cspm
