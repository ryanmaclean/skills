---
description: "Check service health with multi-signal analysis across APM, logs, and error rates"
argument-hint: "[SERVICE] [--from TIMERANGE]"
---

# Datadog Health Check

Check the health of a service by analyzing multiple signals: APM metrics, error logs, and error rates.

## Usage

Use the Datadog CLI to perform health checks:

```bash
# Auto-detect service from git context
dd health

# Check specific service
dd health <service-name>

# Check with custom time range
dd health <service-name> --from 6h
```

## What it analyzes

- **APM Traces**: Request rates, error rates, latency percentiles (p50, p95, p99)
- **Error Logs**: Recent error and critical log entries
- **Service Status**: Overall service health based on multiple signals

## Output

The health check provides:
- Overall health score (healthy/degraded/unhealthy)
- Request throughput and error rates
- Latency percentiles
- Recent errors with timestamps
- Recommendations for investigation

## Common Use Cases

1. **Pre-deployment validation**: Check service health before deploying changes
2. **Incident investigation**: Quickly assess service status during incidents
3. **Post-deployment monitoring**: Verify service health after deployment
4. **Routine monitoring**: Regular health checks during development

## Why Use the CLI?

The `dd health` command provides unique capabilities not available in the Datadog web UI:

- **Multi-signal analysis** - Combines APM, logs, and metrics in a single view
- **Context-aware** - Auto-detects service from git repository
- **Fast execution** - Results in 3ms vs loading multiple dashboards
- **Pre-deployment** - Check health before pushing code (unique CLI capability)
- **Scriptable** - Integrate health checks into CI/CD pipelines
- **Offline mode** - Cached results when working without internet

## Example Prompts

> "Check the health of the api service"
> "Is the payment-service healthy?"
> "Show me the health status for checkout service in the last 3 hours"
> "Health check on user-service"

## Environment Variables

The CLI requires these environment variables:
- `DD_API_KEY`: Datadog API key
- `DD_APP_KEY`: Datadog application key
- `DD_SITE`: Datadog site (default: datadoghq.com)

## Notes

- If no service is specified, the CLI attempts to detect it from git repository context
- Time ranges support: 1m, 5m, 15m, 30m, 1h, 2h, 4h, 6h, 12h, 24h, 7d, 30d
- Health scoring is based on error rates, latency trends, and log severity

## Learn More

- [APM Service Overview](https://docs.datadoghq.com/tracing/services/)
- [Service Health Metrics](https://docs.datadoghq.com/tracing/metrics/)
- [Log Management](https://docs.datadoghq.com/logs/)
