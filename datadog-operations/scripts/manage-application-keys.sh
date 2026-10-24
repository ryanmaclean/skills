#!/bin/bash
# Manage Datadog Application Keys
# Handle application keys for API authentication

set -e

# Configuration
DD_SITE=${DD_SITE:-datadoghq.com}
API_BASE="https://api.${DD_SITE}"

# Default values
ACTION="list"
APP_KEY_ID=""
NAME=""
SCOPES=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --action)
            ACTION="$2"
            shift 2
            ;;
        --key-id)
            APP_KEY_ID="$2"
            shift 2
            ;;
        --name)
            NAME="$2"
            shift 2
            ;;
        --scopes)
            SCOPES="$2"
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

# List application keys
list_keys() {
    response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v2/application_keys")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        total_keys: (.data | length),
        application_keys: [.data[] | {
            id: .id,
            name: .attributes.name,
            created_at: .attributes.created_at,
            last_4: .attributes.last4,
            scopes: .attributes.scopes,
            created_by: .relationships.owned_by.data.id
        }]
    }'
}

# Get application key
get_key() {
    if [ -z "$APP_KEY_ID" ]; then
        echo '{"status":"error","message":"--key-id is required for get action"}' >&2
        exit 1
    fi

    response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v2/application_keys/${APP_KEY_ID}")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        application_key: {
            id: .data.id,
            name: .data.attributes.name,
            created_at: .data.attributes.created_at,
            last_4: .data.attributes.last4,
            scopes: .data.attributes.scopes,
            created_by: .data.relationships.owned_by.data.id
        }
    }'
}

# Create application key
create_key() {
    if [ -z "$NAME" ]; then
        echo '{"status":"error","message":"--name is required for creating application key"}' >&2
        exit 1
    fi

    # Parse scopes if provided
    local scopes_array="null"
    if [ -n "$SCOPES" ]; then
        IFS=',' read -ra SCOPE_LIST <<< "$SCOPES"
        scopes_array=$(printf '%s\n' "${SCOPE_LIST[@]}" | jq -R . | jq -s .)
    fi

    # Build payload
    local payload=$(jq -n \
        --arg name "$NAME" \
        --argjson scopes "$scopes_array" \
        '{
            data: {
                type: "application_keys",
                attributes: {
                    name: $name,
                    scopes: (if $scopes then $scopes else null end)
                }
            }
        }')

    response=$(curl -s -X POST \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "${API_BASE}/api/v2/application_keys")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    # Format response - include the actual key this one time
    echo "$response" | jq '{
        status: "success",
        application_key: {
            id: .data.id,
            name: .data.attributes.name,
            key: .data.attributes.key,
            created_at: .data.attributes.created_at,
            scopes: .data.attributes.scopes
        },
        warning: "Save this key securely - it will not be shown again"
    }'
}

# Update application key
update_key() {
    if [ -z "$APP_KEY_ID" ]; then
        echo '{"status":"error","message":"--key-id is required for update action"}' >&2
        exit 1
    fi

    # Build attributes
    local attributes="{}"
    if [ -n "$NAME" ]; then
        attributes=$(echo "$attributes" | jq --arg name "$NAME" '. + {name: $name}')
    fi
    if [ -n "$SCOPES" ]; then
        IFS=',' read -ra SCOPE_LIST <<< "$SCOPES"
        local scopes_array=$(printf '%s\n' "${SCOPE_LIST[@]}" | jq -R . | jq -s .)
        attributes=$(echo "$attributes" | jq --argjson scopes "$scopes_array" '. + {scopes: $scopes}')
    fi

    # Build payload
    local payload=$(jq -n \
        --argjson attrs "$attributes" \
        '{
            data: {
                type: "application_keys",
                attributes: $attrs
            }
        }')

    response=$(curl -s -X PATCH \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "${API_BASE}/api/v2/application_keys/${APP_KEY_ID}")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        application_key: {
            id: .data.id,
            name: .data.attributes.name,
            scopes: .data.attributes.scopes
        }
    }'
}

# Delete application key
delete_key() {
    if [ -z "$APP_KEY_ID" ]; then
        echo '{"status":"error","message":"--key-id is required for delete action"}' >&2
        exit 1
    fi

    response=$(curl -s -X DELETE \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v2/application_keys/${APP_KEY_ID}")

    # 204 No Content is success
    if [ -z "$response" ]; then
        echo "{\"status\":\"success\",\"message\":\"Application key ${APP_KEY_ID} deleted successfully\"}"
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
