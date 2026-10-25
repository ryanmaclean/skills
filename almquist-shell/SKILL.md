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

## 5. POSIX sh Features That DO Work

Everything in this section is safe on all ash variants (FreeBSD sh, NetBSD sh,
dash, TinyBSD).

### Pipes and redirections
```sh
cmd1 | cmd2 | cmd3
cmd > file 2>&1
cmd >> file
cmd 2> /dev/null
exec 3> logfile; echo "msg" >&3; exec 3>&-
```

### Here-documents
```sh
cat << 'EOF'
literal text, no expansion
EOF

cat << EOF
Expanded: $HOME
EOF

# Strip leading tabs with <<-:
cat <<- EOF
	indented content
EOF
```

### Parameter expansion
```sh
${var}              # simple
${var:-default}     # use default if unset or null
${var:=default}     # assign default if unset or null
${var:?message}     # error and exit if unset or null
${var:+other}       # use other if set and non-null

${#var}             # string length

${var#pattern}      # remove smallest prefix match
${var##pattern}     # remove largest prefix match
${var%pattern}      # remove smallest suffix match
${var%%pattern}     # remove largest suffix match

# Examples:
file="/path/to/foo.txt"
echo "${file##*/}"   # foo.txt  (basename)
echo "${file%.*}"    # /path/to/foo  (strip extension)
echo "${file%/*}"    # /path/to  (dirname)
```

### Command substitution
```sh
result=$(command)          # preferred form
result=`command`           # legacy backtick form (avoid nesting)
nested=$(outer $(inner))   # $() nests cleanly
```

### Arithmetic
```sh
n=$((2 + 3))
n=$(( n * 4 ))
n=$(( n % 7 ))

# Comparison in arithmetic context:
if [ $((a > b)) -eq 1 ]; then ...
# Better: use -lt/-gt/-eq in [ ]:
if [ "$a" -gt "$b" ]; then ...
```

### Conditionals
```sh
if [ "$x" = "foo" ]; then
  echo "match"
elif [ "$x" = "bar" ]; then
  echo "bar"
else
  echo "other"
fi

case "$x" in
  foo)    echo "foo" ;;
  bar|baz) echo "bar or baz" ;;
  *.txt)  echo "text file" ;;
  *)      echo "default" ;;
esac
```

### test / [ ] operators
```sh
# String tests:
[ -z "$s" ]         # empty string
[ -n "$s" ]         # non-empty string
[ "$a" = "$b" ]     # string equal (note: = not ==)
[ "$a" != "$b" ]    # string not equal

# Numeric tests:
[ "$a" -eq "$b" ]   # equal
[ "$a" -ne "$b" ]   # not equal
[ "$a" -lt "$b" ]   # less than
[ "$a" -le "$b" ]   # less than or equal
[ "$a" -gt "$b" ]   # greater than
[ "$a" -ge "$b" ]   # greater than or equal

# File tests:
[ -f "$f" ]         # regular file exists
[ -d "$d" ]         # directory exists
[ -e "$p" ]         # path exists (any type)
[ -r "$f" ]         # readable
[ -w "$f" ]         # writable
[ -x "$f" ]         # executable
[ -s "$f" ]         # non-empty file

# Logical:
[ "$a" = "x" ] && [ "$b" = "y" ]  # AND (portable)
[ "$a" = "x" ] || [ "$b" = "y" ]  # OR  (portable)
! [ "$a" = "x" ]                   # NOT
```

### Loops
```sh
# for — iterate list:
for item in alpha beta gamma; do
  echo "$item"
done

# for — iterate positional params:
for arg; do
  echo "$arg"
done

# while:
while [ "$count" -lt 10 ]; do
  count=$((count + 1))
done

# until:
until ping -c1 -W1 host >/dev/null 2>&1; do
  sleep 5
done

# Iterate lines of a file:
while IFS= read -r line; do
  echo "line: $line"
done < /etc/hosts
```

### Functions
```sh
# Define:
myfunc() {
  local x="$1"      # local is supported in ash
  echo "arg: $x"
  return 0
}

# Call:
myfunc "hello"
status=$?

# Functions see and can modify parent-scope variables (dynamic scoping)
# Use local to limit scope
```

### Traps
```sh
# Cleanup on exit:
cleanup() { rm -f "$tmpfile"; }
trap cleanup EXIT

# Catch signals:
trap 'echo interrupted; exit 1' INT TERM

# Ignore signal:
trap '' HUP

# Reset to default:
trap - INT

# List signals: trap -l
```

### set options for robust scripts
```sh
set -e          # exit on error (errexit)
set -u          # error on unset variable (nounset)
set -o pipefail # pipeline fails if any stage fails (FreeBSD sh, dash)

# Combined (common idiom):
set -euo pipefail

# NOTE: NetBSD sh may not support pipefail — check target:
# FreeBSD sh: pipefail supported
# NetBSD sh:  pipefail NOT supported (use set -e only)
# dash:       pipefail supported
```

**NetBSD sh pipefail caveat:** For scripts targeting `nbrpi3xx` or `nb-vm-xx`,
omit `pipefail` or guard with a capability check. Use `set -eu` only for
maximum portability.

## 6. UTF-8 and Unicode Support

### FreeBSD sh
- Handles UTF-8 natively with `LANG=C.UTF-8` or `LANG=en_US.UTF-8`
- Character class patterns (`[:alpha:]`, etc.) are locale-aware
- IFS field splitting is UTF-8 aware

### BMP Unicode in filenames — tested safe on FreeBSD/NetBSD ash
```sh
# These BMP characters work correctly in filenames and script output:
# ※ (U+203B) — decision marker
# ∴ (U+2234) — merge-ready marker
# ⚀–⚅ (U+2680–U+2685) — priority dice tokens
# ♠♥♦♣ (U+2660–U+2663) — suit/team tokens
touch "task-⚀-critical"
ls -1 | grep '^task-'
```

### SMP characters (U+10000+) — AVOID in filenames
Per project `CLAUDE.md`: 4-byte UTF-8 (playing card glyphs, etc.) causes
breakage in bash, zsh, find, and Rust tooling. Use BMP `token` for filenames,
SMP `glyph` only for rich display surfaces (SwiftUI, web, terminal emulator
title where confirmed safe).

```sh
# SAFE for filenames (BMP):
file="report-⚀.txt"
file="status-※.txt"

# AVOID in filenames (SMP — 4-byte UTF-8):
# file="card-🂡.txt"   # breaks tooling
```

### Dollar-quote C-style escapes (FreeBSD sh extension)
```sh
# Unicode by code point in string literals:
msg=$'\u2234 merge ready'   # ∴ merge ready
sep=$'\u2500\u2500\u2500'   # ─── (box drawing)
```

## 7. Fleet Script Patterns

### Portable shebang
```sh
#!/bin/sh
# Always use /bin/sh — never /bin/bash on BSD fleet nodes
```

### Standard error-safe header
```sh
#!/bin/sh
set -eu
# For FreeBSD sh targets only — add pipefail:
# set -euo pipefail
```

### Temp files
```sh
tmpfile=$(mktemp /tmp/myscript.XXXXXX)
trap 'rm -f "$tmpfile"' EXIT

# mktemp is available on FreeBSD, NetBSD, and dash systems
```

### Logging helper
```sh
log() { printf '[%s] %s\n' "$(date '+%Y-%m-%dT%H:%M:%S')" "$*" >&2; }
die() { log "ERROR: $*"; exit 1; }

log "Starting deployment"
[ -f /etc/rc.conf ] || die "/etc/rc.conf not found"
```

### Remote execution via Ansible raw module
```sh
# In playbook YAML:
- name: Run script on BSD host
  raw: /bin/sh /path/to/script.sh
  # Do NOT use shell: module with bash-specific syntax
```

### getopts option parsing
```sh
#!/bin/sh
usage() { printf 'Usage: %s [-v] [-o file] arg\n' "$0" >&2; exit 1; }

verbose=0
outfile=""
while getopts 'vo:h' opt; do
  case "$opt" in
    v) verbose=1 ;;
    o) outfile="$OPTARG" ;;
    h) usage ;;
    ?) usage ;;
  esac
done
shift $((OPTIND - 1))
```

### rc.d service script (FreeBSD)
```sh
#!/bin/sh

# PROVIDE: myservice
# REQUIRE: NETWORKING
# KEYWORD: shutdown

. /etc/rc.subr

name="myservice"
rcvar="myservice_enable"
command="/usr/local/sbin/myserviced"
command_args="--config /usr/local/etc/myservice.conf"
pidfile="/var/run/${name}.pid"

load_rc_config "$name"
run_rc_command "$1"
```

Enable/disable:
```sh
sysrc myservice_enable=YES
service myservice start
service myservice status
```

### rc.d service script (NetBSD)
```sh
#!/bin/sh

# PROVIDE: myservice
# REQUIRE: NETWORKING

$_rc_subr_loaded . /etc/rc.subr

name="myservice"
rcvar=$name
command="/usr/pkg/sbin/myserviced"

load_rc_config $name
run_rc_command "$1"
```

Enable in `/etc/rc.conf`:
```sh
myservice=YES
```

### Checking installed packages portably
```sh
# FreeBSD:
pkg info -e python3 && echo "installed" || echo "not installed"

# NetBSD (pkgsrc):
pkgin list | grep -q "^python3" && echo "installed" || echo "not installed"
```

## 8. Why NOT bash on Fleet Nodes

| Reason | Detail |
|--------|--------|
| **GPL-3.0 license** | bash is GPL-3.0 — copyleft, violates project license policy (MIT/BSD/Apache-2.0 only) |
| **Not in FreeBSD base** | bash requires `pkg install bash` (ports); not present on minimal/TinyBSD installs |
| **Not in NetBSD base** | bash requires `pkgin install bash`; not present on base installs |
| **TinyBSD contract** | TinyBSD v1 defaults: zero added packages, `/bin/sh` only |
| **Size and speed** | ash is ~92–200 KB; bash is several MB. On Raspberry Pi Zero/1, matters |
| **Complexity** | bash bashisms creep into scripts and break portability silently |
| **Ansible default** | Ansible `shell:` module defaults to `/bin/sh`; bash must be explicitly forced |

### ash is sufficient
Everything needed for fleet automation — service control, config templating,
health checks, deploy hooks — is expressible in POSIX sh. The constraints
are features: they enforce portable, auditable scripts.

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
