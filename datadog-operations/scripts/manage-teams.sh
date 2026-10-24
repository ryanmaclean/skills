#!/bin/bash
# Manage Datadog Teams
# Handle team organization and access control

set -e

# Configuration
DD_SITE=${DD_SITE:-datadoghq.com}
API_BASE="https://api.${DD_SITE}"

# Default values
ACTION="list"
TEAM_ID=""
NAME=""
HANDLE=""
DESCRIPTION=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --action)
            ACTION="$2"
            shift 2
            ;;
        --team-id)
            TEAM_ID="$2"
            shift 2
            ;;
        --name)
            NAME="$2"
            shift 2
            ;;
        --handle)
            HANDLE="$2"
            shift 2
            ;;
        --description)
            DESCRIPTION="$2"
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

# List teams
list_teams() {
    response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v2/team")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        total_teams: (.data | length),
        teams: [.data[] | {
            id: .id,
            name: .attributes.name,
            handle: .attributes.handle,
            description: .attributes.description,
            member_count: (.relationships.users.data | length),
            created_at: .attributes.created_at,
            modified_at: .attributes.modified_at
        }]
    }'
}

# Create team
create_team() {
    if [ -z "$NAME" ]; then
        echo '{"status":"error","message":"--name is required for creating team"}' >&2
        exit 1
    fi

    if [ -z "$HANDLE" ]; then
        echo '{"status":"error","message":"--handle is required for creating team"}' >&2
        exit 1
    fi

    # Build payload
    local payload=$(jq -n \
        --arg name "$NAME" \
        --arg handle "$HANDLE" \
        --arg desc "${DESCRIPTION:-}" \
        '{
            data: {
                type: "team",
                attributes: {
                    name: $name,
                    handle: $handle,
                    description: $desc
                }
            }
        }')

    response=$(curl -s -X POST \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "${API_BASE}/api/v2/team")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        team: {
            id: .data.id,
            name: .data.attributes.name,
            handle: .data.attributes.handle,
            description: .data.attributes.description,
            created_at: .data.attributes.created_at
        }
    }'
}

# Get team
get_team() {
    if [ -z "$TEAM_ID" ]; then
        echo '{"status":"error","message":"--team-id is required for get action"}' >&2
        exit 1
    fi

    response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v2/team/${TEAM_ID}")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        team: {
            id: .data.id,
            name: .data.attributes.name,
            handle: .data.attributes.handle,
            description: .data.attributes.description,
            member_count: (.data.relationships.users.data | length),
            members: [.data.relationships.users.data[] | .id],
            created_at: .data.attributes.created_at,
            modified_at: .data.attributes.modified_at
        }
    }'
}

# Update team
update_team() {
    if [ -z "$TEAM_ID" ]; then
        echo '{"status":"error","message":"--team-id is required for update action"}' >&2
        exit 1
    fi

    # Build payload
    local attributes="{}"
    if [ -n "$NAME" ]; then
        attributes=$(echo "$attributes" | jq --arg name "$NAME" '. + {name: $name}')
    fi
    if [ -n "$HANDLE" ]; then
        attributes=$(echo "$attributes" | jq --arg handle "$HANDLE" '. + {handle: $handle}')
    fi
    if [ -n "$DESCRIPTION" ]; then
        attributes=$(echo "$attributes" | jq --arg desc "$DESCRIPTION" '. + {description: $desc}')
    fi

    local payload=$(jq -n \
        --argjson attrs "$attributes" \
        '{
            data: {
                type: "team",
                attributes: $attrs
            }
        }')

    response=$(curl -s -X PATCH \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "${API_BASE}/api/v2/team/${TEAM_ID}")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        team: {
            id: .data.id,
            name: .data.attributes.name,
            handle: .data.attributes.handle,
            description: .data.attributes.description,
            modified_at: .data.attributes.modified_at
        }
    }'
}

# Delete team
delete_team() {
    if [ -z "$TEAM_ID" ]; then
        echo '{"status":"error","message":"--team-id is required for delete action"}' >&2
        exit 1
    fi

    response=$(curl -s -X DELETE \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v2/team/${TEAM_ID}")

    # 204 No Content is success
    if [ -z "$response" ]; then
        echo "{\"status\":\"success\",\"message\":\"Team ${TEAM_ID} deleted successfully\"}"
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
        list_teams
        ;;
    create)
        create_team
        ;;
    get)
        get_team
        ;;
    update)
        update_team
        ;;
    delete)
        delete_team
        ;;
    *)
        echo "{\"status\":\"error\",\"message\":\"Unknown action: $ACTION (use: list, create, get, update, delete)\"}" >&2
        exit 1
        ;;
esac
