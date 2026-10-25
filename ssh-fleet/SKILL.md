---
name: ssh-fleet
description: SSH hardening for the fleet — ControlMaster config to prevent exit-255 drops in background shells, ServerAliveInterval, the no-guessing-passwords policy, and when to use raw SSH vs Ansible. Use when SSH fails, drops, or when setting up access to fleet hosts.
trigger: ssh, exit 255, controlmaster, ssh drop, fleet access, ssh config
---

# SSH Fleet Access

## Auth policy (MANDATORY)

1. **Never guess, infer, or try passwords.** First auth failure → STOP, ask the user.
2. **Prefer Ansible for fleet ops.** It has the correct keys, users, and sudo config.
3. **No password changes on fleet devices, ever.** Locks the user out.
4. **Network gear (UDM Pro, switches, APs) = explicit approval before any command.**

```bash
# One-off command (preferred over raw ssh)
ansible HOST -i ansible/inventory -m shell -a "CMD"
```

## `~/.ssh/config` for the fleet

```
Host <fleet-host-glob>
    ServerAliveInterval 30
    ServerAliveCountMax 3
    ControlMaster auto
    ControlPath ~/.ssh/cm-%r@%h:%p
    ControlPersist 10m
```

`ControlMaster auto` is what prevents the exit-255 drop pattern — a second SSH to
the same host reuses the existing multiplex channel instead of tearing down.

## Exit 255 — root causes

| Cause | Signal | Fix |
|-------|--------|-----|
| Background bash shell in agent harness, stdin closed on return | SSH dies mid-session | Use `ControlMaster` or dispatch via `ansible` |
| Wifi drop on Pi | Random, sporadic | Retry once; if persists, escalate to user |
| Wrong key | Immediate | Check `ansible/inventory` `ansible_ssh_private_key_file`; NEVER guess |
| Host unreachable (macvlan) | `Connection refused` / timeout | Hop via a bastion host — macvlan IPs are often not reachable directly from macOS |

## Raw SSH vs Ansible decision matrix

| Situation | Use |
|-----------|-----|
| Known, interactive debugging | Raw SSH (ControlMaster on) |
| Fleet-wide config change | Ansible playbook, `serial: 1`, `--limit` first |
| One-off command on one host | `ansible HOST -m shell -a "..."` |
| Deploying secrets | Ansible with `no_log: true` |
| UDM Pro / switches / APs | Neither — ask user first |

## Common mistakes

| Mistake | Fix |
|---------|-----|
| Retrying with a guessed password after failure | STOP, ask user |
| Background SSH to a flaky host exits 255 mid-command | Use ControlMaster or ansible |
| Testing a playbook on all hosts at once | `--limit ONE_HOST` first (FIFO brick incident) |
| SSH to macvlan IP from the wrong network | Hop through a bastion host on the macvlan subnet |
| Changing a password "to simplify" | NEVER — gets lost to context, locks user out |

## Related

- `ansible-fleet` — serial rollout, audit-first patterns
- `macvlan-net` — why macvlan-ranged IPs can be unreachable from macOS

## Deployment-specific details

This skill is generic SSH fleet hygiene. Everything above applies to any
fleet. Concrete host globs, bastion IPs, and fleet-specific policy live in
that deployment's fleet-ops repo.

**For the rust-town fleet:**
- `/Users/studio/rust-town/fleet-ops/CLAUDE.md` — canonical auth policy (as of 2026-04-06)
- `/Users/studio/rust-town/fleet-ops/docs/ssh-config-examples.md` — fleet `~/.ssh/config` (host globs `mister.local 10.0.3.* ubrpi* nbrpi* fbrpi*`, bastion hop via i9 at `10.0.2.230`)
- `fleet-inventory` skill — host/IP/key reference
- `rigfleet-fleet-inventory` skill — one-page fleet reference

**For a different deployment:** substitute your own host globs/bastion IPs
and point this section at your own fleet docs (e.g. `<your-fleet>/docs/ssh-config-examples.md`).
