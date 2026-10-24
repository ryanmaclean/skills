#!/bin/bash
# Query Datadog Host Inventory
# Analyze host metadata, tags, and infrastructure state

set -e

# Configuration
DD_SITE=${DD_SITE:-datadoghq.com}
API_BASE="https://api.${DD_SITE}"

# Default values
HOST_NAME=""
TAG_FILTER=""
SORT_BY="name"
LIMIT=100

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --host)
            HOST_NAME="$2"
            shift 2
            ;;
        --tag)
            TAG_FILTER="$2"
            shift 2
            ;;
        --sort)
            SORT_BY="$2"
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

# List all hosts
list_hosts() {
    local url="${API_BASE}/api/v1/hosts"
    local params="count=${LIMIT}&sort=${SORT_BY}"

    if [ -n "$TAG_FILTER" ]; then
        params="${params}&filter=${TAG_FILTER}"
    fi

    response=$(curl -s -G \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        --data-urlencode "${params}" \
        "${url}")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0]}'
        exit 1
    fi

    # Parse host list
    echo "$response" | jq \
        --arg tag_filter "${TAG_FILTER:-none}" \
        '{
            status: "success",
            filter: $tag_filter,
            host_summary: {
                total_hosts: (.total_returned // 0),
                total_active: (.total_active // 0),
                total_up: (.total_up // 0)
            },
            hosts: [
                .host_list[]?
                | {
                    name: .name,
                    aliases: .aliases,
                    is_up: .is_up,
                    last_reported: .last_reported_time,
                    platform: .meta.platform,
                    cpu_cores: .meta.cpuCores,
                    agent_version: .meta.agent_version,
                    tags_by_source: .tags_by_source,
                    metrics: {
                        cpu: .metrics.cpu,
                        iowait: .metrics.iowait,
                        load: .metrics.load
                    }
                }
            ],
            by_platform: (
                [.host_list[]? | .meta.platform]
                | group_by(.)
                | map({
                    platform: .[0],
                    count: length
                })
            ),
            down_hosts: [
                .host_list[]?
                | select(.is_up == false)
                | {
                    name: .name,
                    last_reported: .last_reported_time,
                    platform: .meta.platform
                }
            ],
            recommendations: [
                (if ([.host_list[]? | select(.is_up == false)] | length) > 0
                    then "Investigate \([.host_list[]? | select(.is_up == false)] | length) down hosts"
                    else null end),
                (if ([.host_list[]? | select(.metrics.cpu > 80)] | length) > 0
                    then "High CPU usage detected on \([.host_list[]? | select(.metrics.cpu > 80)] | length) hosts"
                    else null end),
                "Review host tags for proper organization",
                "Ensure agent versions are up to date"
            ] | map(select(. != null))
        }'
}

# Get specific host details
get_host() {
    if [ -z "$HOST_NAME" ]; then
        echo '{"status":"error","message":"--host is required for get action"}' >&2
        exit 1
    fi

    response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v1/hosts/${HOST_NAME}")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0]}'
        exit 1
    fi

    # Parse host details
    echo "$response" | jq '{
        status: "success",
        host: {
            name: .host.name,
            aliases: .host.aliases,
            is_up: .host.is_up,
            last_reported: .host.last_reported_time,
            metadata: {
                platform: .host.meta.platform,
                cpu_cores: .host.meta.cpuCores,
                memory: .host.meta.memory,
                agent_version: .host.meta.agent_version,
                hostname: .host.meta.hostname,
                machine: .host.meta.machine,
                os: .host.meta.os
            },
            tags_by_source: .host.tags_by_source,
            apps: .host.apps,
            sources: .host.sources,
            metrics: {
                cpu: .host.metrics.cpu,
                iowait: .host.metrics.iowait,
                load: .host.metrics.load
            }
        }
    }'
}

# Mute host
mute_host() {
    if [ -z "$HOST_NAME" ]; then
        echo '{"status":"error","message":"--host is required for mute action"}' >&2
        exit 1
    fi

    local payload=$(jq -n \
        --arg message "Host muted via CLI" \
        '{
            message: $message,
            end: (now + 3600 | floor)
        }')

    response=$(curl -s -X POST \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "${API_BASE}/api/v1/host/${HOST_NAME}/mute")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0]}'
        exit 1
    fi

    echo "{\"status\":\"success\",\"message\":\"Host ${HOST_NAME} muted successfully\",\"action\":\"muted\"}"
}

# Unmute host
unmute_host() {
    if [ -z "$HOST_NAME" ]; then
        echo '{"status":"error","message":"--host is required for unmute action"}' >&2
        exit 1
    fi

    response=$(curl -s -X POST \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v1/host/${HOST_NAME}/unmute")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0]}'
        exit 1
    fi

    echo "{\"status\":\"success\",\"message\":\"Host ${HOST_NAME} unmuted successfully\",\"action\":\"unmuted\"}"
}

# Default to list
if [ -z "$HOST_NAME" ]; then
    list_hosts
else
    get_host
fi
