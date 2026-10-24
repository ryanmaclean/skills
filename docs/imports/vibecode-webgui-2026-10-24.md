# Import: ryanmaclean/vibecode-webgui skills (2026-10-24)

Moved from ryanmaclean/vibecode-webgui@129302a73be9719d213561116319d13ed679fd71 on 2026-10-24
(workstream WS4 of the vibecode-webgui repo split, tracking issues #2125 and #2130 there).
Plain copy; git history stays in the source repository.

```json
{"schema":"ryanlab.import.v1","source":"ryanmaclean/vibecode-webgui","source_sha":"129302a73be9719d213561116319d13ed679fd71","date":"2026-10-24"}
```

## Moved

| Here | Source path | Licence | Notes |
|---|---|---|---|
| datadog-operations/ | .claude/skills/datadog-operations/ | MIT (Datadog CLI Contributors, LICENSE kept); datadog-skill-go/pants Apache-2.0 (Pants) | docs/TEST-RESULTS.md not copied (credential-shaped values); laptop paths rewritten |
| datadog-dashboard-ops/ | datadog-dashboard-ops.skill (zip, unpacked) | MIT (inherited from vibecode-webgui) | |
| gastown-ops/ | .claude/skills/gastown-ops/ | MIT (inherited) | sits beside elixir-gastown-ops/; host paths rewritten to ~/gt |
| apple-silicon/ | .claude/skills/apple-silicon/ | MIT (inherited) | |
| openrouter-models/ | .claude/skills/openrouter-models/ | MIT (inherited) | hard-coded API key default removed |
| context-graph/ | .claude/skills/context-graph/ | MIT (inherited) | tundra_dome_graph.py is Gas Town specific |
| codex-personas/ | .codex/personas/ | MIT (inherited) | historical roundtable persona run notes |

Each directory has a PROVENANCE.md with the same facts.

## Not moved (deleted from vibecode-webgui, upstream is canonical)

| Source path | Upstream | Upstream licence | Reason |
|---|---|---|---|
| .claude/skills/systematic-debugging/ | obra/superpowers skills/systematic-debugging | MIT | unmodified upstream copy (3 blobs identical, rest older revisions); use upstream |
| .claude/skills/verification-before-completion/ | obra/superpowers | MIT | unmodified older upstream copy; use upstream |
| .claude/skills/differential-review/ | trailofbits/skills | CC-BY-SA-4.0 | licence outside the MIT/BSD/Apache allowlist |
| .claude/skills/mcp-builder/, skill-creator/, webapp-testing/ | anthropics/skills | Apache-2.0 upstream; the vendored copies lacked the referenced LICENSE.txt | use upstream |

## Kept in vibecode-webgui

.claude/skills/henshu (an i18n gate for vibecode-webgui messages/ and src/app; project specific, and it would
collide with the separate editorial "henshu" skill name), the other project skills, .claude/hooks,
.claude/settings.json, .codex/config.toml, .codex/README.md and .codex/package.json.
