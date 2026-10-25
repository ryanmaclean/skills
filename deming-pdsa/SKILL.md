---
name: deming-pdsa
description: Run Deming Plan-Do-Study-Act (PDSA/PDCA) cycles with explicit hypotheses, measurable gates, proof probes, and standard-work updates. Use when driving execution under uncertainty, recovering stalled build lanes, coordinating parallel agents, or reducing repeat failures.
---

# Deming PDSA

Use this skill when the user asks for `PDSA`, `PDCA`, `Deming`, `continuous improvement`, `stop-line`, `andon`, `standard work`, or when work is looping without measurable progress.

## Quick Start

1. Create a cycle artifact:
   - `nu scripts/new-pdsa-cycle.nu --title "<problem>" --owner "<owner>"`
2. Fill in `PLAN` first; do not execute before gates are defined.
3. Run one controlled `DO` experiment (single variable change).
4. Run `STUDY` with proof probes (`proveryay verify ...` or `proveryay batch`).
5. Complete `ACT`: adopt/adapt/abandon and update standard work.

## PDSA Contract

Every cycle must include these fields:

- `problem_statement`: concrete defect or bottleneck
- `hypothesis`: what change should improve and why
- `primary_metric`: one numeric metric
- `target`: numeric target and deadline
- `guardrails`: max acceptable regressions
- `owner`: single accountable owner
- `timebox`: explicit start/end window
- `claims`: verifiable assertions with shell probes
- `decision`: `ADOPT`, `ADAPT`, or `ABANDON`
- `standard_work_update`: exact procedure change

If any field is missing, the cycle is incomplete.

## Workflow

### PLAN

- Define baseline from observed facts (not assumptions).
- Pick the smallest experiment that can falsify the hypothesis.
- Define stop-line triggers before execution.
- Write claims in machine-verifiable form.

Output:
- `plans/pdsa/<cycle-id>.md` (plan section complete)

### DO

- Execute exactly the planned experiment.
- Keep one-change-at-a-time discipline.
- Record command, timestamp, and output artifact locations.

Output:
- command logs, build logs, artifact paths

### STUDY

- Compare baseline vs result for primary metric.
- Verify claims with `proveryay`.
- Mark each claim `PASS`, `FAIL`, or `INCONCLUSIVE`.

Required rule:
- No verbal completion without probe evidence.

Output:
- `study_summary` in cycle file with metric delta and proof snippets

### ACT

- `ADOPT`: target met and guardrails hold
- `ADAPT`: partial progress, adjust one variable
- `ABANDON`: no gain or guardrail breach

Then update standard work:
- append one concrete change to `plans/pdsa/STANDARD-WORK.md`

## Claim Format

Use this exact shape in cycle files and agent reports:

```yaml
<claims>
- claim: "L1 build is actively compiling"
  probe: "ssh studio@10.0.2.42 'wc -c /home/studio/bsd-build/logs/buildworld-l1.log'"
  expect: ""
- claim: "Artifact is bootable in QEMU"
  probe: "grep -E 'Booting|login:' /path/to/boot.log"
  expect: "Booting"
</claims>
```

## Stop-Line Rules

Stop execution immediately when any condition is true:

- repeated identical failure signature appears twice in window
- disk guardrail breached
- host unreachable for required lane
- proof probes fail for critical claim

On stop-line:
- open a new PDSA cycle with root-cause hypothesis
- do not retry blindly

## Multi-Agent Use

When using subagents:

- assign each agent one lane or one proof surface
- require `<claims>` block in every agent completion
- treat unverified claims as non-facts
- merge only after at least one independent verifier agent passes

## Output Standard

Return results in this order:

1. `PLAN` summary (hypothesis, metric, target)
2. `DO` actions (commands + artifacts)
3. `STUDY` evidence (probe verdicts)
4. `ACT` decision and standard-work update

Keep it concise and numeric.
