#!/bin/bash
# Query Datadog Container Monitoring
# Retrieves Docker and Kubernetes container metrics and performance data

set -e

# Configuration
DD_SITE=${DD_SITE:-datadoghq.com}
API_BASE="https://api.${DD_SITE}"

# Default values
DURATION="1h"
CONTAINER_ID=""
IMAGE=""
TAG_FILTER=""
LIMIT=100

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --duration)
            DURATION="$2"
            shift 2
            ;;
        --container-id)
            CONTAINER_ID="$2"
            shift 2
            ;;
        --image)
            IMAGE="$2"
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

# Build query filter
build_query() {
    local filters=""

    if [ -n "$CONTAINER_ID" ]; then
        filters="container_id:${CONTAINER_ID}"
    fi

    if [ -n "$IMAGE" ]; then
        if [ -n "$filters" ]; then
            filters="${filters} AND image:${IMAGE}"
        else
            filters="image:${IMAGE}"
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

# Query container metrics
query_containers() {
    # Get container list
    local containers_response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v1/container?sort=-cpu.usage")

    # Query CPU usage metric
    local cpu_query="avg:container.cpu.usage{${QUERY}} by {container_id,container_name,image_name}"
    local cpu_response=$(curl -s -G \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        --data-urlencode "query=${cpu_query}" \
        --data-urlencode "from=${FROM}" \
        --data-urlencode "to=${TO}" \
        "${API_BASE}/api/v1/query")

    # Query memory usage
    local mem_query="avg:container.memory.usage{${QUERY}} by {container_id,container_name,image_name}"
    local mem_response=$(curl -s -G \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        --data-urlencode "query=${mem_query}" \
        --data-urlencode "from=${FROM}" \
        --data-urlencode "to=${TO}" \
        "${API_BASE}/api/v1/query")

    # Query network I/O
    local net_query="avg:container.net.rcvd{${QUERY}} by {container_id,container_name}"
    local net_response=$(curl -s -G \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        --data-urlencode "query=${net_query}" \
        --data-urlencode "from=${FROM}" \
        --data-urlencode "to=${TO}" \
        "${API_BASE}/api/v1/query")

    # Check for errors
    if echo "$cpu_response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$cpu_response" | jq '{status: "error", message: .errors[0]}'
        exit 1
    fi

    # Combine and format results
    jq -n \
        --argjson cpu "$cpu_response" \
        --argjson mem "$mem_response" \
        --argjson net "$net_response" \
        --arg query "$QUERY" \
        --arg duration "$DURATION" \
        '{
            status: "success",
            query: $query,
            duration: $duration,
            containers: {
                cpu_usage: {
                    metric: "container.cpu.usage",
                    series: $cpu.series,
                    avg: (if $cpu.series then [$cpu.series[].pointlist[]?[1]] | add / length else 0 end)
                },
                memory_usage: {
                    metric: "container.memory.usage",
                    series: $mem.series,
                    avg: (if $mem.series then [$mem.series[].pointlist[]?[1]] | add / length else 0 end)
                },
                network_received: {
                    metric: "container.net.rcvd",
                    series: $net.series,
                    total: (if $net.series then [$net.series[].pointlist[]?[1]] | add else 0 end)
                }
            },
            summary: {
                cpu_avg_percent: (if $cpu.series then [$cpu.series[].pointlist[]?[1]] | add / length else 0 end),
                memory_avg_bytes: (if $mem.series then [$mem.series[].pointlist[]?[1]] | add / length else 0 end),
                network_total_bytes: (if $net.series then [$net.series[].pointlist[]?[1]] | add else 0 end),
                container_count: (if $cpu.series then ($cpu.series | length) else 0 end)
            }
        }'
}

# Execute query
query_containers
