#!/bin/bash
# Manage Datadog Notebooks
# Handles listing, getting, and deleting notebooks for documentation and investigation

set -e

# Configuration
DD_SITE=${DD_SITE:-datadoghq.com}
API_BASE="https://api.${DD_SITE}"

# Default values
ACTION="list"
NOTEBOOK_ID=""
QUERY=""
AUTHOR=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --action)
            ACTION="$2"
            shift 2
            ;;
        --notebook-id)
            NOTEBOOK_ID="$2"
            shift 2
            ;;
        --query)
            QUERY="$2"
            shift 2
            ;;
        --author)
            AUTHOR="$2"
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

# List notebooks
list_notebooks() {
    local url="${API_BASE}/api/v1/notebooks"
    local params=""

    if [ -n "$QUERY" ]; then
        params="?query=${QUERY}"
    fi

    if [ -n "$AUTHOR" ]; then
        if [ -n "$params" ]; then
            params="${params}&author=${AUTHOR}"
        else
            params="?author=${AUTHOR}"
        fi
    fi

    response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        -H "Content-Type: application/json" \
        "${url}${params}")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0]}'
        exit 1
    fi

    # Parse and format response
    echo "$response" | jq '{
        status: "success",
        total_notebooks: (.data | length),
        notebooks: [.data[] | {
            id: .id,
            name: .attributes.name,
            author: .attributes.author.handle,
            created: .attributes.created,
            modified: .attributes.modified,
            cells: (.attributes.cells | length),
            description: (.attributes.metadata.description // "")
        }]
    }'
}

# Get notebook
get_notebook() {
    if [ -z "$NOTEBOOK_ID" ]; then
        echo '{"status":"error","message":"--notebook-id is required for get action"}' >&2
        exit 1
    fi

    response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        -H "Content-Type: application/json" \
        "${API_BASE}/api/v1/notebooks/${NOTEBOOK_ID}")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0]}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        notebook: {
            id: .data.id,
            name: .data.attributes.name,
            author: .data.attributes.author.handle,
            created: .data.attributes.created,
            modified: .data.attributes.modified,
            time_status: .data.attributes.time.live_span,
            cells: [.data.attributes.cells[] | {
                type: .type,
                id: .id,
                text: (if .attributes.definition.text then .attributes.definition.text else null end),
                graph_size: (if .attributes.definition.requests then (.attributes.definition.requests | length) else 0 end)
            }]
        }
    }'
}

# Delete notebook
delete_notebook() {
    if [ -z "$NOTEBOOK_ID" ]; then
        echo '{"status":"error","message":"--notebook-id is required for delete action"}' >&2
        exit 1
    fi

    response=$(curl -s -X DELETE \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v1/notebooks/${NOTEBOOK_ID}")

    # 204 No Content is success
    if [ -z "$response" ]; then
        echo "{\"status\":\"success\",\"message\":\"Notebook ${NOTEBOOK_ID} deleted successfully\"}"
    else
        # Check for errors
        if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
            echo "$response" | jq '{status: "error", message: .errors[0]}'
            exit 1
        fi
        echo "$response"
    fi
}

# Execute action
case "$ACTION" in
    list)
        list_notebooks
        ;;
    get)
        get_notebook
        ;;
    delete)
        delete_notebook
        ;;
    *)
        echo "{\"status\":\"error\",\"message\":\"Unknown action: $ACTION (use: list, get, delete)\"}" >&2
        exit 1
        ;;
esac
