---
name: preflight
description: >
  Pre-dispatch quota check. Run before delegating tasks to Codex, Auto-Claude, or
  any agent backend. Shows live utilization for all providers and recommends which
  backend to use based on remaining quota. Warns when any provider is above 70%
  and blocks dispatch recommendations when above 90%.
triggers:
  - "before dispatch"
  - "check quota"
  - "check usage"
  - "preflight"
  - "how much quota"
  - "before delegating"
  - "before spawning"
  - "before running agents"
---

# preflight — Pre-Dispatch Quota Check

Run this before delegating batches of tasks to Codex, Auto-Claude, or any agent
backend. Shows live utilization and recommends the best backend for the planned work.

## Usage

```
/preflight
/preflight 5          # planning to dispatch 5 specs
/preflight 10 codex   # planning 10 codex tasks specifically
```

## What it does

1. Reads live quota from `~/.claude/skills/providers/providers.nu --json`
2. Evaluates each provider against thresholds (green <70%, yellow 70-90%, red >90%)
3. Recommends backend routing based on remaining capacity
4. If any critical provider is >90%, warns before proceeding

## Thresholds

| Level | Usage | Action |
|-------|-------|--------|
| Green  | <70%  | Proceed freely |
| Yellow | 70-90%| Prefer cheaper model (Haiku over Sonnet, Haiku over Codex) |
| Red    | >90%  | Do task directly or use Ollama; avoid spawning agents |

## Backend routing rules

- **Codex session >80%**: Switch to Auto-Claude (Haiku) for simple tasks
- **Claude 5h >80%**: Avoid spawning more Claude Code subagents; use Codex or Ollama
- **Both constrained**: Do small tasks directly; defer large batches
- **All green**: Proceed with planned dispatch

## Instructions for Claude

When this skill is invoked:

1. Run `nu ~/.claude/skills/providers/providers.nu --json` via Bash tool
2. Parse the JSON and evaluate thresholds
3. Print a concise summary table with color-coded status
4. State clearly: which providers are constrained, which are free
5. Recommend: what backend to use for the planned work
6. If the user mentioned a task count (e.g. "/preflight 5"), estimate whether
   quota covers it and at what cost (Codex session burns faster than weekly)
7. Do NOT just print raw numbers — give an actionable recommendation

## Example output

```
Provider         Usage    Status   Reset
──────────────────────────────────────────
Claude 5h        44%      ✓ green  2h 47m
Claude 7d        61%      ✓ green  127h
Claude 7d sonnet 70%      ⚠ yellow 129h
Codex session    21%      ✓ green  12m
Codex weekly     32%      ✓ green  95h

Recommendation: All green. Codex session resets in 12m — if dispatching >3 specs,
wait 12 minutes to start with a fresh session budget. Proceed freely otherwise.
```
