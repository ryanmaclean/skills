#!/bin/bash
# Query Datadog Session Replay
# Analyze RUM session replays for user experience debugging

set -e

# Configuration
DD_SITE=${DD_SITE:-datadoghq.com}
API_BASE="https://api.${DD_SITE}"

# Default values
DURATION="24h"
APPLICATION_ID=""
ENV="prod"
LIMIT=50

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --duration)
            DURATION="$2"
            shift 2
            ;;
        --application-id)
            APPLICATION_ID="$2"
            shift 2
            ;;
        --env)
            ENV="$2"
            shift 2
            ;;
        --limit)
            LIMIT="$2"
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

# Convert duration to timestamps
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

# Build query
build_query() {
    local query="@type:session_replay"

    if [ -n "$APPLICATION_ID" ]; then
        query="$query @application.id:${APPLICATION_ID}"
    fi

    if [ -n "$ENV" ]; then
        query="$query env:${ENV}"
    fi

    echo "$query"
}

QUERY=$(build_query)

# Query session replays
query_replays() {
    local payload=$(jq -n \
        --arg query "$QUERY" \
        --argjson from "$((FROM * 1000))" \
        --argjson to "$((TO * 1000))" \
        --argjson limit "$LIMIT" \
        '{
            filter: {
                query: $query,
                from: ($from | tostring),
                to: ($to | tostring)
            },
            page: {
                limit: $limit
            },
            sort: "-@session.duration"
        }')

    response=$(curl -s -X POST \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "${API_BASE}/api/v2/rum/events/search")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    # Check if data exists
    if ! echo "$response" | jq -e '.data' > /dev/null 2>&1 || [ "$(echo "$response" | jq '.data | length')" -eq 0 ]; then
        echo "{\"status\":\"no_data\",\"query\":\"$QUERY\",\"message\":\"No session replay data found\"}"
        exit 0
    fi

    # Parse session replay data
    echo "$response" | jq \
        --arg duration "$DURATION" \
        --arg query "$QUERY" \
        --arg env "$ENV" \
        '{
            status: "success",
            duration: $duration,
            environment: $env,
            query: $query,
            session_summary: {
                total_sessions: (.data | length),
                total_duration_sec: ([.data[].attributes.session.duration] | add // 0 | . / 1000000000),
                avg_duration_sec: (if (.data | length) > 0 then ([.data[].attributes.session.duration] | add / length / 1000000000 | floor) else 0 end),
                sessions_with_errors: ([.data[] | select(.attributes.session.error_count > 0)] | length),
                sessions_with_frustrations: ([.data[] | select(.attributes.session.frustration_count > 0)] | length)
            },
            sessions_with_issues: [
                .data[]
                | select(.attributes.session.error_count > 0 or .attributes.session.frustration_count > 0)
                | {
                    session_id: .attributes.session.id,
                    user_id: .attributes.usr.id,
                    duration_sec: (.attributes.session.duration / 1000000000 | floor),
                    error_count: .attributes.session.error_count,
                    frustration_count: .attributes.session.frustration_count,
                    view_count: .attributes.session.view_count,
                    action_count: .attributes.session.action_count,
                    application_id: .attributes.application.id,
                    replay_url: "https://app.\($env).datadoghq.com/rum/replay/sessions/\(.attributes.session.id)"
                }
            ] | sort_by(-.error_count),
            longest_sessions: [
                .data[]
                | {
                    session_id: .attributes.session.id,
                    duration_sec: (.attributes.session.duration / 1000000000 | floor),
                    view_count: .attributes.session.view_count,
                    action_count: .attributes.session.action_count,
                    user_id: .attributes.usr.id
                }
            ] | sort_by(-.duration_sec) | .[0:10],
            by_browser: (
                [.data[] | .attributes.device.browser.name]
                | group_by(.)
                | map({
                    browser: .[0],
                    count: length
                })
                | sort_by(-.count)
            ),
            recommendations: [
                (if ([.data[] | select(.attributes.session.error_count > 0)] | length) > 5
                    then "Multiple sessions with errors - review error patterns"
                    else null end),
                (if ([.data[] | select(.attributes.session.frustration_count > 0)] | length) > 5
                    then "User frustration signals detected - investigate UX issues"
                    else null end),
                "Watch session replays for context on error conditions",
                "Focus on high-error sessions for debugging"
            ] | map(select(. != null))
        }'
}

# Query session replays
query_replays
