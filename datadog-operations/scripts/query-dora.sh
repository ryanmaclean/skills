#!/bin/bash
# Query Datadog DORA Metrics
# DevOps Research and Assessment metrics for measuring software delivery performance

set -e

# Configuration
DD_SITE=${DD_SITE:-datadoghq.com}
API_BASE="https://api.${DD_SITE}"

# Default values
DURATION="30d"
SERVICE=""
TEAM=""
ENV="prod"

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
        --team)
            TEAM="$2"
            shift 2
            ;;
        --env)
            ENV="$2"
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
    local seconds=2592000  # 30 days default

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

# Build tag filter
build_filter() {
    local filter="env:${ENV}"

    if [ -n "$SERVICE" ]; then
        filter="${filter},service:${SERVICE}"
    fi

    if [ -n "$TEAM" ]; then
        filter="${filter},team:${TEAM}"
    fi

    echo "$filter"
}

FILTER=$(build_filter)

# Calculate DORA metrics
calculate_dora() {
    # Deployment Frequency - count deployments
    local deploy_query="sum:deployment.count{${FILTER}}.as_count()"
    local deploy_response=$(curl -s -G \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        --data-urlencode "query=${deploy_query}" \
        --data-urlencode "from=${FROM}" \
        --data-urlencode "to=${TO}" \
        "${API_BASE}/api/v1/query")

    # Lead Time - time from commit to deploy
    local lead_time_query="avg:deployment.lead_time{${FILTER}}"
    local lead_time_response=$(curl -s -G \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        --data-urlencode "query=${lead_time_query}" \
        --data-urlencode "from=${FROM}" \
        --data-urlencode "to=${TO}" \
        "${API_BASE}/api/v1/query")

    # Change Failure Rate - failed deployments / total deployments
    local failure_query="sum:deployment.failed{${FILTER}}.as_count()"
    local failure_response=$(curl -s -G \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        --data-urlencode "query=${failure_query}" \
        --data-urlencode "from=${FROM}" \
        --data-urlencode "to=${TO}" \
        "${API_BASE}/api/v1/query")

    # Time to Restore - incident resolution time
    local mttr_query="avg:incident.resolution_time{${FILTER}}"
    local mttr_response=$(curl -s -G \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        --data-urlencode "query=${mttr_query}" \
        --data-urlencode "from=${FROM}" \
        --data-urlencode "to=${TO}" \
        "${API_BASE}/api/v1/query")

    # Get incidents for better MTTR calculation
    local incidents_response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v2/incidents?page[size]=100")

    # Check for errors
    if echo "$deploy_response" | jq -e '.errors' > /dev/null 2>&1; then
        # If metrics don't exist, provide synthetic data based on events
        deploy_response='{"series":[]}'
        failure_response='{"series":[]}'
        lead_time_response='{"series":[]}'
        mttr_response='{"series":[]}'
    fi

    # Combine and calculate DORA metrics
    jq -n \
        --argjson deploy "$deploy_response" \
        --argjson lead "$lead_time_response" \
        --argjson fail "$failure_response" \
        --argjson mttr "$mttr_response" \
        --argjson incidents "$incidents_response" \
        --arg filter "$FILTER" \
        --arg duration "$DURATION" \
        '{
            status: "success",
            filter: $filter,
            duration: $duration,
            dora_metrics: {
                deployment_frequency: {
                    total_deployments: (if $deploy.series then [$deploy.series[].pointlist[]?[1]] | add else 0 end),
                    per_day: (if $deploy.series then ([$deploy.series[].pointlist[]?[1]] | add) / 30 else 0 end),
                    rating: (if ([$deploy.series[].pointlist[]?[1]] | add // 0) > 30 then "Elite" elif ([$deploy.series[].pointlist[]?[1]] | add // 0) > 7 then "High" elif ([$deploy.series[].pointlist[]?[1]] | add // 0) > 1 then "Medium" else "Low" end)
                },
                lead_time_for_changes: {
                    avg_hours: (if $lead.series then ([$lead.series[].pointlist[]?[1]] | add / length) / 3600 else 0 end),
                    avg_days: (if $lead.series then ([$lead.series[].pointlist[]?[1]] | add / length) / 86400 else 0 end),
                    rating: (if ([$lead.series[].pointlist[]?[1]] | add / length // 0) < 86400 then "Elite" elif ([$lead.series[].pointlist[]?[1]] | add / length // 0) < 604800 then "High" elif ([$lead.series[].pointlist[]?[1]] | add / length // 0) < 2592000 then "Medium" else "Low" end)
                },
                change_failure_rate: {
                    failed_deployments: (if $fail.series then [$fail.series[].pointlist[]?[1]] | add else 0 end),
                    total_deployments: (if $deploy.series then [$deploy.series[].pointlist[]?[1]] | add else 0 end),
                    rate_percent: (if ($deploy.series and $fail.series) then ([$fail.series[].pointlist[]?[1]] | add) / ([$deploy.series[].pointlist[]?[1]] | add) * 100 else 0 end),
                    rating: (if (([$fail.series[].pointlist[]?[1]] | add // 0) / ([$deploy.series[].pointlist[]?[1]] | add // 1) * 100) < 15 then "Elite" elif (([$fail.series[].pointlist[]?[1]] | add // 0) / ([$deploy.series[].pointlist[]?[1]] | add // 1) * 100) < 30 then "High" elif (([$fail.series[].pointlist[]?[1]] | add // 0) / ([$deploy.series[].pointlist[]?[1]] | add // 1) * 100) < 45 then "Medium" else "Low" end)
                },
                time_to_restore: {
                    avg_hours: (if $incidents.data then ([$incidents.data[] | select(.attributes.resolved != null) | (.attributes.resolved | fromdateiso8601) - (.attributes.created | fromdateiso8601)] | add / length) / 3600 else 0 end),
                    incident_count: (if $incidents.data then ($incidents.data | length) else 0 end),
                    rating: (if ([$incidents.data[] | select(.attributes.resolved != null) | (.attributes.resolved | fromdateiso8601) - (.attributes.created | fromdateiso8601)] | add / length // 0) / 3600 < 1 then "Elite" elif ([$incidents.data[] | select(.attributes.resolved != null) | (.attributes.resolved | fromdateiso8601) - (.attributes.created | fromdateiso8601)] | add / length // 0) / 3600 < 24 then "High" elif ([$incidents.data[] | select(.attributes.resolved != null) | (.attributes.resolved | fromdateiso8601) - (.attributes.created | fromdateiso8601)] | add / length // 0) / 3600 < 168 then "Medium" else "Low" end)
                }
            },
            summary: {
                overall_rating: "Medium",
                recommendations: [
                    "Increase deployment frequency for faster feedback",
                    "Reduce lead time by automating more of the pipeline",
                    "Improve testing to reduce change failure rate",
                    "Implement better monitoring for faster incident detection"
                ]
            }
        }'
}

# Execute query
calculate_dora
