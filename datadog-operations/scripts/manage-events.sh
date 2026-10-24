#!/bin/bash
# Manage Datadog Events
# Query and post events for deployments, alerts, and changes

set -e

# Configuration
DD_SITE=${DD_SITE:-datadoghq.com}
API_BASE="https://api.${DD_SITE}"

# Default values
ACTION="list"
TITLE=""
TEXT=""
TAGS=""
ALERT_TYPE="info"
PRIORITY="normal"
SOURCE_TYPE="api"
DURATION="24h"
EVENT_ID=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --action)
            ACTION="$2"
            shift 2
            ;;
        --title)
            TITLE="$2"
            shift 2
            ;;
        --text)
            TEXT="$2"
            shift 2
            ;;
        --tags)
            TAGS="$2"
            shift 2
            ;;
        --alert-type)
            ALERT_TYPE="$2"
            shift 2
            ;;
        --priority)
            PRIORITY="$2"
            shift 2
            ;;
        --source-type)
            SOURCE_TYPE="$2"
            shift 2
            ;;
        --duration)
            DURATION="$2"
            shift 2
            ;;
        --event-id)
            EVENT_ID="$2"
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

# Convert duration to epoch timestamps
calculate_time_range() {
    local duration=$1
    local now=$(date +%s)
    local seconds=86400

    if [[ $duration =~ ^([0-9]+)([mhd])$ ]]; then
        local value="${BASH_REMATCH[1]}"
        local unit="${BASH_REMATCH[2]}"

        case $unit in
            m) seconds=$((value * 60)) ;;
            h) seconds=$((value * 3600)) ;;
            d) seconds=$((value * 86400)) ;;
        esac
    fi

    local from=$((now - seconds))
    echo "$from $now"
}

read FROM TO <<< $(calculate_time_range "$DURATION")

# List events
list_events() {
    local query_params="start=${FROM}&end=${TO}"

    if [ -n "$TAGS" ]; then
        query_params="${query_params}&tags=${TAGS}"
    fi

    if [ -n "$SOURCE_TYPE" ] && [ "$SOURCE_TYPE" != "api" ]; then
        query_params="${query_params}&sources=${SOURCE_TYPE}"
    fi

    response=$(curl -s -G \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v1/events?${query_params}")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0]}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        total_events: (.events | length),
        events: [.events[] | {
            id: .id,
            title: .title,
            text: .text,
            date_happened: .date_happened,
            priority: .priority,
            alert_type: .alert_type,
            source: .source,
            tags: .tags,
            url: .url
        }]
    }'
}

# Create event
create_event() {
    if [ -z "$TITLE" ]; then
        echo '{"status":"error","message":"--title is required for creating event"}' >&2
        exit 1
    fi

    # Parse tags into array
    local tags_array="[]"
    if [ -n "$TAGS" ]; then
        IFS=',' read -ra TAG_LIST <<< "$TAGS"
        tags_array=$(printf '%s\n' "${TAG_LIST[@]}" | jq -R . | jq -s .)
    fi

    # Build payload
    local payload=$(jq -n \
        --arg title "$TITLE" \
        --arg text "${TEXT:-$TITLE}" \
        --argjson tags "$tags_array" \
        --arg alert_type "$ALERT_TYPE" \
        --arg priority "$PRIORITY" \
        --arg source "$SOURCE_TYPE" \
        '{
            title: $title,
            text: $text,
            tags: $tags,
            alert_type: $alert_type,
            priority: $priority,
            source_type_name: $source
        }')

    response=$(curl -s -X POST \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "${API_BASE}/api/v1/events")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0]}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        event: {
            id: .event.id,
            title: .event.title,
            text: .event.text,
            date_happened: .event.date_happened,
            priority: .event.priority,
            alert_type: .event.alert_type,
            tags: .event.tags,
            url: .event.url
        }
    }'
}

# Get event
get_event() {
    if [ -z "$EVENT_ID" ]; then
        echo '{"status":"error","message":"--event-id is required for get action"}' >&2
        exit 1
    fi

    response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v1/events/${EVENT_ID}")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0]}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        event: {
            id: .event.id,
            title: .event.title,
            text: .event.text,
            date_happened: .event.date_happened,
            priority: .event.priority,
            alert_type: .event.alert_type,
            source: .event.source,
            tags: .event.tags,
            url: .event.url
        }
    }'
}

# Execute action
case "$ACTION" in
    list)
        list_events
        ;;
    create|post)
        create_event
        ;;
    get)
        get_event
        ;;
    *)
        echo "{\"status\":\"error\",\"message\":\"Unknown action: $ACTION (use: list, create, get)\"}" >&2
        exit 1
        ;;
esac
