#!/bin/bash
# Query Datadog Serverless Monitoring
# Retrieves Lambda, Azure Functions, and Cloud Functions metrics

set -e

# Configuration
DD_SITE=${DD_SITE:-datadoghq.com}
API_BASE="https://api.${DD_SITE}"

# Default values
DURATION="1h"
FUNCTION_NAME=""
REGION=""
RUNTIME=""
TAG_FILTER=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --duration)
            DURATION="$2"
            shift 2
            ;;
        --function|--function-name)
            FUNCTION_NAME="$2"
            shift 2
            ;;
        --region)
            REGION="$2"
            shift 2
            ;;
        --runtime)
            RUNTIME="$2"
            shift 2
            ;;
        --tags)
            TAG_FILTER="$2"
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

# Build query filter
build_query() {
    local filters=""

    if [ -n "$FUNCTION_NAME" ]; then
        filters="functionname:${FUNCTION_NAME}"
    fi

    if [ -n "$REGION" ]; then
        if [ -n "$filters" ]; then
            filters="${filters} AND region:${REGION}"
        else
            filters="region:${REGION}"
        fi
    fi

    if [ -n "$RUNTIME" ]; then
        if [ -n "$filters" ]; then
            filters="${filters} AND runtime:${RUNTIME}"
        else
            filters="runtime:${RUNTIME}"
        fi
    fi

    if [ -n "$TAG_FILTER" ]; then
        if [ -n "$filters" ]; then
            filters="${filters} AND ${TAG_FILTER}"
        else
            filters="${TAG_FILTER}"
        fi
    fi

    if [ -z "$filters" ]; then
        filters="*"
    fi

    echo "$filters"
}

QUERY=$(build_query)

# Query serverless metrics
query_serverless() {
    # Invocations
    local invoc_query="sum:aws.lambda.invocations{${QUERY}} by {functionname,region}.as_count()"
    local invoc_response=$(curl -s -G \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        --data-urlencode "query=${invoc_query}" \
        --data-urlencode "from=${FROM}" \
        --data-urlencode "to=${TO}" \
        "${API_BASE}/api/v1/query")

    # Errors
    local error_query="sum:aws.lambda.errors{${QUERY}} by {functionname,region}.as_count()"
    local error_response=$(curl -s -G \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        --data-urlencode "query=${error_query}" \
        --data-urlencode "from=${FROM}" \
        --data-urlencode "to=${TO}" \
        "${API_BASE}/api/v1/query")

    # Duration
    local duration_query="avg:aws.lambda.duration{${QUERY}} by {functionname,region}"
    local duration_response=$(curl -s -G \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        --data-urlencode "query=${duration_query}" \
        --data-urlencode "from=${FROM}" \
        --data-urlencode "to=${TO}" \
        "${API_BASE}/api/v1/query")

    # Cold starts (enhanced metrics)
    local cold_start_query="sum:aws.lambda.enhanced.invocations{cold_start:true,${QUERY}} by {functionname}.as_count()"
    local cold_start_response=$(curl -s -G \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        --data-urlencode "query=${cold_start_query}" \
        --data-urlencode "from=${FROM}" \
        --data-urlencode "to=${TO}" \
        "${API_BASE}/api/v1/query")

    # Throttles
    local throttle_query="sum:aws.lambda.throttles{${QUERY}} by {functionname,region}.as_count()"
    local throttle_response=$(curl -s -G \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        --data-urlencode "query=${throttle_query}" \
        --data-urlencode "from=${FROM}" \
        --data-urlencode "to=${TO}" \
        "${API_BASE}/api/v1/query")

    # Check for errors
    if echo "$invoc_response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$invoc_response" | jq '{status: "error", message: .errors[0]}'
        exit 1
    fi

    # Combine and format results
    jq -n \
        --argjson invoc "$invoc_response" \
        --argjson error "$error_response" \
        --argjson dur "$duration_response" \
        --argjson cold "$cold_start_response" \
        --argjson throttle "$throttle_response" \
        --arg query "$QUERY" \
        --arg duration "$DURATION" \
        '{
            status: "success",
            query: $query,
            duration: $duration,
            serverless: {
                invocations: {
                    metric: "aws.lambda.invocations",
                    series: $invoc.series,
                    total: (if $invoc.series then [$invoc.series[].pointlist[]?[1]] | add else 0 end)
                },
                errors: {
                    metric: "aws.lambda.errors",
                    series: $error.series,
                    total: (if $error.series then [$error.series[].pointlist[]?[1]] | add else 0 end)
                },
                duration: {
                    metric: "aws.lambda.duration",
                    series: $dur.series,
                    avg_ms: (if $dur.series then [$dur.series[].pointlist[]?[1]] | add / length else 0 end)
                },
                cold_starts: {
                    metric: "aws.lambda.enhanced.invocations (cold_start:true)",
                    series: $cold.series,
                    total: (if $cold.series then [$cold.series[].pointlist[]?[1]] | add else 0 end)
                },
                throttles: {
                    metric: "aws.lambda.throttles",
                    series: $throttle.series,
                    total: (if $throttle.series then [$throttle.series[].pointlist[]?[1]] | add else 0 end)
                }
            },
            summary: {
                function_count: (if $invoc.series then ($invoc.series | length) else 0 end),
                total_invocations: (if $invoc.series then [$invoc.series[].pointlist[]?[1]] | add else 0 end),
                total_errors: (if $error.series then [$error.series[].pointlist[]?[1]] | add else 0 end),
                error_rate: (if $invoc.series and $error.series then
                    ([$error.series[].pointlist[]?[1]] | add) / ([$invoc.series[].pointlist[]?[1]] | add) * 100
                    else 0 end),
                avg_duration_ms: (if $dur.series then [$dur.series[].pointlist[]?[1]] | add / length else 0 end),
                cold_start_count: (if $cold.series then [$cold.series[].pointlist[]?[1]] | add else 0 end),
                throttle_count: (if $throttle.series then [$throttle.series[].pointlist[]?[1]] | add else 0 end)
            }
        }'
}

# Execute query
query_serverless
