---
name: simd-vectors
description: >
  SIMD vector operations for embedding similarity on bare metal. ARM64 NEON, x86 AVX.
  Trigger: cosine similarity, SIMD, NEON, AVX, embedding local, vector math, dot product, bare metal inference
---

# SIMD Vectors — Embedding Similarity on Bare Metal

Every Pi has ARM64 NEON (128-bit = 4x f32). i9/4090 have AVX2 (256-bit = 8x f32).
Compute cosine similarity locally without Ollama.

## The Math

cosine_sim(a, b) = dot(a,b) / (norm(a) * norm(b))

1024-dim embeddings:
- Without SIMD: 1024 multiplies + 1024 adds = ~2048 ops
- NEON (Pi): 256 SIMD ops (4x speedup)
- AVX2 (i9): 128 SIMD ops (8x speedup)

## Zig Implementation (stable, cross-platform)

```zig
const Vec = @Vector(4, f32);

fn dot_product_simd(a: []const f32, b: []const f32) f32 {
    var sum: Vec = @splat(0.0);
    var i: usize = 0;
    while (i + 4 <= a.len) : (i += 4) {
        const va: Vec = a[i..][0..4].*;
        const vb: Vec = b[i..][0..4].*;
        sum += va * vb;
    }
    return @reduce(.Add, sum);
}
```

Export via C ABI, call from Rust.

## Rust Implementation (nightly only)

```rust
#![feature(portable_simd)]
use std::simd::f32x4;
// Only on nightly — not stable as of 1.94
```

## Practical Use

Qdrant stores vectors. For LOCAL recall (no network):
1. Cache recent vectors in memory
2. New query → SIMD cosine similarity against cache
3. If match > threshold → use cached result
4. Else → full Qdrant search over network

Arena allocator per query: allocate → compute → free. Zero GC.

## Ericsson Connection

Erlang per-process heap (1986) = arena allocator.
SIMD per-wisp vector math (2024) = same pattern + hardware acceleration.
