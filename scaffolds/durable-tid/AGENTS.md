# AGENTS.md

## Role

Own only the minimal durable commit/order primitive.

## Contract

```
append
commit -> tid
recover -> last_tid
read(tid)
```

## Do not climb upward

Do not add:
- cards
- agents
- job queues
- filesystem paths
- HTTP
- MCP
- OpenLineage persistence
- generic io_uring-style opcodes

## Do not descend prematurely

Do not optimize for:
- VirtIO
- NVMe
- FPGA
- SmartNIC

until the semantic baseline is measured.

## Prior art

Read `docs/PRIOR_ART.md`.

## Licensing

BSD-2-Clause project code only.
No GPL/LGPL/AGPL dependencies.

Copilot primary; @codex fallback.
