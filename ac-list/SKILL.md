---
name: ac-list
description: Check Auto-Claude kanban card status for any project without the GUI. Shows all specs with [OK] (complete), [--] (pending), [>>] (in-progress) status. Use when checking what specs are pending, complete, or running in any Auto-Claude project.
---

# ac-list — Auto-Claude Headless Kanban Status

Shows all specs and their kanban state for a project without opening the GUI.

## Command

```bash
PYTHONPATH=/Applications/Auto-Claude.app/Contents/Resources/python-site-packages \
  env -u CLAUDECODE \
  /Applications/Auto-Claude.app/Contents/Resources/python/bin/python3 \
  /Applications/Auto-Claude.app/Contents/Resources/backend/run.py \
  --list --project-dir /path/to/project
```

## Status symbols

| Symbol | Meaning |
|--------|---------|
| `[OK]` | Complete — all subtasks done |
| `[--]` | Pending / initialized — not started |
| `[>>]` | In progress — currently running |
| `[XX]` | Failed |

## Filter examples

```bash
# Count pending specs
... --list --project-dir $HOME/efi | grep "\[--\]" | wc -l

# Show only complete
... --list --project-dir $HOME/efi | grep "\[OK\]"

# Show only pending
... --list --project-dir $HOME/efi | grep "\[--\]"

# Show in-progress
... --list --project-dir $HOME/efi | grep "\[>>\]"
```

## Projects

| Project | Path |
|---------|------|
| dub (EFI bootloader) | `$HOME/efi` |
| bop (jobcard system) | `$HOME/bop` |
| zam (unikernel VM reader) | `$HOME/zam` |

## Notes

- `env -u CLAUDECODE` is required — strips the Claude Code env var so run.py uses OAuth, not an API key
- This reads directly from `.auto-claude/specs/*/` on disk — no GUI, no network
- Complements `./dispatch.nu status` which tracks wave-level completion; this shows Auto-Claude's own subtask-level view
- `[OK]` in ac-list + entry in `dispatch-state.json` completed = fully done
- `[OK]` in ac-list but NOT in `dispatch-state.json` = completed but `mark-done` never fired → run `./dispatch.nu mark-done NNN`
