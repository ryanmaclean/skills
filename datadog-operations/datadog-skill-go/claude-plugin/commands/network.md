---
description: "Query Network Performance Monitoring for network traffic, connections, and latency"
argument-hint: "[--from TIMERANGE] [--source SOURCE] [--dest DEST]"
---

# Datadog Network Performance Monitoring

Query Network Performance Monitoring to analyze network traffic, connections, dependencies, and performance between services.

## What is Network Monitoring?

Network Monitoring provides network-level visibility:
- **Traffic analysis** - Volume, protocols, connections
- **Service dependencies** - Network-level service map
- **Performance metrics** - Latency, retransmits, throughput
- **Security** - Unexpected connections, data exfiltration

**Official Documentation**: https://www.datadoghq.com/product/network-monitoring/

## Usage

```bash
# Query all network traffic
dd network

# Filter by source
dd network --source api-service

# Filter by destination
dd network --dest database-service

# Time range
dd network --from 1h
```

## Why Use the CLI?

- **Network debugging** - Quickly identify connection issues
- **Dependency discovery** - See actual network dependencies
- **Performance analysis** - Track network latency
- **Security monitoring** - Detect unusual traffic patterns

## Example Prompts

> "Show me network traffic between services"
> "What's the network latency to the database?"
> "Find network connections from api-service"

## Learn More

- [Network Monitoring](https://www.datadoghq.com/product/network-monitoring/)