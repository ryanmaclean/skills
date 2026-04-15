---
name: irclaw-cluster-ops
description: Use when operating the fleet of Raspberry Pis, NAS hosts, and workstations via irclaw IRC bots, Ansible, Ollama (kimi-k2.5:cloud), or long-running deploys. Covers IRC DM protocol, bot security policy, zellij for long tasks, OS upgrades, and gastown/polecat deployments.
---

# irclaw Cluster Ops

## IRC / irclaw bot protocol

**IRC server:** `ergo:6697` (TLS, Ergo IRC)
**Bot nicks:** match hostname — `ubrpiYYY`, `ubrpiYYY`, `i9-zfs-popular`, etc.
**Bot model:** `kimi-k2.5:cloud` via Ollama on each host

### DM a bot via irssi (preferred)

Use the running irssi session — do NOT try raw OpenSSL connections (bots won't reply to unregistered nicks in time):

```bash
# From irssi: /msg ubrpiYYY what's the status of the mix release build?
# Or check a command: /msg ubrpiYYY run: df -h
```

To send programmatically from Claude Code, use ansible to poke the bot's local irclaw gateway instead:

```bash
ansible ubrpi502 -i ansible/inventory -m raw \
  -a "curl -s http://localhost:3000/status 2>/dev/null || journalctl -u irclaw-gateway -n 5 --no-pager"
```

### Bot security policy — [autonomy] allowed_commands

Bots will **refuse** destructive OS-level commands regardless of approval:
- `do-release-upgrade` — blocked, use Ansible playbook instead
- `rm -rf /` level commands — blocked
- Anything not in `allowed_commands` in `~/.irclaw/config.toml`

To check what a bot allows:
```bash
ansible ubrpi401 -i ansible/inventory -m raw \
  -a "grep -A20 '\[autonomy\]' ~/.irclaw/config.toml"
```

To expand allowed commands: edit via `ansible/16-qnas-autonomy-block.yml` pattern or `irclaw-config` skill.

## Long-running tasks — always use Zellij

**NEVER use `nohup` or background tasks for deploys.** They get killed by timeouts.

```bash
# Create/reuse a named session
zellij attach --create gas-city

# Run a long task in a named pane (stays alive after command exits for inspection)
zellij --session gas-city run --name "pi-deploy-ubrpiYYY" -- bash -c "
  cd $HOME/rust-town/irclaw && \
  ansible-playbook -i ansible/inventory ansible/22-gastown-pis.yml \
    --limit ubrpiYYY \
    -e 'gastown_cookie=ELIXIR_GASTOWNBEAM2026' \
    -v 2>&1 | tee /tmp/pi-deploy-ubrpi502.log
  echo EXIT: \$?
"

# Watch log in another pane
zellij --session gas-city run --name "log-tail" -- bash -c "tail -f /tmp/pi-deploy-ubrpi502.log"
```

Note: `zellij attach --create` only works interactively. From Claude Code, check for an existing session first:
```bash
zellij list-sessions --no-formatting 2>/dev/null | grep -v EXITED | head -5
```

If no live session exists, ask the user to open one: `! zellij attach --create gas-city`

## Ollama / kimi-k2.5:cloud

Ollama runs on coordinators (pop4090, i9, i7). Bots use `kimi-k2.5:cloud` — a cloud-routed model, not local weights.

```bash
# Check Ollama on pop4090
curl -s http://10.10.2.34:11434/api/tags | jq '.models[].name'

# Query kimi directly
curl -s http://10.10.2.34:11434/api/generate \
  -d '{"model":"kimi-k2.5:cloud","prompt":"status check","stream":false}' | jq '.response'

# Check which model each bot is using
ansible ubrpi401 -i ansible/inventory -m raw \
  -a "grep 'model\|api_url' ~/.irclaw/config.toml | head -5"
```

## OS upgrades (22.04 → 24.04)

**Always Ansible, never via bot.** Use the existing playbook:

```bash
# Dry-run check first
ansible ubrpiYYY -i ansible/inventory -m raw \
  -a "sudo do-release-upgrade -c"

# Full upgrade — run in Zellij, takes 20-30 min + reboot
zellij --session gas-city run --name "upgrade-ubrpiYYY" -- bash -c "
  cd $HOME/rust-town/irclaw && \
  ansible-playbook -i ansible/inventory ansible/upgrade-pis.yml \
    --limit ubrpiYYY -v 2>&1 | tee /tmp/upgrade-ubrpiYYY.log
  echo EXIT: \$?
"
```

The playbook (`ansible/upgrade-pis.yml`) runs:
1. `dpkg --configure -a` + fix broken packages
2. `apt-get upgrade -y`
3. `DEBIAN_FRONTEND=noninteractive do-release-upgrade -f DistUpgradeViewNonInteractive`
4. Post-upgrade Python install

## ELIXIR_gastown / Polecat deploys

**Key facts:**
- Release name: `elixir_gastown` (not `gastown`) — path is `_build/prod/rel/elixir_gastown/bin/elixir_gastown`
- BEAM node names use `egt@` prefix: `egt@7950x4090pop`, `egt@i9-zfs-pop`, `egt@i7-zfs-pop`
- Cookie: `ELIXIR_GASTOWNBEAM2026`
- NIF (gastown_nif.so) must be compiled on-host — never rsync macOS `.so` to Linux
- `:zstd` is gated on `zstd_available?()` — safe on ARM64 without the NIF

**Networking and naming:**
- sname (short names) is correct for this cluster — all nodes use sname. Do NOT change to name/longnames. Requires bidirectional /etc/hosts entries on both coordinators and workers for cross-host resolution.
- After any deploy, patch /etc/hosts on new workers: add coordinator IPs (i9-zfs-pop=10.0.2.230, 7950x4090pop=10.0.2.34, i7-zfs-pop=10.0.3.117). Also add worker IPs to coordinator /etc/hosts for bidirectional resolution.
- GASTOWN_PEERS env var must be set in /etc/gastown-worker.env — this is what application.ex:connect_peers/0 reads.

**Build strategy for Pis:** Build once on ubrpi502 (Pi 5, 16GB), distribute tarball to others.

```bash
# Verify federation
ansible pop4090 -i ansible/inventory -m shell \
  --become --become-user $USER \
  -a "/opt/gastown/_build/prod/rel/elixir_gastown/bin/elixir_gastown rpc 'Node.list() |> inspect()'"

# Clean stale _build before retry (fixes File.RenameError on mix release)
ansible ubrpi502 -i ansible/inventory -m shell \
  --become --become-user ubuntu \
  -a "rm /opt/gastown/_build && cd /opt/gastown && git pull"
```

## Common mistakes

| Mistake | Fix |
|---|---|
| Assuming BEAM name | Always check with `Node.self()` — it's `egt@7950x4090pop` |
| Running long ansible via nohup | Use Zellij — nohup tasks get killed |
| DM-ing bots via raw | Use irssi or check status via ansible |
| Expecting bot to run `do-release-upgrade` | Use `ansible/upgrade-pis.yml` |
| Deploying to all Pis at once | Canary on ubrpi401 first, then roll out |
| Pi with ubuntu user | Pis uses `ansible_user=string` — always pass `-e gastown_user=string` |
| rsync from macOS for NIF | Compile on-host; exclude `priv/native/` from rsync |
| env.sh overrides systemd env | Mix release env.sh runs after EnvironmentFile and hardcodes RELEASE_COOKIE and RELEASE_DISTRIBUTION — fix in rel/env.sh.eex using ${VAR:-default} syntax |
| ELIXIR_GASTOWN_PEERS vs ELIXIR_GASTOWN_COORDINATOR_NODE | Application reads ELIXIR_GASTOWN_PEERS but playbook wrote ELIXIR_GASTOWN_COORDINATOR_NODE — workers silently don't auto-connect |
