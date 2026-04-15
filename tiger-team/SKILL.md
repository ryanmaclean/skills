---
name: tiger-team
description: Use when facing a hard single problem that benefits from multiple expert perspectives simultaneously — debugging, architecture decisions, security review, incident response. Spawns 2-4 specialist agents (attacker/defender/architect/pragmatist) attacking the SAME problem from different angles.
---

# Tiger Team — Multi-Perspective Attack on a Single Hard Problem

## Overview

A tiger team is a classic ops concept: a small group of elite specialists brought in
to crack ONE hard problem fast, from multiple attack angles simultaneously. Think
security red teams, incident response, or NASA's tiger teams that fixed Apollo 13.

Each specialist gets the **same problem statement** but a **different persona** injected
into their prompt. They run in parallel Zellij panes. You (the leader) read all outputs
and synthesize the best path forward.

### When to use what

| Tool | Use when... |
|------|-------------|
| **tiger-team** | One hard problem, need multiple expert lenses (debug, arch decision, security review) |
| **zellij-team** | Multiple independent tasks that can run in parallel |
| **a-team** | Sequential pipeline where each step feeds the next |

## Specialist Roles

| Role | Persona | Lens |
|------|---------|------|
| `attacker` | Red-team adversary | Finds where the approach breaks, stress-tests assumptions, identifies failure modes |
| `defender` | Safety-first engineer | Finds what's solid, builds the safest path forward, identifies invariants to preserve |
| `architect` | Systems thinker | Looks at structural/systemic angle, considers interfaces, coupling, and long-term implications |
| `pragmatist` | Ship-it engineer | What's the fastest working solution right now, minimal viable fix, unblocks immediately |

## Quick Start

```bash
# All 4 specialists on a hard problem
nu ~/.claude/skills/tiger-team/tiger-team.nu \
  --problem "Why won't i9 federate with Pi workers in BEAM cluster"

# Just 2 specialists for a quick take
nu ~/.claude/skills/tiger-team/tiger-team.nu \
  --problem "Auth middleware stores session tokens in plaintext" \
  --roles [attacker pragmatist]

# Dry run to see what would be spawned
nu ~/.claude/skills/tiger-team/tiger-team.nu --dry-run \
  --problem "Debug why BEAM cluster won't federate"
```

## How Results Are Synthesized

After all specialists finish (or you've read enough):

1. Read each pane's output — look for convergence and divergence
2. Where specialists **agree**: high-confidence path
3. Where they **disagree**: the interesting tension that needs your judgment
4. The pragmatist's answer is your "do this now"; the architect's is your "do this right"
5. The attacker found the risks; the defender found the guardrails

## Layout

Specialists open in a **new Zellij tab** named `tiger-<timestamp>`:
- 2 specialists: side by side
- 3-4 specialists: 2x2 grid

## Agent Flags

Same as zellij-team: workers run with `--dangerously-skip-permissions --permission-mode bypassPermissions`.
You are watching and can ctrl-c any pane.
