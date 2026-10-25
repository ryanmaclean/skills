---
name: gastown-sling
description: Use when dispatching tasks to gastown polecats — submitting beads, checking status, choosing runtimes, or understanding the coordinator-worker architecture.
---

# Gastown Sling — Dispatching Tasks to Polecats

## Architecture

**Polecats run on the coordinator** (i9-zfs-pop:4000), not on Pi workers.
Pi workers with `GASTOWN_ROLE=polecat_worker` are BEAM federation nodes that suppress coordinator-only services (rate limiter). They don't receive polecats.

`active polecats` in `/api/health` = tasks currently running on coordinator. 0 = idle (normal).

## Sling API

```bash
# Submit and immediately dispatch
curl -s -X POST http://10.0.2.230:4000/api/sling \
  -H 'Content-Type: application/json' \
  -d '{"description": "task prompt here", "runtime": "claude"}'
# → {"ok":true, "bead_id":"gt-xxxxx", "runtime":"claude"}

# Check bead status
curl -s http://10.0.2.230:4000/api/sling/gt-xxxxx

# List all beads
curl -s -H 'Authorization: Bearer $GASTOWN_API_TOKEN' http://10.0.2.230:4000/api/beads
```

## Available Runtimes

| Runtime | What it runs | Notes |
|---------|-------------|-------|
| `claude` | `claude -p <prompt>` | Claude Code CLI, non-interactive |
| `codex` | `codex exec --full-auto` | Codex CLI, sandboxed |
| `ollama` | `ollama run qwen3-coder:480b-cloud` | Local Ollama HTTP |
| `ollama_kimi` | `kimi-k2.5:cloud` at 10.0.2.42 | pop4090 Ollama |
| `stub` | `echo` | Testing only, instant |

Default: `claude` (or `stub` if baseline mode off).

## Writing Good Bead Descriptions

The `description` is the full prompt sent to the runtime. Be specific:

```bash
# Good — self-contained, actionable
curl -X POST http://10.0.2.230:4000/api/sling \
  -H 'Content-Type: application/json' \
  -d '{
    "description": "Clone all public repos from github.com/ryanmaclean to Gitea at http://10.0.2.230:3001. Use the Gitea API at /api/v1/repos/migrate. Token in GITEA_TOKEN env var. Work in /tmp/sync-$(date +%s). Log progress. When done, list all synced repos.",
    "runtime": "claude"
  }'
```

## Mix Task (local only)

```bash
# On i9, inside elixir-gastown dir
mix gastown.sling BEAD_ID --runtime claude
```

## Bead States

`open` → `in_progress` → `done` | `failed` | `abandoned`

## Common Mistakes

| Mistake | Fix |
|---------|-----|
| Expecting Pi workers to run polecats | Polecats run on coordinator only |
| "0 polecats" = broken | 0 = idle, submit a bead to see count go up |
| Short descriptions | The description IS the prompt — be detailed |
| Wrong Ollama host for kimi | Use `ollama_kimi` runtime, not `ollama` |
