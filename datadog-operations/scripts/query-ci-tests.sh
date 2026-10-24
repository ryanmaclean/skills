#!/bin/bash
# Query Datadog CI Visibility Test Runs
# Analyze test performance, failures, and flakiness

set -e

# Configuration
DD_SITE=${DD_SITE:-datadoghq.com}
API_BASE="https://api.${DD_SITE}"

# Default values
DURATION="24h"
SERVICE=""
ENV="prod"
STATUS=""
BRANCH=""
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
        --status)
            STATUS="$2"
            shift 2
            ;;
        --branch)
            BRANCH="$2"
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

# Convert duration to timestamp
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
    local query="@test.status:*"

    if [ -n "$SERVICE" ]; then
        query="$query @test.service:${SERVICE}"
    fi

    if [ -n "$ENV" ]; then
        query="$query @test.environment:${ENV}"
    fi

    if [ -n "$STATUS" ]; then
        query="$query @test.status:${STATUS}"
    fi

    if [ -n "$BRANCH" ]; then
        query="$query @git.branch:${BRANCH}"
    fi

    echo "$query"
}

QUERY=$(build_query)

# Query CI test runs
query_tests() {
    local payload=$(jq -n \
        --arg query "$QUERY" \
        --argjson from "$FROM" \
        --argjson to "$TO" \
        --argjson limit "$LIMIT" \
        '{
            filter: {
                query: $query,
                from: ($from * 1000 | tostring),
                to: ($to * 1000 | tostring)
            },
            page: {
                limit: $limit
            },
            sort: "-@test.duration"
        }')

    response=$(curl -s -X POST \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "${API_BASE}/api/v2/ci/tests/events/search")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    # Parse test results
    echo "$response" | jq \
        --arg duration "$DURATION" \
        --arg query "$QUERY" \
        '{
            status: "success",
            duration: $duration,
            query: $query,
            test_summary: {
                total_tests: (.data | length),
                passed: ([.data[] | select(.attributes.test.status == "pass")] | length),
                failed: ([.data[] | select(.attributes.test.status == "fail")] | length),
                skipped: ([.data[] | select(.attributes.test.status == "skip")] | length),
                total_duration_ms: ([.data[].attributes.test.duration] | add // 0),
                avg_duration_ms: (if (.data | length) > 0 then ([.data[].attributes.test.duration] | add / length | floor) else 0 end)
            },
            flaky_tests: [
                .data[]
                | select(.attributes.test.is_flaky == true)
                | {
                    name: .attributes.test.name,
                    suite: .attributes.test.suite,
                    service: .attributes.test.service,
                    duration_ms: .attributes.test.duration,
                    branch: .attributes.git.branch
                }
            ],
            slowest_tests: [
                .data[]
                | {
                    name: .attributes.test.name,
                    suite: .attributes.test.suite,
                    service: .attributes.test.service,
                    status: .attributes.test.status,
                    duration_ms: .attributes.test.duration,
                    branch: .attributes.git.branch
                }
            ] | sort_by(-.duration_ms) | .[0:10],
            failed_tests: [
                .data[]
                | select(.attributes.test.status == "fail")
                | {
                    name: .attributes.test.name,
                    suite: .attributes.test.suite,
                    service: .attributes.test.service,
                    error_message: .attributes.test.error_message,
                    branch: .attributes.git.branch,
                    commit: .attributes.git.commit.sha
                }
            ],
            by_service: (
                [.data[] | .attributes.test.service]
                | group_by(.)
                | map({
                    service: .[0],
                    count: length
                })
            ),
            recommendations: [
                (if ([.data[] | select(.attributes.test.is_flaky == true)] | length) > 0
                    then "Fix \([.data[] | select(.attributes.test.is_flaky == true)] | length) flaky tests to improve reliability"
                    else null end),
                (if ([.data[] | select(.attributes.test.status == "fail")] | length) > 5
                    then "High failure rate detected - investigate common failures"
                    else null end),
                (if (([.data[].attributes.test.duration] | add // 0) / 1000) > 300
                    then "Total test duration > 5 minutes - consider parallelization"
                    else null end)
            ] | map(select(. != null))
        }'
}

# Get test statistics
get_statistics() {
    local payload=$(jq -n \
        --arg query "$QUERY" \
        --argjson from "$FROM" \
        --argjson to "$TO" \
        '{
            filter: {
                query: $query,
                from: ($from * 1000 | tostring),
                to: ($to * 1000 | tostring)
            },
            compute: [
                {
                    aggregation: "count",
                    metric: "@test.status"
                },
                {
                    aggregation: "avg",
                    metric: "@test.duration"
                }
            ],
            group_by: [
                {
                    facet: "@test.service"
                }
            ]
        }')

    response=$(curl -s -X POST \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "${API_BASE}/api/v2/ci/tests/analytics/aggregate")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0].detail}'
        exit 1
    fi

    echo "$response" | jq '{
        status: "success",
        statistics: {
            by_service: [
                .data[]
                | {
                    service: .by["@test.service"],
                    test_count: .compute.c0,
                    avg_duration_ms: (.compute.c1 | floor)
                }
            ]
        }
    }'
}

# Query tests by default
query_tests
