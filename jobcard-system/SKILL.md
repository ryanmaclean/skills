---
name: jobcard-system
description: Use when working in the jobcard repository, dispatching agents, writing specs, creating cards, or understanding the card lifecycle. This is a multi-agent task orchestration system — NOT the transit data standard.
---

# Jobcard System (bop)

## Naming — COMPLETE (2026-03-01)

| Role | Name |
|------|------|
| CLI binary | **`bop`** |
| Quick Look bundle | `sh.bop.ql` |
| Host app bundle | `sh.bop` |
| Card UTI | `sh.bop.card` |
| launchd labels | `sh.bop.dispatcher`, `sh.bop.merge-gate` |

```bash
bop new implement feat-auth      # deal a card into pending/
bop status                       # card counts by state
bop dispatcher --once            # process one pending card
bop merge-gate --once            # process one done card
bop inspect <id>                 # show meta + spec + log tail
bop approve <id>                 # clear decision_required gate
bop kill <id>                    # terminate running agent
bop logs <id>                    # colorized log tail (tailspin-style)
bop poker open <id>              # start estimation round
bop poker submit <id> [--glyph 🂻] [--name alice]
bop poker reveal <id>            # reveal all estimates
bop poker consensus <id> --glyph 🂻  # lock in final estimate
bop factory install              # generate + load launchd plists
bop factory status               # check if services are running
bop factory start                # start both services
bop factory stop                 # stop both services
bop factory uninstall            # unload + remove plists
bop icons                        # FSEvents watcher: auto-updates Finder icons on state change
bop gantt                        # ANSI Gantt timeline in terminal (auto-fits pane)
bop gantt -w 80                  # force width (Zellij panes)
bop gantt --html                 # write .cards/bop-gantt.html
bop gantt -o                     # write HTML + open in browser
bop events                       # OpenLineage event stream
bop events --card <id>           # filter events by card
```

`bop factory install` also sets up the icon watcher alongside dispatcher and merge-gate.

## MICROCLAW Principle

**Fewer files. Sharper precision. Minimum viable factory.**

- Every file has exactly one job
- Agents can participate with just `mkdir` + `echo '{}' > meta.json`
- The FORMAT is the product; the CLI is one thin reference impl
- Like **Apache Iceberg** (table format, any engine reads/writes)

## Core Framing

The filesystem IS the state machine. Playing cards ARE the identity system.
The FORMAT is the product.

```sh
# Minimum viable card — no CLI needed
mkdir -p .cards/pending/my-task.jobcard
echo '{"id":"my-task","stage":"implement","created":"2026-03-01T00:00:00Z"}' \
  > .cards/pending/my-task.jobcard/meta.json
echo "# My Task\nDo X" > .cards/pending/my-task.jobcard/spec.md
```

## Architecture

```
.cards/
  stages/           ← stage instruction files (spec.md, plan.md, implement.md, qa.md)
  templates/        ← factory recipes (implement, full, cheap, qa-only)
  system_context.md ← prepended to every agent prompt
  <team>/<state>/<id>.jobcard/
    meta.json       ← card identity, stage pipeline, glyph, labels, subtasks
    spec.md         ← what to build
    prompt.md       ← rendered from template, sent to agent
    logs/           ← stdout.log, stderr.log, pid, lease.json
    output/         ← result.md, cards.yaml (child card seeds)
    QuickLook/      ← Thumbnail.png for Finder preview
```

## State Machine

```
pending/ → running/ → done/ → merged/
                    ↓
                 failed/
```

Transitions are atomic `fs::rename`. Dispatcher owns `pending/→running/`.
Merge-gate owns `done/→merged/` (runs acceptance_criteria first).

## Filename Glyph Lifecycle (Poker)

Use filename prefix for poker visibility while keeping `meta.id` canonical.

- Before consensus: `🂠-feat-auth.jobcard`
- After consensus: `🂻-feat-auth.jobcard`
- Keep `meta.id` stable as `feat-auth` (CLI queries stay stable)

Safe rename flow (filesystem-first, same volume):

1. Only rename outside `running/` (prefer `pending/`; allow `done/`/`failed/` for cleanup).
2. Hold card lock / ownership gate; assert destination path does not exist.
3. Apply atomic rename (`fs::rename`).
4. If poker consensus finalized a glyph, write `meta.glyph` in the same critical section.
5. Refresh visual artifacts (`QuickLook/Thumbnail.png` + Finder icon).

## Stage Pipeline (Factory Engine)

Templates define the stage pipeline via these `meta.json` fields:

| Field | Type | Example |
|-------|------|---------|
| `stage_chain` | `Vec<String>` | `["implement", "qa"]` |
| `stage_models` | `Map<String,String>` | `{"implement":"opus","qa":"sonnet"}` |
| `stage_providers` | `Map<String,String>` | `{"implement":"claude","qa":"codex"}` |
| `stage_budgets` | `Map<String,u64>` | `{"implement":32000,"qa":8000}` |

When a stage finishes (exit 0), dispatcher reads `stage_chain`, creates next-stage
child card in `pending/` inheriting spec + glyph + prior output. Final stage → done/.

All four fields are on the Meta struct (landed 2026-03-01). `render_prompt` substitutes
`{{stage_instructions}}`, `{{stage_index}}`, `{{stage_count}}`, `{{prior_stage_output}}`.
Dispatcher auto-advances stages via `maybe_advance_stage()`.

## Stage Instruction Files

`.cards/stages/` — one markdown file per stage, ~50 tokens each:

| File | Agent's job |
|------|------------|
| `spec.md` | Write a spec under 500 words. No code. |
| `plan.md` | Read spec, produce ordered steps under 800 words. |
| `implement.md` | Write code, run tests, exit 0 when green. |
| `qa.md` | Review as a different agent. Be skeptical. |

Custom stages: create `.cards/stages/<name>.md`, add to template's `stage_chain`.

## Context Window Architecture

Every token earns its slot. Dispatcher builds prompts by layering:

```
system_context.md        ~150 tokens   domain fix
stages/<stage>.md        ~50 tokens    stage behavior
card header              ~20 tokens    id + glyph + stage index
spec.md                  variable      the work
prior_stage_output       variable      previous stage result
acceptance_criteria      ~10 tokens    shell gates
```

Template variables in `prompt.md`:
- `{{system_context}}` — `.cards/system_context.md` (WORKS)
- `{{stage_instructions}}` — `.cards/stages/<stage>.md` (WORKS)
- `{{stage_index}}` / `{{stage_count}}` — position in chain (WORKS)
- `{{prior_stage_output}}` — previous stage's `output/result.md` (WORKS)
- `{{spec}}`, `{{plan}}`, `{{acceptance_criteria}}`, `{{memory}}` — WORK

## Templates

Live in `.cards/templates/`. Four shipping templates:

| Template | Stages | Models | Cost | Use When |
|----------|--------|--------|------|----------|
| **implement** | implement → qa | opus → sonnet | $$ | Clear requirements |
| **full** | spec → plan → implement → qa | sonnet → sonnet → opus → sonnet | $$$ | Complex/ambiguous |
| **cheap** | implement | ollama-local | free | Small fix, local model |
| **qa-only** | qa | sonnet | $ | Code review, audit |

**Create your own:** `cp -c -R .cards/templates/implement.jobcard .cards/templates/mine.jobcard`
then edit `meta.json` (stage_chain, stage_models, stage_providers, stage_budgets).
See `plan.md` §8 for annotated meta.json reference.

COW-clone: `cp -c` (macOS APFS) or `cp --reflink=auto` (Linux).

## APFS Copy, Dedupe, Compression (macOS)

- Fast template clone: `cp -c -R` (clonefile/COW on APFS).
- Metadata-safe bundle copy: `ditto --clone --extattr --acl --qtn --preserveHFSCompression <src> <dst>`.
- Keep default atomic replacement behavior; avoid `ditto --nonAtomicCopies` for card bundles.
- Cold-card storage option: `ditto --hfsCompression` (or preserve existing compression with default `--preserveHFSCompression`).
- For archive export/import, use `ditto -c -k` / `ditto -x -k`; prefer staging + atomic move into `.cards/`.

## Filename Tooling Compatibility (macOS 26.3 validation)

With `🂠-feat-auth.jobcard` on APFS, these worked in tests:

- `rg`, `tail`, `eza`, `bat`
- `bop inspect feat-auth`
- `bop logs feat-auth`
- `bop logs -f feat-auth` when the card is in `running/`

`exa` is no longer a Homebrew formula; standardize on `eza`.

## Filesystem Hardening

1. **Dispatcher lock:** `.cards/.locks/dispatcher.lock/owner.json`
   - Acquire with atomic `mkdir`; reject if live owner PID exists
   - Reclaim stale lock (dead PID) and continue

2. **Run lease per card:** `logs/lease.json`
   - Fields: `run_id`, `pid`, `pid_start_time`, `started_at`, `heartbeat_at`, `host`
   - Dispatcher heartbeats every 5s; stale floor is 30s

3. **Orphan reaping:** reap when PID dead OR lease stale
   - Normalize stale stage metadata (`running → pending/failed`)

4. **Idempotent controls:**
   - `bop kill <id>` succeeds even with stale PID
   - `bop retry <id>` clears failure reason + normalizes stage

## Meta.json Fields

| Field | Meaning |
|-------|---------|
| `glyph` | SMP playing card (suit=team, rank=priority) |
| `token` | BMP symbol for terminals/filenames |
| `stage` | Current stage: spec / plan / implement / qa |
| `stage_chain` | Full pipeline: `["implement","qa"]` |
| `stage_models` | Model per stage: `{"implement":"opus"}` |
| `stage_providers` | Adapter per stage: `{"implement":"claude"}` |
| `stage_budgets` | Token budget per stage: `{"implement":32000}` |
| `stages.<s>.status` | pending / running / done / blocked / failed |
| `zellij_session` | Zellij session name for running card |
| `zellij_pane` | Zellij pane ID for running card |
| `worktree_branch` | Git branch — never touch main |
| `acceptance_criteria` | Shell commands merge-gate runs |
| `decision_required` | true = merge-gate holds; `bop approve` clears |
| `provider_chain` | Failover order: `["claude","codex","ollama-local"]` |
| `timeout_seconds` | Kill agent after this many seconds |
| `estimates` | Planning poker: `{"alice":"🂻","bob":"🂺"}` |
| `poker_round` | "open" / "revealed" / absent |
| `labels` | `[{"name":"High Impact","kind":"effort"}]` |
| `subtasks` | `[{"id":"s1","title":"Do X","done":false}]` |
| `progress` | 0–100 overall completion % |

## Adapter Contract

```
adapter.zsh <workdir> <prompt_file> <stdout_log> <stderr_log> [timeout]
```

| Exit | Meaning | Action |
|------|---------|--------|
| 0 | Success | → done/ |
| 75 | Rate-limited | → pending/, rotate provider |
| other | Failure | → failed/ |

Available: `claude.zsh`, `codex.zsh`, `ollama-local.zsh`, `goose.zsh`, `aider.zsh`, `opencode.zsh`, `mock.zsh`.

## Planning Poker

Suit → perspective: ♠=complexity ♥=effort ♦=risk ♣=value.
Rank → magnitude: Ace=1, 2-10=face, Jack=13, Queen=21, King=40.
Joker (🃟/🃏) → needs breakdown, blocks consensus.

## Scripts

| Script | Purpose |
|--------|---------|
| `scripts/land_safe.zsh` | JJ/git safe-landing with gate checks |
| `scripts/bop_bop.zsh` | Goal→card→Zellij session bootstrap |
| `scripts/bop_focus.zsh` | Navigate Zellij panes by card id |
| `scripts/macos_cards_maintenance.zsh` | Thumbnail refresh, cold card compression |
| `scripts/render_card_thumbnail.swift` | Generate QuickLook/Thumbnail.png |
| `scripts/set_card_icon.swift` | Stage-coloured Finder icons for .jobcard bundles |

All zsh scripts use `ROOT=${0:A:h:h}`. `setopt NULL_GLOB` for `.jobcard` globs.
Never use `done` as a variable (zsh reserved — use `n_done`).

## Quick Look / macOS

- Xcode project: `macos/macos.xcodeproj`
- Extension: `macos/bop/PreviewViewController.swift`
- Host app: `macos/JobCardHost/` (registers `bop://` URL scheme)
- Build: Xcode → Product → Build
- Register: `lsregister -f ~/Applications/JobCardHost.app`
- Bundle IDs: `sh.bop` (host), `sh.bop.ql` (extension), `sh.bop.card` (UTI)

## Three Steering Layers

Every dispatched agent is shaped by (inner wins):

1. **Skills** (`~/.claude/skills/`) — HOW to approach work
2. **CLAUDE.md** — project constraints, shell rules, architecture
3. **`system_context.md`** — runtime domain fix ("NOT GTFS transit data")
