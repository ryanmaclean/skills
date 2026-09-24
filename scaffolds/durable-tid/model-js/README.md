# durable-tid model-js

The smallest executable model of the durable-tid lower-bound persistence
invariant (see `../docs/DESIGN.md` and the binding skills `durable-commit`,
`deterministic-replay`, `lower-bound-review`).

Zero runtime dependencies — plain ECMAScript modules, tested with Node's
built-in `node:test` runner (Node >= 18). No build step.

## Files

- `model.mjs` — the semantic states, actions, a pure total transition
  function `step(state, action, param)`, an exhaustive breadth-first
  `exploreReachableStates()`, `checkInvariants()`, and `replay()`.
- `model.test.mjs` — tests for every binding invariant, plus direct
  example traces and the full happy path.
- `fixtures/` — machine-readable output: `exploration-summary.json` is
  written on every test run (states visited, edges explored, pass/fail);
  a `*-failing-trace*.json` fixture would appear here if an invariant ever
  regressed, preserved by the test that caught it.

## States modeled

Only the five semantic states from `DESIGN.md`, plus one internal
pseudo-phase for "nothing submitted yet":

```
EMPTY (pseudo, not asserted as semantic)
  -> PROPOSED / COHERENT_VISIBLE (submit)
  -> DEVICE_COMPLETE (device_complete)
  -> PERSISTENT (durable_ack; assigns the tid)
  -> TRUSTED_COMPLETE (publish_completion)
```

`submit` lands directly in `COHERENT_VISIBLE` (acceptance and visibility
are modeled as simultaneous); `PROPOSED` is kept in the `Phase` enum only
because `../docs/RECORD-V0.md`'s `status` enum reserves the value for a
future model that separates "proposed" from "visible".

`epoch` and `tid` are bounded small integers (not unbounded counters) so
the reachable-state exploration is exhaustive and terminates — see the
comments in `model.mjs` for why this doesn't weaken any invariant.

## Invariants tested

1. trusted completion implies persistence
2. recovery never invents commit
3. duplicate request commits at most once
4. stale epoch cannot authorize current state
5. same committed ordered facts replay to same root/state (determinism)

Every invariant is checked two ways: exhaustively, over every edge
`exploreReachableStates()` discovers from the initial state, and directly,
with a hand-written example trace, so a bug in the BFS harness itself can't
hide a real regression.

## Running

Not run on this laptop (build-box policy) — CI runs it via
`node --test` in `.github/workflows/durable-tid-record-v0.yml`. Locally,
from this directory: `node --test` (or `npm test`, which just shells out to
the same command — no install step, no dependencies to fetch).
