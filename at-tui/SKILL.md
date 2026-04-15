---
name: at-tui
description: "Auto-Tundra TUI agent interface. Use when you need to inspect project state, navigate tabs, query beads/agents/sessions, or automate TUI interactions from an agent context. The TUI can run headless for programmatic access."
author: $USER
version: 1.0.0
---

# Auto-Tundra TUI — Agent Interface Guide

The `at-tui` crate provides a ratatui terminal dashboard that agents can drive programmatically. Use it to inspect auto-tundra project state, navigate beads/agents/sessions, and automate workflows.

## Quick Start

```bash
# Interactive mode with demo data
cargo run -p at-tui -- --offline

# Interactive mode connected to running daemon
cargo run -p at-tui -- --api http://localhost:9090

# Headless mode (agent automation — JSON in/out, no rendering)
echo '{"cmd":"query_state"}' | cargo run -p at-tui -- --headless --offline

# Headless with live API
printf '{"cmd":"tab","args":[2]}\n{"cmd":"query_tab"}\n' | cargo run -p at-tui -- --headless
```

## Architecture

```
Agent (stdin JSON) ──┐
Human (keyboard)  ───┤──► AppCommand dispatcher ──► App state ──► Renderer (or JSON stdout)
API (at-bridge)   ───┘                                              ↑ skipped in --headless
```

Three input modes:
1. **Keyboard** — vim-style keys (human)
2. **`:` command mode** — type `:tab 3` in the status bar
3. **`--headless` stdin pipe** — send JSON commands, receive JSON responses

## CLI Flags

| Flag | Description |
|------|-------------|
| `--offline` | Use demo data, don't connect to at-bridge |
| `--api <url>` | API base URL (default: `http://localhost:9090`) |
| `--headless` | No terminal rendering; JSON stdin/stdout only |

## 17 Tabs

| # | Key | Name | Data |
|---|-----|------|------|
| 0 | `1` | Dashboard | KPI cards, agent summary, activity feed |
| 1 | `2` | Agents | Agent table (name, role, CLI, status) |
| 2 | `3` | Beads | 5-column kanban (Backlog→Done) |
| 3 | `4` | Sessions | Terminal sessions table |
| 4 | `5` | Convoys | Progress gauges per convoy |
| 5 | `6` | Costs | Token usage & cost breakdown |
| 6 | `7` | Analytics | Analytics dashboard |
| 7 | `8` | Config | Config file viewer |
| 8 | `9` | MCP | MCP server status |
| 9 | `0` / `R` | Roadmap | Priority-sorted feature list |
| 10 | `I` | Ideation | Split pane: ideas + detail |
| 11 | `W` | Worktrees | Git worktree status |
| 12 | `G` | GitHub Issues | Split pane: issues + detail |
| 13 | `P` | GitHub PRs | Split pane: PRs + detail |
| 14 | `S` | Stacks | Stacked diffs tree view |
| 15 | `X` | Context | Context steering levels + memory |
| 16 | `L` | Changelog | Collapsible version history |

## Headless JSON Protocol

### Sending Commands

Two formats accepted (both work in headless):

**JSON format** (recommended for agents):
```json
{"cmd":"tab","args":[3]}
{"cmd":"query_state"}
{"cmd":"select","args":[2]}
{"cmd":"query_selected"}
{"cmd":"query_tab"}
{"cmd":"quit"}
```

**Text format** (also works):
```
:tab 3
:query state
:select 2
:query selected
:quit
```

### Command Reference

| Command | JSON | Text | Description |
|---------|------|------|-------------|
| Switch tab | `{"cmd":"tab","args":[N]}` | `:tab N` | Go to tab N (0-16) |
| Next tab | `{"cmd":"next_tab"}` | `:next` | Cycle forward |
| Prev tab | `{"cmd":"prev_tab"}` | `:prev` | Cycle backward |
| Select item | `{"cmd":"select","args":[N]}` | `:select N` | Select item N in list |
| Move up | `{"cmd":"up"}` | `:up` | Previous item |
| Move down | `{"cmd":"down"}` | `:down` | Next item |
| Move left | `{"cmd":"left"}` | `:left` | Kanban/sub-tab left |
| Move right | `{"cmd":"right"}` | `:right` | Kanban/sub-tab right |
| Refresh | `{"cmd":"refresh"}` | `:refresh` | Reload config |
| Query state | `{"cmd":"query_state"}` | `:query state` | Full state JSON |
| Query tab | `{"cmd":"query_tab"}` | `:query tab` | Current tab data JSON |
| Query selected | `{"cmd":"query_selected"}` | `:query selected` | Selected item JSON |
| Quit | `{"cmd":"quit"}` | `:quit` | Exit TUI |

### Response Format

**Events** (emitted automatically):
```json
{"event":"started","tabs":17,"offline":true}
{"event":"tab_changed","tab":3,"tab_name":"Sessions"}
{"event":"data_refreshed","agents":5,"beads":12}
{"event":"ok"}
{"event":"quit"}
{"event":"error","message":"unknown command: foo"}
```

**Query responses** (raw JSON, not wrapped in event):

`query_state`:
```json
{
  "current_tab": 0,
  "tab_name": "Dashboard",
  "selected_index": 0,
  "api_connected": false,
  "offline": true,
  "counts": {
    "agents": 5, "beads": 10, "sessions": 4, "convoys": 3,
    "costs": 4, "mcp_servers": 4, "worktrees": 3,
    "github_issues": 3, "github_prs": 2, "roadmap_items": 4,
    "ideas": 3, "stacks": 3, "changelog": 2, "memory_entries": 2
  }
}
```

`query_tab` (example for Sessions tab):
```json
[
  {"id":"sess-01","agent":"mayor-alpha","cli_type":"Claude","status":"running","duration":"12m 34s","cpu":"2.1%"},
  {"id":"sess-02","agent":"deacon-bravo","cli_type":"Claude","status":"running","duration":"8m 12s","cpu":"1.4%"}
]
```

`query_selected` (example for a bead):
```json
{"id":"bd-002","title":"Implement auth module","status":"Slung","lane":"Critical"}
```

## Agent Workflow Examples

### Inspect all beads
```bash
printf '{"cmd":"tab","args":[2]}\n{"cmd":"query_tab"}\n{"cmd":"quit"}\n' \
  | cargo run -p at-tui -- --headless --offline 2>/dev/null \
  | grep '^\[' | jq .
```

### Check which agents are active
```bash
printf '{"cmd":"tab","args":[1]}\n{"cmd":"query_tab"}\n{"cmd":"quit"}\n' \
  | cargo run -p at-tui -- --headless 2>/dev/null \
  | grep '^\[' | jq '[.[] | select(.status == "Active")]'
```

### Get GitHub issues
```bash
printf '{"cmd":"tab","args":[12]}\n{"cmd":"query_tab"}\n{"cmd":"quit"}\n' \
  | cargo run -p at-tui -- --headless 2>/dev/null \
  | grep '^\[' | jq '[.[] | select(.state == "open")]'
```

### Multi-step inspection
```bash
cat <<'EOF' | cargo run -p at-tui -- --headless --offline 2>/dev/null
{"cmd":"query_state"}
{"cmd":"tab","args":[14]}
{"cmd":"query_tab"}
{"cmd":"tab","args":[2]}
{"cmd":"select","args":[1]}
{"cmd":"query_selected"}
{"cmd":"quit"}
EOF
```

## Key Files

| File | Purpose |
|------|---------|
| `crates/at-tui/src/main.rs` | Entry point, CLI args, event loop, headless mode |
| `crates/at-tui/src/app.rs` | App state, 17 tab data, key handler, `apply_data()` |
| `crates/at-tui/src/command.rs` | `AppCommand` enum, text/JSON parsers, executor |
| `crates/at-tui/src/api_client.rs` | Blocking HTTP client for at-bridge REST API |
| `crates/at-tui/src/ui.rs` | Master renderer, tab bar with badges, command bar |
| `crates/at-tui/src/tabs/*.rs` | Individual tab renderers (17 files) |
| `crates/at-tui/src/effects.rs` | tachyonfx visual effects (fade, sweep, glow, burst) |
| `crates/at-tui/src/widgets/` | toast.rs, gauge_bar.rs, status_bar.rs, help_modal.rs |

## Keyboard Shortcuts (Interactive Mode)

| Key | Action |
|-----|--------|
| `1`-`9` | Switch to tab 1-9 |
| `0` | Switch to tab 10 (Roadmap) |
| `R/I/W/G/P/S/X/L` | Jump to Roadmap/Ideation/Worktrees/Issues/PRs/Stacks/Context/Changelog |
| `Tab` / `Shift-Tab` | Cycle tabs forward/backward |
| `j`/`k` or `↓`/`↑` | Navigate list items |
| `h`/`l` or `←`/`→` | Kanban columns (Beads) / sub-tabs (Context) |
| `Enter` | Toggle expand/collapse (Changelog) |
| `:` | Enter command mode |
| `?` | Help modal |
| `r` | Refresh config |
| `q` | Quit |

## API Data Refresh

When not in `--offline` mode, a background thread polls the at-bridge API every 5 seconds:

- Endpoints: `/api/agents`, `/api/beads`, `/api/kpi`, `/api/sessions`, `/api/convoys`, `/api/costs`, `/api/mcp/servers`, `/api/worktrees`, `/api/github/issues`, `/api/github/prs`, `/api/roadmap`, `/api/ideation/ideas`, `/api/stacks`, `/api/changelog`, `/api/memory`
- Data flows via `flume` channel from refresh thread → main thread
- Status bar shows: `LIVE` (connected), `OFFLINE` (demo data), `...` (connecting)

## Development Notes

- Binary crate (`[[bin]]`), no `[lib]` — tests use `#[path = "..."]` includes
- Build: `cargo build -p at-tui`
- Test: `cargo test -p at-tui` (114 tests)
- `App::new(offline: bool)` — pass `true` for demo data
- Effects: `EffectManager` wraps tachyonfx, called in `ui::render()` after widgets
- Toast: `app.toasts.push(Toast::new("message", ToastLevel::Success))`
