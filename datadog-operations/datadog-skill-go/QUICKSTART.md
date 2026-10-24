# Datadog CLI - Quick Start Guide

Get up and running with the Datadog CLI in under 5 minutes.

---

## Installation

### Option 1: Download Pre-Built Binary (Recommended)

**macOS (Apple Silicon)**:
```bash
curl -L https://github.com/yourusername/datadog-cli-go/releases/latest/download/dd-darwin-arm64 -o dd
chmod +x dd
sudo mv dd /usr/local/bin/
```

**macOS (Intel)**:
```bash
curl -L https://github.com/yourusername/datadog-cli-go/releases/latest/download/dd-darwin-amd64 -o dd
chmod +x dd
sudo mv dd /usr/local/bin/
```

**Linux (AMD64)**:
```bash
curl -L https://github.com/yourusername/datadog-cli-go/releases/latest/download/dd-linux-amd64 -o dd
chmod +x dd
sudo mv dd /usr/local/bin/
```

**Linux (ARM64)**:
```bash
curl -L https://github.com/yourusername/datadog-cli-go/releases/latest/download/dd-linux-arm64 -o dd
chmod +x dd
sudo mv dd /usr/local/bin/
```

**Windows (PowerShell)**:
```powershell
curl -L https://github.com/yourusername/datadog-cli-go/releases/latest/download/dd-windows-amd64.exe -o dd.exe
# Add to PATH or run from current directory
```

### Option 2: Build from Source

```bash
# Clone repository
git clone https://github.com/yourusername/datadog-cli-go.git
cd datadog-cli-go

# Build
make build

# Or build for all platforms
make build-all

# Binary location
./dd --version
```

---

## Configuration

### Set Environment Variables

```bash
# Required: Datadog API credentials
export DD_API_KEY="your_datadog_api_key"
export DD_APP_KEY="your_datadog_app_key"

# Optional: Datadog site (default: datadoghq.com)
export DD_SITE="datadoghq.com"  # US1
# export DD_SITE="datadoghq.eu"    # EU
# export DD_SITE="ddog-gov.com"    # US1-FED
```

### Get Your API Keys

1. Go to Datadog: https://app.datadoghq.com/organization-settings/api-keys
2. Create or copy your API key
3. Create or copy your Application key
4. Export them as shown above

---

## Verify Installation

```bash
# Check version
dd --version

# Test basic command (auto-detects service from current directory)
dd context
```

**Expected Output**:
```json
{
  "service": "your-service-name",
  "confidence": 90,
  "source": "git-remote",
  "detected_from": "github.com/username/repo"
}
```

---

## Common Commands

### 1. APM Traces

```bash
# Get recent traces for current service
dd apm

# JSON output
dd apm --json

# Specify service
dd apm --service my-service
```

### 2. Logs

```bash
# Search logs for current service
dd logs

# Search with query
dd logs --query "error OR warning"

# Last hour
dd logs --from 1h
```

### 3. Metrics

```bash
# Get metrics for current service
dd metrics

# Specific metric
dd metrics --query "system.cpu.user"
```

### 4. Service Health

```bash
# Multi-signal health check
dd health

# Check specific service
dd health --service my-api
```

### 5. Deployment Readiness

```bash
# Check if service is ready for deployment
dd deploy

# Returns exit code 0 if ready, 1 if not
```

### 6. SLO Status

```bash
# View SLO compliance
dd slos

# Specific SLO
dd slos --slo "API Response Time"
```

### 7. Monitors

```bash
# List all monitors
dd monitors

# Create monitor
dd monitors --create --file monitor.json

# Update monitor
dd monitors --update 12345 --file monitor.json
```

### 8. RUM (Real User Monitoring)

```bash
# Get RUM metrics
dd rum

# Core Web Vitals
dd rum --metrics lcp,fid,cls
```

### 9. Cost Analysis

```bash
# Analyze infrastructure costs
dd cost

# Get recommendations
dd cost --recommendations
```

### 10. LLM Observability

```bash
# Track LLM usage and costs
dd llm

# Specific model
dd llm --model gpt-4
```

---

## Quick Examples

### Check Service Before Deployment

```bash
#!/bin/bash
# pre-deploy.sh

echo "Checking deployment readiness..."

# Check service health
if dd health --json | jq -e '.status == "healthy"' > /dev/null; then
  echo "✅ Service is healthy"
else
  echo "❌ Service health check failed"
  exit 1
fi

# Check for recent errors
if dd logs --query "status:error" --from 5m --json | jq -e '.total == 0' > /dev/null; then
  echo "✅ No recent errors"
else
  echo "❌ Recent errors detected"
  exit 1
fi

# Check SLO compliance
if dd slos --json | jq -e '.overall_compliance > 99.9' > /dev/null; then
  echo "✅ SLOs are met"
else
  echo "⚠️  SLO compliance below threshold"
fi

echo "✅ Ready for deployment"
```

### Monitor Application Performance

```bash
#!/bin/bash
# monitor.sh

while true; do
  echo "=== $(date) ==="

  # Get current error rate
  dd apm --json | jq '.error_rate'

  # Get latency p95
  dd apm --json | jq '.latency.p95'

  # Check for anomalies
  dd watchdog --json | jq '.anomalies | length'

  sleep 60
done
```

### Daily Health Report

```bash
#!/bin/bash
# daily-report.sh

echo "# Daily Health Report - $(date +%Y-%m-%d)"
echo ""

echo "## Service Health"
dd health

echo ""
echo "## SLO Status"
dd slos

echo ""
echo "## Recent Incidents"
dd incidents --from 24h

echo ""
echo "## Cost Summary"
dd cost --period 24h

echo ""
echo "## Watchdog Anomalies"
dd watchdog --from 24h
```

---

## Output Formats

### JSON Output (for scripts)

```bash
# All commands support --json flag
dd context --json
dd apm --json
dd health --json
```

**Pipe to jq for filtering**:
```bash
# Get only error traces
dd apm --json | jq '.traces[] | select(.error == true)'

# Get services with high error rate
dd health --json | jq '.services[] | select(.error_rate > 0.01)'
```

### Human-Readable (default)

```bash
# Conversational output for terminal use
dd health
dd apm
dd cost
```

---

## Help & Documentation

### Get Help

```bash
# General help
dd --help

# Command-specific help
dd apm --help
dd logs --help
dd metrics --help
```

### Available Commands

```bash
# List all 22 commands
dd --help
```

**Query Operations** (12):
- `context` - Auto-detect service from git
- `apm` - APM traces and performance
- `logs` - Log search and analysis
- `metrics` - Time-series metrics
- `security` - Security signals
- `slos` - SLO monitoring
- `watchdog` - Anomaly detection
- `database` - Database monitoring
- `catalog` - Service catalog
- `rum` - Real User Monitoring
- `network` - Network performance
- `cicd` - CI/CD Visibility

**Management Operations** (5):
- `monitors` - Monitor management
- `incidents` - Incident management
- `dashboards` - Dashboard management
- `workflows` - Workflow automation
- `synthetics` - Synthetic test management

**Smart Operations** (2):
- `health` - Multi-signal health check
- `deploy` - Deployment readiness

**FinOps** (2):
- `llm` - LLM observability
- `cost` - Cost analysis

**Utility** (1):
- `version` - Show version information

---

## Troubleshooting

### API Key Issues

```bash
# Verify credentials are set
echo $DD_API_KEY
echo $DD_APP_KEY

# Test API connection
dd context --json
```

**Error: "Invalid API key"**
- Check your API key at: https://app.datadoghq.com/organization-settings/api-keys
- Ensure you're using the correct Datadog site (DD_SITE)

### Service Not Detected

```bash
# Check what context is detected
dd context

# Manually specify service
dd apm --service my-service
```

### No Data Returned

**Check time range**:
```bash
# Default is last 15 minutes, try longer
dd apm --from 1h
dd logs --from 24h
```

**Check service name**:
```bash
# List all services in Datadog
dd catalog
```

---

## Performance

The Go implementation is significantly faster than the Python version:

- **Startup**: 3ms (67x faster)
- **Memory**: 10MB (67% less)
- **Binary Size**: 11-12MB (self-contained)
- **Dependencies**: 0 runtime dependencies

---

## Next Steps

### Learn More

- **README.md**: Complete feature list and examples
- **CONTRIBUTING.md**: Build from source, run tests
- **API_REFERENCE.md**: Full API documentation
- **GITHUB-SETUP-GUIDE.md**: Deploy your own instance

### Advanced Usage

- **CI/CD Integration**: Use `dd deploy` in pipelines
- **Monitoring Scripts**: Automate health checks
- **Custom Dashboards**: Combine with jq and other tools
- **Alerting**: Use exit codes for monitoring

### Get Involved

- **Report Issues**: https://github.com/yourusername/datadog-cli-go/issues
- **Contribute**: See CONTRIBUTING.md
- **Discussions**: https://github.com/yourusername/datadog-cli-go/discussions

---

## Quick Reference Card

```bash
# Setup
export DD_API_KEY="..." DD_APP_KEY="..."

# Most Used Commands
dd context              # Detect service
dd health               # Service health
dd deploy               # Deployment check
dd apm                  # Recent traces
dd logs --query "error" # Error logs
dd slos                 # SLO status

# JSON Output (for scripting)
dd health --json | jq '.status'

# Help
dd --help
dd apm --help
```

---

**Version**: v0.1.0
**Last Updated**: January 21, 2026
**Status**: Production Ready

For detailed documentation, see [README.md](README.md)
