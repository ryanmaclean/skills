#!/bin/bash
# Manage Datadog Restriction Policies
# Control access to resources with granular RBAC policies

set -e

# Configuration
DD_SITE=${DD_SITE:-datadoghq.com}
API_BASE="https://api.${DD_SITE}"

# Default values
ACTION="list"
RESOURCE_ID=""
PRINCIPALS=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --action)
            ACTION="$2"
            shift 2
            ;;
        --resource-id)
            RESOURCE_ID="$2"
            shift 2
            ;;
        --principals)
            PRINCIPALS="$2"
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

# List restriction policies
list_policies() {
    response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v2/restriction_policy")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        total_policies: (.data | length),
        restriction_policies: [.data[] | {
            id: .id,
            resource_type: .attributes.resource_type,
            resource_id: .attributes.resource_id,
            principals: .relationships.principals.data,
            created_at: .attributes.created_at,
            modified_at: .attributes.modified_at
        }]
    }'
}

# Get specific policy
get_policy() {
    if [ -z "$RESOURCE_ID" ]; then
        echo '{"status":"error","message":"--resource-id is required for get action"}' >&2
        exit 1
    fi

    response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v2/restriction_policy/${RESOURCE_ID}")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        policy: {
            id: .data.id,
            resource_type: .data.attributes.resource_type,
            resource_id: .data.attributes.resource_id,
            principals: [.data.relationships.principals.data[] | {
                type: .type,
                id: .id
            }],
            created_at: .data.attributes.created_at,
            modified_at: .data.attributes.modified_at
        }
    }'
}

# Create restriction policy
create_policy() {
    if [ -z "$RESOURCE_ID" ] || [ -z "$PRINCIPALS" ]; then
        echo '{"status":"error","message":"--resource-id and --principals are required"}' >&2
        exit 1
    fi

    # Parse principals (format: "user:uuid1,team:uuid2,role:uuid3")
    local principals_json=$(echo "$PRINCIPALS" | jq -R 'split(",") | map(split(":") | {type: .[0], id: .[1]})')

    local payload=$(jq -n \
        --arg resource_id "$RESOURCE_ID" \
        --argjson principals "$principals_json" \
        '{
            data: {
                type: "restriction_policy",
                attributes: {
                    resource_id: $resource_id
                },
                relationships: {
                    principals: {
                        data: $principals
                    }
                }
            }
        }')

    response=$(curl -s -X POST \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "${API_BASE}/api/v2/restriction_policy")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        policy: {
            id: .data.id,
            resource_id: .data.attributes.resource_id,
            principals: [.data.relationships.principals.data[] | {type: .type, id: .id}],
            created_at: .data.attributes.created_at
        }
    }'
}

# Update restriction policy
update_policy() {
    if [ -z "$RESOURCE_ID" ] || [ -z "$PRINCIPALS" ]; then
        echo '{"status":"error","message":"--resource-id and --principals are required"}' >&2
        exit 1
    fi

    # Parse principals
    local principals_json=$(echo "$PRINCIPALS" | jq -R 'split(",") | map(split(":") | {type: .[0], id: .[1]})')

    local payload=$(jq -n \
        --arg resource_id "$RESOURCE_ID" \
        --argjson principals "$principals_json" \
        '{
            data: {
                type: "restriction_policy",
                id: $resource_id,
                attributes: {
                    resource_id: $resource_id
                },
                relationships: {
                    principals: {
                        data: $principals
                    }
                }
            }
        }')

    response=$(curl -s -X PATCH \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "${API_BASE}/api/v2/restriction_policy/${RESOURCE_ID}")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        policy: {
            id: .data.id,
            resource_id: .data.attributes.resource_id,
            principals: [.data.relationships.principals.data[] | {type: .type, id: .id}],
            modified_at: .data.attributes.modified_at
        }
    }'
}

# Delete restriction policy
delete_policy() {
    if [ -z "$RESOURCE_ID" ]; then
        echo '{"status":"error","message":"--resource-id is required for delete action"}' >&2
        exit 1
    fi

    response=$(curl -s -X DELETE \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v2/restriction_policy/${RESOURCE_ID}")

    # 204 No Content is success
    if [ -z "$response" ]; then
        echo "{\"status\":\"success\",\"message\":\"Restriction policy for ${RESOURCE_ID} deleted successfully\"}"
    else
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
        list_policies
        ;;
    get)
        get_policy
        ;;
    create)
        create_policy
        ;;
    update)
        update_policy
        ;;
    delete)
        delete_policy
        ;;
    *)
        echo "{\"status\":\"error\",\"message\":\"Unknown action: $ACTION (use: list, get, create, update, delete)\"}" >&2
        exit 1
        ;;
esac
