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
