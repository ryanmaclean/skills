---
name: almquist-shell
description: >
  Reference for writing portable /bin/sh scripts targeting ash (Almquist shell) —
  the default shell on FreeBSD, NetBSD, and TinyBSD fleet nodes. Covers POSIX
  features, bash incompatibilities, fleet patterns, rc.d service scripts, and
  UTF-8/Unicode filename safety rules.
  Trigger phrases: ash, almquist, /bin/sh, posix shell, freebsd shell,
  sh scripting, dash, tinybsd shell, netbsd shell, portable shell.
---

# Almquist Shell (ash / /bin/sh) — Fleet Reference

## 1. History and Lineage

Kenneth Almquist wrote ash in the late 1980s as a clean reimplementation of the
UNIX System V shell. The first public release was posted to Usenet on
**30 May 1989**. Almquist's stated goals: fast, small, and POSIX-compatible —
original binary was 92 KB.

**BSD adoption timeline:**

| Year | Event |
|------|-------|
| 1989 | Almquist posts ash to Usenet (comp.sources.unix) |
| ~1993 | ash replaces the Bourne shell as `/bin/sh` on FreeBSD and NetBSD |
| 1997 | Herbert Xu ports NetBSD ash to Debian Linux |
| 2002 | Debian port renamed **dash** (Debian Almquist shell) |
| 2006 | Ubuntu adopts dash as `/bin/sh` for faster boot scripting |
| 2011 | Debian 6 makes dash the default `/bin/sh` |

**Variants in active use:**

- `FreeBSD sh` — extended ash in FreeBSD base, supports `pipefail`, `$'...'` quoting
- `NetBSD sh` — ash in NetBSD base, strictly POSIX-oriented
- `dash` — Debian/Ubuntu `/bin/sh`, smallest and strictest
- `busybox sh` — embedded systems; limited subset

## 2. License

FreeBSD sh (and the original ash) is licensed under the **3-clause BSD license**.
This is explicitly compatible with the project's MIT/BSD/Apache-2.0 policy.

Contrast with **bash**, which is GPL-3.0 — copyleft, incompatible with project
license policy, and therefore **not allowed** in fleet base scripts.

## 3. Where It Runs in Our Fleet

| Host group | Platform | Shell binary | Notes |
|------------|----------|-------------|-------|
| `fbrpi4xx` | FreeBSD on Raspberry Pi 4 | `/bin/sh` (FreeBSD sh) | Full `pipefail` support |
| `nbrpi3xx` | NetBSD on Raspberry Pi 3 | `/bin/sh` (NetBSD sh) | POSIX-strict |
| `nb-vm-xx` | NetBSD VMs (QEMU) | `/bin/sh` (NetBSD sh) | POSIX-strict |
| TinyBSD nodes | TinyBSD base system | `/bin/sh` (ash-derived) | Zero extra deps |

**TinyBSD v1 defaults contract:** shell = `/bin/sh`, no bash, no zsh, no additional
shell packages. Scripts for TinyBSD MUST be pure POSIX sh with no bashisms.

**Ansible remote execution** against these hosts uses `raw` module or
`ansible_shell_executable: /bin/sh`. Never set `ansible_shell_executable: /bin/bash`
on BSD fleet nodes — bash requires ports/pkg install.

## 4. Key Differences from bash — What DOES NOT Work

These constructs are bash-specific and will fail silently or fatally on ash:

### Arrays
```sh
# BASH ONLY — will fail on ash:
arr=(one two three)
echo "${arr[1]}"

# ASH portable alternatives:
set -- one two three          # use positional parameters
echo "$2"                     # second element

# or IFS trick to iterate:
items="one two three"
for item in $items; do echo "$item"; done
```

### [[ ]] double-bracket conditionals
```sh
# BASH ONLY:
if [[ "$x" == foo* ]]; then ...

# ASH — use [ ] or case:
if [ "$x" = "foo" ]; then ...
case "$x" in foo*) ...; esac
```

### $RANDOM
```sh
# BASH ONLY:
n=$RANDOM

# ASH portable:
n=$(od -An -N2 -tu2 /dev/urandom | tr -d ' ')
# or: awk 'BEGIN{srand(); print int(rand()*32768)}'
```

### $PIPESTATUS
```sh
# BASH ONLY:
cmd1 | cmd2
echo "${PIPESTATUS[0]}"  # exit status of cmd1

# ASH with pipefail (FreeBSD sh only):
set -o pipefail
cmd1 | cmd2    # pipeline fails if any stage fails

# Fully portable: use temp files or process substitution workaround
```

### Process substitution
```sh
# BASH ONLY:
diff <(cmd1) <(cmd2)

# ASH portable:
cmd1 > /tmp/a.$$ && cmd2 > /tmp/b.$$
diff /tmp/a.$$ /tmp/b.$$
rm -f /tmp/a.$$ /tmp/b.$$
```

### Here-strings
```sh
# BASH ONLY:
read var <<< "value"

# ASH portable:
var=$(printf '%s' "value")
# or:
read var << 'EOF'
value
EOF
```

### Brace expansion
```sh
# BASH ONLY:
for i in {1..10}; do ...

# ASH portable:
i=1; while [ "$i" -le 10 ]; do
  echo "$i"
  i=$((i + 1))
done
```

### function keyword
```sh
# BASH ONLY:
function myfunc() { echo hi; }

# ASH — omit keyword, parentheses only:
myfunc() { echo hi; }
```

### `echo -e` behavior
```sh
# Inconsistent across ash variants — use printf instead:
printf 'line1\nline2\n'
printf '%s\n' "$var"
```


## References

- `references/posix-features.md` — POSIX sh features that work (sections 5–6)
- `references/fleet-patterns.md` — fleet script patterns + why not bash (sections 7–8)
## 9. Quick Reference Card

```
DOES WORK (ash/POSIX sh)        DOES NOT WORK (bash-only)
─────────────────────────────   ──────────────────────────
[ ] test                        [[ ]] conditional
$()  command substitution       <() process substitution
$(( )) arithmetic               <<< here-string
${var##pat} expansion           ${arr[n]} array index
for/while/until/case            {1..10} brace expansion
name() { } functions            function name() {}
trap ... EXIT/INT/TERM          $RANDOM
set -eu pipefail (FreeBSD)      $PIPESTATUS
local inside functions          declare -a/-A arrays
printf (preferred over echo)    echo -e (unreliable)
getopts                         bash-specific getopts extensions
IFS tricks for splitting        mapfile / readarray
```

## 10. Useful External Tools (Available on BSD Base)

| Tool | Purpose | Notes |
|------|---------|-------|
| `printf` | Formatted output | Prefer over `echo` |
| `awk` | Text processing, arithmetic | One-true-awk on FreeBSD |
| `sed` | Stream editing | POSIX sed on both |
| `grep` | Pattern search | `-E` for extended regex |
| `find` | File search | POSIX `find`, no `-printf` on NetBSD |
| `mktemp` | Temp file creation | Available base system |
| `od` | Octal dump / binary read | Useful for random numbers |
| `expr` | Arithmetic (legacy) | Use `$(( ))` instead |
| `cut`, `tr` | Field/char extraction | Base system |
| `sort`, `uniq` | Sorting/dedup | Base system |

## See Also

- `freebsd` skill — FreeBSD host management, pkg, rc, pf
- `netbsd` skill — NetBSD host management, pkgsrc, rc
- `ansible-fleet` skill — fleet-wide script deployment
- `unicode-glyphs` skill — BMP/SMP token selection for filenames
