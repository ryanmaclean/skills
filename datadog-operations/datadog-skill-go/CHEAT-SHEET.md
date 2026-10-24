# Datadog CLI - Cheat Sheet

**Ultra-compact reference** - Print this page for quick access

---

## Setup

```bash
export DD_API_KEY="your-api-key"
export DD_APP_KEY="your-app-key"
export DD_SITE="datadoghq.com"  # Optional
```

---

## Essential Commands

### Service Context & Health
```bash
dd context                              # Auto-detect service
datadog-cli health check --service <name>        # Health status
```

### APM & Traces
```bash
datadog-cli apm services --from 1h               # List services
datadog-cli apm traces --service <name>          # Get traces
dd spans search --query "error"         # Search spans
```

### Logs
```bash
datadog-cli logs search --query "error" --from 1h    # Search logs
datadog-cli logs tail --follow                       # Live stream
```

### Metrics
```bash
datadog-cli metrics query --metric <name> --from 1h  # Query metric
datadog-cli metrics list --pattern "cpu*"            # List metrics
```

### Monitors & Incidents
```bash
datadog-cli monitors list                        # List monitors
datadog-cli monitors get --id <id>               # Get monitor
datadog-cli incidents list --status open         # Open incidents
```

### SLOs
```bash
datadog-cli slos list                            # List SLOs
datadog-cli slos get --id <id>                   # Get SLO
dd error-budgets check --slo-id <id>    # Check budget
```

### Cost & Usage
```bash
dd cost estimate --from 1m              # Monthly cost
dd usage-insights analyze --from 1m     # Usage analysis
```

### ML & Predictions
```bash
dd anomalies detect --metric <name> --from 7d       # Detect anomalies
dd ml-insights train --service <name> --from 30d    # Train model
dd predictions predict --target incidents --horizon 24h  # Predict
dd recommendations suggest --service <name>              # Recommendations
```

---

## Common Patterns

### Investigation Workflow
```bash
dd context                                  # 1. Identify service
datadog-cli health check --service api               # 2. Check health
datadog-cli logs search --query "error" --service api --from 1h  # 3. Find errors
datadog-cli apm traces --service api --filter "error:true"       # 4. Get traces
dd correlation analyze --service api        # 5. Correlate issues
```

### SLO Monitoring
```bash
datadog-cli slos list                                # 1. List all SLOs
datadog-cli slos get --id abc123                     # 2. Check status
dd error-budgets check --slo-id abc123      # 3. Check budget
dd slo-history query --slo-id abc123 --from 7d  # 4. View history
```

### Deployment Validation
```bash
datadog-cli health check --service api --env staging     # 1. Check staging
datadog-cli deploy validate --service api --version v2   # 2. Validate safety
# Deploy to production
datadog-cli health check --service api --env production  # 3. Verify production
```

---

## Global Flags

| Flag | Description | Example |
|------|-------------|---------|
| `--json` | JSON output | `--json` |
| `--from` | Start time | `--from 1h`, `--from 7d` |
| `--to` | End time | `--to 30m` |
| `--service` | Filter by service | `--service api` |
| `--env` | Filter by environment | `--env production` |

---

## Time Formats

| Format | Meaning |
|--------|---------|
| `30m` | 30 minutes ago |
| `1h` | 1 hour ago |
| `24h` | 24 hours ago |
| `7d` | 7 days ago |
| `1w` | 1 week ago |
| `1M` | 1 month ago |

---

## Output Modes

### Text (Default)
```bash
datadog-cli apm services --from 1h
# Human-readable table
```

### JSON (Scripting)
```bash
datadog-cli apm services --from 1h --json
# Machine-parseable

# With jq
datadog-cli apm services --from 1h --json | jq '.services[]'
```

---

## All Commands (by Phase)

### Foundation
`context` `apm` `logs` `metrics` `llm` `database` `security` `watchdog` `catalog`

### Data Management
`events` `tags` `integrations`

### SRE & Reliability
`slos` `slo-corrections` `error-budgets` `slo-history`

### FinOps
`cost` `usage-insights`

### Management (22 commands)
`incidents` `monitors` `dashboards` `workflows` `synthetics` `rum` `network` `cicd` `dora` `cases` `containers` `kubernetes` `serverless` `status-pages` `on-call` `downtimes` `notebooks` `teams` `users` `roles` `service-accounts` `api-keys` `application-keys`

### Smart Operations
`health` `deploy`

### Analytics
`anomalies` `correlation` `impact-analysis`

### Automation
`auto-remediate` `change-management` `capacity-scale`

### ML & Predictions
`ml-insights` `predictions` `recommendations`

### Utilities
`audit-logs` `spans` `service-map` `version`

---

## Quick Automation Examples

### Health Check Script
```bash
for service in $(dd apm services --json | jq -r '.services[].name'); do
  dd health check --service "$service"
done
```

### Cost Alert
```bash
cost=$(dd cost estimate --from 1m --json | jq -r '.total')
if [ $cost -gt 10000 ]; then
  echo "Alert: Monthly cost exceeds $10,000"
fi
```

### SLO Budget Alert
```bash
budget=$(dd error-budgets check --slo-id abc123 --json | jq -r '.remaining')
if [ $budget -lt 10 ]; then
  echo "Alert: SLO budget at ${budget}%"
fi
```

---

## Help Commands

```bash
dd --help                   # All commands
datadog-cli apm --help               # Command help
datadog-cli apm services --help      # Action help
```

---

## Performance

- **Startup**: 8ms (25x faster than Python)
- **Memory**: ~25MB
- **Binary**: 18MB (single file, no dependencies)

---

## Documentation

- **This Cheat Sheet**: Ultra-compact reference
- **QUICK-REFERENCE.md**: One-page detailed reference
- **EXAMPLES.md**: 10 real-world scenarios
- **FAQ.md**: 35+ common questions
- **QUICKSTART.md**: 5-minute tutorial
- **README.md**: Complete user guide

---

## Common Issues

### Authentication Failed
```bash
echo $DD_API_KEY    # Check if set
echo $DD_APP_KEY    # Check if set
```

### Command Not Found
```bash
which dd            # Check if in PATH
export PATH="/usr/local/bin:$PATH"
```

### Slow Performance
- Use narrower time ranges: `--from 1h` instead of `--from 24h`
- Add service filters: `--service api`
- Use specific queries instead of broad searches

---

**Datadog CLI v0.1.0** | **54 Commands** | **8ms Startup**
**github.com/your-org/datadog-cli**
