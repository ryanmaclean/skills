---
name: bsd-portability
description: Lint shell scripts, Makefiles, CI workflow run blocks, Dockerfiles, expect scripts, and Nushell external-command calls for GNU-coreutils-only or BSD-only syntax that breaks across the fleet's mixed FreeBSD/NetBSD/OpenBSD/macOS/Linux/Windows/Alpine targets. Trigger phrases: "GNU flag", "portable shell", "works on FreeBSD", "stat -c", "bsd portability lint".
---

# bsd-portability

Fleet runtime environments in play: FreeBSD 15 (Pis, VMs, build host), NetBSD
10/11 (Pis, VMs), OpenBSD 7.8 (Pi 5 target), macOS (studio, laptops), Ubuntu
20.04/22.04/24.04 Pis, Pop!_OS workstations, Windows (2 hosts), GitHub
ubuntu-latest and macos-latest runners, Alpine/busybox in some QEMU agent
specs, and Docker images in vibecode-webgui/fleet-ops.

A shell idiom that "just works" on the author's dev machine (usually macOS or
an Ubuntu Pi) routinely fails silently or loudly on one of the other seven
runtime families. This skill's job is to catch that class of bug via static
lint before it ships.

## When to use this

- Reviewing or writing any `*.sh`, `Makefile*`, GitHub Actions / Gitea Actions
  `run:` block, `Dockerfile`, `*.exp` (expect script), or Nushell script that
  shells out via `^cmd`.
- Someone says "works on FreeBSD", "portable shell", "GNU flag", "stat -c",
  "bsd portability lint", or reports a script that behaves differently on a
  BSD host than on Linux/macOS.
- Before merging any script destined to run on more than one fleet OS.

## How to use it

```
nu ~/.claude/skills/bsd-portability/lint.nu <path> [--json] [--fix-suggestions]
```

- `<path>` — file or directory to scan. Directories are scanned recursively
  for `*.sh`, `Makefile*`, `*.yml`/`*.yaml` (workflow run blocks only),
  `Dockerfile*`, `*.exp`, and `*.nu` (only `^cmd` external-call lines).
- `--json` — machine-readable output (default; JSON is always emitted, this
  flag is accepted for explicitness/back-compat).
- `--fix-suggestions` — include the `fix` field in each finding (on by
  default; findings always carry `fix`, this flag is a no-op kept for CLI
  ergonomics).
- Exit code: `1` if any findings, `0` if clean. Output schema:
  `{"schema_version": "v1", "path": ..., "findings": [...], "count": N}`.
  Each finding: `{file, line, rule, snippet, fix}`.

## Ground truth, not vibes

This lint encodes ~20 known GNU-vs-BSD/uutils divergences (see `lint.nu`'s
`rules` table for the authoritative, versioned list: sed -i, mktemp
templates, stat -c/-f, date -d, realpath -m, timeout, nproc, sha256sum,
cp --reflink, bash shebang, grep -P, find -printf, GNU tar long opts, du/df
flag divergence, env -u, which, sha256 -q, and the Nushell external-redirect
footgun). Each rule carries a `message` explaining *why* it breaks and on
*which* OS family, plus a concrete `fix`. Do not hand-wave "this might not
be portable" — run the lint and cite the specific rule id.

## Fleet-specific fixes preferred, in order

1. Nushell builtin (`hash sha256`, `date`, `path expand`, `sys cpu | length`,
   `ls` pipelines) — already the house standard for new scripts, works
   identically on every target OS with zero forking.
2. Explicit OS branch (`if $nu.os-info.name == "macos" { ... }`) when a
   builtin doesn't cover the case.
3. POSIX-portable external command as a last resort, only when both GNU and
   BSD accept the exact same invocation.

Never add a `|| true` / `2>/dev/null` catch-all that converts a missing-flag
error into a silently wrong result — several fleet incidents (smolBSD git,
zam dd/apt-get/zellij, brap/bsd openssl) trace back to exactly that pattern
combined with the Nushell external-redirect footgun (`^cmd ... 2>/dev/null`
being passed as a literal argv token instead of a real redirect).

## Files

- `lint.nu` — the scanner. `def main [path: string, --json, --fix-suggestions]`.
- `tests/lint-test.nu` — fixture-based positive/negative test for every rule.
  Run with `nu tests/lint-test.nu` from the skill directory.
