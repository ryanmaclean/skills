---
name: jail-factory
description: Run bop card work inside disposable FreeBSD jails on fbryz3070 — golden ZFS dataset cloned per card, nullfs workdir r/w + skills r/o, destroy on exit. Use when the factory needs worker isolation, per-card credential scoping, or sandboxed agent execution on the BSD fleet.
---

# jail-factory — disposable worker jails on fbryz3070

Host: `root@fbryz3070.local` (FreeBSD 15.0-p11, 128GB, NVIDIA 3070, pup+dd agent installed)

## Layout

- `jails` — 40G file-backed zpool at `/jails.img` (host is UFS; destroy pool + rm img to undo)
- `jails/golden@v2` — base snapshot: FreeBSD 15.0 + nu 0.116.1 (cargo-built — pkg binary needs FBSD_1.9), jq, git, pcre2, **claude-code 2.1.283** (linuxulator), codex (cargo-built, compiling)
- `jails/f-<card>` — per-card clone, mounted at `/jails/f-<card>`
- `/root/jail-factory/factory-run.nu` — lifecycle: clone → nullfs+linprocfs/linsysfs/fdescfs mounts → `jexec nu` → destroy → propagate exit code
- bop adapter: `bop-bench-wt/adapters/jail.nu` (stages workdir via scp, `BOP_JAIL_CMD` sets inner command)

## Usage

```bash
ssh root@fbryz3070.local \
  'nu /root/jail-factory/factory-run.nu <card-id> <host-workdir> "<nu command>"'
```

Example:

```bash
nu factory-run.nu card-42 /work/card-42 "open /work/spec.md | print; exit 0"
```

## Inside the jail

| Path | Source | Mode |
|---|---|---|
| `/work` | host workdir (nullfs) | r/w — card workspace |
| `/skills` | `/root/skills-factory` (nullfs) | r/o — skill corpus |
| `/dev` | devfs | minimal |

`ip4=inherit` currently — tighten to a vnet/epair when credential scoping lands.

## bop adapter contract

Wrap `factory-run.nu` in `adapters/jail.nu` conforming to
`<workdir> <prompt_file> <stdout_log> <stderr_log> [timeout]` — exit 75 for
rate-limit (EX_TEMPFAIL) so the provider chain rotates.

## Gotchas

- nushell + codex from pkg do NOT run on this base (built for `FBSD_1.9`, p11 ships `FBSD_1.8`) — both are cargo-built instead
- `claude` runs under Linux emulation: jail needs `linux=new` + linprocfs/linsysfs mounts + `fdescfs -o linrdlnk` at `/compat/linux/dev/fd`, all mounted host-side (jail lacks mount perms for them)
- `jj` is not packaged for FreeBSD — git only inside jails
- Always `umount -f` ALL mounts (incl. linux fs) before `zfs destroy` or the dataset stays busy
- `freebsd-update` refuses non-interactive runs; use `freebsd-update cron` (random sleep) or interactive

## Roadmap

- [x] `adapters/jail.nu` in bop repo calling this over ssh
- [x] claude-code in golden via linuxulator
- [ ] codex cargo build finishing (`/tmp/codex-build.log` on host)
- [ ] Per-card env file (credential scoping — no inherited DD keys)
- [ ] `rctl` limits: cpu/mem/pids per card
- [ ] vnet networking instead of ip4=inherit
- [ ] `fleet-eval` probe: `verify "jail f-X ran" --probe "jls -j f-X" --expect empty`
