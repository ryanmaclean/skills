---
description: "Query APM traces and analyze service performance, errors, and latency"
argument-hint: "[--service SERVICE] [--duration TIMERANGE] [--status STATUS] [--limit N]"
---

# Datadog APM Trace Analytics

Query and analyze APM (Application Performance Monitoring) traces to investigate performance issues, errors, and service behavior.

## Usage

```bash
# Query traces (auto-detects service)
dd apm

# Query with time range
dd apm --duration 2h

# Query specific service
dd apm --service api-service --duration 1h

# Filter by status
dd apm --status error --duration 6h
dd apm --status ok

# Limit results
dd apm --limit 10 --duration 24h
```

## Trace Statuses

- `error`: Failed requests (HTTP 5xx, exceptions, errors)
- `ok`: Successful requests (HTTP 2xx, 3xx)
- `all`: All traces regardless of status

## Output

APM trace query provides:
- **Trace Count**: Number of matching traces
- **Request Rate**: Requests per second
- **Error Rate**: Percentage of failed requests
- **Latency Stats**: p50, p75, p95, p99 percentiles
- **Top Resources**: Most frequently called endpoints
- **Top Errors**: Most common error messages
- **Slowest Traces**: Traces with highest latency

## Common Use Cases

1. **Performance investigation**: Identify slow endpoints and latency issues
2. **Error spike analysis**: Investigate sudden increases in error rates
3. **Latency troubleshooting**: Find performance bottlenecks
4. **Resource profiling**: Analyze specific endpoint performance
5. **Dependency analysis**: Understand service call patterns

## Why Use the CLI?

- **Fast access** - Query APM traces in 3ms vs loading APM dashboard
- **Scriptable** - Automate trace analysis in CI/CD and monitoring scripts
- **Context-aware** - Auto-detects service from current git repository
- **Filtering power** - Precise queries with status, resource, and time filters
- **Offline capability** - Works with cached trace metadata
- **Natural language** - Query using conversational prompts

## Example Prompts

> "Show me error traces for checkout service in the last hour"
> "What's the p99 latency for api-service?"
> "Find slow traces in payment-service from the last 2 hours"
> "Analyze APM traces for user-service with errors"
> "Show me the last 24 hours of APM data"

## Time Range Examples

- `1m`, `5m`, `15m`, `30m` - Minutes
- `1h`, `2h`, `4h`, `6h`, `12h` - Hours
- `24h`, `7d`, `30d` - Days

## Environment Variables

Required:
- `DD_API_KEY`: Datadog API key
- `DD_APP_KEY`: Datadog application key
- `DD_SITE`: Datadog site (default: datadoghq.com)

## Filtering Tips

1. **By Status**: Use `--status error` to focus on failures
2. **By Resource**: Use `--resource` to analyze specific endpoints
3. **By Time**: Use `--duration` to look at recent or historical data
4. **Combine Filters**: Stack filters for precise queries

## Output Format

Results include:
- Service name and time range
- Aggregate statistics (counts, rates, percentiles)
- Top-N lists (resources, errors, traces)
- Individual trace details when applicable

## Notes

- Default time range is 1 hour
- Trace data may have slight delays (typically < 1 minute)
- Resource names match Datadog APM resource conventions
- Supports all APM-instrumented services

## Recent Updates

**✅ FIXED (Jan 22, 2026 - Iteration 25):**
APM command now fully functional. Previous API validation errors have been resolved by fixing the request format to match Datadog's API v2 specifications. All trace analytics features are now working.

## Learn More

- [APM Documentation](https://docs.datadoghq.com/tracing/)
- [Trace Search](https://docs.datadoghq.com/tracing/trace_explorer/)
- [Service Map](https://docs.datadoghq.com/tracing/services/service_map/)
- [APM Metrics](https://docs.datadoghq.com/tracing/metrics/)
