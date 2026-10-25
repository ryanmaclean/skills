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

