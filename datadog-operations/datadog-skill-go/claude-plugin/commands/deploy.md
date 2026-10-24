---
description: "Validate deployment safety before deploying - checks for active incidents, error spikes, and service health"
argument-hint: "[SERVICE] [--environment ENV]"
---

# Datadog Deploy Safety Check

Validate whether it's safe to deploy a service by checking for active incidents, error spikes, ongoing deployments, and overall service health.

## Usage

```bash
# Auto-detect service from git context
dd deploy

# Check specific service
dd deploy <service-name>

# Check specific environment
dd deploy <service-name> --environment production
```

## Safety Checks

The deploy command performs multiple safety validations:

1. **Active Incidents**: Checks for SEV-1, SEV-2, or SEV-3 incidents affecting the service
2. **Error Rate Spikes**: Analyzes recent error rate trends (last 30 minutes)
3. **Ongoing Deployments**: Detects if another deployment is in progress
4. **Service Health**: Overall health assessment (degraded/unhealthy services)
5. **Monitor Alerts**: Active monitor alerts for the service

## Output

Deploy safety check provides:
- **Safety Decision**: SAFE / UNSAFE / WARNING with reasoning
- **Blocking Issues**: Critical problems preventing deployment
- **Warnings**: Non-blocking concerns to be aware of
- **Recommendations**: Suggested actions before deploying

## Common Use Cases

1. **Pre-merge PR checks**: Validate deploy safety before merging pull requests
2. **CI/CD gates**: Automated safety checks in deployment pipelines
3. **Manual deployments**: Human verification before manual deployments
4. **Rollback decisions**: Determine if rollback is needed during incidents

## Why Use the CLI?

The `dd deploy` command provides unique deployment validation not available elsewhere:

- **Multi-signal analysis** - Combines incidents, errors, monitors, and health in one check
- **Pre-deployment gating** - Block unsafe deployments before they happen (unique to CLI)
- **Context-aware** - Auto-detects service from git repository
- **Fast validation** - Results in 3ms vs manually checking multiple dashboards
- **CI/CD integration** - Perfect for automated deployment gates
- **Exit codes** - Machine-readable safety decisions for automation

## Example Prompts

> "Is it safe to deploy payment-service to production?"
> "Check if I can deploy the api service"
> "Should I deploy user-service right now?"
> "Deploy safety check for checkout-service in staging"

## Environment Variables

Required:
- `DD_API_KEY`: Datadog API key
- `DD_APP_KEY`: Datadog application key
- `DD_SITE`: Datadog site (default: datadoghq.com)

## Exit Codes

- `0`: Safe to deploy
- `1`: Unsafe to deploy (blocking issues)
- `2`: Deploy with caution (warnings present)

## Integration with CI/CD

Use in GitHub Actions, GitLab CI, or other CI/CD systems:

```yaml
- name: Check deploy safety
  run: dd deploy $SERVICE_NAME --environment $ENVIRONMENT
  env:
    DD_API_KEY: ${{ secrets.DD_API_KEY }}
    DD_APP_KEY: ${{ secrets.DD_APP_KEY }}
```

## Notes

- Default environment is "production" if not specified
- Service auto-detection uses git repository name and Datadog service tags
- Safety checks use the last 30 minutes of data by default

## Learn More

- [Deployment Tracking](https://docs.datadoghq.com/tracing/deployment_tracking/)
- [Change Management](https://docs.datadoghq.com/service_management/change/)
- [Incident Management](https://docs.datadoghq.com/service_management/incident_management/)
