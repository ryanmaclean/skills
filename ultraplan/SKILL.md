---
name: ultraplan
description: Council of planners — spawn 4 specialist agents (architect, operator, security, pragmatist) to debate and produce a unified implementation plan. Use for major architectural decisions that affect the fleet.
---

# UltraPlan — Council of Planners

Spawn 4 specialist agents with opposing mandates to stress-test a plan before committing. Each agent argues from their perspective. The orchestrator synthesizes into a unified plan with dissents noted.

## When to Use
- Major architectural changes (embed nu-engine, switch LLM providers, new OS)
- Cross-cutting decisions that affect all fleet nodes
- When "just do it" risks bricking the fleet

## Council Members

### 1. Architect
- Mandate: Elegant design, minimal complexity, clean interfaces
- Asks: "What's the simplest architecture that solves this?"
- Rejects: Over-engineering, premature abstraction, unnecessary layers

### 2. Operator
- Mandate: Reliability, observability, rollback capability
- Asks: "How do I debug this at 3am when it breaks?"
- Rejects: Black boxes, missing logs, no rollback path

### 3. Security
- Mandate: Supply chain integrity, least privilege, attack surface
- Asks: "What can go wrong? What's the blast radius?"
- Rejects: Trusted binaries, shared secrets, network exposure

### 4. Pragmatist
- Mandate: Ship it. Time-to-value. Incremental delivery.
- Asks: "What's the smallest useful increment we can deploy today?"
- Rejects: Big-bang migrations, analysis paralysis, perfection

## Process

1. **Brief**: Orchestrator states the problem and constraints
2. **Independent analysis**: Each agent writes their perspective (parallel)
3. **Debate**: Cross-review — each agent critiques the others
4. **Synthesis**: Orchestrator merges into one plan with:
   - Consensus items (all agree)
   - Compromises (2-3 agree, dissent noted)
   - Blockers (any agent can veto with justification)
5. **Output**: Single plan document with phases, tasks, and risk register

## Invocation

```
/ultraplan <topic>
```

The orchestrator spawns 4 agents, collects their analyses, runs the debate round, and produces the final plan.

## Verification contract (auto-injected by proveryay)

Every agent dispatched through this skill receives the following appended
to its prompt automatically:

<see ~/.claude/skills/proveryay/CLAIMS-TEMPLATE.md — injected verbatim>

This means **every agent MUST emit a `<claims>` block when it makes any
deployment/state-change assertion.** The proveryay SubagentStop hook will
verify each claim in parallel and inject structured verdicts into the
dispatcher's next-turn context.

If the agent's work is purely planning / research / narrative (no state
changes — typical for ultraplan), the agent may omit the `<claims>` block.
Proveryay treats absence as silent (no noise).

Invocation example with template injection:

```sh
# Prompt injection: template auto-appended (idempotent)
PROMPT="${ORIGINAL_PROMPT}

$(cat ~/.claude/skills/proveryay/CLAIMS-TEMPLATE.md)"
```

See:
- `~/.claude/skills/proveryay/SKILL.md` — hook behavior
- `~/.claude/skills/proveryay/CLAIMS-TEMPLATE.md` — canonical format
- `~/.claude/skills/fleet-eval/SKILL.md` — probe taxonomy
