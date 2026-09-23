# Cross-project glossary

Use these terms consistently across the active repos.

| Term | Meaning | Owner |
|---|---|---|
| Card | Durable unit of work/specification | BOP |
| Run | One execution attempt of a card | BOP |
| Worker | Runtime process executing a run | Moth/runtime |
| Session | Worker/model interaction context; not canonical work state | Moth |
| Agent | Model-driven worker role; avoid using as a synonym for card/run | Moth/BOP context |
| Transition | Durable BOP state move such as pending → running | BOP |
| Receipt | Evidence of artifact build/deployment/verification | Genoa |
| Runtime | OS/microVM/jail substrate in which a worker executes | smolFire / agent-jail |
| Version | Filesystem-native durable identity for a state/artifact view | filesystem |
| TID | Native transaction identity, notably HAMMER-style | filesystem |
| Checkpoint | Durable filesystem/log checkpoint, notably LFS-style | filesystem |
| Event | Derived observation of a fact; not canonical state | observability/export |
| Lineage | Relationship among runs, inputs, outputs, and versions | derived/export |
| Dataset | OpenLineage representation of an input/output object | OpenLineage projection |
| Policy | Routing/gating/recommendation logic | skills/Jev where applicable |
| Benchmark record | Shared machine-readable performance/correctness measurement | ryanlab.bench.v1 |

## Rules

1. A Run is not a Session.
2. A Receipt is not a Run.
3. An Event is not canonical state.
4. OpenLineage is a projection, not storage.
5. Filesystem history is not duplicated in BOP when a native identity is available.
6. Agent conversation state is never the sole source of project truth.
