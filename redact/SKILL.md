---
name: redact
description: Use when printing, logging, or verifying any credential, API key, token, or secret — pipe through redact to show only first/last 4 chars (e.g. fc1d...4439) instead of the full value
---

# redact

Pipe-based secret masking. Shows first 4 + last 4 chars for identification without exposing the full value.

## Usage

```bash
# Pipe any output containing secrets
echo $GITEA_API_KEY | redact                  # → fc1d...4439
curl -s http://host/api | redact              # masks tokens in JSON
some-command 2>&1 | redact                    # masks in logs

# Direct arg
redact $SOME_TOKEN                            # → abcd...wxyz

# Custom reveal length
echo $TOKEN | redact --head 6 --tail 6        # → fc1df7...345360
```

## When to Use

**Always** when you need to reference or verify a secret:
```bash
# ❌ NEVER
echo "Token: $GITEA_API_KEY"
TOKEN="abc123..."
echo $TOKEN

# ✅ ALWAYS
echo $GITEA_API_KEY | redact
echo "Token: $(echo $GITEA_API_KEY | redact)"
```

## Auto-Detected Patterns

- 40-char hex strings (Gitea, SHA-like)
- `ghp_` / `ghs_` GitHub tokens
- `glpat-` GitLab tokens
- `Authorization: bearer/token <value>` headers
- `api_key=`, `auth_token=`, `password=`, `secret=`, `token=` key-value pairs

## Installation

Script lives at `~/.local/bin/redact` (Python 3, no dependencies).
Available on all hosts where `~/.local/bin` is in PATH.

## Fleet Distribution

```bash
ansible pis:nas:workstations -i ansible/inventory -m copy \
  -a "src=~/.local/bin/redact dest=~/.local/bin/redact mode=0755"
```
