#!/bin/bash
# Manage Datadog Status Pages
# Handle status pages for customer communication

set -e

# Configuration
DD_SITE=${DD_SITE:-datadoghq.com}
API_BASE="https://api.${DD_SITE}"

# Default values
ACTION="list"
PAGE_ID=""
NAME=""
DESCRIPTION=""
SUBDOMAIN=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --action)
            ACTION="$2"
            shift 2
            ;;
        --page-id)
            PAGE_ID="$2"
            shift 2
            ;;
        --name)
            NAME="$2"
            shift 2
            ;;
        --description)
            DESCRIPTION="$2"
            shift 2
            ;;
        --subdomain)
            SUBDOMAIN="$2"
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

# List status pages
list_pages() {
    response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v1/status_pages")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0]}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        total_pages: (. | length),
        status_pages: [.[] | {
            id: .id,
            name: .name,
            description: .description,
            subdomain: .subdomain,
            url: .url,
            created_at: .created_at,
            modified_at: .modified_at,
            component_count: (.components | length)
        }]
    }'
}

# Get status page
get_page() {
    if [ -z "$PAGE_ID" ]; then
        echo '{"status":"error","message":"--page-id is required for get action"}' >&2
        exit 1
    fi

    response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v1/status_pages/${PAGE_ID}")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0]}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        status_page: {
            id: .id,
            name: .name,
            description: .description,
            subdomain: .subdomain,
            url: .url,
            created_at: .created_at,
            modified_at: .modified_at,
            components: [.components[] | {
                id: .id,
                name: .name,
                status: .status,
                description: .description
            }],
            incidents: [.incidents[] | {
                id: .id,
                name: .name,
                status: .status,
                impact: .impact
            }]
        }
    }'
}

# Create status page
create_page() {
    if [ -z "$NAME" ]; then
        echo '{"status":"error","message":"--name is required for creating status page"}' >&2
        exit 1
    fi

    if [ -z "$SUBDOMAIN" ]; then
        echo '{"status":"error","message":"--subdomain is required for creating status page"}' >&2
        exit 1
    fi

    # Build payload
    local payload=$(jq -n \
        --arg name "$NAME" \
        --arg desc "${DESCRIPTION:-}" \
        --arg subdomain "$SUBDOMAIN" \
        '{
            name: $name,
            description: $desc,
            subdomain: $subdomain
        }')

    response=$(curl -s -X POST \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "${API_BASE}/api/v1/status_pages")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0]}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        status_page: {
            id: .id,
            name: .name,
            subdomain: .subdomain,
            url: .url,
            created_at: .created_at
        }
    }'
}

# Update status page
update_page() {
    if [ -z "$PAGE_ID" ]; then
        echo '{"status":"error","message":"--page-id is required for update action"}' >&2
        exit 1
    fi

    # Build payload
    local payload="{}"
    if [ -n "$NAME" ]; then
        payload=$(echo "$payload" | jq --arg name "$NAME" '. + {name: $name}')
    fi
    if [ -n "$DESCRIPTION" ]; then
        payload=$(echo "$payload" | jq --arg desc "$DESCRIPTION" '. + {description: $desc}')
    fi

    response=$(curl -s -X PATCH \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "${API_BASE}/api/v1/status_pages/${PAGE_ID}")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0]}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        status_page: {
            id: .id,
            name: .name,
            description: .description,
            modified_at: .modified_at
        }
    }'
}

# Delete status page
delete_page() {
    if [ -z "$PAGE_ID" ]; then
        echo '{"status":"error","message":"--page-id is required for delete action"}' >&2
        exit 1
    fi

    response=$(curl -s -X DELETE \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v1/status_pages/${PAGE_ID}")

    # 204 No Content is success
    if [ -z "$response" ]; then
        echo "{\"status\":\"success\",\"message\":\"Status page ${PAGE_ID} deleted successfully\"}"
    else
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
        list_pages
        ;;
    get)
        get_page
        ;;
    create)
        create_page
        ;;
    update)
        update_page
        ;;
    delete)
        delete_page
        ;;
    *)
        echo "{\"status\":\"error\",\"message\":\"Unknown action: $ACTION (use: list, get, create, update, delete)\"}" >&2
        exit 1
        ;;
esac
