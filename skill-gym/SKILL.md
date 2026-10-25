---
name: skill-gym
description: Hill-climb a skill's prompt/SKILL.md against binary eval assertions using the autoresearch-skills harness. Baseline → generate variants → score each → keep the winner → repeat until plateau or 90%. Use when a skill underperforms, when adding a new skill that needs hardening, or when the user says "gym this skill", "hill-climb this skill", "improve pass rate", or wants autonomous skill self-improvement.
---

# skill-gym — hill-climb for skills

Eval gym for the skill corpus. Wraps `autoresearch-skills/` (eval runner +
assertions + variant generator + score history) into a repeatable loop that
improves ONE skill per run — never random edits, always measured deltas.

**Core loop:** seed → score → fail-analyze → variant → re-score → keep/discard → repeat.

## What this is (and isn't)

- IS: mechanical hill-climbing of a skill's prompt text against binary
  assertions in `autoresearch-skills/evals/assertions.py` + test inputs in
  `evals/test_cases.jsonl`, with score history in `results/scores.json`.
- IS NOT: a real LLM-judge (yet). `runner.py` currently scores the prompt
  document itself as a proxy — treat pass rate as a *structural* score, not a
  behavioral one. See "Upgrade path" below.

## Workspace layout (under `autoresearch-skills/`)

```
prompts/current/<skill>.md    # the copy being optimized
prompts/candidates/<id>.md    # generated variants under test
prompts/history/<skill>.<score>.md
evals/test_cases.jsonl        # {"skill": ..., "input": ...} rows
evals/assertions.py           # SKILL_ASSERTIONS per skill + "default" set
results/scores.json           # append-only pass-rate history
results/latest_run.json       # last run detail (analyze failures here)
```

## Procedure

1. **Seed.** Copy the target skill's `SKILL.md` to
   `prompts/current/<skill>.md` if absent or stale:
   `cp <skill>/SKILL.md autoresearch-skills/prompts/current/<skill>.md`
   If the skill has no assertions, add a `SKILL_ASSERTIONS["<skill>"]` block
   to `evals/assertions.py` (5–8 binary checks: required sections, code
   blocks, safety mentions, prohibited content) and 3+ rows to
   `test_cases.jsonl`.
2. **Baseline.** `python3 evals/runner.py --skill <skill>` — record pass rate.
3. **Fail-analyze.** Read `results/latest_run.json` `failed_assertions` +
   `results/scores.json` trend. Name the ONE failure pattern to attack.
4. **Variant.** `python3 evals/generate_variants.py --skill <skill> --count 3`
   — or hand-write ONE focused change addressing the failure pattern.
   One change per cycle. Never random churn.
5. **Re-score.** Evaluate each variant; keep the best ONLY if it strictly
   beats the baseline. `python3 evals/update_baseline.py --skill <skill>
   --variant <id>`; else discard and log why.
6. **Repeat** until: 90% pass rate, OR 5 consecutive iterations without
   improvement, OR manual stop.

## Corpus regression gate (unique to this corpus)

A prompt change that wins for skill X must not degrade sibling skills —
descriptions that bleed into neighbors' trigger space cause mis-activation.
After promoting a baseline, sanity-check the new description against the 3
most-adjacent skills (same prefix or shared triggers) by running their evals
once. If a sibling drops >5 points, revert or narrow the description.

## Claims hygiene

If the improved skill makes deployment/state assertions, its prompt should
retain the proveryay `<claims>` contract (see `proveryay/CLAIMS-TEMPLATE.md`).
Do not optimize a skill by deleting its verification surface to win points.

## APFS branching

For live skill dirs, COW-clone before hand-editing variants:
`cp -cR <skill>/ <skill>.gym-<n>/` — zero-cost scratch copies on macOS.

## Commands

```bash
cd autoresearch-skills
python3 evals/runner.py --skill <skill>
python3 evals/generate_variants.py --skill <skill> --count 3
python3 evals/update_baseline.py --skill <skill> --variant <variant_id>
python3 test_system.py                     # harness self-check
```

## Priority order (from `skills_to_improve.md`, 36 targets)

a-team, ac-ideate, ac-roadmap, agent-skills, apfs, at-tui, bop-brainstorm,
bop-on, bop-team, bop-workflow, council, fleet-irc, gantt,
irclaw-research-team, jj, jobcard-system, leptos, preflight, providers,
quota-gate, redact, searxng, tiger-team, toyota-way — then any skill below
75% in `scores.json`, then new skills with no baseline.

## Upgrade path (known limitation → real evals)

1. `runner.py`: replace the `response = prompt` stub with a real model call
   (Ollama `kimi-k2.6:cloud` is local and free) so assertions score actual
   skill *outputs*, not the prompt text.
2. Add trigger-accuracy assertions: does the `description:` activate on the
   right inputs and NOT on sibling-skill inputs.
3. Emit results in `schemas/bench.v1.schema.json` so runs register in the
   cross-project benchmark contract.
