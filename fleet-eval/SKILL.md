---
name: fleet-eval
description: Verify a claim against reality — shapes assertions into {subject, expected, probe, evidence, verdict} and runs independent probes. Use when another agent reports X works/ships/is deployed and proof is needed before trusting it, or when the user says verify this claim or fleet-eval.
trigger: verify claim, prove it works, is X actually up, smoke test, verification, reality check, claim proof, did it really ship
---

# fleet-eval — Claim -> Proof Protocol

## The gate
Every claim of form "X is deployed / working / reachable / shipped" must pass:
1. A deterministic **probe** (command or API call)
2. Return **expected evidence** (string, status code, file hash, etc.)
3. Emit a structured verdict JSON

## Claim types + standard probes

| Claim shape | Probe | Expected evidence |
|---|---|---|
| "service X at URL is up" | `curl -sS -w '\nHTTP %{http_code}\n' URL --max-time 10` | HTTP 2xx |
| "file F exists with content C" | `sha256sum F` | expected SHA |
| "pkg P installed on host H" | `ansible H -m raw -a "pkg info P"` | pkg info output |
| "process P running on host H" | `ansible H -m raw -a "pgrep -af P"` | non-empty |
| "nick N in IRC channel" | (use fleet-irc skill with NAMES query) | nick present |
| "Gitea release R exists" | `tea release view --repo ... --tag R` | release JSON |
| "schema file valid JSON Schema" | `python -c "import jsonschema; jsonschema.Draft202012Validator.check_schema(json.load(open('F')))"` | no exception |
| "VM V alive" | `ssh ... echo ALIVE` OR `pgrep -af qemu.*V` | "ALIVE" or non-empty |

## Output contract (always)

```json
{
  "claim": "string — the original claim being verified",
  "probe": "string — exact command run",
  "expected": "string — what constituted pass",
  "evidence": "string — actual probe output, truncated to 1KB",
  "verdict": "PASS | FAIL | INCONCLUSIVE",
  "ts": "ISO8601",
  "took_ms": <number>
}
```

Written to stdout as JSON; also appended to `/tmp/fleet-eval.log`.

## Usage patterns

### Invoke after every claim
```
fleet-eval verify "Phase 16 VM is ready" \
  --probe "ssh -J studio@10.0.3.1 -p 2225 builder@localhost echo ALIVE" \
  --expect "ALIVE"
```

### Batch verify (multi-claim reports)
```
fleet-eval batch < claims.jsonl
# Each line: {"claim": "...", "probe": "...", "expected": "..."}
```

### Negative expected (expected-fail)
```
fleet-eval verify "Integrations install fails missing datadog-agent" \
  --probe "..." --expect-exit 1 --expect-stderr-match "missing dependency"
```

## When NOT to use fleet-eval
- Pure planning / brainstorming — nothing to verify yet
- Questions about what something IS (use other skills)

## Composes with
- Any agent making deployment/infra claims
- `superpowers:verification-before-completion` (complementary; this is the runtime version)
- `datadog-metrics` (emit eval results as metrics)

## Related
- `/Users/studio/rust-town/fleet-ops/scripts/fleet-eval.nu` — the runnable
- `/Users/studio/rust-town/fleet-ops/scripts/drift-monitor.nu` — hourly drift checker
- `/Users/studio/rust-town/fleet-ops/data/declared-alive.toml` — drift inventory

## Reflex pattern (for any invoking agent)
When another agent says "X is deployed / working / shipped", invoke:
```
nu /Users/studio/rust-town/fleet-ops/scripts/fleet-eval.nu verify "CLAIM" --probe "CMD" --expect "VALUE"
```
One probe, one verdict. No retry loops. Proof > report.
