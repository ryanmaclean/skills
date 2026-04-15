---
name: a-team
description: Use when a project needs role-based team coordination — PM breakdown, parallel specialist execution, review gates. Best for multi-deliverable projects vs tiger-team (single problem) or zellij-team (independent tasks).
---

# A-Team — Role-Based Project Team Coordination

## Overview

Inspired by [a.team](https://a.team): don't send solo agents — form a **project team**
with specific roles. A PM coordinates, engineers execute, specialists advise,
reviewers gate quality. The team has continuity across the project lifecycle.

## Roles

| Role | Agent | Purpose |
|------|-------|---------|
| `pm` | Project Manager | Reads brief, breaks into tasks, assigns to specialists, synthesizes final output |
| `engineer` | Implementation | Writes code, runs commands, produces artifacts |
| `reviewer` | Quality Gate | Checks engineer + specialist output, flags issues before PM synthesis |
| `specialist` | Domain Expert | Configurable domain: "security", "ansible", "elixir", "performance", etc. |

## Coordination Flow

```
Phase 1: PM Breakdown
  ┌─────────┐
  │   PM    │  reads brief → task list + role assignments
  └────┬────┘
       │
Phase 2: Parallel Execution
  ┌────┴────┬────────────┐
  │ Engineer │ Specialist │  work assigned tasks in parallel
  └────┬────┴─────┬──────┘
       │          │
Phase 3: Review Gate
  ┌────┴──────────┴──┐
  │     Reviewer     │  checks all output, flags issues
  └────────┬─────────┘
           │
Phase 4: PM Synthesis
  ┌────────┴─────────┐
  │   PM (final)     │  integrates, writes summary/ship report
  └──────────────────┘
```

Each phase runs explicitly via `--phase 1|2|3|4` so you see output between phases.

## Quick Start

```bash
# Phase 1: PM breaks down the project
nu ~/.claude/skills/a-team/a-team.nu --phase 1 \
  --brief "Deploy gastown rate limiter to Pi fleet"

# Phase 2: Engineer + Specialist execute in parallel panes
nu ~/.claude/skills/a-team/a-team.nu --phase 2 \
  --brief "Deploy gastown rate limiter to Pi fleet" \
  --specialist "ansible"

# Phase 3: Reviewer checks output
nu ~/.claude/skills/a-team/a-team.nu --phase 3 \
  --brief "Deploy gastown rate limiter to Pi fleet"

# Dry run any phase
nu ~/.claude/skills/a-team/a-team.nu --dry-run --phase 1 \
  --brief "Deploy gastown rate limiter to Pi fleet"
```

## When to Use: a-team vs tiger-team vs zellij-team

| | a-team | tiger-team | zellij-team |
|---|--------|------------|-------------|
| **Shape** | Project team with roles | Same problem, specialist lenses | Parallel tasks, generic workers |
| **Coordination** | Sequential phases + review gates | Parallel analysis, synthesize | Fire-and-forget parallel |
| **Best for** | Multi-deliverable projects | Deep investigation of one issue | Independent, parallelizable tasks |
| **PM?** | Yes — plans, assigns, synthesizes | No — each lens is peer | No — tasks are self-contained |
| **Review gate?** | Yes — reviewer checks before ship | No — synthesis is the gate | No |
| **Example** | "Deploy service to fleet: build, distribute, configure, verify" | "Why is auth latency spiking?" | "Fix these 5 independent bugs" |

## Agent Flags

All agents run with `--dangerously-skip-permissions --permission-mode bypassPermissions`
by default. Override with `--mode default` for interactive permission prompts.

## Output Convention

Each phase writes output to `/tmp/a-team-<brief-hash>/`:
- `phase1-pm-plan.md` — PM's task breakdown
- `phase2-engineer.md` — Engineer's output
- `phase2-specialist.md` — Specialist's output
- `phase3-review.md` — Reviewer's findings
- `phase4-summary.md` — PM's final synthesis
