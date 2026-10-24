---
description: "Query and manage Datadog dashboards - fetch metrics, export configurations, create snapshots"
argument-hint: "[DASHBOARD-NAME] [--export] [--snapshot]"
---

# Datadog Dashboard Management

Query dashboard metrics, export configurations, create snapshots, and access dashboard data programmatically.

## What are Dashboards?

Dashboards are customizable visualizations of your observability data:
- **Timeboards** - Time-synchronized graphs for correlation analysis
- **Screenboards** - Flexible layouts for status boards
- **Host dashboards** - Infrastructure-level metrics
- **Integration dashboards** - Pre-built vendor dashboards
- **Custom dashboards** - Team-specific views

**Official Documentation**: https://docs.datadoghq.com/dashboards/

## Usage

```bash
# List all dashboards
dd dashboards

# Query specific dashboard data
dd dashboards my-service-overview

# Export dashboard configuration
dd dashboards my-dashboard --export

# Create snapshot
dd dashboards my-dashboard --snapshot

# Filter by tags
dd dashboards --tag team:platform

# Get dashboard URL
dd dashboards my-dashboard --url
```

## Dashboard Types

**Service Dashboards**:
- APM service overview
- Error rates and latency
- Throughput and saturation
- Dependency maps

**Infrastructure Dashboards**:
- Host metrics (CPU, memory, disk)
- Container metrics
- Kubernetes cluster health
- Cloud resource utilization

**Business Dashboards**:
- User journey metrics
- Conversion funnels
- Revenue tracking
- SLO compliance

**Custom Dashboards**:
- Team-specific views
- Incident response dashboards
- Executive summaries
- On-call status boards

## Use Cases

### 1. Quick Metric Access
```bash
dd dashboards production-overview
```

Access dashboard metrics without opening browser.

### 2. Dashboard as Code
```bash
dd dashboards my-dashboard --export > dashboard.json
```

Export dashboard configuration for version control.

### 3. Snapshot Sharing
```bash
dd dashboards incident-dashboard --snapshot
```

Create shareable snapshot for incident reports.

### 4. Automated Reporting
```bash
dd dashboards weekly-summary --from 7d --export
```

Generate weekly reports from dashboard data.

## Why Use the CLI?

- **Fast access** - View dashboard data in 3ms
- **Scriptable** - Automate dashboard creation and updates
- **Export/Import** - Manage dashboards as code (GitOps)
- **Snapshot creation** - Document incidents with point-in-time views
- **CI/CD integration** - Include dashboard checks in pipelines
- **Offline access** - Cache dashboard configs locally

## Widget Support

**Graph Widgets**:
- Timeseries
- Toplist
- Heatmap
- Distribution

**Query Widgets**:
- Table
- Query value
- Change
- Geomap

**Status Widgets**:
- Monitor summary
- Service level objective
- Alert value
- Event stream

## Example Prompts

> "Show me the production overview dashboard"
> "Export my team's dashboard to JSON"
> "Create a snapshot of the incident dashboard"
> "List all dashboards tagged with 'sre'"
> "Get the URL for the database dashboard"

## Learn More

- [Dashboard Types](https://docs.datadoghq.com/dashboards/)
- [Widgets](https://docs.datadoghq.com/dashboards/widgets/)
- [Dashboard API](https://docs.datadoghq.com/api/latest/dashboards/)
- [Dashboard as Code](https://docs.datadoghq.com/dashboards/guide/dashboard-as-code/)
