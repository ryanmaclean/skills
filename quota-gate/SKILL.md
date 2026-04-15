---
name: quota-gate
description: >
  Pre-delegation quota check. Run before dispatching Auto-Claude specs or bop
  cards. Shows all provider headroom, recommends the best provider, warns at
  >80%, and blocks dispatch suggestions at >90%. Use when about to delegate a
  task, start a wave, or queue specs. Trigger phrases: "can we dispatch",
  "quota check", "headroom", "before we delegate", "what has capacity",
  "which provider", "are we ok to run".
---

# quota-gate — Pre-Delegation Quota Check

Run this before dispatching Auto-Claude specs, bop dispatcher waves, or any
multi-agent work session. It reads live quota from `bop providers` and gives
a go/warn/block recommendation.

## Usage

```bash
nu ~/.claude/skills/quota-gate/gate.nu
nu ~/.claude/skills/quota-gate/gate.nu --warn-at 70   # tighter warning threshold
nu ~/.claude/skills/quota-gate/gate.nu --block-at 85  # tighter block threshold
```

Or invoke as a skill — I will run it and interpret the output for you.

## What it shows

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  QUOTA GATE — pre-delegation check
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  Claude Code   5h  ████████░░  82%  ⚠  reset in 1h 12m
                7d  █████░░░░░  53%  ✓

  Codex CLI     session  ██░░░░░░░░  21%  ✓
                weekly   ███░░░░░░░  34%  ✓

  Gemini CLI    Pro   █████░░░░░  48%  ✓

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  RECOMMENDATION
  ⚠  Claude 5h window at 82% — consider Codex for next wave
  ✓  Codex has 79% headroom — good for 3–5 subtask specs
  ✓  Gemini has 52% headroom — available for parallel specs
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  GO — dispatch with caution (Claude 5h near limit)
```

## Thresholds

| Level | Usage | Symbol | Meaning |
|-------|-------|--------|---------|
| OK    | <70%  | ✓ green | Full headroom, dispatch freely |
| WARN  | 70–89% | ⚠ amber | Caution — prefer alternative provider |
| BLOCK | ≥90%  | ✗ red  | Do not dispatch to this provider |

Overall verdict:
- **GO** — at least one provider has <70% on primary window
- **CAUTION** — all providers 70–89%; dispatch small batches only
- **HOLD** — all providers ≥90%; wait for reset before dispatching

## Integration with dispatch.nu

Add to the top of any dispatch wave:

```nushell
# Check quota before launching wave
nu ~/.claude/skills/quota-gate/gate.nu
if $env.LAST_EXIT_CODE != 0 {
    print "⚠ quota gate blocked dispatch — check providers"
    exit 1
}
```

## As a pre-delegation reflex

Before saying "dispatch spec 035" or "run the next wave", I will automatically
invoke this gate and report headroom. If Claude 5h is >80%, I will recommend
routing to Codex or Gemini instead.
