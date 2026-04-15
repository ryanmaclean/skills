---
name: vibekanban
description: Use when working on the visual card UI, Quick Look previews, Zellij pane display, card glyphs, or the overall kanban UX of the jobcard system.
---

# Vibekanban

## What it is

Vibekanban is the **visual UX layer** on top of the jobcard system — not a
separate tool. It emerges from three components working together:

1. **Quick Look** — `.jobcard` bundles render as dark playing-card UI in Finder
2. **Zellij** — 7-pane pipeline layout + pane-per-card focus navigation
3. **`bop` CLI** — control plane (`inspect`, `poker`, `status`, `logs`)

## Quick Look Extension

`macos/bop/` — Swift/SwiftUI QL extension. Key file: `PreviewViewController.swift`.

### QL Sandbox Lessons (hard-won)
- Extension MUST be sandboxed (`com.apple.security.app-sandbox = true`)
- `MACH_O_TYPE = mh_execute` (NOT `mh_bundle`)
- `ENABLE_DEBUG_DYLIB = NO`
- `GENERATE_INFOPLIST_FILE = YES` auto-qualifies class name
- Sandboxed QL CANNOT shell out — use `changes.json` for file listing
- `NSFileCoordinator` fallback for reading files in sandbox
- Register: `lsregister -f ~/Applications/JobCardHost.app && pluginkit -a`

### Card Anatomy (top→bottom)
```
[glyph 44pt] [title bold] [P2 badge]
[label pills — colored by kind]
[progress bar]
[subtask dots ●○○○]
──────────────────────
[Spec]──[Plan]──[Code]──[QA]   ← stage pipeline
[acceptance criteria]
[clock 22h ago]   [⬛ Attach]  [● Stop]
                   ^running+session  ^running
```

### Attach Button (green)
Appears when `stages[current_stage].status == "running"` AND `zellij_session` is set.
Opens `bop://card/<id>/session` → host app routes to `zellij attach <session>`.

### Stop Button (coral)
Appears when `stages[current_stage].status == "running"`.

### URL Scheme (bop://)
Host app (`macos/JobCardHost/`) registers `bop://` URL scheme in Info.plist.

| URL | Action |
|-----|--------|
| `bop://card/<id>/session` | `zellij attach bop-<id>` |
| `bop://card/<id>/logs` | Live tail of card logs in terminal |

Routed in `JobCardHostApp.swift` via `.onOpenURL`.

### Logs Tab
QL preview includes a Logs tab with live-tail link. Clicking opens terminal
with colorized log output via `bop logs <id>`.

## Glyph System

See `unicode-glyphs` skill for full reference.

```
suit  = team       ♠=CLI  ♥=Arch  ♦=Quality  ♣=Platform
rank  = priority   Ace=P1  Queen=P2  Jack=P3  5=P4
🃏    = wildcard / needs breakdown (joker — U+1F0CF)
🂿    = RED JOKER — DO NOT USE, renders as ◆ (silent misread)
```

Glyph is set at creation or via `bop poker consensus`. Never change mid-flight.

### Card Back / Reveal Convention

- `🂠` (`U+1F0A0`) in filename or UI means unrevealed estimate/card-back state.
- Consensus reveal flips filename prefix to the chosen face card (example `🂻-feat-auth.jobcard`).
- Keep `meta.id` stable (`feat-auth`) while filename glyph changes.

### Contrast Rule (Playing Card Fonts)

- Playing-card glyphs visually assume a dark backdrop at larger sizes.
- Keep QL card canvas dark; for light host surfaces, wrap glyph in a dark chip/panel.
- There is no reliable "reverse font" Unicode variant for card backs/faces; do inversion at the view layer (background/foreground swap), not by changing codepoint.

## Zellij Integration

### 7-Pane Layout (`layouts/bop.kdl`)

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

Launch: `zellij --layout layouts/bop.kdl`

### bop_focus.zsh — Card Navigator

Navigate all Zellij panes to focus on a specific card:

```zsh
scripts/bop_focus.zsh <card-id>       # populate all panes for card
scripts/bop_focus.zsh --auto          # sweep all panes via zellij action write-chars
```

Uses `zellij action write-chars` to send commands into named panes.

### bop_bop.zsh — Session Bootstrap

Goal→card→Zellij session in one command:

```zsh
scripts/bop_bop.zsh "add auth to the API"
# Creates card: bop-add-auth-to-the-api
# Writes goal to spec.md
# Starts/attaches Zellij session: bop-bop-add-auth-to-the-api
# Runs dispatcher inside session
```

Budget-aware: reads `stage_budgets` from template meta.json.

### Meta Fields for Zellij

```json
{
  "zellij_session": "bop-feat-auth",  // session name
  "zellij_pane": "3"                  // pane ID within session
}
```

Set by dispatcher on spawn. Used by QL Attach button and bop_focus.

### Display Tiers (by running card count)
- 0–5 cards: per-card pane with glyph in title
- 6–20: team-level pane
- 21+: status bar only

## Log Colorization

`bop logs <id>` outputs tailspin-style colorized logs:
- Timestamps → dim
- ERROR/WARN → red/yellow
- File paths → cyan
- Numbers → magenta

Implemented in `colorize_log_line()` in `main.rs`. Detects TTY for auto-disable.

## Label Pill Colors

| kind | color |
|------|-------|
| domain | purple |
| effort | gold |
| scope | teal |
| (other) | purple |

## Thumbnail Pipeline (macOS)

- Preview image: `<card>.jobcard/QuickLook/Thumbnail.png`
- Renderer: `scripts/render_card_thumbnail.swift <meta.json> <out.png>`
- Dispatcher/merge-gate refresh on state transitions
- Also refresh after poker-driven filename glyph rename (`🂠` → face card)
- Batch: `scripts/macos_cards_maintenance.zsh [--compress]`

## Finder Icon Coloring

Cards show their stage as icon color in Finder:

| State | Color |
|-------|-------|
| pending | blue |
| running | orange |
| done | green |
| failed | red |

Driven by `scripts/set_card_icon.swift`. The `bop icons` FSEvents watcher auto-triggers icon updates on state transitions. Bundled into the `bop factory` lifecycle.

## changes.json (merge-gate artifact)

Written at merge time. QL reads this instead of shelling out to git.

```json
{
  "branch": "job/feat-auth",
  "files_changed": [
    {"path": "src/auth.rs", "status": "added"},
    {"path": "src/main.rs", "status": "modified"}
  ],
  "stats": {"files_changed": 2, "insertions": 145, "deletions": 12}
}
```

## Stage Display Names

Never use `.capitalized` — "QA" → "Qa". Use lookup:
```swift
let stageOrder = [("spec","Spec"),("plan","Plan"),("implement","Code"),("qa","QA")]
```

## Render Test (no QL daemon)

```bash
swift scripts/render_card_thumbnail.swift <meta.json> <out.png>
```
