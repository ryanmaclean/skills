---
name: ac-ideate
description: Run Auto-Claude AI-powered ideation for any project directory. Generates low-hanging fruit, high-value features, and UI/UX ideas using the Auto-Claude backend. Use when brainstorming new specs, features, or architectural directions for a project.
---

# ac-ideate — Auto-Claude Ideation Runner

Runs the Auto-Claude ideation pipeline against a project directory and writes
spec ideas to `<project>/.auto-claude/ideation/`.

## Binary location

```
/Applications/Auto-Claude.app/Contents/Resources/backend/runners/ideation_runner.py
```

Always set `PYTHONPATH` to the backend root:

```
PYTHONPATH=/Applications/Auto-Claude.app/Contents/Resources/backend
```

## Basic usage

```bash
# Ideate on current directory
PYTHONPATH=/Applications/Auto-Claude.app/Contents/Resources/backend \
  python3 /Applications/Auto-Claude.app/Contents/Resources/backend/runners/ideation_runner.py \
  --project /path/to/project

# With options
PYTHONPATH=/Applications/Auto-Claude.app/Contents/Resources/backend \
  python3 /Applications/Auto-Claude.app/Contents/Resources/backend/runners/ideation_runner.py \
  --project /path/to/project \
  --types low_hanging_fruit,high_value_features \
  --thinking-level high \
  --max-ideas 8 \
  --refresh
```

## All flags

| Flag | Default | Description |
|------|---------|-------------|
| `--project PATH` | cwd | Project directory to analyze |
| `--output PATH` | `<project>/.auto-claude/ideation` | Where to write results |
| `--types LIST` | all | Comma-separated: `code_improvements`, `ui_ux_improvements`, `documentation_gaps`, `security_hardening`, `performance_optimizations`, `code_quality` |
| `--thinking-level` | `medium` | `low` / `medium` / `high` |
| `--max-ideas N` | 5 | Max ideas per type |
| `--model` | `sonnet` | `haiku`, `sonnet`, `opus`, or full model ID |
| `--refresh` | false | Force regenerate even if results exist |
| `--append` | false | Add to existing session instead of replacing |
| `--no-roadmap` | false | Skip roadmap context |
| `--no-kanban` | false | Skip kanban context |
| `--fast-mode` | false | Faster Opus 4.6 output |

## Open a Zellij tab for ideation

```bash
# New tab — runs ideation then drops to shell
zellij --session $ZELLIJ_SESSION_NAME action new-tab --name "ideation" \
  -- sh -c 'cd /path/to/project && \
    PYTHONPATH=/Applications/Auto-Claude.app/Contents/Resources/backend \
    python3 /Applications/Auto-Claude.app/Contents/Resources/backend/runners/ideation_runner.py \
    --project . --thinking-level high --max-ideas 8 --refresh; exec $SHELL'
```

`$ZELLIJ_SESSION_NAME` is set automatically inside any Zellij session. If running outside, replace with the session name from `zellij list-sessions`.

## Typical workflow

1. **New project** — run with `--types low_hanging_fruit,high_value_features --refresh`
2. **Read results** — `ls <project>/.auto-claude/ideation/`
3. **Promote to specs** — copy ideas into `.auto-claude/specs/<NNN>-<slug>/spec.md`
4. **Dispatch** — `./dispatch.nu run --wave N --yes` or via Auto-Claude UI

## Projects

| Project | Path |
|---------|------|
| dub (EFI bootloader) | `$HOME/efi` |
| bop (jobcard system) | `$HOME/bop` |
| zam (unikernel VM reader) | `$HOME/zam` |

## Notes

- Ideation reads `PLAN.md`, existing specs, and `CLAUDE.md` for context automatically.
- `--refresh` is safe to run repeatedly — it overwrites the previous session.
- Does **not** require an API key — uses Auto-Claude's OAuth session (Claude Code subscription).
- Inside an agent session, `CLAUDECODE` env var is already set; no need to unset it for this runner.
