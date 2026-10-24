---
description: "Query Datadog metrics, timeseries data, and system metrics"
argument-hint: "--query QUERY [--from TIMERANGE] [--aggregation AGG]"
---

# Datadog Metrics Query

Query Datadog metrics and timeseries data for monitoring, alerting, and analysis.

## Usage

```bash
# Query metric
dd metrics --query "avg:system.cpu.user{*}"

# Query with time range
dd metrics --query "avg:system.cpu.user{host:web-1}" --from 2h

# Query with aggregation
dd metrics --query "sum:http.requests{service:api}" --aggregation sum

# Complex metric queries
dd metrics --query "avg:trace.servlet.request.duration{service:checkout,env:production} by {resource_name}"
```

## Metric Query Syntax

Datadog metric queries follow this format:
```
<aggregation>:<metric_name>{<tags>} [by {<tag_keys>}]
```

### Aggregations
- `avg`: Average value
- `sum`: Sum of values
- `min`: Minimum value
- `max`: Maximum value
- `count`: Count of data points

### Common Metrics

**System Metrics:**
- `system.cpu.user`: CPU user time
- `system.cpu.system`: CPU system time
- `system.mem.used`: Memory used
- `system.disk.used`: Disk space used
- `system.net.bytes_rcvd`: Network bytes received
- `system.net.bytes_sent`: Network bytes sent

**APM Metrics:**
- `trace.servlet.request.duration`: Request latency
- `trace.servlet.request.hits`: Request count
- `trace.servlet.request.errors`: Error count

**Custom Metrics:**
- Your application-specific metrics

## Tag Filtering

```bash
# Single tag
dd metrics --query "avg:system.cpu.user{host:web-1}"

# Multiple tags (AND)
dd metrics --query "avg:http.requests{service:api,env:production}"

# Wildcard
dd metrics --query "avg:system.cpu.user{host:web-*}"

# Group by tags
dd metrics --query "avg:http.requests{service:api} by {region}"
```

## Output

Metric queries provide:
- **Metric Name**: Full metric name
- **Time Range**: Queried time period
- **Aggregation**: Aggregation method used
- **Series**: Timeseries data points with:
  - Timestamps
  - Values
  - Tags/groups

## Common Use Cases

1. **Performance monitoring**: Track CPU, memory, disk usage
2. **Application metrics**: Monitor request rates, latency, errors
3. **Capacity planning**: Analyze resource utilization trends
4. **Alert validation**: Verify metrics triggering alerts
5. **Comparison**: Compare metrics across services, hosts, regions

## Why Use the CLI?

- **Fast queries** - Query metrics in 3ms vs loading dashboards
- **Powerful syntax** - Full Datadog metrics query language
- **Scriptable** - Automate metric collection and alerting
- **CI/CD integration** - Validate metrics in deployment pipelines
- **Offline analysis** - Export metrics for local analysis
- **Natural language** - Query using conversational prompts

## Example Prompts

> "Show me CPU usage for web servers"
> "What's the request rate for api service?"
> "Query memory usage for host web-1 in the last 4 hours"
> "Get error count for checkout service"
> "Show me p99 latency for payment-service"

## Time Range Examples

- `1m`, `5m`, `15m`, `30m` - Minutes
- `1h`, `2h`, `4h`, `6h`, `12h` - Hours
- `24h`, `7d`, `30d` - Days

## Environment Variables

Required:
- `DD_API_KEY`: Datadog API key
- `DD_APP_KEY`: Datadog application key
- `DD_SITE`: Datadog site (default: datadoghq.com)

## Advanced Queries

```bash
# Calculate rate
dd metrics "sum:http.requests{service:api}.as_rate()"

# Moving average
dd metrics "avg:system.cpu.user{*}.rollup(avg, 60)"

# Arithmetic operations
dd metrics "avg:system.mem.used{*} / avg:system.mem.total{*} * 100"

# Multiple series
dd metrics "avg:system.cpu.user{env:prod}, avg:system.cpu.user{env:staging}"
```

## Aggregation Options

- `avg`: Average over time period
- `sum`: Sum over time period
- `min`: Minimum value in time period
- `max`: Maximum value in time period
- `count`: Number of data points
- `last`: Most recent value
- `percentile(n)`: Nth percentile (e.g., percentile(95))

## Output Format

JSON-formatted timeseries with:
- `metric`: Metric name
- `query`: Full query string
- `from_timestamp`: Start time (Unix timestamp)
- `to_timestamp`: End time (Unix timestamp)
- `series`: Array of data series with:
  - `tag_set`: Tags for this series
  - `pointlist`: Array of [timestamp, value] pairs

## Query Tips

1. **Start Simple**: Begin with basic queries, add complexity as needed
2. **Use Wildcards**: `{host:web-*}` to query multiple hosts
3. **Group By**: Use `by {tag}` to split series by tag values
4. **Time Alignment**: Longer time ranges use coarser rollup intervals
5. **Rate Calculations**: Use `.as_rate()` for counter metrics

## Notes

- Default time range is 1 hour
- Metric values may be aggregated based on time range
- Tag filtering is case-sensitive
- Wildcard `*` matches all values for a tag
- Use quotes around queries with special characters

## Learn More

- [Metrics Documentation](https://docs.datadoghq.com/metrics/)
- [Metrics Query Language](https://docs.datadoghq.com/dashboards/functions/)
- [Custom Metrics](https://docs.datadoghq.com/metrics/custom_metrics/)
- [Metrics Types](https://docs.datadoghq.com/metrics/types/)
