#!/bin/bash
# Manage Datadog Users
# Handle user access control and administration

set -e

# Configuration
DD_SITE=${DD_SITE:-datadoghq.com}
API_BASE="https://api.${DD_SITE}"

# Default values
ACTION="list"
USER_ID=""
EMAIL=""
NAME=""
ROLE_ID=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --action)
            ACTION="$2"
            shift 2
            ;;
        --user-id)
            USER_ID="$2"
            shift 2
            ;;
        --email)
            EMAIL="$2"
            shift 2
            ;;
        --name)
            NAME="$2"
            shift 2
            ;;
        --role-id)
            ROLE_ID="$2"
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

# List users
list_users() {
    response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v2/users")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        total_users: (.data | length),
        users: [.data[] | {
            id: .id,
            email: .attributes.email,
            name: .attributes.name,
            handle: .attributes.handle,
            status: .attributes.status,
            verified: .attributes.verified,
            created_at: .attributes.created_at,
            disabled: .attributes.disabled,
            roles: [.relationships.roles.data[]? | .id]
        }]
    }'
}

# Get user
get_user() {
    if [ -z "$USER_ID" ]; then
        echo '{"status":"error","message":"--user-id is required for get action"}' >&2
        exit 1
    fi

    response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v2/users/${USER_ID}")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        user: {
            id: .data.id,
            email: .data.attributes.email,
            name: .data.attributes.name,
            handle: .data.attributes.handle,
            status: .data.attributes.status,
            verified: .data.attributes.verified,
            created_at: .data.attributes.created_at,
            modified_at: .data.attributes.modified_at,
            disabled: .data.attributes.disabled,
            roles: [.data.relationships.roles.data[]? | .id],
            teams: [.data.relationships.teams.data[]? | .id]
        }
    }'
}

# Invite user
invite_user() {
    if [ -z "$EMAIL" ]; then
        echo '{"status":"error","message":"--email is required for invite action"}' >&2
        exit 1
    fi

    # Build payload
    local payload=$(jq -n \
        --arg email "$EMAIL" \
        --arg name "${NAME:-}" \
        '{
            data: {
                type: "user_invitation",
                attributes: {
                    email: $email,
                    name: (if $name != "" then $name else null end)
                }
            }
        }')

    response=$(curl -s -X POST \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "${API_BASE}/api/v2/user_invitations")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        invitation: {
            id: .data.id,
            email: .data.attributes.email,
            expires_at: .data.attributes.expires_at,
            invitation_type: .data.attributes.invitation_type
        }
    }'
}

# Disable user
disable_user() {
    if [ -z "$USER_ID" ]; then
        echo '{"status":"error","message":"--user-id is required for disable action"}' >&2
        exit 1
    fi

    response=$(curl -s -X DELETE \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v2/users/${USER_ID}")

    # 204 No Content is success
    if [ -z "$response" ]; then
        echo "{\"status\":\"success\",\"message\":\"User ${USER_ID} disabled successfully\"}"
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
        list_users
        ;;
    get)
        get_user
        ;;
    invite)
        invite_user
        ;;
    disable)
        disable_user
        ;;
    *)
        echo "{\"status\":\"error\",\"message\":\"Unknown action: $ACTION (use: list, get, invite, disable)\"}" >&2
        exit 1
        ;;
esac
