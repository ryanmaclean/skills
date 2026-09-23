# Formal verification plan

## Model first

Specify the protocol before optimizing transport.

Recommended first model: TLA+ or PlusCal.

State should distinguish at least:

```
next_tid
last_allocated_tid
last_committed_tid
durable_records
volatile_records
visible_completions
request_identity
```

## Fault points

Model crashes/resets after every state boundary:

- request accepted
- TID allocated
- payload buffered
- payload issued
- write completed
- flush issued
- durability acknowledged
- completion published

## Required invariants

```
NoDuplicateCommittedTid
MonotonicCommittedTid
CompletionImpliesDurable
RecoveryDoesNotCreateCommit
UncommittedNeverVisible
DuplicateRequestCommitsAtMostOnce
CommittedNeverExceedsAllocated
```

## Liveness

Only prove liveness under explicit fairness/progress assumptions.

## JS role

JavaScript is the executable reference/oracle:
- generate random traces
- inject faults
- compare implementations
- preserve counterexamples as fixtures

## RTL role

SystemVerilog implementation should include assertions corresponding to model invariants.

Use formal tooling before FPGA deployment.

## Hardware role

FPGA and ASIC validation confirm the physical implementation matches the formally verified protocol; they are not substitutes for the proof.


## Additional properties: immutability and deterministic replay

Add these properties to the formal model:

```
CommittedRecordNeverMutates
SameStateAndInputImpliesSameNextState
ReplayOfCommittedPrefixIsDeterministic
CheckpointPlusSuffixEqualsFullReplay
CanonicalRecordHashIsStable
```

The model must distinguish proposal nondeterminism from commit determinism.

Scheduling interleavings may differ, but if the same ordered committed facts result, the reconstructed durable state must be identical.

## Trusted-head model

Evaluate a reduced trusted state:

```
epoch
last_tid
root_hash
```

and prove that external immutable history can be replayed/verified against that head without storing the full history inside the trusted block.
