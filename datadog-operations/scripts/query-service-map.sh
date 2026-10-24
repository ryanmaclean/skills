#!/bin/bash
# Query Datadog Service Map
# Analyze service dependencies, topology, and call patterns

set -e

# Configuration
DD_SITE=${DD_SITE:-datadoghq.com}
API_BASE="https://api.${DD_SITE}"

# Default values
DURATION="1h"
SERVICE=""
ENV="prod"

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

# Convert duration to epoch timestamps
calculate_time_range() {
    local duration=$1
    local now=$(date +%s)
    local seconds=3600

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

# Query service map using APM service dependencies
query_service_map() {
    local filter="env:${ENV}"
    if [ -n "$SERVICE" ]; then
        filter="${filter},service:${SERVICE}"
    fi

    # Query service dependencies
    local dep_query="sum:trace.${SERVICE}.hits{${filter}} by {service,resource}"
    local dep_response=$(curl -s -G \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        --data-urlencode "query=${dep_query}" \
        --data-urlencode "from=${FROM}" \
        --data-urlencode "to=${TO}" \
        "${API_BASE}/api/v1/query")

    # Query service call patterns
    local call_query="avg:trace.${SERVICE}.duration{${filter}} by {service,resource}"
    local call_response=$(curl -s -G \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        --data-urlencode "query=${call_query}" \
        --data-urlencode "from=${FROM}" \
        --data-urlencode "to=${TO}" \
        "${API_BASE}/api/v1/query")

    # Get services list for the environment
    local services_response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v1/services")

    # Check for errors
    if echo "$dep_response" | jq -e '.errors' > /dev/null 2>&1; then
        dep_response='{"series":[]}'
    fi

    # Build service map
    jq -n \
        --argjson deps "$dep_response" \
        --argjson calls "$call_response" \
        --argjson services "$services_response" \
        --arg filter "$filter" \
        --arg duration "$DURATION" \
        '{
            status: "success",
            filter: $filter,
            duration: $duration,
            service_map: {
                dependencies: {
                    series: $deps.series,
                    total_calls: (if $deps.series then [$deps.series[].pointlist[]?[1]] | add else 0 end),
                    services_involved: (if $deps.series then ($deps.series | length) else 0 end)
                },
                call_patterns: {
                    series: $calls.series,
                    avg_duration_ms: (if $calls.series then [$calls.series[].pointlist[]?[1]] | add / length else 0 end)
                },
                topology: {
                    total_services: ($services | length),
                    services: [$services[] | select(.env == $filter or true) | {
                        name: .name,
                        type: .type,
                        last_seen: .last_seen
                    }]
                }
            },
            summary: {
                service_count: (if $deps.series then ($deps.series | length) else 0 end),
                total_dependencies: (if $deps.series then ($deps.series | length) else 0 end),
                avg_call_duration_ms: (if $calls.series then [$calls.series[].pointlist[]?[1]] | add / length else 0 end),
                total_calls: (if $deps.series then [$deps.series[].pointlist[]?[1]] | add else 0 end)
            }
        }'
}

# Execute query
query_service_map
