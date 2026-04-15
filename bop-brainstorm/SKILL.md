---
name: bop-brainstorm
description: Use when the user wants to brainstorm, ideate, or explore ideas for a topic. Creates an ideation card and optionally dispatches it to an agent. Trigger words — brainstorm, ideate, ideation, bstorm, explore ideas.
---

# bop brainstorm

Quick-start an ideation session. Three equivalent commands:

```bash
bop bstorm <topic words>        # canonical (short, unique)
bop brainstorm <topic words>    # alias
bop ideation <topic words>      # alias
```

No quoting needed — words are joined automatically.

## What it does

1. Slugifies the topic into a card ID (lowercase, hyphens, max 40 chars)
2. Creates an ideation card from `.cards/templates/ideation.jobcard/`
3. Writes the topic into `spec.md` as a brainstorm prompt
4. Assigns a playing-card glyph

## The ideation pipeline

```
ideation → qa
```

- **ideation** stage: agent explores ideas, trade-offs, approaches
- **qa** stage: second agent reviews and stress-tests the ideas

Provider defaults: `claude` for ideation, `codex` for QA.

## Examples

```bash
# Brainstorm a feature
bop bstorm plugin system for adapters

# Brainstorm architecture
bop ideation how to scale beyond 10k cards

# Brainstorm with team assignment
bop bstorm --team arch event-driven dispatcher
```

## After creating

```bash
# Edit the spec to add constraints or context
$EDITOR .cards/pending/<glyph>-<slug>.jobcard/spec.md

# Dispatch to an agent
bop dispatcher --once

# Or let factory handle it (if installed)
bop factory status
```

## Troubleshooting

**Nested dispatch:** When running `bop dispatcher` from inside Claude Code, unset the env var first:
```bash
env -u CLAUDECODE bop dispatcher --once
```

**Stale worktrees:** If a card failed previously, its worktree may block re-dispatch:
```bash
git worktree remove --force .worktrees/job-<card-id>
git branch -D job-<card-id>
```

**Rate limits:** If the ideation provider (claude) is rate-limited, the dispatcher rotates to the next provider in the chain. Check cooldowns with `bop status`.

## Output

The agent writes results to `output/result.md` inside the card.
Read it with:

```bash
bop inspect <card-id>
```
