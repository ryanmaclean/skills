#!/bin/bash
# Query Datadog Data Streams Monitoring
# Monitor Kafka, Kinesis, and other streaming data pipelines

set -e

# Configuration
DD_SITE=${DD_SITE:-datadoghq.com}
API_BASE="https://api.${DD_SITE}"

# Default values
DURATION="1h"
SERVICE=""
ENV="prod"
STREAM_TYPE="kafka"

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
        --env)
            ENV="$2"
            shift 2
            ;;
        --type)
            STREAM_TYPE="$2"
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

# Build query
build_query() {
    local query="*"

    if [ -n "$SERVICE" ]; then
        query="$query service:${SERVICE}"
    fi

    if [ -n "$ENV" ]; then
        query="$query env:${ENV}"
    fi

    if [ -n "$STREAM_TYPE" ]; then
        query="$query stream_type:${STREAM_TYPE}"
    fi

    echo "$query"
}

QUERY=$(build_query)

# Query data streams
query_streams() {
    # Query data streams monitoring metrics
    response=$(curl -s -G \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        --data-urlencode "query=${QUERY}" \
        --data-urlencode "from=${FROM}" \
        --data-urlencode "to=${TO}" \
        "${API_BASE}/api/v2/data-streams/pipelines")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        # Try alternative metrics query for stream data
        query_stream_metrics
        return
    fi

    # Check if data exists
    if ! echo "$response" | jq -e '.data' > /dev/null 2>&1 || [ "$(echo "$response" | jq '.data | length')" -eq 0 ]; then
        query_stream_metrics
        return
    fi

    # Parse streaming data
    echo "$response" | jq \
        --arg duration "$DURATION" \
        --arg query "$QUERY" \
        --arg env "$ENV" \
        '{
            status: "success",
            duration: $duration,
            environment: $env,
            query: $query,
            pipeline_summary: {
                total_pipelines: (.data | length),
                total_throughput: ([.data[].attributes.throughput] | add // 0),
                avg_latency_ms: (if (.data | length) > 0 then ([.data[].attributes.latency_ms] | add / length | floor) else 0 end)
            },
            pipelines: [
                .data[]
                | {
                    pipeline_id: .id,
                    name: .attributes.name,
                    service: .attributes.service,
                    stream_type: .attributes.stream_type,
                    throughput_msgs_per_sec: .attributes.throughput,
                    latency_ms: .attributes.latency_ms,
                    lag: .attributes.lag,
                    error_rate: .attributes.error_rate,
                    consumers: .attributes.consumers,
                    topics: .attributes.topics
                }
            ],
            high_latency_pipelines: [
                .data[]
                | select(.attributes.latency_ms > 1000)
                | {
                    name: .attributes.name,
                    latency_ms: .attributes.latency_ms,
                    service: .attributes.service
                }
            ],
            high_lag_pipelines: [
                .data[]
                | select(.attributes.lag > 10000)
                | {
                    name: .attributes.name,
                    lag: .attributes.lag,
                    service: .attributes.service
                }
            ],
            recommendations: [
                (if ([.data[] | select(.attributes.latency_ms > 1000)] | length) > 0
                    then "High latency detected in \([.data[] | select(.attributes.latency_ms > 1000)] | length) pipelines"
                    else null end),
                (if ([.data[] | select(.attributes.lag > 10000)] | length) > 0
                    then "Consumer lag detected - scale consumers or optimize processing"
                    else null end),
                (if ([.data[] | select(.attributes.error_rate > 0.01)] | length) > 0
                    then "Error rate > 1% detected - investigate pipeline errors"
                    else null end)
            ] | map(select(. != null))
        }'
}

# Query stream metrics as fallback
query_stream_metrics() {
    # Query Kafka metrics
    local kafka_query="avg:kafka.consumer_lag{$([ -n "$SERVICE" ] && echo "service:${SERVICE}," || echo "")env:${ENV}} by {topic,partition}"

    response=$(curl -s -G \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        --data-urlencode "query=${kafka_query}" \
        --data-urlencode "from=${FROM}" \
        --data-urlencode "to=${TO}" \
        "${API_BASE}/api/v1/query")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo '{"status":"no_data","message":"No data streams monitoring data available"}'
        exit 0
    fi

    # Parse metrics
    echo "$response" | jq \
        --arg duration "$DURATION" \
        --arg env "$ENV" \
        --arg stream_type "$STREAM_TYPE" \
        '{
            status: "success",
            duration: $duration,
            environment: $env,
            stream_type: $stream_type,
            kafka_metrics: {
                total_topics: (.series | length),
                consumer_lag: [
                    .series[]
                    | {
                        topic: (.scope // "" | split(",") | .[] | select(startswith("topic:")) | split(":")[1]),
                        partition: (.scope // "" | split(",") | .[] | select(startswith("partition:")) | split(":")[1]),
                        avg_lag: ([.pointlist[]?[1]] | add / length | floor),
                        max_lag: ([.pointlist[]?[1]] | max)
                    }
                ],
                high_lag_topics: [
                    .series[]
                    | select(([.pointlist[]?[1]] | max) > 1000)
                    | {
                        topic: (.scope // "" | split(",") | .[] | select(startswith("topic:")) | split(":")[1]),
                        max_lag: ([.pointlist[]?[1]] | max)
                    }
                ]
            },
            recommendations: [
                (if ([.series[] | select(([.pointlist[]?[1]] | max) > 1000)] | length) > 0
                    then "High consumer lag detected - consider scaling consumers"
                    else null end),
                "Monitor throughput and latency metrics for streaming pipelines",
                "Set up alerts for consumer lag thresholds"
            ] | map(select(. != null))
        }'
}

# Query streams
query_streams
