---
name: council
description: >
  Run N opposing expert agents on a problem, each with a distinct persona and
  adversarial mandate. Synthesize votes, dissents, and blockers into a verdict.
  Trigger phrases: "get multiple opinions", "debate this", "stress-test this",
  "devil's advocate", "three perspectives", "adversarial review", "council".
  Use when reviewing specs, architecture decisions, or design contradictions.
  Not for code review (use pr-review-toolkit) or merge conflicts.
---

# Council Skill

Runs a panel of N agents with opposing perspectives on a shared artifact
(specs, design docs, architecture decisions). Each agent has a persona, a
mandate, and permission to block. Returns a synthesized verdict.

## When to use

- Reviewing specs before dispatching implementation agents
- Evaluating architecture decisions with genuine tradeoffs
- Stress-testing a TRIZ resolution from multiple lenses
- Any decision where a single perspective is insufficient

## When NOT to use

- Code review (use pr-review-toolkit)
- Merge conflicts (semantic problem, not adversarial)
- Simple factual questions (overkill)

## Invocation

```
/council <artifact_path_or_description> [--n 3] [--personas <list>]
```

Default N=3. Default personas: Purist, Pragmatist, Adversary.

## Default personas

| Persona | Mandate | Blocks on |
|---------|---------|-----------|
| **Purist** | Methodology/theory correctness. Calls out wrong principle applications, missing foundations, conceptual drift. | Spec contradicts the methodology it's based on |
| **Pragmatist** | Shipping velocity, YAGNI, complexity cost. Cuts scope, flags over-engineering, demands simpler paths. | Spec adds complexity with no near-term payoff |
| **Adversary** | Finds the failure mode. Assumes everything will go wrong. Security, edge cases, missing error handling. | Spec has a critical unhandled failure path |

## Custom personas

Override with `--personas` for domain-specific councils:

```
/council specs/ --personas "Rust systems engineer, UX researcher, Security auditor"
```

## Output format

Each agent returns:
```
VERDICT: APPROVE | APPROVE WITH CONDITIONS | BLOCK
BLOCKER: <one-line reason if blocking>
CONDITIONS: <list of required changes if conditional>
OBSERVATIONS: <key findings>
VOTE: <principle or approach they'd recommend>
```

Synthesizer agent reads all N verdicts and produces:
```
COUNCIL VERDICT: APPROVED | APPROVED WITH CONDITIONS | BLOCKED
BLOCKING AGENTS: <list>
REQUIRED CHANGES: <merged deduplicated list>
DISSENTS: <disagreements that didn't block>
SYNTHESIS: <what the council agrees on>
```

## How to implement (for Claude Code)

1. Read the artifact(s)
2. Spawn N agents in parallel via Agent tool, each with persona prompt + artifact content
3. Collect all N verdicts
4. Run synthesizer pass on combined verdicts
5. Present council verdict to user

## Persona prompt template

```
You are [PERSONA_NAME], a council member reviewing the following artifact.

Your mandate: [MANDATE]

You are adversarial. Your job is not to rubber-stamp — it is to find problems,
challenge assumptions, and block if necessary.

VERDICT options:
- APPROVE: No significant issues.
- APPROVE WITH CONDITIONS: Issues exist but are fixable. List CONDITIONS.
- BLOCK: Critical issue. State BLOCKER clearly in one sentence.

Artifact:
---
[ARTIFACT_CONTENT]
---

Respond in this exact format:
VERDICT: <APPROVE|APPROVE WITH CONDITIONS|BLOCK>
BLOCKER: <one sentence, or "none">
CONDITIONS: <bullet list, or "none">
OBSERVATIONS: <your key findings, 3-5 bullets>
VOTE: <what you'd recommend instead or in addition>
```

## Notes

- Agents run in parallel (independent, not sequential deliberation)
- Synthesis is a separate pass — do not ask agents to read each other's verdicts
- A single BLOCK from any agent surfaces to the user as a council BLOCK
- The council is advisory — human has final say
- Log council verdicts to corpus when reviewing TRIZ resolutions:
  `triz log --council-verdict council-output.json`
