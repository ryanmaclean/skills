---
name: henshu
description: Use when a body of agent-generated work (specs, plans, code, translations, audit trails) has accumulated across multiple sessions and needs editorial curation — cross-checking, consistency, narrative distillation, and gap detection. Triggers include: spec sections that predate discoveries, translations not reviewed for naturalness, spool/audit trails that are audit logs but not readable stories, README that no longer matches project state, stale references after infrastructure changes.
---

# Henshu (編集) — Editorial Curation for Agent-Generated Work

## Overview

Henshu is the editorial pass that turns accumulated agent output into a coherent whole. Named for the Japanese concept of editing-as-compilation: shaping many pieces into something greater than their sum. It is distinct from code review (which judges correctness) and proofreading (which fixes surface errors) — henshu curates.

**Core principle:** Agent teams produce correct-in-isolation artifacts that drift from each other over time. Henshu finds and closes those gaps.

## When to Use

- Multiple agents have written to the same spec/plan across sessions
- Infrastructure changed mid-project (hostnames, ports, paths) and docs partially updated
- Translations were generated quickly and haven't been reviewed for naturalness
- An audit trail (spool, changelog) exists but can't be read as a story
- README or front-facing docs were written early and the project has moved
- You are approaching a demo, release, or handoff and the work needs to be presentable

**NOT for:**
- Real-time review of a single agent's output (use `code-review` skill)
- Spelling/grammar only (overkill — just fix inline)
- Active development where the work is still changing rapidly

## The Five Henshu Tasks

Run all five; they compound.

### 1. Reality Reconciliation
Cross-check every claim in specs/plans against the actual codebase and infrastructure state. Flag anything that was true when written but is no longer true.

Pattern: grep for proper nouns (hostnames, ports, paths, tool names), check each against ground truth.

```
grep -rn "gitea.local\|:3000\|fb-vm-24\|2225\|include MINIMAL" docs/ plans/ sys/
```

### 2. Narrative Distillation
An audit trail is not a document. Distill the spool/changelog into a `docs/PROJECT-LOG.md` that answers:
- What was the original goal?
- What was tried and what was learned?
- What decisions were made and why?
- What is the current state?

Target: someone cold-starting in 5 minutes can understand the project arc.

### 3. i18n Editorial Pass
Machine-speed translations are structurally correct but often unnatural. For each translated file:
- Read it as a native speaker would
- Fix calque constructions (word-for-word translations that don't idiomatically land)
- Verify technical term consistency (pick one translation per term and use it throughout)
- Check that code blocks, paths, and commands are untouched

### 4. Front Door Polish
The README is the first thing a conference attendee or cold agent sees. It should:
- Lead with the most interesting idea (not "this is a minimal FreeBSD image")
- Show the architecture in 3-5 bullet points
- Have working quick-start commands
- Link to the key documents

Run: does it accurately describe the current state? If not, rewrite it.

### 5. Orphan Detection
Identify artifacts that no longer connect to anything:
- Plans that describe work that was completed or abandoned
- Memory files that contradict current state
- Config files for infrastructure that changed
- Translation files for documents that were significantly rewritten

Flag for removal or update; don't silently delete without a commit message.

## Output Format

Henshu produces three artifacts:

1. **`docs/HENSHU-REPORT.md`** — what was found and fixed, what was flagged for human review, what was left alone and why. Structured as: FIXED / FLAGGED / DEFERRED.

2. **`docs/PROJECT-LOG.md`** (if one doesn't exist) — narrative distillation of the audit trail.

3. **Commits** — one commit per henshu task (reality reconciliation, narrative, i18n, README, orphans). Granular commits make it easy to revert a single task if the editor disagreed.

## Quick Reference

| Task | Input | Output | Time |
|------|-------|--------|------|
| Reality reconciliation | spec/plan files + grep | inline edits + FIXED list | fast |
| Narrative distillation | spool/changelog | PROJECT-LOG.md | medium |
| i18n editorial | *.fr.md, *.ja.md | edited translations | slow |
| Front door polish | README.md | rewritten README | fast |
| Orphan detection | all tracked files | FLAGGED list | fast |

## Common Mistakes

**Over-editing**: Henshu does not rewrite correct content for style. If a spec section is accurate and clear, leave it. Flag tone/style issues in the report but don't change them without user direction.

**Silent deletion**: Never delete an "orphaned" file without documenting why in the commit message. It may be referenced somewhere you didn't check.

**Translating code blocks**: Technical content in any language file — commands, paths, code, TOML — must remain identical to the English source. Henshu does not touch these.

**Conflating with implementation**: Henshu is editorial, not development. If it finds a genuine bug (not a documentation bug), it reports it in HENSHU-REPORT.md under FLAGGED and stops — it does not fix code.
