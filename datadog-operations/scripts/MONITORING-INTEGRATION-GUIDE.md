# Monitoring Integration Guide

This guide shows how to add comprehensive monitoring to existing Datadog skill scripts using the monitoring library.

## Integration Steps

### Step 1: Source the Library

Add this near the top of your script (after the shebang and set commands):

```bash
#!/bin/bash
set -e

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source monitoring library
source "${SCRIPT_DIR}/lib/datadog-monitoring.sh"
```

### Step 2: Initialize Monitoring

Add initialization and trap for cleanup:

```bash
# Script name for monitoring
SCRIPT_NAME="$(basename "$0")"

# Initialize monitoring
init_monitoring "$SCRIPT_NAME"

# Ensure cleanup on exit
trap 'finalize_monitoring $?' EXIT INT TERM
```

### Step 3: Wrap Operations

Wrap your main operations with monitoring:

```bash
# Before:
query_api() {
    local result=$(curl -s "$API_URL")
    echo "$result"
}

# After:
query_api() {
    start_operation "api_query"

    send_log "info" "Starting API query" "endpoint:$API_URL"

    local result=$(curl -s "$API_URL")
    local exit_code=$?

    if [ $exit_code -eq 0 ]; then
        send_metric "api.calls" 1 "status:success" "endpoint:metrics"
        end_operation "ok" "endpoint:metrics"
    else
        send_log "error" "API query failed" "exit_code:$exit_code"
        send_metric "api.calls" 1 "status:failure" "endpoint:metrics"
        end_operation "error" "exit_code:$exit_code"
        return $exit_code
    fi

    echo "$result"
}
```

### Step 4: Add Metrics for Key Values

Send metrics for important values:

```bash
# Count items
ITEM_COUNT=$(echo "$response" | jq '.items | length')
send_metric "datadog.skill.items.processed" "$ITEM_COUNT" "script:$SCRIPT_NAME"

# Track sizes
DATA_SIZE=${#response}
send_metric "datadog.skill.response.bytes" "$DATA_SIZE" "endpoint:logs"

# Track execution counts
send_metric "datadog.skill.operations" 1 "operation:search" "status:success"
```

### Step 5: Log Important Events

Add logs for errors and key events:

```bash
# Info logs
send_log "info" "Processing started" "batch_id:$BATCH_ID" "items:$COUNT"

# Warning logs
if [ "$COUNT" -gt 1000 ]; then
    send_log "warn" "Large batch detected" "count:$COUNT"
fi

# Error logs
if [ $exit_code -ne 0 ]; then
    send_log "error" "Operation failed: $ERROR_MSG" "exit_code:$exit_code"
fi
```

## Complete Example: Before and After

### Before (Original Script)

```bash
#!/bin/bash
set -e

# Check environment
if [ -z "$DD_API_KEY" ] || [ -z "$DD_APP_KEY" ]; then
    echo "[ERROR] API keys required" >&2
    exit 1
fi

# Parse arguments
QUERY="${1:-error}"
LIMIT="${2:-100}"

# Query logs
echo "[INFO] Querying logs..." >&2

RESPONSE=$(curl -s -X POST "https://api.datadoghq.com/api/v2/logs/events/search" \
    -H "DD-API-KEY: ${DD_API_KEY}" \
    -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
    -H "Content-Type: application/json" \
    -d "{\"filter\": {\"query\": \"$QUERY\"}, \"page\": {\"limit\": $LIMIT}}")

# Process response
TOTAL=$(echo "$RESPONSE" | jq '.data | length')
echo "[INFO] Found $TOTAL logs" >&2

# Output results
echo "$RESPONSE" | jq .
```

### After (With Monitoring)

```bash
#!/bin/bash
set -e

# Get script directory and source monitoring
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/lib/datadog-monitoring.sh"

# Initialize monitoring
SCRIPT_NAME="$(basename "$0")"
init_monitoring "$SCRIPT_NAME"
trap 'finalize_monitoring $?' EXIT INT TERM

# Check environment
if [ -z "$DD_API_KEY" ] || [ -z "$DD_APP_KEY" ]; then
    send_log "error" "API keys required" "error:missing_credentials"
    echo "[ERROR] API keys required" >&2
    exit 1
fi

# Parse arguments
QUERY="${1:-error}"
LIMIT="${2:-100}"

# Query logs with monitoring
start_operation "query_logs"

echo "[INFO] Querying logs..." >&2
send_log "info" "Starting log query" "query:$QUERY" "limit:$LIMIT"

# Track API call
send_metric "datadog.skill.api.calls" 1 "endpoint:logs" "script:$SCRIPT_NAME"

RESPONSE=$(curl -s -X POST "https://api.datadoghq.com/api/v2/logs/events/search" \
    -H "DD-API-KEY: ${DD_API_KEY}" \
    -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
    -H "Content-Type: application/json" \
    -d "{\"filter\": {\"query\": \"$QUERY\"}, \"page\": {\"limit\": $LIMIT}}")

EXIT_CODE=$?

if [ $EXIT_CODE -ne 0 ]; then
    send_log "error" "API call failed" "exit_code:$EXIT_CODE"
    send_metric "datadog.skill.api.failures" 1 "endpoint:logs"
    end_operation "error" "exit_code:$EXIT_CODE"
    exit $EXIT_CODE
fi

# Process response
TOTAL=$(echo "$RESPONSE" | jq '.data | length')
echo "[INFO] Found $TOTAL logs" >&2

# Send metrics
send_metric "datadog.skill.logs.retrieved" "$TOTAL" "query:$QUERY"
send_log "info" "Log query completed" "count:$TOTAL"

end_operation "ok" "count:$TOTAL"

# Output results
echo "$RESPONSE" | jq .
```

## Minimal Integration (Quick Start)

If you want minimal changes, just add initialization and one trace:

```bash
#!/bin/bash
set -e

# Add these 4 lines at the top
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/lib/datadog-monitoring.sh"
init_monitoring "$(basename "$0")"
trap 'finalize_monitoring $?' EXIT INT TERM

# Your existing script code continues unchanged
# ...
```

This gives you:
- Script start/completion logs
- Total execution time trace
- Script execution metrics
- Error tracking

## Integration Patterns

### Pattern 1: API Wrapper

For scripts that call APIs:

```bash
call_api() {
    local endpoint="$1"
    start_operation "api_call"

    send_metric "api.requests" 1 "endpoint:$endpoint"

    local response=$(curl -s "$URL")
    local code=$?

    if [ $code -eq 0 ]; then
        end_operation "ok" "endpoint:$endpoint"
    else
        send_log "error" "API call failed" "endpoint:$endpoint"
        end_operation "error" "endpoint:$endpoint"
    fi

    echo "$response"
    return $code
}
```

### Pattern 2: Data Processing

For scripts that process data:

```bash
process_data() {
    local input_file="$1"
    start_operation "process_data"

    local input_size=$(wc -c < "$input_file")
    send_metric "processing.input.bytes" "$input_size"

    # Process data
    local result=$(transform_data "$input_file")
    local output_size=${#result}

    send_metric "processing.output.bytes" "$output_size"
    send_log "info" "Processing complete" "input:$input_size" "output:$output_size"

    end_operation "ok" "records:$(echo "$result" | wc -l)"
    echo "$result"
}
```

### Pattern 3: Error Handling

For scripts with complex error handling:

```bash
run_with_retry() {
    local max_attempts=3
    local attempt=1

    start_operation "retry_operation"

    while [ $attempt -le $max_attempts ]; do
        send_log "info" "Attempt $attempt of $max_attempts"

        if execute_operation; then
            send_metric "operation.attempts" "$attempt" "result:success"
            end_operation "ok" "attempts:$attempt"
            return 0
        fi

        send_log "warn" "Attempt $attempt failed, retrying..."
        attempt=$((attempt + 1))
        sleep 2
    done

    send_log "error" "All retry attempts failed"
    send_metric "operation.attempts" "$max_attempts" "result:failure"
    end_operation "error" "attempts:$max_attempts"
    return 1
}
```

## Monitoring Strategy

### What to Monitor

1. **Script Execution**
   - Always monitor: Total script execution time and status
   - Use: `init_monitoring` and `finalize_monitoring`

2. **API Calls**
   - Monitor: API call duration, success rate, endpoint
   - Use: `start_operation` / `end_operation` around API calls
   - Add: `send_metric` for call counts by endpoint

3. **Data Processing**
   - Monitor: Items processed, data volume, processing time
   - Use: `send_metric` for counts and sizes

4. **Errors**
   - Monitor: Error types, frequency, context
   - Use: `send_log` with level "error"
   - Add: Tags with error details

5. **Resource Usage**
   - Monitor: Response sizes, result counts
   - Use: `send_metric` for sizes and counts

### What NOT to Monitor

1. **Simple Variable Assignments** - Too granular
2. **Internal Helper Functions** - Creates noise
3. **Pure Validation Logic** - Usually too fast to matter
4. **Echo Statements** - Don't convert these to logs

### Tagging Strategy

Use consistent tags across all monitoring:

```bash
# Required tags
"script:$SCRIPT_NAME"           # Which script
"operation:query_logs"          # What operation

# Contextual tags
"endpoint:logs"                 # Which API endpoint
"service:web-app"               # Which service queried
"env:production"                # Environment

# Result tags
"status:success"                # Operation result
"exit_code:0"                   # Exit code

# Quantity tags
"count:42"                      # Number of items
"duration_ms:1234"              # Duration
```

## Testing Your Integration

### Test 1: Monitoring Disabled (Default)

```bash
./your-script.sh
```

Script should work exactly as before with no monitoring overhead.

### Test 2: Monitoring Enabled

```bash
DD_MONITORING_ENABLED=true \
DD_MONITORING_SERVICE=dd-skill-test \
DD_API_KEY=your_key \
./your-script.sh
```

Script should work the same but send data to Datadog.

### Test 3: Verify Data in Datadog

After running with monitoring enabled:

1. **Check Traces**: APM > Traces > filter by `service:dd-skill-test`
2. **Check Metrics**: Metrics > Explorer > search `datadog.skill.*`
3. **Check Logs**: Logs > Explorer > filter by `service:dd-skill-test`

### Test 4: Monitoring Failures Don't Break Script

```bash
# Invalid API key - monitoring fails but script continues
DD_MONITORING_ENABLED=true \
DD_API_KEY=invalid \
./your-script.sh
```

Script should still execute successfully.

## Common Issues

### Issue: "Function not found" errors

**Solution**: Ensure you've sourced the library:
```bash
source "${SCRIPT_DIR}/lib/datadog-monitoring.sh"
```

### Issue: No data in Datadog

**Solution**: Check configuration:
```bash
print_monitoring_config
```

Ensure `DD_MONITORING_ENABLED=true` is set.

### Issue: Traces missing but metrics/logs working

**Solution**: Check Datadog Agent:
```bash
curl http://localhost:8126/info
```

Ensure agent is running with APM enabled.

### Issue: Script slower with monitoring

**Solution**: This is expected but should be minimal. Monitoring adds:
- ~10-50ms per API call (metrics/logs)
- ~1-5ms per trace (local agent)

If performance is critical, disable monitoring or reduce frequency.

## Next Steps

1. **Start Simple**: Add just `init_monitoring` and `finalize_monitoring`
2. **Add Key Operations**: Wrap your main functions with `start_operation`/`end_operation`
3. **Add Metrics**: Send metrics for important values
4. **Add Logs**: Log errors and key events
5. **Test and Refine**: Run with monitoring enabled and check Datadog

## Examples in This Repo

- `example-monitored-script.sh` - Full featured example
- `lib/README-MONITORING.md` - Complete API reference
- `lib/datadog-monitoring.sh` - The library itself

## Support

For issues or questions about monitoring integration:
- Review the library README: `scripts/lib/README-MONITORING.md`
- Check the example script: `scripts/example-monitored-script.sh`
- Test with: `print_monitoring_config`
