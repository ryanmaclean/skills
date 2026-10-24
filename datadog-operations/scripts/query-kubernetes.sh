#!/bin/bash
# Query Datadog Kubernetes Monitoring
# Retrieves K8s pod, deployment, and cluster metrics

set -e

# Configuration
DD_SITE=${DD_SITE:-datadoghq.com}
API_BASE="https://api.${DD_SITE}"

# Default values
DURATION="1h"
NAMESPACE=""
POD=""
DEPLOYMENT=""
CLUSTER=""
TAG_FILTER=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --duration)
            DURATION="$2"
            shift 2
            ;;
        --namespace)
            NAMESPACE="$2"
            shift 2
            ;;
        --pod)
            POD="$2"
            shift 2
            ;;
        --deployment)
            DEPLOYMENT="$2"
            shift 2
            ;;
        --cluster)
            CLUSTER="$2"
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

    if [ -n "$NAMESPACE" ]; then
        filters="kube_namespace:${NAMESPACE}"
    fi

    if [ -n "$POD" ]; then
        if [ -n "$filters" ]; then
            filters="${filters} AND pod_name:${POD}"
        else
            filters="pod_name:${POD}"
        fi
    fi

    if [ -n "$DEPLOYMENT" ]; then
        if [ -n "$filters" ]; then
            filters="${filters} AND kube_deployment:${DEPLOYMENT}"
        else
            filters="kube_deployment:${DEPLOYMENT}"
        fi
    fi

    if [ -n "$CLUSTER" ]; then
        if [ -n "$filters" ]; then
            filters="${filters} AND kube_cluster_name:${CLUSTER}"
        else
            filters="kube_cluster_name:${CLUSTER}"
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

# Query Kubernetes metrics
query_kubernetes() {
    # Pod CPU usage
    local cpu_query="avg:kubernetes.cpu.usage.total{${QUERY}} by {pod_name,kube_namespace,kube_deployment}"
    local cpu_response=$(curl -s -G \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        --data-urlencode "query=${cpu_query}" \
        --data-urlencode "from=${FROM}" \
        --data-urlencode "to=${TO}" \
        "${API_BASE}/api/v1/query")

    # Pod memory usage
    local mem_query="avg:kubernetes.memory.usage{${QUERY}} by {pod_name,kube_namespace,kube_deployment}"
    local mem_response=$(curl -s -G \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        --data-urlencode "query=${mem_query}" \
        --data-urlencode "from=${FROM}" \
        --data-urlencode "to=${TO}" \
        "${API_BASE}/api/v1/query")

    # Pod restart count
    local restart_query="sum:kubernetes.pods.running{${QUERY}} by {kube_namespace,kube_deployment}"
    local restart_response=$(curl -s -G \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        --data-urlencode "query=${restart_query}" \
        --data-urlencode "from=${FROM}" \
        --data-urlencode "to=${TO}" \
        "${API_BASE}/api/v1/query")

    # Node status
    local node_query="avg:kubernetes.kubelet.running_pods{${QUERY}} by {host,kube_cluster_name}"
    local node_response=$(curl -s -G \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        --data-urlencode "query=${node_query}" \
        --data-urlencode "from=${FROM}" \
        --data-urlencode "to=${TO}" \
        "${API_BASE}/api/v1/query")

    # Check for errors
    if echo "$cpu_response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$cpu_response" | jq '{status: "error", message: .errors[0]}'
        exit 1
    fi

    # Combine and format results
    jq -n \
        --argjson cpu "$cpu_response" \
        --argjson mem "$mem_response" \
        --argjson restart "$restart_response" \
        --argjson node "$node_response" \
        --arg query "$QUERY" \
        --arg duration "$DURATION" \
        '{
            status: "success",
            query: $query,
            duration: $duration,
            kubernetes: {
                pods: {
                    cpu_usage: {
                        metric: "kubernetes.cpu.usage.total",
                        series: $cpu.series,
                        avg_nanocores: (if $cpu.series then [$cpu.series[].pointlist[]?[1]] | add / length else 0 end)
                    },
                    memory_usage: {
                        metric: "kubernetes.memory.usage",
                        series: $mem.series,
                        avg_bytes: (if $mem.series then [$mem.series[].pointlist[]?[1]] | add / length else 0 end)
                    },
                    running_count: {
                        metric: "kubernetes.pods.running",
                        series: $restart.series,
                        current: (if $restart.series then [$restart.series[].pointlist[-1]?[1]] | add else 0 end)
                    }
                },
                nodes: {
                    running_pods: {
                        metric: "kubernetes.kubelet.running_pods",
                        series: $node.series,
                        total: (if $node.series then [$node.series[].pointlist[]?[1]] | add / length else 0 end)
                    }
                }
            },
            summary: {
                pod_count: (if $cpu.series then ($cpu.series | length) else 0 end),
                avg_cpu_nanocores: (if $cpu.series then [$cpu.series[].pointlist[]?[1]] | add / length else 0 end),
                avg_memory_mb: (if $mem.series then ([$mem.series[].pointlist[]?[1]] | add / length) / 1048576 else 0 end),
                node_count: (if $node.series then ($node.series | length) else 0 end),
                namespaces: (if $cpu.series then [$cpu.series[].scope // "" | split(",") | .[] | select(startswith("kube_namespace:")) | split(":")[1]] | unique else [] end)
            }
        }'
}

# Execute query
query_kubernetes
