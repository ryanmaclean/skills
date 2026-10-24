---
description: "Manage Case Management for issue tracking, resolution workflows, and team collaboration"
argument-hint: "--action <action> [--case-id ID] [--project-id ID] [options]"
---

# Datadog Case Management

Manage cases and projects for structured issue tracking, resolution workflows, and team collaboration in Datadog.

## What is Case Management?

Case Management provides structured workflows for handling:
- **Incidents** - Production issues requiring investigation
- **Bugs** - Software defects needing resolution
- **Tasks** - Work items for teams
- **Issues** - General problems requiring tracking

**Official Documentation**: https://docs.datadoghq.com/api/latest/case-management/

## Features

**Projects**:
- Organize cases by service, team, or initiative
- Custom workflows and statuses
- Access control and permissions

**Cases**:
- Structured issue tracking
- Priority levels (P1-P5)
- Assignment and ownership
- Status tracking and updates

**Collaboration**:
- Comments and discussions
- Timeline of changes
- @mentions and notifications
- Integration with incidents

## Usage

### Project Management

```bash
# List all projects
dd cases --action projects-list

# Create new project
dd cases --action projects-create \
  --project-key my-app \
  --title "My Application"

# Get project details
dd cases --action projects-get --project-id <id>

# Delete project
dd cases --action projects-delete --project-id <id>
```

### Case Management

```bash
# List all cases
dd cases --action list

# Search cases
dd cases --action list --query "status:open priority:P1"

# Create new case
dd cases --action create \
  --title "Database performance degradation" \
  --type-id <type-id> \
  --project-id <project-id> \
  --priority P1 \
  --description "Response times increased 3x"

# Get case details
dd cases --action get --case-id <id>

# Get JSON output
dd cases --action list --json
```

### Case Operations

```bash
# Assign case to user
dd cases --action assign \
  --case-id <id> \
  --assignee user@example.com

# Update case status
dd cases --action update-status \
  --case-id <id> \
  --status "In Progress"

# Update priority
dd cases --action update-priority \
  --case-id <id> \
  --priority P1

# Add comment
dd cases --action comment \
  --case-id <id> \
  --comment "Investigation complete, deploying fix"

# Archive case
dd cases --action archive --case-id <id>

# Unarchive case
dd cases --action unarchive --case-id <id>

# Unassign case
dd cases --action unassign --case-id <id>
```

## Priority Levels

**P1 - Critical**:
- Immediate attention required
- Production down or severely impacted
- All hands on deck

**P2 - High**:
- Resolve within hours
- Significant user impact
- Urgent but not critical

**P3 - Medium** (default):
- Resolve within days
- Normal priority work
- Standard workflow

**P4 - Low**:
- Resolve within weeks
- Minor issues or improvements
- Nice to have

**P5 - Minimal**:
- Resolve when convenient
- Cosmetic or very low impact
- Backlog items

## Use Cases

### 1. Create Incident Case
```bash
dd cases --action create \
  --title "Database connection pool exhausted" \
  --type-id incident-type \
  --project-id prod-db \
  --priority P1 \
  --description "Connection timeouts in production"
```

Track production incidents with full context and resolution workflow.

### 2. Track Bug Resolution
```bash
dd cases --action create \
  --title "UI button not responsive on mobile" \
  --type-id bug-type \
  --project-id mobile-app \
  --priority P2
```

Manage bug lifecycle from discovery to resolution.

### 3. Search Open P1 Cases
```bash
dd cases --action list --query "status:open priority:P1"
```

Quick view of critical open items requiring immediate attention.

### 4. Update Case Status
```bash
dd cases --action update-status \
  --case-id <id> \
  --status "Resolved"
```

Track progress through resolution workflow.

### 5. Assign Cases to Team
```bash
dd cases --action assign \
  --case-id <id> \
  --assignee engineer@company.com
```

Distribute work across team members.

## Why Use the CLI?

- **Fast case creation** - Create cases in 2 seconds vs navigating UI
- **Automation** - Integrate case management into CI/CD workflows
- **Bulk operations** - Script multiple case updates
- **Context awareness** - Auto-populate project and service info
- **Team collaboration** - Quick assignment and status updates

## Example Prompts

> "Create a P1 case for the database incident"
> "Show me all open critical cases"
> "Assign case XYZ to Alice"
> "Add a comment to case 123"
> "Update case status to resolved"
> "List all cases for my-service project"

## Integration

Case Management CLI integrates with:
- **Incidents** - Link cases to incident management
- **Monitors** - Create cases from monitor alerts
- **APM** - Track performance-related cases
- **Service Catalog** - Service-level case tracking

## Learn More

- [Case Management Product Page](https://www.datadoghq.com/product/case-management/)
- [Incident Management](https://docs.datadoghq.com/service_management/incident_management/)
- [Workflow Automation](https://docs.datadoghq.com/service_management/workflows/)
