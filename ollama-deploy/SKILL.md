---
name: ollama-deploy
description: Deploy and manage Ollama LLM on fleet hosts. Use when user mentions ollama, LLM, AI model, or language model deployment.
compatibility: opencode
metadata:
  tools_needed: "Bash, SSH, WebFetch, Glob, Grep, Read, Write"
  triggers: "ollama, LLM, AI model, language model"
---

# Ollama Deployment Skill

## Overview
Use this skill for deploying and managing Ollama (local LLM runtime) on fleet hosts.

## Tools Required
- **Bash** - Run ollama commands, systemctl
- **SSH** - Connect to hosts for direct access
- **WebFetch** - Download Ollama binaries, check releases
- **Glob/Grep** - Find existing Ollama installations
- **Read** - Check configs, model availability
- **Write/Edit** - Create systemd units, configs

## Key Principles

### 1. Architecture Matters
- Raspberry Pis: use `ollama-linux-arm64.tgz`
- x86_64 hosts: use `ollama-linux-amd64.tgz`
- Verify with: `file /usr/local/bin/ollama`

### 2. Staged Downloads for Large Files
Ollama binary is ~1.6GB. Download once locally, then distribute:
```bash
# Download to controller
curl -fsSL https://github.com/ollama/ollama/releases/download/v0.13.1/ollama-linux-arm64.tgz -o /tmp/ollama.tgz

# Distribute via Ansible copy (parallel, fast LAN)
ansible <hosts> -i inventory -b -m copy -a "src=/tmp/ollama.tgz dest=/tmp/ollama.tgz"

# Extract on hosts
ansible <hosts> -i inventory -b -m shell -a "tar -xzf /tmp/ollama.tgz -C /usr/local"
```

### 3. Models Need Separate Pull
After installing Ollama binary, pull models:
```bash
ansible <hosts> -i inventory -b -m shell -a "/usr/local/bin/ollama pull kimi-k2.5:cloud"
```

### 4. Use Systemd for Service Management
```bash
ansible <hosts> -i inventory -b -m systemd -a "name=ollama state=started enabled=yes"
```

## Common Playbooks

### Install Ollama
```bash
ansible-playbook -i inventory 02-ollama.yml
```

### Check Status
```bash
ansible <hosts> -i inventory -m shell -a "/usr/local/bin/ollama list"
ansible <hosts> -i inventory -m systemd -a "name=ollama state=started"
```

## Known Issues

### Wrong Architecture
- **Problem**: juju4.ollama role downloads AMD64 by default
- **Fix**: Manually specify ARM64 URL in playbook

### Model Download Fails
- **Problem**: Permission errors when running `ollama pull` as _ollama user
- **Fix**: Run `ollama pull` as ubuntu user or directly without become_user

### Service Not Starting
- **Check**: `journalctl -u ollama -n 50`
- **Verify binary**: `file /usr/local/bin/ollama`

## Reference Files
```
$HOME/rust-town/irclaw/ansible/02-ollama.yml
```

## Usage
1. **Audit first**: Check if Ollama already installed
2. **Stage download**: Get binary once, distribute
3. **Extract & start**: Use systemd
4. **Pull models**: Pull required models (kimi-k2.5:cloud)