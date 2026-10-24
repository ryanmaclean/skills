#!/bin/bash
# Query Datadog Error Tracking
# Analyze aggregated error issues across APM, logs, and RUM

set -e

# Configuration
DD_SITE=${DD_SITE:-datadoghq.com}
API_BASE="https://api.${DD_SITE}"

# Default values
DURATION="24h"
SERVICE=""
ENV="prod"
STATUS="open"
LIMIT=50

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --duration)
            DURATION="$2"
            shift 2
            ;;
        --service)
            SERVICE="$2"
            shift 2
            ;;
        --env)
            ENV="$2"
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
    local query="*"

    if [ -n "$SERVICE" ]; then
        query="$query service:${SERVICE}"
    fi

    if [ -n "$ENV" ]; then
        query="$query env:${ENV}"
    fi

    if [ -n "$STATUS" ]; then
        query="$query status:${STATUS}"
    fi

    echo "$query"
}

QUERY=$(build_query)

# Query error tracking issues
query_issues() {
    # Use Error Tracking API
    response=$(curl -s -G \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        --data-urlencode "query=${QUERY}" \
        --data-urlencode "start=${FROM}" \
        --data-urlencode "end=${TO}" \
        --data-urlencode "limit=${LIMIT}" \
        --data-urlencode "sort=occurrences" \
        "${API_BASE}/api/v2/error-tracking/issues")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    # Check if data exists
    if ! echo "$response" | jq -e '.data' > /dev/null 2>&1 || [ "$(echo "$response" | jq '.data | length')" -eq 0 ]; then
        echo "{\"status\":\"no_data\",\"query\":\"$QUERY\",\"message\":\"No error tracking issues found\"}"
        exit 0
    fi

    # Parse error issues
    echo "$response" | jq \
        --arg duration "$DURATION" \
        --arg query "$QUERY" \
        --arg env "$ENV" \
        '{
            status: "success",
            duration: $duration,
            environment: $env,
            query: $query,
            error_summary: {
                total_issues: (.data | length),
                total_occurrences: ([.data[].attributes.count] | add // 0),
                impacted_users: ([.data[].attributes.impacted_users] | add // 0),
                open_issues: ([.data[] | select(.attributes.status == "open")] | length),
                resolved_issues: ([.data[] | select(.attributes.status == "resolved")] | length),
                ignored_issues: ([.data[] | select(.attributes.status == "ignored")] | length)
            },
            top_issues_by_frequency: [
                .data[]
                | {
                    issue_id: .id,
                    error_type: .attributes.error_type,
                    message: .attributes.message,
                    occurrences: .attributes.count,
                    impacted_users: .attributes.impacted_users,
                    status: .attributes.status,
                    first_seen: .attributes.first_seen_at,
                    last_seen: .attributes.last_seen_at,
                    service: .attributes.service,
                    environment: .attributes.environment,
                    culprit: .attributes.culprit
                }
            ] | sort_by(-.occurrences) | .[0:10],
            top_issues_by_impact: [
                .data[]
                | {
                    issue_id: .id,
                    error_type: .attributes.error_type,
                    message: .attributes.message,
                    impacted_users: .attributes.impacted_users,
                    occurrences: .attributes.count,
                    status: .attributes.status,
                    service: .attributes.service
                }
            ] | sort_by(-.impacted_users) | .[0:10],
            recent_issues: [
                .data[]
                | select(.attributes.first_seen_at > (now - 86400))
                | {
                    issue_id: .id,
                    error_type: .attributes.error_type,
                    message: .attributes.message,
                    occurrences: .attributes.count,
                    first_seen: .attributes.first_seen_at,
                    service: .attributes.service
                }
            ] | sort_by(-.first_seen_at) | .[0:10],
            by_service: (
                [.data[] | .attributes.service]
                | group_by(.)
                | map({
                    service: .[0],
                    issue_count: length,
                    total_occurrences: ([.[] | .attributes.count] | add // 0)
                })
                | sort_by(-.issue_count)
            ),
            by_error_type: (
                [.data[] | .attributes.error_type]
                | group_by(.)
                | map({
                    error_type: .[0],
                    count: length
                })
                | sort_by(-.count)
            ),
            recommendations: [
                (if ([.data[] | select(.attributes.status == "open")] | length) > 10
                    then "High number of open issues (\([.data[] | select(.attributes.status == "open")] | length)) - prioritize triage"
                    else null end),
                (if ([.data[] | select(.attributes.count > 100)] | length) > 5
                    then "Multiple high-frequency errors detected - investigate common patterns"
                    else null end),
                (if ([.data[] | select(.attributes.impacted_users > 100)] | length) > 0
                    then "High user impact detected - prioritize issues affecting many users"
                    else null end),
                (if ([.data[] | select(.attributes.first_seen_at > (now - 3600))] | length) > 5
                    then "Spike in new errors in last hour - possible recent deployment issue"
                    else null end)
            ] | map(select(. != null))
        }'
}

# Get issue details
get_issue() {
    if [ -z "$1" ]; then
        echo '{"status":"error","message":"Issue ID required"}' >&2
        exit 1
    fi

    local issue_id="$1"

    response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v2/error-tracking/issues/${issue_id}")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    echo "$response" | jq '{
        status: "success",
        issue: {
            id: .data.id,
            error_type: .data.attributes.error_type,
            message: .data.attributes.message,
            occurrences: .data.attributes.count,
            impacted_users: .data.attributes.impacted_users,
            status: .data.attributes.status,
            first_seen: .data.attributes.first_seen_at,
            last_seen: .data.attributes.last_seen_at,
            service: .data.attributes.service,
            environment: .data.attributes.environment,
            culprit: .data.attributes.culprit,
            stack_trace: .data.attributes.stack_trace,
            tags: .data.attributes.tags
        }
    }'
}

# Query issues by default
query_issues
