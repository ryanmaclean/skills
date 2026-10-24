#!/bin/bash
set -e

# Source monitoring library
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/datadog-monitoring.sh" 2>/dev/null || true
init_monitoring "$(basename "$0")"
trap 'finalize_monitoring $?' EXIT INT TERM

# Trigger Datadog Workflows
# Execute automation workflows for incident response and remediation

if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    cat << 'EOF'
Trigger Datadog Workflows

Usage:
  trigger-workflow.sh list
  trigger-workflow.sh run --id WORKFLOW_ID [--input JSON]

Commands:
  list - List available workflows
  run  - Execute a workflow

Examples:
  # List all workflows
  trigger-workflow.sh list

  # Trigger workflow without input
  trigger-workflow.sh run --id abc123

  # Trigger workflow with input data
  trigger-workflow.sh run --id abc123 --input '{"service": "payment-api", "severity": "high"}'

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
        start_operation "list_workflows"
        
        echo "[INFO] Listing workflows..." >&2

        RESPONSE=$(curl -s -X GET "https://api.${DD_SITE}/api/v2/workflows" \
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
        echo "[INFO] Found $TOTAL workflows" >&2
        echo "" >&2

        send_metric "workflow.list.count" "$TOTAL" "command:list"
        end_operation "ok"

        # Output JSON
        cat <<EOF
{
  "total": $TOTAL,
  "workflows": $(echo "$RESPONSE" | jq '[.data[] | {
    id: .id,
    name: .attributes.name,
    description: .attributes.description,
    created: .attributes.created_at,
    modified: .attributes.modified_at
  }]')
}
EOF
        ;;

    run)
        start_operation "run_workflow"
        
        WORKFLOW_ID=""
        INPUT_JSON="{}"

        while [[ $# -gt 0 ]]; do
            case $1 in
                --id)
                    WORKFLOW_ID="$2"
                    shift 2
                    ;;
                --input)
                    INPUT_JSON="$2"
                    shift 2
                    ;;
                *)
                    echo "[ERROR] Unknown argument: $1" >&2
                    end_operation "error" "error:unknown_argument"
                    exit 1
                    ;;
            esac
        done

        if [ -z "$WORKFLOW_ID" ]; then
            echo "[ERROR] --id is required" >&2
            end_operation "error" "error:missing_parameters"
            exit 1
        fi

        echo "[INFO] Triggering workflow: $WORKFLOW_ID" >&2
        echo "[INFO] Input data: $INPUT_JSON" >&2
        echo "" >&2

        REQUEST_BODY=$(cat <<EOF
{
  "data": {
    "attributes": {
      "input": $INPUT_JSON
    }
  }
}
EOF
)

        RESPONSE=$(curl -s -X POST "https://api.${DD_SITE}/api/v2/workflows/${WORKFLOW_ID}/instances" \
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

        INSTANCE_ID=$(echo "$RESPONSE" | jq -r '.data.id')
        echo "[OK] Workflow instance created: $INSTANCE_ID" >&2
        echo "" >&2

        send_metric "workflow.run.count" "1" "workflow_id:$WORKFLOW_ID"
        end_operation "ok"

        # Output JSON
        echo "$RESPONSE" | jq '{
          instance_id: .data.id,
          workflow_id: "'${WORKFLOW_ID}'",
          status: "triggered",
          created_at: .data.attributes.created_at
        }'
        ;;

    *)
        echo "[ERROR] Unknown command: $COMMAND" >&2
        echo "[INFO] Use --help for usage information" >&2
        exit 1
        ;;
esac
