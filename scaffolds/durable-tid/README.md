# durable-tid

A minimal durable ordering primitive.

The entire semantic contract is:

```
submit mutation(s)
commit
→ durable monotonic transaction id
```

Everything else is optional transport or projection.

## Why this exists

Agent systems routinely assign multiple independent identities and persist the same transition in multiple layers:

- queue sequence
- orchestration event id
- filesystem journal position
- database row/version
- trace/event id
- lineage version

This project asks whether one durable commit identity can eliminate most of that duplication.

## Core identity model

- `run_id` — who/what caused the work; external identity
- `tid` — durable ordering/version identity
- `content_hash` — content identity

The minimal provenance tuple is:

```
(run_id, tid, content_hash)
```

## Minimal operations

Version 0 should implement only:

```
append(record)
commit() -> tid
recover() -> last_tid
read(tid)
```

No filesystem namespace.
No task scheduler.
No network protocol.
No OpenLineage database.
No general AIO API.

## Transports are separate

The semantic core must not depend on:

- syscall API
- shared-memory ring
- FreeBSD buf_ring
- io_uring
- VirtIO
- NVMe
- FPGA/PCIe

Those are interchangeable ways to submit work and receive `tid`.

## First experiments

1. single-threaded append-only file
2. mmap append region
3. preallocated segment log
4. shared-memory submission/completion transport
5. FreeBSD/NetBSD kernel transport
6. VirtIO
7. FPGA/PCIe/NVMe-adjacent implementation

## Comparison set

- raw durable-tid log
- FFS
- LFS
- HAMMER1
- HAMMER2

If a filesystem provides substantially more useful semantics for little cost, use it. If not, this is the lower bound.

## License

BSD-2-Clause.
