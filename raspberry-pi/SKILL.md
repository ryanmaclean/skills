---
name: raspberry-pi
description: Manage Raspberry Pi fleet - OS updates, Python upgrades, ARM64-specific configurations. Use when user mentions pi, raspberry, rpi, ubuntu on pi, or ARM64.
compatibility: opencode
metadata:
  tools_needed: "Bash, SSH, Glob, Grep, Read, Write"
  triggers: "raspberry, pi, rpi, arm64, ubuntu on pi"
---

# Raspberry Pi Management Skill

## Overview
Use this skill for managing Raspberry Pis in a fleet - specifically OS updates, Python upgrades, and hardware-specific configurations.

## Tools Required
- **Bash** - Run ansible, apt, ssh commands
- **SSH** - Connect to Pis for direct access
- **Glob/Grep** - Find Pi inventory, scripts
- **Read** - Check OS versions, configs
- **Write/Edit** - Update sources.list, configs

## Critical Differences from x86

### 1. Architecture
- Pis are ARM64 (aarch64), NOT AMD64
- Always verify: `ansible -m setup -a "filter=ansible_architecture"`
- Use ARM64 binaries: `ollama-linux-arm64.tgz`, NOT `.amd64`

### 2. Ubuntu Release Upgrades
In-place upgrades from 20.04 → 22.04 are RISKY on Pis:
- Can break SSH (openssh-server removed during upgrade)
- Can break package manager (dpkg in inconsistent state)
- Can lose network connectivity mid-upgrade

**SAFE Upgrade Method (use this, not do-release-upgrade):**
```bash
# Manual release upgrade (works without TTY)
sed -i 's/focal/jammy/g' /etc/apt/sources.list
apt-get update
DEBIAN_FRONTEND=noninteractive apt-get dist-upgrade -y
```

### 3. Python Version
- Ubuntu 20.04 ships Python 3.8
- Ubuntu 22.04 ships Python 3.10
- Ansible needs Python 3.9+ on target (use raw module for older)
- For native Ollama, use raw module or upgrade OS

### 4. Docker on ARM64
- Docker works on ARM64 but use `arm64` tags
- Pre-built images may not have arm64 variants
- Native installation often better than Docker on Pi

## Safe Update Workflow

### Step 1: ALWAYS Audit First
```bash
# Check connectivity
ansible <hosts> -i inventory -m ping

# Check OS version
ansible <hosts> -i inventory -m setup -a "filter=ansible_distribution*"

# Check Python version
ansible <hosts> -i inventory -m command -a "python3 --version"

# Check disk space
ansible <hosts> -i inventory -m command -a "df -h"
```

### Step 2: For Python < 3.9, Use Raw Module
```bash
# Use raw to update apt (works with any Python)
ansible <hosts> -i inventory -b -m raw -a "apt-get update"

# Or upgrade OS to get newer Python
ansible <hosts> -i inventory -b -m raw -a "sed -i 's/focal/jammy/g' /etc/apt/sources.list && apt-get update"
```

### Step 3: Verify After Changes
```bash
ansible <hosts> -i inventory -m setup -a "filter=ansible_distribution*"
ansible <hosts> -i inventory -m command -a "python3 --version"
```

## Common Issues & Solutions

### SSH Broken After Upgrade
- **Cause**: openssh-server removed but not reinstalled
- **Solution**: Physical access (keyboard/HDMI) or reimage SD card

### Package Manager Broken
- **Cause**: dpkg in inconsistent state after interrupted upgrade
- **Solution**: `sudo dpkg --configure -a` or reimage

### Wrong Architecture Binaries
- **Cause**: Downloading amd64 instead of arm64
- **Fix**: Always verify with `file /usr/bin/binary`

### Network Timeout During Download
- **Cause**: Large file download interrupted
- **Fix**: Stage locally, distribute via Ansible copy

## Raspberry Pi Specific Inventory Variables

```yaml
[pis]
ubrpi41 ansible_host=10.0.3.215 ansible_user=ubuntu
# ...

[pis:vars]
ansible_python_interpreter=/usr/bin/python3
# Pis typically need longer timeouts
ansible_timeout=60
```

## Inventory Location
```
$HOME/rust-town/irclaw/ansible/inventory
```

## Usage

1. **Audit first**: Check OS, Python, disk space
2. **Use raw for Python < 3.9**: Never wait for AnsibleFacts
3. **Test on ONE Pi first**: Before rolling out to fleet
4. **Have recovery plan**: SD card imager ready