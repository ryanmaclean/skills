---
name: rigfleet-fleet-inventory
description: One-page fleet reference — hostnames, IPs, roles, BEAM vs Zig split, polecat vs ii-agent naming. Points at /Users/studio/rust-town/fleet-ops/ansible/inventory as source of truth. Use when looking up a host, deciding where to run a service, or explaining fleet topology.
trigger: fleet hosts, inventory, which pi, polecat, ii-agent host, ubrpi, nbrpi
---

# Fleet Inventory (quick reference)

Source of truth: `/Users/studio/rust-town/fleet-ops/ansible/inventory`.
This skill is a pointer, not a copy — re-read the file if uncertain.

As of 2026-04-06.

## Host groups

| Group | Hosts | Notes |
|-------|-------|-------|
| `[pis]` | `ubrpi401, 405, 407-411, 501, 502` (Ubuntu ARM64), `nbrpi301, 302` (NetBSD) | Pi 3B+/4/5, 1-16 GB |
| `[nas]` | `i7-zfs-pop, i9-zfs-pop, qnas, snas, anas` | Canonical 5-host NAS inventory (see below) |
| `[workstations]` | `studio (M2U), minim4-16, minim4-24, pop4090` | Apple Silicon + RTX |
| `[netbsd_vms]` | `nb-vm-24 (2222), nb-vm-16 (2224)` | QEMU via ProxyJump |
| `[fpga]` | `de10nano (MiSTer ao486)`, `tangconsole138k` (pending) | Softcore BSD targets |
| `[windows]` | `7950x3090win, 5950x3090win, 5950x3070win` | Shell=cmd |
| `[qnas_services]` | macvlan containers at `10.0.3.2xx` | NO INTERNET |

## Canonical NAS inventory (5 hosts, per user 2026-04-15)

Group: `[nas]` in `fleet-ops/ansible/ansible/inventory`. Exactly 5 members — 3 vendor appliances + 2 Linux-with-ZFS boxes.

| Host | IP | Role | Notes |
|------|-----|------|-------|
| `snas` | `10.0.3.137` | Synology appliance (DSM) | SSH tarpitted — ONE attempt policy |
| `qnas` | `10.0.2.35` | QNAP appliance (QTS) | Raw module only; runs qnas_services containers on 10.0.3.x macvlan |
| `anas` | `10.0.3.9` | Asustor appliance (Atom C3538, 16GB) | Raw module only |
| `i7-zfs-pop` | `10.0.3.117` | Linux-with-ZFS (Pop!_OS, i7-8700K, 32GB) | Full Ansible |
| `i9-zfs-pop` | `10.0.2.230` | Linux-with-ZFS (Pop!_OS, i9-10900, 128GB) | Full Ansible; RUNS LEGACY GITEA |

> **`wnas` is NOT a fleet host.** It was a typo in an earlier brief. Do not add it anywhere — inventory, skills, playbooks, or /etc/hosts. The canonical list is the 5 rows above.

Probe policy for `snas`/`qnas`/`anas`: per `~/.claude/CLAUDE.md`, these appliances auto-block brute-force. ONE attempt per task, then STOP — no retry loops.

## Key IPs

| Service | Host | IP | Port |
|---------|------|-----|------|
| Gitea (main) | i9-zfs-pop | `10.0.2.230` | `3001` |
| Gitea (QNAS) | qnas container | `gitea.local` (10.0.3.210) | `3000` |

> **LEGACY MARKER (2026-04-11):** i9 Gitea (10.0.2.230:3001) is LEGACY as of 2026-04-11. Use QNAS Gitea (gitea.local:3000) for new repos. Existing repos stay on i9 until they're touched for other reasons; no preemptive migration.

> **Fleet `.local` DNS (2026-04-15):** All shared services resolve via fleet `/etc/hosts` — see `/Users/studio/rust-town/fleet-ops/docs/fleet-local-dns.md`. Prefer the `.local` name; the raw IP is shown only for debugging.

| Ergo IRC (TLS) | qnas container | `ergo.local` (10.0.3.203) | `6697` |
| Qdrant | qnas container | `qdrant.local` (10.0.3.206) | `6333` |
| Ollama (QNAS, no internet) | qnas container | `ollama.local` (10.0.3.207) | `11434` |
| Ollama (pop4090, kimi) | pop4090 | `10.0.2.42` | `11434` |
| SearXNG | qnas container | `searxng.local` (10.0.3.35) | `80` |
| PostgreSQL | qnas container | `postgres.local` (10.0.3.204) | `5432` |
| Redis/Valkey | qnas container | `redis.local` (10.0.3.205) | `6379` |
| TensorZero (host net) | i9-zfs-pop | `10.0.2.230` | `3000` |
| TensorZero (host net) | qnas (host not macvlan) | `10.0.2.35` | `3000` |
| Gastown coordinator | i9-zfs-pop | `10.0.2.230` | `4000` |

## BEAM vs Zig split

| Runtime | What runs there | Hosts |
|---------|-----------------|-------|
| BEAM / Elixir (gastown) | Federation, rate limiter (coordinator only), polecats | i9 coordinator, Pi workers with `GASTOWN_ROLE=polecat_worker` |
| Zig (ii-agent) | IRC fleet agent, tool harness, `!exec`/`!task` | All Pis + NetBSD hosts |

**Polecats run on coordinator, not Pi workers.** See `gastown-sling`.

## Naming conventions

| Prefix | Meaning |
|--------|---------|
| `ubrpi` | Ubuntu on Raspberry Pi (ARM64) |
| `nbrpi` | NetBSD on Raspberry Pi |
| `fbrpi` | FreeBSD on Raspberry Pi |
| `qnas/snas/anas` | QNAP / Synology / Asustor NAS |
| `minim4-*` | Mac mini M4 |
| `*zfs-pop` | Pop!_OS on ZFS |

## High-risk zone (do NOT touch without user approval)

- `udmpro 10.0.3.5` — UDM Pro firewall/router
- Switches, APs — not in inventory; ask user
- Password changes anywhere — see `ssh-fleet`

## Credentials

Managed by Ansible. Keys live in inventory via `ansible_ssh_private_key_file`.
NEVER guess — see `ssh-fleet` auth policy.

## Common mistakes

| Mistake | Fix |
|---------|-----|
| SSH to `10.0.3.x` from Mac | macvlan — hop via i9 (10.0.2.230) or a Pi |
| Running polecats on a Pi | Pi is a worker node, not host — use i9 |
| Deploying to all Pis at once | `--limit ONE_HOST` first (FIFO brick rule) |
| Using stale group names | Re-read `ansible/inventory` — it changes |
| Assuming QNAS has internet | It doesn't (macvlan) — use host-net on i9 |

## Related

- `ansible-fleet` — playbook patterns, serial rollout
- `ssh-fleet` — access config, auth policy
- `macvlan-net` — why 10.0.3.x has no internet
- `rigfleet-ii-agent-ops` — ii-agent deploy workflow
