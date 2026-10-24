#!/bin/bash
# Manage Datadog Integrations
# View and manage cloud provider and service integrations

set -e

# Configuration
DD_SITE=${DD_SITE:-datadoghq.com}
API_BASE="https://api.${DD_SITE}"

# Default values
ACTION="list"
INTEGRATION=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --action)
            ACTION="$2"
            shift 2
            ;;
        --integration)
            INTEGRATION="$2"
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

# List all integrations
list_integrations() {
    # Get AWS integrations
    local aws_response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v1/integration/aws" 2>/dev/null || echo '{"accounts":[]}')

    # Get Azure integrations
    local azure_response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v1/integration/azure" 2>/dev/null || echo '[]')

    # Get GCP integrations
    local gcp_response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v1/integration/gcp" 2>/dev/null || echo '[]')

    # Get Slack integrations
    local slack_response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v1/integration/slack" 2>/dev/null || echo '[]')

    # Get PagerDuty integrations
    local pd_response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v1/integration/pagerduty" 2>/dev/null || echo '{"services":[]}')

    # Combine all integrations
    jq -n \
        --argjson aws "$aws_response" \
        --argjson azure "$azure_response" \
        --argjson gcp "$gcp_response" \
        --argjson slack "$slack_response" \
        --argjson pd "$pd_response" \
        '{
            status: "success",
            integrations: {
                aws: {
                    count: ($aws.accounts | length),
                    accounts: $aws.accounts
                },
                azure: {
                    count: ($azure | length),
                    accounts: $azure
                },
                gcp: {
                    count: ($gcp | length),
                    projects: $gcp
                },
                slack: {
                    count: ($slack | length),
                    channels: $slack
                },
                pagerduty: {
                    count: ($pd.services | length),
                    services: $pd.services
                }
            },
            summary: {
                total_integrations: (($aws.accounts | length) + ($azure | length) + ($gcp | length) + ($slack | length) + ($pd.services | length)),
                by_provider: {
                    aws: ($aws.accounts | length),
                    azure: ($azure | length),
                    gcp: ($gcp | length),
                    slack: ($slack | length),
                    pagerduty: ($pd.services | length)
                }
            }
        }'
}

# Get specific integration
get_integration() {
    if [ -z "$INTEGRATION" ]; then
        echo '{"status":"error","message":"--integration is required (aws, azure, gcp, slack, pagerduty)"}' >&2
        exit 1
    fi

    local response=""
    case "$INTEGRATION" in
        aws)
            response=$(curl -s \
                -H "DD-API-KEY: ${DD_API_KEY}" \
                -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
                "${API_BASE}/api/v1/integration/aws")
            ;;
        azure)
            response=$(curl -s \
                -H "DD-API-KEY: ${DD_API_KEY}" \
                -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
                "${API_BASE}/api/v1/integration/azure")
            ;;
        gcp)
            response=$(curl -s \
                -H "DD-API-KEY: ${DD_API_KEY}" \
                -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
                "${API_BASE}/api/v1/integration/gcp")
            ;;
        slack)
            response=$(curl -s \
                -H "DD-API-KEY: ${DD_API_KEY}" \
                -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
                "${API_BASE}/api/v1/integration/slack")
            ;;
        pagerduty)
            response=$(curl -s \
                -H "DD-API-KEY: ${DD_API_KEY}" \
                -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
                "${API_BASE}/api/v1/integration/pagerduty")
            ;;
        *)
            echo "{\"status\":\"error\",\"message\":\"Unknown integration: $INTEGRATION\"}" >&2
            exit 1
            ;;
    esac

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0]}'
        exit 1
    fi

    # Format response
    echo "$response" | jq --arg integration "$INTEGRATION" '{
        status: "success",
        integration: $integration,
        data: .
    }'
}

# Execute action
case "$ACTION" in
    list)
        list_integrations
        ;;
    get)
        get_integration
        ;;
    *)
        echo "{\"status\":\"error\",\"message\":\"Unknown action: $ACTION (use: list, get)\"}" >&2
        exit 1
        ;;
esac
