---
name: unifi-ssh
description: Ubiquiti Unifi Dream Machine Pro management via SSH. Use when managing UDM Pro (10.0.3.5) — checking DHCP leases, network devices, client info, device status, logs, firewall, VPN, Protect storage, or rebooting. Commands use `ssh unifi` (host alias for root@10.0.3.5). Triggered by phrases like "unifi", "UDM", "DHCP leases", "network clients", "dream machine", "unifi devices", "check the network", "firewall logs", "VPN status", "protect storage".
---

# Unifi SSH — Complete Reference

Host alias `unifi` → `root@10.0.3.5`

## System Monitoring

```bash
# System info (model, firmware, MAC, public IP, uptime)
ssh unifi "info"

# CPU temp
ssh unifi "ubnt-systool cputemp"

# CPU load
ssh unifi "ubnt-systool cpuload"

# All sensors (temp, fan, voltage)
ssh unifi "sensors"

# System summary (hardware, software versions)
ssh unifi "ubnt-device-info summary"

# Uptime
ssh unifi "uptime"

# Disk usage (data partition)
ssh unifi "du -hs /data"
```

## Network Discovery

```bash
# Find other Unifi devices on LAN
ssh unifi "ubnt-tools ubnt-discover"

# ARP table
ssh unifi "arp -a"

# IPv6 neighbors
ssh unifi "ip neigh"

# Routing table
ssh unifi "netstat -rt -n"

# Interface stats
ssh unifi "ifstat"

# All interfaces
ssh unifi "ifconfig"

# Port status (PoE, link speed)
ssh unifi "ubnt-systool portstatus"
```

## DHCP & DNS

```bash
# DHCP leases with hostnames
ssh unifi "cat /data/udapi-config/dnsmasq.lease"

# Static DHCP config
ssh unifi "cat /run/dnsmasq.dhcp.conf.d/dhcp.dhcpServers-net_Default_br0_10-0-2-0-23.conf"

# Active DNS entries
ssh unifi "cat /run/dnsmasq.dns.conf.d/hosts.d/leases"
```

## VPN

```bash
# IPSec VPN status
ssh unifi "ipsec statusall"

# WireGuard/Teleport status
ssh unifi "wg show"
```

## Firewall

```bash
# Live firewall logs (all)
ssh unifi "tail -f /var/log/ulog/syslogemu.log"

# Live firewall logs (denied only)
ssh unifi "tail -f /var/log/ulog/syslogemu.log | grep Deny"

# NAT rules with line numbers
ssh unifi "iptables -t nat -L PREROUTING --line"
```

## UniFi Protect

```bash
# Protect recordings disk usage
ssh unifi "du -hs /volume1/.srv/unifi-protect/video/ 2>/dev/null || du -hs /data/unifi-protect/ 2>/dev/null"

# List largest files in Protect storage
ssh unifi "du -a /volume1 2>/dev/null | sort -n -r | head -n 20 || echo 'No volume1'"
```

## Services

```bash
# Restart UniFi OS web interface
ssh unifi "/etc/init.d/S95unifios restart"

# Restart UniFi controller
ssh unifi "systemctl restart unifi"

# Restart Protect
ssh unifi "systemctl restart unifi-protect"

# Restart entire UniFi OS
ssh unifi "unifi-os restart"

# Reboot UDM
ssh unifi "reboot"

# Shutdown UDM
ssh unifi "poweroff"
```

## Logs

```bash
# System log
ssh unifi "cat /mnt/data/unifi-os/unifi-core/logs/system.log"

# UniFi server log
ssh unifi "cat /mnt/data/unifi-os/unifi/logs/server.log"

# HTTP error log
ssh unifi "cat /mnt/data/unifi-os/unifi-core/logs/errors.log"

# Discovery log
ssh unifi "cat /mnt/data/unifi-os/unifi-core/logs/discovery.log"

# General messages
ssh unifi "cat /var/log/messages"
```

## Live Log Monitoring

```bash
# Tail system log
ssh unifi "tail -f /var/log/messages"

# Tail UniFi server log
ssh unifi "tail -f /mnt/data/unifi-os/unifi/logs/server.log"
```

## Scripts

Helper scripts are in `scripts/`:
- `get-leases.sh` — formatted DHCP leases
- `sysinfo.sh` — system info summary
- `devices.sh` — discover Unifi devices on network

Run with: `~/.agents/skills/unifi-ssh/scripts/<script>.sh`

## Notes
- SSH alias configured in `~/.ssh/config` as `unifi` → `root@10.0.3.5`
- StrictHostKeyChecking is enabled — first connection may prompt to accept host key
- UDM runs dnsmasq for DHCP/DNS — leases at `/data/udapi-config/dnsmasq.lease`
- UniFi API available locally on port 9443 (no SSL verify) via `localhost:9443/api/...`
