// SPDX-License-Identifier: BSD-2-Clause
import nodeTest from "node:test";
import assert from "node:assert/strict";
import { writeFileSync, mkdirSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";
import * as model from "./model.mjs";
const { Phase, Actions, DOMAIN, TID_MAX, initialState, checkInvariants } = model;
const fixturesDir = join(dirname(fileURLToPath(import.meta.url)), "fixtures");
const clone = x => JSON.parse(JSON.stringify(x));
const A = Object.freeze({ request_id: 1, object_id: 1, operation: 1, value: 0 });
const B = Object.freeze({ request_id: 2, object_id: 1, operation: 1, value: 1 });
const C = Object.freeze({ request_id: 3, object_id: 2, operation: 1, value: 0 });
const item = (action, epoch, request) => ({ action, param: { ...(epoch === undefined ? {} : { epoch }), ...(request ? { request } : {}) } });
const commit = (request, epoch) => [item("submit", epoch, request), item("make_visible", epoch), item("device_complete", epoch), item("durable_ack", epoch)];
function saveFailingTrace(name, fixture) {
  mkdirSync(fixturesDir, { recursive: true });
  writeFileSync(join(fixturesDir, name + ".json"), JSON.stringify(fixture, null, 2) + "\n");
}
// All tests here are synchronous and run without concurrent subtests. Keep
// exact public-model arguments for every call, including operations inside an
// inspect callback and canonical-history/checkpoint negative tests.
let inputCases = null;
// Tagged recursive values preserve explicit undefined and property insertion
// order; omitted arguments remain absent because the outer array keeps arity.
// This suite passes only plain objects/arrays and primitive JSON-domain values.
function encodeInput(value) {
  if (value === undefined) return { type: "undefined" };
  if (value === null) return { type: "null" };
  if (Array.isArray(value)) return { type: "array", items: value.map(encodeInput) };
  if (typeof value === "object") return { type: "object", entries: Object.entries(value).map(([k,v]) => [k, encodeInput(v)]) };
  if (typeof value === "number") return { type: "number", value: Object.is(value, -0) ? "-0" : String(value) };
  return { type: typeof value, value };
}
function decodeInput(value) {
  if (value.type === "undefined") return undefined;
  if (value.type === "null") return null;
  if (value.type === "array") return value.items.map(decodeInput);
  if (value.type === "object") return Object.fromEntries(value.entries.map(([k,v]) => [k, decodeInput(v)]));
  if (value.type === "number") return Number(value.value);
  return value.value;
}
function observe(operation) {
  return (...args) => {
    const entry = { operation, args: encodeInput(args) };
    if (inputCases) inputCases.push(entry);
    try { const result = model[operation](...args); entry.threw = false; return result; }
    catch (error) { entry.threw = true; entry.error = error.message; throw error; }
  };
}
const step = observe("step"), replay = observe("replay");
const exploreReachableStates = observe("exploreReachableStates");
const canonicalRecord = observe("canonicalRecord"), canonicalRequest = observe("canonicalRequest");
const replayCommitted = observe("replayCommitted"), trustedHead = observe("trustedHead");
const verifyHistory = observe("verifyHistory"), checkpoint = observe("checkpoint"), replayCheckpoint = observe("replayCheckpoint");
function test(name, body) {
  nodeTest(name, () => {
    inputCases = [];
    try { body(); }
    catch (error) {
      saveFailingTrace(name.replaceAll(/[^a-zA-Z0-9_-]/g, "-") + "-failing-inputs", {
        kind: "model-input-cases-v1", test: name, cases: inputCases, error: error.message
      });
      throw error;
    } finally { inputCases = null; }
  });
}
function traceCase(name, trace, inspect, start = initialState()) {
  test(name, () => inspect(replay(trace, start)));
}

test("bounded exploration checks all discovered states and edges with explicit coverage", () => {
  const explored = exploreReachableStates(initialState(), 4, { maxStates: 10000, maxEdges: 100000 });
  const summary = checkInvariants(explored);
  mkdirSync(fixturesDir, { recursive: true });
  writeFileSync(join(fixturesDir, "exploration-summary.json"), JSON.stringify(summary, null, 2) + "\n");
  if (summary.violations.length) saveFailingTrace("exploration-failing-trace", { kind: "bounded-exploration-v1", start: explored.start, maxDepth: explored.maxDepth, domain: DOMAIN, failures: summary.violations });
  for (const violation of summary.violations) {
    assert.ok(Array.isArray(violation.trace));
    if (violation.expectedThrow) assert.throws(() => replay(violation.trace, explored.start));
    else assert.deepEqual(replay(violation.trace, explored.start), violation.state);
  }
  assert.equal(summary.limited, false, "resource exhaustion is not exhaustive coverage");
  assert.equal(summary.boundedDepthFullyExplored, true);
  assert.equal(summary.maxDepth, 4);
  assert.equal(summary.passed, true);
  for (const state of explored.states.values()) for (const action of Actions) assert.doesNotThrow(() => step(state, action, { epoch: state.epoch }));
  for (const edge of explored.edges) assert.deepEqual(replay(edge.trace, explored.start), edge.to);
});

test("resource limits and zero depth remain explicit rather than reported as global proof", () => {
  const explored = exploreReachableStates(initialState(), 12, { maxStates: 2, maxEdges: 2 });
  const summary = checkInvariants(explored);
  assert.equal(summary.limited, true);
  assert.equal(summary.boundedDepthFullyExplored, false);
  assert.equal(summary.frontierRemaining, true);
  assert.equal(exploreReachableStates(initialState(), 0).edges.length, 0);
  assert.throws(() => exploreReachableStates(initialState(), -1));
});

traceCase("happy path exposes every distinct semantic boundary", commit(A, 0).concat(item("publish_completion", 0)), s => {
  const proposed = step(initialState(), "submit", { request: A });
  assert.equal(proposed.phase, Phase.PROPOSED); assert.equal(proposed.last_allocated_tid, 0);
  const visible = step(proposed, "make_visible");
  assert.equal(visible.phase, Phase.COHERENT_VISIBLE); assert.equal(visible.last_allocated_tid, 1); assert.equal(visible.tid, 0);
  const device = step(visible, "device_complete");
  assert.equal(device.phase, Phase.DEVICE_COMPLETE); assert.equal(device.history.length, 0);
  assert.deepEqual(step(device, "publish_completion"), device);
  assert.equal(step(device, "durable_ack").phase, Phase.PERSISTENT);
  assert.equal(s.phase, Phase.TRUSTED_COMPLETE); assert.equal(s.tid, 1);
  assert.equal(s.completion.request_id, A.request_id); assert.equal(s.completion.tid, 1);
});

traceCase("duplicate identity after reset and recovery returns original committed fact", [
  ...commit(A, 0), item("reset"), item("submit", 1, A), item("duplicate_submit", 1, A), item("publish_completion", 1), item("recover"), item("submit", 2, A), item("publish_completion", 2)
], s => {
  assert.equal(s.tid, 1); assert.equal(s.history.length, 1);
  assert.equal(s.completion.tid, 1); assert.equal(s.completion.epoch, 2);
  assert.equal(s.history[0].epoch, 0); assert.equal(s.dupSeen, true);
});

traceCase("conflicting payload cannot reuse a durable request identity", [...commit(A, 0), item("reset"), item("submit", 1, { ...A, value: 1 }), item("durable_ack", 1), item("publish_completion", 1)], s => {
  assert.equal(s.outcome, "identity_conflict"); assert.equal(s.phase, Phase.EMPTY);
  assert.equal(s.tid, 1); assert.equal(s.history[0].value, 0); assert.equal(s.completion, null);
});

traceCase("crash before acknowledgement never creates a committed fact", [item("submit", 0, A), item("make_visible", 0), item("device_complete", 0), item("recover")], s => {
  assert.equal(s.phase, Phase.EMPTY); assert.equal(s.tid, 0); assert.equal(s.last_allocated_tid, 0); assert.deepEqual(s.history, []); assert.equal(s.epoch, 1);
});

traceCase("prior committed TID survives crash during the next uncommitted round", [...commit(A, 0), item("reset"), item("submit", 1, B), item("make_visible", 1), item("recover")], s => {
  assert.equal(s.phase, Phase.EMPTY); assert.equal(s.tid, 1); assert.equal(s.last_allocated_tid, 1);
  assert.equal(s.history.length, 1); assert.equal(s.history[0].request_id, A.request_id);
  const second = replay(commit(B, 2), s); assert.equal(second.tid, 2);
});

traceCase("finite capacity never acknowledges or publishes a third commit", [...commit(A, 0), item("reset"), ...commit(B, 1), item("reset"), item("submit", 2, C), item("make_visible", 2), item("device_complete", 2), item("durable_ack", 2), item("publish_completion", 2)], s => {
  assert.equal(s.tid, TID_MAX); assert.equal(s.history.length, TID_MAX);
  assert.equal(s.phase, Phase.PROPOSED); assert.equal(s.outcome, "tid_capacity"); assert.equal(s.completion, null);
  assert.equal(step(s, "durable_ack").tid, TID_MAX);
  assert.equal(step(s, "recover").outcome, "epoch_capacity");
  assert.deepEqual(step(s, "recover").history, s.history);
});

traceCase("stale and future epochs cannot authorize a later request", [...commit(A, 0), item("reset"), item("submit", 1, B), item("make_visible", 1), item("device_complete", 1)], s => {
  for (const epoch of [0, 2]) for (const action of ["submit", "duplicate_submit", "make_visible", "device_complete", "durable_ack", "publish_completion"]) assert.deepEqual(step(s, action, { epoch, request: B }), s);
  assert.deepEqual(step(s, "replay_old_epoch", { epoch: 0 }), s);
  assert.equal(step(s, "durable_ack", { epoch: 1 }).tid, 2);
});

test("different retry and publication schedules yield identical ordered facts and reduced state", () => {
  const one = replay(commit(A, 0));
  const two = replay([item("submit", 0, A), item("duplicate_submit", 0, A), item("make_visible", 0), item("duplicate_submit", 0, A), item("device_complete", 0), item("durable_ack", 0), item("publish_completion", 0)]);
  assert.notDeepEqual(one, two);
  assert.deepEqual(one.history, two.history);
  assert.deepEqual(replayCommitted(one.history), replayCommitted(two.history));
  assert.deepEqual(replayCommitted(clone(one.history)), replayCommitted(one.history));
});

test("canonical facts are frozen and insertion order cannot change canonical bytes", () => {
  const s = replay(commit(A, 0)); const r = s.history[0];
  assert.ok(Object.isFrozen(s.history) && Object.isFrozen(r));
  assert.throws(() => { r.value = 1; }, TypeError);
  assert.equal(canonicalRecord(r), canonicalRecord(Object.fromEntries(Object.entries(r).reverse())));
  assert.equal(canonicalRequest(A), "0101010100");
  assert.throws(() => canonicalRecord({ ...r, extra: 0 }));
  assert.throws(() => canonicalRecord({ ...r, content_hash: "other" }));
  const next = replay([item("reset"), ...commit(B, 1)], s);
  assert.deepEqual(next.history[0], r);
});

test("every checkpoint split converges and tampered suffix or checkpoint is rejected", () => {
  const s = replay([...commit(A, 0), item("reset"), ...commit(B, 1)]);
  const full = replayCommitted(s.history); const head = trustedHead(full, s.epoch);
  for (let i = 0; i <= s.history.length; i++) {
    const cp = checkpoint(s.history.slice(0, i));
    assert.deepEqual(replayCheckpoint(cp, cp.head, s.history.slice(i), head), full);
  }
  const cp = checkpoint(s.history.slice(0, 1));
  assert.throws(() => replayCheckpoint(cp, cp.head, [], head), /trusted head/);
  assert.throws(() => replayCheckpoint(cp, cp.head, [s.history[0], s.history[1]], head), /missing/);
  const damaged = clone(cp); damaged.view.objects[0][1] = 1;
  assert.throws(() => replayCheckpoint(damaged, cp.head, s.history.slice(1), head), /trusted head/);
  const replacedAnchor = { ...cp.head, root_hash: "symbolic-v1:forged" };
  assert.throws(() => replayCheckpoint(cp, replacedAnchor, s.history.slice(1), head), /anchor/);
  assert.throws(() => verifyHistory(s.history.slice(1), head), /missing/);
  assert.throws(() => verifyHistory([...s.history].reverse(), head), /missing/);
  const mutated = clone(s.history); mutated[0].value = 1;
  assert.throws(() => verifyHistory(mutated, head), /content identity/);
  assert.throws(() => verifyHistory(s.history.slice(0, 1), head), /trusted head/);
  assert.throws(() => verifyHistory([...s.history, s.history[1]], head));
});

test("an epoch-advanced checkpoint rejects an older suffix even with ordered TIDs", () => {
  const s = replay([...commit(A, 0), item("reset"), ...commit(B, 1)]);
  const cp = checkpoint(s.history.slice(0, 1), 2);
  assert.throws(() => replayCheckpoint(cp, cp.head, s.history.slice(1), trustedHead(replayCommitted(s.history), 2)), /predates/);
});

traceCase("an earlier committed TID cannot authorize another request completion", [...commit(A, 0), item("reset"), item("submit", 1, B), item("make_visible", 1), item("device_complete", 1), item("publish_completion", 1)], s => {
  assert.equal(s.tid, 1); assert.equal(s.history[0].request_id, A.request_id);
  assert.equal(s.request.request_id, B.request_id); assert.equal(s.phase, Phase.DEVICE_COMPLETE); assert.equal(s.completion, null);
});

test("finite guard defensively rejects an exhausted acknowledgement without changing state", () => {
  const full = replay([...commit(A, 0), item("reset"), ...commit(B, 1)]);
  // Such allocation is unreachable now: make_visible rejects earlier. Keep the
  // old v3 acknowledgement guard as a defensive source regression as well.
  const boundary = { ...clone(full), phase: Phase.DEVICE_COMPLETE, request: C };
  assert.deepEqual(step(boundary, "durable_ack", { epoch: boundary.epoch }), boundary);
  assert.deepEqual(step(boundary, "publish_completion"), boundary);
});


test("checkpoint epoch cannot roll back with either empty or nonempty suffix", () => {
  const first = replay(commit(A, 0));
  const durable = replayCommitted(first.history);
  const cp = checkpoint(first.history, 2);
  assert.deepEqual(replayCheckpoint(cp, cp.head, [], trustedHead(durable, 2)), durable);
  // The record itself is epoch0, but the independent checkpoint has advanced
  // to epoch2. An unchanged root/TID must not authorize rolling that back.
  assert.throws(() => replayCheckpoint(cp, cp.head, [], trustedHead(durable, 0)), /rolls back checkpoint/);
  const later = replay([item("reset"), item("reset"), ...commit(B, 2)], first);
  const final = replayCommitted(later.history);
  const suffix = later.history.slice(1);
  assert.deepEqual(replayCheckpoint(cp, cp.head, suffix, trustedHead(final, 2)), final);
  const rollback = { ...trustedHead(final, 2), epoch: 1 };
  assert.throws(() => replayCheckpoint(cp, cp.head, suffix, rollback), /rolls back checkpoint/);
});


test("failure inputs include nested stale acknowledgement and exact argument arity", () => {
  const current = replay([...commit(A, 0), item("reset"), item("submit", 1, B), item("make_visible", 1), item("device_complete", 1)]);
  const index = inputCases.length;
  const rejected = step(current, "durable_ack", { epoch: 0, request: B });
  const saved = JSON.parse(JSON.stringify(inputCases[index]));
  assert.equal(saved.operation, "step");
  const args = decodeInput(saved.args);
  assert.equal(args[1], "durable_ack"); assert.equal(args[2].epoch, 0);
  assert.deepEqual(model[saved.operation](...args), rejected);
  assert.deepEqual(rejected, current);
  assert.equal(decodeInput(encodeInput([current, "recover"])).length, 2);
  const explicit = decodeInput(encodeInput([current, "recover", undefined]));
  assert.equal(explicit.length, 3); assert.equal(explicit[2], undefined);
  const badIndex = inputCases.length;
  assert.throws(() => canonicalRecord({ invalid: true }));
  assert.equal(inputCases[badIndex].threw, true);
  assert.throws(() => model.canonicalRecord(...decodeInput(inputCases[badIndex].args)));
});
