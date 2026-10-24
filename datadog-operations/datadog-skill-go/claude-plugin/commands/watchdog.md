---
description: "Query Watchdog for automated anomaly detection and intelligent alerts"
argument-hint: "[--from TIMERANGE] [--service SERVICE]"
---

# Datadog Watchdog

Query Watchdog's AI-powered anomaly detection to identify unusual behavior, performance issues, and potential incidents automatically.

## What is Watchdog?

Watchdog uses machine learning to automatically detect anomalies:
- **Performance anomalies** - Latency spikes, error rate increases
- **Infrastructure issues** - Resource saturation, failures
- **Application problems** - Deployment impacts, code issues
- **Security threats** - Unusual access patterns

**Official Documentation**: https://www.datadoghq.com/product/platform/watchdog/

## Usage

```bash
# Query all Watchdog alerts
dd watchdog

# Filter by service
dd watchdog --service api-service

# Filter by time range
dd watchdog --from 24h
```

## Why Use the CLI?

- **Instant insights** - See anomalies without manual analysis
- **Proactive monitoring** - Catch issues before users report
- **Root cause analysis** - AI-suggested correlations
- **Deployment validation** - Auto-detect deployment impacts

## Example Prompts

> "What anomalies has Watchdog detected?"
> "Show me Watchdog alerts for api-service"
> "Any unusual behavior in the last hour?"

## Learn More

- [Watchdog Product Page](https://www.datadoghq.com/product/platform/watchdog/)