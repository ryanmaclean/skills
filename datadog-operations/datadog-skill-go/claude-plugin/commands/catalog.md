---
description: "Query Service Catalog for service metadata, ownership, and dependencies"
argument-hint: "[SERVICE-NAME] [--team TEAM]"
---

# Datadog Service Catalog

Query the Service Catalog to discover services, view metadata, track ownership, and understand dependencies.

## What is Service Catalog?

Service Catalog is a centralized registry of all services:
- **Service inventory** - Complete list of services
- **Ownership information** - Teams, contacts, on-call
- **Dependencies** - Service relationships and data flow
- **Metadata** - Repositories, documentation, runbooks

**Official Documentation**: https://www.datadoghq.com/product/software-catalog/

## Usage

```bash
# List all services
dd catalog

# Query specific service
dd catalog api-service

# Filter by team
dd catalog --team platform
```

## Why Use the CLI?

- **Fast discovery** - Find services instantly
- **Context awareness** - Auto-detects current service from git
- **Scriptable** - Automate service discovery
- **Offline access** - Cached service information

## Example Prompts

> "List all services in the catalog"
> "Show me details for api-service"
> "What services does the platform team own?"

## Learn More

- [Service Catalog](https://www.datadoghq.com/product/software-catalog/)