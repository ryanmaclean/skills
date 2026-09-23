---
name: deterministic-replay
description: >
  Design and test immutable committed state, canonical record encoding,
  deterministic transitions, crash replay, checkpoints, hash chains, and JS/formal
  executable models. Use when implementing replay, recovery, event logs, commit
  records, canonical serialization, hash roots, state-machine tests, JS simulators,
  TLA+/PlusCal, RTL equivalence, or reproducibility.
---

# deterministic-replay — immutable facts, reproducible state

The rule:

> Nondeterminism may propose state; only deterministic mechanisms commit state.

## Required transition model

Define a pure semantic function:

```
Transition(previous_state, canonical_record) -> next_state
```

Given identical prior committed state and identical canonical input, the result must not depend on:

- wall-clock time
- scheduler timing
- CPU/core placement
- agent/model randomness
- retry timing
- unordered map iteration
- process-local hidden state

## Immutability

Committed records never mutate.

A correction is another committed record.

Mutable current-state views must be reconstructable from immutable committed facts.

## Deterministic recovery

Target:

```
checkpoint
+ immutable committed suffix
-> current canonical state
```

Full replay and checkpoint+suffix replay must converge.

## Canonical encoding

Before hashing, signing, comparing, or storing a persistent record, specify:

- schema/version
- fixed field order
- exact integer widths
- endianness
- string/Unicode normalization
- optional/default-field representation
- byte/string encoding
- hash domain separation

Never persist/hash compiler-native structure layout.

Do not rely on ordinary unordered JSON serialization.

## JS executable model

For tiny persistence/state-machine work, prefer exhaustive exploration before random fuzzing.

Model semantic states, not transistor/cache internals.

Useful abstract states:

```
PROPOSED
COHERENT_VISIBLE
DEVICE_COMPLETE
PERSISTENT
TRUSTED_COMPLETE
```

Enumerate actions such as:

```
submit
duplicate_submit
durable_ack
publish_completion
reset
recover
replay_old_epoch
```

Explore all reachable action sequences to a useful depth and retain counterexamples as regression fixtures.

## Assertions

At minimum:

```
CommittedRecordNeverMutates
SameStateAndInputImpliesSameNextState
ReplayOfCommittedPrefixIsDeterministic
CheckpointPlusSuffixEqualsFullReplay
CompletionImpliesPersistent
RecoveryDoesNotCreateCommit
DuplicateRequestCommitsAtMostOnce
CanonicalRecordHashIsStable
```

## Scheduling

Do NOT require deterministic scheduling by default.

Require deterministic outcome semantics:

```
same ordered committed facts
-> same reconstructed state
```

## Formal-to-hardware ladder

```
semantic model
-> TLA+/PlusCal model check
-> JS executable oracle
-> RTL
-> SVA/SymbiYosys
-> FPGA differential traces
-> ASIC only after evidence
```

JS and RTL should consume the same trace fixtures whenever possible.

## Trusted-head experiment

Prefer testing whether hardware can retain only:

```
(epoch, last_tid, root_hash)
```

while immutable history remains external.

## Canonical references

- `docs/CONSTRAINTS.md`
- `scaffolds/durable-tid/docs/FORMAL.md`
- `scaffolds/durable-tid/docs/DESIGN.md`
- `rfcs/0005-formal-verification-before-rtl.md`
