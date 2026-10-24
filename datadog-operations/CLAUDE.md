# Datadog Skills Project

Comprehensive Datadog automation skills for Claude Code - available in **Bash**, **Python**, and **Go**.

## Quick Start

### macOS
```bash
./setup.sh
```

### Linux (Ubuntu/Debian/RHEL/Fedora/Arch/openSUSE)
```bash
./setup-linux.sh
```

### Windows (PowerShell as Administrator)
```powershell
.\setup-windows.ps1
```

### Go CLI (All Platforms - Recommended)
```bash
# macOS (Homebrew)
brew install datadog-cli

# Or download directly - see dd-skill-test-go/README.md
```

---

## Platform-Specific Setup

### macOS

**Automated setup:**
```bash
./setup.sh
```

**Manual setup:**

1. **Install jq** (choose one):
```bash
# Homebrew (recommended)
brew install jq

# MacPorts
sudo port install jq

# Direct download (Apple Silicon)
mkdir -p ~/bin
curl -L -o ~/bin/jq https://github.com/jqlang/jq/releases/download/jq-1.7.1/jq-macos-arm64
chmod +x ~/bin/jq
export PATH="$HOME/bin:$PATH"

# Direct download (Intel)
curl -L -o ~/bin/jq https://github.com/jqlang/jq/releases/download/jq-1.7.1/jq-macos-amd64
chmod +x ~/bin/jq
```

2. **Set environment variables** in `~/.zshrc`:
```bash
export DD_API_KEY="your_api_key"
export DD_APP_KEY="your_application_key"
export DD_SITE="datadoghq.com"
```

3. **Python dependencies** (for advanced scripts):
```bash
python3 -m venv .venv
source .venv/bin/activate
pip install -r python/requirements.txt
```

---

### Linux

**Automated setup:**
```bash
./setup-linux.sh
```

Supports: Ubuntu, Debian, Fedora, RHEL, CentOS, Arch, openSUSE

**Manual setup:**

1. **Install dependencies:**
```bash
# Debian/Ubuntu
sudo apt-get update && sudo apt-get install -y jq curl bc

# Fedora/RHEL 8+
sudo dnf install -y jq curl bc

# RHEL 7/CentOS 7
sudo yum install -y jq curl bc

# Arch Linux
sudo pacman -S jq curl bc

# openSUSE
sudo zypper install -y jq curl bc
```

2. **Set environment variables** in `~/.bashrc` or `~/.zshrc`:
```bash
export DD_API_KEY="your_api_key"
export DD_APP_KEY="your_application_key"
export DD_SITE="datadoghq.com"
```

3. **Python dependencies:**
```bash
python3 -m venv .venv
source .venv/bin/activate
pip install -r python/requirements.txt
```

**Linux-specific notes:**
- The `date` command uses GNU syntax (compatible with all scripts)
- `bc` is required for floating-point arithmetic in some scripts

---

### Windows

**Option 1: PowerShell Setup (Native)**
```powershell
# Run as Administrator
.\setup-windows.ps1
```

**Option 2: WSL (Recommended for Bash scripts)**
```bash
# Install WSL
wsl --install

# Then run Linux setup inside WSL
./setup-linux.sh
```

**Option 3: Git Bash**
```bash
# Install Git for Windows (includes Git Bash)
# https://git-scm.com/download/win

# Run in Git Bash
./setup.sh
```

**Manual Windows setup:**

1. **Install jq:**
```powershell
# winget (Windows 11+)
winget install jqlang.jq

# Chocolatey
choco install jq

# Scoop
scoop install jq

# Manual download
Invoke-WebRequest -Uri "https://github.com/jqlang/jq/releases/download/jq-1.7.1/jq-windows-amd64.exe" -OutFile "$env:USERPROFILE\bin\jq.exe"
```

2. **Set environment variables:**
```powershell
# Session only
$env:DD_API_KEY = "your_api_key"
$env:DD_APP_KEY = "your_application_key"
$env:DD_SITE = "datadoghq.com"

# Permanent (User level)
[Environment]::SetEnvironmentVariable("DD_API_KEY", "your_api_key", "User")
[Environment]::SetEnvironmentVariable("DD_APP_KEY", "your_application_key", "User")
[Environment]::SetEnvironmentVariable("DD_SITE", "datadoghq.com", "User")
```

3. **Python dependencies:**
```powershell
python -m venv .venv
.\.venv\Scripts\Activate.ps1
pip install -r python\requirements.txt
```

**Windows-specific notes:**
- Bash scripts require WSL or Git Bash
- Python scripts work natively in PowerShell
- Go CLI (`datadog-cli.exe`) works natively - **recommended for Windows**

---

## Go CLI (Cross-Platform - Recommended)

The Go CLI provides native binaries for all platforms with zero dependencies:

| Platform | Binary | Installation |
|----------|--------|--------------|
| macOS (Apple Silicon) | `datadog-cli-darwin-arm64` | `brew install datadog-cli` |
| macOS (Intel) | `datadog-cli-darwin-amd64` | `brew install datadog-cli` |
| Linux (x64) | `datadog-cli-linux-amd64` | `apt/dnf/snap install datadog-cli` |
| Linux (ARM) | `datadog-cli-linux-arm64` | Download from releases |
| Windows (x64) | `datadog-cli-windows-amd64.exe` | `choco install datadog-cli` |
| Windows (ARM) | `datadog-cli-windows-arm64.exe` | Download from releases |

**Pre-built binaries available in:** `../dd-skill-test-go/bin/`

See `../dd-skill-test-go/README.md` for full Go CLI documentation.

---

## API Key Permissions

**Required for all operations:**
- API Key: Organization Settings → API Keys
- App Key: Organization Settings → Application Keys

**For workflow automation (`trigger-workflow`):**
Enable "Actions API Access" on your app key:
1. Go to Datadog → Organization Settings → Application Keys
2. Click your app key
3. Enable "Actions API Access"

---

## Available Scripts

### Bash Scripts (65 scripts in `scripts/`)
```bash
# Investigation
bash scripts/query-apm.sh --service my-service --duration 1h
bash scripts/search-logs.sh --query "status:error" --duration 1h
bash scripts/query-metrics.sh --metric system.cpu.user --duration 24h

# Management
bash scripts/manage-monitors.sh list
bash scripts/manage-incidents.sh list
bash scripts/create-dashboard.sh --service my-service --type apm

# Automation
bash scripts/trigger-workflow.sh list
bash scripts/verify-setup.sh
```

### Python Scripts (65 scripts in `python/`)
```bash
source .venv/bin/activate
python python/query_apm.py --service my-service --duration 1h --json
python python/search_logs.py --query "status:error" --json
python python/manage_monitors.py list --json
```

### Go CLI (55 commands)
```bash
datadog-cli apm --service my-service --duration 1h
datadog-cli logs --query "status:error" --duration 1h
datadog-cli monitors list
datadog-cli health
datadog-cli deploy
```

---

## Testing

### Bash/Python Test Harness
```bash
# Read-only tests (safe)
./test-skills.sh

# Write operations test (creates/deletes test resources)
./test-write-operations.sh --confirm
```

### Go Tests
```bash
cd ../dd-skill-test-go
go test ./...                              # Unit tests
./tests/integration/test-all-commands.sh   # Integration tests
```

---

## Feature Parity

| Feature | Bash | Python | Go |
|---------|------|--------|-----|
| APM queries | ✅ | ✅ | ✅ |
| Log search | ✅ | ✅ | ✅ |
| Metrics | ✅ | ✅ | ✅ |
| SLOs | ✅ | ✅ | ✅ |
| Monitors | ✅ | ✅ | ✅ |
| Incidents | ✅ | ✅ | ✅ |
| Synthetics | ✅ | ✅ | ✅ |
| Dashboards | ✅ | ✅ | ✅ |
| Workflows | ✅ | ✅ | ✅ |
| Security signals | ✅ | ✅ | ✅ |
| Cost analysis | ✅ | ✅ | ✅ |
| LLM observability | ✅ | ✅ | ✅ |
| **Total scripts** | **65** | **65** | **55** |

---

## Troubleshooting

### "jq: command not found"
Install jq using the platform-specific instructions above.

### "DD_API_KEY not set"
Set environment variables and reload your shell:
```bash
source ~/.zshrc   # macOS
source ~/.bashrc  # Linux
```

### "API credentials invalid"
1. Verify keys at Datadog → Organization Settings
2. Ensure DD_SITE matches your Datadog region
3. Run `bash scripts/verify-setup.sh` to diagnose

### Windows: "Scripts not running"
- Use WSL or Git Bash for Bash scripts
- Use Python scripts natively: `python python/script_name.py`
- Use Go CLI: `datadog-cli.exe command`

### Python: "externally-managed-environment" error
Use a virtual environment:
```bash
python3 -m venv .venv
source .venv/bin/activate
pip install -r python/requirements.txt
```

---

## Project Structure

```
dd-skill-test/
├── .claude/skills/datadog-operations/SKILL.md  # Claude Code skill
├── scripts/                    # 65 Bash scripts
│   ├── lib/                   # Shared libraries
│   └── *.sh                   # Executable scripts
├── python/                     # 65 Python scripts
│   ├── lib/                   # Shared modules
│   └── *.py                   # Executable scripts
├── setup.sh                    # macOS setup
├── setup-linux.sh              # Linux setup
├── setup-windows.ps1           # Windows setup
├── test-skills.sh              # Read-only test harness
└── test-write-operations.sh    # Write operations test

dd-skill-test-go/               # Go CLI (sibling directory)
├── bin/                        # Pre-built binaries
├── internal/commands/          # 55 commands
└── README.md                   # Full documentation
```
