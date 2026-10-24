---
description: "Query Cloud Cost Management for FinOps - track AWS, Azure, GCP costs and optimize spending"
argument-hint: "[--cloud PROVIDER] [--service SERVICE] [--from TIMERANGE] [--group-by TAG]"
---

# Datadog Cloud Cost Management

Query and analyze cloud infrastructure costs across AWS, Azure, and GCP. Optimize spending with usage insights and cost allocation.

## What is Cloud Cost Management?

Cloud Cost Management provides unified cost visibility across cloud providers:
- **Multi-cloud costs** - AWS, Azure, GCP in one view
- **Cost allocation** - By team, service, environment, or custom tags
- **Optimization insights** - Identify waste and savings opportunities
- **Showback/Chargeback** - Accurate team-level cost reporting
- **Forecasting** - Predict future costs based on trends

**Official Documentation**: https://docs.datadoghq.com/cloud_cost_management/

## Usage

```bash
# Query all cloud costs
dd cost

# Filter by cloud provider
dd cost --cloud aws
dd cost --cloud azure
dd cost --cloud gcp

# Filter by service
dd cost --cloud aws --service ec2
dd cost --cloud aws --service rds

# Group by tags
dd cost --group-by team
dd cost --group-by environment
dd cost --group-by service

# Time range
dd cost --from 30d
dd cost --from 1M

# Cost anomalies
dd cost --anomalies
```

## Cost Categories

**Compute Costs**:
- EC2, Azure VMs, GCE instances
- Kubernetes (EKS, AKS, GKE)
- Serverless (Lambda, Functions, Cloud Run)
- Container services (ECS, ACI)

**Storage Costs**:
- Object storage (S3, Blob, GCS)
- Block storage (EBS, Azure Disks, Persistent Disks)
- Databases (RDS, CosmosDB, Cloud SQL)
- Backups and snapshots

**Network Costs**:
- Data transfer (egress/ingress)
- Load balancers
- VPN connections
- NAT gateways

**Other Services**:
- Managed services (SQS, Event Hub, Pub/Sub)
- Monitoring and logging
- AI/ML services
- CDN and edge services

## Key Metrics

**Cost Analysis**:
- Total monthly cost
- Cost by service
- Cost by team/tag
- Cost trends

**Optimization**:
- Idle resources
- Oversized instances
- Unused volumes
- Orphaned snapshots
- Reserved instance opportunities

**Allocation**:
- Team chargeback
- Environment split (prod vs dev)
- Service attribution
- Project-level costs

## Use Cases

### 1. Monthly Cost Review
```bash
dd cost --from 30d --group-by team
```

Generate team-level cost reports for monthly reviews.

### 2. Service Cost Attribution
```bash
dd cost --cloud aws --group-by service
```

Understand which services consume the most budget.

### 3. Detect Cost Anomalies
```bash
dd cost --anomalies --from 7d
```

Identify unexpected cost spikes before month-end surprises.

### 4. Environment Cost Split
```bash
dd cost --group-by environment
```

Compare production vs development/staging costs.

### 5. Optimization Opportunities
```bash
dd cost --cloud aws --service ec2 --recommendations
```

Find idle instances and rightsizing opportunities.

## Why Use the CLI?

- **Fast reporting** - Generate cost reports in 3ms
- **Scriptable** - Automate cost analysis in CI/CD
- **Budget alerts** - Check costs before deployments
- **Team chargeback** - Export cost data for billing
- **Optimization workflows** - Identify waste programmatically
- **Trend analysis** - Track cost changes over time

## Cost Optimization

**Common Savings**:
- Terminate idle resources
- Rightsize oversized instances
- Purchase reserved instances
- Use spot/preemptible instances
- Delete orphaned resources
- Enable autoscaling

**Waste Detection**:
- EC2 instances with <5% CPU
- Unattached EBS volumes
- Old snapshots
- Unused load balancers
- Stopped but not terminated instances

**FinOps Workflows**:
- Daily cost anomaly checks
- Weekly team cost reports
- Monthly budget reviews
- Quarterly optimization sprints

## Example Prompts

> "Show me total AWS costs for the last 30 days"
> "What are our Azure costs grouped by team?"
> "Find cost anomalies in the last week"
> "What's our Kubernetes spend across all clouds?"
> "Show me idle EC2 instances"
> "Compare prod vs staging costs"

## Learn More

- [Cloud Cost Management](https://docs.datadoghq.com/cloud_cost_management/)
- [AWS Cost Analysis](https://docs.datadoghq.com/cloud_cost_management/aws/)
- [Azure Cost Analysis](https://docs.datadoghq.com/cloud_cost_management/azure/)
- [Cost Recommendations](https://docs.datadoghq.com/cloud_cost_management/recommendations/)
