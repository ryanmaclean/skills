# Design

## Fundamental state

At minimum:

```c
struct tid_record {
    uint64_t object;
    uint64_t run_hi;
    uint64_t run_lo;
    uint32_t op;
    uint32_t length;
    /* payload or payload reference follows */
};
```

A commit establishes a total durable order:

```
TID 41
TID 42
TID 43
...
```

## Required properties

1. monotonic ordering
2. no reuse of committed TIDs
3. crash recovery to the last provably committed TID
4. idempotent retry story
5. optional content hash returned/associated without rereading payload
6. no dependence on wall-clock time for correctness

## Non-requirements

- POSIX path semantics
- multiple users
- permissions model
- directories
- sockets
- transactions spanning external systems
- distributed consensus

## Transport separation

```
caller
  |
  | mutation
  v
durable-tid semantic core
  |
  +-- function call
  +-- mmap queue
  +-- kernel ioctl
  +-- VirtIO
  +-- PCIe/FPGA
  |
  v
durable media
```

A ring is an optimization of submission/completion, not the semantic model.


## Immutability

Committed records are append-only facts.

A later transition may supersede a previous value but does not rewrite it.

A minimal committed fact should be able to represent:

```
epoch
tid
request_id
object_id
operation
parent_tid
content_hash
result
```

The trusted head may eventually collapse to:

```
(epoch, last_tid, root_hash)
```

with history stored externally.

## Determinism

The semantic transition function must be deterministic:

```
Transition(previous_state, canonical_record) -> next_state
```

Nondeterministic systems may propose records, but once a record crosses the commit boundary the lower layer may not depend on clocks, scheduler timing, model randomness, or implicit process state.

Replay of immutable committed records must reconstruct the same canonical state.

## Canonical record encoding

Before hashing/signing/replay, define a canonical wire representation with fixed:
- integer widths
- endianness
- field order
- encoding/version
- normalization rules

Do not use compiler-native structure layout as a persistent format.
