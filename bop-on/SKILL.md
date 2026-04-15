---
name: bop-on
description: O(N) design rule for bop loops. Use when adding cross-cutting behavior (logging, hooks, events, metrics) to dispatcher, merge-gate, or any per-card loop. Trigger words — performance, loop overhead, batch, per-card cost, O(2N).
---

# bop O(N) — keep loops linear

When adding behavior to a per-card loop (dispatcher, merge-gate, reap, status), never introduce per-item syscalls. The loop body should do O(1) memory work per card; all I/O flushes once per iteration.

## Approved patterns

### 1. Collect then flush

```rust
let mut events: Vec<Event> = Vec::new();

for card in cards {
    process(card);                   // existing work
    events.push(build_event(card));  // O(1) memory, no I/O
}

flush_events(&events);  // one write, one POST
```

### 2. Opt-in gate

```rust
let enabled = lineage::is_enabled();  // check once

for card in cards {
    process(card);
    if enabled {
        events.push(...);  // zero work when disabled
    }
}
```

### 3. Defer to background

```rust
let (tx, rx) = mpsc::channel();
std::thread::spawn(move || { for ev in rx { write(ev); } });

for card in cards {
    process(card);
    let _ = tx.send(build_event(card));  // non-blocking
}
```

## Anti-patterns — reject these in review

| Anti-pattern | Why it's O(2N) | Fix |
|---|---|---|
| File append per card in loop | N extra fsync calls | Collect → flush |
| `Command::spawn` per card | N fork+exec calls | Batch or defer |
| Serialize JSON per card unconditionally | N allocs when no consumer | Opt-in gate |
| HTTP POST per card | N network round-trips | Batch POST |
| Re-read config file per card | N file reads | Cache at loop top |

## Where this applies in bop

- `run_dispatcher()` — processes up to `max_workers` cards per iteration
- `run_merge_gate()` — processes all done/ cards per iteration
- `reap_orphans()` — scans all running/ cards
- `print_state_group()` — reads all cards in a state dir for `bop status`
- `cmd_doctor()` acceptance criteria lint — iterates pending/ cards

## COW note

`fs::rename` is metadata-only on APFS/ext4/btrfs — no bytes copied, COW reflinks preserved. It's already O(1). Don't wrap it in anything that adds O(N) I/O.
