#!/bin/bash
# Query Datadog Continuous Profiler
# Analyze CPU, memory, and execution profiles

set -e

# Configuration
DD_SITE=${DD_SITE:-datadoghq.com}
API_BASE="https://api.${DD_SITE}"

# Default values
DURATION="1h"
SERVICE=""
ENV="prod"
PROFILE_TYPE="cpu"

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
        --type)
            PROFILE_TYPE="$2"
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

if [ -z "$SERVICE" ]; then
    echo '{"status":"error","message":"--service is required"}' >&2
    exit 1
fi

# Convert duration to timestamps
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

# Query profiling data
query_profiles() {
    # Build query string
    local query="service:${SERVICE} env:${ENV}"

    # Query profiles using the v2 API
    response=$(curl -s -G \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        --data-urlencode "start=${FROM}" \
        --data-urlencode "end=${TO}" \
        --data-urlencode "query=${query}" \
        "${API_BASE}/api/v2/profiling/profiles")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    # Check if data exists
    if ! echo "$response" | jq -e '.data' > /dev/null 2>&1 || [ "$(echo "$response" | jq '.data | length')" -eq 0 ]; then
        echo "{\"status\":\"no_data\",\"service\":\"$SERVICE\",\"message\":\"No profiling data found for this service\"}"
        exit 0
    fi

    # Parse profile data
    echo "$response" | jq \
        --arg service "$SERVICE" \
        --arg env "$ENV" \
        --arg duration "$DURATION" \
        --arg profile_type "$PROFILE_TYPE" \
        '{
            status: "success",
            service: $service,
            environment: $env,
            duration: $duration,
            profile_type: $profile_type,
            profile_summary: {
                total_profiles: (.data | length),
                profile_types: ([.data[].attributes.profile_type] | unique),
                languages: ([.data[].attributes.language] | unique),
                total_samples: ([.data[].attributes.sample_count] | add // 0)
            },
            top_functions_cpu: (
                [.data[]
                | select(.attributes.profile_type == "cpu")
                | .attributes.top_functions[]?
                | {
                    function: .name,
                    self_cpu_percent: .self_value_percentage,
                    total_cpu_percent: .total_value_percentage,
                    file: .filename,
                    line: .line_number
                }]
                | sort_by(-.self_cpu_percent)
                | .[0:10]
            ),
            top_allocations_memory: (
                [.data[]
                | select(.attributes.profile_type == "heap" or .attributes.profile_type == "memory")
                | .attributes.top_functions[]?
                | {
                    function: .name,
                    allocated_bytes: .self_value,
                    allocation_percent: .self_value_percentage,
                    file: .filename,
                    line: .line_number
                }]
                | sort_by(-.allocated_bytes)
                | .[0:10]
            ),
            runtime_info: [
                .data[]
                | {
                    timestamp: .attributes.start_time,
                    language: .attributes.language,
                    runtime_version: .attributes.runtime_version,
                    host: .attributes.host,
                    profile_type: .attributes.profile_type,
                    sample_count: .attributes.sample_count
                }
            ] | .[0:5],
            recommendations: [
                (if ([.data[] | select(.attributes.profile_type == "cpu")] | length) > 0
                    then "Review top CPU-consuming functions for optimization opportunities"
                    else null end),
                (if ([.data[] | select(.attributes.profile_type == "heap" or .attributes.profile_type == "memory")] | length) > 0
                    then "Analyze memory allocation patterns to reduce GC pressure"
                    else null end),
                (if (.data | length) > 100
                    then "High profiling activity detected - consider sampling rate adjustment"
                    else null end)
            ] | map(select(. != null))
        }'
}

# Get profile summary statistics
get_summary() {
    local query="service:${SERVICE} env:${ENV}"

    response=$(curl -s -X POST \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        -H "Content-Type: application/json" \
        -d "{
            \"filter\": {
                \"query\": \"${query}\",
                \"from\": \"${FROM}000\",
                \"to\": \"${TO}000\"
            },
            \"compute\": [
                {\"aggregation\": \"count\"},
                {\"aggregation\": \"sum\", \"metric\": \"@profiling.sample_count\"}
            ],
            \"group_by\": [{\"facet\": \"@profiling.profile_type\"}]
        }" \
        "${API_BASE}/api/v2/profiling/analytics/aggregate")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    echo "$response" | jq \
        --arg service "$SERVICE" \
        '{
            status: "success",
            service: $service,
            summary: {
                by_profile_type: [
                    .data[]
                    | {
                        profile_type: .by["@profiling.profile_type"],
                        profile_count: .compute.c0,
                        total_samples: (.compute.c1 // 0)
                    }
                ]
            }
        }'
}

# Query profiles by default
query_profiles
