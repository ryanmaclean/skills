---
name: lower-bound-review
description: >
  Architecture review gate for Ryan's lower-bound systems research. Use before
  proposing a new queue, scheduler, database, storage layer, FPGA block, RISC-V
  core, ASIC function, lineage store, runtime, or cross-repo primitive. Forces
  prior-art/research-index checks, lowest-layer ownership, BSD/no-Linux target,
  permissive licensing, storage semantics, TEE-style trust boundaries, and a
  statement of what layer the proposal deletes.
---

# lower-bound-review — delete layers before adding them

Run this before proposing architectural machinery.

## Step 1 — read what is already decided

Check:

- `docs/CONSTRAINTS.md`
- `docs/RESEARCH-INDEX.md`
- `docs/PROJECTS.md`
- target repo `.project.toml`
- target repo `AGENTS.md`

Do not repeat prior-art research already indexed unless:
- new evidence exists
- a relevant standard/project materially changed
- the requirement changed
- measurements contradict the previous conclusion

Record the changed assumption when reopening a conclusion.

## Step 2 — identify the primitive

State the desired invariant without implementation vocabulary.

Bad:

```
we need an io_uring-like ring
```

Better:

```
nothing may become observably complete before durability is established
```

Then ask whether CPU, cache coherence, DRAM, BSD VM/DMA APIs, filesystem, storage controller, or existing permissive IP already owns it.

## Step 3 — lowest owner

Prefer the lowest layer that can own the primitive once.

Existing ownership includes:

- work/run identity -> BOP
- execution -> Moth
- runtime/OS/storage research -> smolFire
- build/deploy/receipts -> Genoa
- policy/workflow -> Skills
- UI/control -> VibeCode
- durable history/version -> filesystem or lower durable primitive
- observation -> Datadog
- lineage -> OpenLineage projection

## Step 4 — free primitives inventory

Before custom hardware, explicitly inventory what comes for free:

CPU/SoC:
- atomics
- cache coherence
- memory barriers
- L1/L2
- DRAM
- vector execution where available
- DMA engines
- MMU/IOMMU or capability mechanisms

BSD:
- mmap/shared memory
- bus_dma synchronization
- kqueue/AIO where relevant
- filesystem durability/history primitives
- device mappings

Use these before inventing another coherence, queue, DMA, or notification layer.

## Step 5 — ARM/CPU vs FPGA

CPU/BSD by default:
- policy
- branch-heavy logic
- orchestration
- filesystems
- recovery strategy
- diagnostics
- crypto/PQC initially
- bulk/vector work
- rapidly changing code

FPGA by exception:
- small deterministic invariant
- high-frequency fixed function
- independent trusted state
- completion gating
- anti-replay/epoch
- measured operation that benefits from hard real-time/determinism

The FPGA must earn each function.

## Step 6 — trust/storage questions

Apply storage-controller and TEE thinking:

- what exactly is durable?
- who can lie about completion?
- can stale state replay?
- what survives reset/power loss?
- what is immutable?
- what is independently measurable?
- can the host forge trusted evidence?

## Step 7 — licensing/platform gate

Target architecture:
- no Linux dependency
- prefer FreeBSD/NetBSD/DragonFly/OpenBSD/bare-metal as appropriate
- project/dependency preference: MIT/BSD/Apache
- reject GPL/LGPL/AGPL dependencies for new core work unless explicitly approved

Linux projects may be comparative prior art, not target architecture.

## Step 8 — proposal must delete something

Every new architectural layer must answer:

1. What invariant does it uniquely own?
2. Why can't an existing lower layer own it?
3. What duplicated state/layer does it remove?
4. What happens if we do nothing?
5. Can the same experiment be modeled in JS first?
6. Can pure RTL test it before adding RISC-V?
7. What measurement would cause us to delete this proposal?

If it only adds another representation of existing truth, reject it.

## Step 9 — preferred experiment order

```
small semantic model
-> JS exhaustive/fault model
-> formal specification
-> native software control
-> pure RTL
-> FPGA
-> optional CPU/RISC-V control plane
-> ASIC only if measured advantage
```

## Output format for reviews

Return:

```
Primitive:
Existing lower-layer support:
Novel missing invariant:
Canonical owner:
JS model:
FPGA minimum:
What stays on CPU/BSD:
What this deletes:
Prior art already checked:
New evidence, if any:
Measurement / kill criterion:
```

Keep the answer concrete and short.
