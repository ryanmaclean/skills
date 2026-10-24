#!/bin/bash
set -e

# Query Datadog Service Catalog
# List services, ownership, and metadata

if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    cat << 'EOF'
Query Datadog Service Catalog

Usage:
  query-service-catalog.sh list [--team TEAM]
  query-service-catalog.sh get --service SERVICE

Commands:
  list - List all services in catalog
  get  - Get details for specific service

Examples:
  # List all services
  query-service-catalog.sh list

  # List services for team
  query-service-catalog.sh list --team backend

  # Get service details
  query-service-catalog.sh get --service payment-api

Environment variables:
  DD_API_KEY   - Datadog API key (required)
  DD_APP_KEY   - Datadog application key (required)
  DD_SITE      - Datadog site (default: datadoghq.com)
EOF
    exit 0
fi

# Check required environment variables
if [ -z "$DD_API_KEY" ] || [ -z "$DD_APP_KEY" ]; then
    echo "[ERROR] DD_API_KEY and DD_APP_KEY must be set" >&2
    exit 1
fi

DD_SITE=${DD_SITE:-datadoghq.com}

# Parse command
COMMAND="$1"
shift

case "$COMMAND" in
    list)
        TEAM_FILTER=""

        while [[ $# -gt 0 ]]; do
            case $1 in
                --team)
                    TEAM_FILTER="$2"
                    shift 2
                    ;;
                *)
                    echo "[ERROR] Unknown argument: $1" >&2
                    exit 1
                    ;;
            esac
        done

        echo "[INFO] Querying service catalog..." >&2
        if [ -n "$TEAM_FILTER" ]; then
            echo "[INFO] Team filter: $TEAM_FILTER" >&2
        fi
        echo "" >&2

        RESPONSE=$(curl -s -X GET "https://api.${DD_SITE}/api/v2/services/definitions" \
            -H "DD-API-KEY: ${DD_API_KEY}" \
            -H "DD-APPLICATION-KEY: ${DD_APP_KEY}")

        # Check for errors
        if echo "$RESPONSE" | jq -e '.errors' > /dev/null 2>&1; then
            echo "[ERROR] API error:" >&2
            echo "$RESPONSE" | jq '.errors' >&2
            exit 1
        fi

        # Filter by team if specified
        if [ -n "$TEAM_FILTER" ]; then
            RESPONSE=$(echo "$RESPONSE" | jq --arg team "$TEAM_FILTER" '{
              data: [.data[] | select(.attributes.contacts[]? | select(.type == "team") | .contact | contains($team))]
            }')
        fi

        TOTAL=$(echo "$RESPONSE" | jq '.data | length')
        echo "[INFO] Found $TOTAL services" >&2
        echo "" >&2

        # Count by type
        WEB=$(echo "$RESPONSE" | jq '[.data[] | select(.attributes.kind // "" | contains("web"))] | length')
        API=$(echo "$RESPONSE" | jq '[.data[] | select(.attributes.kind // "" | contains("api"))] | length')
        DB=$(echo "$RESPONSE" | jq '[.data[] | select(.attributes.kind // "" | contains("db"))] | length')

        echo "[SUMMARY] Service types:" >&2
        echo "  Web: $WEB" >&2
        echo "  API: $API" >&2
        echo "  Database: $DB" >&2
        echo "" >&2

        # Output JSON
        cat <<EOF
{
  "total": $TOTAL,
  "summary": {
    "web": $WEB,
    "api": $API,
    "database": $DB
  },
  "services": $(echo "$RESPONSE" | jq '[.data[] | {
    name: .id,
    kind: .attributes.kind,
    description: .attributes.description,
    tier: .attributes.tier,
    lifecycle: .attributes.lifecycle,
    application: .attributes.application,
    team: ([.attributes.contacts[]? | select(.type == "team") | .contact] | first),
    owner: ([.attributes.contacts[]? | select(.type == "email") | .contact] | first),
    languages: .attributes.languages,
    tags: .attributes.tags,
    repos: [.attributes.integrations.pagerduty?.service_url // empty, .attributes.links[]?.url // empty]
  }]')
}
EOF
        ;;

    get)
        SERVICE_NAME=""

        while [[ $# -gt 0 ]]; do
            case $1 in
                --service)
                    SERVICE_NAME="$2"
                    shift 2
                    ;;
                *)
                    echo "[ERROR] Unknown argument: $1" >&2
                    exit 1
                    ;;
            esac
        done

        if [ -z "$SERVICE_NAME" ]; then
            echo "[ERROR] --service is required" >&2
            exit 1
        fi

        echo "[INFO] Fetching service: $SERVICE_NAME" >&2
        echo "" >&2

        RESPONSE=$(curl -s -X GET "https://api.${DD_SITE}/api/v2/services/definitions/${SERVICE_NAME}" \
            -H "DD-API-KEY: ${DD_API_KEY}" \
            -H "DD-APPLICATION-KEY: ${DD_APP_KEY}")

        # Check for errors
        if echo "$RESPONSE" | jq -e '.errors' > /dev/null 2>&1; then
            echo "[ERROR] API error:" >&2
            echo "$RESPONSE" | jq '.errors' >&2
            exit 1
        fi

        # Output detailed service info
        echo "$RESPONSE" | jq '{
          name: .data.id,
          kind: .data.attributes.kind,
          description: .data.attributes.description,
          tier: .data.attributes.tier,
          lifecycle: .data.attributes.lifecycle,
          application: .data.attributes.application,
          team: ([.data.attributes.contacts[]? | select(.type == "team") | .contact] | first),
          owner: ([.data.attributes.contacts[]? | select(.type == "email") | .contact] | first),
          languages: .data.attributes.languages,
          tags: .data.attributes.tags,
          integrations: .data.attributes.integrations,
          links: .data.attributes.links,
          repos: [.data.attributes.links[]? | select(.type == "repo") | .url],
          docs: [.data.attributes.links[]? | select(.type == "doc") | .url],
          schema_version: .data.attributes.schema_version
        }'
        ;;

    *)
        echo "[ERROR] Unknown command: $COMMAND" >&2
        echo "[INFO] Use --help for usage information" >&2
        exit 1
        ;;
esac
