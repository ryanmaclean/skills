#!/bin/bash
# Manage Datadog Webhooks
# Create, update, and manage webhook integrations

set -e

# Configuration
DD_SITE=${DD_SITE:-datadoghq.com}
API_BASE="https://api.${DD_SITE}"

# Default values
ACTION="list"
WEBHOOK_NAME=""
WEBHOOK_URL=""
CUSTOM_HEADERS=""
PAYLOAD=""
ENCODE_AS="json"

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --action)
            ACTION="$2"
            shift 2
            ;;
        --name)
            WEBHOOK_NAME="$2"
            shift 2
            ;;
        --url)
            WEBHOOK_URL="$2"
            shift 2
            ;;
        --headers)
            CUSTOM_HEADERS="$2"
            shift 2
            ;;
        --payload)
            PAYLOAD="$2"
            shift 2
            ;;
        --encode-as)
            ENCODE_AS="$2"
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

# List webhooks
list_webhooks() {
    response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v1/integration/webhooks/configuration/webhooks")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0]}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        total_webhooks: (. | length),
        webhooks: [.[] | {
            name: .name,
            url: .url,
            encode_as: .encode_as,
            custom_headers: (.custom_headers // "none"),
            created_at: .created_at,
            created_by: .created_by
        }]
    }'
}

# Get webhook details
get_webhook() {
    if [ -z "$WEBHOOK_NAME" ]; then
        echo '{"status":"error","message":"--name is required for get action"}' >&2
        exit 1
    fi

    response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v1/integration/webhooks/configuration/webhooks/${WEBHOOK_NAME}")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0]}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        webhook: {
            name: .name,
            url: .url,
            encode_as: .encode_as,
            custom_headers: .custom_headers,
            payload: .payload,
            created_at: .created_at,
            created_by: .created_by,
            modified_at: .modified_at
        }
    }'
}

# Create webhook
create_webhook() {
    if [ -z "$WEBHOOK_NAME" ] || [ -z "$WEBHOOK_URL" ]; then
        echo '{"status":"error","message":"--name and --url are required for create action"}' >&2
        exit 1
    fi

    # Build payload
    local webhook_payload=$(jq -n \
        --arg name "$WEBHOOK_NAME" \
        --arg url "$WEBHOOK_URL" \
        --arg encode "$ENCODE_AS" \
        --arg headers "${CUSTOM_HEADERS:-}" \
        --arg payload "${PAYLOAD:-}" \
        '{
            name: $name,
            url: $url,
            encode_as: $encode,
            custom_headers: (if $headers != "" then $headers else null end),
            payload: (if $payload != "" then $payload else null end)
        }')

    response=$(curl -s -X POST \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        -H "Content-Type: application/json" \
        -d "$webhook_payload" \
        "${API_BASE}/api/v1/integration/webhooks/configuration/webhooks")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0]}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        webhook: {
            name: .name,
            url: .url,
            encode_as: .encode_as,
            created_at: .created_at
        }
    }'
}

# Update webhook
update_webhook() {
    if [ -z "$WEBHOOK_NAME" ]; then
        echo '{"status":"error","message":"--name is required for update action"}' >&2
        exit 1
    fi

    # Build update payload
    local update_payload="{}"
    if [ -n "$WEBHOOK_URL" ]; then
        update_payload=$(echo "$update_payload" | jq --arg url "$WEBHOOK_URL" '. + {url: $url}')
    fi
    if [ -n "$CUSTOM_HEADERS" ]; then
        update_payload=$(echo "$update_payload" | jq --arg headers "$CUSTOM_HEADERS" '. + {custom_headers: $headers}')
    fi
    if [ -n "$PAYLOAD" ]; then
        update_payload=$(echo "$update_payload" | jq --arg payload "$PAYLOAD" '. + {payload: $payload}')
    fi
    if [ -n "$ENCODE_AS" ] && [ "$ENCODE_AS" != "json" ]; then
        update_payload=$(echo "$update_payload" | jq --arg encode "$ENCODE_AS" '. + {encode_as: $encode}')
    fi

    response=$(curl -s -X PUT \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        -H "Content-Type: application/json" \
        -d "$update_payload" \
        "${API_BASE}/api/v1/integration/webhooks/configuration/webhooks/${WEBHOOK_NAME}")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0]}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        webhook: {
            name: .name,
            url: .url,
            encode_as: .encode_as,
            modified_at: .modified_at
        }
    }'
}

# Delete webhook
delete_webhook() {
    if [ -z "$WEBHOOK_NAME" ]; then
        echo '{"status":"error","message":"--name is required for delete action"}' >&2
        exit 1
    fi

    response=$(curl -s -X DELETE \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v1/integration/webhooks/configuration/webhooks/${WEBHOOK_NAME}")

    # 204 No Content is success
    if [ -z "$response" ]; then
        echo "{\"status\":\"success\",\"message\":\"Webhook ${WEBHOOK_NAME} deleted successfully\"}"
    else
        if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
            echo "$response" | jq '{status: "error", message: .errors[0]}'
            exit 1
        fi
        echo "$response"
    fi
}

# Test webhook
test_webhook() {
    if [ -z "$WEBHOOK_NAME" ]; then
        echo '{"status":"error","message":"--name is required for test action"}' >&2
        exit 1
    fi

    response=$(curl -s -X POST \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v1/integration/webhooks/configuration/webhooks/${WEBHOOK_NAME}/test")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0]}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        test_result: {
            webhook_name: "'$WEBHOOK_NAME'",
            test_successful: (if .status == "success" then true else false end),
            response_code: .response_code,
            message: .message
        }
    }'
}

# Execute action
case "$ACTION" in
    list)
        list_webhooks
        ;;
    get)
        get_webhook
        ;;
    create)
        create_webhook
        ;;
    update)
        update_webhook
        ;;
    delete)
        delete_webhook
        ;;
    test)
        test_webhook
        ;;
    *)
        echo "{\"status\":\"error\",\"message\":\"Unknown action: $ACTION (use: list, get, create, update, delete, test)\"}" >&2
        exit 1
        ;;
esac
