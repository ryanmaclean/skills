---
name: ac-roadmap
description: Run Auto-Claude AI-powered roadmap generation for any project directory. Analyzes project structure, target audience, and existing specs to produce a strategic feature roadmap. Use when planning milestones, phases, or strategic direction for a project.
---

# ac-roadmap — Auto-Claude Roadmap Runner

Runs the Auto-Claude roadmap pipeline against a project directory and writes
results to `<project>/.auto-claude/roadmap/`.

## Binary location

```
/Applications/Auto-Claude.app/Contents/Resources/backend/runners/roadmap_runner.py
```

Always set `PYTHONPATH` to the backend root:

```
PYTHONPATH=/Applications/Auto-Claude.app/Contents/Resources/backend
```

## Basic usage

```bash
# Generate roadmap for current directory
PYTHONPATH=/Applications/Auto-Claude.app/Contents/Resources/backend \
  python3 /Applications/Auto-Claude.app/Contents/Resources/backend/runners/roadmap_runner.py \
  --project /path/to/project

# With competitor analysis and high thinking
PYTHONPATH=/Applications/Auto-Claude.app/Contents/Resources/backend \
  python3 /Applications/Auto-Claude.app/Contents/Resources/backend/runners/roadmap_runner.py \
  --project /path/to/project \
  --thinking-level high \
  --competitor-analysis \
  --refresh
```

## All flags

| Flag | Default | Description |
|------|---------|-------------|
| `--project PATH` | cwd | Project directory to analyze |
| `--output PATH` | `<project>/.auto-claude/roadmap` | Where to write results |
| `--model` | `sonnet` | `haiku`, `sonnet`, `opus`, or full model ID |
| `--thinking-level` | `medium` | `low` / `medium` / `high` |
| `--refresh` | false | Force regenerate even if roadmap exists |
| `--competitor-analysis` | false | Enable competitor analysis phase |
| `--refresh-competitor-analysis` | false | Force refresh competitor analysis (requires `--competitor-analysis`) |

## Open a Zellij tab for roadmap generation

```bash
zellij --session $ZELLIJ_SESSION_NAME action new-tab --name "roadmap" \
  -- sh -c 'cd /path/to/project && \
    PYTHONPATH=/Applications/Auto-Claude.app/Contents/Resources/backend \
    python3 /Applications/Auto-Claude.app/Contents/Resources/backend/runners/roadmap_runner.py \
    --project . --thinking-level high --refresh; exec $SHELL'
```

`$ZELLIJ_SESSION_NAME` is set automatically inside any Zellij session. If running outside, replace with the session name from `zellij list-sessions`.

## Recommended sequence for a new project

Run in this order — each step feeds context into the next:

```bash
# 1. Roadmap first (strategic framing)
ac-roadmap --project . --thinking-level high --refresh

# 2. Ideation second (reads roadmap context automatically)
ac-ideate  --project . --thinking-level high --refresh

# 3. Promote ideas to specs
#    cp .auto-claude/ideation/<idea>.md .auto-claude/specs/<NNN>-<slug>/spec.md

# 4. Dispatch specs
#    ./dispatch.nu run --wave N --yes
```

## Projects

| Project | Path |
|---------|------|
| dub (EFI bootloader) | `$HOME/efi` |
| bop (jobcard system) | `$HOME/bop` |
| zam (unikernel VM reader) | `$HOME/zam` |

## Notes

- Reads `PLAN.md`, `CLAUDE.md`, and existing specs for context automatically.
- `--competitor-analysis` does a web search phase — useful for positioning but slower.
- Does **not** require an API key — uses Auto-Claude's OAuth session (Claude Code subscription).
- Run roadmap before ideation so ideation can incorporate strategic milestones as context.
