# Datadog Monitoring Library

A comprehensive bash library for adding observability to Datadog skill scripts through traces, metrics, and logs.

## Overview

The `datadog-monitoring.sh` library provides a simple, non-intrusive way to instrument bash scripts with Datadog monitoring. When enabled, it sends:

- **Traces** to the Datadog Agent (APM)
- **Metrics** to the Datadog API
- **Logs** to the Datadog Logs API

When disabled, all monitoring functions are no-ops, ensuring scripts continue to work normally without any overhead.

## Quick Start

### 1. Source the Library

```bash
#!/bin/bash
source "$(dirname "${BASH_SOURCE[0]}")/lib/datadog-monitoring.sh"
```

### 2. Initialize Monitoring

```bash
SCRIPT_NAME="my-script"
init_monitoring "$SCRIPT_NAME"
trap 'finalize_monitoring $?' EXIT INT TERM
```

### 3. Use Monitoring Functions

```bash
# Wrap operations with start/end
start_operation "database_query"
# ... your code ...
end_operation "ok" "rows:42"

# Send custom metrics
send_metric "custom.processing.time" 1234 "operation:transform"

# Send logs
send_log "info" "Processing completed successfully" "batch_id:123"
```

## Configuration

Set these environment variables to configure monitoring:

### Required for Monitoring

- `DD_MONITORING_ENABLED=true` - Enable monitoring (default: false)
- `DD_API_KEY` - Datadog API key (for metrics and logs)

### Optional Configuration

- `DD_MONITORING_SERVICE` - Service name (default: "datadog-skill")
- `DD_SITE` - Datadog site (default: "datadoghq.com")
- `DD_AGENT_HOST` - Agent host for traces (default: "localhost")
- `DD_TRACE_AGENT_PORT` - Trace agent port (default: 8126)
- `DD_ENV` - Environment tag (default: "production")
- `DD_HOSTNAME` - Override hostname (default: system hostname)

### Example Configuration

```bash
export DD_MONITORING_ENABLED=true
export DD_MONITORING_SERVICE=datadog-skill
export DD_API_KEY=your_api_key_here
export DD_SITE=datadoghq.com
export DD_ENV=production
```

## API Reference

### Initialization and Cleanup

#### `init_monitoring(script_name)`

Initialize monitoring for a script. Call this at the start of your script.

```bash
init_monitoring "query-metrics.sh"
```

#### `finalize_monitoring([exit_code])`

Finalize monitoring and send completion metrics. Call this at script end or in a trap.

```bash
trap 'finalize_monitoring $?' EXIT INT TERM
```

### Traces

#### `send_trace(script_name, operation, duration_ms, status, tags...)`

Send a trace span to the Datadog Agent.

- `script_name`: Name of the script
- `operation`: Operation name (e.g., "query_api", "process_data")
- `duration_ms`: Duration in milliseconds
- `status`: "ok" or "error"
- `tags`: Optional tags in format "key:value"

```bash
send_trace "search-logs.sh" "api_query" 234 "ok" "service:web" "env:prod"
```

#### `start_operation(operation_name)`

Start timing an operation. Use with `end_operation()`.

```bash
start_operation "database_query"
# ... your code ...
end_operation "ok" "rows:100"
```

#### `end_operation(status, tags...)`

End an operation and automatically send a trace with duration.

```bash
end_operation "ok" "result:success" "count:42"
```

#### `monitor_command(operation_name, command...)`

Wrap any command with automatic monitoring.

```bash
monitor_command "fetch_data" curl -s https://api.example.com/data
```

### Metrics

#### `send_metric(metric_name, value, tags...)`

Send a metric to Datadog API.

- `metric_name`: Metric name (e.g., "datadog.skill.requests")
- `value`: Numeric value
- `tags`: Optional tags in format "key:value"

```bash
send_metric "datadog.skill.api.calls" 1 "endpoint:logs" "status:success"
send_metric "datadog.skill.processing.bytes" 1024 "operation:transform"
```

### Logs

#### `send_log(level, message, tags...)`

Send a log entry to Datadog.

- `level`: Log level (error, warn, info, debug)
- `message`: Log message
- `tags`: Optional tags in format "key:value"

```bash
send_log "info" "Query completed successfully" "operation:search" "count:42"
send_log "error" "API call failed: timeout" "endpoint:metrics" "retry:3"
```

### Utilities

#### `send_script_metrics(script_name, duration_ms, exit_code)`

Send standard script execution metrics.

```bash
send_script_metrics "query-metrics.sh" 5000 0
```

#### `print_monitoring_config()`

Print current monitoring configuration for debugging.

```bash
print_monitoring_config
```

## Integration Examples

### Simple Script

```bash
#!/bin/bash
set -e

# Source monitoring library
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/lib/datadog-monitoring.sh"

# Initialize monitoring
SCRIPT_NAME="simple-script"
init_monitoring "$SCRIPT_NAME"
trap 'finalize_monitoring $?' EXIT INT TERM

# Your script logic with monitoring
start_operation "main_task"
send_log "info" "Starting main task"

# ... your code ...

send_metric "custom.task.items" 42 "status:success"
end_operation "ok"
```

### API Query Script

```bash
#!/bin/bash
set -e

source "$(dirname "${BASH_SOURCE[0]}")/lib/datadog-monitoring.sh"

SCRIPT_NAME="api-query"
init_monitoring "$SCRIPT_NAME"
trap 'finalize_monitoring $?' EXIT INT TERM

query_api() {
    start_operation "api_query"

    send_log "info" "Querying API" "endpoint:$1"
    send_metric "api.requests" 1 "endpoint:$1"

    local response=$(curl -s "$1")
    local exit_code=$?

    if [ $exit_code -eq 0 ]; then
        send_log "info" "API query successful"
        end_operation "ok" "endpoint:$1"
    else
        send_log "error" "API query failed" "exit_code:$exit_code"
        end_operation "error" "exit_code:$exit_code"
        return $exit_code
    fi

    echo "$response"
}

query_api "https://api.datadoghq.com/api/v1/validate"
```

### Data Processing Pipeline

```bash
#!/bin/bash
set -e

source "$(dirname "${BASH_SOURCE[0]}")/lib/datadog-monitoring.sh"

init_monitoring "data-pipeline"
trap 'finalize_monitoring $?' EXIT INT TERM

process_batch() {
    local batch_id="$1"
    local batch_size="$2"

    start_operation "process_batch"
    send_log "info" "Processing batch" "batch_id:$batch_id" "size:$batch_size"

    # Processing logic
    sleep 1

    send_metric "pipeline.batch.size" "$batch_size" "batch_id:$batch_id"
    send_metric "pipeline.batch.processed" 1 "status:success"

    end_operation "ok" "batch_id:$batch_id"
}

# Process multiple batches
for i in {1..5}; do
    process_batch "batch-$i" $((RANDOM % 1000))
done
```

## Viewing Data in Datadog

### Traces (APM)

Navigate to: **APM > Traces**

Filter by: `service:datadog-skill` (or your custom service name)

You'll see:
- Script execution traces
- Individual operation spans
- Duration, status, and tags
- Error traces highlighted

### Metrics

Navigate to: **Metrics > Explorer**

Search for: `datadog.skill.*`

Available metrics:
- `datadog.skill.script.duration` - Script execution time
- `datadog.skill.script.executions` - Execution count
- Custom metrics you've defined

### Logs

Navigate to: **Logs > Explorer**

Filter by: `service:datadog-skill`

You'll see:
- Script start/completion logs
- Custom logs with levels
- Structured tags for filtering
- Correlation with traces via trace IDs

## Performance Considerations

### No Impact When Disabled

When `DD_MONITORING_ENABLED` is not set to `true`, all monitoring functions return immediately without any processing. This ensures zero overhead in production unless explicitly enabled.

### Graceful Failure

All monitoring functions fail silently (return success) if:
- The Datadog Agent is unreachable
- API keys are missing or invalid
- Network requests fail

This ensures your scripts continue to work even if monitoring fails.

### Bash 3.2 Compatibility

The library is fully compatible with bash 3.2, which is the default on macOS. It avoids:
- Associative arrays
- GNU-specific date formats
- Modern bash features not in 3.2

## Troubleshooting

### No Data Appearing in Datadog

1. **Check monitoring is enabled:**
   ```bash
   echo $DD_MONITORING_ENABLED  # Should be "true"
   ```

2. **Verify configuration:**
   ```bash
   print_monitoring_config
   ```

3. **Check Datadog Agent is running:**
   ```bash
   curl -s http://localhost:8126/info
   ```

4. **Verify API keys:**
   ```bash
   curl -s https://api.datadoghq.com/api/v1/validate \
     -H "DD-API-KEY: $DD_API_KEY"
   ```

### Traces Not Appearing

- Ensure the Datadog Agent is running and reachable
- Check `DD_AGENT_HOST` and `DD_TRACE_AGENT_PORT` settings
- Verify the agent has APM enabled

### Metrics/Logs Not Appearing

- Verify `DD_API_KEY` is set correctly
- Check `DD_SITE` matches your Datadog account region
- Logs may take a few minutes to appear in the UI

### Script Errors

If you see errors about functions or variables:
- Ensure you've sourced the library before using functions
- Check bash version: `bash --version` (should be 3.2+)
- Verify file paths are correct

## Advanced Usage

### Custom Trace Context

You can build complex trace hierarchies by managing operation stacks:

```bash
start_operation "parent_operation"
  start_operation "child_operation_1"
  end_operation "ok"

  start_operation "child_operation_2"
  end_operation "ok"
end_operation "ok"
```

### Conditional Monitoring

Enable monitoring only for specific operations:

```bash
if [ "$MONITOR_THIS_OPERATION" = "true" ]; then
    export DD_MONITORING_ENABLED=true
fi
```

### Dynamic Tags

Add dynamic tags based on runtime conditions:

```bash
local tags=("operation:query")

if [ "$ERROR_OCCURRED" = "true" ]; then
    tags+=("has_errors:true")
fi

end_operation "ok" "${tags[@]}"
```

## Best Practices

1. **Always use init/finalize**: Set up proper initialization and cleanup with traps
2. **Use meaningful operation names**: Make traces easy to understand
3. **Tag appropriately**: Add tags that help with filtering and analysis
4. **Log errors**: Always log errors with context
5. **Send relevant metrics**: Track what matters for your use case
6. **Keep monitoring optional**: Scripts should work without monitoring enabled

## See Also

- [Example Monitored Script](../example-monitored-script.sh)
- [Datadog APM Documentation](https://docs.datadoghq.com/tracing/)
- [Datadog Metrics Documentation](https://docs.datadoghq.com/metrics/)
- [Datadog Logs Documentation](https://docs.datadoghq.com/logs/)
