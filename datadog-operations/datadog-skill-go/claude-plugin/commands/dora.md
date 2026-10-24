---
description: "Query DORA Metrics for DevOps performance measurement and Elite team benchmarking"
argument-hint: "[--service SERVICE] [--env ENVIRONMENT] [--metric deployments|failures|metrics|all]"
---

# Datadog DORA Metrics

Query DORA (DevOps Research and Assessment) Metrics to measure DevOps performance and compare against Elite/High/Medium/Low industry benchmarks.

## What are DORA Metrics?

DORA Metrics are the four key metrics that indicate DevOps performance:
- **Deployment Frequency** - How often you deploy to production
- **Lead Time for Changes** - Time from commit to production
- **Change Failure Rate** - Percentage of deployments causing failures
- **Time to Restore (MTTR)** - How quickly you recover from incidents

**Official Documentation**: https://docs.datadoghq.com/api/latest/dora-metrics/

## Performance Tiers

**Elite Teams**:
- Multiple deploys per day
- <1 hour lead time
- <15% failure rate
- <1 hour recovery time

**High Performers**:
- Weekly deployments
- <1 day lead time
- <30% failure rate
- <1 day recovery time

**Medium Performers**:
- Monthly deployments
- <1 month lead time
- <45% failure rate
- <1 week recovery time

## Usage

```bash
# Query all DORA metrics (auto-detect service)
dd dora

# Query specific service and environment
dd dora --service my-service --env production

# Filter by time range and team
dd dora --duration 30d --team platform

# Query only deployments
dd dora --metric deployments

# Query only failures/incidents
dd dora --metric failures

# Get JSON output
dd dora --metric all --json
```

## Key Metrics

**Deployment Frequency**:
- Deployments per day
- Trend over time
- Service-level breakdown

**Lead Time**:
- Hours from commit to production
- Median and p95 values
- Bottleneck identification

**Change Failure Rate**:
- Percentage of failed deployments
- Incident correlation
- Rollback frequency

**Time to Restore (MTTR)**:
- Mean time to recovery
- Incident duration analysis
- Resolution velocity

## Use Cases

### 1. Measure DevOps Performance
```bash
dd dora --service my-app --duration 90d
```

Track your team's performance against DORA research benchmarks.

### 2. Compare Teams
```bash
dd dora --team platform --duration 30d
dd dora --team frontend --duration 30d
```

Benchmark multiple teams to identify improvement opportunities.

### 3. Track Improvement Over Time
```bash
dd dora --service my-service --duration 180d
```

Visualize progress toward Elite performance tier.

### 4. Investigate High Failure Rate
```bash
dd dora --metric failures --service my-service
```

Identify patterns in deployment failures and incidents.

### 5. Production Readiness Check
```bash
dd dora --service my-service --env production
```

Verify service stability before major releases.

## Why Use the CLI?

- **Benchmark performance** - Compare against Elite/High/Medium/Low tiers
- **Track trends** - Monitor improvement over time
- **Team comparisons** - Identify high-performing patterns
- **Deployment confidence** - Check stability before releases
- **Incident analysis** - Understand failure patterns

## Example Prompts

> "What are my DORA metrics for the last 30 days?"
> "Am I an Elite performer?"
> "Show me deployment frequency for production"
> "Check change failure rate for my-service"
> "How quickly do we recover from incidents?"

## Integration

DORA Metrics CLI integrates with:
- **CI/CD pipelines** - Track deployment events automatically
- **Incident management** - Correlate failures with incidents
- **Service Catalog** - Service-level DORA tracking
- **Dashboards** - Export metrics for visualization

## Learn More

- [DORA Metrics Product Page](https://www.datadoghq.com/product/software-delivery-insights/)
- [DORA Research](https://dora.dev/)
- [State of DevOps Report](https://cloud.google.com/devops/state-of-devops)
