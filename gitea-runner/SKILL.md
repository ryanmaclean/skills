---
name: gitea-runner
description: Set up, register, and manage Gitea Actions runners across the fleet. Write release workflows for Rust/Zig/Elixir projects. Debug runner connectivity issues (especially macOS+Tailscale). Use when user mentions gitea actions, CI/CD, runner, workflow, release pipeline, or automated builds.
trigger: gitea actions, gitea runner, act_runner, CI, release workflow, automated build, build pipeline
---

# Gitea Actions Runner

Runner binary: `act_runner` (Go). Latest: v0.3.1.
Workflows: `.gitea/workflows/*.yml` (GitHub Actions compatible with caveats).

## Critical Rules

### Secret Naming
Secret names **CANNOT start with `GITEA_` or `GITHUB_`**. Use `GITEATOKEN`, `RELEASETOKEN`, etc.
Set via API: `curl -X PUT ".../api/v1/repos/<owner>/<repo>/actions/secrets/<NAME>" -H "Authorization: token $TOKEN" -d '{"data":"<value>"}'`

### macOS + Tailscale = Screen Required (Primary) or Wait Loop (Launchd)
Go's TCP dialer fails with "no route to host" when act_runner starts before Tailscale network is ready. This happens with launchd at login/boot. The screen session bypasses this because it starts interactively.

**Primary fix: Use `screen -dmS gitea-runner` on macOS.**

```bash
# Start runner on macOS (most reliable with Tailscale)
screen -dmS gitea-runner bash -c 'cd ~/gitea-runner && ~/.local/bin/act_runner daemon --config config.yaml 2>&1 | tee runner.combined.log'

# Check status
screen -ls | grep gitea

# Attach to see live output
screen -r gitea-runner

# Detach: Ctrl-A, D
```

**Secondary: launchd with network wait loop** — if you need true auto-start at boot, add a retry loop to `run.sh` that waits for Gitea to be reachable before starting. launchd `KeepAlive` handles restart if it crashes. Example pattern:
```bash
GITEA_HOST="${GITEA_HOST:-http://your-gitea-instance:3000}"
for i in $(seq 1 12); do
    HOST=$(echo "$GITEA_HOST" | sed 's|http://||' | cut -d: -f1)
    PORT=$(echo "$GITEA_HOST" | sed 's|http://||' | cut -d: -f2)
    if nc -z -w 2 "$HOST" "$PORT"; then break; fi
    sleep 5
done
exec ~/.local/bin/act_runner daemon --config config.yaml
```

nohup, disown, bare `&` — **all fail** (no retry). Use screen or launchd+wait loop.

### No External Action Dependencies
Workflows must NOT use `actions/checkout@v4`, `actions/upload-artifact@v4`, or any action that fetches from GitHub if your runners have no internet access from job context. Use direct `git clone` and Gitea API calls instead.

## Registration

### Get Registration Token
```bash
# Read token from tea config (adjust path to match your OS/config location)
TOKEN=$(cat "<path-to-tea-config>" | \
  python3 -c "import sys,yaml; print(yaml.safe_load(sys.stdin)['logins'][0]['token'])")

GITEA_URL="${GITEA_HOST:-http://your-gitea-instance:3000}"
REG_TOKEN=$(curl -sf -X POST "$GITEA_URL/api/v1/user/actions/runners/registration-token" \
  -H "Authorization: token $TOKEN" | python3 -c "import sys,json; print(json.load(sys.stdin)['token'])")
```

The tea config file is typically:
- macOS: `~/Library/Application Support/tea/config.yml`
- Linux: `~/.config/tea/config.yml`

### Register Runner (Linux)
```bash
mkdir -p ~/gitea-runner && cd ~/gitea-runner
act_runner generate-config > config.yaml
# Edit config.yaml: set labels, envs.PATH, capacity
act_runner register --no-interactive \
  --instance "$GITEA_URL" \
  --token "$REG_TOKEN" \
  --name "$(hostname)-linux-$(uname -m)" \
  --labels "linux-$(uname -m):host"
```

### Register Runner (macOS)
```bash
mkdir -p ~/gitea-runner && cd ~/gitea-runner
~/.local/bin/act_runner generate-config > config.yaml
# Edit config.yaml: add PATH with ~/.cargo/bin to envs section
~/.local/bin/act_runner register --no-interactive \
  --instance "$GITEA_URL" \
  --token "$REG_TOKEN" \
  --name "$(hostname)-darwin-$(uname -m)" \
  --labels "darwin-$(uname -m):host"
```

### Systemd Service (Linux)
```ini
[Unit]
Description=Gitea Actions Runner
After=network.target

[Service]
User=<your-user>
WorkingDirectory=/home/<your-user>/gitea-runner
ExecStart=/usr/local/bin/act_runner daemon --config /home/<your-user>/gitea-runner/config.yaml
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

### Install act_runner Binary
```bash
# Linux x86_64
curl -L -o /usr/local/bin/act_runner \
  "https://gitea.com/gitea/act_runner/releases/download/v0.3.1/act_runner-0.3.1-linux-amd64"

# Linux ARM64
curl -L -o /usr/local/bin/act_runner \
  "https://gitea.com/gitea/act_runner/releases/download/v0.3.1/act_runner-0.3.1-linux-arm64"

# macOS ARM64
curl -L -o ~/.local/bin/act_runner \
  "https://gitea.com/gitea/act_runner/releases/download/v0.3.1/act_runner-0.3.1-darwin-arm64"

# Windows x86_64 (use PowerShell/curl on Windows)
curl -L -o C:\gitea-runner\act_runner.exe ^
  "https://gitea.com/gitea/act_runner/releases/download/v0.3.1/act_runner-0.3.1-windows-amd64.exe"

# NetBSD ARM64 — NO official binary. Cross-compile from macOS:
cd /tmp && git clone --depth=1 --branch v0.3.1 https://gitea.com/gitea/act_runner.git act_runner_src
cd act_runner_src && GOOS=netbsd GOARCH=arm64 go build -o act_runner-netbsd-arm64 .
scp act_runner-netbsd-arm64 root@<netbsd-host>:/usr/local/bin/act_runner

chmod +x <path>/act_runner
```

### NetBSD pkgsrc Bootstrap (for Rust on NetBSD runners)
```bash
# Bootstrap pkgsrc on NetBSD if no pkg_add available
PKGSRC_BASE="https://cdn.netbsd.org/pub/pkgsrc/packages/NetBSD/aarch64/10.0_2025Q4/All"
cd /tmp
ftp -o pkg_install.tgz "${PKGSRC_BASE}/pkg_install-20250417.tgz"
mkdir -p /usr/pkg && tar xzf pkg_install.tgz -C /usr/pkg
export PATH=/usr/pkg/sbin:/usr/pkg/bin:$PATH
echo "${PKGSRC_BASE}/" > /usr/pkg/etc/pkg_install.conf
pkg_add pkgin
echo "${PKGSRC_BASE}/" > /usr/pkg/etc/pkgin/repositories.conf
pkgin update && pkgin -y install rust-bin  # rust-bin 1.90.0 ~323MB
```

### Windows .runner File Location Issue
After `act_runner register` on Windows, the `.runner` file is created in `%USERPROFILE%\` (home dir), NOT in the working directory. Move it before starting daemon:
```cmd
move C:\Users\<user>\.runner C:\gitea-runner\.runner
```
The daemon will fail with "registration file not found" if `.runner` is not in the working directory.

## Workflow Templates

The templates below use `${{ secrets.GITEA_HOST }}` for the Gitea base URL and
`${{ secrets.GITEATOKEN }}` for the API token. Set both as repo secrets before
triggering a release. See the `## Deployment-specific details` section for
concrete fleet values.

### Rust Release (self-hosted, no external actions)
```yaml
name: Build and Release
on:
  push:
    tags: ['v*']

jobs:
  build-linux-x86_64:
    runs-on: linux-x86_64
    steps:
      - name: Checkout
        run: git clone --depth 1 --branch ${{ github.ref_name }} ${{ secrets.GITEA_HOST }}/${{ github.repository }}.git .
      - name: Build release
        run: cargo build --release
      - name: Get binary name
        run: |
          BIN=$(cargo metadata --no-deps --format-version 1 | python3 -c "import sys,json; bins=[t['name'] for p in json.load(sys.stdin)['packages'] for t in p.get('targets',[]) if 'bin' in t.get('kind',[])]; print(bins[0])")
          cp target/release/$BIN ${BIN}-linux-x86_64
          echo "$BIN" > .binname
      - name: Upload to release
        run: |
          BIN=$(cat .binname)
          TAG="${{ github.ref_name }}"
          GITEA_URL="${{ secrets.GITEA_HOST }}"
          RELEASE_ID=$(curl -sf "$GITEA_URL/api/v1/repos/${{ github.repository }}/releases/tags/$TAG" \
            -H "Authorization: token ${{ secrets.GITEATOKEN }}" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])" 2>/dev/null || echo "")
          if [ -z "$RELEASE_ID" ]; then
            RELEASE_ID=$(curl -sf -X POST "$GITEA_URL/api/v1/repos/${{ github.repository }}/releases" \
              -H "Authorization: token ${{ secrets.GITEATOKEN }}" -H "Content-Type: application/json" \
              -d "{\"tag_name\":\"$TAG\",\"name\":\"$TAG\"}" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
          fi
          curl -sf -X POST "$GITEA_URL/api/v1/repos/${{ github.repository }}/releases/$RELEASE_ID/assets?name=${BIN}-linux-x86_64" \
            -H "Authorization: token ${{ secrets.GITEATOKEN }}" -H "Content-Type: application/octet-stream" \
            --data-binary @${BIN}-linux-x86_64

  build-darwin-arm64:
    runs-on: darwin-arm64
    steps:
      - name: Checkout
        run: git clone --depth 1 --branch ${{ github.ref_name }} ${{ secrets.GITEA_HOST }}/${{ github.repository }}.git .
      - name: Build release
        run: |
          export PATH="$HOME/.cargo/bin:$PATH"
          cargo build --release
      - name: Upload to release
        run: |
          export PATH="$HOME/.cargo/bin:$PATH"
          BIN=$(cargo metadata --no-deps --format-version 1 | python3 -c "import sys,json; bins=[t['name'] for p in json.load(sys.stdin)['packages'] for t in p.get('targets',[]) if 'bin' in t.get('kind',[])]; print(bins[0])")
          cp target/release/$BIN ${BIN}-darwin-arm64
          TAG="${{ github.ref_name }}"
          GITEA_URL="${{ secrets.GITEA_HOST }}"
          RELEASE_ID=$(curl -sf "$GITEA_URL/api/v1/repos/${{ github.repository }}/releases/tags/$TAG" \
            -H "Authorization: token ${{ secrets.GITEATOKEN }}" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])" 2>/dev/null || echo "")
          if [ -z "$RELEASE_ID" ]; then
            RELEASE_ID=$(curl -sf -X POST "$GITEA_URL/api/v1/repos/${{ github.repository }}/releases" \
              -H "Authorization: token ${{ secrets.GITEATOKEN }}" -H "Content-Type: application/json" \
              -d "{\"tag_name\":\"$TAG\",\"name\":\"$TAG\"}" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
          fi
          curl -sf -X POST "$GITEA_URL/api/v1/repos/${{ github.repository }}/releases/$RELEASE_ID/assets?name=${BIN}-darwin-arm64" \
            -H "Authorization: token ${{ secrets.GITEATOKEN }}" -H "Content-Type: application/octet-stream" \
            --data-binary @${BIN}-darwin-arm64
```

### Zig Cross-Compile Release (ii-agent style)
```yaml
name: Build and Release
on:
  push:
    tags: ['v*']

jobs:
  cross-compile:
    runs-on: darwin-arm64
    steps:
      - name: Checkout
        run: git clone --depth 1 --branch ${{ github.ref_name }} ${{ secrets.GITEA_HOST }}/${{ github.repository }}.git .
      - name: Build all targets
        run: sh build-all-targets.sh
      - name: Upload binaries
        run: |
          TAG="${{ github.ref_name }}"
          GITEA_URL="${{ secrets.GITEA_HOST }}"
          RELEASE_ID=$(curl -sf -X POST "$GITEA_URL/api/v1/repos/${{ github.repository }}/releases" \
            -H "Authorization: token ${{ secrets.GITEATOKEN }}" -H "Content-Type: application/json" \
            -d "{\"tag_name\":\"$TAG\",\"name\":\"$TAG\"}" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
          for f in ../../packages/builds-*/*; do
            NAME=$(basename "$f")
            curl -sf -X POST "$GITEA_URL/api/v1/repos/${{ github.repository }}/releases/$RELEASE_ID/assets?name=$NAME" \
              -H "Authorization: token ${{ secrets.GITEATOKEN }}" -H "Content-Type: application/octet-stream" \
              --data-binary @"$f"
          done
```

### Elixir Release
```yaml
name: Build and Release
on:
  push:
    tags: ['v*']

jobs:
  build:
    runs-on: linux-x86_64
    steps:
      - name: Checkout
        run: git clone --depth 1 --branch ${{ github.ref_name }} ${{ secrets.GITEA_HOST }}/${{ github.repository }}.git .
      - name: Build release
        run: |
          export PATH="/opt/homebrew/opt/elixir/bin:/opt/homebrew/opt/erlang/bin:$PATH"
          export MIX_ENV=prod
          mix local.hex --force && mix local.rebar --force
          mix deps.get --only prod
          mix release
      - name: Upload tarball
        run: |
          TAG="${{ github.ref_name }}"
          TARBALL=$(find _build/prod/rel -name "*.tar.gz" | head -1)
          # ... same upload pattern as Rust
```

## Gitea Actions vs GitHub Actions — Key Differences

| Feature | Gitea | GitHub |
|---------|-------|--------|
| Workflow dir | `.gitea/workflows/` (preferred) | `.github/workflows/` |
| `runs-on` | Simple only: `runs-on: label` | Complex expressions supported |
| `concurrency` | **Ignored** | Supported |
| `permissions` | **Ignored** | Supported |
| `timeout-minutes` | **Ignored** | Supported |
| Action URLs | Supports absolute: `uses: https://gitea.com/...@main` | GitHub-only |
| Secret prefix | No `GITEA_` or `GITHUB_` prefix | No `GITHUB_` prefix |
| Expression functions | Only `always()` | Many |

## Troubleshooting

### Runner won't connect (macOS)
1. Check Tailscale: `tailscale status` — if running, use `screen -dmS` (never launchd/nohup)
2. Check runner logs: `cat ~/gitea-runner/runner.combined.log`
3. Verify Gitea reachable: `curl -s "$GITEA_URL/api/v1/version"`

### Workflow triggered but fails immediately
1. Check `[actions] ENABLED = true` in Gitea app.ini: `docker exec gitea cat /data/gitea/conf/app.ini | grep -A2 actions`
2. Check runner has matching label for `runs-on`
3. Check secret is set: `curl -s ".../api/v1/repos/<repo>/actions/secrets" -H "Authorization: token $TOKEN"`

### "invalid variable or secret name"
Secret name starts with `GITEA_` or `GITHUB_`. Rename to remove the prefix.

### Build fails on checkout
Replace `uses: actions/checkout@v4` with direct git clone:
```yaml
- run: git clone --depth 1 --branch ${{ github.ref_name }} ${{ secrets.GITEA_HOST }}/${{ github.repository }}.git .
```

## Adding a New Runner to the Fleet

1. Install act_runner binary (see URLs above; NetBSD requires cross-compile)
2. Get registration token from API
3. `act_runner register --no-interactive --instance "$GITEA_URL" --token $TOKEN --name <name> --labels <platform>:host`
4. Edit config.yaml: set `envs.PATH`, `capacity`, `timeout`, `labels`
5. Linux: create systemd service, `systemctl enable --now`
6. NetBSD: deploy `/etc/rc.d/gitea_runner` script (nohup-based), add `gitea_runner=YES` to `/etc/rc.conf`
7. macOS: `screen -dmS gitea-runner bash -c 'cd ~/gitea-runner && act_runner daemon --config config.yaml'`
8. Windows: `schtasks /create /tn gitea-runner /tr "wscript.exe C:\gitea-runner\start-runner.vbs" /sc ONLOGON /ru <user> /f`
9. Set `GITEATOKEN` secret on repos that need this runner

## References

- Gitea Actions docs: https://docs.gitea.com/usage/actions/overview
- act_runner docs: https://docs.gitea.com/usage/actions/act-runner
- act_runner releases: https://gitea.com/gitea/act_runner/releases
- Gitea release action: https://gitea.com/actions/gitea-release-action
- Secrets docs: https://docs.gitea.com/usage/actions/secrets
- Comparison with GitHub Actions: https://docs.gitea.com/usage/actions/comparison

## Deployment-specific details

This skill is generic Gitea Actions. Everything above applies to any Gitea
instance. Concrete host coordinates, runner IPs, and fleet-specific CI
conventions for the rust-town fleet live in the deployment docs.

**For the rust-town fleet:**
- `fleet-ops/docs/gitea-ci.md` — Gitea instance URLs, runner host table (5 hosts
  with IPs and labels), secret-name conventions, fork-repo locations,
  macOS Tailscale workaround reference

**For a different deployment:** treat the examples above as templates.
Set `GITEA_HOST` to your Gitea base URL, substitute your own runner hostnames,
and point this section at your own fleet docs.
