#!/bin/bash
# Manage Datadog On-Call Schedules
# Handle on-call scheduling and rotations

set -e

# Configuration
DD_SITE=${DD_SITE:-datadoghq.com}
API_BASE="https://api.${DD_SITE}"

# Default values
ACTION="list"
SCHEDULE_ID=""
NAME=""
TIMEZONE="UTC"
TEAM_ID=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --action)
            ACTION="$2"
            shift 2
            ;;
        --schedule-id)
            SCHEDULE_ID="$2"
            shift 2
            ;;
        --name)
            NAME="$2"
            shift 2
            ;;
        --timezone)
            TIMEZONE="$2"
            shift 2
            ;;
        --team-id)
            TEAM_ID="$2"
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

# List on-call schedules
list_schedules() {
    response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v2/oncall/schedules")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        total_schedules: (.data | length),
        schedules: [.data[] | {
            id: .id,
            name: .attributes.name,
            timezone: .attributes.timezone,
            team_id: .relationships.team.data.id,
            created_at: .attributes.created_at,
            modified_at: .attributes.modified_at,
            current_on_call: [.relationships.current_on_call.data[]? | .id]
        }]
    }'
}

# Get on-call schedule
get_schedule() {
    if [ -z "$SCHEDULE_ID" ]; then
        echo '{"status":"error","message":"--schedule-id is required for get action"}' >&2
        exit 1
    fi

    response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v2/oncall/schedules/${SCHEDULE_ID}")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        schedule: {
            id: .data.id,
            name: .data.attributes.name,
            timezone: .data.attributes.timezone,
            team_id: .data.relationships.team.data.id,
            created_at: .data.attributes.created_at,
            modified_at: .data.attributes.modified_at,
            current_on_call: [.data.relationships.current_on_call.data[]? | {
                user_id: .id,
                type: .type
            }],
            schedule_layers: .data.attributes.schedule_layers
        }
    }'
}

# Create on-call schedule
create_schedule() {
    if [ -z "$NAME" ]; then
        echo '{"status":"error","message":"--name is required for creating schedule"}' >&2
        exit 1
    fi

    # Build payload
    local payload=$(jq -n \
        --arg name "$NAME" \
        --arg tz "$TIMEZONE" \
        --arg team "${TEAM_ID:-}" \
        '{
            data: {
                type: "oncall_schedule",
                attributes: {
                    name: $name,
                    timezone: $tz
                },
                relationships: (if $team != "" then {
                    team: {
                        data: {
                            type: "team",
                            id: $team
                        }
                    }
                } else null end)
            }
        }')

    response=$(curl -s -X POST \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "${API_BASE}/api/v2/oncall/schedules")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        schedule: {
            id: .data.id,
            name: .data.attributes.name,
            timezone: .data.attributes.timezone,
            created_at: .data.attributes.created_at
        }
    }'
}

# Update on-call schedule
update_schedule() {
    if [ -z "$SCHEDULE_ID" ]; then
        echo '{"status":"error","message":"--schedule-id is required for update action"}' >&2
        exit 1
    fi

    # Build attributes
    local attributes="{}"
    if [ -n "$NAME" ]; then
        attributes=$(echo "$attributes" | jq --arg name "$NAME" '. + {name: $name}')
    fi
    if [ -n "$TIMEZONE" ] && [ "$TIMEZONE" != "UTC" ]; then
        attributes=$(echo "$attributes" | jq --arg tz "$TIMEZONE" '. + {timezone: $tz}')
    fi

    local payload=$(jq -n \
        --argjson attrs "$attributes" \
        '{
            data: {
                type: "oncall_schedule",
                attributes: $attrs
            }
        }')

    response=$(curl -s -X PATCH \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "${API_BASE}/api/v2/oncall/schedules/${SCHEDULE_ID}")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        schedule: {
            id: .data.id,
            name: .data.attributes.name,
            timezone: .data.attributes.timezone,
            modified_at: .data.attributes.modified_at
        }
    }'
}

# Delete on-call schedule
delete_schedule() {
    if [ -z "$SCHEDULE_ID" ]; then
        echo '{"status":"error","message":"--schedule-id is required for delete action"}' >&2
        exit 1
    fi

    response=$(curl -s -X DELETE \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v2/oncall/schedules/${SCHEDULE_ID}")

    # 204 No Content is success
    if [ -z "$response" ]; then
        echo "{\"status\":\"success\",\"message\":\"On-call schedule ${SCHEDULE_ID} deleted successfully\"}"
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
        list_schedules
        ;;
    get)
        get_schedule
        ;;
    create)
        create_schedule
        ;;
    update)
        update_schedule
        ;;
    delete)
        delete_schedule
        ;;
    *)
        echo "{\"status\":\"error\",\"message\":\"Unknown action: $ACTION (use: list, get, create, update, delete)\"}" >&2
        exit 1
        ;;
esac
