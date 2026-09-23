# Research already completed / do-not-repeat index

Updated: 2026-09-23

Use this index before starting architecture research. Re-run only when new evidence, a new requirement, or a changed implementation materially alters the question.

## Durable ordering / storage semantics

Already researched:
- HAMMER1 transaction IDs/history
- HAMMER2 snapshots/PFS/version semantics
- NetBSD LFS checkpoints/history direction
- FFS/WAPBL/fss baseline
- raw append-log lower-bound idea
- namespace-vs-identity distinction

Canonical docs:
- rfcs/0002-filesystem-is-canonical-state.md
- scaffolds/durable-tid/docs/DESIGN.md
- scaffolds/durable-tid/docs/PRIOR_ART.md
- docs/CONSTRAINTS.md

## Submission/completion mechanisms

Already researched:
- io_uring
- FreeBSD AIO/kqueue
- FreeBSD buf_ring
- netmap rings
- NVMe SQ/CQ
- SPDK
- VirtIO queue model
- DragonFly LWKT/message-port concepts

Conclusion:
- ring/queue is transport, not semantic core

Canonical docs:
- scaffolds/durable-tid/docs/PRIOR_ART.md
- docs/CONSTRAINTS.md

## Formal verification

Already decided:
- model before RTL
- TLA+/PlusCal semantic spec
- JS executable oracle/state exploration
- SystemVerilog assertions/SymbiYosys
- hardware differential/fault testing

Canonical docs:
- rfcs/0005-formal-verification-before-rtl.md
- scaffolds/durable-tid/docs/FORMAL.md
- smolFire/docs/FORMAL-DURABLE-TID-PLAN.md

## FPGA / ASIC path

Already decided:
- three SuperStation Ones are the pre-ASIC lab
- pure RTL first
- BRAM-only first
- persistence later
- ASIC only after measured FPGA benefit

Canonical docs:
- rfcs/0004-three-superstation-pre-asic-lab.md
- scaffolds/durable-tid/docs/HARDWARE.md
- smolFire/docs/SUPERSTATION-PRE-ASIC-PLAN.md

## Open RISC-V / reusable IP

Already researched:
- lowRISC Ibex
- OpenTitan reusable IP
- LiteX
- PicoRV32
- CVA6 / CV-X-IF
- FROST
- VexRiscv/VexiiRiscv licensing caveat
- CHERI-RISC-V capability/DMA direction
- NVIDIA BlueField RISC-V DPA
- Intel/Altera IPU
- ScaleFlux computational storage
- Samsung SmartSSD

Conclusion:
- RISC-V is optional control plane, not v0 datapath
- Ibex/OpenTitan is default reuse track
- custom ISA only after MMIO measurements

Canonical docs:
- docs/research/open-riscv-durable-tid-2026.md
- scaffolds/durable-tid/docs/REPLAN-RISCV.md
- smolFire/docs/RISCV-DURABLE-TID-REPLAN.md

## 2026 papers

Already collected:
- UnICom
- Oxbow
- Oracle io_uring production architecture
- NVM crash-consistency + PlusCal/TLA+
- C3
- storage-correctness survey
- BoostX-NTI
- Zero2M
- MlsDisk
- WriteGuards

Canonical doc:
- docs/research/2026-durable-storage-papers.md

## Cross-project ownership

Already decided:
- BOP: work/run identity + filesystem state-machine orchestration
- Moth: reusable execution harness
- smolFire: runtime/OS/storage research
- Genoa: build/deploy/receipts
- Skills: policy/workflow/shared research registry
- VibeCode: UI/control surface
- filesystem/durable primitive: persistence/version truth
- Datadog: observation
- OpenLineage: projection/interchange

Canonical docs:
- docs/PROJECTS.md
- docs/GLOSSARY.md
- rfcs/0001-one-home-per-primitive.md

## Rule for agents

Before starting new research, search this index and linked canonical docs.

If the question is already answered, extend the existing result rather than re-running the same survey.

If challenging an existing conclusion, record:
- new evidence
- what assumption changed
- expected architectural consequence
