#!/bin/bash
# Manage Datadog Custom Metrics
# Query and manage custom metrics metadata and tags

set -e

# Configuration
DD_SITE=${DD_SITE:-datadoghq.com}
API_BASE="https://api.${DD_SITE}"

# Default values
ACTION="list"
METRIC_NAME=""
TAG_CONFIG=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --action)
            ACTION="$2"
            shift 2
            ;;
        --metric)
            METRIC_NAME="$2"
            shift 2
            ;;
        --tags)
            TAG_CONFIG="$2"
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

# List custom metrics
list_metrics() {
    # Get active metrics from the last hour
    local now=$(date +%s)
    local from=$((now - 3600))

    response=$(curl -s -G \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        --data-urlencode "from=${from}" \
        "${API_BASE}/api/v1/metrics")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0]}'
        exit 1
    fi

    # Parse metrics
    echo "$response" | jq '{
        status: "success",
        total_metrics: (.metrics | length),
        metrics: [
            .metrics[]
            | select(startswith("custom.") or (. | test("^[a-z_]+\\.[a-z_]+\\.[a-z_]+$")))
        ] | .[0:100],
        from_time: .from
    }'
}

# Get metric metadata
get_metadata() {
    if [ -z "$METRIC_NAME" ]; then
        echo '{"status":"error","message":"--metric is required for metadata action"}' >&2
        exit 1
    fi

    response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v1/metrics/${METRIC_NAME}")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0]}'
        exit 1
    fi

    # Parse metadata
    echo "$response" | jq '{
        status: "success",
        metric: {
            name: "'$METRIC_NAME'",
            type: .type,
            description: .description,
            short_name: .short_name,
            unit: .unit,
            per_unit: .per_unit,
            integration: .integration,
            interval: .interval
        }
    }'
}

# Update metric metadata
update_metadata() {
    if [ -z "$METRIC_NAME" ]; then
        echo '{"status":"error","message":"--metric is required for update action"}' >&2
        exit 1
    fi

    # Build payload (example with common fields)
    local payload=$(jq -n \
        --arg metric "$METRIC_NAME" \
        '{
            type: "gauge",
            description: "Custom metric",
            short_name: $metric,
            unit: "unit"
        }')

    response=$(curl -s -X PUT \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "${API_BASE}/api/v1/metrics/${METRIC_NAME}")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0]}'
        exit 1
    fi

    echo '{"status":"success","message":"Metric metadata updated"}'
}

# Query metric tags
query_tags() {
    if [ -z "$METRIC_NAME" ]; then
        echo '{"status":"error","message":"--metric is required for tags action"}' >&2
        exit 1
    fi

    local now=$(date +%s)
    local from=$((now - 3600))

    response=$(curl -s -G \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        --data-urlencode "from=${from}" \
        "${API_BASE}/api/v1/metrics/${METRIC_NAME}/tags")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0]}'
        exit 1
    fi

    # Parse tags
    echo "$response" | jq \
        --arg metric "$METRIC_NAME" \
        '{
            status: "success",
            metric: $metric,
            tags: .tags
        }'
}

# Search metrics
search_metrics() {
    local query="${METRIC_NAME:-*}"

    response=$(curl -s -G \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        --data-urlencode "q=${query}" \
        "${API_BASE}/api/v1/search")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0]}'
        exit 1
    fi

    # Parse search results
    echo "$response" | jq \
        --arg query "$query" \
        '{
            status: "success",
            query: $query,
            results: {
                metrics: .results.metrics,
                hosts: (.results.hosts | length),
                tags: (.results.tags | length)
            }
        }'
}

# Get metric volumes (usage)
query_volumes() {
    response=$(curl -s -G \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v1/usage/custom_metrics")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0]}'
        exit 1
    fi

    # Parse volumes
    echo "$response" | jq '{
        status: "success",
        custom_metrics_usage: [
            .usage[]
            | {
                date: .date,
                custom_metrics_avg: .avg_custom_metrics_count,
                custom_metrics_max: .max_custom_metrics_count,
                custom_metrics_hour_avg: .avg_custom_metrics_hour_count
            }
        ] | .[0:30]
    }'
}

# Execute action
case "$ACTION" in
    list)
        list_metrics
        ;;
    metadata)
        get_metadata
        ;;
    update)
        update_metadata
        ;;
    tags)
        query_tags
        ;;
    search)
        search_metrics
        ;;
    volumes)
        query_volumes
        ;;
    *)
        echo "{\"status\":\"error\",\"message\":\"Unknown action: $ACTION (use: list, metadata, update, tags, search, volumes)\"}" >&2
        exit 1
        ;;
esac
