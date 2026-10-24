---
description: "Manage Datadog incidents - list, create, update, and close incidents"
argument-hint: "ACTION [OPTIONS]"
---

# Datadog Incident Management

Manage Datadog incidents through the command line - list active incidents, create new incidents, update status, and close resolved incidents.

## Usage

```bash
# List all active incidents
dd incidents list

# List incidents with filters
dd incidents list --status stable
dd incidents list --service api-service
dd incidents list --service payment-service --status resolved

# Create new incident
dd incidents create --title "High error rate in payment API" --severity SEV-2

# Create incident with service context
dd incidents create --title "Database connection pool exhausted" --service db-service --severity SEV-1

# Update incident status
dd incidents update --id <incident-id> --status stable

# Close incident
dd incidents close <incident-id>
```

## Incident Actions

- `list`: List incidents with optional filters
- `create`: Create a new incident
- `update`: Update incident status or details
- `close`: Close a resolved incident

## Severity Levels

- `SEV-1`: Critical - Service down, major functionality impaired
- `SEV-2`: High - Significant impact, workaround available
- `SEV-3`: Medium - Minor impact, non-critical functionality affected
- `SEV-4`: Low - Minimal impact, cosmetic issues
- `SEV-5`: Informational - No customer impact

## Incident Statuses

- `active`: Incident is actively being worked on
- `stable`: Incident is stabilized but not fully resolved
- `resolved`: Incident has been fully resolved
- `completed`: Incident investigation and remediation complete

## Output

Incident commands provide:
- **List**: Incident ID, title, severity, status, service, created time
- **Create**: New incident ID and details
- **Update**: Updated incident information
- **Close**: Confirmation of closure

## Common Use Cases

1. **Incident response**: Create incidents when issues are detected
2. **Status tracking**: Monitor active and stable incidents
3. **Service filtering**: View incidents for specific services
4. **Workflow automation**: Integrate with alerting and paging systems
5. **Post-mortem prep**: List resolved incidents for analysis

## Why Use the CLI?

- **Fast incident creation** - Create incidents in 3ms from terminal
- **ChatOps integration** - Trigger from Slack/Teams commands
- **Context-aware** - Auto-detects service from git repository
- **Automation** - Integrate with alerting and monitoring systems
- **Scriptable workflows** - Automate incident lifecycle management
- **No UI dependency** - Manage incidents when dashboard is unavailable

## Example Prompts

> "What incidents are currently active for the api service?"
> "Create a SEV-2 incident for high error rate in payment API"
> "List all resolved incidents from the last week"
> "Update incident 12345 to stable status"
> "Close incident 67890"
> "Show me all SEV-1 incidents"

## Creating Incidents

Required fields:
- `--title`: Incident title (clear, descriptive)
- `--severity`: SEV-1 through SEV-5

Optional fields:
- `--service`: Associated service name
- `--customer-impact`: Description of customer impact
- `--fields`: Additional custom fields (JSON)

## Environment Variables

Required:
- `DD_API_KEY`: Datadog API key
- `DD_APP_KEY`: Datadog application key
- `DD_SITE`: Datadog site (default: datadoghq.com)

## Incident Lifecycle

```
active -> stable -> resolved -> completed
   ↓         ↓         ↓
   └─────────┴─────────┘
        (reopen)
```

## Integration Tips

1. **Alerting**: Create incidents from monitor alerts automatically
2. **ChatOps**: Integrate with Slack/Teams for incident commands
3. **CI/CD**: Create incidents for deployment failures
4. **Dashboards**: Link incidents to relevant dashboards

## Output Format

JSON-formatted incident data with:
- `id`: Unique incident identifier
- `title`: Incident title
- `severity`: SEV-1 through SEV-5
- `status`: current status
- `service`: Associated service(s)
- `created`: Creation timestamp
- `customer_impact`: Impact description
- `commander`: Incident commander user

## Advanced Usage

```bash
# List SEV-1 and SEV-2 incidents
dd incidents list --severity "SEV-1,SEV-2"

# Create with customer impact
dd incidents create \
  --title "API Gateway Latency Spike" \
  --severity SEV-2 \
  --service api-gateway \
  --customer-impact "15% of API requests experiencing 2-5s delays"

# Update with custom fields
dd incidents update --id 12345 --fields '{"root_cause": "database connection leak"}'
```

## Notes

- Incident IDs are returned when creating new incidents
- Use incident IDs for update and close operations
- Service names must match Datadog service tags
- Severity and status are case-insensitive
- Closed incidents remain in history for reporting

## Learn More

- [Incident Management](https://docs.datadoghq.com/service_management/incident_management/)
- [Incident Timeline](https://docs.datadoghq.com/service_management/incident_management/incident_details/)
- [Incident Workflows](https://docs.datadoghq.com/service_management/incident_management/incident_settings/)
