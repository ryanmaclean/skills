---
name: zellij-team
description: Use when spawning parallel Claude agent workers in visible Zellij panes — multi-agent visual coordination with real-time observation and ctrl-c control. Replaces tmux-based agent teams.
---

# Zellij Team — Parallel Claude Agents in Zellij Panes

## Overview

Spawn N parallel `claude` workers, each in its own Zellij pane, so you can
watch them work simultaneously and ctrl-c any pane to kill a runaway agent.

This is the Zellij-native equivalent of Claude Code's tmux agent teams backend.
The current session is the **team leader**. Workers are spawned with explicit
task prompts and run until they exit or you kill them.

## Quick Start

```bash
# Spawn 3 agents on explicit tasks
nu ~/.claude/skills/zellij-team/zellij-team.nu \
  --tasks ["Fix the auth bug in src/auth.ex" \
           "Write tests for ElixirGastown.LLM.RateLimiter" \
           "Update README with new rate-limiter docs"]

# Spawn from task files
nu ~/.claude/skills/zellij-team/zellij-team.nu \
  --files [tasks/auth-fix.md tasks/test-rate-limiter.md tasks/docs.md]

# Custom session / tab name
nu ~/.claude/skills/zellij-team/zellij-team.nu \
  --tab "gas-city-workers" \
  --tasks ["task A" "task B"]
```

## Interaction Model

| Action | How |
|--------|-----|
| Watch agents | Look at each pane — they stream output live |
| Kill one agent | `ctrl-c` in that pane |
| Kill all | Close the Zellij tab |
| Switch panes | `Alt+arrow` |
| Zoom a pane | `ctrl-p z` |
| Send input to agent | Click pane, type (only if agent paused at a question) |

## Layout

Workers open in a **new tab** in the current Zellij session, auto-split:

- 2 workers → side by side (vertical split)
- 3–4 workers → 2×2 grid
- 5–6 workers → 2 rows, 3 cols
- 7+ workers → 3 rows

The tab is named `team-<timestamp>` by default, or `--tab NAME`.

## Agent Flags Used

Workers run with:
```
claude --dangerously-skip-permissions --permission-mode bypassPermissions
```

Rationale: workers are short-lived, task-scoped subprocesses. You (the leader)
are watching and can ctrl-c. Full interactive permission prompts would stall them.

Override with `--mode default` if you want workers to prompt for permissions.

## Token Efficiency

Each worker has its own context window. Keep task prompts focused:
- Good: "Fix the 3 failing tests in test/rate_limiter_test.exs"
- Bad: "You are an expert engineer, please consider all aspects of..."

Use `--effort low` on workers for routine tasks, `high` only for complex ones.

## Key Rule: Never Hardcode Session Name

Uses `$ZELLIJ_SESSION_NAME` — see zellij skill for the canonical pattern.
