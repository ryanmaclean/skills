# Cross-project architectural constraints

Status: Binding design constraints for active research  
Updated: 2026-09-23

These constraints capture conclusions already reached through repo review, prior-art research, formal-method planning, and lower-layer modeling. Do not re-open them casually. Revisit only with new evidence, measurements, or a clearly different requirement.

## C1 — One primitive, one canonical owner

Prefer the lowest layer that can own a primitive once.

Higher layers may project, index, visualize, or cache a fact, but they must not silently become a second source of truth.

## C2 — Separate identity, order, content, durability, and presentation

Do not collapse these concepts.

- logical identity: e.g. run/request/object ID
- durable order/version: monotonic TID/sequence
- content identity: hash
- durability: explicit acknowledgement that persistence conditions are satisfied
- presentation: path, dashboard, UI, lineage view, event stream

A useful minimal provenance tuple is:

```
(run_id, tid, content_hash)
```

## C3 — Visibility is not durability

Accepted, written, write-complete, durable, and visible-complete are distinct states.

Never expose completion merely because a request was accepted or a write returned.

```
acceptance
!= write completion
!= durability
!= visible completion
```

## C4 — Push invariants down; pull policy up

Lowest viable layers own:
- ordering
- atomicity
- durability
- integrity
- idempotence

Higher layers own:
- routing
- model/provider choice
- retries as policy
- budgets
- approvals
- UI/presentation

## C5 — Ordering must have one source

Do not create independent ordering/version counters in BOP, telemetry, lineage, or UI if a lower durable layer already provides one.

Wall-clock timestamps are metadata for humans/observation, not correctness primitives.

## C6 — Retries reuse operation identity

A retry should resubmit the same logical operation identity wherever possible.

Do not create a new logical operation merely because transport or execution retried.

The authoritative layer decides whether the operation already committed.

## C7 — Completion is an explicit acknowledgement

"Not running" or "no longer present" is not completion.

Completion means a stated invariant has been satisfied.

For durable work, completion must be downstream of durability acknowledgement.

## C8 — Queues and rings are transports

A queue, ring, socket, io_uring, VirtIO queue, NVMe queue pair, or FPGA FIFO must not define application semantics.

Semantics must survive transport replacement.

## C9 — Backpressure is explicit

Expose accepted/busy/rejected or equivalent state.

Do not rely on latency growth, hidden buffering, or retry storms as the primary backpressure mechanism.

## C10 — Recovery starts at the crash boundary

For every component, explicitly classify:
- volatile state
- durable state
- reconstructable state
- state that must never be reconstructed

Recovery must never invent a committed transition.

## C11 — Derived views are disposable

Datadog, OpenLineage, GitHub summaries, VibeCode views, dashboards, and event projections should be rebuildable from canonical state.

Observation must not become canonical state accidentally.

## C12 — Filesystem namespace is representation unless proven otherwise

Paths/directories are human-friendly views.

Do not assume path == identity.

Prefer:
```
object_id -> current path
```
over:
```
path -> identity
```

when designing durable primitives.

## C13 — Formalize before optimizing hardware

For durable-ordering primitives:

```
TLA+/PlusCal
-> executable JS model
-> exhaustive/fault state exploration
-> RTL
-> formal RTL assertions
-> FPGA
-> ASIC gate
```

Do not treat FPGA testing as a substitute for proof.

## C14 — The smallest primitive currently under study is durable ordered visibility

The current lower-bound question is not "how do we build a queue?" or "how do we build a filesystem?"

It is:

> What is the smallest mechanism that ensures nothing becomes observable out of durable order?

Candidate minimal models:
- hardware allocates monotonic TID
- caller supplies sequence and hardware only enforces durable ordered visibility

Both should be modeled before committing to identity allocation in hardware.

## C15 — Do not start with a CPU

RISC-V is an optional control plane around the primitive.

Pure RTL comes first.

If a CPU is later useful, prefer a small permissively licensed core such as Ibex and keep it off the commit datapath unless measurements justify otherwise.

## C16 — Reuse permissive hardware IP before writing generic blocks

Before creating CPU, DMA, SHA, FIFO, bus, or generic ring logic, evaluate existing MIT/BSD/Apache-compatible IP.

Current preferred research track:
- Ibex
- OpenTitan reusable IP
- LiteX as an integration harness where useful

## C17 — ASIC requires an evidence gate

Do not tape out because the design is elegant.

ASIC is justified only after FPGA measurements show a meaningful gain in at least one of:
- latency
- CPU cycles
- determinism
- power
- elimination of software layers

## C18 — Three-SuperStation differential lab is the default hardware validation setup

- SS1-A: software/reference oracle
- SS1-B: RTL/FPGA DUT
- SS1-C: fault injection

Use identical traces and compare recovered committed state.

## C19 — No duplicated lineage/state database by default

OpenLineage is a projection/interchange format.

Datadog is observation.

Neither is canonical durable state.

## C20 — New architecture proposals must state what layer disappears

A proposal that adds a layer must explain:
- what invariant it uniquely owns
- why an existing lower layer cannot own it
- what duplicated state it removes

If it only adds another representation of existing truth, reject or redesign it.
