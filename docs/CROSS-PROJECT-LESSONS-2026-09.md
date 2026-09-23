# Cross-project lessons — 2026-09

The skills repo is the **policy/workflow layer**, not canonical runtime state.

## Important reuse

- `quota-gate` is the deterministic baseline for BOP provider routing before Jev/System One.
- `bop-on` captures the O(N) rule; preserve it for any new metrics/events/lineage hooks.
- `apfs` contains COW/clone lessons relevant to BOP storage comparisons.
- `gantt` is a useful execution-history view, but history should come from BOP/filesystem-native facts when possible.
- `dispatch-nu` and related dispatch skills should converge on the same Copilot/Codex fallback policy.

## Boundary

Skills may recommend/route/visualize. They should not become the source of truth for run state, filesystem version, or lineage.

## Agent assignment

Copilot is primary. Codex fallback is `@codex` issue/PR delegation.
