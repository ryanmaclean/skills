#!/bin/bash
set -e

# Source monitoring library
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/datadog-monitoring.sh" 2>/dev/null || true
init_monitoring "$(basename "$0")"
trap 'finalize_monitoring $?' EXIT INT TERM

# Manage Datadog Monitors
# List, create, mute, unmute, and delete monitors

if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    cat << 'EOF'
Manage Datadog Monitors

Usage:
  manage-monitors.sh list [--service SERVICE] [--tag TAG]
  manage-monitors.sh create --name NAME --query QUERY --message MESSAGE [--type TYPE]
  manage-monitors.sh mute --id MONITOR_ID [--duration HOURS]
  manage-monitors.sh unmute --id MONITOR_ID
  manage-monitors.sh delete --id MONITOR_ID

Commands:
  list    - List monitors with optional filters
  create  - Create a new monitor
  mute    - Mute a monitor
  unmute  - Unmute a monitor
  delete  - Delete a monitor

Monitor Types:
  metric alert - Alert on metric threshold (default)
  apm alert    - Alert on APM metrics
  log alert    - Alert on log patterns
  composite    - Combine multiple monitors

Examples:
  # List all monitors
  manage-monitors.sh list

  # List monitors for service
  manage-monitors.sh list --service payment-api

  # Create error rate monitor
  manage-monitors.sh create \
    --name "High Error Rate" \
    --query "avg(last_5m):sum:trace.express.request.errors{service:my-service}.as_count() > 10" \
    --message "Error rate is high @slack-alerts"

  # Mute monitor for 2 hours
  manage-monitors.sh mute --id 12345 --duration 2

  # Unmute monitor
  manage-monitors.sh unmute --id 12345

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
        start_operation "list_monitors"
        
        SERVICE=""
        TAG=""
        while [[ $# -gt 0 ]]; do
            case $1 in
                --service)
                    SERVICE="$2"
                    shift 2
                    ;;
                --tag)
                    TAG="$2"
                    shift 2
                    ;;
                *)
                    echo "[ERROR] Unknown argument: $1" >&2
                    end_operation "error" "error:unknown_argument"
                    exit 1
                    ;;
            esac
        done

        echo "[INFO] Listing monitors..." >&2

        # Build query parameters
        PARAMS=""
        if [ -n "$SERVICE" ]; then
            PARAMS="?tags=service:${SERVICE}"
        fi
        if [ -n "$TAG" ]; then
            if [ -n "$PARAMS" ]; then
                PARAMS="${PARAMS}&tags=${TAG}"
            else
                PARAMS="?tags=${TAG}"
            fi
        fi

        RESPONSE=$(curl -s -X GET "https://api.${DD_SITE}/api/v1/monitor${PARAMS}" \
            -H "DD-API-KEY: ${DD_API_KEY}" \
            -H "DD-APPLICATION-KEY: ${DD_APP_KEY}")

        # Check for errors
        if echo "$RESPONSE" | jq -e '.errors' > /dev/null 2>&1; then
            echo "[ERROR] API error:" >&2
            echo "$RESPONSE" | jq '.errors' >&2
            end_operation "error" "error:api_error"
            exit 1
        fi

        TOTAL=$(echo "$RESPONSE" | jq 'length')
        echo "[INFO] Found $TOTAL monitors" >&2
        echo "" >&2

        # Count by status
        ALERT=$(echo "$RESPONSE" | jq '[.[] | select(.overall_state == "Alert")] | length')
        WARN=$(echo "$RESPONSE" | jq '[.[] | select(.overall_state == "Warn")] | length')
        OK=$(echo "$RESPONSE" | jq '[.[] | select(.overall_state == "OK" or .overall_state == "No Data")] | length')

        echo "[SUMMARY] Monitor states:" >&2
        echo "  Alert: $ALERT" >&2
        echo "  Warn: $WARN" >&2
        echo "  OK/No Data: $OK" >&2
        echo "" >&2

        send_metric "monitor.list.count" "$TOTAL" "command:list"
        end_operation "ok"

        # Output JSON
        cat <<EOF
{
  "total": $TOTAL,
  "summary": {
    "alert": $ALERT,
    "warn": $WARN,
    "ok": $OK
  },
  "monitors": $(echo "$RESPONSE" | jq '[.[] | {
    id: .id,
    name: .name,
    type: .type,
    query: .query,
    state: .overall_state,
    tags: .tags,
    message: .message
  }]')
}
EOF
        ;;

    create)
        start_operation "create_monitor"
        
        NAME=""
        QUERY=""
        MESSAGE=""
        TYPE="metric alert"

        while [[ $# -gt 0 ]]; do
            case $1 in
                --name)
                    NAME="$2"
                    shift 2
                    ;;
                --query)
                    QUERY="$2"
                    shift 2
                    ;;
                --message)
                    MESSAGE="$2"
                    shift 2
                    ;;
                --type)
                    TYPE="$2"
                    shift 2
                    ;;
                *)
                    echo "[ERROR] Unknown argument: $1" >&2
                    end_operation "error" "error:unknown_argument"
                    exit 1
                    ;;
            esac
        done

        if [ -z "$NAME" ] || [ -z "$QUERY" ] || [ -z "$MESSAGE" ]; then
            echo "[ERROR] --name, --query, and --message are required" >&2
            end_operation "error" "error:missing_parameters"
            exit 1
        fi

        echo "[INFO] Creating monitor: $NAME" >&2

        REQUEST_BODY=$(cat <<EOF
{
  "name": "$NAME",
  "type": "$TYPE",
  "query": "$QUERY",
  "message": "$MESSAGE",
  "tags": [],
  "options": {
    "notify_no_data": true,
    "no_data_timeframe": 20
  }
}
EOF
)

        RESPONSE=$(curl -s -X POST "https://api.${DD_SITE}/api/v1/monitor" \
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

        MONITOR_ID=$(echo "$RESPONSE" | jq -r '.id')
        echo "[OK] Monitor created with ID: $MONITOR_ID" >&2
        echo "" >&2

        send_metric "monitor.create.count" "1" "command:create" "type:$TYPE"
        end_operation "ok"

        echo "$RESPONSE" | jq '{
          id: .id,
          name: .name,
          type: .type,
          query: .query,
          message: .message,
          created: .created,
          status: "created"
        }'
        ;;

    mute)
        start_operation "mute_monitor"
        
        MONITOR_ID=""
        DURATION=""

        while [[ $# -gt 0 ]]; do
            case $1 in
                --id)
                    MONITOR_ID="$2"
                    shift 2
                    ;;
                --duration)
                    DURATION="$2"
                    shift 2
                    ;;
                *)
                    echo "[ERROR] Unknown argument: $1" >&2
                    end_operation "error" "error:unknown_argument"
                    exit 1
                    ;;
            esac
        done

        if [ -z "$MONITOR_ID" ]; then
            echo "[ERROR] --id is required" >&2
            end_operation "error" "error:missing_parameters"
            exit 1
        fi

        echo "[INFO] Muting monitor $MONITOR_ID" >&2

        # Calculate end time if duration specified
        END_PARAM=""
        if [ -n "$DURATION" ]; then
            END_TIME=$(($(date +%s) + (DURATION * 3600)))
            END_PARAM=", \"end\": $END_TIME"
        fi

        REQUEST_BODY="{\"scope\": \"*\"${END_PARAM}}"

        RESPONSE=$(curl -s -X POST "https://api.${DD_SITE}/api/v1/monitor/${MONITOR_ID}/mute" \
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

        echo "[OK] Monitor muted" >&2
        echo "" >&2

        send_metric "monitor.mute.count" "1" "command:mute"
        end_operation "ok"

        echo "$RESPONSE" | jq '{
          id: .id,
          name: .name,
          status: "muted"
        }'
        ;;

    unmute)
        start_operation "unmute_monitor"
        
        MONITOR_ID=""

        while [[ $# -gt 0 ]]; do
            case $1 in
                --id)
                    MONITOR_ID="$2"
                    shift 2
                    ;;
                *)
                    echo "[ERROR] Unknown argument: $1" >&2
                    end_operation "error" "error:unknown_argument"
                    exit 1
                    ;;
            esac
        done

        if [ -z "$MONITOR_ID" ]; then
            echo "[ERROR] --id is required" >&2
            end_operation "error" "error:missing_parameters"
            exit 1
        fi

        echo "[INFO] Unmuting monitor $MONITOR_ID" >&2

        RESPONSE=$(curl -s -X POST "https://api.${DD_SITE}/api/v1/monitor/${MONITOR_ID}/unmute" \
            -H "DD-API-KEY: ${DD_API_KEY}" \
            -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
            -H "Content-Type: application/json" \
            -d '{"scope": "*"}')

        # Check for errors
        if echo "$RESPONSE" | jq -e '.errors' > /dev/null 2>&1; then
            echo "[ERROR] API error:" >&2
            echo "$RESPONSE" | jq '.errors' >&2
            end_operation "error" "error:api_error"
            exit 1
        fi

        echo "[OK] Monitor unmuted" >&2
        echo "" >&2

        send_metric "monitor.unmute.count" "1" "command:unmute"
        end_operation "ok"

        echo "$RESPONSE" | jq '{
          id: .id,
          name: .name,
          status: "unmuted"
        }'
        ;;

    delete)
        start_operation "delete_monitor"
        
        MONITOR_ID=""

        while [[ $# -gt 0 ]]; do
            case $1 in
                --id)
                    MONITOR_ID="$2"
                    shift 2
                    ;;
                *)
                    echo "[ERROR] Unknown argument: $1" >&2
                    end_operation "error" "error:unknown_argument"
                    exit 1
                    ;;
            esac
        done

        if [ -z "$MONITOR_ID" ]; then
            echo "[ERROR] --id is required" >&2
            end_operation "error" "error:missing_parameters"
            exit 1
        fi

        echo "[INFO] Deleting monitor $MONITOR_ID" >&2

        RESPONSE=$(curl -s -X DELETE "https://api.${DD_SITE}/api/v1/monitor/${MONITOR_ID}" \
            -H "DD-API-KEY: ${DD_API_KEY}" \
            -H "DD-APPLICATION-KEY: ${DD_APP_KEY}")

        # Check for errors
        if echo "$RESPONSE" | jq -e '.errors' > /dev/null 2>&1; then
            echo "[ERROR] API error:" >&2
            echo "$RESPONSE" | jq '.errors' >&2
            end_operation "error" "error:api_error"
            exit 1
        fi

        echo "[OK] Monitor deleted" >&2
        echo "" >&2

        send_metric "monitor.delete.count" "1" "command:delete"
        end_operation "ok"

        cat <<EOF
{
  "id": $MONITOR_ID,
  "status": "deleted"
}
EOF
        ;;

    *)
        echo "[ERROR] Unknown command: $COMMAND" >&2
        echo "[INFO] Use --help for usage information" >&2
        exit 1
        ;;
esac
