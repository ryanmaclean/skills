---
name: ansible-fleet
description: Deploy and manage software on Linux fleet hosts (Pis, NAS, workstations) via Ansible. Use when user mentions deploy, ansible, fleet, hosts, inventory, or deployment.
compatibility: opencode
metadata:
  tools_needed: "Bash, SSH, Glob, Grep, Read, Write, WebFetch"
  triggers: "ansible, fleet, deploy, inventory, playbook"
---

# Ansible Fleet Deployment Skill

## Overview
Use this skill for deploying and managing software on fleet of Linux hosts (Raspberry Pis, NAS, workstations) via Ansible.

## Tools Required
- **Bash** - Run ansible-playbook, ansible commands
- **SSH** - Connect to remote hosts
- **Glob/Grep** - Find playbook files, inventory
- **Read** - Examine existing playbooks, inventory
- **Write/Edit** - Create/update playbooks
- **WebFetch** - Research Ansible Galaxy roles, documentation

## Key Principles

### 1. ALWAYS Audit First
Before any changes, run Ansible facts to understand current state:
```bash
ansible <hosts> -i inventory -m setup
ansible <hosts> -i inventory -m package_facts
ansible <hosts> -i inventory -m service_facts
ansible <hosts> -i inventory -m shell -a "which <software>"
```

### 2. Use Separate Playbooks Per Component
Never try to do everything in one playbook. Create separate playbooks:
- `01-datadog.yml` - Datadog Agent
- `02-ollama.yml` - Ollama LLM
- `03-irclaw.yml` - irclaw gateway

### 3. Handle Architecture Correctly
- Raspberry Pis are ARM64 (aarch64), not AMD64
- Use `ollama-linux-arm64.tgz` for Pis, `ollama-linux-amd64.tgz` for x86_64
- Detect with: `ansible -m setup -a "filter=ansible_architecture"`

### 4. Stage Large Downloads
For large files (1GB+), download once to controller, then distribute:
```bash
# Download to controller
curl -fsSL <url> -o /tmp/file.tgz

# Distribute to hosts via Ansible copy
ansible <hosts> -i inventory -b -m copy -a "src=/tmp/file.tgz dest=/tmp/file.tgz"
```

### 5. Use Official Installers
irclaw installer handles Node.js automatically:
```bash
curl -fsSL https://irclaw.ai/install.sh | bash -s -- --no-onboard
```

### 6. Use Ansible Galaxy Roles Where Available
- `datadog.dd.agent` - Datadog Agent collection
- `juju4.ollama` - Ollama role (but fix ARM64 URL manually)
- `nodesource.node` - Node.js (supports 18.x, 20.x, 22.x, 23.x)
- `irclaw.installer.irclaw` - irclaw collection

## Common Playbook Patterns

### Deploy to Multiple Hosts (Parallel)
```bash
ansible-playbook -i inventory <playbook.yml>
```

### Deploy to Specific Hosts
```bash
ansible-playbook -i inventory <playbook.yml> --limit "host1,host2,host3"
```

### Check What's Installed
```bash
ansible <hosts> -i inventory -m command -a "which <software>"
ansible <hosts> -i inventory -m package_facts
```

### Service Management
```bash
ansible <hosts> -i inventory -b -m systemd -a "name=<service> state=started enabled=yes"
```

## Known Issues & Fixes

### Ollama Wrong Architecture
- **Problem**: juju4.ollama role downloads AMD64 by default
- **Fix**: Override URL: `ollama_url: "https://github.com/ollama/ollama/releases/download/v0.13.1/ollama-linux-arm64.tgz"`

### Node.js 24 Not Supported by Ansible Roles
- **Problem**: nodesource.node role only supports up to 23.x
- **Fix**: Use official irclaw installer which handles Node.js automatically

### Model Download Fails with become_user
- **Problem**: Permission errors when downloading Ollama models as _ollama user
- **Fix**: Run `ollama pull` directly without become_user

### Long Timeouts
- **Problem**: Downloading large files times out
- **Fix**: Use staged download approach, or increase Ansible timeout

## Reference Files

```
$HOME/rust-town/irclaw/ansible/
├── inventory              # Fleet inventory
├── 01-datadog.yml         # Datadog deployment
├── 02-ollama.yml          # Ollama deployment  
├── 03-irclaw.yml          # irclaw deployment
└── requirements.yml       # Ansible Galaxy requirements
```

## Inventory Groups
- `[pis]` - Raspberry Pis (ARM64)
- `[nas]` - NAS devices
- `[all:vars]` - Global variables

## Usage

1. **Audit first**: Check what's already installed
2. **Create/fix playbooks**: Based on audit results
3. **Run playbooks**: Use `--limit` to test, then full fleet
4. **Verify**: Check installation worked