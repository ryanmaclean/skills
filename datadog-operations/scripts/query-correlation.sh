#!/bin/bash
# Query Datadog Event Correlation
# Correlates events across multiple signals for root cause analysis

set -e

# Configuration
DD_SITE=${DD_SITE:-datadoghq.com}
API_BASE="https://api.${DD_SITE}"

# Default values
DURATION="1h"
SERVICE=""
EVENT_TYPE=""
TAG_FILTER=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --duration)
            DURATION="$2"
            shift 2
            ;;
        --service)
            SERVICE="$2"
            shift 2
            ;;
        --event-type)
            EVENT_TYPE="$2"
            shift 2
            ;;
        --tags)
            TAG_FILTER="$2"
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
    local seconds=3600

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

# Build query filter
build_tags() {
    local tags=""

    if [ -n "$SERVICE" ]; then
        tags="service:${SERVICE}"
    fi

    if [ -n "$TAG_FILTER" ]; then
        if [ -n "$tags" ]; then
            tags="${tags},${TAG_FILTER}"
        else
            tags="${TAG_FILTER}"
        fi
    fi

    echo "$tags"
}

TAGS=$(build_tags)

# Query events
query_events() {
    local query_params="start=${FROM}&end=${TO}"

    if [ -n "$TAGS" ]; then
        query_params="${query_params}&tags=${TAGS}"
    fi

    if [ -n "$EVENT_TYPE" ]; then
        query_params="${query_params}&sources=${EVENT_TYPE}"
    fi

    local response=$(curl -s -G \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v1/events?${query_params}")

    echo "$response"
}

# Query APM errors
query_apm_errors() {
    local service_filter=""
    if [ -n "$SERVICE" ]; then
        service_filter="service:${SERVICE}"
    else
        service_filter="*"
    fi

    local query="trace.servlet.request.errors{${service_filter}} by {service,resource_name}.as_count()"
    local response=$(curl -s -G \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        --data-urlencode "query=${query}" \
        --data-urlencode "from=${FROM}" \
        --data-urlencode "to=${TO}" \
        "${API_BASE}/api/v1/query")

    echo "$response"
}

# Query logs
query_error_logs() {
    local log_filter=""
    if [ -n "$SERVICE" ]; then
        log_filter="service:${SERVICE} status:error"
    else
        log_filter="status:error"
    fi

    local payload=$(jq -n \
        --arg query "$log_filter" \
        --argjson from "$FROM" \
        --argjson to "$TO" \
        '{
            query: $query,
            time: {
                from: ($from | tostring + "000"),
                to: ($to | tostring + "000")
            },
            sort: "desc",
            limit: 100
        }')

    local response=$(curl -s -X POST \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "${API_BASE}/api/v2/logs/events/search")

    echo "$response"
}

# Query monitors
query_monitor_alerts() {
    local response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v1/monitor")

    # Filter to only recent alerts
    echo "$response" | jq --argjson from "$FROM" \
        '[.[] | select(.overall_state != "OK" and (.modified // 0) > $from)]'
}

# Correlate all signals
correlate_events() {
    # Gather all signals
    local events=$(query_events)
    local apm_errors=$(query_apm_errors)
    local error_logs=$(query_error_logs)
    local monitor_alerts=$(query_monitor_alerts)

    # Check for errors
    if echo "$events" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$events" | jq '{status: "error", message: .errors[0]}'
        exit 1
    fi

    # Correlate and format results
    jq -n \
        --argjson events "$events" \
        --argjson apm "$apm_errors" \
        --argjson logs "$error_logs" \
        --argjson monitors "$monitor_alerts" \
        --arg service "${SERVICE:-all}" \
        --arg duration "$DURATION" \
        '{
            status: "success",
            service: $service,
            duration: $duration,
            correlation: {
                events: {
                    total: ($events.events | length),
                    by_type: ($events.events | group_by(.source) | map({
                        source: .[0].source,
                        count: length
                    })),
                    recent: ($events.events[0:10] | map({
                        title: .title,
                        text: .text,
                        date_happened: .date_happened,
                        source: .source,
                        tags: .tags
                    }))
                },
                apm_errors: {
                    total: (if $apm.series then [$apm.series[].pointlist[]?[1]] | add else 0 end),
                    by_service: (if $apm.series then [$apm.series[] | {
                        service: (.scope // "" | split(",") | map(select(startswith("service:"))) | .[0] // "" | split(":")[1] // ""),
                        errors: ([.pointlist[]?[1]] | add)
                    }] else [] end)
                },
                error_logs: {
                    total: ($logs.meta.page.total // 0),
                    by_service: (if $logs.data then ($logs.data | group_by(.attributes.service) | map({
                        service: (.[0].attributes.service // "unknown"),
                        count: length
                    })) else [] end)
                },
                monitor_alerts: {
                    total: ($monitors | length),
                    by_severity: ($monitors | group_by(.priority // 3) | map({
                        priority: .[0].priority,
                        count: length
                    })),
                    active: ($monitors | map({
                        id: .id,
                        name: .name,
                        state: .overall_state,
                        modified: .modified
                    }))
                }
            },
            summary: {
                total_signals: (($events.events | length) + (if $apm.series then ($apm.series | length) else 0 end) + ($logs.meta.page.total // 0) + ($monitors | length)),
                event_count: ($events.events | length),
                apm_error_count: (if $apm.series then [$apm.series[].pointlist[]?[1]] | add else 0 end),
                log_error_count: ($logs.meta.page.total // 0),
                monitor_alert_count: ($monitors | length),
                correlation_strength: (if (($events.events | length) + (if $apm.series then ($apm.series | length) else 0 end) + ($logs.meta.page.total // 0)) > 5 then "high" elif (($events.events | length) + ($logs.meta.page.total // 0)) > 2 then "medium" else "low" end)
            }
        }'
}

# Execute correlation
correlate_events
