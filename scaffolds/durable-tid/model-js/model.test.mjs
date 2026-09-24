import test from "node:test";
import assert from "node:assert/strict";
import { writeFileSync, mkdirSync, existsSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";
import {
  Phase,
  Actions,
  initialState,
  step,
  exploreReachableStates,
  checkInvariants,
  replay,
} from "./model.mjs";

const here = dirname(fileURLToPath(import.meta.url));
const fixturesDir = join(here, "fixtures");

function saveFailingTrace(name, payload) {
  if (!existsSync(fixturesDir)) mkdirSync(fixturesDir, { recursive: true });
  writeFileSync(join(fixturesDir, `${name}.json`), JSON.stringify(payload, null, 2) + "\n");
}

test("step() is total: every action from every reachable state returns a defined, non-throwing result", () => {
  const { states } = exploreReachableStates();
  for (const state of states.values()) {
    for (const action of Actions) {
      assert.doesNotThrow(() => step(state, action, { epoch: state.epoch }));
    }
  }
});

test("exhaustive exploration terminates and produces a machine-readable summary", () => {
  const explored = exploreReachableStates(initialState(), 6);
  const summary = checkInvariants(explored);
  // Machine-readable summary shape a CI step or another agent can parse.
  assert.equal(typeof summary.statesVisited, "number");
  assert.equal(typeof summary.edgesExplored, "number");
  assert.ok(summary.statesVisited > 0);
  assert.ok(summary.edgesExplored > 0);
  assert.ok(Array.isArray(summary.violations));
});

test("invariant: trusted completion implies persistence", () => {
  const explored = exploreReachableStates();
  const summary = checkInvariants(explored);
  const failures = summary.violations.filter(
    (v) => v.invariant === "trusted_completion_implies_persistence"
  );
  if (failures.length > 0) saveFailingTrace("trusted-completion-implies-persistence", failures);
  assert.deepEqual(failures, []);
});

test("invariant: recovery never invents commit", () => {
  const explored = exploreReachableStates();
  const summary = checkInvariants(explored);
  const failures = summary.violations.filter(
    (v) => v.invariant === "recovery_never_invents_commit" ||
      v.invariant === "recovery_lands_outside_known_phases"
  );
  if (failures.length > 0) saveFailingTrace("recovery-never-invents-commit", failures);
  assert.deepEqual(failures, []);
});

test("invariant: duplicate request commits at most once", () => {
  const explored = exploreReachableStates();
  const summary = checkInvariants(explored);
  const failures = summary.violations.filter(
    (v) => v.invariant === "duplicate_commits_at_most_once"
  );
  if (failures.length > 0) saveFailingTrace("duplicate-commits-at-most-once", failures);
  assert.deepEqual(failures, []);

  // Direct example, independent of BFS: submitting, then duplicating the
  // same request many times, then completing the round, must yield tid=1
  // exactly once — never 0, never >1.
  let state = initialState();
  state = step(state, "submit", { epoch: 0 });
  for (let i = 0; i < 5; i++) state = step(state, "duplicate_submit");
  state = step(state, "device_complete", { epoch: 0 });
  state = step(state, "durable_ack", { epoch: 0 });
  assert.equal(state.tid, 1);
  assert.equal(state.dupSeen, true);
  const beforeExtraDup = state.tid;
  state = step(state, "duplicate_submit");
  assert.equal(state.tid, beforeExtraDup, "duplicate after commit must not bump tid");
});

test("invariant: stale epoch cannot authorize current state", () => {
  const explored = exploreReachableStates();
  const summary = checkInvariants(explored);
  const failures = summary.violations.filter(
    (v) => v.invariant === "stale_epoch_cannot_authorize"
  );
  if (failures.length > 0) saveFailingTrace("stale-epoch-cannot-authorize", failures);
  assert.deepEqual(failures, []);

  // Direct example: adopt epoch 1, then try to durable_ack with epoch 0.
  let state = initialState();
  state = step(state, "submit", { epoch: 1 });
  state = step(state, "device_complete", { epoch: 1 });
  const beforeAck = state;
  const rejected = step(state, "durable_ack", { epoch: 0 });
  assert.deepEqual(rejected, beforeAck, "stale-epoch durable_ack must be a no-op");

  const replayed = step(state, "replay_old_epoch", { epoch: 0 });
  assert.deepEqual(replayed, beforeAck, "replay_old_epoch must never change state");
});

test("invariant: same committed ordered facts replay to same root/state (determinism)", () => {
  const facts = [
    { action: "submit", param: { epoch: 0 } },
    { action: "duplicate_submit" },
    { action: "device_complete", param: { epoch: 0 } },
    { action: "durable_ack", param: { epoch: 0 } },
    { action: "publish_completion" },
  ];
  const first = replay(facts);
  const second = replay(facts);
  assert.deepEqual(first, second);
  assert.equal(first.phase, Phase.TRUSTED_COMPLETE);
  assert.equal(first.tid, 1);

  // Replaying from a different (but equal) fresh start must also agree.
  const third = replay(facts, initialState());
  assert.deepEqual(first, third);
});

test("full happy path reaches TRUSTED_COMPLETE with tid=1 and no dangling dup flag surprises", () => {
  let state = initialState();
  assert.equal(state.phase, Phase.EMPTY);
  state = step(state, "submit", { epoch: 0 });
  assert.equal(state.phase, Phase.COHERENT_VISIBLE);
  state = step(state, "device_complete", { epoch: 0 });
  assert.equal(state.phase, Phase.DEVICE_COMPLETE);
  state = step(state, "durable_ack", { epoch: 0 });
  assert.equal(state.phase, Phase.PERSISTENT);
  assert.equal(state.tid, 1);
  state = step(state, "publish_completion");
  assert.equal(state.phase, Phase.TRUSTED_COMPLETE);
  assert.equal(state.dupSeen, false);
});

test("recover before durable_ack loses the in-flight round (crash before persistence)", () => {
  let state = initialState();
  state = step(state, "submit", { epoch: 0 });
  state = step(state, "device_complete", { epoch: 0 });
  const beforeCrash = state;
  const recovered = step(state, "recover");
  assert.equal(recovered.phase, Phase.EMPTY);
  assert.equal(recovered.tid, beforeCrash.tid);
  assert.equal(recovered.tid, 0);
});

test("recover after durable_ack preserves the persisted tid and phase", () => {
  let state = initialState();
  state = step(state, "submit", { epoch: 0 });
  state = step(state, "device_complete", { epoch: 0 });
  state = step(state, "durable_ack", { epoch: 0 });
  state = step(state, "publish_completion");
  const recovered = step(state, "recover");
  assert.equal(recovered.phase, Phase.PERSISTENT);
  assert.equal(recovered.tid, 1);
});

test("writes a machine-readable exploration summary fixture", () => {
  const explored = exploreReachableStates();
  const summary = checkInvariants(explored);
  if (!existsSync(fixturesDir)) mkdirSync(fixturesDir, { recursive: true });
  writeFileSync(
    join(fixturesDir, "exploration-summary.json"),
    JSON.stringify(
      {
        statesVisited: summary.statesVisited,
        edgesExplored: summary.edgesExplored,
        ok: summary.ok,
        violationCount: summary.violations.length,
      },
      null,
      2
    ) + "\n"
  );
  assert.ok(summary.ok, "no invariant violations across the full exploration");
});
