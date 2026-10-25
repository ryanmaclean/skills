---
name: paperclip
description: Operate Paperclip companies and agents safely and deterministically. Use when assigning work, unblocking heartbeat loops, repairing stale in_progress issues, and standardizing API command patterns.
---

# paperclip

Use this skill for Paperclip operations work.

## API command discipline

- Prefer wrapper script when available: `$AGENT_HOME/scripts/paperclip_api.sh`.
- For mutating requests, include `X-Paperclip-Run-Id` when run context exists.
- Build JSON with `jq -n`; avoid hand-built JSON strings.

## Heartbeat-safe task handling

1. Get assigned issues (`todo,in_progress,blocked`) for current agent.
2. Checkout issue once; if HTTP 409, do not retry.
3. Work only assigned scope.
4. Patch issue status with concise evidence comment.

## Stale in_progress recovery

When issues stay `in_progress` without active runs:

1. Query recent runs for assignee.
2. If no active run and no recent progress, patch to `blocked` or `todo` with reason.
3. Add exact unblock action (command, credential, or acceptance criteria).

## Auth + Paperclip interaction

- Do not start interactive auth inside agent heartbeats.
- Prefer preflight checks and deterministic env injection (`adapterConfig.env`).
- If credentials are missing, fail fast with a clear blocked update.

## Comment quality

Every run update should include:

- status line (`done`, `blocked`, or `in_progress`)
- what was executed
- concrete evidence (IDs, command result summary)
- next action if blocked
