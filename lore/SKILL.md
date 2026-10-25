---
name: lore
description: Agent-facing discovery + query interface to fleet_ledger (Qdrant long-term memory). Verbs query, get, add, list, help. Use when you need to check "what do we know about X", "have we seen Y before", "did anyone catalog Z", or "lookup in fleet memory".
trigger: fleet memory, fleet ledger, lore query, have we seen this, what do we know about, did we already try, lookup in fleet memory, institutional memory, qdrant search, remember last time
author: Ryan MacLean
version: 1.1.0
---

# lore — fleet_ledger discovery + query

Fleet institutional memory. Reads and writes `fleet_ledger` at
`10.0.3.206:6333` (Qdrant, 1024-dim Cosine, arctic-embed2 via Ollama at
`localhost:11434`). AX-first: structured JSON in, structured JSON out.

## Why this skill exists

Today's `context-orchestration-fleet-ledger` decision named fleet_ledger
as durable long-term memory. Every sub-agent dispatched was hand-rolling
`ledger-upsert.py` or raw curl. `lore` is the one verb surface. All five
verbs return `schema_version: "v1"` so breaking changes are detectable.

## Verbs

Run `lore help` for the live JSON manifest. Summary:

| Verb   | Purpose                                  | Primary flags |
|--------|------------------------------------------|---------------|
| query  | Semantic search (embeds & kNN)           | `--top-k N` `--min-score F` `--kind k1,k2` |
| get    | Exact lookup by (kind, subject)          | `--kind K` or `kind:subject` |
| add    | Upsert (idempotent UUIDv5 replace)       | `--kind` `--subject` `--text` `--data` |
| list   | Scroll, metadata only (no text)          | `--kind` `--since` `--limit` |
| help   | Self-describing manifest                 | `--verb V` to drill down |

## Example — query

Input:
```
lore query "fb-vm-24 disk" --top-k 3
```

Output (truncated):
```json
{
  "schema_version": "v1",
  "verb": "query",
  "count": 3,
  "results": [
    {"id":"...", "score":0.82, "kind":"decision",
     "subject":"fb-vm-24-disk-fix", "ts":"2026-04-15T...",
     "text_preview":"fb-vm-24 disk grown from 6 to 24 GiB ...",
     "data":{...}}
  ]
}
```

Zero-result form is actionable, not silent:
```json
{"schema_version":"v1","verb":"query","count":0,"results":[],
 "suggestion":"try lore list --kind=decision --since=2026-04-01"}
```

## Example — get

```
lore get decision:fb-vm-24-disk-fix
# or
lore get fb-vm-24-disk-fix --kind decision
```

Returns the full payload as `{schema_version, verb:"get", found:true, record:{id,kind,subject,...}}`
or `{found:false, how_to_fix:"try lore query \"fb-vm-24 disk\""}`.

## Example — add (idempotent)

```
lore add --kind decision --subject my-decision-2026-04-15 \
  --text "We chose X over Y because ..." \
  --data '{"chose":"X","over":"Y"}'
```

Output:
```json
{"schema_version":"v1","verb":"add","id":"<uuid5>",
 "count_before":55,"count_after":55,"idempotent_replace":true}
```

If run twice with the same `(kind, subject)`, the second run returns
`idempotent_replace:true` and `count_before == count_after`. The calling
agent uses this to de-dupe its own activity.

## Example — list

```
lore list --kind decision --since 2026-04-15 --limit 20
```

Output:
```json
{"schema_version":"v1","verb":"list","count":N,
 "records":[{"id":"...","kind":"decision","subject":"...","ts":"..."}, ...]}
```

No `text` field — list view is metadata only by design.

## Example — help

```
lore help            # full manifest (all 5 verbs)
lore help --verb query   # just query's schema
```

Returns JSON, not prose. Canonical discovery entrypoint.

## Errors (structured, always)

```json
{"schema_version":"v1","error":"QDRANT_API_KEY env var not set",
 "how_to_fix":"export QDRANT_API_KEY=... (see ~/.zshrc)"}
```

Exit codes: `0` success, `2` user error (bad flags / missing env),
`3` upstream error (qdrant/ollama down), `4` not-found.

## Gotchas (hard rules)

- **Canonical UUIDv5 namespace**: `6b1f2c3d-4e5f-5a6b-7c8d-9e0f1a2b3d4e`.
  Never change. Changing it means every `lore add` stops being idempotent
  against prior records. The key string is `"{kind}:{subject}"` (no
  whitespace, lowercase kind).
- **Key shape `(kind, subject)`**: `subject` is the human-memorable slug,
  `kind` is one of `attempt|decision|probe|gem|blocker|workaround`. Same pair → same id.
  - `attempt` — past try with outcome + evidence
  - `decision` — standing policy / architectural choice with rationale
  - `probe` — reusable claim→verify pattern (claim_shape + probe_cmd + expected)
  - `gem` — session conversation signal (user correction)
  - `blocker` — a workstream hit a wall; successor agents retrieve before attempting
  - `workaround` — graceful-degradation pattern for when the normal path isn't available
- **3-strikes cleanup convention**: when an agent stops at 3-strikes, the
  canonical cleanup is `lore add --kind blocker --subject <workstream>-<date>
  --text @STOP-REPORT --data '{"status":"blocked","attempted":[...],"next_strategies":[...]}'`.
  Successor agents SHOULD `lore query <workstream>` before re-deriving.
- **`gem` excluded from `query` by default**: gems are raw corrections
  ("stop doing X") with noisy language; exclude unless explicitly asked
  with `--kind gem` or `--kind attempt,decision,probe,gem`.
- **Score floor 0.4 default**: arctic-embed2 cosine scores below 0.4 are
  near-noise. Tune via `--min-score`. Don't read tea leaves below 0.3.
- **Ollama endpoint: the runnable tries localhost:11434 first.
  embedding fails, set LORE_OLLAMA_URL=http://10.0.3.207:11434 (fleet embedder on qnas macvlan).
  `10.0.3.207:11434` via `LORE_OLLAMA_URL` env.
- **No secrets in code**: `QDRANT_API_KEY` via env only. Runnable refuses
  with structured error if missing.

## Cross-refs

- Design doc: `/Users/studio/rust-town/.planning/fleet-ledger-design-2026-04-15.md`
- Architectural decision record: fleet_ledger entry `context-orchestration-fleet-ledger`
- Seed script: `/Users/studio/rust-town/fleet-ops/scripts/fleet-ledger-ingest.nu`
- Related skills: `fleet-eval` (probe runner), `proveryay` (claim verifier), `qcli` (low-level Qdrant CLI — all collections)

## Files

- Skill doc: `/Users/studio/.claude/skills/lore/SKILL.md` (this file)
- Runnable: `/Users/studio/.local/bin/lore` (Python 3, MIT licensed)
- Self-test: `lore help | python3 -m json.tool` (must parse as JSON)
