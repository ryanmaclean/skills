# Ryan Lab project registry

## Mandatory reading

Before architecture or cross-project work:

- [Binding architectural constraints](CONSTRAINTS.md)
- [Research already completed / do-not-repeat index](RESEARCH-INDEX.md)
- [Cross-project glossary](GLOSSARY.md)

These documents are authoritative unless new evidence or requirements are recorded explicitly.

This is the temporary cross-project registry until/unless a dedicated portfolio repository is created.

## Lifecycle values

- `active` — current implementation work
- `experimental` — active research/prototype
- `reference` — comparison/prior-art repo; do not make it a dependency without a specific reason
- `maintenance` — maintained for compatibility, not a primary research target
- `archived` — historical only
- `mirror` — read-only copy of a canonical repo kept for locality or availability (e.g. LAN release-asset cache); never accepts direct writes and never holds facts the canonical repo lacks

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
| `datadog-agent-freebsd` | active | Datadog Agent BSD port: patch series, native build recipes, gates, release definitions. Build/deploy receipts are Genoa-owned (`genoa.receipt.v1`, RFC 0006); it does not keep its own receipt store. |

## Reference/control repos

| Repo | Status | Use |
|---|---|---|
| `gastown` | reference | orchestration complexity/control baseline |
| `tundra` | reference | broad orchestration/telemetry patterns |
| `autoresearch` | reference | fixed-budget autonomous experiment method |
| `dd-agent-FreeBSD` | reference | prior FreeBSD Datadog lessons (FreeBSD ports-Makefile lineage, v7.2x). Not the active port; see `datadog-agent-freebsd`. |

## Mirrors

| Repo | Status | Mirror of | Use |
|---|---|---|---|
| QNAS Gitea `studio/datadog-agent-FreeBSD` | mirror (after RFC 0006 migration) | `datadog-agent-freebsd` releases | LAN release-asset cache for fleet installs. Its legacy `master`/open PRs and `builds/ledger.jsonl` are frozen history; nothing new is written there first. |

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
