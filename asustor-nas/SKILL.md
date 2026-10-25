# Asustor NAS (ANAS) Management Skill

## When to Use

Use when managing the Asustor AS6510T NAS: Docker containers, networking,
storage, ADM configuration, or fleet agent deployment. Trigger phrases:
"asustor", "anas", "10.0.3.9", "AS6510T".

## Host Details

| Field | Value |
|-------|-------|
| Hostname | anas |
| IP | 10.0.3.9 (eth0-br, /23 network) |
| SSH | root@10.0.3.9 |
| Model | Asustor AS6510T |
| CPU | Intel Atom C3538 (4-core Denverton) |
| RAM | 16 GB |
| OS | ADM 5.1.1.RCI1 (Linux 6.6.x) |
| Disks | 10 bays |
| Web UI | https://10.0.3.9:5443 (TLS 1.3, HTTPS-only) |

## Network Interfaces

| Interface | Type | IP | Notes |
|-----------|------|-----|-------|
| eth0 | 2.5G Realtek | bridged to eth0-br | Primary LAN |
| eth1 | 2.5G Realtek | NO-CARRIER | Unused |
| eth2 | 10G Intel X553 | 10.0.2.1/24 | **ROUTE CONFLICT -- see below** |
| eth3 | 10G Intel X553 | 10.0.11.1/24 | Secondary network |
| eth0-br | Bridge (eth0) | 10.0.3.9/23 | Default gateway via 10.0.3.5 |
| nat-br | NAT bridge | 10.172.5.1/24 | Docker NAT |

## Critical: The /24 vs /23 Route Fix

The LAN is 10.0.2.0/23 (10.0.2.0 - 10.0.3.255). ADM's `netmand` assigns
eth2 the address 10.0.2.1/24, which creates a kernel route:

    10.0.2.0/24 dev eth2

This /24 route is *more specific* than the /23 on eth0-br, so all traffic to
10.0.2.x hosts (i9-zfs-pop at 10.0.2.230, workstations, etc.) gets routed to
eth2 instead of eth0-br -- which is wrong because eth2 is either disconnected
or on a different physical segment.

**Fix (immediate):**
```
ip route del 10.0.2.0/24 dev eth2
```

**Fix (persistent):** A startup script at `/usr/local/etc/init.d/S99fix-routes`
runs after ADM boot to delete the conflicting route. The script sleeps 10s to
let netmand finish, then removes the route and logs via syslog.

**Root cause:** ADM's network manager (`netmand`) re-creates the route on every
boot based on `/etc/nas.conf` [Network] section where DefaultGateway=Lan3.
The eth2 /24 assignment is baked into ADM config. Changing it via ADM UI
risks breaking the 10G link. The startup script is the safest workaround.

## ADM Configuration

- Config file: `/etc/nas.conf` (INI format)
- Network daemon: `netmand` (manages interfaces, routes, DNS)
- DefaultGateway setting: `Lan3` (maps to eth0-br bridge)
- Apps installed via App Central: docker-ce, portainer-ce, jellyfin, plex,
  virtualbox, git, python3, php8.x, letsencrypt, emby, unifi-network-app

## Docker Management

Access via SSH as root. Docker CE is installed from App Central.

```bash
# List containers
ansible anas -i ansible/inventory -m raw -a 'docker ps'

# Restart a container
ansible anas -i ansible/inventory -m raw -a 'docker restart CONTAINER'

# View logs
ansible anas -i ansible/inventory -m raw -a 'docker logs --tail 50 CONTAINER'
```

**Running containers (typical):**
- `irclaw` -- fleet IRC agent (irclaw:v0.7.7), nick=asus-x86-01-polecat
- `ollama` -- local LLM inference (ollama/ollama)
- `datadog-agent` -- monitoring (datadog/agent:7)
- `ImmichServerDocker` -- photo management (port 22283)
- `ImmichPostgresVEC` -- Immich Postgres with pgvectors
- `ImmichRedisD` -- Immich Valkey/Redis
- `ImmichMachineLearningD` -- Immich ML
- `Jellyfin` -- media server (ports 28096, 28920)
- `UnifiNetworkApp` -- Unifi controller (port 29443)
- `PortainerCE` -- Docker UI (port 19943)
- `mongodb` -- MongoDB for Unifi

## Startup Scripts

ADM uses `/usr/local/etc/init.d/` for startup ordering:
- `S*` -- start scripts (run at boot, ascending order)
- `K*` -- stop scripts (run at shutdown, ascending order)
- Format: `S##name` or `K##name` -> symlinks to AppCentral CONTROL scripts
- Custom scripts: place as `S99yourscript`, must be `chmod +x`

**Our custom scripts:**
- `S99fix-routes` -- deletes the conflicting 10.0.2.0/24 eth2 route

## Ansible Usage

ANAS uses `raw` module only (no Python on ADM's BusyBox):

```bash
# One-off command
ansible anas -i ansible/inventory -m raw -a 'COMMAND'

# IMPORTANT: Avoid {{ }} in -a strings (Jinja conflict)
# Use printf or simple commands instead of heredocs with {{ }}
```

## Warnings

1. **ADM firmware updates may reset `/usr/local/etc/init.d/`** -- verify
   S99fix-routes survives after any ADM update.
2. **netmand overwrites routes** -- do not try to fix via `/etc/nas.conf`
   Network section; use the startup script instead.
3. **No Python** -- only `raw` and `script` Ansible modules work.
4. **HTTPS-only** -- ADM web UI requires TLS on port 5443.
5. **Never guess passwords** -- if SSH auth fails, stop and ask the user.
