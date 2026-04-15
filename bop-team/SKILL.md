---
name: bop-team
description: Dogfood orchestrator for working on the bop repo itself. Use when working on the bop jobcard system — create cards before coding, execute through dispatcher flow, and verify system output.
---

# bop-team: Dogfood Orchestrator

Use when working on the bop repo itself. Before writing code, deal a card.
After finishing, verify the system ran it.

## The Rule

Every unit of work on bop becomes a bop card. No exceptions.

## Agent Roster (budget order, cheapest first)

| Agent | Binary | Cost | Best for |
|-------|--------|------|----------|
| ollama | `ollama` | free | fast local, drafts, docs |
| opencode | `opencode` | $ | code tasks, mid-complexity |
| codex | `codex` | $ | code tasks, OpenAI models |
| claude | `claude` | $$ | complex reasoning, architecture |

Check availability:
```zsh
for a in ollama opencode codex claude; do which $a 2>/dev/null && echo "  $a: available"; done
```

## Templates

Four shipping templates in `.cards/templates/`:

| Template | Stages | Use When |
|----------|--------|----------|
| **implement** | implement → qa | Clear requirements |
| **full** | spec → plan → implement → qa | Complex/ambiguous |
| **cheap** | implement (local) | Small fix, local model |
| **ideation** | ideation → qa | Brainstorming (`bop bstorm`) |
| **qa-only** | qa | Code review, audit |

## Factory (launchd lifecycle)

```zsh
bop factory install    # generate + load plists (includes icon watcher)
bop factory status     # check if services are running
bop factory start      # start dispatcher + merge-gate + icons
bop factory stop       # stop all services
bop factory uninstall  # unload + remove plists
```

## Workflow

```zsh
# 1a. Deal an implement card
bop new implement <id>

# 1b. Or brainstorm first
bop bstorm <topic words>

# 2. Note the card ID and session name printed
# 3. Work inside the zellij session (or your current session)
# 4. Write output to output/result.md
# 5. Verify it ran
make check
./target/debug/bop status

# 6. The card auto-moves to done/ when dispatcher exits 0
```

## Installation (Quick Look)

```zsh
# Build in Xcode → Product → Build
# Then register:
LSREG=/System/Library/Frameworks/CoreServices.framework/Versions/A/Frameworks/LaunchServices.framework/Versions/A/Support/lsregister
$LSREG -f ~/Applications/JobCardHost.app

# Preview a card
qlmanage -p examples/running-feat.jobcard
```

## 7-Pane Zellij Layout

```
┌────────────────┬────────────────┐
│  bop status    │  spec.md       │
│  (board)       │  (current card)│
├────────────────┼────────────────┤
│  dispatcher    │  agent stdout  │
│  log --follow  │                │
├────────────────┼────────────────┤
│  merge-gate    │  agent stderr  │
│  log --follow  │                │
├────────────────┴────────────────┤
│  shell  (bop commands live here)│
└─────────────────────────────────┘
```

Use `layouts/bop.kdl` to launch this layout: `zellij --layout layouts/bop.kdl`

## License Compliance

All adapters in this repo use MIT/Apache-2.0-licensed tools only:
- claude: Anthropic TOS (API usage)
- codex CLI: MIT (github.com/openai/codex)
- ollama: MIT (github.com/ollama/ollama)
- opencode: MIT (github.com/sst/opencode)

## Zellij Session Protocol

Sessions are named `bop-<card-id>`. Always resumable:
```zsh
zellij attach "bop-<id>" 2>/dev/null || zellij -s "bop-<id>"
```

The `zellij_session` field in meta.json is the source of truth.
Quick Look renders a live "Attach" button (green) when the field is present and stage is running.

## Troubleshooting

**Nested Claude sessions:** When dispatching from inside Claude Code, `CLAUDECODE=1` blocks spawning. The adapter already runs `unset CLAUDECODE`, but if launching dispatcher manually use `env -u CLAUDECODE bop dispatcher --once`.

**Stale worktrees:** If a card fails, its git worktree persists and blocks re-dispatch. Clean with:
```zsh
git worktree remove --force .worktrees/job-<card-id>
git branch -D job-<card-id>
```

**Icon\r (macOS):** Finder drops `Icon\r` files in directories with custom icons. The dispatcher and `bop status` filter by `.jobcard` extension to ignore these.

**Provider cooldowns:** When a provider hits rate limits (exit 75), the dispatcher rotates to the next provider in `provider_chain`. If ALL providers are rate-limited, cards requeue until cooldowns expire (default 300s).

## Dogfood Check

Before marking any bop work complete, verify:
- [ ] A card exists for the work (`bop status`)
- [ ] `make check` passes
- [ ] The feature you built actually processed at least one card
