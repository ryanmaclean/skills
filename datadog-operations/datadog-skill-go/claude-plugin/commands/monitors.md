---
description: "Query and manage Datadog monitors - check alert status, mute monitors, update thresholds"
argument-hint: "[MONITOR-NAME] [--status STATUS] [--mute] [--unmute]"
---

# Datadog Monitor Management

Query, manage, and configure Datadog monitors to control alerting, manage alert fatigue, and ensure critical notifications reach the right teams.

## What are Monitors?

Monitors are Datadog's alerting system:
- **Metric monitors** - Alert on threshold breaches
- **APM monitors** - Alert on trace metrics (latency, errors)
- **Log monitors** - Alert on log patterns
- **Composite monitors** - Combine multiple conditions
- **Anomaly monitors** - ML-powered anomaly detection

**Official Documentation**: https://docs.datadoghq.com/monitors/

## Usage

```bash
# List all monitors
dd monitors list

# List monitors filtered by status
dd monitors list --status alert
dd monitors list --status ok
dd monitors list --status no-data

# Filter by service
dd monitors list --service api-service

# Mute a monitor
dd monitors mute --name "api-latency-high" --duration 2h

# Unmute a monitor
dd monitors unmute --name "api-latency-high"

# Create a new monitor
dd monitors create --name "High Error Rate" --query "avg(last_5m):..." --message "Alert message"

# Delete a monitor
dd monitors delete --id 12345
```

## Monitor Types

**Metric Monitors**:
- Threshold alerts (e.g., CPU > 80%)
- Change alerts (50% increase)
- Anomaly detection
- Outlier detection

**Service Monitors**:
- APM error rates
- APM latency (p50, p95, p99)
- Service availability
- Trace volume

**Integration Monitors**:
- Host up/down
- Container health
- Database availability
- Cloud service status

**Log Monitors**:
- Error pattern detection
- Security event alerts
- Compliance violations
- Application errors

## Use Cases

### 1. Check Current Alerts
```bash
dd monitors --status alert
```

See all actively firing monitors across your infrastructure.

### 2. Mute During Maintenance
```bash
dd monitors api-service --mute --duration 1h --reason "Deploying v2.0"
```

Prevent alert fatigue during planned changes.

### 3. Monitor Health Check
```bash
dd monitors --from 7d
```

Review monitor effectiveness and detect flapping alerts.

### 4. Quick Alert Investigation
```bash
dd monitors database-latency-high
```

Get monitor details, threshold, and recent history instantly.

## Why Use the CLI?

- **Instant access** - Check alert status in 3ms vs dashboard loading
- **Bulk operations** - Mute multiple monitors during deployments
- **Automation** - Integrate monitor management into CI/CD
- **Context-aware** - Auto-detects service from current directory
- **Maintenance windows** - Quick muting before deployments
- **Scriptable** - Automate alert management workflows

## Example Prompts

> "Show me all monitors in alert state"
> "Mute the api-latency monitor for 2 hours"
> "What monitors are currently firing?"
> "Check the status of database monitors"
> "List all no-data monitors"

## Learn More

- [Monitor Types](https://docs.datadoghq.com/monitors/types/)
- [Managing Monitors](https://docs.datadoghq.com/monitors/manage/)
- [Downtiming](https://docs.datadoghq.com/monitors/downtimes/)
