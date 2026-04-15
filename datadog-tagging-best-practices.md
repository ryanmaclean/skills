# Datadog Tagging Best Practices

## Overview

This skill defines Datadog's official tagging best practices for unified service tagging across all telemetry (APM traces, logs, metrics).

## The Three Core Tags (Required)

Datadog reserves three standard tags that automatically connect all telemetry:

| Tag | Environment Variable | Description | Example Values |
|-----|---------------------|-------------|----------------|
| `env` | `DD_ENV` | Deployment environment | `production`, `staging`, `development`, `local` |
| `service` | `DD_SERVICE` | Application/service name | `payment-api`, `user-service`, `ddtrace-agent-v3` |
| `version` | `DD_VERSION` | Code version/release | `1.2.3`, `v2.0.0`, `0.3.0` |

## Additional Recommended Tags

| Tag | Purpose | Example Values |
|-----|---------|----------------|
| `team` | Ownership/cost allocation | `platform`, `backend`, `infra` |
| `component` | Sub-service component | `api`, `worker`, `scheduler` |
| `cluster` | Kubernetes cluster name | `tundra-dome`, `gastown`, `minim4-tundra` |
| `namespace` | K8s namespace | `default`, `datadog`, `airflow` |

## Tag Naming Rules

1. **Lowercase only** - Tags must be lowercase
2. **Start with letter** - Tags must begin with a letter
3. **Alphanumeric + underscores** - Use `a-z`, `0-9`, `_`, `-`, `:`, `/`, `.`
4. **Max 200 characters** - Tag names limited to 200 chars
5. **Colons for namespacing** - Use `:` for hierarchical tags (e.g., `team:backend`)
6. **No spaces** - Use underscores or hyphens instead

## Implementation Patterns

### Python Scripts (ddtrace)

```python
#!/usr/bin/env python3
from __future__ import annotations

import os

# Datadog Unified Service Tagging
try:
    from ddtrace import config, patch_all, tracer

    # Core tags (required)
    config.service = os.environ.get("DD_SERVICE", "script-name")
    config.env = os.environ.get("DD_ENV", "development")
    config.version = os.environ.get("DD_VERSION", "0.1.0")

    # Additional tags
    tracer.set_tags({
        "team": "platform",
        "component": "automation",
    })

    patch_all()
except ImportError:
    pass
```

### Environment Variables (Shell/CI)

```bash
export DD_ENV=production
export DD_SERVICE=my-service
export DD_VERSION=1.2.3
export DD_TAGS="team:backend,component:api,cluster:tundra-dome"
```

### Kubernetes Labels

```yaml
metadata:
  labels:
    tags.datadoghq.com/env: "production"
    tags.datadoghq.com/service: "my-service"
    tags.datadoghq.com/version: "1.2.3"
```

## Service Naming Conventions

| Pattern | Example | Use Case |
|---------|---------|----------|
| `{app}-{component}` | `crew-api` | Microservices |
| `{domain}-{function}` | `payment-processor` | Domain-driven |
| `{script-name}` | `ddtrace-agent-v3` | Standalone scripts |
| `{package-name}` | `tundra-automation` | Python packages |

## Anti-Patterns to Avoid

- **Over-tagging** - Each tag adds cost; keep tag cardinality low
- **High-cardinality values** - Avoid UUIDs, timestamps, user IDs as tag values
- **Inconsistent naming** - Use same tag names across all services
- **Missing core tags** - Always set `env`, `service`, `version`
- **Hardcoded values** - Use environment variables for flexibility

## Version Tag Strategies

| Strategy | Format | Example |
|----------|--------|---------|
| Semantic versioning | `MAJOR.MINOR.PATCH` | `1.2.3` |
| Git SHA (short) | `{sha}` | `abc1234` |
| Date-based | `YYYYMMDD.N` | `20260205.1` |
| Branch + SHA | `{branch}-{sha}` | `main-abc1234` |

## Cost Optimization

- Limit custom tags to essential ones (team, component, cluster)
- Use tag inheritance from infrastructure where possible
- Avoid creating new tags for each deployment
- Use `DD_TAGS` for static tags shared across requests

## References

- [Unified Service Tagging](https://docs.datadoghq.com/getting_started/tagging/unified_service_tagging/)
- [Tagging Best Practices](https://learn.datadoghq.com/courses/tagging-best-practices)
- [Assigning Tags](https://docs.datadoghq.com/getting_started/tagging/assigning_tags/)
