#!/bin/bash
# Query Datadog Application Security Management (ASM)
# Analyze application security threats, attacks, and vulnerabilities

set -e

# Configuration
DD_SITE=${DD_SITE:-datadoghq.com}
API_BASE="https://api.${DD_SITE}"

# Default values
DURATION="24h"
SERVICE=""
ENV="prod"
ATTACK_TYPE=""
LIMIT=100

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
        --attack-type)
            ATTACK_TYPE="$2"
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
    local query="@appsec.type:*"

    if [ -n "$SERVICE" ]; then
        query="$query service:${SERVICE}"
    fi

    if [ -n "$ENV" ]; then
        query="$query env:${ENV}"
    fi

    if [ -n "$ATTACK_TYPE" ]; then
        query="$query @appsec.attack_type:${ATTACK_TYPE}"
    fi

    echo "$query"
}

QUERY=$(build_query)

# Query ASM threats
query_threats() {
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
            sort: "-timestamp"
        }')

    response=$(curl -s -X POST \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "${API_BASE}/api/v2/logs/events/search")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    # Check if data exists
    if ! echo "$response" | jq -e '.data' > /dev/null 2>&1 || [ "$(echo "$response" | jq '.data | length')" -eq 0 ]; then
        echo "{\"status\":\"no_data\",\"query\":\"$QUERY\",\"message\":\"No ASM threat data found\"}"
        exit 0
    fi

    # Parse ASM data
    echo "$response" | jq \
        --arg duration "$DURATION" \
        --arg query "$QUERY" \
        --arg env "$ENV" \
        '{
            status: "success",
            duration: $duration,
            environment: $env,
            query: $query,
            threat_summary: {
                total_attacks: (.data | length),
                unique_ips: ([.data[].attributes.network.client.ip] | unique | length),
                attack_types: ([.data[].attributes.appsec.attack_type] | group_by(.) | map({type: .[0], count: length})),
                services_targeted: ([.data[].attributes.service] | unique | length)
            },
            recent_attacks: [
                .data[]
                | {
                    timestamp: .attributes.timestamp,
                    service: .attributes.service,
                    attack_type: .attributes.appsec.attack_type,
                    rule_name: .attributes.appsec.rule.name,
                    attacker_ip: .attributes.network.client.ip,
                    http_method: .attributes.http.method,
                    url_path: .attributes.http.url_path,
                    user_agent: .attributes.http.useragent,
                    threat_score: .attributes.appsec.threat_score
                }
            ] | sort_by(.timestamp) | reverse | .[0:20],
            top_attackers: (
                [.data[] | .attributes.network.client.ip]
                | group_by(.)
                | map({
                    ip: .[0],
                    attack_count: length
                })
                | sort_by(-.attack_count)
                | .[0:10]
            ),
            by_attack_type: (
                [.data[] | .attributes.appsec.attack_type]
                | group_by(.)
                | map({
                    attack_type: .[0],
                    count: length
                })
                | sort_by(-.count)
            ),
            by_service: (
                [.data[] | .attributes.service]
                | group_by(.)
                | map({
                    service: .[0],
                    attack_count: length
                })
                | sort_by(-.attack_count)
            ),
            recommendations: [
                (if ([.data[] | .attributes.network.client.ip] | group_by(.) | map(length) | max) > 10
                    then "Repeated attacks from same IPs - consider IP blocking"
                    else null end),
                (if ([.data[] | select(.attributes.appsec.attack_type == "sql_injection")] | length) > 5
                    then "SQL injection attempts detected - review input validation"
                    else null end),
                (if ([.data[] | select(.attributes.appsec.attack_type == "xss")] | length) > 5
                    then "XSS attempts detected - review output encoding"
                    else null end),
                "Enable ASM blocking mode for high-severity threats",
                "Review and tune ASM rules based on attack patterns"
            ] | map(select(. != null))
        }'
}

# Query ASM threats
query_threats
