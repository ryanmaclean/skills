---
name: w5
description: Investigative-journalism protocol for agents. Use when asked to investigate, dig into, audit, verify, or get to the bottom of a claim/event/mystery. Enforces Who/What/When/Where/Why/How, multi-source corroboration, primary-source priority, timeline construction, and adversarial balance. Not for simple Q&A — use when the answer itself is contested or non-obvious.
---

# W5 — Investigative Protocol

A repeatable checklist inspired by W5's long-running investigative practice
(Who, What, When, Where, Why — plus How). Use when the answer is contested,
the narrative is convenient, or a single source is pushing a conclusion.
Treat every assertion as a suspect until corroborated.

## When to invoke

Trigger phrases: "investigate", "dig into", "get to the bottom of",
"audit this", "verify this claim", "research this story", "what really
happened with X", "follow up on X", "is X true?", "why did X fail",
"who decided Y".

## When NOT to invoke

- Simple factual Q&A with a canonical answer (docs, man pages)
- Opinion or style requests
- Code generation / refactoring
- Pure planning with no contested fact
- One-shot `fleet-eval` verifications (single probe, single verdict)

If the question has an uncontested answer reachable in one lookup, skip W5.

## The W5 protocol (5 phases)

Run these in order. Do not skip. Each phase has a gate.

### Phase 1 — Scope (the 5 Ws + H)

Before searching anything, write the six questions down as an explicit list:

| # | Question | Example |
|---|----------|---------|
| 1 | **Who** | Who is involved? Who benefits? Who decides? |
| 2 | **What** | What exactly is claimed? What is the specific event? |
| 3 | **When** | What is the time window? First mention? Current state? |
| 4 | **Where** | In which system/repo/host/jurisdiction did it happen? |
| 5 | **Why** | What is the stated motive? What is the plausible motive? |
| 6 | **How** | What mechanism? What sequence of steps? |

**Gate:** if you cannot form all six questions, the investigation premise
is unclear. Stop and ask the caller, or narrow the scope.

### Phase 2 — Sources (identify ≥3 candidates before reading)

List at least three candidate sources and classify each:

| Tier | Kind | Examples |
|------|------|----------|
| **Primary** | Original record, direct stakeholder | Commit, signed document, raw log, on-the-record interview, system telemetry |
| **Secondary** | Commentary on primary | Plans, postmortems, chat messages referencing the event |
| **Tertiary** | Summary of secondary | News digests, summary threads, rumor |

Budget: allow time for primary sources first. Only consult secondary after
primary is exhausted. Never start with tertiary.

**Gate:** if only tertiary sources are available, flag the investigation
as "thin" in the output.

### Phase 3 — Corroborate (≥2 independent sources per claim)

For every factual assertion that will appear in the verdict:

- Require **two independent sources** — same-owner sources are not
  independent (e.g. two blog posts from the same author).
- If only one source exists, mark the claim `[uncorr]` — do not promote it
  to asserted fact.
- Distinguish "X said Y" from "Y is true." The former needs one source;
  the latter needs two.
- Watch for laundered single-sourcing: tertiary A cites tertiary B cites
  primary C. That is still one source.

### Phase 4 — Timeline + Synthesis

- Build a dated timeline: `YYYY-MM-DD[THH:MM] — event — source`.
- Walk the timeline looking for inconsistencies — gaps, reversed causality,
  implausible gaps between cause and effect.
- **Adversarial step (required):** ask "if the stated narrative is wrong,
  what would it look like instead? What evidence would that alternate
  narrative predict? Does any of it exist?"
- Record **dry holes** — hypotheses you tested and ruled out, with the
  reason. Protects the next agent from repeating the search.

### Phase 5 — Verdict + "What's missing"

Before writing the verdict, answer the terminal question:

> "What would change my mind? What am I not seeing?"

If the honest answer is "nothing could change my mind," the investigation
is biased — go back to Phase 3 and look harder for disconfirming evidence.

Only then write the verdict.

## Output contract

Every W5 investigation produces a markdown report with these sections
(headings exact):

```
## 5W table
| Who | What | When | Where | Why | How |

## Sources
- [tier] [type] url-or-path — one-line summary

## Timeline
- YYYY-MM-DD — event — [source-ref]

## Verdict
<2-5 sentence conclusion, each sentence tagged with source refs>

## Dry holes
- hypothesis — ruled out because <reason> — [source-ref]

## What could change the verdict
- <list of evidence that, if surfaced, would flip or weaken the verdict>
```

If the verdict asserts deployment or runtime state, also emit a
`<claims>` block so `proveryay` can verify automatically.

## Citation types (glossary)

| Tag | Meaning |
|-----|---------|
| `[doc]` | Primary document (commit, signed record, raw log, schema) |
| `[on-record]` | Named interview or named source, attributable |
| `[off-record]` | Unnamed but known to the investigator, attributable with care |
| `[anon]` | Anonymous source, no attribution possible |
| `[circ]` | Circumstantial inference (no direct evidence, pattern-based) |
| `[uncorr]` | Single-source claim; corroboration attempted and failed |

Every line in the Verdict or Timeline must carry at least one tag.

## Anti-patterns to avoid

- **Lede with speculation.** State the primary evidence first; speculate later, clearly labeled.
- **Modal chaining.** "Could / might / suggests / appears to" stacked without grounding is narrative, not reporting.
- **Stakeholder framing.** Press releases, vendor blogs, and authored postmortems are NOT neutral. Quote them, do not adopt their framing.
- **Skip the adversarial step.** If the narrative feels complete, that is precisely when to test it.
- **Stop-when-you-have-enough.** Do not stop at the first explanation that supports your prior. Require disconfirming search.
- **Anonymous-source inflation.** `[anon]` is not a substitute for `[doc]`.

## Examples

### Example A — "Investigate why deployment X failed"

- **Scope:** Who pushed X, what failed, when did first failure occur, where (which host/service), why per the stated log, how did it get past CI.
- **Sources:** `[doc]` commit + CI run logs (primary), `[doc]` host syslog (primary), `[on-record]` deployer's message (secondary), chat thread (tertiary — only for leads).
- **Corroborate:** CI log + host syslog must agree on the failure boundary before the verdict asserts a cause.
- **Timeline:** commit → CI pass → deploy → first error → rollback. Look for the gap between CI pass and first error — config drift often lives there.
- **Adversarial:** "If the deployer's stated cause is wrong, what else could explain both the CI pass and the host failure?" (env var drift, a dependency bumped out-of-band, a clock skew).
- **Verdict:** state mechanism with source tags; list dry holes (hypotheses tested and ruled out).

### Example B — "Verify that library Y is under license Z"

- **Scope:** Which version? Which file declares the license? Which jurisdiction?
- **Sources:** `[doc]` `LICENSE` file in the repo at the exact version tag (primary), `[doc]` package manifest (primary), `[tertiary]` SPDX aggregators (only as a cross-check, never as a primary).
- **Corroborate:** `LICENSE` file + package manifest must agree.
- **Adversarial:** "What if the `LICENSE` file was changed after publish? What if a vendored subdirectory has a different license?" — check git history and recurse into `vendor/`, `third_party/`, etc.
- **Verdict:** cite the exact file + commit SHA. If any vendored component has a different license, surface it.

## Integration with other skills

- **`proveryay`** — emit a `<claims>` block for any deployment or runtime assertion in the verdict so the SubagentStop hook can verify probes in parallel.
- **`fleet-eval`** — use for any probe-based verification step (single claim → single probe → verdict).
- **`lore`** — save the completed investigation as a `kind=decision` record with full citation set so future agents can reuse the work.
- **`council`** / **`tiger-team`** — invoke when the investigation benefits from multiple adversarial lenses (e.g. security vs ops vs architecture on the same incident).
- **`toyota-way`** (5 Whys) — deepens the "Why" step of the 5 Ws; W5 covers scope across all six dimensions, 5 Whys drills the causal chain vertically.

## How this differs from adjacent skills

| Skill | Use when |
|-------|----------|
| **W5** | Claim is contested; you need multi-source corroboration and a timeline before trusting the answer |
| **triz** | You already know the facts; the problem is a contradiction between two design qualities |
| **council** | You need adversarial review of a design/spec, not fact-finding |
| **tiger-team** | One hard problem needs 2–4 specialists attacking it simultaneously |
| **toyota-way / 5 Whys** | Root-cause drill on a known failure — causal chain, not scope |
| **fleet-eval** | One claim, one probe, one verdict — no investigation needed |

## Deployment-specific details

For fleet investigations in rust-town:

- **Primary sources:** Gitea repos + commits, Datadog events/logs, lore records, fleet manpages.
- **Secondary sources:** plans in `.planning/`, skill files.
- **Tertiary sources:** chat logs, IRC transcripts, session handoffs.
- Save completed investigations with `lore add --kind decision` so the next agent inherits the citation set.
- Fleet search: `searxng.local` (local fallback), `context7` MCP for upstream docs, existing skills first.

For a different deployment, swap these with your equivalents: code host,
monitoring system, long-term memory store.
