#!/bin/bash
set -e

# Example: Monitored Script
# Demonstrates how to integrate Datadog monitoring into a script
# This example queries metrics and sends traces, metrics, and logs to Datadog

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source the monitoring library
source "${SCRIPT_DIR}/lib/datadog-monitoring.sh"

# Script name for monitoring
SCRIPT_NAME="example-monitored-script"

# Initialize monitoring and set up trap for cleanup
init_monitoring "$SCRIPT_NAME"

# Trap to ensure monitoring finalization on exit
trap 'finalize_monitoring $?' EXIT INT TERM

# Function to demonstrate monitored operations
query_datadog_metrics() {
    local metric="$1"
    local service="$2"

    # Start timing this operation
    start_operation "query_metrics"

    # Log the operation start
    send_log "info" "Starting metric query: $metric for service: $service" \
        "operation:query_metrics" "metric:$metric"

    # Check required environment variables
    if [ -z "$DD_API_KEY" ] || [ -z "$DD_APP_KEY" ]; then
        send_log "error" "Missing required API keys" \
            "operation:query_metrics" "error:missing_credentials"
        end_operation "error" "error:missing_credentials"
        return 1
    fi

    local site="${DD_SITE:-datadoghq.com}"
    local from=$(($(date +%s) - 3600))
    local to=$(date +%s)

    # Build query
    local query="avg:${metric}{service:${service}}"

    # Send metric for API call
    send_metric "datadog.skill.api.calls" 1 \
        "endpoint:metrics_query" "script:$SCRIPT_NAME"

    # Make the API call
    local response
    local exit_code=0

    response=$(curl -s -X GET "https://api.${site}/api/v1/query" \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        -G \
        --data-urlencode "query=${query}" \
        --data-urlencode "from=${from}" \
        --data-urlencode "to=${to}") || exit_code=$?

    if [ $exit_code -ne 0 ]; then
        send_log "error" "API call failed with exit code: $exit_code" \
            "operation:query_metrics" "exit_code:$exit_code"
        end_operation "error" "exit_code:$exit_code"
        return 1
    fi

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        local error_msg=$(echo "$response" | jq -r '.errors[0]' 2>/dev/null || echo "Unknown error")
        send_log "error" "API returned error: $error_msg" \
            "operation:query_metrics" "error:api_error"
        end_operation "error" "error:api_error"
        return 1
    fi

    # Check if data exists
    local series_count=0
    if echo "$response" | jq -e '.series' > /dev/null 2>&1; then
        series_count=$(echo "$response" | jq '.series | length')
    fi

    # Send metric for results
    send_metric "datadog.skill.metrics.series_count" "$series_count" \
        "metric:$metric" "service:$service" "script:$SCRIPT_NAME"

    if [ "$series_count" -eq 0 ]; then
        send_log "warn" "No data found for metric query" \
            "operation:query_metrics" "metric:$metric" "series_count:0"
        end_operation "ok" "series_count:0"
        echo "No data found"
        return 0
    fi

    # Calculate statistics
    local point_count=$(echo "$response" | jq '[.series[].pointlist[][1] | select(. != null)] | length')

    send_log "info" "Query completed successfully: $point_count data points" \
        "operation:query_metrics" "series_count:$series_count" "point_count:$point_count"

    # End the operation successfully
    end_operation "ok" "series_count:$series_count" "point_count:$point_count"

    echo "Found $series_count series with $point_count data points"
    return 0
}

# Function to demonstrate error handling with monitoring
simulate_data_processing() {
    local data_size="$1"

    start_operation "process_data"

    send_log "info" "Processing data batch" \
        "operation:process_data" "data_size:$data_size"

    # Simulate processing with metric
    send_metric "datadog.skill.data.processed_bytes" "$data_size" \
        "script:$SCRIPT_NAME"

    # Simulate some work
    sleep 0.5

    # Random success/failure for demonstration
    local random_val=$((RANDOM % 10))
    if [ $random_val -gt 7 ]; then
        send_log "error" "Data processing failed: validation error" \
            "operation:process_data" "error:validation_failed"
        end_operation "error" "error:validation_failed"
        return 1
    fi

    send_log "info" "Data processing completed successfully" \
        "operation:process_data"
    end_operation "ok"

    return 0
}

# Main execution
main() {
    echo "================================================"
    echo "Datadog Monitored Script Example"
    echo "================================================"
    echo ""

    # Print monitoring configuration
    if [ "${DD_MONITORING_ENABLED:-false}" = "true" ]; then
        echo "Monitoring is ENABLED"
        print_monitoring_config
    else
        echo "Monitoring is DISABLED (set DD_MONITORING_ENABLED=true to enable)"
    fi
    echo ""

    # Parse arguments
    local metric="${1:-system.cpu.user}"
    local service="${2:-web-app}"

    echo "Example 1: Query Metrics with Monitoring"
    echo "----------------------------------------"
    echo "Metric: $metric"
    echo "Service: $service"
    echo ""

    # Execute monitored operation
    if query_datadog_metrics "$metric" "$service"; then
        echo "Success: Metric query completed"
        send_metric "datadog.skill.example.success" 1 "operation:query_metrics"
    else
        echo "Error: Metric query failed"
        send_metric "datadog.skill.example.failure" 1 "operation:query_metrics"
    fi

    echo ""
    echo "Example 2: Data Processing with Monitoring"
    echo "-------------------------------------------"

    # Process multiple batches
    for i in 1 2 3; do
        local data_size=$((RANDOM % 1000 + 500))
        echo "Processing batch $i (${data_size} bytes)..."

        if simulate_data_processing "$data_size"; then
            echo "  Batch $i: Success"
        else
            echo "  Batch $i: Failed"
        fi
    done

    echo ""
    echo "Example 3: Using monitor_command Helper"
    echo "----------------------------------------"

    # This wraps any command with automatic monitoring
    if monitor_command "list_scripts" ls -l "${SCRIPT_DIR}"/*.sh; then
        echo "Command completed successfully"
    else
        echo "Command failed"
    fi

    echo ""
    echo "================================================"
    echo "Script execution complete"
    echo ""

    if [ "${DD_MONITORING_ENABLED:-false}" = "true" ]; then
        echo "Check Datadog for traces, metrics, and logs:"
        echo "  - Traces: APM > Traces > service:${DD_MONITORING_SERVICE:-dd-skill-test}"
        echo "  - Metrics: Metrics > Explorer > datadog.skill.*"
        echo "  - Logs: Logs > Explorer > service:${DD_MONITORING_SERVICE:-dd-skill-test}"
    fi

    return 0
}

# Run main function
main "$@"
