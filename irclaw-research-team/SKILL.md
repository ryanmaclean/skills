---
name: irclaw-research-team
description: |
  Use when spawning parallel research agents across the irclaw fleet — fetching URLs, parsing docs with liteparse, querying Ollama, and writing structured markdown findings. Use when doing multi-source research, deep-diving GitHub repos, Datadog docs, or any URL list. Covers the full pipeline: curl → liteparse → ollama → research/*.md
---

# irclaw Research Team

## Overview

Spawn N parallel `claude` workers in Zellij panes. Each agent owns one research domain, fetches URLs, optionally parses docs via liteparse, and writes findings to `research/<domain>.md`. You watch them live and ctrl-c any runaway pane.

## Quick Start

```bash
# Launch 4 research agents in a new Zellij tab
nu ~/.claude/skills/zellij-team/zellij-team.nu \
  --files [research/tasks/deep-dive-gastown.md \
           research/tasks/deep-dive-elixir.md \
           research/tasks/deep-dive-llm.md \
           research/tasks/deep-dive-pipeline.md]
```

Task files live in `research/tasks/`. Each is a self-contained prompt with context, URLs, and output path.

## liteparse Pipeline

liteparse does NOT fetch URLs itself — pipe curl output in:

```bash
# Parse a remote PDF
curl -sL "https://example.com/doc.pdf" -o /tmp/doc.pdf
npx @llamaindex/liteparse /tmp/doc.pdf --output json > research/doc.json

# Parse an HTML page (save first)
curl -sL "https://example.com/page" -o /tmp/page.html
npx @llamaindex/liteparse /tmp/page.html --output json > research/page.json
```

## Ollama Headless Query (no UI needed)

```bash
# Query local Ollama directly — skip vibecode-webgui for agent use
curl -s http://localhost:11434/api/generate \
  -d '{"model":"llama3","prompt":"Summarize: <paste content>","stream":false}' \
  | jq -r .response
```

For fleet nodes: Ollama runs on each node at `localhost:11434`. No central coordinator needed for per-node research.

## Full Nushell Pipeline

```nu
# curl → liteparse → ollama → append to research file
def research-url [url: string, output_file: string, model: string = "llama3"] {
    let tmp = $"/tmp/(random chars -l 8).html"
    ^curl -sL $url -o $tmp
    let parsed = (^npx @llamaindex/liteparse $tmp --output json | from json)
    let text = ($parsed.pages | each { |p| $p.text } | str join "\n")
    let summary = (^curl -s http://localhost:11434/api/generate
        -d ($'{"model":"($model)","prompt":"Summarize for technical research: ($text | str substring 0..3000)","stream":false}')
        | from json | get response)
    $"\n## ($url)\n($summary)\n" | save --append $output_file
    rm $tmp
}
```

## Task File Format

Each file in `research/tasks/` follows this template:

```markdown
Working dir: /path/to/project

Context: [what's already known — point to existing research/*.md]

URLs to fetch:
1. https://... — what to extract
2. https://... — what to extract

Output: Append findings to research/<domain>.md under ## Deep Dive

Key questions:
- Question 1?
- Question 2?
```

## Research Output Structure

```
research/
  tasks/           ← agent task prompts (input)
  gastown-irclaw.md
  elixir-beam.md
  llm-ollama-webllm.md
  datadog-fleet.md
  tools-vibecode-liteparse.md
  pipeline-design.md
  SUMMARY.md       ← synthesized by leader after all agents complete
```

## Zellij Controls

| Action | Key |
|--------|-----|
| Switch panes | `Alt+arrow` |
| Kill one agent | `ctrl-c` in pane |
| Zoom pane | `ctrl-p z` |
| Switch tabs | `Alt+[` / `Alt+]` |
| Detach session | `ctrl-p d` |

## Fleet Distribution

Skills live at `~/.claude/skills/` on each node. Ansible playbook to sync:

```bash
ansible-playbook -i ansible/inventory ansible/sync-skills.yml
```

## Key Constraints

- **liteparse needs a local file** — always curl first
- **zeroclaw is private** — skip `github.com/zeroclaw-labs/irclaw`, use local ansible playbooks for context
- **Ollama port** — always `localhost:11434`, not a fleet-wide coordinator
- **ARM64 nodes** — liteparse runs fine via npx on aarch64; no binary issues
