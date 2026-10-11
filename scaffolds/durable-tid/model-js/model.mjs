// SPDX-License-Identifier: BSD-2-Clause
// Finite semantic oracle only. Symbolic hashes are injective terms, NOT crypto.
// Binding design: ../docs/{DESIGN,FORMAL}.md and the three repository skills.
export const Phase = Object.freeze({ EMPTY: "EMPTY", PROPOSED: "PROPOSED", COHERENT_VISIBLE: "COHERENT_VISIBLE", DEVICE_COMPLETE: "DEVICE_COMPLETE", PERSISTENT: "PERSISTENT", TRUSTED_COMPLETE: "TRUSTED_COMPLETE" });
export const SEMANTIC_STATES = Object.freeze(Object.values(Phase).filter(p => p !== Phase.EMPTY));
export const Actions = Object.freeze(["submit", "duplicate_submit", "make_visible", "device_complete", "durable_ack", "publish_completion", "reset", "recover", "replay_old_epoch"]);
export const EPOCH_VALUES = Object.freeze([0, 1, 2]);
export const TID_MAX = 2;
export const DOMAIN = Object.freeze({ version: 1, tid_max: TID_MAX, epochs: EPOCH_VALUES, request_ids: Object.freeze([1, 2, 3]), object_ids: Object.freeze([1, 2]), values: Object.freeze([0, 1]), operations: Object.freeze([1]) });
const copy = value => JSON.parse(JSON.stringify(value));
function freeze(value) { if (value && typeof value === "object") { Object.values(value).forEach(freeze); Object.freeze(value); } return value; }
function requireThat(ok, message) { if (!ok) throw new TypeError(message); }
const equal = (a, b) => JSON.stringify(a) === JSON.stringify(b);
function fields(value, names) { requireThat(value !== null && typeof value === "object" && !Array.isArray(value) && equal(Object.keys(value).sort(), [...names].sort()), "noncanonical fields"); }
function byte(n) { requireThat(Number.isInteger(n) && n >= 0 && n <= 255, "not uint8"); return n.toString(16).padStart(2, "0"); }
function count(n) { requireThat(Number.isInteger(n) && n >= 0 && n <= 0xffffffff, "not uint32"); return n.toString(16).padStart(8, "0"); }
function ascii(s) { requireThat(typeof s === "string" && /^[\x20-\x7e]*$/.test(s), "non-ASCII symbolic term"); return [...s].map(c => byte(c.charCodeAt(0))).join(""); }
// Each part is framed by uint32 big-endian BYTE length, then exact bytes in hex.
// Tags and payloads are ASCII; no Unicode, optionals, omitted fields or defaults.
function term(tag, parts) { return "symbolic-v1:" + tag + ":" + parts.map(p => count(p.length / 2) + p).join(""); }
function requestValid(r) { return r && DOMAIN.request_ids.includes(r.request_id) && DOMAIN.object_ids.includes(r.object_id) && r.operation === 1 && DOMAIN.values.includes(r.value) && equal(Object.keys(r).sort(), ["object_id", "operation", "request_id", "value"]); }
export function canonicalRequest(r) { requireThat(requestValid(r), "request outside finite domain"); return [1, r.request_id, r.object_id, r.operation, r.value].map(byte).join(""); }
export function contentHash(r) { canonicalRequest(r); return term("content", [[1, r.operation, r.value].map(byte).join("")]); }
export function canonicalRecord(r) {
  fields(r, ["version", "epoch", "tid", "request_id", "object_id", "operation", "parent_tid", "content_hash", "result", "value"]);
  requireThat(r.version === 1 && EPOCH_VALUES.includes(r.epoch) && r.tid >= 1 && r.tid <= TID_MAX && Number.isInteger(r.tid) && r.parent_tid === r.tid - 1 && r.result === 1, "invalid record identity/order/result");
  const req = requestOf(r); canonicalRequest(req);
  requireThat(r.content_hash === contentHash(req), "content identity mismatch");
  const prefix = [r.version, r.epoch, r.tid, r.request_id, r.object_id, r.operation, r.parent_tid, r.result, r.value].map(byte).join("");
  const hashBytes = ascii(r.content_hash);
  return prefix + count(hashBytes.length / 2) + hashBytes;
}
function requestOf(r) { return { request_id: r.request_id, object_id: r.object_id, operation: r.operation, value: r.value }; }
function emptyView() { return { objects: [], requests: [] }; }
function canonicalView(view) {
  fields(view, ["objects", "requests"]);
  requireThat(Array.isArray(view.objects) && Array.isArray(view.requests) && view.objects.length <= 2 && view.requests.length <= TID_MAX, "invalid view bounds");
  let last = 0; const objectBytes = [];
  for (const o of view.objects) {
    requireThat(Array.isArray(o) && o.length === 2 && DOMAIN.object_ids.includes(o[0]) && o[0] > last && DOMAIN.values.includes(o[1]), "noncanonical object view");
    last = o[0]; objectBytes.push(...o);
  }
  last = 0; const requestBytes = []; const tids = new Set();
  for (const r of view.requests) {
    requireThat(Array.isArray(r) && r.length === 5 && r[0] > last && r[4] >= 1 && r[4] <= TID_MAX && !tids.has(r[4]), "noncanonical request index");
    canonicalRequest({ request_id: r[0], object_id: r[1], operation: r[2], value: r[3] });
    tids.add(r[4]); last = r[0]; requestBytes.push(...r);
  }
  return byte(1) + count(view.objects.length) + objectBytes.map(byte).join("") + count(view.requests.length) + requestBytes.map(byte).join("");
}
function genesis() { const view = emptyView(); return { tid: 0, last_epoch: 0, root: term("genesis", [canonicalView(view)]), view }; }
function rootFor(previous, record, view) { return term("root", [ascii(previous), canonicalRecord(record), canonicalView(view)]); }
export function reduceCommitted(previous, record) {
  canonicalRecord(record); canonicalView(previous.view);
  requireThat(record.tid === previous.tid + 1 && record.parent_tid === previous.tid && record.epoch >= previous.last_epoch, "missing/reordered/stale committed record");
  requireThat(!previous.view.requests.some(r => r[0] === record.request_id), "duplicate committed request identity");
  const view = {
    objects: [...previous.view.objects.filter(o => o[0] !== record.object_id), [record.object_id, record.value]].sort((a,b) => a[0]-b[0]),
    requests: [...previous.view.requests, [record.request_id, record.object_id, record.operation, record.value, record.tid]].sort((a,b) => a[0]-b[0]),
  };
  return freeze({ tid: record.tid, last_epoch: record.epoch, root: rootFor(previous.root, record, view), view });
}
export function replayCommitted(records) { requireThat(Array.isArray(records), "history must be ordered array"); return records.reduce(reduceCommitted, freeze(genesis())); }
export function trustedHead(durable, epoch = durable.last_epoch) {
  requireThat(EPOCH_VALUES.includes(epoch) && epoch >= durable.last_epoch, "invalid trusted epoch");
  return freeze({ epoch, last_tid: durable.tid, root_hash: durable.root });
}
function verifyHead(durable, head) {
  fields(head, ["epoch", "last_tid", "root_hash"]);
  requireThat(EPOCH_VALUES.includes(head.epoch) && head.epoch >= durable.last_epoch && head.last_tid === durable.tid && head.root_hash === durable.root, "history does not match independently trusted head");
  return durable;
}
export function verifyHistory(records, head) { return verifyHead(replayCommitted(records), head); }
// The caller retains the checkpoint head as a separate trusted anchor. Its
// root binds the derived view plus last committed record and predecessor root.
export function checkpoint(records, epoch) {
  const state = replayCommitted(records);
  const prior = records.length ? replayCommitted(records.slice(0, -1)).root : null;
  return freeze({ version: 1, head: trustedHead(state, epoch ?? state.last_epoch), view: copy(state.view), last_record: records.length ? copy(records.at(-1)) : null, previous_root: prior });
}
export function replayCheckpoint(cp, anchor, suffix, finalHead) {
  fields(cp, ["version", "head", "view", "last_record", "previous_root"]);
  requireThat(cp.version === 1 && equal(cp.head, anchor), "untrusted checkpoint anchor");
  canonicalView(cp.view); let state;
  if (cp.head.last_tid === 0) {
    requireThat(cp.last_record === null && cp.previous_root === null && equal(cp.view, emptyView()), "invalid genesis checkpoint");
    state = genesis();
  } else {
    canonicalRecord(cp.last_record);
    requireThat(cp.last_record.tid === cp.head.last_tid && typeof cp.previous_root === "string" && cp.previous_root.startsWith("symbolic-v1:"), "checkpoint boundary mismatch");
    state = { tid: cp.last_record.tid, last_epoch: cp.last_record.epoch, view: copy(cp.view), root: rootFor(cp.previous_root, cp.last_record, cp.view) };
  }
  verifyHead(state, anchor);
  requireThat(EPOCH_VALUES.includes(finalHead?.epoch) && finalHead.epoch >= anchor.epoch, "final head rolls back checkpoint trust epoch");
  requireThat(Array.isArray(suffix), "suffix must be ordered array");
  // A suffix cannot predate the checkpoint's current trust epoch.
  requireThat(suffix.every(r => r.epoch >= anchor.epoch), "suffix predates checkpoint epoch");
  state = suffix.reduce(reduceCommitted, state);
  return freeze(verifyHead(state, finalHead));
}
export function initialState() { const d = genesis(); return freeze({ phase: Phase.EMPTY, epoch: 0, tid: 0, last_allocated_tid: 0, request: null, completion: null, history: [], root: d.root, view: d.view, outcome: "idle", dupSeen: false }); }
function durableOf(s) { return replayCommitted(s.history); }
function result(s, edits) { return freeze({ ...copy(s), ...edits }); }
function exactEpoch(s, p) { return p.epoch === undefined || p.epoch === s.epoch; }
const DEFAULT_REQUEST = Object.freeze({ request_id: 1, object_id: 1, operation: 1, value: 0 });
export function step(s, action, param = {}) {
  requireThat(Actions.includes(action), "unknown action");
  const p = param ?? {};
  if (action === "replay_old_epoch") return result(s, {});
  if (!["reset", "recover"].includes(action) && !exactEpoch(s, p)) return result(s, {});
  if (action === "reset") {
    if (s.epoch === EPOCH_VALUES.at(-1)) return result(s, { outcome: "epoch_capacity" });
    return result(s, { phase: Phase.EMPTY, epoch: s.epoch + 1, last_allocated_tid: s.tid, request: null, completion: null, outcome: "reset", dupSeen: false });
  }
  if (action === "recover") {
    if (s.epoch === EPOCH_VALUES.at(-1)) return result(s, { outcome: "epoch_capacity" });
    const d = verifyHistory(s.history, { epoch: s.epoch, last_tid: s.tid, root_hash: s.root });
    requireThat(d.root === s.root && equal(d.view, s.view) && d.tid === s.tid, "stored trusted root/view mismatch");
    return result(s, { phase: Phase.EMPTY, epoch: s.epoch + 1, last_allocated_tid: d.tid, request: null, completion: null, outcome: "recovered", dupSeen: false });
  }
  if (action === "submit" || action === "duplicate_submit") {
    const r = p.request ?? (action === "duplicate_submit" ? s.request : DEFAULT_REQUEST);
    if (!requestValid(r)) return result(s, { outcome: "invalid_request" });
    const committed = s.history.find(f => f.request_id === r.request_id);
    if (committed) {
      if (canonicalRequest(requestOf(committed)) !== canonicalRequest(r)) return result(s, { outcome: "identity_conflict" });
      if (s.request && canonicalRequest(s.request) !== canonicalRequest(r)) return result(s, { outcome: "busy" });
      return result(s, { phase: Phase.PERSISTENT, request: copy(r), completion: null, last_allocated_tid: s.tid, outcome: "already_committed", dupSeen: true });
    }
    if (s.phase !== Phase.EMPTY) {
      return result(s, { dupSeen: equal(s.request, r) || s.dupSeen, outcome: s.request?.request_id === r.request_id ? (canonicalRequest(s.request) === canonicalRequest(r) ? "duplicate_pending" : "identity_conflict") : "busy" });
    }
    return result(s, { phase: Phase.PROPOSED, request: copy(r), completion: null, outcome: "accepted", dupSeen: false });
  }
  if (action === "make_visible" && s.phase === Phase.PROPOSED) {
    if (s.tid >= TID_MAX) return result(s, { outcome: "tid_capacity" });
    return result(s, { phase: Phase.COHERENT_VISIBLE, last_allocated_tid: s.tid + 1, outcome: "allocated" });
  }
  if (action === "device_complete" && s.phase === Phase.COHERENT_VISIBLE) return result(s, { phase: Phase.DEVICE_COMPLETE, outcome: "device_complete" });
  if (action === "durable_ack" && s.phase === Phase.DEVICE_COMPLETE) {
    if (s.tid >= TID_MAX || s.last_allocated_tid !== s.tid + 1) return result(s, {});
    const record = { version: 1, epoch: s.epoch, tid: s.tid + 1, ...copy(s.request), parent_tid: s.tid, content_hash: contentHash(s.request), result: 1 };
    const d = reduceCommitted(durableOf(s), record);
    return result(s, { phase: Phase.PERSISTENT, tid: d.tid, history: [...copy(s.history), record], root: d.root, view: d.view, outcome: "committed" });
  }
  if (action === "publish_completion" && s.phase === Phase.PERSISTENT) {
    const record = s.history.find(f => canonicalRequest(requestOf(f)) === canonicalRequest(s.request));
    requireThat(record, "completion has no committed fact");
    return result(s, { phase: Phase.TRUSTED_COMPLETE, completion: { epoch: s.epoch, request_id: record.request_id, tid: record.tid, content_hash: record.content_hash }, outcome: "complete" });
  }
  return result(s, {});
}
export function replay(trace, start = initialState()) { return trace.reduce((s, item) => step(s, typeof item === "string" ? item : item.action, typeof item === "string" ? undefined : item.param), start); }
const key = s => JSON.stringify(s);
export function exploreReachableStates(start = initialState(), maxDepth = 6, limits = { maxStates: 10000, maxEdges: 100000 }) {
  requireThat(Number.isInteger(limits.maxStates) && limits.maxStates > 0 && Number.isInteger(limits.maxEdges) && limits.maxEdges > 0, "positive exploration limits required");
  requireThat(Number.isInteger(maxDepth) && maxDepth >= 0 && maxDepth <= 32, "depth must be 0..32");
  const states = new Map([[key(start), start]]); const paths = new Map([[key(start), []]]); const edges = []; const transitionErrors = []; let frontier = [start]; let limited = false;
  search: for (let depth = 0; depth < maxDepth && frontier.length; depth++) {
    const next = [];
    for (const state of frontier) for (const action of Actions) {
      const requests = (action === "submit" || action === "duplicate_submit") ? DOMAIN.request_ids.flatMap(request_id => DOMAIN.object_ids.flatMap(object_id => DOMAIN.values.map(value => ({ request_id, object_id, operation: 1, value })))) : [null];
      const epochs = ["reset", "recover", "replay_old_epoch"].includes(action) ? [undefined] : EPOCH_VALUES;
      for (const request of requests) for (const epoch of epochs) {
        const param = { ...(epoch === undefined ? {} : { epoch }), ...(request ? { request } : {}) };
        if (edges.length + transitionErrors.length >= limits.maxEdges) { limited = true; break search; }
        const trace = [...paths.get(key(state)), { action, param }];
        let to;
        try { to = step(state, action, param); }
        catch (e) { transitionErrors.push({ invariant: "transition_threw", state, trace, detail: e.message, expectedThrow: true }); continue; }
        if (!states.has(key(to)) && states.size >= limits.maxStates) { limited = true; break search; }
        edges.push({ from: state, action, param, to, trace });
        if (!states.has(key(to))) { states.set(key(to), to); paths.set(key(to), trace); next.push(to); }
      }
    }
    frontier = next;
  }
  return { start, maxDepth, domain: DOMAIN, states, paths, edges, transitionErrors, limits: { ...limits }, limited, frontierRemaining: limited || frontier.length > 0 };
}
export function checkInvariants(exploration) {
  const { states, edges, paths } = exploration; const violations = [...exploration.transitionErrors];
  const fail = (invariant, state, trace, detail) => violations.push({ invariant, state, trace, detail });
  for (const s of states.values()) {
    try {
      const d = verifyHistory(s.history, { epoch: s.epoch, last_tid: s.tid, root_hash: s.root });
      requireThat(equal(d.view, s.view) && s.tid <= s.last_allocated_tid && s.last_allocated_tid <= TID_MAX, "durable/allocated/view mismatch");
      if (s.phase === Phase.TRUSTED_COMPLETE) {
        const r = s.history.find(r => r.request_id === s.completion?.request_id && r.tid === s.completion?.tid && r.content_hash === s.completion?.content_hash);
        requireThat(r && s.completion.epoch === s.epoch && canonicalRequest(requestOf(r)) === canonicalRequest(s.request), "completion does not belong to current request and durable prefix");
      }
      for (let i = 0; i <= s.history.length; i++) {
        const cp = checkpoint(s.history.slice(0, i));
        requireThat(equal(replayCheckpoint(cp, cp.head, s.history.slice(i), trustedHead(d, s.epoch)), d), "checkpoint suffix diverged");
      }
      for (const record of s.history) requireThat(canonicalRecord(record) === canonicalRecord(Object.fromEntries(Object.entries(record).reverse())), "canonical encoding depends on insertion order");
    } catch (e) { fail("durable_identity_replay_and_completion", s, paths.get(key(s)), e.message); }
  }
  for (const edge of edges) {
    const { from, to, action, param, trace } = edge;
    if (!equal(to.history.slice(0, from.history.length), from.history)) fail("committed_record_never_mutates", to, trace);
    if (!["durable_ack"].includes(action) && !equal(from.history, to.history)) fail("only_durability_ack_commits", to, trace);
    if (to.history.length > from.history.length && (to.tid !== from.tid + 1 || to.history.length !== from.history.length + 1)) fail("strict_commit_order", to, trace);
    if (action === "replay_old_epoch" || (param.epoch !== undefined && param.epoch !== from.epoch)) {
      if (!equal(from, to)) fail("noncurrent_epoch_cannot_authorize", to, trace);
    }
    if (!equal(step(from, action, param), to)) fail("same_input_same_next_state", to, trace);
  }
  return { model: "durable-tid-semantic-v1", domain: DOMAIN, maxDepth: exploration.maxDepth, frontierRemaining: exploration.frontierRemaining, limits: exploration.limits, limited: exploration.limited, boundedDepthFullyExplored: !exploration.limited, statesVisited: states.size, edgesExplored: edges.length, transitionAttempts: edges.length + exploration.transitionErrors.length, passed: violations.length === 0, violations, scope: "bounded semantic exploration; symbolic hashes; no liveness, cryptographic or physical durability proof" };
}
