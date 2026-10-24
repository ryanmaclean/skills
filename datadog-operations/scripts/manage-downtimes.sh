#!/bin/bash
# Manage Datadog Monitor Downtimes
# Handles listing, creating, and canceling downtimes for scheduled maintenance

set -e

# Configuration
DD_SITE=${DD_SITE:-datadoghq.com}
API_BASE="https://api.${DD_SITE}"

# Default values
ACTION="list"
DOWNTIME_ID=""
MESSAGE=""
SCOPE=""
MONITOR_ID=""
MONITOR_TAGS=""
START_TIME=""
END_TIME=""
DURATION=""
RRULE=""
TIMEZONE="UTC"
CURRENT_ONLY=false

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --action)
            ACTION="$2"
            shift 2
            ;;
        --downtime-id)
            DOWNTIME_ID="$2"
            shift 2
            ;;
        --message)
            MESSAGE="$2"
            shift 2
            ;;
        --scope)
            SCOPE="$2"
            shift 2
            ;;
        --monitor-id)
            MONITOR_ID="$2"
            shift 2
            ;;
        --monitor-tags)
            MONITOR_TAGS="$2"
            shift 2
            ;;
        --start)
            START_TIME="$2"
            shift 2
            ;;
        --end)
            END_TIME="$2"
            shift 2
            ;;
        --duration)
            DURATION="$2"
            shift 2
            ;;
        --rrule)
            RRULE="$2"
            shift 2
            ;;
        --timezone)
            TIMEZONE="$2"
            shift 2
            ;;
        --current-only)
            CURRENT_ONLY=true
            shift
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

# List downtimes
list_downtimes() {
    local url="${API_BASE}/api/v2/downtime"

    if [ "$CURRENT_ONLY" = true ]; then
        url="${url}?current_only=true"
    fi

    response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        -H "Content-Type: application/json" \
        "${url}")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    # Parse and format response
    echo "$response" | jq '{
        status: "success",
        total_downtimes: (.meta.page.total_count // 0),
        active_count: ([.data[] | select(.attributes.status == "active")] | length),
        scheduled_count: ([.data[] | select(.attributes.status == "scheduled")] | length),
        downtimes: [.data[] | {
            id: .id,
            message: .attributes.message,
            scope: .attributes.scope,
            status: .attributes.status,
            start: .attributes.schedule.start,
            end: .attributes.schedule.end,
            recurring: (.attributes.schedule.recurrences != null),
            timezone: .attributes.display_timezone,
            monitor_id: .attributes.monitor_identifier.monitor_id,
            monitor_tags: .attributes.monitor_identifier.monitor_tags
        }]
    }'
}

# Create downtime
create_downtime() {
    if [ -z "$START_TIME" ]; then
        echo '{"status":"error","message":"--start is required for creating downtime"}' >&2
        exit 1
    fi

    # Build attributes
    local attributes="{}"

    if [ -n "$MESSAGE" ]; then
        attributes=$(echo "$attributes" | jq --arg msg "$MESSAGE" '. + {message: $msg}')
    fi

    if [ -n "$SCOPE" ]; then
        attributes=$(echo "$attributes" | jq --arg scope "$SCOPE" '. + {scope: $scope}')
    fi

    # Build monitor identifier
    if [ -n "$MONITOR_ID" ]; then
        attributes=$(echo "$attributes" | jq --argjson mid "$MONITOR_ID" '. + {monitor_identifier: {monitor_id: $mid}}')
    elif [ -n "$MONITOR_TAGS" ]; then
        if [ "$MONITOR_TAGS" = "*" ]; then
            attributes=$(echo "$attributes" | jq '. + {monitor_identifier: {monitor_tags: ["*"]}}')
        else
            IFS=',' read -ra TAGS <<< "$MONITOR_TAGS"
            tags_json=$(printf '%s\n' "${TAGS[@]}" | jq -R . | jq -s .)
            attributes=$(echo "$attributes" | jq --argjson tags "$tags_json" '. + {monitor_identifier: {monitor_tags: $tags}}')
        fi
    fi

    # Build schedule
    local schedule="{}"
    if [ -n "$RRULE" ]; then
        # Recurring downtime
        local recurrence="{}"
        recurrence=$(echo "$recurrence" | jq --arg rrule "$RRULE" '. + {rrule: $rrule}')

        if [ -n "$START_TIME" ]; then
            recurrence=$(echo "$recurrence" | jq --arg start "$START_TIME" '. + {start: $start}')
        fi

        if [ -n "$DURATION" ]; then
            recurrence=$(echo "$recurrence" | jq --arg dur "$DURATION" '. + {duration: $dur}')
        fi

        schedule=$(echo "$schedule" | jq --argjson rec "$recurrence" --arg tz "$TIMEZONE" '{
            recurrences: [$rec],
            timezone: $tz
        }')
    else
        # One-time downtime
        schedule=$(echo "$schedule" | jq --arg start "$START_TIME" '. + {start: $start}')
        if [ -n "$END_TIME" ]; then
            schedule=$(echo "$schedule" | jq --arg end "$END_TIME" '. + {end: $end}')
        fi
    fi

    attributes=$(echo "$attributes" | jq --argjson sched "$schedule" --arg tz "$TIMEZONE" '. + {
        schedule: $sched,
        display_timezone: $tz
    }')

    # Build payload
    local payload=$(jq -n --argjson attrs "$attributes" '{
        data: {
            type: "downtime",
            attributes: $attrs
        }
    }')

    # Create downtime
    response=$(curl -s -X POST \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "${API_BASE}/api/v2/downtime")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        downtime: {
            id: .data.id,
            message: .data.attributes.message,
            scope: .data.attributes.scope,
            status: .data.attributes.status,
            start: .data.attributes.schedule.start,
            end: .data.attributes.schedule.end,
            recurring: (.data.attributes.schedule.recurrences != null),
            timezone: .data.attributes.display_timezone
        }
    }'
}

# Get downtime
get_downtime() {
    if [ -z "$DOWNTIME_ID" ]; then
        echo '{"status":"error","message":"--downtime-id is required for get action"}' >&2
        exit 1
    fi

    response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        -H "Content-Type: application/json" \
        "${API_BASE}/api/v2/downtime/${DOWNTIME_ID}")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        downtime: {
            id: .data.id,
            message: .data.attributes.message,
            scope: .data.attributes.scope,
            status: .data.attributes.status,
            start: .data.attributes.schedule.start,
            end: .data.attributes.schedule.end,
            recurring: (.data.attributes.schedule.recurrences != null),
            timezone: .data.attributes.display_timezone,
            monitor_id: .data.attributes.monitor_identifier.monitor_id,
            monitor_tags: .data.attributes.monitor_identifier.monitor_tags,
            schedule: .data.attributes.schedule
        }
    }'
}

# Cancel downtime
cancel_downtime() {
    if [ -z "$DOWNTIME_ID" ]; then
        echo '{"status":"error","message":"--downtime-id is required for cancel action"}' >&2
        exit 1
    fi

    response=$(curl -s -X DELETE \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v2/downtime/${DOWNTIME_ID}")

    # 204 No Content is success
    if [ -z "$response" ]; then
        echo "{\"status\":\"success\",\"message\":\"Downtime ${DOWNTIME_ID} canceled successfully\"}"
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
        list_downtimes
        ;;
    create)
        create_downtime
        ;;
    get)
        get_downtime
        ;;
    cancel|delete)
        cancel_downtime
        ;;
    *)
        echo "{\"status\":\"error\",\"message\":\"Unknown action: $ACTION (use: list, create, get, cancel)\"}" >&2
        exit 1
        ;;
esac
