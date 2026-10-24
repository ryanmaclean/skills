---
description: "Auto-detect current service context from git repository and environment"
argument-hint: "[--show] [--set SERVICE]"
---

# Datadog Context Detection

Automatically detect the current service context from your git repository, environment variables, and working directory to provide contextual Datadog queries.

## What is Context Detection?

Context detection intelligently infers which Datadog service you're working on:
- **Git integration** - Detects service from repository name
- **Environment variables** - Reads DD_SERVICE, DD_ENV
- **Configuration files** - Parses datadog.yaml, APM config
- **Directory structure** - Infers service from path patterns
- **Manual override** - Set context explicitly when needed

This is a **unique CLI capability** not available in the Datadog web UI or API.

## Usage

```bash
# Show current context
dd context

# Set context manually
dd context --set api-service

# Clear context
dd context --clear

# Use detected context in queries
dd apm              # Automatically queries current service
dd logs             # Filters to current service
dd metrics          # Shows current service metrics
```

## Context Sources (Priority Order)

1. **Manual Override** (highest priority)
   ```bash
   dd context --set payment-service
   ```

2. **Environment Variables**
   ```bash
   export DD_SERVICE=api-service
   export DD_ENV=production
   ```

3. **Git Repository**
   - Repository name mapping
   - Git remote URL parsing
   - Service catalog integration

4. **Configuration Files**
   - `datadog.yaml` (DD_SERVICE field)
   - `apm.config` (service name)
   - `.ddconfig` (custom config)

5. **Directory Structure**
   - `/services/api-service/` → api-service
   - `/apps/frontend/` → frontend
   - Pattern-based inference

## Why This is Unique

**Not Available in Web UI**:
- The Datadog web UI requires manual service selection
- No automatic detection based on your current work

**Not Available in API**:
- The Datadog API requires explicit service names
- No context inference from environment

**CLI Exclusive Benefits**:
- Zero-friction queries (no need to specify service)
- Works offline (uses cached context)
- Integrates with local development workflow
- Context-aware command completion

## Use Cases

### 1. Automatic Service Queries
```bash
# Working in /projects/api-service/
dd apm              # Automatically shows API service traces
dd logs --tail      # Tails API service logs
dd metrics          # Shows API service metrics
```

Context detection eliminates the need for `--service api-service` flags.

### 2. Multi-Service Development
```bash
# Switch between services
cd /projects/frontend/
dd context          # Shows: frontend (from git)

cd /projects/api/
dd context          # Shows: api-service (from git)
```

Context automatically updates as you navigate projects.

### 3. Environment-Aware Queries
```bash
export DD_ENV=staging
dd apm              # Automatically filters to staging environment
```

### 4. Manual Override for Testing
```bash
dd context --set test-service
dd deploy --check   # Checks test-service deployment
dd context --clear  # Return to auto-detection
```

## Context Detection Logic

```
1. Check manual override (dd context --set)
   ↓
2. Check environment variables (DD_SERVICE, DD_ENV)
   ↓
3. Check git remote (parse service from repo URL)
   ↓
4. Check datadog.yaml or apm.config
   ↓
5. Check directory structure patterns
   ↓
6. Check service catalog (match repo to service)
   ↓
7. Prompt user if ambiguous
```

## Configuration

**~/.ddconfig** (optional):
```yaml
# Service name patterns
patterns:
  - regex: "^api-.*"
    service: "api-service"
  - regex: "^frontend-.*"
    service: "frontend"

# Repository mappings
repos:
  "myorg/api-repo": "api-service"
  "myorg/web-repo": "frontend"

# Default environment
default_env: "production"
```

## Why Use Context Detection?

- **Faster queries** - No need to specify service every time
- **Fewer errors** - Automatic detection prevents typos
- **Better UX** - Intuitive workflow aligned with development
- **Environment awareness** - Automatically filters to correct env
- **Offline capability** - Works without API calls
- **Unique to CLI** - Not available in UI or API

## Example Prompts

> "Show me the current context"
> "What service am I working on?"
> "Switch context to payment-service"
> "Show APM traces for the current service"
> "Check health of the current service"

## Learn More

This is a CLI-exclusive feature. Context detection combines:
- [Service Catalog](https://docs.datadoghq.com/service_catalog/)
- [Unified Service Tagging](https://docs.datadoghq.com/getting_started/tagging/unified_service_tagging/)
- [APM Configuration](https://docs.datadoghq.com/tracing/trace_collection/)
