# Finite durable-TID semantic oracle

This model belongs to the experimental durable primitive described in ../docs/DESIGN.md and FORMAL.md. It owns no production state and adds no transport, filesystem, hardware, dependency or telemetry layer. Project code is BSD-2-Clause. Existing model.mjs and model.test.mjs are edited; no new helper script is introduced.

## Scope and bounds

Request IDs are 1..3, object IDs 1..2, values 0..1, operation1 means replace the selected object's value, result1 means committed, TIDs1..2 and trust epochs0..2. These finite model symbols are distinct identities, not production identifier widths or business operations. Allocation stops at TID capacity; reset/recovery refuses to advance beyond epoch capacity. Neither wraps, clamps a successful commit, nor reuses a committed TID. Capacity is a progress limit, not an implementation of unbounded counters.

submit reaches PROPOSED; make_visible separately allocates and reaches COHERENT_VISIBLE. device_complete is not persistence. Only durable_ack appends an immutable fact and advances the durable TID/root. publish_completion looks up the exact current request in the committed prefix, including required predecessors. It cannot reuse a previous request's positive scalar TID as evidence.

The eight requested actions remain; make_visible is added to represent the previously unreachable proposal boundary. Explicit noncurrent action epochs, including future epochs, are rejected. Reset and recovery advance the trust epoch before permitting another round, clearing volatile allocation/request/completion state while keeping facts and the committed lower bound. At maximum epoch they refuse the transition. There is no liveness claim, even under fairness, once a finite capacity is reached.

## Identity and immutable facts

The durable history is the single authority for request idempotence. Repeating an already committed request returns its original TID without a new fact, including after reset/recovery. Reusing that request ID with different object/operation/value rejects as identity_conflict. A conflicting pending request is also rejected; an unrelated pending request reports busy. An uncommitted proposal can be lost on reset and creates no durable identity binding. Completion receipts may name an older committed fact under the current trust epoch; that does not rewrite the original fact's epoch.

State classification:

- Durable authority: immutable history, trusted root/last committed TID and current trust epoch. The model contains both sides of the trust boundary; the history is conceptually external to the tiny trusted head.
- Reconstructable views: object values and the committed request-to-payload/TID index; these derive exclusively from immutable facts.
- Volatile: phase, pending request, uncommitted allocation, current visible completion and diagnostic outcome/duplicate marker.
- Forbidden reconstruction: an unacknowledged proposal is never promoted to a committed fact.

All returned model states and facts are recursively frozen. New facts are appended; transitions never edit old facts. This is an executable-language safeguard plus invariant checking, not physical immutability.

## Canonical model representation

There is no RECORD-V0.md in the pinned source tree. This model therefore defines its own explicitly versioned finite reference representation, not an approved production wire format or hardware ABI.

canonicalRequest encodes five unsigned8-bit integers in order: version1, request_id, object_id, operation, value. canonicalRecord encodes version, epoch, tid, request_id, object_id, operation, parent_tid, result, value as unsigned8-bit integers, then a uint32 big-endian byte count and ASCII bytes for content_hash. Parent TID must equal tid-1. Every field is required, extra fields reject, and there are no optional/default persisted fields. Bounds are stricter finite domains listed above. No floats, Unicode, platform-native layout or unordered JSON serialization is hashed.

The derived view encodes version1, uint32 big-endian object count, sorted uint8 object/value pairs, uint32 big-endian request count, then sorted uint8 request/object/operation/value/TID tuples. Duplicate IDs/TIDs, unsorted entries and values outside the finite domain reject. Hex text represents exact bytes; it is not an alternative byte encoding. Symbolic terms use fixed ASCII tags and uint32 big-endian byte-length framing of every part.

contentHash uses the symbolic-v1:content domain over version/operation/value bytes. The empty root uses symbolic-v1:genesis over the empty canonical view. Each symbolic-v1:root term contains the previous root bytes, canonical record and complete resulting canonical view, each framed separately. These are injectively represented symbolic hash terms, deliberately not compressed cryptographic digests. They establish model equality/inequality under the ideal collision-free abstraction only. No collision resistance, native hash implementation, authenticated media or cryptographic proof is claimed.

## Committed replay and checkpoints

replay(trace) replays operational actions solely for fault fixtures. It is not the committed-record oracle. replayCommitted(records) independently reduces ordered immutable records, rejects missing/reordered/duplicate identities and verifies content identity. Different scheduling/retry/publication traces yielding the same records reconstruct the same view and symbolic root.

verifyHistory(records, trustedHead) compares reconstructed state with a separately supplied trusted (epoch,last_tid,root_hash). A claimed head supplied by the same adversary as history is not an authority. The current trust epoch may exceed the last committed record's epoch after a reset; all facts must fit beneath it.

checkpoint(prefix) emits an external derived view, last record and predecessor root plus a checkpoint head. replayCheckpoint requires a separately retained trusted checkpoint head AND final trusted head. It recomputes the checkpoint root binding, rejects altered state/anchor, then requires a contiguous ordered suffix whose record epochs do not predate the checkpoint trust epoch. The final trusted head must not lower the checkpoint trust epoch, including with an empty suffix and an older last committed record. Full replay and every checkpoint split converge. Checkpoint integrity here assumes the caller truly retains the independent anchors; it is not a production checkpoint-authentication protocol.

## Exploration, failures and validation

exploreReachableStates(start,maxDepth,limits) deterministically visits the finite action/request/epoch domain breadth-first. maxDepth is0..32; defaults are depth6,10000states,100000transition attempts. Resource limits never imply exhausted coverage: machine-readable limited, boundedDepthFullyExplored, frontierRemaining, domain, depth, limits and counts distinguish the cases. passed means only no violations among inspected states/edges, not global acceptance. The authored default test explicitly requires complete depth4 coverage within its resource limits; no state counts or successful run are claimed yet. Direct longer traces cover multiple commits, resets, identity conflict, capacity and checkpoint tampering.

Every discovered edge retains its shortest initial-to-source prefix plus failing action. Transition exceptions retain the full input trace. The BFS failure container is saved before any assertion about replayability, so a faulty counterexample reconstruction cannot erase the evidence.

The synchronous test harness also records every authored step/replay, canonical record/request, committed replay, head, checkpoint, history verification and exploration call, including calls inside traceCase inspection callbacks. On failure it writes model-input-cases-v1 with operation names, tagged arguments, observed throw classification/diagnostic, test name and assertion error. Reproduce a case by decoding args and invoking the named model export. Tagged plain objects/arrays/primitives preserve field order, explicit undefined and argument arity; this is scoped to these authored inputs, not an arbitrary JavaScript object serializer. It does not serialize functions, cyclic objects or Map results. initialState is the deterministic raw constructor; checkInvariants consumes the raw derived exploration graph. Reproduce that checker by rerunning the captured exploreReachableStates arguments, then checkInvariants on the resulting graph, not by JSON roundtripping Maps.

Thus a nested stale acknowledgement or second commit is retained as its own exact model call rather than being lost behind the leading operational trace. Checkpoint negative tests retain their canonical input objects as well. A dedicated unrun regression checks nested stale-ACK argument roundtrip, omitted versus explicit undefined arity and captured thrown record-validation input. Summary and failure files are written only by future test execution; none were generated as current evidence.

All tests in this candidate are authored and unrun. The existing node:test runner remains a future execution dependency, subject to admitted off-laptop tool/license closure. No local parser, import, runtime, tests or CI were run. Formal refinement, unbounded allocation, fairness/liveness, real durability acknowledgements, cryptographic/production encoding authority and native/hardware behavior remain separate gates.
