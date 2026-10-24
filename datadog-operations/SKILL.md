---
name: datadog
description: Query Datadog APM traces, logs, metrics, SLOs, security signals, and service catalog. Create monitors, dashboards, and synthetic tests. Manage incidents and workflows. Use when investigating performance issues, searching logs, checking SLOs, analyzing costs, or automating Datadog operations.
---

# Datadog Operations Skill

Comprehensive Datadog automation: query APM/logs/metrics/RUM/database, create monitors/dashboards/synthetics, manage incidents, trigger workflows, analyze costs and LLM usage.

## Quick Setup

### macOS
```bash
./setup.sh
```

### Linux
```bash
./setup-linux.sh
```

### Windows (PowerShell as Administrator)
```powershell
.\setup-windows.ps1
```

## Platform-Specific Setup

### macOS

**Install jq** (required for JSON processing):
```bash
# Homebrew (recommended)
brew install jq

# MacPorts
sudo port install jq

# Direct binary (Apple Silicon)
curl -L -o ~/bin/jq https://github.com/jqlang/jq/releases/download/jq-1.7.1/jq-macos-arm64
chmod +x ~/bin/jq

# Direct binary (Intel)
curl -L -o ~/bin/jq https://github.com/jqlang/jq/releases/download/jq-1.7.1/jq-macos-amd64
chmod +x ~/bin/jq
```

**Set environment variables** in `~/.zshrc`:
```bash
export DD_API_KEY="your_api_key"
export DD_APP_KEY="your_application_key"
export DD_SITE="datadoghq.com"
```

### Linux

**Install dependencies:**
```bash
# Debian/Ubuntu
sudo apt-get install -y jq curl bc

# Fedora/RHEL
sudo dnf install -y jq curl bc

# Arch
sudo pacman -S jq curl bc
```

**Set environment variables** in `~/.bashrc`:
```bash
export DD_API_KEY="your_api_key"
export DD_APP_KEY="your_application_key"
export DD_SITE="datadoghq.com"
```

### Windows

**Option 1: WSL (Recommended)**
```bash
wsl --install
# Then run Linux setup inside WSL
./setup-linux.sh
```

**Option 2: Git Bash**
Install Git for Windows, then run `./setup.sh` in Git Bash.

**Option 3: Native PowerShell**
```powershell
# Install jq
winget install jqlang.jq
# Or: choco install jq

# Set environment variables
[Environment]::SetEnvironmentVariable("DD_API_KEY", "your_key", "User")
[Environment]::SetEnvironmentVariable("DD_APP_KEY", "your_app_key", "User")
```

Use Python scripts directly: `python python\script_name.py`

## Required Permissions

Get keys from Datadog → Organization Settings → API Keys / Application Keys

**For workflow automation (`trigger-workflow`):**
Enable "Actions API Access" on your app key:
Datadog → Organization Settings → Application Keys → Click key → Enable "Actions API Access"

## Quick Reference

### Investigation Scripts

| Script | Purpose | Example |
|--------|---------|---------|
| `query-apm.sh` | Find slow endpoints | `bash scripts/query-apm.sh --service my-service --duration 1h` |
| `search-logs.sh` | Search logs for errors | `bash scripts/search-logs.sh --query "status:error" --duration 1h` |
| `query-security-signals.sh` | Find security threats | `bash scripts/query-security-signals.sh --severity critical` |
| `query-watchdog.sh` | Anomaly detection | `bash scripts/query-watchdog.sh --service my-service` |
| `query-metrics.sh` | Fetch metrics data | `bash scripts/query-metrics.sh --metric "system.cpu.user"` |
| `analyze-usage-cost.sh` | FinOps cost analysis | `bash scripts/analyze-usage-cost.sh --duration 30d` |
| `analyze-llm.sh` | LLM observability | `bash scripts/analyze-llm.sh --service my-llm-app` |
| `query-slos.sh` | SLO status | `bash scripts/query-slos.sh --service payment-api` |
| `query-service-catalog.sh` | Service metadata | `bash scripts/query-service-catalog.sh list` |
| `query-database.sh` | DB performance | `bash scripts/query-database.sh --host postgres-prod` |
| `query-rum.sh` | Frontend performance | `bash scripts/query-rum.sh --application abc-123` |
| `query-kubernetes.sh` | K8s workloads | `bash scripts/query-kubernetes.sh --cluster prod` |
| `query-containers.sh` | Container metrics | `bash scripts/query-containers.sh --duration 1h` |
| `query-network.sh` | Network monitoring | `bash scripts/query-network.sh --duration 1h` |

### Automation Scripts

| Script | Purpose | Example |
|--------|---------|---------|
| `manage-monitors.sh` | Create/mute monitors | `bash scripts/manage-monitors.sh list` |
| `create-dashboard.sh` | Generate dashboards | `bash scripts/create-dashboard.sh --service my-service` |
| `trigger-workflow.sh` | Execute workflows | `bash scripts/trigger-workflow.sh list` |
| `manage-incidents.sh` | Incident management | `bash scripts/manage-incidents.sh list` |
| `manage-synthetics.sh` | Synthetic tests | `bash scripts/manage-synthetics.sh list` |
| `manage-on-call.sh` | On-call scheduling | `bash scripts/manage-on-call.sh list` |
| `manage-status-pages.sh` | Status pages | `bash scripts/manage-status-pages.sh list` |
| `verify-setup.sh` | Validate config | `bash scripts/verify-setup.sh` |

## Workflows

### Investigate Production Issue
```bash
bash scripts/query-watchdog.sh --service affected-service --duration 24h
bash scripts/query-apm.sh --service affected-service --duration 1h
bash scripts/search-logs.sh --service affected-service --status error --duration 1h
```

### Check Cost & Usage
```bash
bash scripts/analyze-usage-cost.sh --duration 30d --product all
bash scripts/analyze-usage-cost.sh --duration 30d | jq '.recommendations[] | select(.priority == "high")'
```

### Monitor LLM Application
```bash
bash scripts/analyze-llm.sh --service my-genai-app --duration 24h
```

## Output Format

All scripts return structured JSON to stdout, status messages to stderr:
```bash
bash scripts/query-apm.sh --service my-service 2>/dev/null | jq '.summary'
```

## Python Alternative

Python versions available in `python/` directory:
```bash
python3 -m venv .venv && source .venv/bin/activate
pip install -r python/requirements.txt
python python/query_apm.py --service my-service --json
```

## Go CLI Alternative

Single-binary CLI available in `../dd-skill-test-go/`:
```bash
# Native on all platforms - no dependencies
datadog-cli apm --service my-service --duration 1h
datadog-cli logs --query "status:error"
datadog-cli monitors list
datadog-cli health
```

See `../dd-skill-test-go/README.md` for installation.

## Notes

- **Advanced Scripts:** Some scripts require Python. Run `./setup.sh` first.
- **Windows:** Use WSL, Git Bash, or Python scripts directly.
- **Go CLI:** Recommended for Windows - native binary, no dependencies.
