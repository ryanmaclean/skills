---
description: "Search and retrieve logs with filtering by service, status, time range, and custom queries"
argument-hint: "[--query QUERY] [--service SERVICE] [--duration TIMERANGE] [--status LEVEL] [--limit N]"
---

# Datadog Log Search

Search and retrieve logs from Datadog with flexible filtering options for debugging, investigation, and monitoring.

## Usage

```bash
# Search all logs (last 24h by default)
dd logs

# Search with query string
dd logs --query "error database timeout"

# Filter by service
dd logs --service api-service

# Filter by log level and duration
dd logs --status error --duration 1h

# Combine filters
dd logs --query "database connection" --service payment-service --status error --duration 24h

# Limit results
dd logs --service user-service --limit 50 --duration 6h
```

## Log Levels (Status)

- `error`: Error-level logs
- `warn`: Warning-level logs
- `info`: Informational logs
- `debug`: Debug-level logs

## Query Syntax

Supports Datadog log search query syntax:
- **Simple text**: `"database timeout"`
- **Field filters**: `env:production service:api`
- **Wildcards**: `error*`, `*timeout`
- **Boolean**: `error AND database`, `timeout OR connection`
- **Negation**: `-debug`, `NOT info`

## Output

Log search provides:
- **Log Count**: Number of matching logs
- **Time Range**: Searched time period
- **Log Entries**: Individual log messages with:
  - Timestamp
  - Service name
  - Log level (status)
  - Message content
  - Additional attributes (tags, host, etc.)

## Common Use Cases

1. **Error investigation**: Find error logs during incidents
2. **Deployment validation**: Check logs after deployment
3. **Debugging**: Search for specific error messages or patterns
4. **Audit trails**: Review user actions or system events
5. **Performance analysis**: Find slow query logs or timeouts

## Why Use the CLI?

- **Fast searches** - Query logs in 3ms without browser loading
- **Powerful syntax** - Full Datadog query language support
- **Scriptable** - Automate log analysis and alerting
- **Context-aware** - Auto-detects service from git repository
- **Tail mode** - Real-time log streaming (coming soon)
- **JSON output** - Machine-parseable results for automation

## Example Prompts

> "Find all error logs containing 'database timeout' in the last 24 hours"
> "Show me recent error logs for payment-service"
> "Search logs for 'connection refused' in api-service"
> "Get the last 100 logs from checkout-service"
> "Find warning logs about memory usage"

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

1. **Start Broad**: Begin with service/time filters, then add query strings
2. **Use Log Levels**: Filter by status to reduce noise
3. **Combine Filters**: Stack multiple filters for precision
4. **Limit Results**: Use `--limit` for large result sets

## Output Format

JSON-formatted log entries with:
- `timestamp`: ISO 8601 timestamp
- `service`: Service name
- `status`: Log level
- `message`: Log message content
- `attributes`: Additional log attributes (tags, context)
- `host`: Host/container name

## Performance Notes

- Default limit: 100 logs
- Maximum limit: 1000 logs
- Default time range: 1 hour
- Logs are returned in reverse chronological order (newest first)

## Advanced Queries

```bash
# Multiple services
dd logs --service "api-service OR payment-service"

# Specific environment
dd logs "env:production error"

# Exclude debug logs
dd logs "service:api-service -debug"

# Time range with status
dd logs --status error --from 6h
```

## Notes

- Query syntax follows Datadog log search conventions
- Log ingestion may have slight delays (typically < 30 seconds)
- Use quotes for multi-word queries
- Service names are case-sensitive

## Learn More

- [Log Management](https://docs.datadoghq.com/logs/)
- [Log Search Syntax](https://docs.datadoghq.com/logs/explorer/search_syntax/)
- [Log Processing](https://docs.datadoghq.com/logs/log_configuration/processing/)
- [Log Patterns](https://docs.datadoghq.com/logs/explorer/patterns/)
