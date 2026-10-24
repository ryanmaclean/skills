#!/bin/bash
# Manage Datadog Cases
# Handle Case Management for issue tracking and resolution

set -e

# Configuration
DD_SITE=${DD_SITE:-datadoghq.com}
API_BASE="https://api.${DD_SITE}"

# Default values
ACTION="list"
CASE_ID=""
TITLE=""
DESCRIPTION=""
TYPE="standard"
PRIORITY="normal"
STATUS="open"

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --action)
            ACTION="$2"
            shift 2
            ;;
        --case-id)
            CASE_ID="$2"
            shift 2
            ;;
        --title)
            TITLE="$2"
            shift 2
            ;;
        --description)
            DESCRIPTION="$2"
            shift 2
            ;;
        --type)
            TYPE="$2"
            shift 2
            ;;
        --priority)
            PRIORITY="$2"
            shift 2
            ;;
        --status)
            STATUS="$2"
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

# List cases
list_cases() {
    response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v2/cases")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        total_cases: (.data | length),
        cases: [.data[] | {
            id: .id,
            title: .attributes.title,
            status: .attributes.status,
            priority: .attributes.priority,
            type: .attributes.type,
            created_at: .attributes.created_at,
            modified_at: .attributes.modified_at,
            assignee: .relationships.assignee.data.id
        }]
    }'
}

# Get case
get_case() {
    if [ -z "$CASE_ID" ]; then
        echo '{"status":"error","message":"--case-id is required for get action"}' >&2
        exit 1
    fi

    response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v2/cases/${CASE_ID}")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        case: {
            id: .data.id,
            title: .data.attributes.title,
            description: .data.attributes.description,
            status: .data.attributes.status,
            priority: .data.attributes.priority,
            type: .data.attributes.type,
            created_at: .data.attributes.created_at,
            modified_at: .data.attributes.modified_at,
            assignee: .data.relationships.assignee.data.id,
            created_by: .data.relationships.created_by.data.id
        }
    }'
}

# Create case
create_case() {
    if [ -z "$TITLE" ]; then
        echo '{"status":"error","message":"--title is required for creating case"}' >&2
        exit 1
    fi

    # Build payload
    local payload=$(jq -n \
        --arg title "$TITLE" \
        --arg desc "${DESCRIPTION:-}" \
        --arg type "$TYPE" \
        --arg priority "$PRIORITY" \
        '{
            data: {
                type: "case",
                attributes: {
                    title: $title,
                    description: $desc,
                    type: $type,
                    priority: $priority
                }
            }
        }')

    response=$(curl -s -X POST \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "${API_BASE}/api/v2/cases")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        case: {
            id: .data.id,
            title: .data.attributes.title,
            status: .data.attributes.status,
            priority: .data.attributes.priority,
            created_at: .data.attributes.created_at
        }
    }'
}

# Update case status
update_case() {
    if [ -z "$CASE_ID" ]; then
        echo '{"status":"error","message":"--case-id is required for update action"}' >&2
        exit 1
    fi

    # Build payload
    local payload=$(jq -n \
        --arg status "$STATUS" \
        '{
            data: {
                type: "case",
                attributes: {
                    status: $status
                }
            }
        }')

    response=$(curl -s -X PATCH \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "${API_BASE}/api/v2/cases/${CASE_ID}")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        case: {
            id: .data.id,
            title: .data.attributes.title,
            status: .data.attributes.status,
            modified_at: .data.attributes.modified_at
        }
    }'
}

# Execute action
case "$ACTION" in
    list)
        list_cases
        ;;
    get)
        get_case
        ;;
    create)
        create_case
        ;;
    update)
        update_case
        ;;
    *)
        echo "{\"status\":\"error\",\"message\":\"Unknown action: $ACTION (use: list, get, create, update)\"}" >&2
        exit 1
        ;;
esac
