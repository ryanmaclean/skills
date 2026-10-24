# skills
Skills for agents


## Project registry

This repository also serves as the temporary cross-project architecture and policy hub.

- [Project registry](docs/PROJECTS.md)
- [Cross-project glossary](docs/GLOSSARY.md)
- [RFC 0001: One home per primitive](rfcs/0001-one-home-per-primitive.md)
- [RFC 0002: Filesystem-native durable state](rfcs/0002-filesystem-is-canonical-state.md)
- [RFC 0003: OpenLineage is a projection](rfcs/0003-openlineage-is-a-projection.md)
- [Shared benchmark schema](schemas/bench.v1.schema.json)

Each participating repository carries a `.project.toml` pointing back here.


## Architecture guard skills

- [durable-commit](durable-commit/SKILL.md) — persistence, durability, trusted completion, epochs/TIDs, crash consistency
- [deterministic-replay](deterministic-replay/SKILL.md) — immutable committed facts, canonical encoding, replay, JS/formal/RTL equivalence
- [lower-bound-review](lower-bound-review/SKILL.md) — architecture gate before adding queues, schedulers, storage layers, FPGA blocks, RISC-V, or ASIC functions

These skills operationalize the binding constraints in [docs/CONSTRAINTS.md](docs/CONSTRAINTS.md) and should be used before extending lower-bound storage or hardware work.


## Skills imported from vibecode-webgui

Moved from ryanmaclean/vibecode-webgui@129302a73be9719d213561116319d13ed679fd71 on 2026-10-24
([import record](docs/imports/vibecode-webgui-2026-10-24.md)):

- [datadog-operations](datadog-operations/SKILL.md) — Datadog APM/logs/metrics/RUM queries, monitors, dashboards, incidents (MIT, Datadog CLI Contributors)
- [datadog-dashboard-ops](datadog-dashboard-ops/SKILL.md) — fetch, fix and push Datadog dashboards
- [gastown-ops](gastown-ops/SKILL.md) — Gas Town bead lifecycle, Kafka, Airflow, rig coordination
- [apple-silicon](apple-silicon/SKILL.md) — Metal/MLX/CoreML/Virtualization.framework tuning
- [openrouter-models](openrouter-models/SKILL.md) — pick a working free OpenRouter model
- [context-graph](context-graph/README.md) — codebase relationship graph builder
- [codex-personas](codex-personas/PERSONAS_README.md) — roundtable MCP persona definitions
