---
name: durable-commit
description: >
  Design/review gate for durable storage, trusted completion, FPGA/TEE persistence,
  crash consistency, commit sequencing, epochs, TIDs, anti-replay, and storage
  acknowledgements. Use when discussing commit, fsync, FUA, flush, persistence,
  durable completion, storage controllers, FPGA sequencing, TEE/RPMB-like state,
  immutable logs, or attested storage. Rejects designs that confuse acceptance,
  visibility, device completion, and durability.
---

# durable-commit — trusted persistence semantics

Use this skill whenever a design crosses a persistence boundary.

## Mandatory model

Keep these states distinct:

```
proposal
!= CPU/program visibility
!= coherent/DMA visibility
!= device acceptance
!= device completion
!= durable persistence
!= trusted visible completion
```

Never use one as evidence for another unless the underlying device/OS contract explicitly guarantees it.

## Canonical identities

Keep these concepts separate:

```
run_id       logical execution identity
request_id   idempotency identity
object_id    durable object identity
tid          durable order/version
content_hash content identity
epoch        reset/trust domain
timestamp    observation only
path         presentation only
```

A useful evidence tuple is:

```
(epoch, tid, object_id, content_hash)
```

Higher layers may attach run/request identity.

## Commit rule

A completion may become visible only after its declared durability predicate is true.

```
TrustedComplete(N) => Persistent(N)
```

For ordered commits:

```
TrustedComplete(N) => required predecessors are also persistent
```

## Immutability

Committed facts are append-only.

Corrections and supersession create new committed facts; they do not rewrite history.

Prefer a minimal trusted head:

```
(epoch, last_tid, root_hash)
```

with immutable history stored outside the trusted block.

## Retry rule

Retries reuse the same logical `request_id`.

The authoritative persistence layer decides whether that request has already committed.

Do not turn transport retries into new logical mutations.

## Recovery review

For every state variable classify:

- volatile
- durable
- reconstructable
- forbidden to reconstruct

Recovery must never invent a commit.

## Hardware boundary

FPGA/ASIC logic should own only invariants that earn dedicated hardware.

Good candidates:
- completion gating
- monotonic durable sequence
- epoch / anti-replay
- tiny integrity root
- independent progress/error state

Keep on CPU/BSD unless measurement proves otherwise:
- filesystem
- policy
- recovery strategy
- networking
- agent runtime
- PQC protocol
- generic DMA/SHA/ring implementations

## Storage-background questions

Before accepting a design, ask:

1. What does the media/controller call "complete"?
2. Is volatile controller cache involved?
3. What do FUA/flush/barrier semantics actually guarantee?
4. What happens across power loss?
5. What is the authoritative persistence acknowledgement?
6. Can stale completions or old epochs be replayed?
7. Can a compromised host forge the claimed completion?

## TEE / trusted-boundary rule

Treat the lower block as a root of durable truth, not merely an accelerator.

The host may propose a mutation. It must not be able to fabricate a trusted committed result the lower boundary did not observe.

## Do not build

Reject by default:
- duplicate lineage/state databases
- hardware filesystem/path semantics
- general-purpose async I/O APIs
- queue/ring semantics baked into the persistence contract
- Linux as a target architecture
- PQC in the per-commit hot path
- a RISC-V softcore before pure RTL earns the need

## Canonical references

Read before changing the architecture:

- `docs/CONSTRAINTS.md`
- `docs/RESEARCH-INDEX.md`
- `scaffolds/durable-tid/docs/DESIGN.md`
- `scaffolds/durable-tid/docs/FORMAL.md`
- `rfcs/0004-three-superstation-pre-asic-lab.md`
- `rfcs/0005-formal-verification-before-rtl.md`
