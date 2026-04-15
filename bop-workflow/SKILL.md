---
name: bop-workflow
description: Use when explaining how to use bop from scratch — bare minimum setup, creating cards, dispatching agents, and reading results. Good for onboarding a new user who just has a project folder.
---

# bop: Bare Minimum Workflow

**bop** is a job card system. You write specs; agents do the work.
Cards live in `.cards/` as directories. The filesystem is the state machine.

---

## One-Time Setup (per machine)

```bash
# Install
brew install bop           # or: cargo install bop

# Set your API key (Claude adapter)
export ANTHROPIC_API_KEY=sk-...    # add to ~/.zshrc

# Done. No server, no database.
```

---

## Per-Project Setup (one-time per folder)

```bash
cd my-project/

bop init                   # creates .cards/ skeleton
bop factory install        # starts dispatcher + merge-gate as background services
```

`bop init` creates:
```
.cards/
  pending/       ← new work lands here
  running/       ← agent is working on it
  done/          ← agent finished
  merged/        ← accepted into your codebase
  failed/        ← something went wrong
  templates/     ← how cards are shaped (implement, qa, etc.)
  providers.json ← which agents handle which cards
  system_context.md ← tell agents what this project is about
```

Tell agents what they're working on:
```bash
echo "This is a Rust CLI tool. Tests run with: cargo test" > .cards/system_context.md
```

---

## Daily Loop

### 1. Deal a card
```bash
bop new implement fix-auth-bug
```
This drops a card in `pending/`. The dispatcher picks it up automatically.

### 2. Watch it run
```bash
bop status                 # card counts: 2 pending, 1 running, 5 done
bop inspect fix-auth-bug   # show spec + live log tail
```

Cards move: `pending/ → running/ → done/ → merged/`

### 3. Read the result
```bash
bop inspect fix-auth-bug   # shows output/result.md
```

Or browse in Finder — `.jobcard` bundles show as playing-card thumbnails.

---

## The Agent Prompt

Every card's `prompt.md` is rendered and sent to the agent. Key variables:

```
{{system_context}}      ← your .cards/system_context.md
{{stage_instructions}}  ← what this stage should do (e.g. "write code, run tests")
{{spec}}                ← your spec.md
{{acceptance_criteria}} ← shell commands that must pass before merge
```

Acceptance criteria runs automatically before merge:
```json
"acceptance_criteria": ["cargo test", "cargo clippy -- -D warnings"]
```

---

## Writing a Good Spec

Create or edit `.cards/pending/<id>.jobcard/spec.md`:

```markdown
# Fix Authentication Bug

## Problem
The login flow silently swallows JWT validation errors, causing
200 OK responses on invalid tokens.

## Expected
Return 401 with `{"error": "invalid_token"}` when JWT fails.

## Files to touch
- src/auth/middleware.rs (line ~45, validate_jwt fn)
- tests/auth_test.rs (add rejection test)
```

That's it. Agent reads the spec and does the work.

---

## Templates

| Template | Use when |
|----------|----------|
| `implement` | You have clear requirements |
| `full` | Fuzzy requirements (spec→plan→implement→qa) |
| `cheap` | Small fix, use local Ollama model |
| `qa-only` | Code review an existing branch |

```bash
bop new full design-new-api          # 4-stage pipeline
bop new cheap fix-typo               # local model, free
bop new qa-only review-pr-42         # just QA stage
```

---

## Providers (Which Agent Runs)

`.cards/providers.json` maps provider names to adapter scripts:

```json
{
  "providers": {
    "claude": { "command": "adapters/claude.zsh", "rate_limit_exit": 75 },
    "ollama":  { "command": "adapters/ollama-local.zsh", "rate_limit_exit": 75 }
  }
}
```

On rate-limit (exit 75), the dispatcher rotates to the next provider automatically.

---

## Three Things Users Touch

1. **`spec.md`** — what to build (plain markdown)
2. **`system_context.md`** — what the project is (orientation for every agent)
3. **`acceptance_criteria`** in meta.json — shell tests that gate merge

Everything else — prompts, providers, stages, git branches, logs — is handled automatically.

---

## Factory Commands

```bash
bop factory status         # check services are running
bop factory start          # start dispatcher + merge-gate
bop factory stop           # pause processing
bop factory install        # register as launchd services (survives reboots)
bop factory uninstall      # remove launchd services
```

---

## Useful Inspection Commands

```bash
bop status                 # counts by state
bop inspect <id>           # full detail: spec + meta + log tail
bop logs <id>              # live log stream
bop approve <id>           # clear a decision_required gate
bop kill <id>              # terminate a running agent
```
