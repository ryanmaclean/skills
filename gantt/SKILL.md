---
name: gantt
description: Generate and view a Gantt timeline of bop card runs — ANSI terminal (default) or HTML. Use when discussing execution history, parallelism, throughput, or agent performance. Trigger words — gantt, timeline, parallelism, throughput, duration, how long, when did.
---

# bop gantt — agent execution timeline

Shows horizontal bars per card run, clustered by session, with state colours and duration labels. Default output is ANSI terminal art that fills the current pane.

## Quick use

```bash
bop gantt              # ANSI chart in terminal (auto-fits pane width)
bop gantt -w 80        # force 80-column width (useful in Zellij panes)
bop gantt --html       # write .cards/bop-gantt.html
bop gantt -o           # write HTML and open in browser
```

## Output modes

| Flag | Mode | Where |
|------|------|-------|
| (none) | ANSI ▓-blocks | stdout, fits current terminal/pane |
| `--html` | HTML file | `.cards/bop-gantt.html` |
| `-o` | HTML + open | browser |

## ANSI mode details

- Auto-detects terminal width via `ioctl(TIOCGWINSZ)` — works in Zellij panes, tmux, SSH
- Falls back to `$COLUMNS`, then 100
- `-w N` overrides auto-detection
- 2-char padding per side for breathing room
- State colours use ANSI 256-colour palette
- Bars use `▓` fill blocks with matched foreground+background colour

## What it shows

- **Horizontal bars** — one per card run, width ∝ wall-clock duration
- **Clusters** — runs separated by >30 min gap get their own time axis
- **State colours** — pending (blue), running (amber), done (green), failed (red), merged (violet)
- **Duration labels** — e.g. `2.7m`, `4.8m` next to each bar
- **Stats footer** — total runs, compute minutes, tokens, cost
- **Legend** — coloured dots with state names

## Data source

Reads `meta.json` from every `.jobcard/` in `.cards/{pending,running,done,failed,merged}/`. Uses `runs[].started_at` and `runs[].ended_at` for bar placement. Cards with no runs or <2s duration are excluded.

Open-ended runs (no `ended_at`) are capped at `duration_s` if available, otherwise 10 minutes.

## Complementary views

| View | Command | Shows |
|------|---------|-------|
| Gantt (terminal) | `bop gantt` | ANSI bars in current pane |
| Gantt (browser) | `bop gantt -o` | HTML with hover tooltips |
| Card list | `bop list --state all` | Current state of all cards |
| Events log | `bop events --limit 50` | OpenLineage event stream |
| Calendar | `open .cards/bop-agents.ics` | VEVENTs in Apple Calendar |
| Inspect | `bop inspect <id>` | Single card detail + cost |
