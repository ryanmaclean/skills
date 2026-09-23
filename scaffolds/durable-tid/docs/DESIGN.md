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
