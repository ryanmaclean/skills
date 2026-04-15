---
name: zellij
description: Session, layout, and pane management for Zellij terminal multiplexer. Use when managing Zellij sessions, layouts, or automating terminal workflows.
---

# Zellij — Session, Layout, and Pane Management

## Core rule: never hardcode the session name

Zellij injects `$ZELLIJ_SESSION_NAME` into every process inside a session.
Always read it — never write `"pmr"`, `"bop"`, or any literal name.

```nu
# Nushell — canonical zellij_session function used in all dispatch.nu files
def zellij_session [] {
  if "ZELLIJ_SESSION_NAME" in $env {
    $env.ZELLIJ_SESSION_NAME
  } else {
    # Running outside Zellij — auto-detect
    let sessions = (^zellij list-sessions --no-formatting 2>/dev/null
      | lines | where { |l| ($l | str trim) != "" })
    if ($sessions | length) == 1 {
      $sessions | first | str trim
    } else if ($sessions | length) > 1 {
      error make { msg: $"Multiple sessions active: ($sessions | str join ', '). Set ZELLIJ_SESSION_NAME." }
    } else {
      error make { msg: "No Zellij session found. Start one with: zellij --session <name>" }
    }
  }
}
```

```bash
# Bash equivalent
SESSION=${ZELLIJ_SESSION_NAME:-$(zellij list-sessions --no-formatting | head -1 | awk '{print $1}')}
```

## Sessions

```bash
# Start or resume (canonical pattern — no hardcoded name)
zellij attach --create bop        # bop project
zellij attach --create efi        # efi/dub project
zellij attach --create zam        # zam project

# List active sessions
zellij list-sessions
zellij list-sessions --no-formatting   # machine-readable, no ANSI

# Detach without killing
Ctrl+p d

# Kill a session
zellij kill-session <name>
```

## Layouts

Each project has a `.kdl` layout file:

| Project | Layout | Launch |
|---------|--------|--------|
| bop | `layouts/bop.kdl` | `zellij --session bop --layout layouts/bop.kdl` |
| efi | `layouts/efi.kdl` | `zellij --session efi --layout layouts/efi.kdl` |
| zam | `layouts/zam.kdl` | `zellij --session zam --layout layouts/zam.kdl` |

### bop.kdl structure

Two tabs: `bop` (primary) and `dispatch` (engine runners).

**bop tab** (7-pane card lifecycle layout):
```
┌──────────┬──────────────────┬──────────────────┐
│  BOARD   │  SPEC            │  QA              │
│  watch   │  what to build   │  did it work?    │
│  status  ├──────────────────┼──────────────────┤
│          │  STDOUT          │  STDERR          │
│          │  agent output    │  agent errors    │
├──────────┴──────────────────┴──────────────────┤
│  INSPECTOR — bop inspect <id>                  │
├─────────────────────────────────────────────────┤
│  SHELL — bop commands (focus/retry/approve)    │
└─────────────────────────────────────────────────┘
```

**dispatch tab**: dispatcher (75%) | merge-gate (75%) | dispatch-shell (25%)

### KDL quick reference

```kdl
layout {
    default_tab_template {
        children
        pane size=2 borderless=true {
            plugin location="zellij:status-bar"
        }
    }

    tab name="main" focus=true {
        pane split_direction="horizontal" {
            pane name="top" size="70%" { command "nu" }
            pane name="bottom" size="30%" { command "nu" }
        }
    }

    tab name="logs" {
        pane name="log-tail" {
            command "nu"
            args "-c" "tail -f /tmp/bop.log"
        }
    }
}
```

`split_direction="horizontal"` = children stacked top/bottom.
`split_direction="vertical"` = children side by side.

## Spawning panes from scripts

```bash
# Fire-and-forget — pane closes when command exits
zellij --session $ZELLIJ_SESSION_NAME run \
  --name my-card \
  --close-on-exit \
  -- sh -c "nu /path/to/adapter.nu ..."

# Inspectable — pane stays open after command exits (for debugging)
zellij --session $ZELLIJ_SESSION_NAME run \
  --name my-card \
  -- sh -c "nu /path/to/adapter.nu ..."

# Open a new tab
zellij --session $ZELLIJ_SESSION_NAME action new-tab --name "logs"

# Navigate to an existing tab by name
zellij --session $ZELLIJ_SESSION_NAME action go-to-tab-name "bop"
```

**Use `--close-on-exit` for**: dispatch worker panes (adapter runs).
**Omit `--close-on-exit` for**: interactive panes, debugging, long-lived agents.

## PATH gotcha — critical for dispatch workers

`nu` (Nushell) is **not** in PATH inside Zellij panes spawned via `zellij run`.
Always use the full path:

```bash
# WRONG — fails silently in detached pane context
nu dispatch.nu mark-done 022

# CORRECT
/opt/homebrew/bin/nu dispatch.nu mark-done 022
```

Same applies to `bop` binary: use `$ROOT/target/debug/bop` or add to PATH explicitly:
```bash
export PATH=$HOME/bop/target/debug:$PATH
```

## Suspending TUI to drop to shell (`!` key in bop ui)

**Do NOT use `Ctrl+O`** — that is Zellij's built-in Session mode key and will
be intercepted before reaching any app running inside a pane.

`bop ui` uses `!` (normal mode, no modifier) for the subshell drop, inspired
by vim's `:!` convention. Zellij does not intercept bare printable keys in
normal mode.

```rust
// Triggered by KeyCode::Char('!') in the AppEvent::Key handler
// Suspend TUI
crossterm::terminal::disable_raw_mode()?;
crossterm::execute!(stdout(), crossterm::terminal::LeaveAlternateScreen)?;

// Run shell in card's worktree
std::process::Command::new(std::env::var("SHELL").unwrap_or("sh".into()))
    .current_dir(&card_worktree)
    .status()?;

// Restore TUI
crossterm::execute!(stdout(), crossterm::terminal::EnterAlternateScreen)?;
crossterm::terminal::enable_raw_mode()?;
// trigger full redraw
```

This does NOT open a new Zellij pane — it suspends and restores within the
existing pane. Works correctly inside Zellij because alternate screen is
stacked (Zellij has its own alternate screen above it).

## Zellij keys already taken — do NOT use in TUI apps

These are intercepted by Zellij before reaching pane content:

| Key | Zellij action |
|-----|---------------|
| `Ctrl+O` | Session mode (session manager) |
| `Ctrl+P` | Pane mode |
| `Ctrl+T` | Tab mode |
| `Ctrl+N` | New pane |
| `Ctrl+H` | Move pane focus left |
| `Ctrl+B` | Scroll up |
| `Ctrl+F` | Scroll down |
| `Ctrl+G` | Unlock (in locked mode) |

**Safe to use in TUI apps**: bare printable keys (`!`, `e`, `n`, `r`, etc.),
`Alt+<key>` (Zellij uses Alt+arrow but not most Alt+letter combos),
function keys `F1`–`F12`.

## Key bindings (Zellij default — navigation)

| Key | Action |
|-----|--------|
| `Alt + arrow` | Move focus between panes |
| `Alt + n` | New pane |
| `Alt + [` / `]` | Previous / next tab |
| `Ctrl+p d` | Detach from session (session stays alive) |
| `Ctrl+p z` | Toggle pane fullscreen |
| `Ctrl+p x` | Close focused pane |
| `Ctrl+p r` | Rename focused pane |

## Integration with dispatch.nu (all three projects)

All three projects (bop, efi, zam) use the same `zellij_session` pattern:

```
dispatch.nu run --wave N --yes
  → zellij_session []          # reads $ZELLIJ_SESSION_NAME
  → zellij --session <name> run --name <spec-id> --close-on-exit -- sh -c <AC cmd>
  → pane runs Auto-Claude agent
  → on exit: mark-done / mark-failed called via /opt/homebrew/bin/nu
```

Lock file (`dispatch-lock.json`) records `session: (zellij_session)` so rescue
operations can target the correct session.

## Nested panes — bop ui card tabs (spec 029 future)

Each running card will optionally get a tab with nested panes:
- Top pane: adapter stdout (live)
- Bottom pane: log tail (`logs/stderr`)

Tab naming convention: `card-<id>` (e.g. `card-feat-auth`).
Created via `zellij action new-tab --name card-<id>`, then pane splits via
`zellij action new-pane --direction down`.

## Projects and session names

| Project | Canonical session | Layout |
|---------|-------------------|--------|
| bop | `bop` | `layouts/bop.kdl` |
| efi (dub) | `efi` | `layouts/efi.kdl` |
| zam | `zam` | `layouts/zam.kdl` |

These are current names — they WILL change. Always use `$ZELLIJ_SESSION_NAME`.

## Quick reference

```bash
# Get current session name (canonical method)
SESSION=${ZELLIJ_SESSION_NAME:-$(zellij list-sessions --no-formatting | head -1 | awk '{print $1}')}

# Start/attach to session
zellij attach --create <session-name>

# List sessions
zellij list-sessions --no-formatting

# Run command in new pane
zellij --session $SESSION run --name pane-name --close-on-exit -- command

# Detach without killing
Ctrl+p d

# Navigate between panes
Alt + arrow keys
```

## Common mistakes

- **Hardcoding session names**: Never write literal names like "bop" or "efi" — always read `$ZELLIJ_SESSION_NAME`
- **Using Ctrl+O in TUI apps**: Zellij intercepts this key for Session mode — use bare printable keys like `!` instead
- **Not using full paths to binaries**: `nu` and other tools are not in PATH inside spawned panes — use full paths like `/opt/homebrew/bin/nu`
- **Forgetting --close-on-exit**: Worker panes should close when done to avoid resource leaks
- **Using Ctrl+O for subshell drop**: Use bare `!` key instead, as Ctrl+O is intercepted by Zellij
