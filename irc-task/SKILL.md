---
name: irc-task
description: Use when assigning tasks to fleet agents via IRC, sending one-shot messages to #agents, DM-ing a specific bot, or scripting IRC interactions using sic or ii (suckless IRC tools).
---

# IRC Task Assignment — sic and ii

## Fleet IRC

- **Server:** `10.0.3.203:6697` (TLS, Ergo IRC)
- **Channel:** `#agents` — all fleet bots listen here
- **Nick format:** matches hostname — `ubrpi502`, `i9-zfs-pop`, etc.
- **Protocol:** `fleet: <task>` broadcasts to all; `ubrpi502: <task>` targets one bot

## Tools

| Tool | What it is | TLS | Best for |
|------|-----------|-----|---------|
| **ii** | FIFO filesystem IRC | Yes (`-t`) | Persistent sessions, scripted dispatch |
| **sic** | stdin/stdout IRC | No (use socat) | Interactive, piped one-liners |
| **openssl s_client** | Raw TLS | Yes | One-shot messages without ii |

Source: All suckless tools forked at `http://10.0.2.230:3001/studio/` — includes sic, ii, dwm, st, dmenu, slstatus, surf, tabbed, sent, scroll, slock, sbase, ubase, sinit, abduco, dvtm, quark, vis, blind, farbfeld, wmii, and more.

## One-Shot Task via openssl (fastest)

```bash
# Send task to #agents and exit
(
  echo "NICK claude_dispatch"
  echo "USER claude_dispatch 0 * :task dispatcher"
  sleep 2
  echo "JOIN #agents"
  sleep 1
  echo "PRIVMSG #agents :fleet: check disk space on all Pis"
  sleep 2
  echo "QUIT"
) | openssl s_client -connect 10.0.3.203:6697 -quiet 2>/dev/null
```

## ii — Persistent Session (recommended for scripted workflows)

```bash
# Start ii in background (creates ~/irc/ directory tree)
ii -s 10.0.3.203 -p 6697 -t -n ryan -i ~/irc &
II_PID=$!
sleep 3

# Join agents channel
echo "/j #agents" > ~/irc/10.0.3.203/in

# Wait for join
sleep 2

# Send task
echo "fleet: check git sync status" > "~/irc/10.0.3.203/#agents/in"

# Read responses
tail -f "~/irc/10.0.3.203/#agents/out"
```

## ii — DM a Specific Bot

```bash
# Send DM directly to ubrpi502
echo "/m ubrpi502 ubrpi502: what is the disk usage?" > ~/irc/10.0.3.203/in
# Or open a direct channel:
echo "/j ubrpi502" > ~/irc/10.0.3.203/in
echo "ubrpi502: run df -h" > ~/irc/10.0.3.203/ubrpi502/in
tail -20 ~/irc/10.0.3.203/ubrpi502/out
```

## sic with TLS Proxy

sic itself doesn't support TLS — use socat as a proxy or use openssl directly.

```bash
# Start socat TLS proxy in background
socat TCP4-LISTEN:6668,reuseaddr OPENSSL:10.0.3.203:6697,verify=0 &
SOCAT_PID=$!

# Use sic through proxy
sic -h 127.0.0.1 -p 6668 -n ryan
# Then in sic:
# :j #agents
# fleet: check disk space

kill $SOCAT_PID
```

## Task Dispatch Pattern (scripted, no interaction)

```nu
# Nushell: send task and capture first response
def irc-task [task: string, timeout: int = 30] {
  let out_file = $"/tmp/irc-out-(random uuid)"

  (
    "NICK claude_dispatch\n" +
    "USER claude_dispatch 0 * :dispatcher\n"
  ) | save --raw /tmp/irc-cmd.txt

  sleep 2sec
  $"JOIN #agents\n" | save --append /tmp/irc-cmd.txt
  sleep 1sec
  $"PRIVMSG #agents :($task)\n" | save --append /tmp/irc-cmd.txt
  sleep ($timeout | into duration --unit sec)
  "QUIT\n" | save --append /tmp/irc-cmd.txt

  open --raw /tmp/irc-cmd.txt
    | ^openssl s_client -connect 10.0.3.203:6697 -quiet e>| ignore
    | lines
    | where { |l| $l =~ "PRIVMSG #agents" }
}
```

## Message Format for Fleet Bots

| Pattern | Meaning |
|---------|---------|
| `fleet: <task>` | All bots respond |
| `all: <task>` | All bots respond (alias) |
| `ubrpi502: <task>` | Only ubrpi502 responds |
| `ubrpi401,ubrpi502: <task>` | Specific subset |

Bots execute tasks via zeroclaw agent on each node.

## ii Directory Structure

```
~/irc/
  10.0.3.203/
    in          ← write server commands here (NICK, JOIN, etc.)
    out         ← server messages
    #agents/
      in        ← write channel messages here
      out       ← channel output
    ubrpi502/
      in        ← DM to ubrpi502
      out       ← DM responses
```

## Common Mistakes

| Mistake | Fix |
|---------|-----|
| sic hangs on TLS port | Use socat proxy or openssl s_client instead |
| ii creates no files | Wait 3+ seconds for connection to establish |
| Bot doesn't respond | Check bot is connected: look at #agents/out for join events |
| SASL auth needed | Ergo doesn't require SASL for registered nicks on this fleet |

## Verification contract

Tasks dispatched over IRC trigger remote fleet-bot sub-agents. Their
replies on `#agents/out` or DM `out` files are reports — they need
proofs. Every sub-agent that takes a task via this skill MUST emit a
proveryay claims block as part of its final IRC reply (or the
corresponding report file).

Reference: `~/.claude/skills/proveryay/CLAIMS-TEMPLATE.md`
(YAML-under-XML is canonical; atomic JSON is accepted).

Example appropriate to IRC-dispatched fleet work:

```
<claims>
- claim: "ii-agent running on the responding host"
  probe: "systemctl is-active ii-agent"
  expect: "active"
- claim: "Requested file exists at reported path"
  probe: "ls -l /tmp/irc-task-<id>.out"
  expect: "-rw-"
- claim: "Ergo IRC reachable from this bot"
  probe: "nc -z -w 3 ergo.local 6697 && echo open"
  expect: "open"
- claim: "Task artifact uploaded to Gitea"
  probe: "curl -sS -m 5 http://gitea.local:3000/api/v1/repos/studio/fleet-logs/contents/<id>.json"
  expect: "sha"
- claim: "Disk headroom OK on this host"
  probe: "df -Pk / | awk 'NR==2{print ($4>1048576)?\"OK\":\"LOW\"}'"
  expect: "OK"
</claims>
```

**Sub-agents spawned by this skill MUST emit this block in their final
report.** The `proveryay` hook auto-verifies on SubagentStop; without the
block, claims are unprovable and the AX-first contract is violated.
