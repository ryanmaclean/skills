---
description: "Query and trigger Datadog Workflow Automation - automate incident response and remediation"
argument-hint: "[WORKFLOW-NAME] [--trigger] [--status] [--history]"
---

# Datadog Workflow Automation

Query, trigger, and manage automated workflows for incident response, remediation actions, and operational tasks.

## What is Workflow Automation?

Workflow Automation enables codeless automation for common operational tasks:
- **Incident response** - Auto-remediation, escalation, notification
- **Resource management** - Scaling, provisioning, cleanup
- **Data collection** - Diagnostic gathering, log retrieval
- **Integration actions** - Cross-tool automation

**Official Documentation**: https://docs.datadoghq.com/service_management/workflows/

## Usage

```bash
# List all workflows
dd workflows list

# Get specific workflow details
dd workflows get --id abc123def

# Execute a workflow
dd workflows execute --id abc123def

# Execute with parameters
dd workflows execute --id abc123def --params '{"service":"api","replicas":3}'

# Execute and wait for completion
dd workflows execute --id abc123def --wait --timeout 300

# Create a new workflow
dd workflows create --name "Scale API Service" --file workflow.json

# Update a workflow
dd workflows update --id abc123def --file workflow-updated.json

# Delete a workflow
dd workflows delete --id abc123def
```

## Common Workflow Types

**Incident Response**:
- Auto-create incident tickets
- Notify on-call engineers
- Escalate to management
- Create Slack war rooms
- Gather diagnostic data

**Auto-Remediation**:
- Restart unhealthy containers
- Scale services under load
- Clear disk space
- Reset stuck queues
- Rollback deployments

**Diagnostics**:
- Collect logs and traces
- Run health checks
- Query database status
- Fetch metrics snapshots
- Generate incident reports

**Maintenance**:
- Schedule downtime
- Rotate credentials
- Clean up resources
- Update configurations
- Sync service catalogs

## Use Cases

### 1. Trigger Incident Response
```bash
dd workflows incident-response --trigger --params '{"severity":"critical","service":"api"}'
```

Automatically execute incident response playbook.

### 2. Auto-Remediation
```bash
dd workflows restart-service --trigger --params '{"service":"payment-api"}'
```

Restart a service experiencing issues without manual intervention.

### 3. Check Workflow Status
```bash
dd workflows scale-up-database --status
```

Monitor workflow execution progress.

### 4. Review Recent Executions
```bash
dd workflows --history --from 7d
```

Audit workflow executions for compliance and troubleshooting.

## Why Use the CLI?

- **Instant triggering** - Execute workflows in 3ms
- **Scriptable** - Integrate workflows into existing scripts
- **Emergency access** - Trigger remediation without UI access
- **Automation** - Chain workflows with other CLI commands
- **Audit trail** - Query workflow execution history
- **Testing** - Test workflows before productionizing

## Workflow Actions

**Datadog Actions**:
- Create/update monitors
- Mute/unmute monitors
- Create incidents
- Add incident timeline entries
- Update service catalog

**Cloud Integrations**:
- AWS (Lambda, EC2, ECS, S3)
- Azure (Functions, VMs, AKS)
- GCP (Cloud Functions, GCE, GKE)
- Kubernetes (scale, restart, describe)

**Communication**:
- Slack messages
- PagerDuty pages
- Email notifications
- Webhook calls
- Microsoft Teams

**Ticketing**:
- Jira issue creation
- ServiceNow tickets
- GitHub issues
- Linear tasks

## Example Prompts

> "Trigger the incident response workflow"
> "Show me workflow execution history"
> "Run the auto-scaling workflow for the API service"
> "Check the status of the remediation workflow"
> "List all workflows related to database maintenance"

## Learn More

- [Workflow Automation](https://docs.datadoghq.com/service_management/workflows/)
- [Workflow Actions](https://docs.datadoghq.com/service_management/workflows/actions_catalog/)
- [Workflow Triggers](https://docs.datadoghq.com/service_management/workflows/trigger/)
