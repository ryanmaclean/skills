---
name: jail-factory
description: Run bop card work inside disposable FreeBSD jails on fbryz3070 — golden ZFS dataset cloned per card, nullfs workdir r/w + skills r/o, destroy on exit. Use when the factory needs worker isolation, per-card credential scoping, or sandboxed agent execution on the BSD fleet.
---

# jail-factory — disposable worker jails on fbryz3070

Host: `root@fbryz3070.local` (FreeBSD 15.0-p11, 128GB, NVIDIA 3070, pup+dd agent installed)

## Layout

- `jails` — 40G file-backed zpool at `/jails.img` (host is UFS; destroy pool + rm img to undo)
- `jails/golden@v1` — base snapshot: FreeBSD 15.0 + nu 0.116.1 (cargo-built — pkg binary needs FBSD_1.9), jq, git, pcre2
- `jails/f-<card>` — per-card clone, mounted at `/jails/f-<card>`
- `/root/jail-factory/factory-run.nu` — lifecycle: clone → nullfs mounts → `jexec nu` → destroy → propagate exit code

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

- nushell from pkg does NOT run on this base (needs `FBSD_1.9` symbols); the golden uses the cargo-built 0.116.1 at `/usr/local/bin/nu`
- `jj` is not packaged for FreeBSD — git only inside jails
- Always `umount -f` dev/nullfs mounts before `zfs destroy` or the dataset stays busy
- `freebsd-update` refuses non-interactive runs; use `freebsd-update cron` (has random sleep) or interactive

## Roadmap

- [ ] `adapters/jail.nu` in bop repo calling this over ssh
- [ ] Per-card env file (credential scoping — no inherited DD keys)
- [ ] `rctl` limits: cpu/mem/pids per card
- [ ] vnet networking instead of ip4=inherit
- [ ] `fleet-eval` probe: `verify "jail f-X ran" --probe "jls -j f-X" --expect empty`
