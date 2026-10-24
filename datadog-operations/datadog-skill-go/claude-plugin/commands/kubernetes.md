---
description: "Query Kubernetes pod and cluster monitoring for namespace, deployment, and service health"
argument-hint: "[--namespace NAMESPACE] [--deployment DEPLOYMENT] [--state running|pending|failed]"
---

# Datadog Kubernetes Monitoring

Query Kubernetes pod and cluster monitoring to track pod health, deployment status, and cluster resources across namespaces.

## What is Kubernetes Monitoring?

Kubernetes Monitoring provides comprehensive visibility into:
- **Pod Health** - Running, pending, failed pod states
- **Deployments** - Rollout status and replica tracking
- **Services** - Service-to-pod relationships
- **Nodes** - Cluster node health and capacity
- **Namespaces** - Resource organization and isolation

**Official Documentation**: https://docs.datadoghq.com/containers/kubernetes/

## Usage

```bash
# List all running pods
dd kubernetes

# List pods in specific namespace
dd kubernetes --namespace production

# List pods for a deployment
dd kubernetes --deployment web-app

# Find failing pods
dd kubernetes --state failed

# Check pending pods (scheduling issues)
dd kubernetes --state pending

# List pods on specific node
dd kubernetes --node ip-10-0-1-100

# Search for pods by name
dd kubernetes --pod api

# Combine filters
dd kubernetes --namespace prod --deployment api --state running

# Get JSON output
dd kubernetes --namespace prod --json
```

## Pod States

**running** - Pod is running normally
**pending** - Pod is waiting to be scheduled
**failed** - Pod has failed to start
**succeeded** - Pod completed successfully
**unknown** - Pod state is unknown

## Key Features

**Namespace Filtering**:
- Query pods by namespace
- Cross-namespace views
- Namespace health summary

**Deployment Tracking**:
- Monitor deployment rollouts
- Track replica counts
- Identify deployment issues

**Service Discovery**:
- Service-to-pod mapping
- Endpoint health tracking
- Load balancer status

**Node Monitoring**:
- Pod distribution across nodes
- Node capacity and usage
- Scheduling problem detection

## Use Cases

### 1. Monitor Namespace Health
```bash
dd kubernetes --namespace production
```

Quick overview of all pods in production namespace.

### 2. Find Failing Pods
```bash
dd kubernetes --state failed
```

Identify pods that have failed for immediate troubleshooting.

### 3. Check Pending Pods
```bash
dd kubernetes --state pending
```

Find pods stuck in pending state (often resource or scheduling issues).

### 4. Track Deployment Rollout
```bash
dd kubernetes --deployment my-app
```

Monitor deployment rollout progress and health.

### 5. Investigate Node Issues
```bash
dd kubernetes --node problem-node
```

See which pods are running on a specific node.

### 6. Search Pods by Name
```bash
dd kubernetes --pod api
```

Find all pods with "api" in their name across namespaces.

### 7. Combined Debugging
```bash
dd kubernetes --namespace prod --state failed
```

Narrow down to failing pods in specific namespace.

## Why Use the CLI?

- **Fast pod queries** - Check pod status in seconds vs navigating UI
- **Namespace filtering** - Quickly scope to specific environments
- **Deployment tracking** - Monitor rollouts from terminal
- **Failure detection** - Instantly find failed or pending pods
- **Node debugging** - Investigate node-specific issues
- **Automation** - Script Kubernetes health checks

## Example Prompts

> "Show me all pods in production namespace"
> "Find failing pods"
> "Check pending pods in staging"
> "List pods for web-app deployment"
> "What pods are running on node xyz?"
> "Show me all api pods"

## Debugging Workflows

**New Deployment Troubleshooting:**
```bash
# 1. Check deployment pods
dd kubernetes --deployment new-app

# 2. Find any failing pods
dd kubernetes --deployment new-app --state failed

# 3. Check if any are pending
dd kubernetes --deployment new-app --state pending
```

**Namespace Health Check:**
```bash
# 1. Check all pods in namespace
dd kubernetes --namespace production

# 2. Find any failures
dd kubernetes --namespace production --state failed

# 3. Check resource constraints
dd kubernetes --namespace production --state pending
```

**Node Investigation:**
```bash
# 1. See what's on the node
dd kubernetes --node problem-node

# 2. Check for failures
dd kubernetes --node problem-node --state failed
```

## Integration

Kubernetes Monitoring CLI integrates with:
- **Container Monitoring** - Underlying container health
- **APM** - Trace Kubernetes service requests
- **Logs** - Aggregate pod logs
- **Metrics** - CPU, memory, network metrics
- **Events** - Kubernetes event stream

## Kubernetes Tags

Datadog automatically tags Kubernetes containers:
- `kube_namespace` - Namespace name
- `kube_deployment` - Deployment name
- `kube_service` - Service name
- `pod_name` - Pod name
- `kube_container_name` - Container within pod
- `host` - Node name

## Common Patterns

**Check Production Health:**
```bash
dd kubernetes --namespace production --state running
```

**Find Issues Across All Namespaces:**
```bash
dd kubernetes --state failed
dd kubernetes --state pending
```

**Monitor Specific Service:**
```bash
dd kubernetes --service my-service
```

**Debug Deployment:**
```bash
dd kubernetes --deployment my-app --state failed
```

## Learn More

- [Kubernetes Monitoring Product Page](https://www.datadoghq.com/containers/kubernetes-monitoring/)
- [Kubernetes Documentation](https://docs.datadoghq.com/containers/kubernetes/)
- [Orchestrator Explorer](https://docs.datadoghq.com/infrastructure/containers/orchestrator_explorer/)
- [Cluster Monitoring](https://docs.datadoghq.com/containers/cluster_agent/)

## Related Commands

- `dd containers` - General container monitoring
- `dd apm` - Trace Kubernetes services
- `dd logs` - Query pod logs
- `dd metrics` - Pod resource metrics
