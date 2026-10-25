---
name: qcli
description: Qdrant CLI for fleet agents — wraps the Qdrant REST API with built-in embedding via Ollama snowflake-arctic-embed2. Use for vector search, collection browsing, upserts, and purges across fleet Qdrant collections.
---

# qcli

Qdrant CLI for fleet agents. Wraps Qdrant REST API with built-in embedding via ollama snowflake-arctic-embed2.

## Location

- **This host (studios-macbook-pro):** `/Users/studio/qcli/qcli`
- **mbp-m1:** Deploy to `~/qcli/qcli` (scp from studios-macbook-pro or clone from gitea)
- **fbryz3070:** Deploy to `/root/qcli/qcli`
- **qnas:** Not needed (Qdrant runs here, use curl directly if needed)

## Config

Config at `~/.qcli/config.json`:
```json
{
  "qdrant_url": "http://10.0.3.206:6333",
  "ollama_url": "http://10.0.3.207:11434",
  "embed_model": "snowflake-arctic-embed2",
  "api_key": "<64-char hex from qdrant container env>"
}
```

API key source: `QDRANT_API_KEY` env var overrides config file.

## Commands

### Browse
```bash
qcli collections                    # list all collections + point counts
qcli collection <name>              # collection details (dim, distance, count)
qcli count <name>                   # point count
qcli scroll <name> [--limit N]      # paginate points with payloads
qcli get <collection> <point_id>    # single point
```

### Search (auto-embeds query via snowflake-arctic-embed2)
```bash
qcli search <collection> "query text" [--limit 5]
qcli search-raw <collection> --vector-file vec.json
```

### Write
```bash
qcli upsert <collection> --text "..." --heading "..." [--source "..."] [--source-type "..."]
qcli upsert-file <collection> beads.jsonl   # bulk JSONL ingest
```

### Purge
```bash
qcli purge <collection> --filter key=value --dry-run   # preview
qcli purge <collection> --filter key=value              # execute
```

### Config
```bash
qcli config                         # show current config (key masked)
qcli config set api_key <value>     # set config value
```

### Global flags
- `--json` — machine-readable JSON output on any command

## Collections (as of 2026-05-08)

| Collection | Points | Purpose |
|---|---|---|
| lessons | 16 | Operational learnings (1024-dim cosine) |
| council_memory | 1235 | Council/agent deliberations |
| research | 940 | Research findings |
| gotchas | 157 | Known pitfalls |
| fleet_ledger | 190 | Fleet operational log |
| claude_code_memory | 45 | Claude Code session memory |
| agent_memory | 49 | Agent working memory |
| claude_memory | 43 | Claude conversation memory |
| bonzibuddy_pets | 24 | Pet data |
| fleet_memory_* | 4 each | Per-host fleet memory (12 hosts) |

## Embedding Details

- Model: `snowflake-arctic-embed2` (566M params, F16)
- Dimensions: 1024
- Distance: Cosine
- Ollama endpoint: `POST http://10.0.3.207:11434/api/embed`

## Dependencies

- Python 3.9+ (stdlib only, no pip packages)
- Network access to 10.0.3.206 (Qdrant) and 10.0.3.207 (ollama) — reachable via Tailscale subnet route from qnas

## Do NOT

- Do NOT store secrets, API keys, or credentials in Qdrant
- Do NOT purge without `--dry-run` first
- Do NOT bulk-upsert without checking collection dimensions match (1024 for snowflake)
