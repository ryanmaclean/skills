---
name: jj
description: Use when working in a jj (Jujutsu) repository. Covers the core workflow, workspace management, and common operations. jj is a newer VCS — do not assume git commands work here.
---

# jj (Jujutsu) VCS Cheatsheet

jj is a Git-compatible VCS with a different mental model. Every change is always
a commit — there is no staging area. The working copy IS a commit (`@`).

## Core Concepts

| git concept | jj equivalent |
|-------------|--------------|
| staged changes | doesn't exist — edits are auto-tracked |
| `git commit` | `jj describe -m "msg"` + `jj new` |
| `git status` | `jj status` |
| `git log` | `jj log` |
| `git diff` | `jj diff` |
| `git checkout <branch>` | `jj edit <rev>` |
| `git branch` | `jj branch list` |
| worktree | `jj workspace` |

## Daily Workflow

```bash
# See what changed
jj status
jj diff

# Commit the current working copy change
jj describe -m "feat: add shell completions"
jj new          # creates a new empty change on top

# One-liner equivalent
jj commit -m "feat: add shell completions"

# View history
jj log
jj log -r '::@'   # ancestors of current change
```

## Workspaces (multi-agent isolation)

jj workspaces are like git worktrees but tracked in `.jj/`:

```bash
# List workspaces
jj workspace list

# Add a workspace (creates a directory with its own working copy)
jj workspace add ../my-feature-workspace

# Forget (remove) a workspace when done
jj workspace forget my-feature-workspace
```

## Revsets (query language)

```bash
jj log -r 'main..@'          # commits between main and current
jj log -r '@-'               # parent of current
jj log -r 'description("fix")'  # commits containing "fix"
```

## Key Differences from Git

1. **No staging area.** All file changes in the working directory are automatically part of the current change (`@`).

2. **Commits are mutable.** `jj describe` edits the current commit's message without creating a new one.

3. **`jj new` moves forward.** After describing a change, `jj new` creates a new empty commit on top. This is the equivalent of "I'm done with this commit, start a new one."

4. **`@` is always a commit.** The working copy is always represented as a commit, even when empty.

5. **Branches are optional.** jj uses anonymous commits; branches are just named pointers.

## In the bop context

When running as a dispatched agent in a jj workspace:

```bash
# Check you're in the right workspace
jj workspace list

# Make your changes (edit files normally)
# Then commit:
jj describe -m "feat: what I implemented"
jj new

# Verify your commit is recorded
jj log -r 'main..@-'
```

The merge-gate will squash and push your changes to main via `jj squash`.
Do NOT try to push to origin yourself.

## Common Mistakes

- ❌ `git add && git commit` — doesn't work, changes won't be recorded as jj commits
- ❌ `git push` — use `jj git push` if needed, but the merge-gate handles this
- ❌ `jj squash` before done — squash flattens into parent, reserve for cleanup
- ✅ `jj status` to see current state
- ✅ `jj describe -m "..."` + `jj new` to commit and move on
