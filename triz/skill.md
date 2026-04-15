---
name: triz
description: >
  TRIZ contradiction matrix oracle for software and systems design.
  Use when improving one quality worsens another — speed vs correctness,
  isolation vs sharing, autonomy vs control, simplicity vs completeness.
  Trigger phrases: "tradeoff", "tension between", "can't have both",
  "worsens when", "at the cost of", "faster but less reliable",
  "more isolated but harder to share". Not for bugs, merge conflicts, or tool choices.
  Prerequisite: triz binary must be installed (run `triz doctor` to verify).
---

# TRIZ Oracle

## The contradiction test

"If I make it more X, it becomes less Y." If yes — TRIZ applies.
If not a genuine tension, use a different tool (see below).

## Workflow

Copy this checklist and check off as you go:

```
Contradiction Analysis:
- [ ] 1. State the contradiction: "improving X worsens Y"
- [ ] 2. Map to TRIZ parameters — see [parameters.md](parameters.md)
- [ ] 3. Run: scripts/solve.nu "X" "Y"
- [ ] 4. Review recommended principles — see [principles.md](principles.md)
- [ ] 5. Apply: write the specific solution using the best principle
- [ ] 6. Verify: does it actually resolve the tension without introducing a worse one?
```

**Feedback loop:**
- Contradiction unclear → run `scripts/whys.nu` first
- Parameter must be X AND not-X simultaneously → use [references/physical_contradictions.md](references/physical_contradictions.md) instead of the matrix
- Empty matrix cell → reframe the parameters; see [references/examples.md](references/examples.md) for analogies
- Principle doesn't fit the domain → try the next one in the recommendation list
- None fit → run `scripts/fmea.nu PRINCIPLE_ID` to understand failure modes
- Contradiction keeps recurring → check [references/evolution_trends.md](references/evolution_trends.md) — may be structural
- Unsure how deep to search → check [references/innovation_levels.md](references/innovation_levels.md) first

## Scripts

| Script | Use |
|--------|-----|
| `scripts/solve.nu "X" "Y"` | Matrix lookup + principle application |
| `scripts/solve.nu "X" "Y" -c "context"` | With situational context |
| `scripts/solve.nu "X" "Y" --json` | Machine-readable output |
| `scripts/whys.nu` | 5 Whys drill → articulate contradiction interactively |
| `scripts/whys.nu --json` | Whys output with ready `solve_cmd` field |
| `scripts/fmea.nu 35` | Failure mode analysis for principle 35 |
| `scripts/fmea.nu --stdin` | Pipe from `solve.nu --json` |
| `scripts/lookup.nu "speed"` | Find matching parameter by keyword |
| `scripts/lookup.nu --principle 35` | Explain a principle |

Scripts are executable (`chmod +x`). Run from the skill directory or by full path.

## Core architectural rule

The matrix lookup is always first. LLM applies the principle after lookup.
Never ask the LLM to classify the contradiction before the lookup.

## Quick reference — top principles for software

| # | Principle | Software pattern |
|---|-----------|-----------------|
| 1 | Segmentation | Microservices, feature flags, namespaces |
| 2 | Extraction | Separate config from code; pure functions |
| 5 | Combining | Batch operations; co-location |
| 6 | Universality | One format, many consumers (e.g. Iceberg) |
| 10 | Preliminary action | Schema migrations before deploys; pre-warming |
| 13 | Inversion | Pull not push; reactive not polling |
| 15 | Dynamics | Adaptive timeouts; circuit breakers |
| 23 | Feedback | Observability gates; acceptance criteria in spec |
| 24 | Intermediary | Message queues; API gateways |
| 25 | Self-service | Self-healing systems; auto-remediation |
| 35 | Parameter changes | Feature flags; runtime configuration |
| 40 | Composite materials | Tiered storage; hybrid consistency models |

## Common software contradictions — cheatsheet

Real matrix cell values. Use when `triz` binary is unavailable.
Format: **improving → worsening**: principles (in priority order)

| Improving | Worsening | Principles | Key idea |
|-----------|-----------|-----------|----------|
| Speed (9) | Reliability (27) | 11, 35, 27, 28 | Cushion; param change; copy of object |
| Speed (9) | System complexity (36) | 10, 28, 4, 34 | Preliminary action; copies; discard/recover |
| Speed (9) | Build/test time (32) | 12, 36, 18, 31 | Equipotentiality; phase change; thermal expansion |
| Speed (9) | Automation coverage (38) | 13, 35, 8, 1 | Inversion; param change; anti-weight; segmentation |
| Reliability (27) | Adaptability (35) | 3, 35, 10, 40 | Local quality; param change; prelim action; composite |
| Reliability (27) | Build time (32) | 35, 3, 22, 39 | Param change; local quality; convert harm to benefit |
| Automation (38) | Reliability (27) | 11, 2, 13, 39 | Cushion; extraction; inversion; pneumatic/hydraulic |
| Automation (38) | System complexity (36) | 35, 28, 6, 37 | Param change; copies; universality; thermal expansion |
| Adaptability (35) | System complexity (36) | 24, 26, 28, 18 | Intermediary; copying; copies; mediation |
| System complexity (36) | Reliability (27) | 1, 24, 6, 35 | Segmentation; intermediary; universality; param change |
| Ease of operation (33) | System complexity (36) | 35, 1, 3, 24 | Param change; segmentation; local quality; intermediary |
| Observability (18) | System complexity (36) | 2, 35 | Extraction; param change |

**How to read:** improving Speed while worsening Reliability → try P11 (Cushion/Beforehand compensation) first, then P35 (Parameter changes), etc.

**Without the binary:** look up the improving/worsening pair above, then read the principle definition from [references/principles.md](references/principles.md).

## References (load on demand)

| File | Load when |
|------|-----------|
| [references/principles.md](references/principles.md) | Need full 40 principles with approach/sub-principles |
| [references/parameters.md](references/parameters.md) | Need full 39 parameters to map a concept |
| [references/examples.md](references/examples.md) | Need domain analogies for an empty matrix cell |
| [references/physical_contradictions.md](references/physical_contradictions.md) | Parameter must be X AND not-X simultaneously |
| [references/evolution_trends.md](references/evolution_trends.md) | Contradiction keeps recurring; predicting next bottleneck |
| [references/innovation_levels.md](references/innovation_levels.md) | Calibrating search depth; setting expectations |

## When TRIZ does not apply

| Situation | Better tool |
|-----------|-------------|
| Merge conflict | Understand semantic intent; resolve manually |
| Which library to use | Capability matrix comparison |
| Bug in code | Systematic debugging; read the error |
| Naming | User research; naming criteria checklist |
| Priority between features | Planning poker; value/effort matrix |
| Two options, no tension | Just pick one |
