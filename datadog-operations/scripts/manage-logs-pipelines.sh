#!/bin/bash
# Manage Datadog Logs Pipelines
# Configure log processing, parsing, and enrichment

set -e

# Configuration
DD_SITE=${DD_SITE:-datadoghq.com}
API_BASE="https://api.${DD_SITE}"

# Default values
ACTION="list"
PIPELINE_ID=""
PIPELINE_NAME=""
FILTER_QUERY=""
ENABLED="true"

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --action)
            ACTION="$2"
            shift 2
            ;;
        --pipeline-id)
            PIPELINE_ID="$2"
            shift 2
            ;;
        --name)
            PIPELINE_NAME="$2"
            shift 2
            ;;
        --filter)
            FILTER_QUERY="$2"
            shift 2
            ;;
        --enabled)
            ENABLED="$2"
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

# List pipelines
list_pipelines() {
    response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v1/logs/config/pipelines")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0]}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        total_pipelines: (. | length),
        pipelines: [.[] | {
            id: .id,
            name: .name,
            is_enabled: .is_enabled,
            is_read_only: .is_read_only,
            filter: .filter.query,
            processor_count: (.processors | length),
            processors: [.processors[] | {
                type: .type,
                name: .name,
                is_enabled: .is_enabled
            }]
        }]
    }'
}

# Get pipeline details
get_pipeline() {
    if [ -z "$PIPELINE_ID" ]; then
        echo '{"status":"error","message":"--pipeline-id is required for get action"}' >&2
        exit 1
    fi

    response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v1/logs/config/pipelines/${PIPELINE_ID}")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0]}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        pipeline: {
            id: .id,
            name: .name,
            is_enabled: .is_enabled,
            is_read_only: .is_read_only,
            filter: .filter,
            processors: .processors
        }
    }'
}

# Create pipeline
create_pipeline() {
    if [ -z "$PIPELINE_NAME" ] || [ -z "$FILTER_QUERY" ]; then
        echo '{"status":"error","message":"--name and --filter are required for create action"}' >&2
        exit 1
    fi

    # Build payload with basic pipeline structure
    local payload=$(jq -n \
        --arg name "$PIPELINE_NAME" \
        --arg filter "$FILTER_QUERY" \
        --argjson enabled "$([ "$ENABLED" = "true" ] && echo true || echo false)" \
        '{
            name: $name,
            is_enabled: $enabled,
            filter: {
                query: $filter
            },
            processors: []
        }')

    response=$(curl -s -X POST \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "${API_BASE}/api/v1/logs/config/pipelines")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0]}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        pipeline: {
            id: .id,
            name: .name,
            is_enabled: .is_enabled,
            filter: .filter.query
        }
    }'
}

# Update pipeline
update_pipeline() {
    if [ -z "$PIPELINE_ID" ]; then
        echo '{"status":"error","message":"--pipeline-id is required for update action"}' >&2
        exit 1
    fi

    # Get existing pipeline
    existing=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v1/logs/config/pipelines/${PIPELINE_ID}")

    # Build update payload
    local payload="$existing"

    if [ -n "$PIPELINE_NAME" ]; then
        payload=$(echo "$payload" | jq --arg name "$PIPELINE_NAME" '.name = $name')
    fi

    if [ -n "$FILTER_QUERY" ]; then
        payload=$(echo "$payload" | jq --arg filter "$FILTER_QUERY" '.filter.query = $filter')
    fi

    if [ -n "$ENABLED" ]; then
        payload=$(echo "$payload" | jq --argjson enabled "$([ "$ENABLED" = "true" ] && echo true || echo false)" '.is_enabled = $enabled')
    fi

    response=$(curl -s -X PUT \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "${API_BASE}/api/v1/logs/config/pipelines/${PIPELINE_ID}")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0]}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        pipeline: {
            id: .id,
            name: .name,
            is_enabled: .is_enabled,
            filter: .filter.query
        }
    }'
}

# Delete pipeline
delete_pipeline() {
    if [ -z "$PIPELINE_ID" ]; then
        echo '{"status":"error","message":"--pipeline-id is required for delete action"}' >&2
        exit 1
    fi

    response=$(curl -s -X DELETE \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v1/logs/config/pipelines/${PIPELINE_ID}")

    # 204 No Content or empty response is success
    if [ -z "$response" ] || [ "$response" = "null" ]; then
        echo "{\"status\":\"success\",\"message\":\"Pipeline ${PIPELINE_ID} deleted successfully\"}"
    else
        if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
            echo "$response" | jq '{status: "error", message: .errors[0]}'
            exit 1
        fi
        echo "$response"
    fi
}

# List pipeline processors
list_processors() {
    if [ -z "$PIPELINE_ID" ]; then
        echo '{"status":"error","message":"--pipeline-id is required for processors action"}' >&2
        exit 1
    fi

    response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v1/logs/config/pipelines/${PIPELINE_ID}")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0]}'
        exit 1
    fi

    echo "$response" | jq '{
        status: "success",
        pipeline_id: .id,
        pipeline_name: .name,
        total_processors: (.processors | length),
        processors: [.processors[] | {
            type: .type,
            name: .name,
            is_enabled: .is_enabled,
            source: .source,
            target: .target,
            grok: .grok,
            samples: .samples
        }]
    }'
}

# Execute action
case "$ACTION" in
    list)
        list_pipelines
        ;;
    get)
        get_pipeline
        ;;
    create)
        create_pipeline
        ;;
    update)
        update_pipeline
        ;;
    delete)
        delete_pipeline
        ;;
    processors)
        list_processors
        ;;
    *)
        echo "{\"status\":\"error\",\"message\":\"Unknown action: $ACTION (use: list, get, create, update, delete, processors)\"}" >&2
        exit 1
        ;;
esac
