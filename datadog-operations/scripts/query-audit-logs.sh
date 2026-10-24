#!/bin/bash
# Query Datadog Audit Logs
# Retrieve audit logs for compliance and security tracking

set -e

# Configuration
DD_SITE=${DD_SITE:-datadoghq.com}
API_BASE="https://api.${DD_SITE}"

# Default values
DURATION="24h"
QUERY=""
SORT="-timestamp"
LIMIT=100

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --duration)
            DURATION="$2"
            shift 2
            ;;
        --query)
            QUERY="$2"
            shift 2
            ;;
        --sort)
            SORT="$2"
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

# Convert duration to epoch timestamps
calculate_time_range() {
    local duration=$1
    local now=$(date +%s)
    local seconds=86400  # 24 hours default

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

# Convert to milliseconds
FROM_MS=$((FROM * 1000))
TO_MS=$((TO * 1000))

# Query audit logs
query_audit_logs() {
    # Build filter query
    local filter_query="${QUERY:-*}"

    # Build payload
    local payload=$(jq -n \
        --arg query "$filter_query" \
        --arg from "$FROM_MS" \
        --arg to "$TO_MS" \
        --arg sort "$SORT" \
        --argjson limit "$LIMIT" \
        '{
            filter: {
                query: $query,
                from: $from,
                to: $to
            },
            sort: $sort,
            page: {
                limit: $limit
            }
        }')

    response=$(curl -s -X POST \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "${API_BASE}/api/v2/audit/events/search")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    # Format response
    echo "$response" | jq --arg duration "$DURATION" '{
        status: "success",
        duration: $duration,
        total_events: (.meta.page.total_count // 0),
        events: [.data[] | {
            id: .id,
            type: .type,
            timestamp: .attributes.timestamp,
            message: .attributes.message,
            actor: {
                user: .attributes.attributes.actor.user,
                name: .attributes.attributes.actor.name,
                type: .attributes.attributes.actor.type
            },
            action: .attributes.attributes.action,
            resource: {
                type: .attributes.attributes.resource.type,
                name: .attributes.attributes.resource.name,
                id: .attributes.attributes.resource.id
            },
            tags: .attributes.tags
        }],
        summary: {
            by_action: ([.data[] | .attributes.attributes.action] | group_by(.) | map({action: .[0], count: length})),
            by_user: ([.data[] | .attributes.attributes.actor.user] | group_by(.) | map({user: .[0], count: length})),
            by_resource_type: ([.data[] | .attributes.attributes.resource.type] | group_by(.) | map({type: .[0], count: length}))
        }
    }'
}

# Execute query
query_audit_logs
