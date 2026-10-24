#!/bin/bash
# Query Datadog Anomaly Detection
# Detects anomalies across metrics, logs, traces, and infrastructure

set -e

# Configuration
DD_SITE=${DD_SITE:-datadoghq.com}
API_BASE="https://api.${DD_SITE}"

# Default values
DURATION="24h"
SERVICE=""
METRIC=""
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
        --metric)
            METRIC="$2"
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

# Build query filter
build_query() {
    local filters=""

    if [ -n "$SERVICE" ]; then
        filters="service:${SERVICE}"
    fi

    if [ -n "$TAG_FILTER" ]; then
        if [ -n "$filters" ]; then
            filters="${filters} AND ${TAG_FILTER}"
        else
            filters="${TAG_FILTER}"
        fi
    fi

    if [ -z "$filters" ]; then
        filters="*"
    fi

    echo "$filters"
}

QUERY=$(build_query)

# Get monitors with anomaly detection
query_anomaly_monitors() {
    local monitors_response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v1/monitor?monitor_tags=type:anomaly")

    echo "$monitors_response"
}

# Query specific metric with anomaly detection
query_metric_anomaly() {
    local metric_name="${METRIC:-system.cpu.user}"

    # Use anomaly function in query
    local anomaly_query="anomalies(avg:${metric_name}{${QUERY}}, 'basic', 2)"

    local response=$(curl -s -G \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        --data-urlencode "query=${anomaly_query}" \
        --data-urlencode "from=${FROM}" \
        --data-urlencode "to=${TO}" \
        "${API_BASE}/api/v1/query")

    echo "$response"
}

# Main anomaly detection
detect_anomalies() {
    # Get anomaly monitors
    local monitors=$(query_anomaly_monitors)

    # Query metric anomalies if metric specified
    local metric_anomalies="{}"
    if [ -n "$METRIC" ] || [ -n "$SERVICE" ]; then
        metric_anomalies=$(query_metric_anomaly)
    fi

    # Check for errors
    if echo "$monitors" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$monitors" | jq '{status: "error", message: .errors[0]}'
        exit 1
    fi

    # Combine and format results
    echo "$monitors" | jq \
        --argjson metric "$metric_anomalies" \
        --arg query "$QUERY" \
        --arg duration "$DURATION" \
        '{
            status: "success",
            query: $query,
            duration: $duration,
            anomaly_detection: {
                monitors: {
                    total: (. | length),
                    anomaly_monitors: [.[] | select(.type == "query alert" and (.query | contains("anomalies("))) | {
                        id: .id,
                        name: .name,
                        query: .query,
                        status: .overall_state,
                        message: .message,
                        tags: .tags
                    }]
                },
                metric_anomalies: (if $metric.series then {
                    series: $metric.series,
                    detected: ($metric.series | length > 0)
                } else null end)
            },
            summary: {
                active_anomaly_monitors: ([.[] | select(.overall_state != "OK")] | length),
                total_anomaly_monitors: (. | length),
                alert_rate: (if (. | length) > 0 then ([.[] | select(.overall_state != "OK")] | length) / (. | length) * 100 else 0 end)
            }
        }'
}

# Execute detection
detect_anomalies
