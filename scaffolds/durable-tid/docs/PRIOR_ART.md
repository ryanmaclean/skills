# Prior art

The point of durable-tid is not to invent queues, logs, or transaction IDs independently. It combines lessons from lower-level systems while keeping only the smallest semantics needed.

## HAMMER1

HAMMER uses monotonically increasing 64-bit transaction IDs to identify historical filesystem data. This is the strongest semantic prior art for using one durable TID as historical/version identity.

Useful lesson:
- transaction order can itself be a first-class persistent identifier.

## NVMe / SPDK

NVMe exposes queue pairs made of submission and completion queues in host memory. Commands are fixed-size descriptors and devices DMA data directly. SPDK commonly polls completions rather than interrupting.

Useful lesson:
- submission/completion is transport;
- the hot path can be descriptor + DMA + completion;
- queue ownership can remove locks.

Do not copy NVMe's storage command set into the semantic core.

## DragonFly LWKT message ports

DragonFly uses lightweight message passing and CPU-localization in kernel subsystems.

Useful lesson:
- ownership/message passing can replace shared locking;
- endpoint mechanics can vary without changing higher-level semantics.

## FreeBSD buf_ring / netmap

FreeBSD already has low-level ring primitives and mmap-based high-throughput packet rings.

Useful lesson:
- do not invent a lockless ring algorithm before measuring existing BSD primitives.

## io_uring

io_uring demonstrates registered state, shared submission/completion queues, batching, and reduced syscall overhead.

Useful lesson:
- setup once; submit tiny descriptors;
- but durable-tid is not a general-purpose io_uring clone.

Linux remains external prior art/benchmark host only because target project licensing and OS goals are BSD-first/permissive.

## Raw write-ahead/commit logs

Databases and consensus systems demonstrate append + durable commit markers, but most bring indexing, replication, transactions, SQL, or distributed semantics that are explicitly outside the initial scope.

## Research question

Can:

```
append + durable commit -> tid
```

provide enough truth that BOP, filesystem namespace views, Datadog, and OpenLineage can all be projections?
