---
description: "Execute any Datadog CLI command - comprehensive access to all 22 CLI operations"
argument-hint: "COMMAND [ARGS...]"
---

# Datadog CLI - Complete Access

Execute any Datadog CLI command directly. This skill provides access to all 22 CLI commands.

## Available Commands

### Core Operations
- `health` - Check service health with multi-signal analysis
- `deploy` - Validate deployment safety
- `apm` - Query APM traces
- `logs` - Search and retrieve logs
- `metrics` - Query metrics and timeseries data

### Incident & Monitor Management
- `incidents` - Manage incidents (list, create, update, close)
- `monitors` - Manage monitors (list, create, update, delete, mute)

### Observability Features
- `dashboards` - Manage dashboards (list, get, create, update, delete)
- `synthetics` - Manage synthetic tests (list, get, create, update, delete, run)
- `rum` - Query Real User Monitoring data
- `network` - Query Network Performance Monitoring data
- `database` - Query Database Monitoring data
- `security` - Query Security Monitoring signals
- `watchdog` - Get Watchdog anomaly detection alerts

### Advanced Features
- `workflows` - Execute Datadog workflows
- `slos` - Manage Service Level Objectives
- `services` - Query service catalog
- `cost` - Track and analyze observability costs
- `llm` - Query LLM Observability metrics
- `cicd` - Query CI/CD Visibility data
- `context` - Detect execution context (git repo, service name, etc.)
- `version` - Show CLI version and build info

## Usage

```bash
# Use any Datadog CLI command
dd <command> [args...]

# Examples
dd health api-service
dd apm checkout-service --status error --from 2h
dd logs "database timeout" --service payment-service
dd incidents list --status active
dd monitors list --tags "team:platform"
```

## Common Patterns

### Health Checks
```bash
dd health                           # Auto-detect service
dd health <service>                 # Specific service
dd health <service> --from 6h       # Custom time range
```

### Deploy Safety
```bash
dd deploy                           # Auto-detect service
dd deploy <service>                 # Specific service
dd deploy <service> --environment production
```

### APM Analysis
```bash
dd apm <service>                    # All traces
dd apm <service> --status error     # Error traces only
dd apm <service> --resource "GET /api/users"
```

### Log Search
```bash
dd logs                             # Recent logs
dd logs "error message"             # Search query
dd logs --service <service> --status error
dd logs --from 24h --limit 500
```

### Incident Management
```bash
dd incidents list                   # List incidents
dd incidents list --status active --service api
dd incidents create --title "..." --severity SEV-2
dd incidents update <id> --status stable
dd incidents close <id>
```

### Metrics Query
```bash
dd metrics "avg:system.cpu.user{*}"
dd metrics "avg:trace.servlet.request.duration{service:api}"
dd metrics "sum:http.requests{env:prod} by {service}"
```

## Global Flags

All commands support:
- `--json` - Output in JSON format
- `--verbose` - Enable verbose logging
- `--debug` - Enable debug logging
- `--help` - Show command help

## Environment Variables

Required for all commands:
- `DD_API_KEY`: Datadog API key
- `DD_APP_KEY`: Datadog application key
- `DD_SITE`: Datadog site (default: datadoghq.com)

Optional:
- `DD_CLI_PATH`: Path to Datadog CLI binary (auto-detected)

## Command Help

Get help for any command:
```bash
dd help
dd <command> --help
dd apm --help
```

## Why Use the CLI?

The Datadog CLI provides unique advantages over the web UI and API:

- **Lightning fast** - 3ms startup time (Go binary) vs browser loading
- **Context-aware** - Auto-detects service from git repository
- **Offline capability** - Cached results and configurations
- **Scriptable** - Automate observability workflows
- **CI/CD integration** - Perfect for deployment gates and validation
- **Multi-signal analysis** - health and deploy commands combine multiple signals
- **Pre-deployment validation** - Check before pushing (unique CLI feature)
- **Natural language** - Use conversational prompts with AI
- **No browser needed** - Fast terminal access during incidents

## Example Prompts

> "Use Datadog CLI to check api-service health"
> "Run dd apm checkout-service --status error"
> "Execute dd logs command to find database errors"
> "Use the Datadog CLI to list active incidents"
> "Query metrics for CPU usage on web servers"

## Output Format

All commands output JSON by default, which includes:
- Command metadata (timestamp, execution time, CLI version)
- Result data specific to each command
- Error messages if applicable

## Error Handling

The CLI provides clear error messages for:
- Authentication failures (invalid API/App keys)
- Invalid arguments or flags
- API rate limits
- Network connectivity issues
- Service not found errors

## Performance

- **Startup time**: ~3ms (Go binary)
- **API latency**: Depends on Datadog API response times
- **Timeout**: 30 seconds default for API calls
- **Retry**: Automatic retry with exponential backoff

## Notes

- All commands require valid Datadog credentials
- Service names are case-sensitive and must match Datadog tags
- Time ranges support various formats: 1m, 1h, 24h, 7d, 30d
- Results are limited by Datadog API constraints
- Use `--json` for programmatic parsing
- CLI version: 1.0.0 (20/20 iterations complete)

## Full Command Reference

For detailed help on each command, use:
```bash
dd help
dd <command> --help
```

Or refer to the individual skill files for:
- `/datadog health` - Health check documentation
- `/datadog deploy` - Deploy safety documentation
- `/datadog apm` - APM trace documentation
- `/datadog logs` - Log search documentation
- `/datadog incidents` - Incident management documentation
- `/datadog metrics` - Metrics query documentation

## Learn More

**Official Documentation:**
- [Datadog API](https://docs.datadoghq.com/api/latest/)
- [APM](https://docs.datadoghq.com/tracing/)
- [Log Management](https://docs.datadoghq.com/logs/)
- [Infrastructure Monitoring](https://docs.datadoghq.com/infrastructure/)
- [LLM Observability](https://docs.datadoghq.com/llm_observability/)

**CLI Resources:**
- [Unified Service Tagging](https://docs.datadoghq.com/getting_started/tagging/unified_service_tagging/)
- [Service Catalog](https://docs.datadoghq.com/service_catalog/)
- [Deployment Tracking](https://docs.datadoghq.com/tracing/deployment_tracking/)
