---
name: tea
description: Gitea CLI (tea) for managing repos, issues, PRs, and code review. Use when interacting with Gitea repos, browsing code, creating issues, or managing PRs. Fleet agents use tea instead of curl for security scoping.
---

# tea — Gitea CLI

`tea` is the official Gitea CLI (like `gh` for GitHub). MIT licensed.

For concrete Gitea instance URLs, login names, and deployment details for a
specific fleet, see `## Deployment-specific details` at the bottom of this
file.

## Setup

- **Binary**: `/usr/local/bin/tea` (linux-arm64, from dl.gitea.com/tea/main/)
- **Config**: `~/.config/tea/config.yml` (login stored here)
- **Auth**: configured via `tea login add` (see Logins section below)

## Why tea instead of curl

Fleet agents use `tea` in `allowed_commands`, NOT `curl`. This prevents:
- Arbitrary URL fetching (prompt injection via web content)
- LAN service access (SearXNG, Qdrant, internal APIs)
- The Armenian spam incident (web search content hijacked tool calls)

`tea` can only talk to configured Gitea instances.

## Quick Reference

| Task | Command |
|------|---------|
| List repos | `tea repos ls` |
| Search repos | `tea repos search -k "irclaw"` |
| List org repos | `tea repos ls --owner studio` |
| Repo details | `tea repos studio/irclaw` |
| Clone repo | `tea clone studio/irclaw` |
| List issues | `tea issues ls -r studio/irclaw` |
| Create issue | `tea issues create -r studio/irclaw --title "..." --body "..."` |
| Close issue | `tea issues close -r studio/irclaw 42` |
| List PRs | `tea pulls ls -r studio/irclaw` |
| Create PR | `tea pulls create -r studio/irclaw --title "..." --base main --head feature` |
| List branches | `tea branches ls -r studio/irclaw` |
| List milestones | `tea milestones ls -r studio/irclaw` |
| List labels | `tea labels ls -r studio/irclaw` |
| Add comment | `tea comment -r studio/irclaw 42 "comment text"` |
| List notifications | `tea notifications` |
| Who am I | `tea whoami` |
| Raw API call | `tea api /repos/studio/irclaw` |
| Browse files | `tea api /repos/studio/irclaw/contents/src` |
| Read file | `tea api /repos/studio/irclaw/raw/README.md` |

## Repos

```sh
# List all repos you have access to
tea repos ls

# List repos for a specific org
tea repos ls --owner studio

# Search across all repos
tea repos search -k "gastown"

# Create a repo under an org
tea repos create --owner studio --name my-repo --description "..." --private

# Fork a repo
tea repos fork studio/irclaw

# Delete a repo (careful!)
tea repos delete studio/my-repo

# Migrate from external source
tea repos migrate --owner studio --name imported-repo \
  --clone-addr "https://github.com/user/repo.git"
```

## Issues (Task Tracking for Agents)

```sh
# List open issues
tea issues ls -r studio/irclaw

# List all issues (open + closed)
tea issues ls -r studio/irclaw --state all

# Filter by label
tea issues ls -r studio/irclaw --labels "bug,priority:high"

# Filter by milestone
tea issues ls -r studio/irclaw --milestones "v1.0"

# Create an issue
tea issues create -r studio/irclaw \
  --title "Add Qdrant memory support" \
  --body "Wire [memory] config section into irclaw-v2 config parser" \
  --labels "enhancement"

# Close an issue
tea issues close -r studio/irclaw 42

# Reopen an issue
tea issues reopen -r studio/irclaw 42

# Add a comment
tea comment -r studio/irclaw 42 "Fixed in commit abc123"
```

## Pull Requests

```sh
# List open PRs
tea pulls ls -r studio/irclaw

# Create a PR (must push branch first via git-safe)
tea pulls create -r studio/irclaw \
  --title "Add memory support" \
  --base main \
  --head feature-memory

# Merge a PR
tea pulls merge -r studio/irclaw 5
```

## Browsing Code (via API)

tea doesn't have a built-in file browser, but `tea api` gives raw API access:

```sh
# List files in a directory
tea api /repos/studio/irclaw/contents/src | python3 -c "
import json,sys
for f in json.load(sys.stdin):
    print(f'{f[\"type\"]:4s} {f[\"name\"]}')"

# Read a file (base64 decoded)
tea api /repos/studio/irclaw/contents/README.md | python3 -c "
import json,sys,base64
d=json.load(sys.stdin)
print(base64.b64decode(d['content']).decode())"

# Read raw file content
tea api /repos/studio/irclaw/raw/Cargo.toml

# List branches
tea api /repos/studio/irclaw/branches

# List commits
tea api /repos/studio/irclaw/commits?limit=5

# Search code (Gitea 1.25+)
tea api /repos/search?q=qdrant&topic=true
```

## Agent Task Workflow

Fleet agents can use tea + git-safe together for autonomous work:

```sh
# 1. Check assigned issues
tea issues ls -r studio/irclaw --assignee <agent-hostname>

# 2. Clone the repo (use the Gitea base URL from your tea login)
git-safe clone <GITEA_URL>/studio/irclaw.git /tmp/work
cd /tmp/work

# 3. Create a branch, make changes
git-safe checkout -b fix-42
# ... edit files ...
git-safe add -A && git-safe commit -m "Fix #42: description"

# 4. Push branch
git-safe push -u origin fix-42

# 5. Create PR
tea pulls create -r studio/irclaw --title "Fix #42" --base main --head fix-42

# 6. Comment on issue
tea comment -r studio/irclaw 42 "PR created: fix-42"
```

## Logins

```sh
# List configured logins
tea logins ls

# Add a new login
tea login add --name <alias> --url http://<gitea-host>:<port> --token <token>

# Switch default login
tea login default <alias>
```

## Output Formats

All list commands support `--output` / `-o`:

```sh
tea repos ls -o json          # JSON (for piping to python3/jq)
tea repos ls -o yaml          # YAML
tea repos ls -o csv           # CSV
tea repos ls -o table         # Table (default)
tea repos ls -o simple        # Simple text
```

## Ansible Deployment

tea is typically deployed via an Ansible playbook that:
- Downloads the binary from `dl.gitea.com/tea/main/tea-<version>-<arch>`
- Writes a `~/.config/tea/config.yml` with a pre-configured login
- Reads the Gitea token from an env var (never hardcoded in the playbook)

See `## Deployment-specific details` below for the concrete playbook name,
login name, and token variable used in this fleet.

## Limitations

- No official NetBSD build available — on NetBSD nodes use `git-safe` only
- `tea api` can hit any Gitea endpoint — but only on configured login hosts
- No file editing through tea (use git-safe clone → edit → push → PR)
- tea 0.12.x — some newer Gitea 1.25 features may not be covered

## Deployment-specific details

This skill is generic tea/Gitea CLI knowledge. Everything above applies to
any Gitea instance. Concrete login names, instance URLs, Ansible playbook
paths, and fleet host restrictions live in the deployment's fleet-ops docs.

**For the rust-town fleet:**
- QNAS Gitea (primary): `http://gitea.local:3000` / `http://10.0.3.210:3000`
- i9 Gitea (legacy fork host): `http://10.0.2.230:3001` — used for Rust
  crate patch forks; prefer QNAS for new repos
- Login name on fleet Pis: `qnas` (points to QNAS Gitea)
- Ansible playbook: `fleet-ops/ansible/29-deploy-tea-cli.yml`
- Token env var: `GITEA_QNAS_API_KEY`
- Not available on: NetBSD Pi nodes (no NetBSD binary) — those use git-safe
- Full CI instance/runner/fork details: `fleet-ops/docs/gitea-ci.md`
- Host inventory: `fleet-ops/ansible/inventory`

**For a different deployment:** run `tea login add --name <alias> --url
<your-gitea-url> --token <token>`, then substitute your login name for
`qnas` in all commands above.
