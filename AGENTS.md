# AGENTS.md

## Role

Agent policy/workflow/skills layer.

## Owns

- workflow guidance
- routing policy baselines
- agent skills/evals

## Do not duplicate

- canonical runtime state
- filesystem version store
- lineage database

## Sibling repos to consult first

- ryanmaclean/bop
- ryanmaclean/smolfire
- ryanmaclean/moth

## Cross-project context

Read `docs/CROSS-PROJECT-LESSONS-2026-09.md` before making architectural changes.

## Agent delegation

- Primary GitHub coding agent: Copilot when assignable/available.
- Fallback: delegate the issue or PR to Codex with `@codex`.
- Do not treat Copilot/Codex state as canonical project state; keep canonical work in repo issues/BOP/filesystem state.


## Binding cross-project constraints

Before proposing architecture or cross-project primitives, read:

- `ryanmaclean/skills/docs/CONSTRAINTS.md`
- `ryanmaclean/skills/docs/RESEARCH-INDEX.md`
- this repo's `.project.toml`

Do not repeat research already indexed unless new evidence or requirements materially change the conclusion.

New architecture proposals must identify:
- the invariant uniquely owned by the new layer
- why a lower existing layer cannot own it
- what duplicated state or layer the proposal removes

Key constraints include:
- visibility is not durability
- identity, order, content hash, and presentation remain distinct
- ordering has one source
- retries reuse logical operation identity
- queues/rings are transport, not semantics
- derived views are disposable
- push invariants down; pull policy up
- formalize durable primitives before RTL/FPGA
- RISC-V is optional control plane, not the v0 datapath
- ASIC requires measured FPGA justification
