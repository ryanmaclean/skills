---
name: arena-actors
description: >
  Arena-per-actor pattern: Ericsson 1986 Erlang heaps = Zig/Rust arena allocators.
  Zero-GC actor model. Trigger: arena allocator, per-process heap, zero GC, wisp memory,
  ephemeral allocation, bumpalo, let it crash memory, Ericsson pattern
---

# Arena Actors — Zero-GC Actor Pattern

Ericsson 1986: each Erlang process gets its own heap. Process dies = free heap. No tracing GC.
Same pattern in Rust/Zig: arena-per-actor, allocate freely, free everything at once.

## The Pattern

```
spawn actor → create arena
actor runs  → allocate in arena (fast, no bookkeeping)
actor dies  → free arena (one operation, instant)
```

Zero tracing GC. Zero reference counting. Zero stop-the-world pauses.

## Rust: bumpalo (MIT)

```rust
use bumpalo::Bump;

let arena = Bump::new();
let data = arena.alloc_slice_copy(&embedding_vector);
let result = cosine_similarity(data, query);
// arena dropped here — everything freed at once
```

## Zig: ArenaAllocator

```zig
var arena = std.heap.ArenaAllocator.init(std.heap.page_allocator);
defer arena.deinit(); // free everything
const alloc = arena.allocator();
const data = try alloc.alloc(f32, 1024);
```

## Applied to Fleet Wisps

Each wisp (ephemeral bead):
1. Arena created (spawn)
2. Work done — allocations in arena only
3. Result sent to caller (message passing)
4. Arena freed (wisp dies)

1GB Pi budget: 100 concurrent wisps × ~10KB each = 1MB total. Nothing.

## vs BEAM

BEAM does the same thing — per-process heap, freed on process exit.
The difference: BEAM has a runtime VM (~40MB). Arena in Rust = 0 bytes overhead.

## Key Libraries

- bumpalo (Rust, MIT) — bump allocator, fast, 0 fragmentation
- typed-arena (Rust, MIT) — typed variant
- std.heap.ArenaAllocator (Zig) — built-in, zero deps
