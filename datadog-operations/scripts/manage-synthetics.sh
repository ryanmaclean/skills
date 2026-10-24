#!/bin/bash
set -e

# Source monitoring library
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/datadog-monitoring.sh" 2>/dev/null || true
init_monitoring "$(basename "$0")"
trap 'finalize_monitoring $?' EXIT INT TERM

# Manage Datadog Synthetic Tests
# Create and query uptime checks and API tests

if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    cat << 'EOF'
Manage Datadog Synthetic Tests

Usage:
  manage-synthetics.sh list [--tag TAG]
  manage-synthetics.sh create-api --name NAME --url URL [--method METHOD]
  manage-synthetics.sh create-browser --name NAME --url URL
  manage-synthetics.sh get --id TEST_ID

Commands:
  list           - List synthetic tests
  create-api     - Create API test (uptime check)
  create-browser - Create browser test
  get            - Get test results

Examples:
  # List all synthetic tests
  manage-synthetics.sh list

  # Create API uptime check
  manage-synthetics.sh create-api \
    --name "Payment API Uptime" \
    --url "https://api.example.com/health" \
    --method GET

  # Create browser test
  manage-synthetics.sh create-browser \
    --name "Login Flow" \
    --url "https://app.example.com/login"

  # Get test results
  manage-synthetics.sh get --id abc-123-def

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
        start_operation "list_synthetics"
        
        TAG_FILTER=""

        while [[ $# -gt 0 ]]; do
            case $1 in
                --tag)
                    TAG_FILTER="$2"
                    shift 2
                    ;;
                *)
                    echo "[ERROR] Unknown argument: $1" >&2
                    end_operation "error" "error:unknown_argument"
                    exit 1
                    ;;
            esac
        done

        echo "[INFO] Listing synthetic tests..." >&2
        if [ -n "$TAG_FILTER" ]; then
            echo "[INFO] Tag filter: $TAG_FILTER" >&2
        fi
        echo "" >&2

        RESPONSE=$(curl -s -X GET "https://api.${DD_SITE}/api/v1/synthetics/tests" \
            -H "DD-API-KEY: ${DD_API_KEY}" \
            -H "DD-APPLICATION-KEY: ${DD_APP_KEY}")

        # Check for errors
        if echo "$RESPONSE" | jq -e '.errors' > /dev/null 2>&1; then
            echo "[ERROR] API error:" >&2
            echo "$RESPONSE" | jq '.errors' >&2
            end_operation "error" "error:api_error"
            exit 1
        fi

        # Filter by tag if specified
        if [ -n "$TAG_FILTER" ]; then
            RESPONSE=$(echo "$RESPONSE" | jq --arg tag "$TAG_FILTER" '{
              tests: [.tests[] | select(.tags[]? | contains($tag))]
            }')
        else
            RESPONSE=$(echo "$RESPONSE" | jq '{tests: .tests}')
        fi

        TOTAL=$(echo "$RESPONSE" | jq '.tests | length')
        echo "[INFO] Found $TOTAL synthetic tests" >&2
        echo "" >&2

        # Count by type
        API=$(echo "$RESPONSE" | jq '[.tests[] | select(.type == "api")] | length')
        BROWSER=$(echo "$RESPONSE" | jq '[.tests[] | select(.type == "browser")] | length')

        echo "[SUMMARY] Test types:" >&2
        echo "  API: $API" >&2
        echo "  Browser: $BROWSER" >&2
        echo "" >&2

        # Count by status
        ACTIVE=$(echo "$RESPONSE" | jq '[.tests[] | select(.status == "live")] | length')
        PAUSED=$(echo "$RESPONSE" | jq '[.tests[] | select(.status == "paused")] | length')

        echo "[SUMMARY] Test status:" >&2
        echo "  Active: $ACTIVE" >&2
        echo "  Paused: $PAUSED" >&2
        echo "" >&2

        send_metric "synthetic.list.count" "$TOTAL" "command:list"
        send_metric "synthetic.api.count" "$API" "type:api"
        send_metric "synthetic.browser.count" "$BROWSER" "type:browser"
        end_operation "ok"

        # Output JSON
        cat <<EOF
{
  "total": $TOTAL,
  "summary": {
    "api": $API,
    "browser": $BROWSER,
    "active": $ACTIVE,
    "paused": $PAUSED
  },
  "tests": $(echo "$RESPONSE" | jq '[.tests[] | {
    id: .public_id,
    name: .name,
    type: .type,
    status: .status,
    url: .config.request.url,
    locations: [.locations[]?],
    tags: .tags,
    monitor_id: .monitor_id
  }]')
}
EOF
        ;;

    create-api)
        start_operation "create_api_synthetic"
        
        NAME=""
        URL=""
        METHOD="GET"

        while [[ $# -gt 0 ]]; do
            case $1 in
                --name)
                    NAME="$2"
                    shift 2
                    ;;
                --url)
                    URL="$2"
                    shift 2
                    ;;
                --method)
                    METHOD="$2"
                    shift 2
                    ;;
                *)
                    echo "[ERROR] Unknown argument: $1" >&2
                    end_operation "error" "error:unknown_argument"
                    exit 1
                    ;;
            esac
        done

        if [ -z "$NAME" ] || [ -z "$URL" ]; then
            echo "[ERROR] --name and --url are required" >&2
            end_operation "error" "error:missing_parameters"
            exit 1
        fi

        echo "[INFO] Creating API synthetic test: $NAME" >&2
        echo "[INFO] URL: $URL" >&2
        echo "[INFO] Method: $METHOD" >&2
        echo "" >&2

        REQUEST_BODY=$(cat <<EOF
{
  "name": "$NAME",
  "type": "api",
  "subtype": "http",
  "status": "live",
  "config": {
    "request": {
      "method": "$METHOD",
      "url": "$URL",
      "timeout": 30
    },
    "assertions": [
      {
        "type": "statusCode",
        "operator": "is",
        "target": 200
      },
      {
        "type": "responseTime",
        "operator": "lessThan",
        "target": 2000
      }
    ]
  },
  "locations": ["aws:us-east-1"],
  "options": {
    "tick_every": 300,
    "min_failure_duration": 0,
    "min_location_failed": 1,
    "monitor_options": {
      "notify_audit": false,
      "notify_no_data": false
    }
  },
  "message": "Synthetic test alert: ${NAME}",
  "tags": ["synthetic"]
}
EOF
)

        RESPONSE=$(curl -s -X POST "https://api.${DD_SITE}/api/v1/synthetics/tests/api" \
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

        TEST_ID=$(echo "$RESPONSE" | jq -r '.public_id')
        echo "[OK] Synthetic test created: $TEST_ID" >&2
        echo "" >&2

        send_metric "synthetic.create.count" "1" "type:api" "method:$METHOD"
        end_operation "ok"

        echo "$RESPONSE" | jq '{
          id: .public_id,
          name: .name,
          type: .type,
          url: .config.request.url,
          status: "created"
        }'
        ;;

    create-browser)
        start_operation "create_browser_synthetic"
        
        NAME=""
        URL=""

        while [[ $# -gt 0 ]]; do
            case $1 in
                --name)
                    NAME="$2"
                    shift 2
                    ;;
                --url)
                    URL="$2"
                    shift 2
                    ;;
                *)
                    echo "[ERROR] Unknown argument: $1" >&2
                    end_operation "error" "error:unknown_argument"
                    exit 1
                    ;;
            esac
        done

        if [ -z "$NAME" ] || [ -z "$URL" ]; then
            echo "[ERROR] --name and --url are required" >&2
            end_operation "error" "error:missing_parameters"
            exit 1
        fi

        echo "[INFO] Creating browser synthetic test: $NAME" >&2
        echo "[INFO] URL: $URL" >&2
        echo "" >&2

        REQUEST_BODY=$(cat <<EOF
{
  "name": "$NAME",
  "type": "browser",
  "status": "live",
  "config": {
    "request": {
      "url": "$URL"
    },
    "assertions": []
  },
  "locations": ["aws:us-east-1"],
  "options": {
    "tick_every": 900,
    "min_failure_duration": 0,
    "min_location_failed": 1,
    "device_ids": ["laptop_large"],
    "monitor_options": {
      "notify_audit": false,
      "notify_no_data": false
    }
  },
  "message": "Browser test alert: ${NAME}",
  "tags": ["synthetic", "browser"],
  "steps": [
    {
      "name": "Navigate to URL",
      "type": "goToUrl",
      "params": {
        "url": "$URL"
      },
      "allowFailure": false,
      "timeout": 60
    }
  ]
}
EOF
)

        RESPONSE=$(curl -s -X POST "https://api.${DD_SITE}/api/v1/synthetics/tests/browser" \
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

        TEST_ID=$(echo "$RESPONSE" | jq -r '.public_id')
        echo "[OK] Browser test created: $TEST_ID" >&2
        echo "" >&2

        send_metric "synthetic.create.count" "1" "type:browser"
        end_operation "ok"

        echo "$RESPONSE" | jq '{
          id: .public_id,
          name: .name,
          type: .type,
          url: .config.request.url,
          status: "created"
        }'
        ;;

    get)
        start_operation "get_synthetic"
        
        TEST_ID=""

        while [[ $# -gt 0 ]]; do
            case $1 in
                --id)
                    TEST_ID="$2"
                    shift 2
                    ;;
                *)
                    echo "[ERROR] Unknown argument: $1" >&2
                    end_operation "error" "error:unknown_argument"
                    exit 1
                    ;;
            esac
        done

        if [ -z "$TEST_ID" ]; then
            echo "[ERROR] --id is required" >&2
            end_operation "error" "error:missing_parameters"
            exit 1
        fi

        echo "[INFO] Fetching test results: $TEST_ID" >&2
        echo "" >&2

        RESPONSE=$(curl -s -X GET "https://api.${DD_SITE}/api/v1/synthetics/tests/${TEST_ID}" \
            -H "DD-API-KEY: ${DD_API_KEY}" \
            -H "DD-APPLICATION-KEY: ${DD_APP_KEY}")

        # Check for errors
        if echo "$RESPONSE" | jq -e '.errors' > /dev/null 2>&1; then
            echo "[ERROR] API error:" >&2
            echo "$RESPONSE" | jq '.errors' >&2
            end_operation "error" "error:api_error"
            exit 1
        fi

        send_metric "synthetic.get.count" "1" "command:get"
        end_operation "ok"

        # Output test details
        echo "$RESPONSE" | jq '{
          id: .public_id,
          name: .name,
          type: .type,
          status: .status,
          url: .config.request.url,
          locations: .locations,
          monitor_id: .monitor_id,
          tags: .tags,
          created_at: .created_at,
          modified_at: .modified_at
        }'
        ;;

    *)
        echo "[ERROR] Unknown command: $COMMAND" >&2
        echo "[INFO] Use --help for usage information" >&2
        exit 1
        ;;
esac
