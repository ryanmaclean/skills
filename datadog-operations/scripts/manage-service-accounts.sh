#!/bin/bash
# Manage Datadog Service Accounts
# Handle service accounts for automation and API access

set -e

# Configuration
DD_SITE=${DD_SITE:-datadoghq.com}
API_BASE="https://api.${DD_SITE}"

# Default values
ACTION="list"
SERVICE_ACCOUNT_ID=""
NAME=""
EMAIL=""
ROLES=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --action)
            ACTION="$2"
            shift 2
            ;;
        --account-id)
            SERVICE_ACCOUNT_ID="$2"
            shift 2
            ;;
        --name)
            NAME="$2"
            shift 2
            ;;
        --email)
            EMAIL="$2"
            shift 2
            ;;
        --roles)
            ROLES="$2"
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

# List service accounts
list_accounts() {
    response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v2/service_accounts")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        total_accounts: (.data | length),
        service_accounts: [.data[] | {
            id: .id,
            name: .attributes.name,
            email: .attributes.email,
            status: .attributes.status,
            disabled: .attributes.disabled,
            created_at: .attributes.created_at,
            roles: [.relationships.roles.data[]? | .id]
        }]
    }'
}

# Get service account
get_account() {
    if [ -z "$SERVICE_ACCOUNT_ID" ]; then
        echo '{"status":"error","message":"--account-id is required for get action"}' >&2
        exit 1
    fi

    response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v2/service_accounts/${SERVICE_ACCOUNT_ID}")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        service_account: {
            id: .data.id,
            name: .data.attributes.name,
            email: .data.attributes.email,
            status: .data.attributes.status,
            disabled: .data.attributes.disabled,
            created_at: .data.attributes.created_at,
            modified_at: .data.attributes.modified_at,
            roles: [.data.relationships.roles.data[]? | .id]
        }
    }'
}

# Create service account
create_account() {
    if [ -z "$NAME" ]; then
        echo '{"status":"error","message":"--name is required for creating service account"}' >&2
        exit 1
    fi

    if [ -z "$EMAIL" ]; then
        echo '{"status":"error","message":"--email is required for creating service account"}' >&2
        exit 1
    fi

    # Parse roles if provided
    local role_relationships="null"
    if [ -n "$ROLES" ]; then
        IFS=',' read -ra ROLE_LIST <<< "$ROLES"
        local role_data=$(printf '%s\n' "${ROLE_LIST[@]}" | jq -R '{type: "roles", id: .}' | jq -s .)
        role_relationships=$(jq -n --argjson roles "$role_data" '{roles: {data: $roles}}')
    fi

    # Build payload
    local payload=$(jq -n \
        --arg name "$NAME" \
        --arg email "$EMAIL" \
        --argjson rels "$role_relationships" \
        '{
            data: {
                type: "service_accounts",
                attributes: {
                    name: $name,
                    email: $email,
                    service_account: true
                },
                relationships: (if $rels then $rels else null end)
            }
        }')

    response=$(curl -s -X POST \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "${API_BASE}/api/v2/service_accounts")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        service_account: {
            id: .data.id,
            name: .data.attributes.name,
            email: .data.attributes.email,
            created_at: .data.attributes.created_at,
            roles: [.data.relationships.roles.data[]? | .id]
        }
    }'
}

# Execute action
case "$ACTION" in
    list)
        list_accounts
        ;;
    get)
        get_account
        ;;
    create)
        create_account
        ;;
    *)
        echo "{\"status\":\"error\",\"message\":\"Unknown action: $ACTION (use: list, get, create)\"}" >&2
        exit 1
        ;;
esac
