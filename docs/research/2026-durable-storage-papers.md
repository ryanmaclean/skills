# 2026 papers relevant to durable-tid / smolFire

Updated: 2026-09-23

This is a curated reading list for the lower-bound durable ordering, storage, I/O, FPGA/ASIC, and formal-verification work.

## Directly relevant

### UnICom: A Universally High-Performant I/O Completion Mechanism for Modern Computer Systems — FAST '26

Riwei Pan et al., USENIX FAST 2026.

Why it matters:
- argues that kernel traps themselves are often not the dominant storage cost
- centralizes completion polling
- combines polling and kernel assistance instead of assuming pure kernel bypass always wins
- useful counterweight to an overly io_uring-centric design

Implication for us:
- keep submission/completion transport separate from durable-tid semantics
- benchmark polling, interrupts, and kernel-assisted paths rather than assuming one mechanism wins

### Oxbow: A Coordinated Architecture for Multi-Component File Systems — OSDI '26

Jongyul Kim et al., USENIX OSDI 2026.

Why it matters:
- splits filesystem work across kernel, userspace, and device
- uses a kernel-bypassing write path while retaining kernel read/interoperability
- introduces Split Journaling and DMA-based snapshots/staging

Implication for us:
- validates examining different ownership boundaries for read, write, metadata, and commit
- our FPGA/ASIC work should not assume the whole storage stack moves into hardware

### io_uring in Oracle Database: A Hybrid Storage I/O Architecture at Production Scale — arXiv 2609.22781

Rajarshi Chowdhury et al., September 2026.

Why it matters:
- per-process ring contexts avoid inter-process synchronization
- shared registered buffers
- measured gains concentrate on batched asynchronous writes, not all I/O

Implication for us:
- do not build a general io_uring clone
- rings are useful transport for selected hot paths
- synchronous/simple paths may remain simpler and cheaper

### Crash consistency in an NVM-enabled hybrid storage system: Problems, solutions, and verification — Journal of Systems Architecture, 2026

Why it matters:
- explicitly models write ordering, granularity, and parallel-write crash-consistency problems
- uses PlusCal/TLA+ model checking to validate the design

Implication for us:
- formalize durable-tid before RTL
- distinguish volatile acceptance, write completion, durability, and visible completion
- TLA+/PlusCal is directly appropriate for our commit FSM

### Testing Storage-System Correctness: Challenges, Fuzzing Limitations, and AI-Augmented Opportunities — arXiv 2602.02614

Ying Wang, Jiahui Chen, Dejun Jiang, February 2026.

Why it matters:
- storage bugs depend on long-horizon state, nondeterministic interleavings, durability, ordering, and recovery
- conventional fuzzing alone is insufficient

Implication for us:
- SS1-C fault injection should be state-aware/model-guided
- formal properties + differential simulation + hardware fault injection is stronger than random fuzzing alone

### C3: Finding Counting-related Crash Consistency Atomicity Bugs for Persistent Memory — ACM TOS, 2026

Why it matters:
- focuses on atomicity relationships between logical metadata and persistent objects

Implication for us:
- treat TID allocation, commit marker, payload persistence, and completion visibility as correlated state
- explicitly verify cross-field atomicity, not only monotonic counters

## Hardware / FPGA / storage co-design

### BoostX-NTI: Fast, Scalable and Flexible Storage Architecture with NVMe/TCP Initiator Acceleration — ISCA 2026

Why it matters:
- moves an entire storage/network I/O path into FPGA/DPU hardware
- removes host CPU and off-chip-memory traversal from the hot path
- supports software fallback/recovery

Implication for us:
- ASIC/FPGA should target bookkeeping/data movement that can be eliminated from CPU execution
- retain a software fallback oracle

### Zero2M: Optimizing Tenant-Level I/O Management for Future Faster NVMe Storage with FPGA — ACM TRETS, 2026

Why it matters:
- explores removing CPU participation from I/O-management paths with FPGA logic

Implication for us:
- compare CPU-core sequencer vs FPGA sequencer before ASIC
- quantify whether deterministic transaction bookkeeping is worth offload

## Related but more indirect

### MlsDisk: Trusted Block Storage for TEEs Based on Layered Secure Logging — FAST '26

Why it matters:
- out-of-place logging
- layered design to make integrity/freshness/consistency easier to reason about

Implication:
- reinforces separating the minimal durable log from higher-level indexing/namespace views

### WriteGuards: Distributed Storage Support for Strongly Consistent Caches — OSDI '26

Why it matters:
- small fencing values prevent delayed writes after ownership changes

Implication:
- useful when we reach multi-node durable-tid
- question whether a small epoch/fence plus local TID is sufficient instead of global total ordering

### TARL: Transaction-Aware Reliable Ledgers for Executable Memory Management in Long-Term Agents — arXiv 2608.03699

Why it matters:
- agent memory is treated as explicit state transitions rather than a binary write/hold decision

Implication:
- conceptually supports treating agent state as typed transitions/ledger actions
- this is inspiration only; it is not storage-layer evidence

## Important older anchors

Not 2026, but keep near the 2026 papers:

- WOFS (OSDI '25): metadata packaged once with a single ordering point
- PoWER (OSDI '25): tool-agnostic verification of crash consistency
- crash-consistency models / Ferrite (ASPLOS '16)
- xNVMe (2024): unified message-passing abstraction across Linux, FreeBSD, SPDK, etc.
- HAMMER/HAMMER2 papers/docs as semantic prior art for persistent transaction IDs/history

## Reading order

1. UnICom
2. Oxbow
3. Oracle io_uring
4. NVM crash consistency + PlusCal/TLA+
5. C3
6. Storage correctness survey
7. BoostX-NTI / Zero2M
8. MlsDisk / WriteGuards

## Design warning

These papers support individual mechanisms, not our complete architecture. The durable-tid project should still start from the smallest semantic contract and measure every layer added above it.
