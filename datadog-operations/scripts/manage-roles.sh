#!/bin/bash
# Manage Datadog Roles
# Handle role and permissions management for fine-grained access control

set -e

# Configuration
DD_SITE=${DD_SITE:-datadoghq.com}
API_BASE="https://api.${DD_SITE}"

# Default values
ACTION="list"
ROLE_ID=""
NAME=""
PERMISSIONS=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --action)
            ACTION="$2"
            shift 2
            ;;
        --role-id)
            ROLE_ID="$2"
            shift 2
            ;;
        --name)
            NAME="$2"
            shift 2
            ;;
        --permissions)
            PERMISSIONS="$2"
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

# List roles
list_roles() {
    response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v2/roles")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        total_roles: (.data | length),
        roles: [.data[] | {
            id: .id,
            name: .attributes.name,
            user_count: .attributes.user_count,
            created_at: .attributes.created_at,
            modified_at: .attributes.modified_at,
            permissions: [.relationships.permissions.data[]? | .id]
        }]
    }'
}

# Get role
get_role() {
    if [ -z "$ROLE_ID" ]; then
        echo '{"status":"error","message":"--role-id is required for get action"}' >&2
        exit 1
    fi

    response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v2/roles/${ROLE_ID}")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        role: {
            id: .data.id,
            name: .data.attributes.name,
            user_count: .data.attributes.user_count,
            created_at: .data.attributes.created_at,
            modified_at: .data.attributes.modified_at,
            permissions: [.data.relationships.permissions.data[]? | .id]
        }
    }'
}

# Create role
create_role() {
    if [ -z "$NAME" ]; then
        echo '{"status":"error","message":"--name is required for creating role"}' >&2
        exit 1
    fi

    # Build payload
    local payload=$(jq -n \
        --arg name "$NAME" \
        '{
            data: {
                type: "roles",
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
        "${API_BASE}/api/v2/roles")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        role: {
            id: .data.id,
            name: .data.attributes.name,
            created_at: .data.attributes.created_at
        }
    }'
}

# Update role
update_role() {
    if [ -z "$ROLE_ID" ]; then
        echo '{"status":"error","message":"--role-id is required for update action"}' >&2
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
                type: "roles",
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
        "${API_BASE}/api/v2/roles/${ROLE_ID}")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        role: {
            id: .data.id,
            name: .data.attributes.name,
            modified_at: .data.attributes.modified_at
        }
    }'
}

# Delete role
delete_role() {
    if [ -z "$ROLE_ID" ]; then
        echo '{"status":"error","message":"--role-id is required for delete action"}' >&2
        exit 1
    fi

    response=$(curl -s -X DELETE \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v2/roles/${ROLE_ID}")

    # 204 No Content is success
    if [ -z "$response" ]; then
        echo "{\"status\":\"success\",\"message\":\"Role ${ROLE_ID} deleted successfully\"}"
    else
        # Check for errors
        if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
            echo "$response" | jq '{status: "error", message: .errors[0].detail}'
            exit 1
        fi
        echo "$response"
    fi
}

# List permissions
list_permissions() {
    response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v2/permissions")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        total_permissions: (.data | length),
        permissions: [.data[] | {
            id: .id,
            name: .attributes.name,
            display_name: .attributes.display_name,
            description: .attributes.description,
            group_name: .attributes.group_name
        }]
    }'
}

# Execute action
case "$ACTION" in
    list)
        list_roles
        ;;
    get)
        get_role
        ;;
    create)
        create_role
        ;;
    update)
        update_role
        ;;
    delete)
        delete_role
        ;;
    list-permissions)
        list_permissions
        ;;
    *)
        echo "{\"status\":\"error\",\"message\":\"Unknown action: $ACTION (use: list, get, create, update, delete, list-permissions)\"}" >&2
        exit 1
        ;;
esac
