#!/bin/bash
# Mock Datadog API Server for Simulation Testing
# Simulates Datadog API responses for testing without hitting real API

# Mock server configuration
MOCK_PORT="${MOCK_PORT:-9999}"
MOCK_LOG_FILE="${MOCK_LOG_FILE:-/tmp/mock-datadog-api.log}"
MOCK_PID_FILE="/tmp/mock-datadog-api.pid"

# Simulation scenarios
SCENARIO="${MOCK_SCENARIO:-normal}"  # normal, high_latency, errors, timeouts, rate_limit

# Start mock server
start_mock_server() {
    if [ -f "$MOCK_PID_FILE" ]; then
        local pid=$(cat "$MOCK_PID_FILE")
        if kill -0 "$pid" 2>/dev/null; then
            echo "[INFO] Mock server already running on port $MOCK_PORT (PID: $pid)"
            return 0
        fi
    fi

    echo "[INFO] Starting mock Datadog API server on port $MOCK_PORT"
    echo "[INFO] Scenario: $SCENARIO"
    echo "[INFO] Logs: $MOCK_LOG_FILE"

    # Start server in background
    nohup bash "$(dirname "${BASH_SOURCE[0]}")/mock-datadog-api-server.sh" > "$MOCK_LOG_FILE" 2>&1 &
    local pid=$!
    echo "$pid" > "$MOCK_PID_FILE"

    # Wait for server to start
    sleep 2

    if kill -0 "$pid" 2>/dev/null; then
        echo "[INFO] Mock server started (PID: $pid)"
        echo "[INFO] Configure scripts to use: export DD_SITE=localhost:$MOCK_PORT"
        return 0
    else
        echo "[ERROR] Failed to start mock server"
        rm -f "$MOCK_PID_FILE"
        return 1
    fi
}

# Stop mock server
stop_mock_server() {
    if [ ! -f "$MOCK_PID_FILE" ]; then
        echo "[WARN] Mock server not running"
        return 0
    fi

    local pid=$(cat "$MOCK_PID_FILE")
    if kill -0 "$pid" 2>/dev/null; then
        echo "[INFO] Stopping mock server (PID: $pid)"
        kill "$pid"
        rm -f "$MOCK_PID_FILE"
        echo "[INFO] Mock server stopped"
    else
        echo "[WARN] Mock server PID $pid not found"
        rm -f "$MOCK_PID_FILE"
    fi
}

# Get mock server status
mock_server_status() {
    if [ ! -f "$MOCK_PID_FILE" ]; then
        echo "[INFO] Mock server is not running"
        return 1
    fi

    local pid=$(cat "$MOCK_PID_FILE")
    if kill -0 "$pid" 2>/dev/null; then
        echo "[INFO] Mock server is running (PID: $pid, Port: $MOCK_PORT)"
        echo "[INFO] Scenario: $SCENARIO"
        return 0
    else
        echo "[WARN] Mock server PID file exists but process not running"
        rm -f "$MOCK_PID_FILE"
        return 1
    fi
}

# Generate mock APM trace response
mock_apm_response() {
    local service="$1"
    local from_ns="$2"
    local to_ns="$3"

    # Simulate different scenarios
    case "$SCENARIO" in
        high_latency)
            sleep 5
            ;;
        errors)
            cat <<'EOF'
{"errors": ["Simulated API error"]}
EOF
            return
            ;;
        rate_limit)
            cat <<'EOF'
{"errors": ["Rate limit exceeded"]}
EOF
            return
            ;;
        timeouts)
            sleep 120
            ;;
    esac

    # Normal response
    cat <<EOF
{
  "data": {
    "buckets": [
      {
        "by": {"resource_name": "GET /api/payments"},
        "computes": {
          "c0": 15420,
          "c1": 45000000,
          "c2": 650000000,
          "c3": 1200000000
        }
      },
      {
        "by": {"resource_name": "POST /api/checkout"},
        "computes": {
          "c0": 8932,
          "c1": 120000000,
          "c2": 890000000,
          "c3": 1500000000
        }
      },
      {
        "by": {"resource_name": "GET /api/orders"},
        "computes": {
          "c0": 25631,
          "c1": 25000000,
          "c2": 180000000,
          "c3": 450000000
        }
      }
    ]
  }
}
EOF
}

# Generate mock log search response
mock_log_search_response() {
    local query="$1"

    case "$SCENARIO" in
        errors)
            cat <<'EOF'
{"errors": ["Simulated log search error"]}
EOF
            return
            ;;
    esac

    cat <<EOF
{
  "data": [
    {
      "id": "log123",
      "attributes": {
        "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%S.000Z")",
        "message": "Payment processed successfully",
        "status": "info",
        "service": "payment-api",
        "tags": ["env:production"]
      }
    },
    {
      "id": "log124",
      "attributes": {
        "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%S.000Z")",
        "message": "Database connection timeout",
        "status": "error",
        "service": "payment-api",
        "tags": ["env:production"]
      }
    }
  ],
  "meta": {
    "page": {
      "after": "next_page_token"
    }
  }
}
EOF
}

# Generate mock security signals response
mock_security_signals_response() {
    case "$SCENARIO" in
        errors)
            cat <<'EOF'
{"errors": ["Simulated security API error"]}
EOF
            return
            ;;
    esac

    cat <<EOF
{
  "data": [
    {
      "id": "signal123",
      "type": "security_monitoring_signal",
      "attributes": {
        "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%S.000Z")",
        "message": "SQL injection attempt detected",
        "severity": "high",
        "rule": {
          "name": "SQL Injection Detection"
        },
        "tags": ["attack:sql-injection", "service:web-app"]
      }
    }
  ],
  "meta": {
    "page": {
      "after": "next_token"
    }
  }
}
EOF
}

# Generate mock metrics response
mock_metrics_response() {
    local metric="$1"

    case "$SCENARIO" in
        errors)
            cat <<'EOF'
{"errors": ["Simulated metrics API error"]}
EOF
            return
            ;;
    esac

    cat <<EOF
{
  "data": {
    "attributes": {
      "values": [
        [$(date +%s)000, 45.6],
        [$(( $(date +%s) - 60 ))000, 42.3],
        [$(( $(date +%s) - 120 ))000, 48.1]
      ]
    }
  }
}
EOF
}

# Generate mock SLO response
mock_slo_response() {
    case "$SCENARIO" in
        errors)
            cat <<'EOF'
{"errors": ["Simulated SLO API error"]}
EOF
            return
            ;;
    esac

    cat <<EOF
{
  "data": [
    {
      "id": "slo123",
      "type": "slos",
      "attributes": {
        "name": "Payment API Availability",
        "description": "99.9% uptime SLO",
        "tags": ["service:payment-api"],
        "slo_type": "metric",
        "type": "metric",
        "target_threshold": 99.9,
        "warning_threshold": 99.95
      }
    }
  ]
}
EOF
}

# Generate mock monitors response
mock_monitors_response() {
    case "$SCENARIO" in
        errors)
            cat <<'EOF'
{"errors": ["Simulated monitors API error"]}
EOF
            return
            ;;
    esac

    cat <<EOF
[
  {
    "id": 12345,
    "name": "High Error Rate",
    "type": "metric alert",
    "query": "avg(last_5m):sum:trace.web.request.errors{service:web-app} > 10",
    "message": "Error rate is high",
    "tags": ["service:web-app"],
    "options": {
      "thresholds": {
        "critical": 10,
        "warning": 5
      }
    },
    "overall_state": "Alert"
  }
]
EOF
}

# Export functions
export -f mock_apm_response
export -f mock_log_search_response
export -f mock_security_signals_response
export -f mock_metrics_response
export -f mock_slo_response
export -f mock_monitors_response
