## Troubleshooting: WhatsApp

### "I sent a message but got no reply"

This is the #1 issue. The message arrives but the bot doesn't respond. Check in this order:

```bash
# 1. Is the bot actually running?
grep -i "whatsapp.*starting\|whatsapp.*listening" ~/.rigfleet/logs/gateway.log | tail -5

# 2. Check for 408 timeout drops (WhatsApp web disconnects frequently)
grep -i "408\|499\|retry" ~/.rigfleet/logs/gateway.err.log | tail -10
# If you see "Web connection closed (status 408). Retry 1/12" — this is normal,
# it auto-recovers. But if retries reach 12/12, the session dropped completely.

# 3. Check for cross-context messaging blocks
grep -i "cross-context.*denied" ~/.rigfleet/logs/gateway.err.log | tail -10
# Common: "Cross-context messaging denied: action=send target provider "whatsapp" while bound to "signal""
# This means the agent was in a Signal session and tried to reply on WhatsApp.
# FIX: The message needs to come through in the WhatsApp session context, not Signal.

# 4. Check the session exists for that contact
cat ~/.rigfleet/agents/main/sessions/sessions.json | jq -r 'to_entries[] | select(.key | test("whatsapp")) | "\(.key) | \(.value.origin.label // "?")"'

# 5. Check if the sender is allowed
cat ~/.rigfleet/rigfleet.json | jq '.channels.whatsapp | {dmPolicy, allowFrom, selfChatMode, groupPolicy}'
# If dmPolicy is "allowlist" and the sender isn't in allowFrom, message is silently dropped.

# 6. Check if it's a group message (groups are disabled by default)
cat ~/.rigfleet/rigfleet.json | jq '.channels.whatsapp.groupPolicy'
# "disabled" means ALL group messages are ignored.

# 7. Check for lane congestion (agent busy with another task)
grep "lane wait exceeded" ~/.rigfleet/logs/gateway.err.log | tail -5
# If the agent is stuck on a long LLM call, new messages queue up.

# 8. Check for agent run timeout
grep "embedded run timeout" ~/.rigfleet/logs/gateway.err.log | tail -5
# Hard limit is 600s (10 min). If the agent's response takes longer, it's killed.
```

### "WhatsApp fully disconnected"

```bash
# Check credential files exist (should be ~1400 files)
ls ~/.rigfleet/credentials/whatsapp/default/ | wc -l

# If 0 files: session was never created or got wiped
# Fix: re-pair with `rigfleet configure`

# Check for QR/pairing events
grep -i "pair\|link\|qr\|scan\|logged out" ~/.rigfleet/logs/gateway.log | tail -10

# Check for Baileys errors
grep -i "baileys\|DisconnectReason\|logout\|stream:error" ~/.rigfleet/logs/gateway.err.log | tail -20

# Nuclear fix: delete credentials and re-pair
# rm -rf ~/.rigfleet/credentials/whatsapp/default/
# rigfleet configure
```

---

## Troubleshooting: Telegram

### "Bots have issues / forget things"

Two separate problems that look the same:

```bash
# 1. Check for config validation errors (THE COMMON ONE)
grep -i "telegram.*unrecognized\|telegram.*invalid\|telegram.*policy" ~/.rigfleet/logs/gateway.err.log | tail -10
# Known issue: the keys "token" and "username" under accounts are not recognized.
# The correct field is "botToken", not "token".

# 2. Check the actual config
cat ~/.rigfleet/rigfleet.json | jq '.channels.telegram'
# Verify each bot has "botToken" (not "token") and "name" fields.

# 3. Check polling status — bots die after getUpdates timeout
grep -i "telegram.*exit\|telegram.*timeout\|getUpdates" ~/.rigfleet/logs/gateway.err.log | tail -10
# "[telegram] [sales] channel exited: Request to 'getUpdates' timed out after 500 seconds"
# This means the bot lost connection to Telegram's API and stopped listening.
# Fix: restart gateway — `rigfleet gateway restart`

# 4. Check the polling offset (if bot "forgets" or replays old messages)
cat ~/.rigfleet/telegram/update-offset-coder.json
cat ~/.rigfleet/telegram/update-offset-sales.json
# If lastUpdateId is stuck or 0, the bot will reprocess old messages.
# To skip to latest: the gateway sets this automatically on restart.

# 5. Check if both bots are starting
grep -i "telegram.*starting\|telegram.*coder\|telegram.*sales" ~/.rigfleet/logs/gateway.log | tail -10

# 6. "Bot forgets" — this is usually a session issue, not Telegram
# Each Telegram user gets their own session in sessions.json.
# Check if the session exists:
cat ~/.rigfleet/agents/main/sessions/sessions.json | jq -r 'to_entries[] | select(.key | test("telegram")) | "\(.key) | \(.value.origin.label // "?")"'

# 7. Check if compaction happened (context window pruned = "forgot")
SESS_ID="paste-session-id"
grep '"compaction"' ~/.rigfleet/agents/main/sessions/$SESS_ID.jsonl | wc -l
# If compaction count > 0, old messages were pruned from context.
# The agent's compaction mode is:
cat ~/.rigfleet/rigfleet.json | jq '.agents.defaults.compaction'
```

### Telegram config fix template

```bash
# Correct Telegram config structure:
cat ~/.rigfleet/rigfleet.json | jq '.channels.telegram = {
  "enabled": true,
  "accounts": {
    "coder": {
      "name": "Bot Display Name",
      "enabled": true,
      "botToken": "your-bot-token-here"
    },
    "sales": {
      "name": "Sales Bot Name",
      "enabled": true,
      "botToken": "your-bot-token-here"
    }
  },
  "dmPolicy": "pairing",
  "groupPolicy": "disabled"
}' > /tmp/oc.json && mv /tmp/oc.json ~/.rigfleet/rigfleet.json
```

---

## Troubleshooting: Signal

### "Signal RPC Failed to send message"

This blocks cron jobs and cross-channel notifications.

```bash
# 1. Check if signal-cli process is alive
ps aux | grep "[s]ignal-cli"

# 2. Check the RPC endpoint
grep -i "signal.*starting\|signal.*8080\|signal.*rpc" ~/.rigfleet/logs/gateway.log | tail -10
# Should see: "[signal] [default] starting provider (http://127.0.0.1:8080)"

# 3. Check for connection instability
grep -i "HikariPool\|reconnecting\|SSE stream error\|terminated" ~/.rigfleet/logs/gateway.err.log | tail -10
# "HikariPool-1 - Thread starvation or clock leap detected" = signal-cli internal DB issue
# "SSE stream error: TypeError: terminated" = lost connection to signal-cli daemon

# 4. Check for rate limiting
grep -i "signal.*rate" ~/.rigfleet/logs/gateway.err.log | tail -5
# "Signal RPC -5: Failed to send message due to rate limiting"

# 5. Check for wrong target format
grep -i "unknown target" ~/.rigfleet/logs/gateway.err.log | tail -5
# "Unknown target "adi" for Signal. Hint: <E.164|uuid:ID|...>"
# The agent must use phone numbers (+1...) or uuid: format, not names.

# 6. Fix profile name warning spam
grep -c "No profile name set" ~/.rigfleet/logs/gateway.err.log
# If high count: run signal-cli updateProfile to set a name

# 7. Test signal-cli directly
ACCT=$(cat ~/.rigfleet/rigfleet.json | jq -r '.channels.signal.account')
echo "Account: $ACCT"
# signal-cli -a $ACCT send -m "test" +TARGET_NUMBER

# 8. Check if the signal-cli daemon needs restart
# The gateway manages signal-cli as a subprocess.
# Restart the whole gateway: rigfleet gateway restart
```

---

## Troubleshooting: Cron Jobs

```bash
# 1. Overview of all jobs
cat ~/.rigfleet/cron/jobs.json | jq -r '.jobs[] | "\(.enabled | if . then "ON " else "OFF" end) \(.state.lastStatus // "never" | if . == "error" then "FAIL" elif . == "ok" then "OK  " else .  end) \(.name)"'

# 2. Failing jobs with error details
cat ~/.rigfleet/cron/jobs.json | jq '.jobs[] | select(.state.lastStatus == "error") | {name, error: .state.lastError, lastRun: (.state.lastRunAtMs | . / 1000 | todate), id}'

# 3. Read the actual run log for a failing job
JOB_ID="paste-job-uuid-here"
tail -20 ~/.rigfleet/cron/runs/$JOB_ID.jsonl | python3 -c "
import sys, json
for line in sys.stdin:
    try:
        obj = json.loads(line)
        if obj.get('type') == 'message':
            role = obj['message']['role']
            text = ''.join(c.get('text','') for c in obj['message'].get('content',[]) if isinstance(c,dict))
            if text.strip():
                print(f'[{role}] {text[:300]}')
    except: pass
"

# 4. Common cron failure causes:
#    - "Signal RPC -1" → Signal daemon down, see Signal section above
#    - "gateway timeout after 10000ms" → gateway was restarting when cron fired
#    - "Brave Search 429" → free tier rate limit hit (2000 req/month)
#    - "embedded run timeout" → job took longer than 600s

# 5. Next scheduled run times
cat ~/.rigfleet/cron/jobs.json | jq -r '.jobs[] | select(.enabled) | "\(.name): \((.state.nextRunAtMs // 0) | . / 1000 | todate)"'

# 6. Disable a broken job temporarily
cat ~/.rigfleet/cron/jobs.json | jq '(.jobs[] | select(.name == "JOB_NAME")).enabled = false' > /tmp/cron.json && mv /tmp/cron.json ~/.rigfleet/cron/jobs.json
```

---

## Troubleshooting: Memory / "It Forgot"

The memory system has 3 layers. When the agent "forgets," one of these broke:

### Layer 1: Context window (within a session)

```bash
# Check compaction count for a session (compaction = old messages pruned)
grep -c '"compaction"' ~/.rigfleet/agents/main/sessions/SESSION_ID.jsonl
# 7 compactions = the agent has "forgotten" its earliest messages 7 times.

# Check compaction mode
cat ~/.rigfleet/rigfleet.json | jq '.agents.defaults.compaction'
# "safeguard" = only compacts when hitting context limit
```

### Layer 2: Workspace memory files

```bash
# What daily memory files exist
ls -la ~/.rigfleet/workspace/memory/

# What's in MEMORY.md (long-term curated)
cat ~/.rigfleet/workspace/MEMORY.md

# Search memory files for something specific
grep -ri "KEYWORD" ~/.rigfleet/workspace/memory/
```

### Layer 3: Vector memory database (SQLite + Gemini embeddings)

```bash
# What files are indexed
sqlite3 ~/.rigfleet/memory/main.sqlite "SELECT path, size, datetime(mtime/1000, 'unixepoch') as modified FROM files;"

# How many chunks (text fragments) exist
sqlite3 ~/.rigfleet/memory/main.sqlite "SELECT COUNT(*) FROM chunks;"

# Search chunks by text (FTS5 full-text search)
sqlite3 ~/.rigfleet/memory/main.sqlite "SELECT substr(text, 1, 200) FROM chunks_fts WHERE chunks_fts MATCH 'KEYWORD' LIMIT 5;"

# Check embedding config
sqlite3 ~/.rigfleet/memory/main.sqlite "SELECT value FROM meta WHERE key='memory_index_meta_v1';" | python3 -m json.tool

# Check for Gemini embedding rate limits (breaks indexing)
grep -i "gemini.*batch.*failed\|RESOURCE_EXHAUSTED\|429" ~/.rigfleet/logs/gateway.err.log | tail -10
# "embeddings: gemini batch failed (2/2); disabling batch" = indexing degraded

# Rebuild memory index (re-index all workspace files)
# Delete the DB and restart gateway — it will rebuild:
# rm ~/.rigfleet/memory/main.sqlite && rigfleet gateway restart
```

---

## Searching Sessions

### Find a person's conversations

```bash
# Search session index by name (case-insensitive)
cat ~/.rigfleet/agents/main/sessions/sessions.json | jq -r 'to_entries[] | select(.value.origin.label // "" | test("NAME"; "i")) | "\(.value.sessionId) | \(.value.lastChannel) | \(.value.origin.label)"'
```

### Find sessions by channel

```bash
cat ~/.rigfleet/agents/main/sessions/sessions.json | jq -r 'to_entries[] | select(.value.lastChannel == "whatsapp") | "\(.value.sessionId) | \(.value.origin.label // .key)"'
# Replace "whatsapp" with: signal, telegram, or check .key for cron sessions
```

### Most recent sessions

```bash
cat ~/.rigfleet/agents/main/sessions/sessions.json | jq -r '[to_entries[] | {id: .value.sessionId, updated: .value.updatedAt, label: (.value.origin.label // .key), ch: (.value.lastChannel // "cron")}] | sort_by(.updated) | reverse | .[:10][] | "\(.updated | . / 1000 | todate) | \(.ch) | \(.label)"'
```

### Search message content across all sessions

```bash
# Quick: find which session files contain a keyword
grep -l "KEYWORD" ~/.rigfleet/agents/main/sessions/*.jsonl

# Detailed: show matching messages with timestamps
grep "KEYWORD" ~/.rigfleet/agents/main/sessions/*.jsonl | python3 -c "
import sys, json
for line in sys.stdin:
    path, data = line.split(':', 1)
    try:
        obj = json.loads(data)
        if obj.get('type') == 'message':
            role = obj['message']['role']
            text = ''.join(c.get('text','') for c in obj['message'].get('content',[]) if isinstance(c,dict))
            if text.strip():
                sid = path.split('/')[-1].replace('.jsonl','')[:8]
                ts = obj.get('timestamp','')[:19]
                print(f'{ts} [{sid}] [{role}] {text[:200]}')
    except: pass
" | head -30
```

### Read a specific session transcript

```bash
# Last 30 messages from a session
tail -50 ~/.rigfleet/agents/main/sessions/SESSION_ID.jsonl | python3 -c "
import sys, json
for line in sys.stdin:
    try:
        obj = json.loads(line)
        if obj.get('type') == 'message':
            role = obj['message']['role']
            text = ''.join(c.get('text','') for c in obj['message'].get('content',[]) if isinstance(c,dict))
            if text.strip() and role != 'toolResult':
                print(f'[{role}] {text[:300]}')
                print()
    except: pass
"
```

---

## Config Editing

### Safe edit pattern

Always: backup, edit with jq, restart.

```bash
cp ~/.rigfleet/rigfleet.json ~/.rigfleet/rigfleet.json.bak.manual
jq 'YOUR_EDIT_HERE' ~/.rigfleet/rigfleet.json > /tmp/oc.json && mv /tmp/oc.json ~/.rigfleet/rigfleet.json
rigfleet gateway restart
```

### Common edits

```bash
# Switch WhatsApp to allowlist
jq '.channels.whatsapp.dmPolicy = "allowlist" | .channels.whatsapp.allowFrom = ["+1XXXXXXXXXX"]' ~/.rigfleet/rigfleet.json > /tmp/oc.json && mv /tmp/oc.json ~/.rigfleet/rigfleet.json

# Enable WhatsApp autopilot (bot responds as you to everyone)
jq '.channels.whatsapp += {dmPolicy: "open", selfChatMode: false, allowFrom: ["*"]}' ~/.rigfleet/rigfleet.json > /tmp/oc.json && mv /tmp/oc.json ~/.rigfleet/rigfleet.json

# Add number to Signal allowlist
jq '.channels.signal.allowFrom += ["+1XXXXXXXXXX"]' ~/.rigfleet/rigfleet.json > /tmp/oc.json && mv /tmp/oc.json ~/.rigfleet/rigfleet.json

# Change model
jq '.agents.defaults.models = {"anthropic/claude-sonnet-4": {"alias": "sonnet"}}' ~/.rigfleet/rigfleet.json > /tmp/oc.json && mv /tmp/oc.json ~/.rigfleet/rigfleet.json

# Set concurrency
jq '.agents.defaults.maxConcurrent = 10 | .agents.defaults.subagents.maxConcurrent = 10' ~/.rigfleet/rigfleet.json > /tmp/oc.json && mv /tmp/oc.json ~/.rigfleet/rigfleet.json

# Disable a plugin
jq '.plugins.entries.imessage.enabled = false' ~/.rigfleet/rigfleet.json > /tmp/oc.json && mv /tmp/oc.json ~/.rigfleet/rigfleet.json
```

### Restore from backup

```bash
# Latest backup
cp ~/.rigfleet/rigfleet.json.bak ~/.rigfleet/rigfleet.json

# List all backups by date
ls -lt ~/.rigfleet/rigfleet.json.bak*

# Validate JSON before restart
python3 -m json.tool ~/.rigfleet/rigfleet.json > /dev/null && echo "OK" || echo "BROKEN"

# Nuclear reset
rigfleet configure
```

---

## Channel Security Modes

| Mode | Behavior | Risk |
|---|---|---|
| `open` + `allowFrom: ["*"]` | Anyone can message, bot responds to all | HIGH — burns API credits, bot speaks as you |
| `allowlist` + `allowFrom: ["+1..."]` | Only listed numbers get through | LOW — explicit control |
| `pairing` | Unknown senders get a code, you approve | LOW — approval gate |
| `disabled` | Channel completely off | NONE |

### Check current security posture

```bash
cat ~/.rigfleet/rigfleet.json | jq '{
  whatsapp: {policy: .channels.whatsapp.dmPolicy, from: .channels.whatsapp.allowFrom, groups: .channels.whatsapp.groupPolicy, selfChat: .channels.whatsapp.selfChatMode},
  signal: {policy: .channels.signal.dmPolicy, from: .channels.signal.allowFrom, groups: .channels.signal.groupPolicy},
  telegram: {policy: .channels.telegram.dmPolicy, groups: .channels.telegram.groupPolicy, bots: [.channels.telegram.accounts | to_entries[] | "\(.key)=\(.value.enabled)"]},
  imessage: {enabled: .channels.imessage.enabled, policy: .channels.imessage.dmPolicy}
}'
```

---

## Workspace Files

| File | What | When to edit |
|---|---|---|
| `SOUL.md` | Personality: tone, style ("no em dashes, lowercase casual") | To change how the bot talks |
| `IDENTITY.md` | Name (Jarvis), creature type, emoji | To rebrand |
| `USER.md` | Owner info, preferences | When user context changes |
| `AGENTS.md` | Operating rules: memory protocol, safety, group chat behavior, heartbeat instructions | To change bot behavior |
| `BOOT.md` | Startup instructions (autopilot notification protocol: WA → Signal) | To change what happens on boot |
| `HEARTBEAT.md` | Periodic checklist (empty = no heartbeat API calls) | To add/remove periodic tasks |
| `MEMORY.md` | Curated long-term memory (loaded only in main/direct sessions) | Bot manages this itself |
| `TOOLS.md` | Contacts, SSH hosts, device nicknames | To add local tool notes |
| `memory/*.md` | Daily raw logs, topic-specific chat logs | Bot writes automatically |

---

## Session JSONL Format

Each `.jsonl` file has one JSON object per line. Types:

| type | What |
|---|---|
| `session` | Session header: id, timestamp, cwd |
| `message` | Conversation turn: role (user/assistant/toolResult), content, model, usage |
| `custom` | Metadata: `model-snapshot`, `rigfleet.cache-ttl` |
| `compaction` | Context window was pruned (old messages dropped) |
| `model_change` | Model was switched mid-session |
| `thinking_level_change` | Thinking level adjusted |

Session index (`sessions.json`) keys:
- Pattern: `agent:main:{channel}:{contact}` or `agent:main:cron:{job-uuid}`
- Fields: `sessionId` (UUID = filename), `lastChannel`, `origin.label` (human name), `origin.from` (canonical address), `updatedAt` (epoch ms), `chatType` (direct/group)

---

## Gateway Startup Sequence

Normal startup takes ~3 seconds:
```
[heartbeat] started
[gateway] listening on ws://127.0.0.1:18789
[browser/service] Browser control service ready
[hooks] loaded 3 internal hook handlers (boot-md, command-logger, session-memory)
[whatsapp] [default] starting provider
[signal] [default] starting provider (http://127.0.0.1:8080)
[telegram] [coder] starting provider
[telegram] [sales] starting provider
[whatsapp] Listening for personal WhatsApp inbound messages.
[signal] signal-cli: Started HTTP server on /127.0.0.1:8080
```

If any line is missing, that component failed to start. Check `gateway.err.log`.

---

## Known Error Patterns

| Error | Meaning | Fix |
|---|---|---|
| `Web connection closed (status 408)` | WhatsApp web timeout, auto-retries up to 12x | Usually self-heals. If reaches 12/12, restart gateway |
| `Signal RPC -1: Failed to send message` | signal-cli daemon lost connection | Restart gateway |
| `Signal RPC -5: Failed to send message due to rate limiting` | Signal rate limit | Wait and retry, reduce message frequency |
| `No profile name set` (signal-cli WARN) | Floods logs, harmless | `signal-cli -a +ACCOUNT updateProfile --given-name "Name"` |
| `Cross-context messaging denied` | Agent tried to send across channels | Not a bug — security guardrail. Message must originate from correct channel session |
| `getUpdates timed out after 500 seconds` | Telegram bot lost polling connection | Restart gateway |
| `Unrecognized keys: "token", "username"` | Wrong config keys for Telegram bots | Use `botToken` not `token` in rigfleet.json |
| `RESOURCE_EXHAUSTED` (Gemini 429) | Embedding rate limit | Reduce workspace file churn, or upgrade Gemini quota |
| `lane wait exceeded` | Agent blocked on long LLM call | Wait, or restart if stuck > 2 min |
| `embedded run timeout: timeoutMs=600000` | Agent response exceeded 10 min | Break task into smaller pieces |
| `gateway timeout after 10000ms` | Gateway unreachable during restart window | Cron fired while gateway was down — transient |

---

