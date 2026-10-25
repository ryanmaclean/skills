---
name: ergo-irc
description: Operate the fleet Ergo IRC server — check status, unban IPs, rehash config, diagnose connection issues. NEVER retry connections in a loop. Use when IRC agents can't connect, agents drop, or Ergo needs config changes.
trigger: ergo, irc server, irc ban, connection throttle, dline, kline, uban, rehash, irc config
---

# Ergo IRC Server — Fleet Operations

Container: `ergo-irc` on QNAS (10.0.2.35)
Hostname: `ergo.local` (preferred; resolves to 10.0.3.203 via fleet `/etc/hosts`, see `/Users/studio/rust-town/fleet-ops/docs/fleet-local-dns.md`)
Macvlan IP: `10.0.3.203` (use only for ARP/neigh/docker debugging, not for client connections)
Ports: 6667 (plaintext), 6697 (TLS)
Config: `/ircd/ircd.yaml` inside the container
Docker path on QNAS: `/share/ZFS530_DATA/.qpkg/container-station/usr/bin/.libs/docker`

## CRITICAL SAFETY RULES

1. **NEVER retry connections in a loop.** One attempt. Check result. Diagnose if it fails.
2. **NEVER kill and restart an IRC agent without checking how many instances are running first.**
3. **NEVER run `--version` or any flag on ii-agent that triggers a connection attempt.**
4. **If an agent has a built-in reconnect loop (ii-agent does, 120s interval), do NOT add manual retries on top.**
5. **If you get "NetworkUnreachable" or "No route to host", the problem is NETWORK not the agent. Diagnose the network.**
6. **Fleet subnet 10.0.3.0/23 is already exempted from throttling. If fleet agents still can't connect, it's not a ban — it's a network issue.**

## Quick Reference

```bash
# Docker command prefix (QNAS)
DOCKER="/share/ZFS530_DATA/.qpkg/container-station/usr/bin/.libs/docker"

# Check container status
ssh studio@10.0.2.35 "$DOCKER ps | grep ergo"

# View recent logs (excluding connection spam)
ssh studio@10.0.2.35 "$DOCKER logs ergo-irc --tail 50 2>&1 | grep -v 'connect-ip'"

# View connection attempts for a specific IP
ssh studio@10.0.2.35 "$DOCKER logs ergo-irc --tail 200 2>&1 | grep '10.0.2.61'"

# Read config
ssh studio@10.0.2.35 "$DOCKER exec ergo-irc cat /ircd/ircd.yaml"

# Rehash (reload config without restart)
ssh studio@10.0.2.35 "$DOCKER exec ergo-irc kill -HUP 1"

# List connected users (via container exec)
ssh studio@10.0.2.35 "$DOCKER exec ergo-irc cat /ircd/ircd.db" | strings | grep -i nick
```

## Current Configuration

```yaml
# Connection throttling
ip-limits:
  max-concurrent-connections: 16
  throttle-duration: 10m
  max-connections-per-duration: 64
  exempted:
    - "localhost"
    - "10.0.3.0/23"   # Fleet subnet — ALL fleet agents exempt

# Listeners
listeners:
  6667: plaintext
  6697: TLS (tls.crt / tls.key in /ircd/)
```

## Diagnosing Connection Issues

### Step 1: Is Ergo running?
```bash
ssh studio@10.0.2.35 "$DOCKER ps | grep ergo"
```

### Step 2: Can YOU reach it?
```bash
nc -z -w 3 10.0.3.203 6697 && echo "OK" || echo "FAIL"
```

### Step 3: Can the FAILING HOST reach it?
```bash
ssh <user>@<host> 'nc -z -w 3 10.0.3.203 6697 && echo "OK" || echo "FAIL"'
```

### Step 4: Is it an ARP/macvlan issue?
```bash
ssh <user>@<host> 'ping -c 1 -W 2 10.0.3.203'    # ICMP works?
ssh <user>@<host> 'ip neigh show 10.0.3.203'       # ARP resolves?
```

If ping works but TCP fails: likely a macvlan L2 issue. The container's MAC isn't being seen by the host.

### Step 5: Check Ergo logs for the failing IP
```bash
DOCKER="/share/ZFS530_DATA/.qpkg/container-station/usr/bin/.libs/docker"
ssh studio@10.0.2.35 "$DOCKER logs ergo-irc --tail 100 2>&1 | grep '<failing-ip>'"
```

### Step 6: Is the host running MULTIPLE agent instances?
```bash
ssh <user>@<host> 'ps aux | grep ii-agent | grep -v grep'
```
**If more than 1: kill ALL, wait 60s, start ONE.**

## Operator Commands (via IRC client)

```
/oper admin <password>           # Login as operator
/UBAN LIST                       # List all bans
/UBAN INFO <nick>                # Get connection details
/UBAN ADD <ip> [duration] [reason]   # Ban an IP
/UBAN REMOVE <ip>                # Remove a ban
/KILL <nick>                     # Force disconnect
/REHASH                          # Reload config
/DEFCON 4                        # Restrict registrations (under attack)
/DEFCON 5                        # Return to normal
```

## Editing Config

1. Copy config out: `ssh studio@10.0.2.35 "$DOCKER cp ergo-irc:/ircd/ircd.yaml /tmp/ircd.yaml"`
2. Edit: `scp studio@10.0.2.35:/tmp/ircd.yaml /tmp/ && vi /tmp/ircd.yaml`
3. Copy back: `scp /tmp/ircd.yaml studio@10.0.2.35:/tmp/ && ssh studio@10.0.2.35 "$DOCKER cp /tmp/ircd.yaml ergo-irc:/ircd/ircd.yaml"`
4. Rehash: `ssh studio@10.0.2.35 "$DOCKER exec ergo-irc kill -HUP 1"`

**What rehash CAN reload:** config, TLS certs, MOTD
**What rehash CANNOT reload:** listener ports, database backend (needs restart)

## MiSTer-Specific Notes

- MiSTer (de10nano, 10.0.2.61) runs as root — this is correct
- ii-agent binary is ARM 32-bit (armv7), at `/media/fat/linux/ii-agent-mister/ii-agent`
- The `--version` flag on this binary TRIGGERS A CONNECTION ATTEMPT — do not use it
- **MiSTer connects PLAINTEXT on port 6667 with `--no-tls`.** The ARM 32-bit TLS handshake fails. This is the correct config.
- Startup command: `nohup /media/fat/linux/ii-agent-mister/ii-agent -n mister-agent -s 10.0.3.203 -p 6667 --no-tls -c "#agents" --no-log > /tmp/mister-agent.log 2>&1 &`
- If MiSTer can ping 10.0.3.203 but TCP fails: macvlan ARP issue. Fix: `ip neigh replace 10.0.3.203 lladdr 02:42:d5:20:e4:f8 dev eth0 nud permanent`

## Macvlan Connectivity Issues

The Ergo container is on a macvlan network. Some hosts can't resolve the container's MAC address. Known workarounds:

1. **From UDM Pro:** Ping both the container IP and the failing host IP. This populates ARP tables on the gateway, which can help bridge the path.
2. **Gratuitous ARP from container:** `ssh studio@10.0.2.35 "$DOCKER exec ergo-irc arping -c 3 -A -I eth0 10.0.3.203"` (if arping available)
3. **Last resort:** Restart the container: `ssh studio@10.0.2.35 "$DOCKER restart ergo-irc"` — but this disconnects ALL users.

## References

- Ergo docs: https://github.com/ergochat/ergo/blob/master/docs/MANUAL.md
- Default config: https://raw.githubusercontent.com/ergochat/ergo/master/default.yaml
- Docker setup: https://github.com/ergochat/ergo/blob/master/distrib/docker/README.md
