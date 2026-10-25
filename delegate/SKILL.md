---
name: delegate
description: Use when the user says "ask an agent to", "create a team to", "IRC task to", "check with <host>", "have the fleet", "spawn", or "delegate" — signals work should be routed to a subagent, fleet bot, or specific node rather than done inline.
---

# Delegate — Routing Work to the Right Executor

## Trigger Phrases → Action

| User says | What to do |
|-----------|-----------|
| `"Ask an agent to <task>"` | Spawn a single background agent via Task tool |
| `"Create a team to <task>"` | Spawn parallel agents via zellij-team or tiger-team skill |
| `"IRC task to <task>"` | Post to `#agents` on IRC — fleet self-assigns |
| `"Check with <host> about <thing>"` | SSH/Ansible query, or DM the bot via IRC |
| `"Have the fleet <task>"` | IRC broadcast to `#agents` |
| `"Ask ubrpi502 to <task>"` | DM specific bot via IRC or SSH directly |
| `"Spawn"` / `"Delegate"` | Route to appropriate method below |

## Method 1: Task Tool (single agent)

```
"Ask an agent to find all Rust files with async/await"
→ Agent tool, focused prompt, run_in_background: true if independent
```

Subagent types: `Explore` (search/find), `general-purpose` (research/multi-step), `Plan` (architecture)

GSD subagents: `gsd-phase-researcher`, `gsd-planner`, `gsd-executor`, `gsd-debugger`, `gsd-verifier`

## Method 2: Agent Teams (parallel)

```
"Create a team to review the auth module"
→ zellij-team skill  — independent parallel tasks
→ tiger-team skill   — one hard problem, multiple expert lenses (attacker/defender/architect/pragmatist)
```

## Method 3: IRC Fleet Dispatch

```
"IRC task to check disk space on all Pis"
→ Post to #agents via i9 zeroclaw (connected to IRC at 10.0.3.203:6697)
```

IRC message format:
- `all: <task>` or `fleet: <task>` — every bot responds
- `ubrpi502: <task>` — only that bot responds

**Send from Claude Code:**
```bash
ssh studio@10.0.2.230 "zeroclaw channel send '#agents' 'fleet: <task>'"
ssh studio@10.0.2.230 "zeroclaw channel send 'ubrpi401' 'ubrpi401: <task>'"
```

## Method 4: SSH / Ansible (specific node)

```
"Check with ubrpi502 about build status"
→ ansible ubrpi502 -i ansible/inventory -m raw -a "<command>"
→ ssh ubuntu@<ip> "<command>"
```

Long-running work: use Zellij on the node (`ssh node` → `zellij attach research` → `claude`)

Web access: `http://<node-ip>:7681` (ttyd, port 7681 on all fleet nodes)

## Decision Table

| Situation | Method |
|-----------|--------|
| Quick code search | Task (Explore) |
| Multi-step research | Task (general-purpose) |
| Multiple independent problems | zellij-team |
| One hard problem, need multiple angles | tiger-team |
| Fleet-wide survey / self-organizing | IRC #agents |
| Specific node query | SSH / Ansible |
| Long-running agent session | Zellij + SSH |

## Key Fleet Nodes

| Node | IP | Role |
|------|----|------|
| ubrpi502 | 10.0.3.193 | Best Pi build node (Pi 5, 16GB) |
| pop4090 / 7950x4090pop | 10.0.2.42 | Heavy LLM, gastown coordinator |
| i9-zfs-pop | 10.0.2.230 | Gitea, IRC gateway, gastown coordinator |
| #agents IRC | 10.0.3.203:6697 | Fleet-wide broadcast |

## Verification contract

Routing work through this skill spawns sub-agents (Task tool, zellij-team,
tiger-team, IRC fleet bots, etc.). Every sub-agent dispatched by this
skill MUST attach a proveryay claims block to its final report so the
parent can verify instead of trust.

Reference: `~/.claude/skills/proveryay/CLAIMS-TEMPLATE.md`
(YAML-under-XML is the canonical format; atomic JSON is also accepted).

Example appropriate to delegation work (the sub-agent ran a fleet task and
is reporting back):

```
<claims>
- claim: "Task completed on ubrpi502"
  probe: "ansible ubrpi502 -i ansible/inventory -m raw -a 'test -f /tmp/delegate-done'"
  expect: "SUCCESS"
- claim: "IRC dispatch received by #agents"
  probe: "tail -5 ~/irc/10.0.3.203/#agents/out | grep -c '<task-id>'"
  expect: "1"
- claim: "Sub-agent produced expected artifact"
  probe: "ls -l /Users/studio/rust-town/out/<task-id>.json"
  expect: ".json"
- claim: "Remote service reachable after change"
  probe: "curl -sS -m 5 -o /dev/null -w '%{http_code}' http://gitea.local:3000/api/v1/version"
  expect: "200"
- claim: "No other fleet host degraded"
  probe: "ansible linux_pis_reachable -i ansible/inventory -m ping -o | grep -c SUCCESS"
  expect: "7"
</claims>
```

**Sub-agents spawned by this skill MUST emit this block in their final
report.** `proveryay` (SubagentStop hook) will verify each probe in
parallel and inject verdicts into the parent agent's next turn — claims
without the block are taken on faith and are therefore rejected by the
AX-first discipline.
