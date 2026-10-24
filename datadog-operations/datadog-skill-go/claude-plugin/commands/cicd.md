---
description: "Query CI/CD Visibility for pipeline performance, test analytics, and deployment tracking"
argument-hint: "[PIPELINE] [--from TIMERANGE] [--branch BRANCH] [--status STATUS]"
---

# Datadog CI/CD Visibility

Query CI Visibility to monitor pipeline performance, track test execution, analyze build failures, and optimize deployment workflows.

## What is CI/CD Visibility?

CI Visibility provides comprehensive pipeline observability:
- **Pipeline monitoring** - Build times, failure rates, bottlenecks
- **Test analytics** - Flaky tests, test duration, coverage trends
- **Deployment tracking** - Release velocity, deployment frequency
- **Cost optimization** - CI resource usage and efficiency

**Official Documentation**: https://docs.datadoghq.com/continuous_integration/

## Usage

```bash
# Query all pipelines
dd cicd

# Query specific pipeline
dd cicd my-app-pipeline

# Filter by branch
dd cicd --branch main

# Filter by status
dd cicd --status failed

# Time range
dd cicd --from 24h
```

## Key Metrics

**Pipeline Performance**:
- Build duration (p50, p95, p99)
- Queue time vs execution time
- Pipeline success rate
- Failure frequency

**Test Analytics**:
- Test execution time
- Flaky test detection
- Test coverage trends
- Failing test patterns

**Deployment Metrics**:
- Deployment frequency (DORA metric)
- Lead time for changes
- Change failure rate
- Mean time to recovery

**Resource Efficiency**:
- CI minutes consumed
- Parallelization effectiveness
- Resource cost per build
- Waste detection

## Use Cases

### 1. Find Slow Pipelines
```bash
dd cicd --from 7d
```

Identifies pipelines with increasing build times or bottlenecks.

### 2. Track Flaky Tests
```bash
dd cicd my-pipeline --from 30d
```

Detects tests that intermittently fail, reducing developer frustration.

### 3. Monitor Deployment Frequency
```bash
dd cicd --branch main --from 7d
```

Tracks deployment velocity and measures DevOps performance.

### 4. Investigate Build Failures
```bash
dd cicd --status failed --from 24h
```

Quick access to recent failures for rapid troubleshooting.

## Why Use the CLI?

- **Fast debugging** - Check pipeline status in 3ms vs loading CI dashboard
- **Context awareness** - Auto-detects current repo and branch from git
- **Pre-commit checks** - Verify pipeline health before pushing code
- **Automation** - Integrate pipeline checks into local workflows
- **Test analysis** - Identify flaky tests before they merge

## Example Prompts

> "Show me failed pipelines in the last 24 hours"
> "What's the build time trend for the main branch?"
> "Find flaky tests in my-app-pipeline"
> "Check CI/CD status before I push"

## Learn More

- [CI Visibility Product Page](https://www.datadoghq.com/product/ci-cd-monitoring/)
- [Test Visibility](https://docs.datadoghq.com/continuous_integration/tests/)
- [Pipeline Visibility](https://docs.datadoghq.com/continuous_integration/pipelines/)
