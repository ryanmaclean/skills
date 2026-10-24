# Datadog Skill - Quick Start

Get started with Datadog operations in 5 minutes.

## Setup (30 seconds)

```bash
# Set API credentials
export DD_API_KEY=your_api_key
export DD_APP_KEY=your_app_key
export DD_SITE=datadoghq.com  # or your Datadog site

# Verify setup
bash scripts/verify-setup.sh
```

Get keys from: Datadog → Organization Settings → API Keys / Application Keys

## Common Tasks

### 1. Investigate Service Issues (One Command)

```bash
# Run comprehensive investigation
bash scripts/investigate-service.sh --service payment-api
```

Returns:
- APM performance issues
- Security threats
- Watchdog anomalies
- Error logs
- SLO status
- **Overall severity assessment**
- **Recommended actions**

**Output:**
```json
{
  "overall_status": "warning",
  "issues_found": 2,
  "severity": {"critical": 0, "warnings": 2},
  "recommendations": [
    {"priority": "high", "action": "Optimize slow endpoints or increase capacity"}
  ]
}
```

### 2. Find and Fix Slow Endpoints

```bash
# Find slow endpoints
bash scripts/query-apm.sh --service payment-api --duration 1h

# Create monitor for slowest endpoint
bash scripts/manage-monitors.sh create \
  --name "Payment API - Slow Checkout" \
  --query "avg(last_5m):avg:trace.express.request.duration{service:payment-api,resource_name:/checkout} > 500" \
  --message "Checkout endpoint slow @slack-ops"
```

### 3. Track Error Budget

```bash
# Calculate remaining error budget
bash scripts/calculate-error-budget.sh --service payment-api

# Check SLO status
bash scripts/query-slos.sh --service payment-api
```

**Output:**
```json
{
  "status": "warning",
  "error_budget": {
    "remaining_errors": 450,
    "budget_remaining_percent": 45.3
  },
  "burn_rate": {
    "errors_per_day": 15.2,
    "days_to_exhaustion": "29.6"
  }
}
```

### 4. Security Investigation

```bash
# Check for threats
bash scripts/query-security-signals.sh --severity critical --duration 24h

# Search logs for attack patterns
bash scripts/search-logs.sh --query "sql injection OR xss" --duration 24h
```

### 5. Monitor New Service

```bash
# Create uptime check
bash scripts/manage-synthetics.sh create-api \
  --name "Payment API Health" \
  --url "https://api.example.com/health"

# Create dashboard
bash scripts/create-dashboard.sh --service payment-api --title "Payment API" --type apm

# Create error rate monitor
bash scripts/manage-monitors.sh create \
  --name "High Error Rate" \
  --query "avg(last_5m):sum:trace.express.request.errors{service:payment-api}.as_count() / sum:trace.express.request.hits{service:payment-api}.as_count() > 0.05" \
  --message "Error rate > 5% @pagerduty"
```

## Integration Workflows

### Investigate → Create Incident → Trigger Workflow

```bash
# 1. Investigate
RESULT=$(bash scripts/investigate-service.sh --service payment-api 2>/dev/null)
STATUS=$(echo "$RESULT" | jq -r '.overall_status')

# 2. Create incident if critical
if [ "$STATUS" = "critical" ]; then
  INCIDENT=$(bash scripts/manage-incidents.sh create \
    --title "Payment API Critical Issue" \
    --service payment-api \
    --severity SEV-1 2>/dev/null)

  INCIDENT_ID=$(echo "$INCIDENT" | jq -r '.id')

  # 3. Trigger remediation workflow
  bash scripts/trigger-workflow.sh run --id remediation-workflow-123 \
    --input "{\"incident_id\": \"$INCIDENT_ID\", \"service\": \"payment-api\"}"
fi
```

### APM → Logs Correlation (Trace IDs)

```bash
# 1. Find slow requests with trace IDs
TRACES=$(bash scripts/query-apm.sh --service payment-api --duration 1h 2>/dev/null)

# 2. Extract trace IDs from errors
TRACE_IDS=$(echo "$TRACES" | jq -r '.endpoints[]? | select(.error_count > 0) | .trace_id // empty' | head -5 | tr '\n' ' ')

# 3. Search logs for those traces
if [ -n "$TRACE_IDS" ]; then
  bash scripts/search-logs.sh --query "trace_id:($TRACE_IDS)" --duration 1h
fi
```

### Error Budget → Alert → Incident

```bash
# 1. Check error budget
BUDGET=$(bash scripts/calculate-error-budget.sh --service payment-api 2>/dev/null)
DAYS_LEFT=$(echo "$BUDGET" | jq -r '.burn_rate.days_to_exhaustion')

# 2. Create incident if < 7 days
if [ "$DAYS_LEFT" != "infinite" ] && (( $(echo "$DAYS_LEFT < 7" | bc -l) )); then
  bash scripts/manage-incidents.sh create \
    --title "Error Budget Critical - $DAYS_LEFT days remaining" \
    --service payment-api \
    --severity SEV-2
fi
```

### Weekly Service Report

```bash
#!/bin/bash
# Generate weekly service health report

SERVICE="payment-api"
DATE=$(date +%Y-%m-%d)

{
  echo "# Service Health Report: $SERVICE"
  echo "Date: $DATE"
  echo ""

  echo "## Investigation"
  bash scripts/investigate-service.sh --service "$SERVICE" --duration 7d 2>/dev/null | jq

  echo "## Error Budget"
  bash scripts/calculate-error-budget.sh --service "$SERVICE" 2>/dev/null | jq

  echo "## Cost Analysis"
  bash scripts/analyze-usage-cost.sh --duration 7d --product all 2>/dev/null | jq '.cost_summary'

} > "report-${SERVICE}-${DATE}.json"

echo "Report generated: report-${SERVICE}-${DATE}.json"
```

## Cheat Sheet

**Investigation:**
```bash
investigate-service.sh --service <name>      # Full investigation
query-apm.sh --service <name>                # Performance
search-logs.sh --query "status:error"        # Error logs
query-security-signals.sh --severity critical # Threats
```

**Automation:**
```bash
manage-monitors.sh create --name <name> --query <query> --message <msg>
create-dashboard.sh --service <name> --title <title> --type apm
manage-synthetics.sh create-api --name <name> --url <url>
manage-incidents.sh create --title <title> --service <name> --severity SEV-1
```

**Analysis:**
```bash
calculate-error-budget.sh --service <name>   # Error budget
query-slos.sh --service <name>               # SLO status
analyze-usage-cost.sh --duration 30d         # Costs
analyze-llm.sh --service <name>              # LLM usage
```

## Next Steps

1. **See all scripts:** `ls scripts/*.sh`
2. **Get help:** `bash scripts/<script-name>.sh --help`
3. **Full documentation:** Read [SKILL.md](SKILL.md)
4. **Integration examples:** See README.md
5. **Platform coverage:** See [COVERAGE.md](COVERAGE.md)

## Troubleshooting

**API Authentication Failed:**
```bash
# Check credentials are set
echo $DD_API_KEY $DD_APP_KEY

# Verify they work
bash scripts/verify-setup.sh
```

**No Data Found:**
- Ensure service name matches exactly (case-sensitive)
- Check data retention for your Datadog plan
- Verify APM/logs are being ingested

**Script Errors:**
- All scripts require DD_API_KEY and DD_APP_KEY
- Use `--help` flag to see usage
- Check stderr output for detailed error messages

## Tips

- **JSON parsing:** Pipe to `jq` for specific fields
- **Silent mode:** Add `2>/dev/null` to hide status messages
- **Save results:** Redirect to file `> results.json`
- **Chain scripts:** Use `$()` to capture output and pipe between scripts

**Example:**
```bash
# Find slow endpoint, create monitor
ENDPOINT=$(bash scripts/query-apm.sh --service payment-api 2>/dev/null | \
  jq -r '.endpoints[0].name')

bash scripts/manage-monitors.sh create \
  --name "Slow: $ENDPOINT" \
  --query "avg(last_5m):avg:trace.express.request.duration{resource_name:$ENDPOINT} > 500" \
  --message "Endpoint slow @ops"
```
