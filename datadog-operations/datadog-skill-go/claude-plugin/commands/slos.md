---
description: "Query Service Level Objectives to check error budgets and SLO compliance"
argument-hint: "[SLO-NAME] [--from TIMERANGE]"
---

# Datadog Service Level Objectives (SLOs)

Query and monitor Service Level Objectives to track service reliability, error budgets, and SLO compliance.

## What are SLOs?

Service Level Objectives define target reliability levels:
- **Availability SLOs** - Uptime targets (e.g., 99.9%)
- **Latency SLOs** - Response time targets (e.g., p95 < 200ms)
- **Error rate SLOs** - Error targets (e.g., < 0.1%)
- **Error budgets** - Remaining allowed errors before SLO breach

**Official Documentation**: https://www.datadoghq.com/product/service-level-objectives/

## Usage

```bash
# List all SLOs
dd slos

# Query specific SLO
dd slos api-availability

# Check error budget
dd slos api-latency --from 30d

# Filter by time range
dd slos --from 7d
```

## SLO Metrics

**Status**:
- Current compliance percentage
- Target vs actual
- Trend (improving/degrading)
- Time until breach

**Error Budget**:
- Remaining budget
- Burn rate
- Projected exhaustion date
- Historical consumption

**Alerts**:
- SLO breach warnings
- Error budget depletion alerts
- Recovery notifications

## Use Cases

### 1. Check SLO Status
```bash
dd slos --from 7d
```

Shows all SLOs, compliance %, and error budget status.

### 2. Monitor Error Budget
```bash
dd slos api-availability --from 30d
```

Tracks error budget consumption and burn rate.

### 3. Pre-Deployment Check
```bash
dd slos --from 24h
```

Verify SLO health before deploying changes.

## Why Use the CLI?

- **Fast checks** - 3ms startup vs dashboard loading
- **Pre-deploy validation** - Check SLOs before deployment
- **Automation** - Integrate with CI/CD pipelines
- **Error budget tracking** - Quick burn rate checks

## Example Prompts

> "Check all SLO statuses"
> "What's the error budget for api-availability SLO?"
> "Are we on track to meet our latency SLO?"
> "Show me SLO compliance for the last 30 days"

## Learn More

- [SLO Product Page](https://www.datadoghq.com/product/service-level-objectives/)
- [SLO Best Practices](https://docs.datadoghq.com/service_management/service_level_objectives/)