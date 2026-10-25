---
name: gh-auth
description: Enforce safe authentication behavior for GitHub and other CLIs. Use when handling credentials, checking auth status, or debugging agent auth failures so login churn and lockouts never happen.
---

# gh-auth

Use this skill whenever auth is involved.

## Non-negotiable rules

- Never run interactive login commands unless the user explicitly asks for login in this turn.
- Never repeat login attempts in a loop.
- If auth status is already healthy, stop there and continue the task.
- Prefer non-interactive credential propagation (env vars, instructions-path policy, secure files) over re-auth.

## Safe auth flow

1. Check status only.
2. If healthy, do not mutate auth state.
3. If unhealthy, choose one:
- Continue read-only work without auth.
- Use existing token/secret delivery already provided by the operator.
- Mark blocked with exact unblock action.

## Allowed checks (read-only)

```bash
gh auth status -h github.com || true
git config --global --get credential.helper || true
env | grep -E '^(GH_TOKEN|GITHUB_TOKEN)=' || true
```

## Forbidden by default

- `gh auth login`
- `gh auth logout`
- any browser/device-code auth initiation

## Paperclip-specific guidance

- For agent runs, prefer setting `adapterConfig.env.GH_TOKEN` through the Paperclip API.
- If no non-interactive credential is available, patch issue status to `blocked` with concise unblock instructions.
- Do not start auth flows from heartbeat loops.
