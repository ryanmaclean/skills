#!/bin/bash
set -e

# Source monitoring library
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/datadog-monitoring.sh" 2>/dev/null || true
init_monitoring "$(basename "$0")"
trap 'finalize_monitoring $?' EXIT INT TERM

# Create Datadog Dashboards
# Generate dashboards from service data or templates

if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    cat << 'EOF'
Create Datadog Dashboards

Usage:
  create-dashboard.sh --service SERVICE --title TITLE [--type TYPE]
  create-dashboard.sh --template TEMPLATE --service SERVICE

Dashboard Types:
  apm       - APM performance dashboard (latency, errors, throughput)
  logs      - Log analysis dashboard
  security  - Security monitoring dashboard
  cost      - Cost analysis dashboard
  llm       - LLM observability dashboard

Examples:
  # Create APM dashboard for service
  create-dashboard.sh --service payment-api --title "Payment API Performance" --type apm

  # Create security dashboard
  create-dashboard.sh --service payment-api --title "Payment API Security" --type security

  # Create cost analysis dashboard
  create-dashboard.sh --title "Infrastructure Costs" --type cost

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

start_operation "create_dashboard"

# Parse arguments
SERVICE=""
TITLE=""
TYPE="apm"

while [[ $# -gt 0 ]]; do
    case $1 in
        --service)
            SERVICE="$2"
            shift 2
            ;;
        --title)
            TITLE="$2"
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

if [ -z "$TITLE" ]; then
    echo "[ERROR] --title is required" >&2
    end_operation "error" "error:missing_parameters"
    exit 1
fi

echo "[INFO] Creating dashboard: $TITLE" >&2
echo "[INFO] Type: $TYPE" >&2
if [ -n "$SERVICE" ]; then
    echo "[INFO] Service: $SERVICE" >&2
fi
echo "" >&2

# Build service filter
SERVICE_FILTER=""
if [ -n "$SERVICE" ]; then
    SERVICE_FILTER="service:${SERVICE}"
fi

# Generate dashboard based on type
case "$TYPE" in
    apm)
        DASHBOARD_JSON=$(cat <<EOF
{
  "title": "$TITLE",
  "description": "APM performance metrics for ${SERVICE:-all services}",
  "widgets": [
    {
      "definition": {
        "type": "timeseries",
        "requests": [
          {
            "q": "avg:trace.express.request.duration{$SERVICE_FILTER} by {resource_name}",
            "display_type": "line"
          }
        ],
        "title": "Request Latency by Endpoint"
      }
    },
    {
      "definition": {
        "type": "query_value",
        "requests": [
          {
            "q": "avg:trace.express.request.duration{$SERVICE_FILTER}",
            "aggregator": "avg"
          }
        ],
        "title": "Average Latency (ms)",
        "precision": 2
      }
    },
    {
      "definition": {
        "type": "timeseries",
        "requests": [
          {
            "q": "sum:trace.express.request.errors{$SERVICE_FILTER}.as_count()",
            "display_type": "bars"
          }
        ],
        "title": "Error Count"
      }
    },
    {
      "definition": {
        "type": "timeseries",
        "requests": [
          {
            "q": "sum:trace.express.request.hits{$SERVICE_FILTER}.as_count()",
            "display_type": "area"
          }
        ],
        "title": "Request Throughput"
      }
    },
    {
      "definition": {
        "type": "toplist",
        "requests": [
          {
            "q": "top(avg:trace.express.request.duration{$SERVICE_FILTER} by {resource_name}, 10, 'mean', 'desc')"
          }
        ],
        "title": "Slowest Endpoints"
      }
    },
    {
      "definition": {
        "type": "query_value",
        "requests": [
          {
            "q": "sum:trace.express.request.errors{$SERVICE_FILTER}.as_count() / sum:trace.express.request.hits{$SERVICE_FILTER}.as_count() * 100",
            "aggregator": "avg"
          }
        ],
        "title": "Error Rate (%)",
        "precision": 2
      }
    }
  ],
  "layout_type": "ordered"
}
EOF
)
        ;;

    logs)
        DASHBOARD_JSON=$(cat <<EOF
{
  "title": "$TITLE",
  "description": "Log analysis for ${SERVICE:-all services}",
  "widgets": [
    {
      "definition": {
        "type": "timeseries",
        "requests": [
          {
            "q": "sum:datadog.estimated_usage.logs.ingested_bytes{$SERVICE_FILTER}.as_count()",
            "display_type": "bars"
          }
        ],
        "title": "Log Volume (Bytes)"
      }
    },
    {
      "definition": {
        "type": "log_stream",
        "query": "status:error ${SERVICE_FILTER}",
        "columns": ["host", "service", "message"],
        "title": "Recent Errors"
      }
    },
    {
      "definition": {
        "type": "toplist",
        "requests": [
          {
            "q": "top(count:logs{$SERVICE_FILTER} by {status}, 10, 'sum', 'desc')"
          }
        ],
        "title": "Log Status Breakdown"
      }
    }
  ],
  "layout_type": "ordered"
}
EOF
)
        ;;

    security)
        DASHBOARD_JSON=$(cat <<EOF
{
  "title": "$TITLE",
  "description": "Security monitoring for ${SERVICE:-all services}",
  "widgets": [
    {
      "definition": {
        "type": "query_value",
        "requests": [
          {
            "q": "sum:datadog.security.appsec.threat{$SERVICE_FILTER}.as_count()",
            "aggregator": "sum"
          }
        ],
        "title": "Security Threats (24h)",
        "precision": 0
      }
    },
    {
      "definition": {
        "type": "timeseries",
        "requests": [
          {
            "q": "sum:datadog.security.appsec.threat{$SERVICE_FILTER} by {attack_type}.as_count()",
            "display_type": "bars"
          }
        ],
        "title": "Threats by Type"
      }
    },
    {
      "definition": {
        "type": "toplist",
        "requests": [
          {
            "q": "top(sum:datadog.security.appsec.threat{$SERVICE_FILTER} by {http.client_ip}.as_count(), 10, 'sum', 'desc')"
          }
        ],
        "title": "Top Attack Sources (IP)"
      }
    },
    {
      "definition": {
        "type": "log_stream",
        "query": "source:security $SERVICE_FILTER severity:high",
        "columns": ["severity", "message", "source"],
        "title": "High Severity Events"
      }
    }
  ],
  "layout_type": "ordered"
}
EOF
)
        ;;

    cost)
        DASHBOARD_JSON=$(cat <<EOF
{
  "title": "$TITLE",
  "description": "Datadog usage and cost analysis",
  "widgets": [
    {
      "definition": {
        "type": "timeseries",
        "requests": [
          {
            "q": "sum:datadog.estimated_usage.apm.ingested_spans{*}.as_count()",
            "display_type": "area"
          }
        ],
        "title": "APM Spans Ingested"
      }
    },
    {
      "definition": {
        "type": "timeseries",
        "requests": [
          {
            "q": "sum:datadog.estimated_usage.logs.ingested_bytes{*}.as_count()",
            "display_type": "area"
          }
        ],
        "title": "Log Volume Ingested"
      }
    },
    {
      "definition": {
        "type": "query_value",
        "requests": [
          {
            "q": "avg:datadog.estimated_usage.hosts{*}",
            "aggregator": "avg"
          }
        ],
        "title": "Average Hosts",
        "precision": 0
      }
    },
    {
      "definition": {
        "type": "toplist",
        "requests": [
          {
            "q": "top(sum:datadog.estimated_usage.apm.ingested_spans{*} by {service}.as_count(), 10, 'sum', 'desc')"
          }
        ],
        "title": "Top Services by APM Volume"
      }
    }
  ],
  "layout_type": "ordered"
}
EOF
)
        ;;

    llm)
        DASHBOARD_JSON=$(cat <<EOF
{
  "title": "$TITLE",
  "description": "LLM observability for ${SERVICE:-all services}",
  "widgets": [
    {
      "definition": {
        "type": "timeseries",
        "requests": [
          {
            "q": "sum:llm.tokens.total{$SERVICE_FILTER}.as_count()",
            "display_type": "area"
          }
        ],
        "title": "Total Tokens Used"
      }
    },
    {
      "definition": {
        "type": "query_value",
        "requests": [
          {
            "q": "avg:llm.request.duration{$SERVICE_FILTER}",
            "aggregator": "avg"
          }
        ],
        "title": "Average Latency (ms)",
        "precision": 2
      }
    },
    {
      "definition": {
        "type": "timeseries",
        "requests": [
          {
            "q": "sum:llm.tokens.prompt{$SERVICE_FILTER} by {llm.model}.as_count()",
            "display_type": "bars"
          }
        ],
        "title": "Token Usage by Model"
      }
    },
    {
      "definition": {
        "type": "toplist",
        "requests": [
          {
            "q": "top(sum:llm.tokens.total{$SERVICE_FILTER} by {resource_name}.as_count(), 10, 'sum', 'desc')"
          }
        ],
        "title": "Most Expensive Operations"
      }
    },
    {
      "definition": {
        "type": "query_value",
        "requests": [
          {
            "q": "sum:llm.request.error{$SERVICE_FILTER}.as_count() / sum:llm.request.total{$SERVICE_FILTER}.as_count() * 100",
            "aggregator": "avg"
          }
        ],
        "title": "Error Rate (%)",
        "precision": 2
      }
    }
  ],
  "layout_type": "ordered"
}
EOF
)
        ;;

    *)
        echo "[ERROR] Unknown dashboard type: $TYPE" >&2
        echo "[INFO] Valid types: apm, logs, security, cost, llm" >&2
        end_operation "error" "error:invalid_type"
        exit 1
        ;;
esac

# Create dashboard
echo "[INFO] Sending request to Datadog API..." >&2

RESPONSE=$(curl -s -X POST "https://api.${DD_SITE}/api/v1/dashboard" \
    -H "DD-API-KEY: ${DD_API_KEY}" \
    -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
    -H "Content-Type: application/json" \
    -d "$DASHBOARD_JSON")

# Check for errors
if echo "$RESPONSE" | jq -e '.errors' > /dev/null 2>&1; then
    echo "[ERROR] API error:" >&2
    echo "$RESPONSE" | jq '.errors' >&2
    end_operation "error" "error:api_error"
    exit 1
fi

DASHBOARD_ID=$(echo "$RESPONSE" | jq -r '.id')
DASHBOARD_URL=$(echo "$RESPONSE" | jq -r '.url // empty')

if [ -z "$DASHBOARD_URL" ]; then
    DASHBOARD_URL="https://app.${DD_SITE}/dashboard/${DASHBOARD_ID}"
fi

echo "" >&2
echo "[OK] Dashboard created successfully" >&2
echo "[INFO] Dashboard ID: $DASHBOARD_ID" >&2
echo "[INFO] URL: $DASHBOARD_URL" >&2
echo "" >&2

send_metric "dashboard.create.count" "1" "type:$TYPE"
end_operation "ok"

# Output JSON
echo "$RESPONSE" | jq '{
  id: .id,
  title: .title,
  url: (.url // "https://app.'${DD_SITE}'/dashboard/" + .id),
  type: "'${TYPE}'",
  status: "created"
}'
