---
name: providers
description: >
  Unified AI provider quota monitor. Shows Claude Code, Codex CLI, and Gemini CLI
  OAuth rate-limit utilization bars with reset countdowns. Reads credentials from
  each tool's local auth files — no API keys required.
---

# providers — AI Quota Monitor

Shows live OAuth quota usage for Claude, Codex, and Gemini in one view.

## Quick usage

```bash
nu ~/.claude/skills/providers/providers.nu          # all providers
nu ~/.claude/skills/providers/providers.nu --claude # Claude only
nu ~/.claude/skills/providers/providers.nu --codex  # Codex only
nu ~/.claude/skills/providers/providers.nu --gemini # Gemini only
nu ~/.claude/skills/providers/providers.nu --watch 30   # auto-refresh every 30s
nu ~/.claude/skills/providers/providers.nu --json       # raw JSON
```

From within a project (if providers.nu is in scripts/):

```bash
nu scripts/providers.nu
```

## Sample output

```
Claude Code
  5h        ████░░░░░░  44%   reset in 2h 47m
  7d        ██████░░░░  61%   reset in 127h 47m
  7d sonnet ███████░░░  70%   reset in 129h 47m

Codex CLI
  session   ██░░░░░░░░  21%   reset in 0h 12m
  weekly    ███░░░░░░░  32%   reset in 95h 12m

Gemini CLI
  Pro       █████░░░░░  52%   reset in 21h 0m
  Flash     ██░░░░░░░░  18%   reset in 21h 0m
```

Color scale: green <70% · amber 70–89% · red 90%+

## Credentials

| Provider | File | Field |
|----------|------|-------|
| Claude | `~/.claude/.credentials.json` | `claudeAiOauth.accessToken` |
| Codex | `~/.codex/auth.json` | `access_token` |
| Gemini | `~/.gemini/oauth_creds.json` | `access_token` |

## APIs

| Provider | Endpoint | Key fields |
|----------|----------|-----------|
| Claude | `GET https://api.anthropic.com/api/oauth/usage` | `five_hour.utilization`, `seven_day.utilization` (0–100) |
| Codex | `GET https://chatgpt.com/backend-api/wham/usage` | `session.percent_used`, `weekly.percent_used` (0–100) |
| Gemini | `POST .../loadCodeAssist` → `POST .../retrieveUserQuota` | `remainingFraction` (0.0–1.0, inverted to get used%) |

## Gemini token refresh

If `expiry_date` in `~/.gemini/oauth_creds.json` is in the past, the script
automatically attempts a token refresh by locating the Gemini CLI's `oauth2.js`
and extracting the client credentials. On failure it prints a warning and
suggests running `gemini` once manually.

## Notes

- All providers are fetched in sequence; a failure on one does not block others
- `--json` outputs a `{claude, codex, gemini}` object with raw API responses
- Gemini quota uses `remainingFraction` (what's left), converted to used%:
  `used = (1 - remainingFraction) * 100`
- If a provider binary or credentials file is absent, it shows `✗ <reason>` and continues
