---
name: rigfleet-ii-agent-ops
description: Build, deploy, and verify the Zig ii-agent across 5 cross-compile targets. Use when cutting a new ii-agent version, deploying to a Pi, or debugging post-deploy reconnection to IRC #agents. Bound to /Users/studio/rust-town/ii-agent/.
trigger: ii-agent, zig build, deploy ii-agent, ii-agent version, !version
---

# ii-agent Build + Deploy Ops

Project: `/Users/studio/rust-town/ii-agent/`
Current: v1.0.0 (c04b93e) — 100 tests green as of 2026-04-06.

## Build (cross-compile)

```bash
cd /Users/studio/rust-town/ii-agent

zig build -Dtarget=aarch64-linux-musl -Doptimize=ReleaseSafe
zig build -Dtarget=aarch64-linux-gnu  -Doptimize=ReleaseSafe
zig build -Dtarget=aarch64-netbsd     -Doptimize=ReleaseSafe
zig build -Dtarget=x86_64-linux-gnu   -Doptimize=ReleaseSafe
zig build -Dtarget=aarch64-macos      -Doptimize=ReleaseSafe
```

Output: `zig-out/bin/ii-agent` → copy into `packages/builds-<ver>/<target>/`.

## Run tests

```bash
cd /Users/studio/rust-town/ii-agent && zig build test
```

## Deploy to a single host

```bash
# 1. Stage binary
scp zig-out/bin/ii-agent ubuntu@ubrpi405:/usr/local/bin/ii-agent.new

# 2. Atomic swap + restart via ansible (correct keys/sudo)
ansible ubrpi405 -i /Users/studio/rust-town/fleet-ops/ansible/inventory -m shell -a \
  "systemctl stop ii-agent \
    && mv /usr/local/bin/ii-agent.new /usr/local/bin/ii-agent \
    && systemctl start ii-agent"
```

Never deploy fleet-wide untested — see `ansible-fleet` FIFO-brick rule.

## Verify reconnection

After restart the bot rejoins `#agents` on fleet IRC (10.0.3.203 TLS 6697):

1. Wait for `JOIN #agents` from `nick=<host>` (e.g. `ubrpi405`)
2. Send `!version` in channel → bot replies with new `v<ver>`
3. Send `!sysinfo` → sanity check memory/uptime

See `fleet-irc` for connecting.

## Built-in commands (11 as of v0.12+)

`!version`, `!sysinfo`, `!exec`, `!task`, `!ls`, `!cat`, `!du`, `!df`, `!ps`,
`!help`, `!ping`. `!exec` and `!task` delegate to the tool harness.

## Historical bugs (hit >=1 per version)

| Bug | Version fixed | Symptom |
|-----|---------------|---------|
| FIFO blocked reads | 0.8 | hung on `/tmp/*_cmd` with no reader |
| sigchld handling | 0.11 | zombie processes after `!exec` |
| `!sysinfo` memory on Linux | 0.14.2 | always printed 0 MB |
| `du -sh` false positive | 0.14.0 | reported wrong size |
| OSD path parsing | 0.13 | truncated path at first space |

## Release artifacts

After build, publish to Gitea — see `gitea-release`.

## Common mistakes

| Mistake | Fix |
|---------|-----|
| Using `scp` to replace a running binary | Stage `.new`, stop service, `mv`, start |
| Deploying all hosts at once | `--limit` one host, verify, then expand |
| Forgetting `ReleaseSafe` → debug build | Always `-Doptimize=ReleaseSafe` for packages |
| Running `zig build` from repo root | Must `cd src/ii-agent/` first |
| Assuming bot reconnected | Check IRC `JOIN` + `!version` response explicitly |

## Related

- `ansible-fleet` — deploy safety (serial, --limit, --check)
- `fleet-irc` — verify reconnection
- `rigfleet-fleet-inventory` — host IPs and keys
- `gitea-release` — publish artifacts
- `zig-cc` — underlying cross-compile mechanics
