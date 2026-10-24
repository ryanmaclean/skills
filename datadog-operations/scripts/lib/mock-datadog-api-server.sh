#!/bin/bash
# Simple HTTP server for mocking Datadog API
# Uses netcat for maximum portability

MOCK_PORT="${MOCK_PORT:-9999}"
SCENARIO="${MOCK_SCENARIO:-normal}"

source "$(dirname "${BASH_SOURCE[0]}")/mock-datadog-api.sh"

echo "[INFO] Mock Datadog API Server starting"
echo "[INFO] Port: $MOCK_PORT"
echo "[INFO] Scenario: $SCENARIO"
echo "[INFO] Press Ctrl+C to stop"

# Simple HTTP request handler
handle_request() {
    local request_line
    local method
    local path
    local headers=""
    local body=""

    # Read request line
    read request_line
    method=$(echo "$request_line" | cut -d' ' -f1)
    path=$(echo "$request_line" | cut -d' ' -f2)

    # Read headers
    while read -r line; do
        line=$(echo "$line" | tr -d '\r\n')
        [ -z "$line" ] && break
        headers="$headers$line\n"
    done

    echo "[$(date)] $method $path" >&2

    # Generate response based on path
    local response_body=""
    local status="200 OK"

    case "$path" in
        */api/v2/spans/analytics/aggregate)
            response_body=$(mock_apm_response "test-service" "0" "999999999")
            ;;
        */api/v2/logs/events/search)
            response_body=$(mock_log_search_response "service:test")
            ;;
        */api/v2/security_monitoring/signals*)
            response_body=$(mock_security_signals_response)
            ;;
        */api/v1/query*)
            response_body=$(mock_metrics_response "system.cpu.user")
            ;;
        */api/v1/slo*)
            response_body=$(mock_slo_response)
            ;;
        */api/v1/monitor*)
            response_body=$(mock_monitors_response)
            ;;
        */api/v2/logs)
            # Log intake endpoint - just return success
            response_body='{"status":"ok"}'
            ;;
        */api/v2/series)
            # Metrics intake endpoint - just return success
            response_body='{"status":"ok"}'
            ;;
        *)
            status="404 Not Found"
            response_body='{"errors":["Endpoint not mocked"]}'
            ;;
    esac

    # Apply scenario effects
    if [ "$SCENARIO" = "rate_limit" ]; then
        status="429 Too Many Requests"
        response_body='{"errors":["Rate limit exceeded"]}'
    fi

    # Send HTTP response
    echo -ne "HTTP/1.1 $status\r\n"
    echo -ne "Content-Type: application/json\r\n"
    echo -ne "Content-Length: ${#response_body}\r\n"
    echo -ne "Connection: close\r\n"
    echo -ne "\r\n"
    echo -n "$response_body"
}

# Start server loop
while true; do
    # Listen for connection using netcat
    if command -v nc >/dev/null 2>&1; then
        nc -l "$MOCK_PORT" | handle_request | nc -l "$MOCK_PORT" >/dev/null 2>&1
    elif command -v netcat >/dev/null 2>&1; then
        netcat -l "$MOCK_PORT" | handle_request | netcat -l "$MOCK_PORT" >/dev/null 2>&1
    else
        echo "[ERROR] netcat (nc) not available - cannot start mock server"
        exit 1
    fi
done
