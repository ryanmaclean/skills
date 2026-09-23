# Ryan Lab project registry

This is the temporary cross-project registry until/unless a dedicated portfolio repository is created.

## Lifecycle values

- `active` — current implementation work
- `experimental` — active research/prototype
- `reference` — comparison/prior-art repo; do not make it a dependency without a specific reason
- `maintenance` — maintained for compatibility, not a primary research target
- `archived` — historical only

## Active system

| Repo | Status | Canonical ownership |
|---|---|---|
| `bop` | active | card/run identity, filesystem state machine, dispatcher lifecycle |
| `moth` | active | tiny agent execution harness, actors, subagents, session/tools |
| `smolfire` | active | smallest BSD runtime/microVM and filesystem/runtime benchmark harness |
| `genoa` | active | artifact/image construction, deployment, verification, receipts |
| `skills` | active | workflow/policy/evaluation layer and this registry |
| `vibecode-webgui` | active | IDE/UI/control surface |
| `agent-jail` | experimental | BSD jail isolation control |

## Reference/control repos

| Repo | Status | Use |
|---|---|---|
| `gastown` | reference | orchestration complexity/control baseline |
| `tundra` | reference | broad orchestration/telemetry patterns |
| `autoresearch` | reference | fixed-budget autonomous experiment method |
| `dd-agent-FreeBSD` | reference | prior FreeBSD Datadog lessons |

## Ownership rule

Prefer the lowest layer that can own a primitive once.

- work/run identity → BOP
- agent execution → Moth
- OS/runtime → smolFire
- jail isolation → agent-jail
- build/deploy/provenance receipts → Genoa
- policy/skills → skills
- user-facing IDE/control surface → VibeCode
- history/version truth → filesystem
- experiment methodology → autoresearch
- observation → Datadog
- lineage interchange → OpenLineage

Every participating repo carries a `.project.toml` pointing back here.
