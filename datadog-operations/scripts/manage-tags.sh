#!/bin/bash
# Manage Datadog Host Tags
# Handles viewing and managing tags for infrastructure organization

set -e

# Configuration
DD_SITE=${DD_SITE:-datadoghq.com}
API_BASE="https://api.${DD_SITE}"

# Default values
ACTION="list"
HOST=""
TAGS=""
SOURCE=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --action)
            ACTION="$2"
            shift 2
            ;;
        --host)
            HOST="$2"
            shift 2
            ;;
        --tags)
            TAGS="$2"
            shift 2
            ;;
        --source)
            SOURCE="$2"
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

# List all host tags
list_tags() {
    local url="${API_BASE}/api/v1/tags/hosts"

    if [ -n "$SOURCE" ]; then
        url="${url}?source=${SOURCE}"
    fi

    response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${url}")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0]}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        total_hosts: (.tags | length),
        hosts: [.tags | to_entries[] | {
            host: .key,
            tags: .value
        }]
    }'
}

# Get host tags
get_host_tags() {
    if [ -z "$HOST" ]; then
        echo '{"status":"error","message":"--host is required for get action"}' >&2
        exit 1
    fi

    local url="${API_BASE}/api/v1/tags/hosts/${HOST}"

    if [ -n "$SOURCE" ]; then
        url="${url}?source=${SOURCE}"
    fi

    response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${url}")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0]}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        host: .tags.host,
        tags: .tags.tags
    }'
}

# Add tags to host
add_tags() {
    if [ -z "$HOST" ]; then
        echo '{"status":"error","message":"--host is required for add action"}' >&2
        exit 1
    fi

    if [ -z "$TAGS" ]; then
        echo '{"status":"error","message":"--tags is required for add action"}' >&2
        exit 1
    fi

    # Parse tags into array
    IFS=',' read -ra TAG_LIST <<< "$TAGS"
    local tags_array=$(printf '%s\n' "${TAG_LIST[@]}" | jq -R . | jq -s .)

    # Build payload
    local payload=$(jq -n \
        --argjson tags "$tags_array" \
        --arg source "${SOURCE:-api}" \
        '{
            tags: $tags,
            source: $source
        }')

    response=$(curl -s -X POST \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "${API_BASE}/api/v1/tags/hosts/${HOST}")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0]}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        host: .host,
        tags: .tags
    }'
}

# Update host tags (replace all)
update_tags() {
    if [ -z "$HOST" ]; then
        echo '{"status":"error","message":"--host is required for update action"}' >&2
        exit 1
    fi

    if [ -z "$TAGS" ]; then
        echo '{"status":"error","message":"--tags is required for update action"}' >&2
        exit 1
    fi

    # Parse tags into array
    IFS=',' read -ra TAG_LIST <<< "$TAGS"
    local tags_array=$(printf '%s\n' "${TAG_LIST[@]}" | jq -R . | jq -s .)

    # Build payload
    local payload=$(jq -n \
        --argjson tags "$tags_array" \
        --arg source "${SOURCE:-api}" \
        '{
            tags: $tags,
            source: $source
        }')

    response=$(curl -s -X PUT \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "${API_BASE}/api/v1/tags/hosts/${HOST}")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0]}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        host: .host,
        tags: .tags
    }'
}

# Delete host tags
delete_tags() {
    if [ -z "$HOST" ]; then
        echo '{"status":"error","message":"--host is required for delete action"}' >&2
        exit 1
    fi

    local url="${API_BASE}/api/v1/tags/hosts/${HOST}"

    if [ -n "$SOURCE" ]; then
        url="${url}?source=${SOURCE}"
    fi

    response=$(curl -s -X DELETE \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${url}")

    # 204 No Content is success
    if [ -z "$response" ]; then
        echo "{\"status\":\"success\",\"message\":\"Tags deleted for host ${HOST}\"}"
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
        list_tags
        ;;
    get)
        get_host_tags
        ;;
    add)
        add_tags
        ;;
    update)
        update_tags
        ;;
    delete)
        delete_tags
        ;;
    *)
        echo "{\"status\":\"error\",\"message\":\"Unknown action: $ACTION (use: list, get, add, update, delete)\"}" >&2
        exit 1
        ;;
esac
