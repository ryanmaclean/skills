#!/bin/bash
# Manage Datadog API Keys
# Handle primary authentication keys for API access

set -e

# Configuration
DD_SITE=${DD_SITE:-datadoghq.com}
API_BASE="https://api.${DD_SITE}"

# Default values
ACTION="list"
KEY_ID=""
NAME=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --action)
            ACTION="$2"
            shift 2
            ;;
        --key-id)
            KEY_ID="$2"
            shift 2
            ;;
        --name)
            NAME="$2"
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

# List API keys
list_keys() {
    response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v2/api_keys")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    # Format response (mask keys for security)
    echo "$response" | jq '{
        status: "success",
        total_keys: (.data | length),
        api_keys: [.data[] | {
            id: .id,
            name: .attributes.name,
            created_at: .attributes.created_at,
            last_4: .attributes.last4,
            created_by: .relationships.created_by.data.id
        }]
    }'
}

# Get API key
get_key() {
    if [ -z "$KEY_ID" ]; then
        echo '{"status":"error","message":"--key-id is required for get action"}' >&2
        exit 1
    fi

    response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v2/api_keys/${KEY_ID}")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        api_key: {
            id: .data.id,
            name: .data.attributes.name,
            created_at: .data.attributes.created_at,
            modified_at: .data.attributes.modified_at,
            last_4: .data.attributes.last4,
            created_by: .data.relationships.created_by.data.id,
            modified_by: .data.relationships.modified_by.data.id
        }
    }'
}

# Create API key
create_key() {
    if [ -z "$NAME" ]; then
        echo '{"status":"error","message":"--name is required for creating API key"}' >&2
        exit 1
    fi

    # Build payload
    local payload=$(jq -n \
        --arg name "$NAME" \
        '{
            data: {
                type: "api_keys",
                attributes: {
                    name: $name
                }
            }
        }')

    response=$(curl -s -X POST \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "${API_BASE}/api/v2/api_keys")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    # Format response - include the actual key this one time
    echo "$response" | jq '{
        status: "success",
        api_key: {
            id: .data.id,
            name: .data.attributes.name,
            key: .data.attributes.key,
            created_at: .data.attributes.created_at
        },
        warning: "Save this key securely - it will not be shown again"
    }'
}

# Update API key
update_key() {
    if [ -z "$KEY_ID" ]; then
        echo '{"status":"error","message":"--key-id is required for update action"}' >&2
        exit 1
    fi

    if [ -z "$NAME" ]; then
        echo '{"status":"error","message":"--name is required for update action"}' >&2
        exit 1
    fi

    # Build payload
    local payload=$(jq -n \
        --arg name "$NAME" \
        '{
            data: {
                type: "api_keys",
                attributes: {
                    name: $name
                }
            }
        }')

    response=$(curl -s -X PATCH \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "${API_BASE}/api/v2/api_keys/${KEY_ID}")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        api_key: {
            id: .data.id,
            name: .data.attributes.name,
            modified_at: .data.attributes.modified_at
        }
    }'
}

# Delete API key
delete_key() {
    if [ -z "$KEY_ID" ]; then
        echo '{"status":"error","message":"--key-id is required for delete action"}' >&2
        exit 1
    fi

    response=$(curl -s -X DELETE \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v2/api_keys/${KEY_ID}")

    # 204 No Content is success
    if [ -z "$response" ]; then
        echo "{\"status\":\"success\",\"message\":\"API key ${KEY_ID} deleted successfully\"}"
    else
        # Check for errors
        if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
            echo "$response" | jq '{status: "error", message: .errors[0].detail}'
            exit 1
        fi
        echo "$response"
    fi
}

# Execute action
case "$ACTION" in
    list)
        list_keys
        ;;
    get)
        get_key
        ;;
    create)
        create_key
        ;;
    update)
        update_key
        ;;
    delete)
        delete_key
        ;;
    *)
        echo "{\"status\":\"error\",\"message\":\"Unknown action: $ACTION (use: list, get, create, update, delete)\"}" >&2
        exit 1
        ;;
esac
