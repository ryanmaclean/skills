---
name: freebsd-build-vm
description: Build, package, and release FreeBSD aarch64 artifacts via a dedicated QEMU VM. Use when building FreeBSD ports, native Rust/Zig/Go binaries, or TinyBSD images for an aarch64 fleet.
trigger: freebsd build, freebsd vm, aarch64 build, datadog-agent port, tinybsd, conclave freebsd, native freebsd build, fbuild
---

# FreeBSD aarch64 Build VM

Index + invocation reference for a FreeBSD aarch64 build pipeline. For deep
mechanics see the cross-referenced skills — do not duplicate them here.

## When to use this skill

Any time you need to produce a FreeBSD aarch64 binary or pkg:

- Building FreeBSD ports (e.g. `sysutils/datadog-agent`, custom ports)
- Native-building Rust/Zig/Go projects for FreeBSD
  (vs Zig cross-compile — see `zig-cc` skill for the cross path)
- Producing a TinyBSD Pi image
- New FreeBSD port submission needing a real ports-tree environment
- Any tool whose cross-compile trips on openssl-sys / libc FFI

## The VM

A typical FreeBSD aarch64 build VM setup:

- **Image:** FreeBSD 15.0-RELEASE aarch64 (cloud-init qcow2)
- **QEMU:** `qemu-system-aarch64` with HVF accel (macOS host), 4 cores, 8GB RAM
- **SSH:** forwarded port on the VM host → port 22 in the VM
- **Login:** `builder` (cloud-init creates the user in group `wheel` with
  NOPASSWD sudo). Root SSH is denied by default FreeBSD sshd policy
  (`PermitRootLogin` unset → `no`)
- **Jump-through from a remote workstation:**
  `ssh -J <user>@<host> -p <port> builder@localhost`
- **Ansible inventory pattern** (group `[build_vms]`):
  ```
  fbuild ansible_host=localhost ansible_port=<port> ansible_user=builder \
    ansible_become=true ansible_become_method=sudo \
    ansible_python_interpreter=/usr/local/bin/python3.11 \
    ansible_ssh_common_args='-o ProxyJump=<user>@<jump-host>'
  ```

VM lifecycle (start/stop/serial console/screen session) → `qemu-fleet` skill.
The VM runs under `screen -dmS fbuild` with a Unix-socket serial console —
never `-serial stdio` in detached mode (per `qemu-fleet`).

## Launching the VM (canonical)

**Use a launch wrapper — never run `qemu-system-aarch64` directly.** The
wrapper script should be deployed to the VM host via Ansible and cut from a
versioned template in your fleet-ops repo.

Launch (idempotent — kills stale screen of same name, relaunches detached):

```
ansible <vm-host> -m raw -a "~/vms/run-<vm-name>.sh"
ansible <vm-host> -m raw -a "screen -ls; pgrep -f qemu-system-aarch64"
```

The wrapper MANDATES `screen -dmS fbuild ...`. Bare `exec`, `nohup`,
`setsid`, and `-daemonize` all FAIL in different ways (SIGHUP, stdio
detach, monitor reattach). If you ever edit the launcher, keep the screen
invocation — the failure without it is deterministic.

## Getting source in / artifacts out

NOTE: virtfs/9p availability depends on your kernel config. Verify before relying on it.
If 9p kernel modules are not loaded in the guest, use `scp -J <jump-host> -P <port>`.

Preferred flow using `scp`:

- **Source IN.** Use `scp -J <jump-host> -P <port> <local_file> builder@localhost:<dest>`
  to transfer source into the VM.
- **Artifacts OUT.** Inside the VM: copy to `/home/builder/artifacts/`.
  Pull from the VM host with `scp -J <jump-host> -P <port> builder@localhost:/home/builder/artifacts/* ~/artifacts/`.
- **Ports tree (pre-seed).** Clone the Gitea ports mirror directly inside the VM
  or rsync from a staging directory on the VM host.

## Networking

The launcher defaults to SLIRP with `hostfwd` for SSH — NAT internet, but
**cannot reach the fleet LAN directly**. Fleet-internal services (e.g. a
private Gitea) are not reachable from inside the VM unless you set up:

1. **SSH remote-forward from the VM host into the VM.** First SSH to the host,
   THEN open a second SSH to the VM with `-R <port>:<internal-host>:<port>`.
   (ProxyJump + `-R` does NOT work — `-R` only applies to the last hop.)
2. **`guestfwd` in the QEMU launch args** — maps a virtual IP inside the VM
   to a real host via `nc` on the VM host. Bake this into the launch template.
3. **vmnet-bridged** so the VM is a first-class LAN member. See `qemu-fleet`
   skill. Requires QEMU to run with root privileges on macOS.

Options 2 and 3 are permanent fixes; option 1 is a reliable workaround.

## Directory layout inside the VM

- `/usr/src` — FreeBSD src tree (TinyBSD / kernel work)
- `/usr/ports` — full ports tree (mirror from your Gitea or official CDN)
- `/home/builder/artifacts/` — `.pkg` / binary output (login user is `builder`)
- `/home/builder/source/` — Rust / Zig / Go native builds outside the ports tree
- Ports tree + distfiles in `/usr/ports` are owned by root; use `sudo make package`
  from the builder account

## Standard pipeline (5 steps)

```
[source mirror] -> [fbuild build] -> [artifact]
               -> [Gitea release] -> [Ansible rollout]
               -> [fleet verify]
```

1. **Source.** Clone from your Gitea mirror — faster, offline-tolerant, and
   avoids stale upstream URL rot.
2. **Build.** In the VM via `make package` (ports), `cargo build --release`
   (Rust), or `zig build` (Zig native).
3. **Collect.** Output to `/home/builder/artifacts/` and `scp` to your workstation.
4. **Publish.** `tea release create` to your Gitea — see `gitea-release` skill.
5. **Deploy.** Ansible playbook pulls the release asset — see `ansible-fleet`.

## Invocation examples

### Build a port (generic)

```bash
# 0) Pre-flight (from workstation):
ssh <user>@<vm-host> 'pgrep -f qemu-system-aarch64' || echo "VM dead — see qemu-fleet"

# 0a) Establish Gitea reverse tunnel into the VM (SSH twice, NOT ProxyJump -R):
ssh <user>@<vm-host> \
  "ssh -p <vm-port> -R <gitea-port>:<gitea-host>:<gitea-port> -N -f builder@localhost"

# 1) Log in and set up the ports tree from the Gitea mirror:
ssh -J <user>@<vm-host> -p <vm-port> builder@localhost
sudo mkdir -p /usr/ports && sudo chown builder /usr/ports
git clone http://127.0.0.1:<gitea-port>/<org>/freebsd-ports.git /usr/ports
sudo rm -rf /usr/ports/sysutils/<port-name>
sudo git clone http://127.0.0.1:<gitea-port>/<org>/<port-repo>.git \
  /usr/ports/sysutils/<port-name>

# 2) Build in a persistent screen (never nohup on the VM):
cd /usr/ports/sysutils/<port-name>
screen -dmS dibuild bash -c 'sudo make BATCH=yes package 2>&1 | tee /tmp/build.log; \
  echo EXIT=$? >> /tmp/build.log'

# 3) Collect artifacts:
mkdir -p /home/builder/artifacts
sudo cp /usr/ports/sysutils/<port-name>/work/pkg/*.pkg /home/builder/artifacts/

# 4) Ship to workstation and publish:
scp -J <user>@<vm-host> -P <vm-port> \
  'builder@localhost:/home/builder/artifacts/*' \
  <local-artifacts-dir>/
```

### Export artifact from VM

```bash
scp -J <user>@<vm-host> -P <vm-port> \
    builder@localhost:/home/builder/artifacts/*.pkg \
    ~/artifacts/
# (inside the VM the build user must first `sudo cp work/pkg/*.pkg
# /home/builder/artifacts/` — the ports WRKDIR is root-owned)
```

### Build a Zig project native inside the VM

```bash
cd /home/builder/source
git clone http://127.0.0.1:<gitea-port>/<org>/<project>.git
cd <project>
zig build -Doptimize=ReleaseFast
```

### Publish a release to Gitea

```bash
tea login add --url http://<gitea-host>:<gitea-port> --token $GITEA_TOKEN
tea release create \
  --repo <org>/<repo> \
  --tag <target>/v<version>-<fleet-suffix> \
  --asset /path/to/artifact.pkg
```

See `gitea-release` skill for the full publish flow and `tea` skill for auth.

### Fleet rollout

```bash
cd <fleet-ops-dir>
ansible-playbook -i ansible/inventory \
  ansible/deploy-from-gitea-release.yml \
  --extra-vars "target=<target> tag=<target>/v<version>-<suffix>"
```

Typical playbook names:
- `ansible/build-and-release-freebsd.yml` — runs inside the build VM, tagged per
  target. `serial: 1` on `[build_vms]` — never parallel in this VM.
- `ansible/deploy-from-gitea-release.yml` — fleet distribution.
- `ansible/verify-fleet-artifact.yml` — post-install smoke test.

See `ansible-fleet` skill for safety rules (serial rollout, `--check`,
`--limit ONE_HOST` first).

## Release-tag naming convention

`<target>/v<upstream_version>-<fleet_suffix>` — the deploy playbook parses
`tag.split('/')[0]` to route the install handler. Examples:

- `datadog-integrations/v7.24.0-rigfleet1`
- `ii-agent/v5.4.0-freebsd-aarch64`
- `tinybsd-pi/v0.1.0-2026-04-14`

## Era matching (multi-VM build matrix)

**Principle:** era-match the build VM to the port's upstream vintage. Faster
than forward-porting stale `GH_TUPLE` URLs, deprecated `USES` syntax, and
gcc14 warnings-as-errors on older Go cgo code.

**RAM check first:** before spinning an era VM, confirm headroom on the VM
host via `vm_stat | head -4` plus the QEMU RSS sum. Default rule: max 2
concurrent VMs per host. Shut ephemeral VMs down after the artifact lands.

### Decision tree: which VM should I build on?

1. Port `Makefile` timestamp **< 2022**? → use a FreeBSD 13.x era VM
2. Port `Makefile` timestamp **2022–2024**? → use a FreeBSD 14.x era VM
3. Port **current (2024+)**, or your own code? → use FreeBSD 15.x (default)
4. Build fails with `GH_TUPLE` 404s or deprecated `USES`? →
   drop one era BEFORE forward-porting.
5. Never ship a 13.x-built artifact to FreeBSD 15 hosts without verifying
   `pkg add` on one host (backward-compat likely but not guaranteed for
   anything linking libc).

### Release tag includes the era

`<target>/v<version>-<vm-name>-<fleet-suffix>` — the `vm-name` segment
identifies the build era so operators know which FreeBSD the artifact came from.

## Lessons learned — launch wrapper failure modes

`-nohup`, `-detached`, `setsid`, and bare `exec qemu-system-aarch64` all
FAIL deterministically in at least one common scenario:

- **Bare `exec`.** Parent ssh/ansible tears down → SIGHUP → QEMU dies.
- **`nohup`.** Ignores SIGHUP but keeps QEMU attached to the terminal's
  stdio; `-serial stdio` blows up, and the monitor cannot be reattached.
- **`setsid`.** Detaches the session, but the QEMU monitor/serial go to
  /dev/null — you can't reach the VM console to fix a broken boot.
- **`-daemonize`.** Backgrounds QEMU but still fights with `-serial
  stdio`; and you lose monitor access without explicit `-monitor`
  telemetry, which future us always forgets to wire up.

**The only reliable pattern is `screen -dmS fbuild qemu ... -serial
unix:...`.** Named, reattachable, survives parent exit, monitor is
reachable via `screen -r fbuild`, serial via
`socat - UNIX-CONNECT:/tmp/<vm-name>.sock`. If someone edits the launcher
and removes the screen invocation, this WILL happen again.

## Alternative builder — any fleet FreeBSD aarch64 host (native, no VM)

If the QEMU VM is dead or its SLIRP network can't reach your Gitea, you do
NOT need a VM at all. Any FreeBSD 15 aarch64 host with sufficient RAM and
disk can serve as a native builder — same ISA as deploy targets, direct LAN
access, no tunneling.

### Safety rules — production hosts

Builds MUST be fully isolated from the host's ports tree:

- `WRKDIRPREFIX=/tmp/build/wrk` — isolates WRK from `/usr/obj`
- `PORTSDIR=/tmp/build/ports` — isolates ports tree from `/usr/ports`
- `DISTDIR=/tmp/build/distfiles` — isolates distfiles
- `PACKAGES=/tmp/build/packages` — isolates output pkgs
- `BATCH=yes` — no interactive prompts
- Never `pkg install` build artifacts on a production host — scp to workstation, publish to Gitea, deploy via Ansible
- `rm -rf /tmp/build` after every build (mandatory)

### Invocation

```bash
H=<freebsd-builder-host>
IV=<ansible-inventory-path>
B=/tmp/build

# Pre-flight
ansible $H -i $IV -m raw -a "freebsd-version; df -h /; \
  fetch -qo - http://<gitea-host>:<gitea-port>/api/v1/version"

# Clone from Gitea — direct, no tunnel needed
ansible $H -i $IV -m raw -a "mkdir -p $B && cd $B && \
  git clone --depth 1 <ports-repo-url> ports && \
  rm -rf ports/sysutils/<port-name> && \
  git clone <port-overlay-url> ports/sysutils/<port-name>"

# Build deps
ansible $H -i $IV -m raw -a "pkg install -y python3 pkgconf gmake py311-setuptools"

# Build, fully isolated
ansible $H -i $IV -m raw -a "cd $B/ports/sysutils/<port-name> && \
  env WRKDIRPREFIX=$B/wrk PORTSDIR=$B/ports DISTDIR=$B/distfiles PACKAGES=$B/packages \
    make BATCH=yes package"

# Collect, scp to workstation, cleanup
ansible $H -i $IV -m raw -a "find $B/packages $B/wrk -name '*.pkg' -type f"
scp <builder-host>:<pkg_path> <local-artifacts-dir>/
ansible $H -i $IV -m raw -a "rm -rf $B"  # MANDATORY
```

### When to prefer native host over VM

- VM is dead or SLIRP can't reach Gitea — native host bypasses both.
- One-off build; don't need a persistent ports tree.
- Target-matches-builder sanity (same hardware, no cross-era risk).

### When to prefer VM over native host

- Builds taking over 30 min (don't hog a production host that long).
- Builds needing > 4GB peak RAM on a host that shares RAM with other services.
- Any build that would touch `/usr/ports` system-wide (migrations, TinyBSD).
- Era-mismatched ports — production hosts only boot the current release.

## Common gotchas

- **`GH_TUPLE` rot.** Older ports have GitHub distfile URLs that were renamed
  or deleted. Rewrite every `GH_TUPLE` entry to your Gitea mirror.
- **Disk pressure.** A 40GB VM disk fills fast on a Go build. `df -h /` between
  builds, trim `/usr/ports/distfiles` and `work/` after each run.
- **`DD_API_KEY` never in artifacts.** Config stays in Ansible vault at
  deploy time; scan releases with the `redact` skill before publishing.
- **One build at a time.** The VM is not poudriere. Ansible uses
  `serial: 1` on `[build_vms]` — do not override.
- **Cloud-init first-boot delay.** 3–5 min on first boot; be patient.
- **Gitea asset size.** Large images (~1GB) may hit Gitea's release-asset
  limit — confirm before upload; external store may be needed.
- **QEMU HVF quirks.** Per `qemu-fleet`: always Unix-socket serial, never
  `-serial stdio` in a detached `screen` session.

## Dependencies

- `qemu-fleet` — VM lifecycle, serial console, screen session pattern
- `tea` — Gitea CLI auth and operations
- `gitea-release` — release + asset upload flow
- `ansible-fleet` — fleet deploy safety rules
- `freebsd-pi` — target hosts that consume these artifacts
- `freebsd` — general FreeBSD admin (`pkg`, `doas`, `rc.conf`)
- `zig-cc` — cross-compile alternative when native build is overkill
- `redact` — scrub secrets before publishing

## When NOT to use this skill

- **macOS binaries** → build on Mac hardware natively.
- **Linux amd64** → build on a Linux x86_64 host.
- **x86_64 FreeBSD** → this skill is aarch64-focused.
- **Single one-off aarch64 binary** → if `zig cc` cross-compile works
  (see `zig-cc`), skip the VM.

## Future

- poudriere for parallel port builds
- Dedicated Gitea Actions runner inside the VM (see `gitea-runner`)
- Second builder VM on a different host for HA / throughput

## Verification contract (REQUIRED for all build claims)

Every VM launch or native build MUST end with a `<claims>` block proving the artifact exists:

```<claims>
- claim: "VM is reachable on SSH"
  probe: "ssh -o ConnectTimeout=5 -p <vm-port> builder@localhost echo ALIVE"
  expect: "ALIVE"
- claim: "Artifact .pkg exists"
  probe: "ls /home/builder/artifacts/*.pkg"
  expect: ".pkg"
</claims>
```

Verdicts via `fleet-eval.nu verify`. See `fleet-eval` skill.

## Deployment-specific details

This skill is generic FreeBSD aarch64 build VM knowledge. Everything above
applies to any fleet with a QEMU FreeBSD VM on an Apple Silicon host. Concrete
host coordinates, Ansible stanzas, port rollout state, and the QEMU launch
wrapper for the rust-town fleet live in the deployment docs.

**For the rust-town fleet:**
- `fleet-ops/docs/build-vm.md` — VM inventory (fb-vm-24, fb13-vm, fb14-vm),
  SSH ProxyJump patterns (minim4-24, port 2222/2226/2227), Ansible `[build_vms]`
  stanza, RAM policy, QEMU launch wrapper template variables, FreeBSD Gruber
  port rollout status
- `fleet-ops/vm-templates/run-freebsd-vm.sh.template` — canonical launch
  wrapper (deploy via Ansible to `~/vms/run-fb-vm-24.sh` on minim4-24)
- `fleet-ops/build-artifacts/receipt.schema.json` — receipt schema that every
  artifact must conform to
- `fleet-ops/ansible/inventory` — source of truth for `[build_vms]` and
  `[freebsd_pis]` groups; use `fbrpi406` (or any 8GB FreeBSD 15 Pi) as the
  native-Pi builder fallback
- `.planning/phases/16-build-box-freebsd/01-PLAN.md` — provisioning history
- `.planning/phases/17-freebsd-build-pipeline/01-PLAN.md` — pipeline contract
- `.planning/phases/17.1-multi-era-bsd-matrix/01-PLAN.md` — multi-era VM matrix

**For a different deployment:** treat the examples above as templates.
Substitute your own VM host, port numbers, Gitea URL, and artifact store. Point
this section at your own fleet docs (e.g. `<your-fleet>/docs/build-vm.md`).
