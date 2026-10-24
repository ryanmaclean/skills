---
description: "Query container monitoring for Docker, Kubernetes pods, and orchestrated environments"
argument-hint: "[--tags TAGS] [--image IMAGE] [--state running|exited|paused]"
---

# Datadog Container Monitoring

Query container monitoring data for Docker, Kubernetes pods, and other containerized workloads across your infrastructure.

## What is Container Monitoring?

Container Monitoring provides real-time visibility into:
- **Container Health** - Running, stopped, failed containers
- **Resource Usage** - CPU, memory, network, disk
- **Image Tracking** - Which images are deployed where
- **Orchestration** - Kubernetes pods, Docker Swarm, ECS tasks

**Official Documentation**: https://docs.datadoghq.com/containers/

## Usage

```bash
# List all containers
dd containers

# List running containers only
dd containers --state running

# Filter by image name
dd containers --image nginx

# Filter by tags (environment and service)
dd containers --tags "env:production,service:web"

# Kubernetes pods in production namespace
dd containers --tags "kube_namespace:production"

# Get JSON output
dd containers --state running --json
```

## Container States

**running** - Container is actively running
**exited** - Container has stopped
**paused** - Container execution is paused
**created** - Container created but not started
**dead** - Container is in dead state

## Key Features

**Multi-Platform Support**:
- Docker containers
- Kubernetes pods
- ECS tasks
- Container orchestrators
- Standalone containers

**Filtering & Grouping**:
- Filter by tags, image, state
- Group by service, team, environment
- Sort by name, start time
- Pagination support

**Kubernetes Integration**:
Kubernetes pods appear as containers with automatic tags:
- `kube_namespace` - Namespace name
- `kube_deployment` - Deployment name
- `kube_service` - Service name
- `pod_name` - Pod name
- `kube_container_name` - Container within pod

## Use Cases

### 1. Monitor Container Health
```bash
dd containers --state running
```

Quick overview of all healthy running containers across infrastructure.

### 2. Identify Failed Containers
```bash
dd containers --state exited
```

Find containers that have stopped or failed for troubleshooting.

### 3. Audit Container Images
```bash
dd containers --image postgres
```

Track all containers using a specific image (useful for security updates).

### 4. Filter by Environment
```bash
dd containers --tags "env:production"
```

View only production containers for targeted monitoring.

### 5. Kubernetes Pod Monitoring
```bash
dd containers --tags "kube_namespace:prod"
```

Monitor Kubernetes pods in specific namespace.

### 6. Find Specific Service
```bash
dd containers --tags "service:api,env:prod"
```

Combine multiple tags to find exact containers.

## Why Use the CLI?

- **Fast queries** - Check container status in seconds
- **Kubernetes integration** - Query pods directly
- **Image tracking** - Find containers by image name
- **State monitoring** - Quickly identify failed containers
- **Automation** - Script container health checks

## Example Prompts

> "Show me all running containers"
> "List containers running nginx"
> "Find failed containers in production"
> "Show Kubernetes pods in default namespace"
> "What containers are using the api:latest image?"

## Integration

Container Monitoring CLI integrates with:
- **Kubernetes** - Pod and cluster monitoring
- **Docker** - Container and image tracking
- **ECS/Fargate** - AWS container services
- **APM** - Correlate containers with traces
- **Logs** - Container log aggregation

## Filtering Tips

**By Environment:**
```bash
dd containers --tags "env:production"
dd containers --tags "env:staging"
```

**By Service:**
```bash
dd containers --tags "service:web"
dd containers --tags "service:api"
```

**By Kubernetes Namespace:**
```bash
dd containers --tags "kube_namespace:production"
dd containers --tags "kube_namespace:monitoring"
```

**Combine Multiple Filters:**
```bash
dd containers --tags "env:prod,service:api,kube_namespace:default"
```

## Learn More

- [Container Monitoring Product Page](https://www.datadoghq.com/product/container-monitoring/)
- [Container Monitoring Documentation](https://docs.datadoghq.com/containers/)
- [Kubernetes Monitoring](https://docs.datadoghq.com/containers/kubernetes/)
- [Docker Monitoring](https://docs.datadoghq.com/containers/docker/)

## Related Commands

- `dd kubernetes` - Kubernetes-specific pod monitoring
- `dd apm` - Trace application performance in containers
- `dd logs` - Query container logs
- `dd metrics` - Container resource metrics
