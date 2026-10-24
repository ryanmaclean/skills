#!/bin/bash
# Query Datadog CI/CD Visibility
# Retrieves CI pipeline and test execution data

set -e

# Configuration
DD_SITE=${DD_SITE:-datadoghq.com}
API_BASE="https://api.${DD_SITE}"

# Default values
DURATION="7d"
SERVICE=""
BRANCH=""
ENV=""
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
        --branch)
            BRANCH="$2"
            shift 2
            ;;
        --env)
            ENV="$2"
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
    local seconds=604800  # 7 days default

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

    if [ -n "$BRANCH" ]; then
        if [ -n "$filters" ]; then
            filters="${filters} AND git.branch:${BRANCH}"
        else
            filters="git.branch:${BRANCH}"
        fi
    fi

    if [ -n "$ENV" ]; then
        if [ -n "$filters" ]; then
            filters="${filters} AND env:${ENV}"
        else
            filters="env:${ENV}"
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

# Query CI/CD metrics
query_cicd() {
    # Pipeline duration
    local duration_query="avg:ci.pipeline.duration{${QUERY}} by {service,git.branch}"
    local duration_response=$(curl -s -G \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        --data-urlencode "query=${duration_query}" \
        --data-urlencode "from=${FROM}" \
        --data-urlencode "to=${TO}" \
        "${API_BASE}/api/v1/query")

    # Pipeline success rate
    local success_query="sum:ci.pipeline.finished{${QUERY},ci.status:success}.as_count()"
    local success_response=$(curl -s -G \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        --data-urlencode "query=${success_query}" \
        --data-urlencode "from=${FROM}" \
        --data-urlencode "to=${TO}" \
        "${API_BASE}/api/v1/query")

    # Pipeline failures
    local failure_query="sum:ci.pipeline.finished{${QUERY},ci.status:error}.as_count()"
    local failure_response=$(curl -s -G \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        --data-urlencode "query=${failure_query}" \
        --data-urlencode "from=${FROM}" \
        --data-urlencode "to=${TO}" \
        "${API_BASE}/api/v1/query")

    # Test execution time
    local test_duration_query="avg:ci.test.duration{${QUERY}} by {test.suite}"
    local test_duration_response=$(curl -s -G \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        --data-urlencode "query=${test_duration_query}" \
        --data-urlencode "from=${FROM}" \
        --data-urlencode "to=${TO}" \
        "${API_BASE}/api/v1/query")

    # Test failures
    local test_fail_query="sum:ci.test.finished{${QUERY},test.status:fail}.as_count()"
    local test_fail_response=$(curl -s -G \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        --data-urlencode "query=${test_fail_query}" \
        --data-urlencode "from=${FROM}" \
        --data-urlencode "to=${TO}" \
        "${API_BASE}/api/v1/query")

    # Check for errors
    if echo "$duration_response" | jq -e '.errors' > /dev/null 2>&1; then
        # If CI metrics don't exist, return empty results
        duration_response='{"series":[]}'
        success_response='{"series":[]}'
        failure_response='{"series":[]}'
        test_duration_response='{"series":[]}'
        test_fail_response='{"series":[]}'
    fi

    # Combine and format results
    jq -n \
        --argjson dur "$duration_response" \
        --argjson success "$success_response" \
        --argjson fail "$failure_response" \
        --argjson test_dur "$test_duration_response" \
        --argjson test_fail "$test_fail_response" \
        --arg query "$QUERY" \
        --arg duration "$DURATION" \
        '{
            status: "success",
            query: $query,
            duration: $duration,
            cicd: {
                pipelines: {
                    avg_duration_seconds: (if $dur.series then ([$dur.series[].pointlist[]?[1]] | add / length) else 0 end),
                    total_success: (if $success.series then [$success.series[].pointlist[]?[1]] | add else 0 end),
                    total_failures: (if $fail.series then [$fail.series[].pointlist[]?[1]] | add else 0 end),
                    success_rate: (if ($success.series and $fail.series) then
                        ([$success.series[].pointlist[]?[1]] | add) / (([$success.series[].pointlist[]?[1]] | add) + ([$fail.series[].pointlist[]?[1]] | add)) * 100
                        else 0 end)
                },
                tests: {
                    avg_duration_ms: (if $test_dur.series then ([$test_dur.series[].pointlist[]?[1]] | add / length) * 1000 else 0 end),
                    total_failures: (if $test_fail.series then [$test_fail.series[].pointlist[]?[1]] | add else 0 end),
                    suites: (if $test_dur.series then ($test_dur.series | length) else 0 end)
                }
            },
            summary: {
                pipeline_count: (if $dur.series then ($dur.series | length) else 0 end),
                avg_pipeline_duration_minutes: (if $dur.series then ([$dur.series[].pointlist[]?[1]] | add / length) / 60 else 0 end),
                pipeline_success_rate: (if ($success.series and $fail.series) then
                    ([$success.series[].pointlist[]?[1]] | add) / (([$success.series[].pointlist[]?[1]] | add) + ([$fail.series[].pointlist[]?[1]] | add)) * 100
                    else 0 end),
                test_failure_count: (if $test_fail.series then [$test_fail.series[].pointlist[]?[1]] | add else 0 end)
            }
        }'
}

# Execute query
query_cicd
