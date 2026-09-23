# RFC 0005 — Formal verification before RTL

Status: Accepted experimental plan  
Date: 2026-09-23

## Decision

The durable-TID protocol must be specified and model-checked before RTL implementation.

The semantic core is intentionally small enough to make formal verification practical.

## Core operations

```
append(record)
commit() -> tid
recover() -> last_tid
read(tid)
```

## Safety properties

At minimum prove:

- committed TIDs never decrease
- committed TIDs never repeat
- completion(tid) implies durable(tid)
- recovery never invents a commit
- allocated-but-uncommitted TIDs are never exposed as committed
- duplicate request identity commits at most once
- last_committed <= last_allocated

## Liveness

Under explicit assumptions:

- storage eventually acknowledges durability
- faults eventually stop
- the request remains valid

then an accepted request eventually completes.

Do not hide liveness assumptions.

## Verification ladder

1. TLA+/PlusCal semantic specification
2. model checking / exhaustive state exploration
3. JavaScript reference simulator checked against generated traces
4. RTL implementation
5. SVA/SymbiYosys proofs
6. FPGA differential testing
7. hardware fault injection
8. ASIC only after the RTL implementation is proven against the model

## Refinement requirement

A transport or implementation is correct only if it refines the durable-TID semantic model.

Examples:
- function-call implementation
- mmap/shared ring
- kernel interface
- VirtIO
- FPGA
- ASIC

The transport is not allowed to change commit semantics.

## Projection boundary

Datadog, OpenLineage, filesystem namespace views, and BOP events are projections. None may influence or redefine durable commit state.
