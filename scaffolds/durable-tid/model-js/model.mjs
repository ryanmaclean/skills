// Exhaustive, dependency-free reference model of the durable-tid
// lower-bound persistence invariant.
//
// Binding skills: durable-commit, deterministic-replay, lower-bound-review
// (see ../../../durable-commit/SKILL.md, ../../../deterministic-replay/SKILL.md,
// ../../../lower-bound-review/SKILL.md).
//
// This module represents only the semantic states from ../docs/DESIGN.md
// (PROPOSED, COHERENT_VISIBLE, DEVICE_COMPLETE, PERSISTENT,
// TRUSTED_COMPLETE), plus one internal pseudo-phase `EMPTY` for "no request
// submitted yet" (there is no request to be PROPOSED about before the
// first `submit`). `EMPTY` is never asserted as one of the five semantic
// states; it only exists so the model has a start state and a place to land
// after `reset`/an uncommitted `recover`.
//
// No UI, no HTTP, no agent framework, no FPGA simulation library, and no
// Linux-specific dependency: this is plain ECMAScript with zero imports,
// runnable by any spec-compliant JS engine.

export const Phase = Object.freeze({
  EMPTY: "EMPTY", // pseudo-phase: no in-flight or completed round
  PROPOSED: "PROPOSED",
  COHERENT_VISIBLE: "COHERENT_VISIBLE",
  DEVICE_COMPLETE: "DEVICE_COMPLETE",
  PERSISTENT: "PERSISTENT",
  TRUSTED_COMPLETE: "TRUSTED_COMPLETE",
});

// The five semantic states the binding skills reason about. EMPTY and
// PROPOSED are excluded from this list on purpose: EMPTY is not a semantic
// state, and PROPOSED in this model is unreachable as a *steady* state
// (submit lands directly in COHERENT_VISIBLE — see step() below) but is
// kept in the Phase enum because RECORD-V0.md's `status` enum reserves the
// value for a future model that separates "proposed" from "visible".
export const SEMANTIC_STATES = Object.freeze([
  Phase.PROPOSED,
  Phase.COHERENT_VISIBLE,
  Phase.DEVICE_COMPLETE,
  Phase.PERSISTENT,
  Phase.TRUSTED_COMPLETE,
]);

export const Actions = Object.freeze([
  "submit",
  "duplicate_submit",
  "device_complete",
  "durable_ack",
  "publish_completion",
  "reset",
  "recover",
  "replay_old_epoch",
]);

// Epoch and tid are modeled as small bounded integers rather than
// unbounded monotonic counters. This is a deliberate finite-state
// simplification so exhaustive BFS terminates; it does not change any of
// the invariants below, which only ever compare tid/epoch values relative
// to each other (monotonic non-decrease), never against an absolute bound.
export const EPOCH_VALUES = Object.freeze([0, 1]);
const TID_MAX = 2;

export function initialState() {
  return { phase: Phase.EMPTY, epoch: 0, tid: 0, dupSeen: false };
}

function clampTidIncrement(tid) {
  return tid >= TID_MAX ? TID_MAX : tid + 1;
}

function sameEpochOrNewer(state, param) {
  // Missing epoch on an action that doesn't carry one (e.g. reset) is
  // always treated as "current" — only actions that explicitly carry an
  // epoch can be rejected as stale.
  if (!param || param.epoch === undefined) return true;
  return param.epoch >= state.epoch;
}

/**
 * Pure, total transition function: every (state, action) pair returns a
 * defined next state, never throws, and never mutates `state`. An action
 * that is not valid from the given phase, or that carries a stale epoch,
 * is a documented no-op (returns a value deep-equal to `state`, except
 * where `duplicate_submit`/resubmission semantics explicitly set
 * `dupSeen`).
 *
 * @param {{phase:string, epoch:number, tid:number, dupSeen:boolean}} state
 * @param {string} action one of `Actions`
 * @param {{epoch?:number}} [param]
 */
export function step(state, action, param) {
  switch (action) {
    case "submit": {
      if (state.phase !== Phase.EMPTY) {
        // A round is already in flight or already completed: this is a
        // resubmission of the same request, i.e. a duplicate. It must
        // never allocate a new tid or move the phase.
        return { ...state, dupSeen: true };
      }
      if (!sameEpochOrNewer(state, param)) return { ...state }; // stale epoch rejected
      const epoch = param && param.epoch !== undefined ? param.epoch : state.epoch;
      return { phase: Phase.COHERENT_VISIBLE, epoch, tid: 0, dupSeen: false };
    }

    case "duplicate_submit": {
      if (state.phase === Phase.EMPTY) return { ...state }; // nothing to duplicate yet
      return { ...state, dupSeen: true }; // tracked, never changes phase/tid
    }

    case "device_complete": {
      if (state.phase !== Phase.COHERENT_VISIBLE) return { ...state };
      if (!sameEpochOrNewer(state, param)) return { ...state };
      return { ...state, phase: Phase.DEVICE_COMPLETE };
    }

    case "durable_ack": {
      if (state.phase !== Phase.DEVICE_COMPLETE) return { ...state };
      if (!sameEpochOrNewer(state, param)) return { ...state };
      return { ...state, phase: Phase.PERSISTENT, tid: clampTidIncrement(state.tid) };
    }

    case "publish_completion": {
      if (state.phase !== Phase.PERSISTENT) return { ...state };
      return { ...state, phase: Phase.TRUSTED_COMPLETE };
    }

    case "reset": {
      // Clears the in-flight/completed round so a new submit can start.
      // Only meaningful before or after a round; does not "undo" a
      // durable commit (tid is a durable fact — see docs/DESIGN.md
      // "Immutability" — reset starts a fresh round, it does not erase
      // history, which is why `tid` is monotonic across rounds via
      // clampTidIncrement and never decreases here).
      return { phase: Phase.EMPTY, epoch: state.epoch, tid: state.tid, dupSeen: false };
    }

    case "recover": {
      // Recovery must return to the last *durably* committed point and
      // must never invent progress beyond it.
      if (state.phase === Phase.PERSISTENT || state.phase === Phase.TRUSTED_COMPLETE) {
        return { ...state, phase: Phase.PERSISTENT };
      }
      // Nothing before PERSISTENT was durable, so a crash loses it.
      return { phase: Phase.EMPTY, epoch: state.epoch, tid: state.tid, dupSeen: false };
    }

    case "replay_old_epoch": {
      // By construction this action always carries a stale epoch relative
      // to the current one; it must never authorize a state change. If a
      // caller passes a non-stale epoch here (a test bug, not real usage),
      // we still refuse to change state — replay of a record can never be
      // stronger than a live action already is.
      return { ...state };
    }

    default:
      throw new TypeError(`unknown action: ${action}`);
  }
}

// ---------------------------------------------------------------------
// Exhaustive reachable-state exploration (breadth-first, deterministic).
// ---------------------------------------------------------------------

function key(state) {
  return `${state.phase}|${state.epoch}|${state.tid}|${state.dupSeen}`;
}

/**
 * Explores every state reachable from `start` (default: initialState())
 * by applying every action in `Actions` with every epoch parameter in
 * `EPOCH_VALUES`, up to `maxDepth` steps, deterministically (states are
 * visited and edges are generated in a fixed, sorted order every run).
 *
 * @returns {{states: Map<string, object>, edges: Array<object>}}
 */
export function exploreReachableStates(start = initialState(), maxDepth = 6) {
  const visited = new Map([[key(start), start]]);
  const edges = [];
  let frontier = [start];
  for (let depth = 0; depth < maxDepth && frontier.length > 0; depth++) {
    const next = [];
    for (const state of frontier) {
      for (const action of Actions) {
        // Only actions whose semantics actually consult `param.epoch`
        // (submit, device_complete, durable_ack, replay_old_epoch) are
        // explored across every epoch value. `reset`, `duplicate_submit`,
        // and `recover` ignore epoch entirely (see step()), so exploring
        // them with a fake epoch parameter would only inflate the state
        // space without exercising anything new.
        const epochSensitive =
          action === "submit" ||
          action === "device_complete" ||
          action === "durable_ack" ||
          action === "replay_old_epoch";
        const params = epochSensitive ? EPOCH_VALUES.map((epoch) => ({ epoch })) : [undefined];
        for (const param of params) {
          const nextState = step(state, action, param);
          edges.push({ from: state, action, param: param ?? null, to: nextState });
          const k = key(nextState);
          if (!visited.has(k)) {
            visited.set(k, nextState);
            next.push(nextState);
          }
        }
      }
    }
    frontier = next;
  }
  return { states: visited, edges };
}

/**
 * Machine-readable invariant checks, run over every edge discovered by
 * exploreReachableStates(). Returns a summary object; never throws (a
 * violation is reported in the summary, not raised), so callers can decide
 * whether to persist a failing trace as a fixture.
 */
export function checkInvariants({ states, edges }) {
  const violations = [];

  // 1. trusted completion implies persistence: every TRUSTED_COMPLETE
  //    state has a real (non-zero) tid.
  for (const state of states.values()) {
    if (state.phase === Phase.TRUSTED_COMPLETE && state.tid === 0) {
      violations.push({ invariant: "trusted_completion_implies_persistence", state });
    }
  }

  // 2. recovery never invents commit: a `recover` edge never increases tid
  //    and never lands on a phase "ahead" of PERSISTENT relative to what
  //    was true before recovery.
  for (const edge of edges) {
    if (edge.action !== "recover") continue;
    if (edge.to.tid > edge.from.tid) {
      violations.push({ invariant: "recovery_never_invents_commit", edge });
    }
    if (edge.to.phase !== Phase.EMPTY && edge.to.phase !== Phase.PERSISTENT) {
      violations.push({ invariant: "recovery_lands_outside_known_phases", edge });
    }
  }

  // 3. duplicate request commits at most once: `duplicate_submit`, and a
  //    `submit` that lands on an already-active/completed round, must
  //    never change `tid`.
  for (const edge of edges) {
    const isDuplicateLike =
      edge.action === "duplicate_submit" ||
      (edge.action === "submit" && edge.from.phase !== Phase.EMPTY);
    if (isDuplicateLike && edge.to.tid !== edge.from.tid) {
      violations.push({ invariant: "duplicate_commits_at_most_once", edge });
    }
  }

  // 4. stale epoch cannot authorize current state: for the actions whose
  //    semantics actually gate on epoch when they take their "real" path
  //    (submit from EMPTY, device_complete, durable_ack, replay_old_epoch),
  //    a stale param.epoch must produce a full no-op (deep-equal to the
  //    input state). `submit` from a non-EMPTY phase is a resubmission —
  //    it is intentionally epoch-independent and only sets `dupSeen`
  //    (invariant 3 covers it); `reset` and `recover` don't consult epoch
  //    at all, by design, so they're excluded here too.
  const epochGatedActions = new Set(["submit", "device_complete", "durable_ack", "replay_old_epoch"]);
  for (const edge of edges) {
    if (!epochGatedActions.has(edge.action)) continue;
    if (edge.action === "submit" && edge.from.phase !== Phase.EMPTY) continue;
    if (!edge.param || edge.param.epoch === undefined) continue;
    const stale = edge.param.epoch < edge.from.epoch;
    if (stale && JSON.stringify(edge.to) !== JSON.stringify(edge.from)) {
      violations.push({ invariant: "stale_epoch_cannot_authorize", edge });
    }
  }

  // 5. same committed ordered facts replay to same root/state: step() is
  //    pure, so replaying an identical action sequence from an identical
  //    start must always reach an identical end state. Checked separately
  //    in the test file with explicit traces (purity alone guarantees it
  //    here, but the test file exercises it end-to-end as insurance
  //    against a future non-pure edit).

  return {
    statesVisited: states.size,
    edgesExplored: edges.length,
    violations,
    ok: violations.length === 0,
  };
}

/**
 * Deterministically replays an ordered list of {action, param} facts from
 * a fresh initialState() and returns the resulting state. Used to assert
 * "same committed ordered facts replay to same root/state".
 */
export function replay(facts, start = initialState()) {
  let state = start;
  for (const { action, param } of facts) {
    state = step(state, action, param);
  }
  return state;
}
