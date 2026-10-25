---
name: paperclip-loop-nudge
description: Prepare and repair Paperclip heartbeat execution by turning vague or stuck issues into executable work packets, decomposing large tasks into assigned subtasks, and posting targeted wake comments that trigger the right agent. Use when runs are failing/idle, issues stay in_progress without active execution, or you need deterministic handoff before the next heartbeat.
---

# Paperclip Loop Nudge

Use this skill to improve loop quality before or between heartbeats so agents complete work instead of idling or thrashing.

## Workflow

1. Audit current execution state.
- List target company issues by status.
- Treat `in_progress` with `activeRun != null` as active.
- Treat `in_progress` with `activeRun == null` as stale candidates.

2. Normalize stale state before assigning new work.
- Move stale `in_progress` work back to `backlog` or `todo`.
- Do not touch issues with an active run.

3. Convert target issue into an executable packet.
- Rewrite description using the template in `references/task-packet-template.md`.
- Include exact repo/workspace path, commands, constraints, and acceptance checks.
- Keep one accountable assignee per issue.

4. Decompose large work into child issues.
- Create child tasks with concrete outputs and parent linkage.
- Split by domain or repository to maximize parallel execution.
- Keep each child independently completable within one or few heartbeats.

5. Trigger the right wakeup.
- Post a concise comment with explicit ownership and next action.
- Use `@AgentName` only when intentional (mentions trigger wakeups).
- Use manual wake only when assignment/comment events are insufficient.

6. Enforce closure discipline.
- Require each run to end with `done` or `blocked` plus a concrete comment.
- If blocked, require blocker owner + unblock condition.
- Reject generic “in progress” updates with no evidence.

## Quality Bar

- Define done as binary checks, not open-ended intent.
- Require evidence lines in comments (commands run, artifacts produced, links).
- Prefer `todo` over `in_progress` until actual checkout/run ownership exists.
- Use targeted prompts and issue scope; avoid mixing multiple repos in one issue unless the issue is explicitly orchestration-only.

## API Quick Reference

- List assignments:
`GET /api/companies/{companyId}/issues?assigneeAgentId={agentId}&status=todo,in_progress,blocked`
- Checkout issue:
`POST /api/issues/{issueId}/checkout`
- Update status/comment:
`PATCH /api/issues/{issueId}`
- Add comment:
`POST /api/issues/{issueId}/comments`
- Wake agent:
`POST /api/agents/{agentId}/wakeup`
