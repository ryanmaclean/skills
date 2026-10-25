---
name: proveryay
description: Automatic verification hook for sub-agent claims. Fires on SubagentStop, extracts structured <claims> blocks from agent reports, verifies each via fleet-eval probes in parallel, injects verdicts into Claude's context so claims cannot be trusted without proof. Reinforces the "claim -> proof" reflex.
trigger: verify agent claim, proveryay, automatic verification, hook, prove it, was that actually true, did it actually work, trust but verify
---

# proveryay — auto-verify sub-agent claims

**prove + yay** — the dopamine of a green verdict reinforces the reflex.

## When it fires

`SubagentStop` hook. Automatic. No Claude intervention. Claude will see the
verdict in the next turn's context as stdout injected by the harness. If no
`<claims>` block is present in the sub-agent's final message, the hook is
silent — zero noise for agents that don't emit structured claims.

## What agents should emit

Append a single claims block to the final report. Two formats are accepted;
pick one, do not mix. Full reference in `./CLAIMS-TEMPLATE.md`.

**Format 1 — YAML-under-XML (canonical):**

```
<claims>
- claim: "fb-vm-24 is reachable via jump host"
  probe: "ssh -J studio@10.0.3.1 -p 2225 builder@localhost echo ALIVE"
  expect: "ALIVE"
- claim: "bop-catalog endpoint returns v1 schema"
  probe: "curl -sS -m 10 'http://10.0.3.210:3000/string/bop/releases/download/catalog%2Flatest/latest.json'"
  expect: "schema_version"
</claims>
```

**Format 2 — Atomic-claim JSON (FactScore / SAFE / Claimify style):**

````
```json
{"atomic_claims": [
  {"claim": "gitea up", "probe": "curl -sS -m 5 http://10.0.3.210:3000/api/v1/version", "expect": "version"}
]}
```
````

**Rejected format (emits error, does NOT silently pass):**
`<claim><subject>...</subject><probe>...</probe><expected>...</expected></claim>` —
this is the verification-theatre anti-pattern. Agents self-declare
`<verdict>PASS</verdict>` without an independent probe. `proveryay` detects
and rejects it.

Contract:
- `claim` — short human description.
- `probe` — exact shell command that runs in ≤10s.
- `expect` — substring that must appear in probe stdout (or omit for exit-code pass).

## Manual invocation

```
# single claim
proveryay verify "gitea up" --probe "curl -sS -m 5 http://10.0.3.210:3000/api/v1/version" --expect "version"

# batch: pipe in <claims>...</claims> text OR bare YAML list
cat claims.yaml | proveryay batch
```

## Reading the output

```
🎯 PROVERYAY (3 claims checked)
  ✓ fb-vm-24 is reachable - PASS (38ms)
  ✗ bop-catalog v1 schema - FAIL: evidence='curl: (7) Failed to connect'
  ? qdrant collection - INCONCLUSIVE: probe timed out
Overall: 1/3 PASSED
```

- `✓` PASS — probe ran, expected evidence present.
- `✗` FAIL — probe ran, evidence absent.
- `?` INCONCLUSIVE — timeout or unclassifiable.
- `!` INVALID — malformed claim (missing probe).

Full evidence (untruncated) goes to `/tmp/proveryay.log` (JSONL).

## Canonical probe patterns

| Claim shape | Probe | Expect |
|---|---|---|
| service X up at URL | `curl -sS -w 'HTTP %{http_code}' --max-time 10 URL` | `HTTP 200` |
| pkg P installed on host H | `ansible H -m raw -a "pkg info P"` | pkg name |
| process P running on H | `ansible H -m raw -a "pgrep -af P"` | pattern |
| file F has sha | `sha256sum F` | expected hex |
| HTTP status | `curl -sS -o /dev/null -w '%{http_code}' URL` | `200` |

See `fleet-eval` skill for the complete taxonomy.

## When NOT to emit `<claims>`

Only for **deployment / state-change assertions**. Skip for:
- planning docs, research, brainstorming
- opinion answers
- narrative summaries

Rule of thumb: if your sentence is "X is deployed / running / reachable / shipped / installed / has content C" — emit a claim. Otherwise don't.

## Circuit breaker

After 3 consecutive `fleet-eval` errors (not FAILs — actual exec errors), the
hook disables itself for the session and injects a single warning. State at
`/tmp/proveryay.circuit`. Reset:

```
rm /tmp/proveryay.circuit
```

Session PASSes reset the counter automatically.

## Cross-refs

- `fleet-eval` — the underlying runnable and claim taxonomy
- `superpowers:verification-before-completion` — manual counterpart
- `datadog-metrics` — emit `rigfleet.eval.verdict` alongside
- `sc:reflect` — post-task review; complements auto-verification

## Files

- Hook runnable: `/Users/studio/.local/bin/proveryay`
- Claims template for sub-agent prompts: `./CLAIMS-TEMPLATE.md`
- Self-test: `/Users/studio/rust-town/fleet-ops/scripts/proveryay-self-test.sh`
- Logs: `/tmp/proveryay.log` (JSONL)
- Hook registration: `~/.claude/settings.json` under `hooks.SubagentStop`
