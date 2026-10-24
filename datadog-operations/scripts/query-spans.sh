#!/bin/bash
# Query Datadog APM Spans
# Detailed trace analysis and latency investigation

set -e

# Configuration
DD_SITE=${DD_SITE:-datadoghq.com}
API_BASE="https://api.${DD_SITE}"

# Default values
DURATION="1h"
SERVICE=""
OPERATION=""
RESOURCE=""
TAG_FILTER=""
LIMIT=100

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
        --operation)
            OPERATION="$2"
            shift 2
            ;;
        --resource)
            RESOURCE="$2"
            shift 2
            ;;
        --tags)
            TAG_FILTER="$2"
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

# Build query
build_query() {
    local filters=""

    if [ -n "$SERVICE" ]; then
        filters="service:${SERVICE}"
    fi

    if [ -n "$OPERATION" ]; then
        if [ -n "$filters" ]; then
            filters="${filters} AND operation_name:${OPERATION}"
        else
            filters="operation_name:${OPERATION}"
        fi
    fi

    if [ -n "$RESOURCE" ]; then
        if [ -n "$filters" ]; then
            filters="${filters} AND resource_name:${RESOURCE}"
        else
            filters="resource_name:${RESOURCE}"
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

# Query spans using APM
query_spans() {
    # Query span metrics
    local span_query="avg:trace.span.duration{${QUERY}} by {service,operation,resource}"
    local span_response=$(curl -s -G \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        --data-urlencode "query=${span_query}" \
        --data-urlencode "from=${FROM}" \
        --data-urlencode "to=${TO}" \
        "${API_BASE}/api/v1/query")

    # Query span errors
    local error_query="sum:trace.span.errors{${QUERY}} by {service,error.type}"
    local error_response=$(curl -s -G \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        --data-urlencode "query=${error_query}" \
        --data-urlencode "from=${FROM}" \
        --data-urlencode "to=${TO}" \
        "${API_BASE}/api/v1/query")

    # Check for errors
    if echo "$span_response" | jq -e '.errors' > /dev/null 2>&1; then
        span_response='{"series":[]}'
        error_response='{"series":[]}'
    fi

    # Format results
    jq -n \
        --argjson spans "$span_response" \
        --argjson errors "$error_response" \
        --arg query "$QUERY" \
        --arg duration "$DURATION" \
        '{
            status: "success",
            query: $query,
            duration: $duration,
            spans: {
                duration: {
                    metric: "trace.span.duration",
                    series: $spans.series,
                    avg_duration_ns: (if $spans.series then [$spans.series[].pointlist[]?[1]] | add / length else 0 end),
                    p95_duration_ns: (if $spans.series then ([$spans.series[].pointlist[]?[1]] | sort | .[(length * 0.95 | floor)]) else 0 end)
                },
                errors: {
                    metric: "trace.span.errors",
                    series: $errors.series,
                    total_errors: (if $errors.series then [$errors.series[].pointlist[]?[1]] | add else 0 end),
                    error_types: (if $errors.series then [$errors.series[].scope // "" | split(",") | .[] | select(startswith("error.type:")) | split(":")[1]] | unique else [] end)
                }
            },
            summary: {
                span_count: (if $spans.series then ($spans.series | length) else 0 end),
                avg_duration_ms: (if $spans.series then ([$spans.series[].pointlist[]?[1]] | add / length) / 1000000 else 0 end),
                total_errors: (if $errors.series then [$errors.series[].pointlist[]?[1]] | add else 0 end),
                services: (if $spans.series then [$spans.series[].scope // "" | split(",") | .[] | select(startswith("service:")) | split(":")[1]] | unique else [] end)
            }
        }'
}

# Execute query
query_spans
