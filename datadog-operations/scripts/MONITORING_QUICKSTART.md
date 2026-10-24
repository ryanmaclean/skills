# Monitoring Quick Start

## Enable Monitoring in 30 Seconds

### 1. Set Environment Variables
```bash
export DD_MONITORING_ENABLED=true
export DD_API_KEY=your_datadog_api_key
```

### 2. Run Any Script
```bash
./scripts/query-apm.sh --service my-service
```

That's it! The script now sends traces, metrics, and logs to Datadog.

## What You Get

### Automatic Traces
Every script execution creates a trace showing:
- Total execution time
- API call duration
- Success or failure status
- Error details if failed

### Business Metrics
Each script reports relevant metrics:
- **query-apm.sh**: Endpoint counts, request volumes, slow endpoints
- **search-logs.sh**: Log counts, error counts, warning counts
- **query-security-signals.sh**: Signal counts by severity
- **query-watchdog.sh**: Anomaly counts by type
- **query-metrics.sh**: Data point counts, anomaly detections
- **query-slos.sh**: SLO status, budget consumption

### Execution Logs
Start and completion events with:
- Script name
- Duration
- Exit code
- Execution status

## Configuration Options

### Required
```bash
DD_MONITORING_ENABLED=true  # Enable monitoring
DD_API_KEY=xxx              # Datadog API key
```

### Optional
```bash
DD_SITE=datadoghq.com              # Datadog site (default: datadoghq.com)
DD_MONITORING_SERVICE=dd-skill-test # Service name (default: dd-skill-test)
DD_AGENT_HOST=localhost             # Agent for traces (default: localhost)
DD_TRACE_AGENT_PORT=8126           # Trace port (default: 8126)
DD_ENV=production                  # Environment tag
DD_HOSTNAME=$(hostname)            # Custom hostname
```

## View Your Data

### In Datadog UI

1. **Traces**: APM → Traces → Filter by service:`dd-skill-test`
2. **Metrics**: Metrics Explorer → Search for `datadog.skill.*`
3. **Logs**: Logs → Filter by service:`dd-skill-test`

### Key Metrics to Watch

```
datadog.skill.script.duration          # How long scripts take
datadog.skill.script.executions        # How often scripts run
datadog.skill.apm.endpoints.count      # APM query results
datadog.skill.logs.errors              # Error log counts
datadog.skill.security.signals.count   # Security signals found
datadog.skill.slos.breaching           # SLOs in breach
```

## Disable Monitoring

Just unset the environment variable:
```bash
unset DD_MONITORING_ENABLED
```

Or set it to false:
```bash
export DD_MONITORING_ENABLED=false
```

Scripts will continue to work normally with zero monitoring overhead.

## Common Use Cases

### Monitor Script Performance
```bash
# Enable monitoring
export DD_MONITORING_ENABLED=true
export DD_API_KEY=xxx

# Run script
./scripts/query-apm.sh --service web-api

# Check trace in Datadog to see:
# - Total execution time
# - API call duration
# - Any errors
```

### Track Error Rates Over Time
```bash
# Run log searches regularly
while true; do
  ./scripts/search-logs.sh --status error --duration 1h
  sleep 300
done

# View in Datadog:
# Metric: datadog.skill.logs.errors
# Chart: Line graph over time
```

### Alert on Script Failures
Create a monitor in Datadog:
```
Metric: datadog.skill.script.executions
Filter: status:failure
Alert when: > 0 failures in 5 minutes
```

### Dashboard Example
```
Widget 1: Script Execution Count (by script name)
  Metric: sum:datadog.skill.script.executions{*} by {script}

Widget 2: Average Script Duration
  Metric: avg:datadog.skill.script.duration{*} by {script}

Widget 3: Error Rate
  Metric: sum:datadog.skill.script.executions{status:failure}

Widget 4: Recent Traces
  Type: Trace List
  Filter: service:dd-skill-test
```

## Troubleshooting

### Monitoring Not Working?

Check environment variables:
```bash
env | grep DD_
```

Test library:
```bash
source scripts/lib/datadog-monitoring.sh
print_monitoring_config
```

### Scripts Still Work Without Monitoring?

Yes! That's by design. The integration:
- Fails silently if disabled
- Doesn't break scripts if library is missing
- Has zero overhead when disabled

### Want More Detail?

See full documentation:
- `MONITORING_INTEGRATION.md` - Complete integration guide
- `MONITORING_CHANGES_SUMMARY.md` - What changed in each script
- `scripts/lib/datadog-monitoring.sh` - Library source code

## Integration Pattern

If you want to add monitoring to your own scripts:

```bash
#!/bin/bash
set -e

# 1. Add this at the top
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/datadog-monitoring.sh" 2>/dev/null || true
init_monitoring "$(basename "$0")"
trap 'finalize_monitoring $?' EXIT INT TERM

# 2. Wrap API calls
start_operation "api_call"
result=$(curl ...)
if [ $? -ne 0 ]; then
    end_operation "error"
else
    end_operation "ok"
fi

# 3. Send business metrics
count=$(echo "$result" | jq 'length')
send_metric "my.custom.metric" "$count" "status:ok"

# Script continues normally...
```

That's all you need!
