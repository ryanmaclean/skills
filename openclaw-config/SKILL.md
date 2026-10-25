---
name: openclaw-config
description: Manage RigFleet bot configuration — channels, agents, security, and autopilot settings. Use when changing openclaw/RigFleet bot channels, agent assignments, or autopilot behavior.
version: 3.0.0
---

# RigFleet Operations Runbook

Diagnose and fix real problems. Every command here is tested and works.

---

## Quick Health Check

Run this first when anything seems wrong. Copy-paste the whole block:

```bash
echo "=== GATEWAY ===" && \
ps aux | grep -c "[o]penclaw" && \
echo "=== CONFIG JSON ===" && \
python3 -m json.tool ~/.rigfleet/rigfleet.json > /dev/null 2>&1 && echo "JSON: OK" || echo "JSON: BROKEN" && \
echo "=== CHANNELS ===" && \
cat ~/.rigfleet/rigfleet.json | jq -r '.channels | to_entries[] | "\(.key): policy=\(.value.dmPolicy // "n/a") enabled=\(.value.enabled // "implicit")"' && \
echo "=== PLUGINS ===" && \
cat ~/.rigfleet/rigfleet.json | jq -r '.plugins.entries | to_entries[] | "\(.key): \(.value.enabled)"' && \
echo "=== CREDS ===" && \
ls ~/.rigfleet/credentials/whatsapp/default/ 2>/dev/null | wc -l | xargs -I{} echo "WhatsApp keys: {} files" && \
for d in ~/.rigfleet/credentials/telegram/*/; do bot=$(basename "$d"); [ -f "$d/token.txt" ] && echo "Telegram $bot: OK" || echo "Telegram $bot: MISSING"; done && \
[ -f ~/.rigfleet/credentials/bird/cookies.json ] && echo "Bird cookies: OK" || echo "Bird cookies: MISSING" && \
echo "=== CRON ===" && \
cat ~/.rigfleet/cron/jobs.json | jq -r '.jobs[] | "\(.name): enabled=\(.enabled) status=\(.state.lastStatus // "never") \(.state.lastError // "")"' && \
echo "=== RECENT ERRORS ===" && \
tail -10 ~/.rigfleet/logs/gateway.err.log 2>/dev/null && \
echo "=== MEMORY DB ===" && \
sqlite3 ~/.rigfleet/memory/main.sqlite "SELECT COUNT(*) || ' chunks, ' || (SELECT COUNT(*) FROM files) || ' files indexed' FROM chunks;" 2>/dev/null
```

---

## File Map

```
~/.rigfleet/
├── rigfleet.json                    # MAIN CONFIG — channels, auth, gateway, plugins, skills
├── rigfleet.json.bak*               # Auto-backups (.bak, .bak.1, .bak.2 ...)
├── exec-approvals.json              # Exec approval socket config
│
├── agents/main/
│   ├── agent/auth-profiles.json     # Anthropic auth tokens
│   └── sessions/
│       ├── sessions.json            # SESSION INDEX — keys are like agent:main:whatsapp:+1234
│       └── *.jsonl                  # Session transcripts (one JSON per line)
│
├── workspace/                       # Agent workspace (git-tracked)
│   ├── SOUL.md                      # Personality, writing style, tone rules
│   ├── IDENTITY.md                  # Name, creature type, vibe
│   ├── USER.md                      # Owner context and preferences
│   ├── AGENTS.md                    # Session behavior, memory rules, safety
│   ├── BOOT.md                      # Boot instructions (autopilot notification protocol)
│   ├── HEARTBEAT.md                 # Periodic task checklist (empty = skip heartbeat)
│   ├── MEMORY.md                    # Curated long-term memory (main session only)
│   ├── TOOLS.md                     # Contacts, SSH hosts, device nicknames
│   ├── memory/                      # Daily logs: YYYY-MM-DD.md, topic-chat.md
│   └── skills/                      # Workspace-level skills
│
├── memory/main.sqlite               # Vector memory DB (Gemini embeddings, FTS5 search)
│
├── logs/
│   ├── gateway.log                  # Runtime: startup, channel init, config reload, shutdown
│   ├── gateway.err.log              # Errors: connection drops, API failures, timeouts
│   └── commands.log                 # Command execution log
│
├── cron/
│   ├── jobs.json                    # Job definitions (schedule, payload, delivery target)
│   └── runs/                        # Per-job run logs: {job-uuid}.jsonl
│
├── credentials/
│   ├── whatsapp/default/            # Baileys session: ~1400 app-state-sync-key-*.json files
│   ├── telegram/{botname}/token.txt # Bot tokens (one per bot account)
│   └── bird/cookies.json            # X/Twitter auth cookies
│
├── extensions/{name}/               # Custom plugins (TypeScript)
│   ├── rigfleet.plugin.json         # {"id", "channels", "configSchema"}
│   ├── index.ts                     # Entry point
│   └── src/                         # channel.ts, actions.ts, runtime.ts, types.ts
│
├── identity/                        # device.json, device-auth.json
├── devices/                         # paired.json, pending.json
├── media/inbound/                   # Received images, audio files
├── media/browser/                   # Browser screenshots
├── browser/rigfleet/user-data/      # Chromium profile (~180MB)
├── tools/signal-cli/                # Signal CLI binary
├── subagents/runs.json              # Sub-agent execution log
├── canvas/index.html                # Web canvas UI
└── telegram/
    ├── update-offset-coder.json     # {"lastUpdateId": N} — Telegram polling cursor
    └── update-offset-sales.json     # Reset these to 0 to replay missed messages
```

---


## Deep references

- `references/troubleshooting.md` — WhatsApp, Telegram, Signal, cron, memory troubleshooting guides
- `references/extending.md` — session JSONL format, gateway startup, error patterns, extending RigFleet, license
## License

MIT
