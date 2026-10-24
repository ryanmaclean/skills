# Monitoring Library Quick Reference

## Setup (Add to your script)

```bash
source "$(dirname "${BASH_SOURCE[0]}")/lib/datadog-monitoring.sh"
init_monitoring "$(basename "$0")"
trap 'finalize_monitoring $?' EXIT INT TERM
```

## Environment Variables

```bash
export DD_MONITORING_ENABLED=true        # Required to enable
export DD_API_KEY=your_key_here          # Required for metrics/logs
export DD_MONITORING_SERVICE=my-service  # Optional (default: datadog-skill)
export DD_SITE=datadoghq.com            # Optional (default: datadoghq.com)
```

## Core Functions

### Operations (for timing)

```bash
start_operation "operation_name"
# ... your code ...
end_operation "ok" "tag1:value1" "tag2:value2"
```

### Metrics (gauge values)

```bash
send_metric "metric.name" 42 "tag1:value1"
```

### Logs (structured logging)

```bash
send_log "info" "Message" "tag1:value1"    # Levels: info, warn, error, debug
```

### Monitor Commands (wrap any command)

```bash
monitor_command "operation_name" your_command --args
```

### Direct Trace (manual timing)

```bash
send_trace "script.sh" "operation" 1234 "ok" "tag:value"
```

## Common Patterns

### API Call

```bash
start_operation "api_call"
send_metric "api.calls" 1 "endpoint:metrics"

response=$(curl -s "$URL")
exit_code=$?

if [ $exit_code -eq 0 ]; then
    end_operation "ok" "endpoint:metrics"
else
    send_log "error" "API failed" "exit_code:$exit_code"
    end_operation "error" "exit_code:$exit_code"
fi
```

### Data Processing

```bash
start_operation "process_data"
count=$(process_items "$data")
send_metric "items.processed" "$count"
send_log "info" "Processed $count items" "count:$count"
end_operation "ok" "count:$count"
```

### Error Handling

```bash
if [ $error_occurred -eq 1 ]; then
    send_log "error" "Operation failed: $reason"
    send_metric "errors" 1 "type:$error_type"
    end_operation "error" "reason:$reason"
    exit 1
fi
```

## Metric Naming Convention

```
datadog.skill.[category].[metric]

Examples:
  datadog.skill.api.calls
  datadog.skill.script.duration
  datadog.skill.items.processed
  datadog.skill.errors.count
```

## Tag Format

```
key:value

Examples:
  script:query-metrics.sh
  operation:api_call
  status:success
  endpoint:logs
  count:42
```

## View Data in Datadog

- **Traces**: APM → Traces → `service:datadog-skill`
- **Metrics**: Metrics → Explorer → `datadog.skill.*`
- **Logs**: Logs → Explorer → `service:datadog-skill`

## Troubleshooting

```bash
# Check config
print_monitoring_config

# Verify agent
curl http://localhost:8126/info

# Test with monitoring disabled (default)
./script.sh

# Test with monitoring enabled
DD_MONITORING_ENABLED=true DD_API_KEY=your_key ./script.sh
```

## Zero Impact Guarantee

When `DD_MONITORING_ENABLED` is not set or set to anything other than `true`:
- All functions return immediately (no-op)
- Zero performance overhead
- Scripts work exactly as before
- No API calls are made

## Bash 3.2 Compatible

Works on macOS default bash without any special requirements.
