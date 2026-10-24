#!/bin/bash
set -e

# Source monitoring library
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/datadog-monitoring.sh" 2>/dev/null || true
init_monitoring "$(basename "$0")"
trap 'finalize_monitoring $?' EXIT INT TERM

# Manage Datadog Incidents
# Create, update, and query incidents

if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    cat << 'EOF'
Manage Datadog Incidents

Usage:
  manage-incidents.sh list [--status STATUS]
  manage-incidents.sh create --title TITLE --service SERVICE [--severity SEVERITY]
  manage-incidents.sh update --id INCIDENT_ID --status STATUS
  manage-incidents.sh get --id INCIDENT_ID

Commands:
  list   - List incidents with optional status filter
  create - Create a new incident
  update - Update incident status
  get    - Get incident details

Status values:
  active, stable, resolved

Severity values:
  SEV-1, SEV-2, SEV-3, SEV-4, SEV-5, UNKNOWN

Examples:
  # List active incidents
  manage-incidents.sh list --status active

  # Create critical incident
  manage-incidents.sh create \
    --title "Payment API Down" \
    --service payment-api \
    --severity SEV-1

  # Update incident status
  manage-incidents.sh update --id abc123 --status resolved

  # Get incident details
  manage-incidents.sh get --id abc123

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
        start_operation "list_incidents"
        
        STATUS_FILTER=""

        while [[ $# -gt 0 ]]; do
            case $1 in
                --status)
                    STATUS_FILTER="$2"
                    shift 2
                    ;;
                *)
                    echo "[ERROR] Unknown argument: $1" >&2
                    end_operation "error" "error:unknown_argument"
                    exit 1
                    ;;
            esac
        done

        echo "[INFO] Listing incidents..." >&2
        if [ -n "$STATUS_FILTER" ]; then
            echo "[INFO] Status filter: $STATUS_FILTER" >&2
        fi
        echo "" >&2

        # Build query parameters
        PARAMS=""
        if [ -n "$STATUS_FILTER" ]; then
            PARAMS="?filter[state]=${STATUS_FILTER}"
        fi

        RESPONSE=$(curl -s -X GET "https://api.${DD_SITE}/api/v2/incidents${PARAMS}" \
            -H "DD-API-KEY: ${DD_API_KEY}" \
            -H "DD-APPLICATION-KEY: ${DD_APP_KEY}")

        # Check for errors
        if echo "$RESPONSE" | jq -e '.errors' > /dev/null 2>&1; then
            echo "[ERROR] API error:" >&2
            echo "$RESPONSE" | jq '.errors' >&2
            end_operation "error" "error:api_error"
            exit 1
        fi

        TOTAL=$(echo "$RESPONSE" | jq '.data | length')
        echo "[INFO] Found $TOTAL incidents" >&2
        echo "" >&2

        # Count by status
        ACTIVE=$(echo "$RESPONSE" | jq '[.data[] | select(.attributes.state == "active")] | length')
        STABLE=$(echo "$RESPONSE" | jq '[.data[] | select(.attributes.state == "stable")] | length')
        RESOLVED=$(echo "$RESPONSE" | jq '[.data[] | select(.attributes.state == "resolved")] | length')

        echo "[SUMMARY] Incident status:" >&2
        echo "  Active: $ACTIVE" >&2
        echo "  Stable: $STABLE" >&2
        echo "  Resolved: $RESOLVED" >&2
        echo "" >&2

        send_metric "incident.list.count" "$TOTAL" "command:list"
        send_metric "incident.active.count" "$ACTIVE" "command:list"
        end_operation "ok"

        # Output JSON
        cat <<EOF
{
  "total": $TOTAL,
  "summary": {
    "active": $ACTIVE,
    "stable": $STABLE,
    "resolved": $RESOLVED
  },
  "incidents": $(echo "$RESPONSE" | jq '[.data[] | {
    id: .id,
    title: .attributes.title,
    state: .attributes.state,
    severity: .attributes.severity,
    created: .attributes.created,
    modified: .attributes.modified,
    customer_impact: .attributes.customer_impact_scope
  }]')
}
EOF
        ;;

    create)
        start_operation "create_incident"
        
        TITLE=""
        SERVICE=""
        SEVERITY="UNKNOWN"

        while [[ $# -gt 0 ]]; do
            case $1 in
                --title)
                    TITLE="$2"
                    shift 2
                    ;;
                --service)
                    SERVICE="$2"
                    shift 2
                    ;;
                --severity)
                    SEVERITY="$2"
                    shift 2
                    ;;
                *)
                    echo "[ERROR] Unknown argument: $1" >&2
                    end_operation "error" "error:unknown_argument"
                    exit 1
                    ;;
            esac
        done

        if [ -z "$TITLE" ] || [ -z "$SERVICE" ]; then
            echo "[ERROR] --title and --service are required" >&2
            end_operation "error" "error:missing_parameters"
            exit 1
        fi

        echo "[INFO] Creating incident: $TITLE" >&2
        echo "[INFO] Service: $SERVICE" >&2
        echo "[INFO] Severity: $SEVERITY" >&2
        echo "" >&2

        REQUEST_BODY=$(cat <<EOF
{
  "data": {
    "type": "incidents",
    "attributes": {
      "title": "$TITLE",
      "severity": "$SEVERITY",
      "customer_impacted": false,
      "fields": {
        "service": {
          "type": "textbox",
          "value": "$SERVICE"
        }
      }
    }
  }
}
EOF
)

        RESPONSE=$(curl -s -X POST "https://api.${DD_SITE}/api/v2/incidents" \
            -H "DD-API-KEY: ${DD_API_KEY}" \
            -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
            -H "Content-Type: application/json" \
            -d "$REQUEST_BODY")

        # Check for errors
        if echo "$RESPONSE" | jq -e '.errors' > /dev/null 2>&1; then
            echo "[ERROR] API error:" >&2
            echo "$RESPONSE" | jq '.errors' >&2
            end_operation "error" "error:api_error"
            exit 1
        fi

        INCIDENT_ID=$(echo "$RESPONSE" | jq -r '.data.id')
        echo "[OK] Incident created: $INCIDENT_ID" >&2
        echo "" >&2

        send_metric "incident.create.count" "1" "severity:$SEVERITY"
        end_operation "ok"

        # Output JSON
        echo "$RESPONSE" | jq '{
          id: .data.id,
          title: .data.attributes.title,
          severity: .data.attributes.severity,
          state: .data.attributes.state,
          created: .data.attributes.created,
          status: "created"
        }'
        ;;

    update)
        start_operation "update_incident"
        
        INCIDENT_ID=""
        NEW_STATUS=""

        while [[ $# -gt 0 ]]; do
            case $1 in
                --id)
                    INCIDENT_ID="$2"
                    shift 2
                    ;;
                --status)
                    NEW_STATUS="$2"
                    shift 2
                    ;;
                *)
                    echo "[ERROR] Unknown argument: $1" >&2
                    end_operation "error" "error:unknown_argument"
                    exit 1
                    ;;
            esac
        done

        if [ -z "$INCIDENT_ID" ] || [ -z "$NEW_STATUS" ]; then
            echo "[ERROR] --id and --status are required" >&2
            end_operation "error" "error:missing_parameters"
            exit 1
        fi

        echo "[INFO] Updating incident $INCIDENT_ID to status: $NEW_STATUS" >&2
        echo "" >&2

        REQUEST_BODY=$(cat <<EOF
{
  "data": {
    "type": "incidents",
    "id": "$INCIDENT_ID",
    "attributes": {
      "state": "$NEW_STATUS"
    }
  }
}
EOF
)

        RESPONSE=$(curl -s -X PATCH "https://api.${DD_SITE}/api/v2/incidents/${INCIDENT_ID}" \
            -H "DD-API-KEY: ${DD_API_KEY}" \
            -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
            -H "Content-Type: application/json" \
            -d "$REQUEST_BODY")

        # Check for errors
        if echo "$RESPONSE" | jq -e '.errors' > /dev/null 2>&1; then
            echo "[ERROR] API error:" >&2
            echo "$RESPONSE" | jq '.errors' >&2
            end_operation "error" "error:api_error"
            exit 1
        fi

        echo "[OK] Incident updated" >&2
        echo "" >&2

        send_metric "incident.update.count" "1" "new_status:$NEW_STATUS"
        end_operation "ok"

        # Output JSON
        echo "$RESPONSE" | jq '{
          id: .data.id,
          title: .data.attributes.title,
          state: .data.attributes.state,
          modified: .data.attributes.modified,
          status: "updated"
        }'
        ;;

    get)
        start_operation "get_incident"
        
        INCIDENT_ID=""

        while [[ $# -gt 0 ]]; do
            case $1 in
                --id)
                    INCIDENT_ID="$2"
                    shift 2
                    ;;
                *)
                    echo "[ERROR] Unknown argument: $1" >&2
                    end_operation "error" "error:unknown_argument"
                    exit 1
                    ;;
            esac
        done

        if [ -z "$INCIDENT_ID" ]; then
            echo "[ERROR] --id is required" >&2
            end_operation "error" "error:missing_parameters"
            exit 1
        fi

        echo "[INFO] Fetching incident: $INCIDENT_ID" >&2
        echo "" >&2

        RESPONSE=$(curl -s -X GET "https://api.${DD_SITE}/api/v2/incidents/${INCIDENT_ID}" \
            -H "DD-API-KEY: ${DD_API_KEY}" \
            -H "DD-APPLICATION-KEY: ${DD_APP_KEY}")

        # Check for errors
        if echo "$RESPONSE" | jq -e '.errors' > /dev/null 2>&1; then
            echo "[ERROR] API error:" >&2
            echo "$RESPONSE" | jq '.errors' >&2
            end_operation "error" "error:api_error"
            exit 1
        fi

        send_metric "incident.get.count" "1" "command:get"
        end_operation "ok"

        # Output JSON with key details
        echo "$RESPONSE" | jq '{
          id: .data.id,
          title: .data.attributes.title,
          state: .data.attributes.state,
          severity: .data.attributes.severity,
          customer_impacted: .data.attributes.customer_impacted,
          created: .data.attributes.created,
          modified: .data.attributes.modified,
          resolved: .data.attributes.resolved,
          timeline: [.included[]? | select(.type == "incident_timeline") | {
            type: .attributes.content.content_type,
            message: .attributes.content.message,
            timestamp: .attributes.timestamp
          }]
        }'
        ;;

    *)
        echo "[ERROR] Unknown command: $COMMAND" >&2
        echo "[INFO] Use --help for usage information" >&2
        exit 1
        ;;
esac
