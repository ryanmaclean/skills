#!/bin/bash
# Query Datadog Network Performance Monitoring
# Retrieves network flow data, connections, and latency metrics

set -e

# Configuration
DD_SITE=${DD_SITE:-datadoghq.com}
API_BASE="https://api.${DD_SITE}"

# Default values
DURATION="1h"
SOURCE=""
DEST=""
PORT=""
TAG_FILTER=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --duration)
            DURATION="$2"
            shift 2
            ;;
        --source)
            SOURCE="$2"
            shift 2
            ;;
        --dest|--destination)
            DEST="$2"
            shift 2
            ;;
        --port)
            PORT="$2"
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
build_query() {
    local filters=""

    if [ -n "$SOURCE" ]; then
        filters="source_service:${SOURCE}"
    fi

    if [ -n "$DEST" ]; then
        if [ -n "$filters" ]; then
            filters="${filters} AND dest_service:${DEST}"
        else
            filters="dest_service:${DEST}"
        fi
    fi

    if [ -n "$PORT" ]; then
        if [ -n "$filters" ]; then
            filters="${filters} AND dest_port:${PORT}"
        else
            filters="dest_port:${PORT}"
        fi
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

# Query network metrics
query_network() {
    # Bytes sent
    local bytes_sent_query="sum:network.bytes_sent{${QUERY}} by {source_service,dest_service}"
    local bytes_sent_response=$(curl -s -G \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        --data-urlencode "query=${bytes_sent_query}" \
        --data-urlencode "from=${FROM}" \
        --data-urlencode "to=${TO}" \
        "${API_BASE}/api/v1/query")

    # Bytes received
    local bytes_rcvd_query="sum:network.bytes_rcvd{${QUERY}} by {source_service,dest_service}"
    local bytes_rcvd_response=$(curl -s -G \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        --data-urlencode "query=${bytes_rcvd_query}" \
        --data-urlencode "from=${FROM}" \
        --data-urlencode "to=${TO}" \
        "${API_BASE}/api/v1/query")

    # TCP latency
    local latency_query="avg:network.tcp.rtt{${QUERY}} by {source_service,dest_service}"
    local latency_response=$(curl -s -G \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        --data-urlencode "query=${latency_query}" \
        --data-urlencode "from=${FROM}" \
        --data-urlencode "to=${TO}" \
        "${API_BASE}/api/v1/query")

    # Connection count
    local conn_query="sum:network.tcp.connections{${QUERY}} by {source_service,dest_service}"
    local conn_response=$(curl -s -G \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        --data-urlencode "query=${conn_query}" \
        --data-urlencode "from=${FROM}" \
        --data-urlencode "to=${TO}" \
        "${API_BASE}/api/v1/query")

    # Retransmits
    local retrans_query="sum:network.tcp.retransmits{${QUERY}} by {source_service,dest_service}"
    local retrans_response=$(curl -s -G \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        --data-urlencode "query=${retrans_query}" \
        --data-urlencode "from=${FROM}" \
        --data-urlencode "to=${TO}" \
        "${API_BASE}/api/v1/query")

    # Check for errors
    if echo "$bytes_sent_response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$bytes_sent_response" | jq '{status: "error", message: .errors[0]}'
        exit 1
    fi

    # Combine and format results
    jq -n \
        --argjson sent "$bytes_sent_response" \
        --argjson rcvd "$bytes_rcvd_response" \
        --argjson lat "$latency_response" \
        --argjson conn "$conn_response" \
        --argjson retrans "$retrans_response" \
        --arg query "$QUERY" \
        --arg duration "$DURATION" \
        '{
            status: "success",
            query: $query,
            duration: $duration,
            network: {
                bytes_sent: {
                    metric: "network.bytes_sent",
                    series: $sent.series,
                    total: (if $sent.series then [$sent.series[].pointlist[]?[1]] | add else 0 end)
                },
                bytes_received: {
                    metric: "network.bytes_rcvd",
                    series: $rcvd.series,
                    total: (if $rcvd.series then [$rcvd.series[].pointlist[]?[1]] | add else 0 end)
                },
                tcp_latency: {
                    metric: "network.tcp.rtt",
                    series: $lat.series,
                    avg_ms: (if $lat.series then [$lat.series[].pointlist[]?[1]] | add / length else 0 end)
                },
                connections: {
                    metric: "network.tcp.connections",
                    series: $conn.series,
                    total: (if $conn.series then [$conn.series[].pointlist[]?[1]] | add else 0 end)
                },
                retransmits: {
                    metric: "network.tcp.retransmits",
                    series: $retrans.series,
                    total: (if $retrans.series then [$retrans.series[].pointlist[]?[1]] | add else 0 end)
                }
            },
            summary: {
                total_bytes_sent: (if $sent.series then [$sent.series[].pointlist[]?[1]] | add else 0 end),
                total_bytes_received: (if $rcvd.series then [$rcvd.series[].pointlist[]?[1]] | add else 0 end),
                avg_latency_ms: (if $lat.series then [$lat.series[].pointlist[]?[1]] | add / length else 0 end),
                total_connections: (if $conn.series then [$conn.series[].pointlist[]?[1]] | add else 0 end),
                total_retransmits: (if $retrans.series then [$retrans.series[].pointlist[]?[1]] | add else 0 end),
                flow_count: (if $sent.series then ($sent.series | length) else 0 end)
            }
        }'
}

# Execute query
query_network
