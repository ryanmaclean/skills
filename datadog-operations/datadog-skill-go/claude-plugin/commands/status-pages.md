---
description: "Manage Datadog Status Pages for customer communication"
argument-hint: "--action <action> [--page-id ID] [--component-id ID] [options]"
---

# Datadog Status Pages Management

Manage Status Pages for customer and stakeholder communication about service availability and incidents. Create pages, add components, and track service degradations.

## What are Status Pages?

Status Pages are part of Datadog's Service Management suite, enabling:
- **Customer Communication** - Public status pages for service availability
- **Component Tracking** - Monitor individual service components
- **Incident Reporting** - Communicate degradations and outages
- **Subscriber Notifications** - Alert stakeholders of changes
- **Transparency** - Build trust through proactive communication

**Official Documentation**: https://docs.datadoghq.com/service_management/status_pages/

## Usage

### Page Management

```bash
# List all status pages
dd status-pages --action list-pages

# Create new status page
dd status-pages --action create-page --name "API Status" --subdomain api-status

# Get page details
dd status-pages --action get-page --page-id abc123

# Update page
dd status-pages --action update-page --page-id abc123 --name "Updated Name"

# Delete page
dd status-pages --action delete-page --page-id abc123
```

### Component Management

```bash
# List components on a page
dd status-pages --action list-components --page-id abc123

# Add component to page
dd status-pages --action create-component --page-id abc123 --name "Authentication API"

# Get component details
dd status-pages --action get-component --page-id abc123 --component-id xyz789

# Update component status
dd status-pages --action update-component \
  --page-id abc123 \
  --component-id xyz789 \
  --status operational

# Remove component
dd status-pages --action delete-component --page-id abc123 --component-id xyz789
```

### Degradation (Incident) Management

```bash
# List all degradations/incidents
dd status-pages --action list-degradations

# Create incident
dd status-pages --action create-degradation \
  --page-id abc123 \
  --name "Authentication Service Degraded" \
  --severity major \
  --status investigating \
  --message "We are investigating authentication issues" \
  --notify

# Update incident status
dd status-pages --action update-degradation \
  --page-id abc123 \
  --degradation-id xyz789 \
  --status identified \
  --message "Root cause identified, deploying fix"

# Resolve incident
dd status-pages --action resolve-degradation \
  --page-id abc123 \
  --degradation-id xyz789

# Get JSON output
dd status-pages --action list-pages --json
```

## Actions

### Page Management Actions
- **list-pages**, **list** - List all status pages
- **create-page**, **create** - Create new status page
- **get-page**, **get** - Get status page details
- **update-page**, **update** - Update status page
- **delete-page**, **delete** - Delete status page

### Component Management Actions
- **list-components**, **components** - List components on a page
- **create-component**, **add-component** - Add component to page
- **get-component** - Get component details
- **update-component** - Update component
- **delete-component**, **remove-component** - Remove component

### Degradation/Incident Actions
- **list-degradations**, **degradations**, **incidents** - List all degradations
- **create-degradation**, **create-incident** - Create new incident
- **get-degradation**, **get-incident** - Get incident details
- **update-degradation**, **update-incident** - Update incident status
- **resolve-degradation**, **resolve-incident**, **resolve** - Resolve incident

## Degradation Statuses

**investigating** - Team is investigating the issue
**identified** - Root cause has been identified
**monitoring** - Fix deployed, monitoring for stability
**resolved** - Issue fully resolved

## Severity Levels

**critical** - Major service outage affecting all users
**major** - Significant degradation affecting many users
**minor** - Minor issues affecting some users
**maintenance** - Planned maintenance window

## Use Cases

### 1. Create Public Status Page

```bash
# Create page for external customers
dd status-pages --action create-page \
  --name "Service Status" \
  --subdomain company-status \
  --description "Real-time service status updates"
```

Create branded status pages for customer communication.

### 2. Add Service Components

```bash
# Add components to track
dd status-pages --action create-component \
  --page-id abc123 \
  --name "API Gateway" \
  --description "Primary API endpoint"

dd status-pages --action create-component \
  --page-id abc123 \
  --name "Database" \
  --description "Primary data store"
```

Break down services into components for granular status.

### 3. Report Service Degradation

```bash
# Create incident for degraded service
dd status-pages --action create-degradation \
  --page-id abc123 \
  --name "Elevated API Latency" \
  --severity minor \
  --status investigating \
  --message "We are seeing increased response times on API calls" \
  --notify
```

Proactively communicate issues to stakeholders.

### 4. Update Incident Progress

```bash
# Provide incident update
dd status-pages --action update-degradation \
  --page-id abc123 \
  --degradation-id xyz789 \
  --status identified \
  --message "Issue identified as database contention. Scaling database capacity."
```

Keep stakeholders informed throughout incident lifecycle.

### 5. Resolve and Close Incident

```bash
# Mark incident as resolved
dd status-pages --action resolve-degradation \
  --page-id abc123 \
  --degradation-id xyz789
```

Close incidents when service is fully restored.

### 6. Automate Status Updates

```bash
#!/bin/bash
# Automated incident creation from monitoring

PAGE_ID="abc123"
SEVERITY="major"

# Create incident
RESPONSE=$(dd status-pages --action create-degradation \
  --page-id "$PAGE_ID" \
  --name "API Service Down" \
  --severity "$SEVERITY" \
  --status investigating \
  --notify \
  --json)

DEGRAD_ID=$(echo "$RESPONSE" | jq -r '.data.id')
echo "Created incident: $DEGRAD_ID"
```

Integrate status page updates with monitoring and alerting.

## Why Use the CLI?

- **Fast incident response** - Create status pages in seconds
- **Automation** - Script incident creation and updates
- **CI/CD integration** - Update status during deployments
- **Component management** - Track individual service components
- **Stakeholder communication** - Keep customers informed
- **JSON output** - Parse responses for further automation

## Example Prompts

> "Create a status page for my API service"
> "Add database component to the status page"
> "Report an incident: API is degraded"
> "Update incident status to identified"
> "List all active incidents"
> "Resolve the authentication incident"

## Integration with Other Commands

Status Pages CLI integrates with:
- **incidents** - Sync with internal incident management
- **monitors** - Trigger status updates from monitors
- **deploy** - Update status during deployments
- **health** - Auto-create incidents from health checks

## Workflow Example

**Complete Incident Communication Workflow:**

```bash
# 1. Create status page (one-time setup)
PAGE_ID=$(dd status-pages --action create-page \
  --name "Production Services" \
  --subdomain prod-status \
  --json | jq -r '.data.id')

# 2. Add service components
dd status-pages --action create-component \
  --page-id "$PAGE_ID" \
  --name "API Service"

dd status-pages --action create-component \
  --page-id "$PAGE_ID" \
  --name "Database"

# 3. Detect issue and create incident
dd status-pages --action create-degradation \
  --page-id "$PAGE_ID" \
  --name "API Latency Spike" \
  --severity major \
  --status investigating \
  --notify

# 4. Update as investigation progresses
dd status-pages --action update-degradation \
  --page-id "$PAGE_ID" \
  --degradation-id "$DEGRAD_ID" \
  --status identified \
  --message "High database query volume causing latency"

# 5. Update when fix is deployed
dd status-pages --action update-degradation \
  --page-id "$PAGE_ID" \
  --degradation-id "$DEGRAD_ID" \
  --status monitoring \
  --message "Query optimization deployed, monitoring results"

# 6. Resolve when confirmed fixed
dd status-pages --action resolve-degradation \
  --page-id "$PAGE_ID" \
  --degradation-id "$DEGRAD_ID"
```

## Common Patterns

**Maintenance Window Communication:**
```bash
dd status-pages --action create-degradation \
  --page-id "$PAGE_ID" \
  --name "Scheduled Database Maintenance" \
  --severity maintenance \
  --status monitoring \
  --message "Database maintenance scheduled for 2AM-4AM UTC"
```

**Multi-Component Incident:**
```bash
dd status-pages --action create-degradation \
  --page-id "$PAGE_ID" \
  --name "Platform-Wide Degradation" \
  --severity critical \
  --status investigating \
  --message "Investigating connectivity issues affecting multiple services"
```

**Status Page Audit:**
```bash
# List all pages
dd status-pages --action list-pages

# List active incidents
dd status-pages --action list-degradations

# Check specific page components
dd status-pages --action list-components --page-id "$PAGE_ID"
```

## Setup Requirements

**Permissions Required:**
- `status_pages_settings_read` - Read status pages
- `status_pages_settings_write` - Create/update pages and components
- `status_pages_incident_write` - Create/update degradations

**Environment Variables:**
- `DD_API_KEY` - Datadog API key
- `DD_APP_KEY` - Datadog application key

## Troubleshooting

**Permission errors:**
1. Verify API/App keys have status pages permissions
2. Check user account has status pages access
3. Ensure proper roles assigned

**Page not accessible:**
1. Verify page ID is correct
2. Check subdomain is unique
3. Ensure custom domain is properly configured

**Notifications not sending:**
1. Use `--notify` flag for degradation updates
2. Verify subscribers are configured on the page
3. Check notification settings in Datadog UI

## Learn More

- [Status Pages Documentation](https://docs.datadoghq.com/service_management/status_pages/)
- [Status Pages Product Page](https://www.datadoghq.com/product/incident-management/status-pages/)
- [Incident Management Best Practices](https://docs.datadoghq.com/service_management/incident_management/)
- [Integrating Monitors with Status Pages](https://docs.datadoghq.com/monitors/guide/integrate-monitors-with-statuspage/)

## Related Commands

- `dd incidents` - Internal incident management
- `dd monitors` - Monitor management
- `dd cases` - Case management for issue tracking
- `dd deploy` - Pre-deployment safety checks
