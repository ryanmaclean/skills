---
name: elixir-gastown-ops
description: Use when operating, debugging, or deploying the elixir-gastown Elixir/BEAM fleet — cluster federation, release builds, worker deploys, seccomp fixes, sname/name distribution issues, rate limiter, or any gastown service on Pi workers or coordinator nodes.
---

# Gastown Ops

## Fleet Topology

| Role | Hosts | User | Service |
|------|-------|------|---------|
| Coordinator | i9-zfs-pop, i7-zfs-pop, 7950x4090pop | `<user>` | `gastown.service` |
| Pi workers | ubrpi401/405/407/408/409/410/411/501/502 | ubuntu (ubrpi501: string) | `gastown-worker.service` |

- BEAM cookie: `GASTOWNBEAM2026`
- Node naming: `egt@<hostname>` — **sname** on all nodes
- Coordinator Phoenix port: **4000** (i9 only, `PHX_SERVER=true`)
- Workers: `PHX_SERVER=false`, `GASTOWN_ROLE=polecat_worker`
- Repo: `http://<coordinator-ip>:3001/<user>/elixir-gastown.git`

## Check Cluster

```bash
# Node count from i9
ansible -i ansible/inventory i9-zfs-pop -m shell \
  -a '/opt/gastown/_build/prod/rel/elixir_gastown/bin/elixir_gastown rpc "Node.list() |> inspect() |> IO.puts"'

# From a Pi worker (avoids i9 sname/name issues)
ansible -i ansible/inventory ubrpi407 -m shell \
  -a '/opt/gastown/_build/prod/rel/elixir_gastown/bin/elixir_gastown rpc "Node.list() |> inspect() |> IO.puts"'
```

Expected: 10 nodes (3 coordinators + 7 Pi workers, excluding self).

## Build + Deploy

**Full rebuild on ubrpi502 + distribute to all Pis:**
```bash
ansible-playbook -i ansible/inventory ansible/22-gastown-pis.yml -v
```

**Distribute only (skip build):**
```bash
ansible-playbook -i ansible/inventory ansible/22-gastown-pis.yml \
  --limit ubrpi401,ubrpi407,... -v
# Note: no --skip-tags build — Play 1 targets ubrpi502 only; --limit to pis skips it
```

**Rebuild coordinator on i9:**
```bash
ansible -i ansible/inventory i9-zfs-pop -m shell -a '
  cd /opt/gastown && sudo -u $DEPLOY_USER git pull origin main
  # If local changes block pull:
  sudo -u $DEPLOY_USER git stash && sudo -u $DEPLOY_USER git pull origin main
  sudo -u $DEPLOY_USER bash -c "export HOME=/home/$DEPLOY_USER MIX_ENV=prod PATH=/usr/local/bin:/usr/bin:/bin; mix release --overwrite 2>&1 | tail -5"
  systemctl restart gastown
' -b
```

**Rustler NIF missing from coordinator release:**
```bash
# Build NIF manually, then rebuild release
ansible -i ansible/inventory i9-zfs-pop -m shell -a '
  cd /opt/gastown/native/gastown_nif
  sudo -u $DEPLOY_USER bash -c "PATH=/usr/local/bin:/usr/bin:/bin cargo build --release"
  NIF=target/release/libgastown_nif.so
  DEST=/opt/gastown/_build/prod/rel/elixir_gastown/lib/elixir_gastown-0.3.0/priv/native/gastown_nif.so
  mkdir -p $(dirname $DEST) && cp $NIF $DEST && chown $DEPLOY_USER:$DEPLOY_USER $DEST
' -b
# Then re-run mix release --overwrite and restart
```

## Common Fixes

### sname/name mismatch (i9 not federating)
i9 `/etc/gastown.env` must have `RELEASE_DISTRIBUTION=sname` — not `name`.
All nodes use sname. Check:
```bash
ansible -i ansible/inventory i9-zfs-pop,pis -m shell \
  -a 'grep RELEASE_DISTRIBUTION /etc/gastown.env /etc/gastown-worker.env 2>/dev/null'
```

### seccomp SIGSYS on coordinator (syscall 92 = chown)
ERTS dirty I/O scheduler calls `chown` — blocked by `SystemCallFilter=~@privileged`:
```bash
ansible -i ansible/inventory i9-zfs-pop -m shell -a '
  sed -i "/SystemCallFilter=~@privileged/a SystemCallFilter=chown fchown lchown fchownat" \
    /etc/systemd/system/gastown.service
  systemctl daemon-reload && systemctl restart gastown
' -b
```
Verify with: `journalctl -k | grep "syscall=92"` — should stop after fix.

### :global sync storm (nodes ejecting each other)
Symptom: nodes connect then immediately disconnect, "overlapping partitions" in logs.
Cause: old binary with `:global.register_name` in `check_singleton!`.
Fix: distribute new tarball (commit `da9fca2`+), restart workers with 15s stagger:
```bash
for h in <worker-ip-1> <worker-ip-2> <worker-ip-3>; do
  ssh ubuntu@$h "sudo systemctl restart gastown-worker"; sleep 15
done
```

### RELEASE_NODE wrong format
Must be short name only — no `@host` suffix:
```
RELEASE_NODE=egt          ✅
RELEASE_NODE=egt@ubrpi407 ❌  (invalid for sname)
```

### ubrpi501 chown failure in playbook
`gastown_user` must be `{{ ansible_user }}`, not hardcoded `ubuntu`.
ubrpi501 uses `ansible_user=string`.

### Worker not joining cluster (node not in Node.list)
1. Check `/etc/gastown-worker.env` — `RELEASE_NODE=egt`, `RELEASE_DISTRIBUTION=sname`
2. Check bidirectional `/etc/hosts` — Pi must resolve coordinator hostname AND coordinator must resolve Pi hostname
3. Add missing entry: `ansible -i inventory i9-zfs-pop,pop4090,i7-zfs-pop -b -m lineinfile -a 'path=/etc/hosts regexp=".*\s<hostname>$" line="<ip> <hostname>" state=present'`
4. Restart worker: `sudo systemctl restart gastown-worker`

## Rate Limiter

HTTP endpoint on i9: `GET http://<i9-ip>:4000/api/llm-permit` → `{"ok":true,"wait_ms":N}`
Release endpoint: `POST http://<i9-ip>:4000/api/llm-release`
Zeroclaw config on Pis: `/etc/gastown-worker.env` → irclaw `config.toml` has `pre_request_url`.
Deploy irclaw rate limit config: `ansible-playbook -i inventory ansible/23-irclaw-rate-limit.yml`

## Key Playbooks

| Playbook | Purpose |
|----------|---------|
| `22-gastown-pis.yml` | Build on ubrpi502 + deploy to all Pis |
| `23-irclaw-rate-limit.yml` | Set pre/post_request_url on Pi irclaw configs |
| `24-dd-fix-gastown-pis.yml` | Fix DD tags, process check, DogStatsD on Pis |

## Hostnames vs Node Names

Pi hostnames drop leading zeros: `ubrpi401` → hostname `ubrpi41`, node `:egt@ubrpi41`.
Exception: `ubrpi410`, `ubrpi411`, `ubrpi501`, `ubrpi502` keep full name.
