## Extending RigFleet

RigFleet has 4 extension layers. Each solves a different problem:

| Layer | What | Where | How to add |
|---|---|---|---|
| **Skills** | Knowledge + workflows the agent loads on demand | `/opt/homebrew/lib/node_modules/rigfleet/skills/` or `~/.rigfleet/workspace/skills/` | `clawdhub install <slug>` or `npx add-skill <repo>` |
| **Extensions** | Custom channel plugins (TypeScript) | `~/.rigfleet/extensions/{name}/` | Create `rigfleet.plugin.json` + TypeScript source |
| **Channels** | Messaging platforms (built-in) | `rigfleet.json → channels.*` + `plugins.entries.*` | Configure in rigfleet.json, add credentials |
| **Cron jobs** | Scheduled autonomous tasks | `~/.rigfleet/cron/jobs.json` | Agent creates them via tool, or edit jobs.json directly |

### Skills: ClawdHub Ecosystem

Skills are the primary way to extend what the agent knows and can do. They're markdown files with optional scripts/assets that get loaded into context when relevant.

```bash
# Search for skills (vector search across the registry)
clawdhub search "postgres optimization"
clawdhub search "image generation"

# Browse latest skills
clawdhub explore

# Install a skill
clawdhub install supabase-postgres-best-practices
clawdhub install nano-banana-pro

# Install a specific version
clawdhub install my-skill --version 1.2.3

# List what's installed
clawdhub list

# Update all installed skills
clawdhub update --all

# Update a specific skill
clawdhub update my-skill
clawdhub update my-skill --force  # overwrite local changes
```

**Currently installed skills (bundled with RigFleet):**

| Category | Skills |
|---|---|
| **Messaging** | discord, slack, imsg, wacli, voice-call |
| **Social/Web** | bird (X/Twitter), blogwatcher, github, trello, notion |
| **Google** | gog, google-workspace-mcp, goplaces, local-places |
| **Media** | nano-banana-pro (Gemini image gen), openai-image-gen, video-frames, gifgrep, pixelation, sag (TTS), openai-whisper, sherpa-onnx-tts, songsee, camsnap |
| **Coding agents** | coding-agent (Codex/Claude Code/Pi), ccbg (background runner), tmux |
| **Productivity** | apple-notes, apple-reminders, bear-notes, things-mac, obsidian, himalaya (email) |
| **Smart home** | openhue (Philips Hue), eightctl (Eight Sleep), sonoscli, blucli (BluOS) |
| **Dev tools** | github, worktree, starter, desktop, supabase-postgres-best-practices, superdesign |
| **Content** | remotion-best-practices, remotion-fastest-tech-stack, humanizer, summarize, market, buildinpublic |
| **Meta** | skill-creator, clawdhub, find-skills, add-skill, model-usage, session-logs, recentplans, canvas |

### Creating Your Own Skill

A skill is just a folder with a `SKILL.md`:

```
my-skill/
├── SKILL.md              # Required: YAML frontmatter + markdown instructions
├── scripts/              # Optional: executable scripts
├── references/           # Optional: docs loaded on demand
└── assets/               # Optional: templates, images
```

**SKILL.md format:**
```markdown
---
name: my-skill
description: What this does and WHEN to trigger it. The description is the primary
  trigger — the agent reads this to decide whether to load the full skill.
---

# My Skill

Instructions go here. Only loaded AFTER the skill triggers.
Keep under 500 lines. Split large content into references/ files.
```

**Key principle: the context window is a shared resource.** Only include what the agent doesn't already know. Prefer concise examples over verbose explanations.

```bash
# Publish to ClawdHub
clawdhub login
clawdhub publish ./my-skill --slug my-skill --name "My Skill" --version 1.0.0

# Or publish to GitHub for npx add-skill
# (see add-skill skill for details)
```

### Multi-Agent Orchestration

RigFleet can spawn other AI agents (Codex, Claude Code, Pi) as background workers. This is how you run parallel coding tasks, reviews, or any work that benefits from multiple agents.

**The pattern:** `bash pty:true background:true workdir:/path command:"agent 'task'"`

```bash
# Spawn Codex to build something (background, auto-approve)
bash pty:true workdir:~/project background:true command:"codex exec --full-auto 'Build a REST API for todos'"
# Returns sessionId for tracking

# Spawn Claude Code on a different task
bash pty:true workdir:~/other-project background:true command:"claude 'Refactor the auth module'"

# Monitor all running agents
process action:list

# Check output of a specific agent
process action:log sessionId:XXX

# Send input if agent asks a question
process action:submit sessionId:XXX data:"yes"

# Kill a stuck agent
process action:kill sessionId:XXX
```

**Parallel PR reviews:**
```bash
# Fetch all PR refs
git fetch origin '+refs/pull/*/head:refs/remotes/origin/pr/*'

# Launch one agent per PR
bash pty:true workdir:~/project background:true command:"codex exec 'Review PR #86. git diff origin/main...origin/pr/86'"
bash pty:true workdir:~/project background:true command:"codex exec 'Review PR #87. git diff origin/main...origin/pr/87'"
```

**Parallel issue fixing with git worktrees:**
```bash
git worktree add -b fix/issue-78 /tmp/issue-78 main
git worktree add -b fix/issue-99 /tmp/issue-99 main

bash pty:true workdir:/tmp/issue-78 background:true command:"codex --yolo 'Fix issue #78: description. Commit and push.'"
bash pty:true workdir:/tmp/issue-99 background:true command:"codex --yolo 'Fix issue #99: description. Commit and push.'"
```

**Auto-notify when agent finishes:**
```bash
bash pty:true workdir:~/project background:true command:"codex --yolo exec 'Your task.

When completely finished, run: rigfleet gateway wake --text \"Done: summary\" --mode now'"
```

### Adding a New Channel

Channels are messaging platforms the agent can communicate through. Built-in: WhatsApp, Signal, Telegram, iMessage, Discord, Slack.

**Enable a built-in channel:**
```bash
# 1. Enable the plugin
jq '.plugins.entries.discord.enabled = true' ~/.rigfleet/rigfleet.json > /tmp/oc.json && mv /tmp/oc.json ~/.rigfleet/rigfleet.json

# 2. Add channel config
jq '.channels.discord = {enabled: true, dmPolicy: "pairing", groupPolicy: "disabled"}' ~/.rigfleet/rigfleet.json > /tmp/oc.json && mv /tmp/oc.json ~/.rigfleet/rigfleet.json

# 3. Add credentials (channel-specific)
# 4. Restart gateway
rigfleet gateway restart
```

**Build a custom channel extension:**

```
~/.rigfleet/extensions/{name}/
├── rigfleet.plugin.json    # {"id": "name", "channels": ["name"], "configSchema": {...}}
├── package.json            # Standard npm package
├── index.ts                # Entry point
└── src/
    ├── channel.ts          # Inbound message handling + outbound send
    ├── actions.ts          # Tool actions the agent can invoke
    ├── runtime.ts          # Lifecycle: start, stop, health check
    ├── config-schema.ts    # JSON schema for plugin config
    └── types.ts            # TypeScript types
```

```bash
# List installed extensions
ls ~/.rigfleet/extensions/

# View extension manifests
cat ~/.rigfleet/extensions/*/rigfleet.plugin.json | jq .

# Check extension source files
find ~/.rigfleet/extensions/ -name "*.ts" -not -path "*/node_modules/*"
```

### Cross-Channel Communication

The agent can receive on one channel and send on another, but there are guardrails:

```bash
# Check cross-context settings
cat ~/.rigfleet/rigfleet.json | jq '.tools.message.crossContext'
# allowAcrossProviders: true = agent CAN send across channels
# marker.enabled: false = no "[via Signal]" prefix on cross-channel messages

# If you see "Cross-context messaging denied" errors:
# The agent tried to send from a session bound to channel A to channel B.
# This is a security feature. To allow it:
jq '.tools.message.crossContext.allowAcrossProviders = true' ~/.rigfleet/rigfleet.json > /tmp/oc.json && mv /tmp/oc.json ~/.rigfleet/rigfleet.json
```

**BOOT.md notification protocol** (already configured):
The agent receives WhatsApp messages, responds on WhatsApp, then sends a notification summary to Signal. This is the primary cross-channel pattern — autopilot on one channel, control center on another.

### Canvas: Web UI for Connected Devices

Push HTML/games/dashboards to connected Mac/iOS/Android nodes:

```bash
# Check canvas config
cat ~/.rigfleet/rigfleet.json | jq '.canvasHost // "not configured"'

# List connected nodes
rigfleet nodes list

# Present HTML content on a node
# canvas action:present node:<node-id> target:http://localhost:18793/__moltbot__/canvas/my-page.html

# Canvas files live in:
ls ~/.rigfleet/canvas/
```

### Voice Calls

Initiate phone calls via Twilio/Telnyx/Plivo:

```bash
# Check if voice-call plugin is enabled
cat ~/.rigfleet/rigfleet.json | jq '.plugins.entries["voice-call"] // "not configured"'

# CLI usage
rigfleet voicecall call --to "+15555550123" --message "Hello"
rigfleet voicecall status --call-id <id>
```

### Cron: Scheduled Autonomous Tasks

```bash
# View all jobs
cat ~/.rigfleet/cron/jobs.json | jq '.jobs[] | {name, enabled, schedule: .schedule, channel: .payload.channel, to: .payload.to}'

# Job schedule types:
# "kind": "at", "atMs": <epoch>          — one-shot at specific time
# "kind": "every", "everyMs": <ms>       — recurring interval

# Job delivery targets:
# channel + to fields determine where results go (signal, whatsapp, telegram)
# sessionTarget: "isolated" = fresh context each run (no memory of previous runs)

# To add a job, the agent creates it via tool, or edit jobs.json:
# See existing jobs as templates in ~/.rigfleet/cron/jobs.json
```

---

## License

MIT
