---
name: dispatch-nu
description: >
  Wave-based spec dispatcher for any project using dispatch.codex.nu (Codex+AC hybrid)
  or dispatch.nu (Auto-Claude only). Spawns Zellij panes, tracks state in JSON,
  pre-plans via Ollama cloud, routes specs to Codex CLI or Auto-Claude by agent field.
  Use when checking wave status, launching specs, rescuing stalled panes, resetting
  failed specs, or bootstrapping a new project dispatcher.
---

# dispatch.codex.nu — Hybrid Spec Dispatcher

Routes specs to **Codex CLI** (`gpt-5.3-codex`) or **Auto-Claude** (`run.py`) per spec.
Pre-plans every spec via **Ollama cloud** (`qwen3-coder:480b-cloud`) before dispatch.
Tracks state in `.codex/dispatch-state.json`. Spawns named Zellij panes.

## Projects

| Project | Dispatcher | Session |
|---------|-----------|---------|
| triz | `$HOME/triz/dispatch.codex.nu` | wise-xylophone |
| dub (EFI) | `$HOME/dub/dispatch.codex.nu` | wise-xylophone |
| bop | `$HOME/bop/dispatch.nu` | bop |

## Quick reference

```bash
nu dispatch.codex.nu plan                        # show all waves + agent + plan status
nu dispatch.codex.nu status                      # done / failed / pending counts
nu dispatch.codex.nu pre-plan                    # queue all specs through Ollama now
nu dispatch.codex.nu run --wave 1 --yes          # run wave 1 (skips confirmation)
nu dispatch.codex.nu run --yes                   # run all pending waves
nu dispatch.codex.nu watch-handoff --from-wave 1 --to-wave 2 --yes
nu dispatch.codex.nu mark-done 011              # manually mark complete
nu dispatch.codex.nu mark-failed 011            # manually mark failed
nu dispatch.codex.nu reset --spec 011           # reset single spec to pending
nu dispatch.codex.nu reset                      # reset ALL (nuclear)
nu dispatch.codex.nu retry                      # re-queue all failed specs
nu dispatch.codex.nu review 011                 # print spec.md
nu dispatch.codex.nu list                       # table of all specs
nu dispatch.codex.nu lock                       # check active run lock
```

## Launching panes (two-pane pattern)

```bash
# Pane A — run current wave
zellij --session wise-xylophone run --name "proj-wave1" --close-on-exit -- \
  nu -c "cd $HOME/PROJ && nu dispatch.codex.nu run --wave 1 --yes"

# Pane B — auto-chain to next wave when A finishes
zellij --session wise-xylophone run --name "proj-handoff" --close-on-exit -- \
  nu -c "cd $HOME/PROJ && nu dispatch.codex.nu watch-handoff --from-wave 1 --to-wave 2 --yes"
```

## Spec table fields

| Field | Values | Meaning |
|-------|--------|---------|
| `wave` | 0–N | Dependency order — higher = later |
| `cost` | 1–4 | ⚀trivial ⚁small ⚂medium ⚃complex — drives cooldown |
| `agent` | `"codex"` / `"ac"` | Routes to Codex CLI or Auto-Claude run.py |
| `mode` | `"direct"` / `"isolated"` | direct=writes to repo, isolated=worktree (needs merge) |

## Cooldown delays (rate limit pacing)

| Cost | Delay | Typical specs |
|------|-------|---------------|
| 1 trivial | 15s | config, gitignore |
| 2 small | 30s | docs, thin wrappers |
| 3 medium | 60s | parsers, commands, extensions |
| 4 complex | 90s | LLM integration, TUI, MCP server |

Batch pause of 2 min every 4 specs on wave 2+.

## Ollama pre-planning

Runs `qwen3-coder:480b-cloud` on each spec.md before Codex touches it.
Writes `.auto-claude/specs/{spec}/plan.md` with:
- Risk register (top 5 risks + mitigations)
- File order (what to create, in order)
- Crate audit (APIs to verify before writing)
- Gotchas (known pitfalls)

Codex receives both `spec.md` and `plan.md` as context → fewer wrong turns.

```bash
# Pre-plan all specs now (run while waiting for wave 1):
nu dispatch.codex.nu pre-plan
```

## State file

```
.codex/dispatch-state.json
{"completed": ["001","002",...], "failed": [], "skipped": []}
```

Pane shell calls `mark-done NNN` or `mark-failed NNN` on exit.
Signal files also written to `/tmp/triz-done-NNN` for cross-script coordination.

## Rescue playbook

```bash
# Stale lock (process died without clearing):
rm .codex/dispatch-lock.json
nu dispatch.codex.nu retry

# Pane exited but mark-done never fired:
nu dispatch.codex.nu mark-done 011

# Spec produced bad output, redo it:
nu dispatch.codex.nu reset --spec 011
nu dispatch.codex.nu run --wave 2 --yes

# Rate limited mid-wave:
nu dispatch.codex.nu retry
nu dispatch.codex.nu run --wave 2 --yes
```

## Bootstrap a new project

Copy the template and edit the `specs []` table and constants:

```bash
cp ~/.claude/skills/dispatch-nu/dispatch.codex.nu.template /path/to/project/dispatch.codex.nu
# Edit: PROJECT_DIR, specs table (id/name/wave/cost/agent/mode)
# Seed completed: ["any-pre-done-specs"] in state
nu dispatch.codex.nu pre-plan   # plan everything via Ollama
nu dispatch.codex.nu plan       # review
nu dispatch.codex.nu run --wave 1 --yes
```

## dub-specific notes (dispatch.codex.nu)

- Uses `gpt-5.3-codex -c model_reasoning_effort=xhigh`
- Wave 1 is strictly sequential (each spec is the next's input)
- Isolated specs need manual merge before next wave
- `watch-handoff` waits for wave N then auto-starts wave N+1

## triz-specific notes

- 001+002 pre-seeded as completed in state
- Codex specs: 011, 017, 018, 019, 020 (pure Rust / TypeScript / config)
- AC specs: 003, 004, 005, 006, 007, 008, 009, 012, 013, 014, 016 (complex LLM/Nu)
- Wave 1: `011-core-commands` via Codex (no LLM deps, deterministic Rust)
