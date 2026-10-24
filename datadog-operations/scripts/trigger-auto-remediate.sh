#!/bin/bash
# Trigger Datadog Auto-Remediation
# Automated remediation workflows based on detected conditions

set -e

# Configuration
DD_SITE=${DD_SITE:-datadoghq.com}
API_BASE="https://api.${DD_SITE}"

# Default values
ACTION="list"
WORKFLOW_ID=""
TRIGGER_CONDITION=""
SERVICE=""
SEVERITY="high"

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --action)
            ACTION="$2"
            shift 2
            ;;
        --workflow-id)
            WORKFLOW_ID="$2"
            shift 2
            ;;
        --trigger)
            TRIGGER_CONDITION="$2"
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

# List remediation workflows
list_workflows() {
    # Get workflows that are tagged for auto-remediation
    response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v1/workflows")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0]}'
        exit 1
    fi

    # Filter for auto-remediation workflows
    echo "$response" | jq '{
        status: "success",
        total_workflows: ([.data[] | select(.tags[]? | contains("auto-remediate"))] | length),
        remediation_workflows: [.data[] | select(.tags[]? | contains("auto-remediate")) | {
            id: .id,
            name: .name,
            description: .description,
            tags: .tags,
            trigger_type: .trigger.type,
            enabled: .enabled,
            last_execution: .last_execution,
            success_rate: .metrics.success_rate
        }]
    }'
}

# Trigger remediation workflow
trigger_workflow() {
    if [ -z "$WORKFLOW_ID" ]; then
        echo '{"status":"error","message":"--workflow-id is required for trigger action"}' >&2
        exit 1
    fi

    # Build trigger payload
    local payload=$(jq -n \
        --arg trigger "${TRIGGER_CONDITION:-manual}" \
        --arg service "${SERVICE:-}" \
        --arg severity "$SEVERITY" \
        '{
            trigger: {
                type: $trigger,
                metadata: {
                    service: (if $service != "" then $service else null end),
                    severity: $severity,
                    timestamp: (now | tostring),
                    source: "auto-remediate-cli"
                }
            }
        }')

    response=$(curl -s -X POST \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "${API_BASE}/api/v1/workflows/${WORKFLOW_ID}/execute")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0]}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        execution: {
            workflow_id: .workflow_id,
            execution_id: .id,
            status: .status,
            started_at: .started_at,
            trigger: .trigger
        }
    }'
}

# Get workflow execution status
get_execution() {
    if [ -z "$WORKFLOW_ID" ]; then
        echo '{"status":"error","message":"--workflow-id is required for status action"}' >&2
        exit 1
    fi

    response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v1/workflows/${WORKFLOW_ID}/executions")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0]}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        workflow_id: .workflow_id,
        total_executions: (.executions | length),
        recent_executions: [.executions[0:10][] | {
            execution_id: .id,
            status: .status,
            started_at: .started_at,
            completed_at: .completed_at,
            duration_ms: .duration_ms,
            result: .result
        }],
        success_rate: (if (.executions | length) > 0 then
            ([.executions[] | select(.status == "success")] | length) / (.executions | length) * 100
            else 0 end)
    }'
}

# List common remediation patterns
list_patterns() {
    cat << 'PATTERNS'
{
  "status": "success",
  "remediation_patterns": [
    {
      "name": "High CPU Auto-Scale",
      "trigger": "cpu > 80% for 5 minutes",
      "action": "Scale up pod replicas by 50%",
      "severity": "high",
      "tags": ["kubernetes", "auto-scale", "cpu"]
    },
    {
      "name": "Memory Leak Recovery",
      "trigger": "memory > 90% for 10 minutes",
      "action": "Rolling restart of affected pods",
      "severity": "critical",
      "tags": ["kubernetes", "memory", "restart"]
    },
    {
      "name": "Service Health Recovery",
      "trigger": "error_rate > 5% for 3 minutes",
      "action": "Route traffic to backup region",
      "severity": "high",
      "tags": ["failover", "traffic-management"]
    },
    {
      "name": "Disk Space Cleanup",
      "trigger": "disk > 85% used",
      "action": "Clean old logs and temp files",
      "severity": "medium",
      "tags": ["disk", "cleanup", "maintenance"]
    },
    {
      "name": "Database Connection Pool",
      "trigger": "db_connections > 90% of max",
      "action": "Increase connection pool size",
      "severity": "high",
      "tags": ["database", "connection-pool"]
    }
  ]
}
PATTERNS
}

# Execute action
case "$ACTION" in
    list)
        list_workflows
        ;;
    trigger)
        trigger_workflow
        ;;
    status)
        get_execution
        ;;
    patterns)
        list_patterns
        ;;
    *)
        echo "{\"status\":\"error\",\"message\":\"Unknown action: $ACTION (use: list, trigger, status, patterns)\"}" >&2
        exit 1
        ;;
esac
