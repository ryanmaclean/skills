## Claims block (REQUIRED when you make verifiable deployment/state assertions)

Emit a structured claims block at the end of your report whenever you assert:
- "X is deployed / running / reachable / shipped / installed"
- "file F has content C" / "pkg P version V"
- "endpoint U returns R"

`proveryay` accepts **two** formats. Pick one; do not mix.

---

### Format 1 — YAML-under-XML (canonical)

```
<claims>
- claim: "<short human description>"
  probe: "<exact command that verifies it>"
  expect: "<substring the probe output must contain>"
</claims>
```

Example:

```
<claims>
- claim: "ii-agent v5.4.0 running on ubrpi411"
  probe: "ansible ubrpi411 -m raw -a 'systemctl is-active ii-agent'"
  expect: "active"
- claim: "Gitea catalog endpoint live"
  probe: "curl -sS -m 5 http://10.0.3.210:3000/api/v1/version"
  expect: "version"
</claims>
```

---

### Format 2 — Atomic-claim JSON (FactScore / SAFE / Claimify style)

One JSON object with an `atomic_claims` array. Accepted bare or inside a
fenced ` ```json ` block:

````
```json
{"atomic_claims": [
  {"claim": "ls /tmp exists", "probe": "ls -d /tmp", "expect": "/tmp"},
  {"claim": "gitea up", "probe": "curl -sS -m 5 http://10.0.3.210:3000/api/v1/version", "expect": "version"}
]}
```
````

Each claim is treated as a single atomic proposition per FactScore/SAFE
methodology — one claim, one probe, one expected substring. No nesting.

---

### REJECTED format — XML-per-field

**Do not emit this.** `proveryay` detects it and emits an error instead of
silently skipping. The anti-pattern:

```
<claim id="x">
  <subject>...</subject>
  <probe>...</probe>
  <expected>...</expected>
  <evidence>...</evidence>
  <verdict>PASS</verdict>
</claim>
```

This format lets an agent self-declare `<verdict>PASS</verdict>` without any
independent probe — it is the verification-theatre failure mode. Use Format 1
or Format 2, never this.

---

### Rules

- `probe` must be self-contained and run in ≤10s.
- Quote strings containing colons or shell metacharacters.
- Multiple claims: repeat the `- claim:` list item (Format 1) or add array
  entries (Format 2).
- Omit the block entirely for planning, brainstorming, or research — only
  emit when you are asserting verifiable state.

The `proveryay` hook verifies every claim automatically on `SubagentStop`.
Do not invoke it manually — emit the block and the verdict lands in the
parent agent's next turn.

Force-select a parser (debug/testing only):

```
cat claims.txt | proveryay batch --format yaml     # YAML-under-XML only
cat claims.txt | proveryay batch --format json     # atomic JSON only
cat claims.txt | proveryay batch                   # auto-detect (default)
```
