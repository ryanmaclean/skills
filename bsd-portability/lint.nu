#!/usr/bin/env nu
# bsd-portability lint: scan shell/Makefile/CI/Dockerfile/expect/nu-external-call
# surfaces for GNU-coreutils-only or BSD-only syntax that breaks across the
# fleet's mixed FreeBSD/NetBSD/OpenBSD/macOS/Linux/Windows/Alpine targets.
#
# Rule table: each rule is {id, pattern, message, fix}. `pattern` is a Rust
# `regex`-crate pattern (no lookaround support) used for the generic path;
# four rules that genuinely need lookaround in their spec are matched by a
# dedicated custom-logic function instead (see `custom-match`).

def rules [] {
  [
    {id: "gnu-sed-inplace-no-suffix", pattern: 'sed\s+-i\s+', message: "sed -i with no explicit backup suffix is GNU/uutils syntax; BSD/macOS sed requires a suffix argument (even an empty '' one), and OpenBSD/toybox sed parse -i differently again.", fix: "Use sed -i '' -e '...' for BSD/macOS, branch on uname, or avoid sed -i entirely and do the edit via a temp file + mv or Nushell's 'str replace | save'."}
    {id: "mktemp-suffix-after-x-run", pattern: 'mktemp\s+(-d\s+)?[''"]?[^\s''"]*XXXXXX\.[A-Za-z0-9]+', message: "A suffix placed after the XXXXXX run in a mktemp template is only randomized by GNU/uutils mktemp; BSD, macOS, toybox and busybox mktemp replace only the trailing X's, so the same literal filename is reused every run.", fix: "Put the XXXXXX at the very end of the template (mktemp /tmp/name.XXXXXX), or use Nushell's built-in mktemp, which handles this portably."}
    {id: "mktemp-t-no-trailing-x", pattern: 'mktemp\s+-d?\s*-t\s+[A-Za-z0-9_.-]+', message: "mktemp -d -t NAME with no XXXXXX is rejected by GNU/uutils mktemp ('too few X's in template') and only works on BSD/macOS, where -t treats the argument as a prefix rather than a template.", fix: "Always include a literal XXXXXX in the template: mktemp -d -t NAME.XXXXXX, which both GNU and BSD mktemp accept."}
    {id: "gnu-stat-dash-c", pattern: '\bstat\s+-c\b', message: "stat -c is GNU/uutils/toybox syntax; FreeBSD, NetBSD, OpenBSD and macOS stat use -f with a different format string, and branching only on 'is this macOS' misses every other BSD.", fix: "Branch on linux vs not-linux (not just macos), or replace stat entirely with a Nushell 'ls' pipeline (ls -l | get size/modified), which works identically everywhere."}
    {id: "bsd-stat-dash-f-on-non-bsd", pattern: 'stat\s+-f\s*[''"]?%', message: "stat -f is BSD/macOS syntax; on GNU/uutils/toybox stat, -f means '--file-system' and silently returns filesystem info instead of file info.", fix: "Detect the OS first and use stat -c on GNU/Linux hosts, or replace the call with a portable Nushell 'ls' pipeline."}
    {id: "gnu-date-dash-d", pattern: 'date\s+-d\s', message: "date -d (parse an arbitrary date string) is a GNU/uutils/toybox/NetBSD extension; FreeBSD, macOS and OpenBSD date have no -d and instead need -j -f INFMT STRING +OUTFMT.", fix: "Use Nushell's built-in 'date' for parsing/formatting, or branch explicitly: BSD/macOS -> date -j -f ..., GNU/NetBSD -> date -d ..., with no silent catch-all fallback that maps failure to a bogus value."}
    {id: "gnu-realpath-dash-m", pattern: 'realpath\s+-m\b', message: "realpath -m is GNU/uutils-only; FreeBSD and macOS realpath have no -m and exit non-zero on a non-existent path, which a trailing '|| echo' fallback can mask instead of actually handling.", fix: "Use Nushell's 'path expand', or implement the missing-path-tolerant behavior explicitly rather than relying on -m."}
    {id: "timeout-no-macos-fallback", pattern: '\btimeout\s+\d+[a-zA-Z]*\s+\S', message: "timeout(1) is not in macOS base (only via Homebrew's GNU coreutils as gtimeout) and is absent from OpenBSD/NetBSD<11 base; calling it unconditionally fails on those hosts.", fix: "Probe for timeout, fall back to gtimeout on macOS, and for hosts with neither, implement a watchdog in Nushell (spawn the job, sleep, kill on timeout)."}
    {id: "gnu-nproc", pattern: '\$\(nproc\)|\bnproc\b', message: "nproc is GNU coreutils/uutils/toybox; it is absent from macOS, OpenBSD and NetBSD base (FreeBSD 13.2+ has it).", fix: "Use Nushell's `(sys cpu | length)`, or `getconf _NPROCESSORS_ONLN`, both available on every target OS in this fleet."}
    {id: "gnu-sha256sum-unguarded", pattern: '\bsha256sum\b', message: "sha256sum (GNU coreutils/uutils/FreeBSD-14+) does not exist on macOS (shasum -a 256), OpenBSD (sha256) or NetBSD (cksum -a sha256) and is being called with no fallback chain.", fix: "Use Nushell's built-in 'hash sha256', or chain sha256sum -> shasum -a 256 -> sha256 -q as a fallback probe (already done correctly in several brap/zam scripts — copy that pattern)."}
    {id: "gnu-cp-reflink", pattern: 'cp\s+.*--reflink', message: "cp --reflink is GNU coreutils/uutils syntax; BSD/macOS cp has no such flag (macOS uses the unrelated 'cp -c' for APFS clonefile).", fix: "Branch by OS: macOS -> cp -c, Linux -> cp --reflink=auto with a plain 'cp -R' fallback, everything else -> plain cp."}
    {id: "bash-shebang-hardcoded", pattern: '^#!/bin/bash\b', message: "/bin/bash does not exist on FreeBSD, NetBSD, OpenBSD or Alpine (bash lives at /usr/local/bin/bash, /usr/pkg/bin/bash, or is entirely absent); this shebang gives 'bad interpreter' there.", fix: "Use '#!/usr/bin/env bash' at minimum, or per house style rewrite the script as Nushell (.nu) with 'def main [...]', which is already the standard for new scripts."}
    {id: "grep-perl-regex", pattern: 'grep\s+.*-[A-Za-z]*P[A-Za-z]*\b', message: "grep -P (PCRE) is GNU-only (and 'experimental' even there); it's missing from BSD/macOS grep, uutils grep and busybox/toybox grep, and appears to work on some hosts only because ugrep happens to shadow the system grep on PATH.", fix: "Rewrite the pattern with portable -E extended regex, or do the extraction in Nushell / awk instead of grep -P."}
    {id: "find-printf", pattern: 'find\s+.*-printf\b', message: "find -printf is a GNU extension; it is not implemented by BSD find or (per current evidence) uutils findutils, and wrapping it in '2>/dev/null || true' silently turns the failure into an empty result set instead of surfacing it.", fix: "Use find -print0 followed by a stat/ls pass for the fields you need, or a Nushell 'ls **/*' glob piped through 'get modified/size'."}
    {id: "gnu-tar-long-opts", pattern: 'tar\s+.*--(transform|xform|sort=name|owner=0|group=0|numeric-owner)\b', message: "--transform/--xform/--sort=name/--owner=0/--group=0/--numeric-owner are GNU-tar-only long options; bsdtar (the default tar on macOS, FreeBSD and NetBSD) rejects them outright.", fix: "Use bsdtar's '-s /old/new/' for renaming, drop reproducible-archive-only flags on bsdtar-only hosts, or use a small Nushell/Rust archiver for that one job."}
    {id: "du-gnu-only-flags", pattern: 'du\s+.*(-sb\b|--max-depth)', message: "du -sb (byte-granularity output) and --max-depth are GNU-only; BSD/macOS du has no -b and spells depth as -d N.", fix: "Use 'du -sk' (always available) and multiply by 1024 for bytes, or use 'du -d N', which both GNU and BSD accept."}
    {id: "df-gnu-vs-bsd-flags", pattern: 'df\s+-g\b', message: "df -g (gigabyte output) is BSD-only; GNU/uutils/busybox df has no -g and errors on it.", fix: "Use 'df -k' or 'df -m' (portable everywhere) and convert units yourself, or parse Nushell's 'sys disks' output instead of shelling to df."}
    {id: "nushell-external-literal-redirect", pattern: '\^[a-zA-Z0-9_./-]+.*2>/dev/null', message: "Inside Nushell, a bash-style '2>/dev/null' after an external (^cmd) call is passed through as a literal argv token rather than interpreted as a redirect, so the target command receives a bogus extra argument instead of having stderr suppressed. This exact bug has recurred independently across smolBSD (git), zam (dd, apt-get, zellij), and brap/bsd (openssl).", fix: "Use Nushell's own redirection syntax: '^cmd ... e> /dev/null' (or 'o> file e> file2' for both streams), never a shell-style string."}
    {id: "env-dash-u-not-posix", pattern: '\benv\s+-u\s', message: "env -u NAME is not POSIX; GNU/uutils/toybox/busybox/FreeBSD/macOS support it, but OpenBSD's base env(1) implements only -i.", fix: "Use Nushell's 'hide-env NAME' when calling from a .nu script, or 'unset NAME; exec cmd' inside a POSIX sh wrapper."}
    {id: "which-not-posix", pattern: '\bwhich\s+[\w.-]+', message: "which(1) is not POSIX, is absent on stock Windows, and its output/exit-status details differ across debianutils, busybox and BSD which implementations.", fix: "Use 'command -v NAME' (POSIX; works under sh/bash/dash/ash) or Nushell's built-in 'which', which already behaves portably."}
    {id: "sha256-bsd-flag", pattern: '\bsha256\s+-q\b', message: "sha256 -q is FreeBSD/macOS-md5-family syntax with no fallback to a GNU/Linux host, where the sha256(1) binary doesn't exist at all.", fix: "Chain the check: try sha256sum, then shasum -a 256, then sha256 -q — or just use Nushell's built-in 'hash sha256' everywhere and drop the external tool entirely."}
  ]
}

# Rules whose upstream spec needs lookaround (unsupported by the regex crate
# nu's =~ uses) get a dedicated matcher here instead of the generic path.
def custom-match [rule_id: string, line: string]: nothing -> bool {
  if $rule_id == "gnu-sed-inplace-no-suffix" {
    if not ($line =~ 'sed\s+-i\s+') { return false }
    let m = ($line | parse -r 'sed\s+-i\s*(?P<suf>\S*)')
    if ($m | is-empty) { return true }
    let suf = ($m | get 0.suf)
    return (not (($suf | str starts-with '.') or ($suf == "''") or ($suf == '""')))
  }
  if $rule_id == "mktemp-t-no-trailing-x" {
    return (($line =~ 'mktemp\s+-d?\s*-t\s+[A-Za-z0-9_.-]+') and not ($line =~ 'XXXXXX'))
  }
  if $rule_id == "gnu-sha256sum-unguarded" {
    if not ($line =~ '\bsha256sum\b') { return false }
    let idx = ($line | str index-of 'sha256sum')
    if $idx < 0 { return false }
    let before_start = (if $idx > 40 { $idx - 40 } else { 0 })
    let before_win = ($line | str substring $before_start..$idx)
    let llen = ($line | str length)
    let after_end = ([($idx + 9 + 60) $llen] | math min)
    let after_win = ($line | str substring ($idx + 9)..$after_end)
    return (not (($before_win | str contains '||') or ($after_win | str contains '||')))
  }
  if $rule_id == "sha256-bsd-flag" {
    if not ($line =~ '\bsha256\s+-q\b') { return false }
    let idx = ($line | str index-of 'sha256')
    if $idx < 0 { return false }
    let llen = ($line | str length)
    let after_end = ([($idx + 80) $llen] | math min)
    let after_win = ($line | str substring $idx..$after_end)
    return (not (($after_win | str contains 'shasum') or ($after_win | str contains 'sha256sum')))
  }
  false
}

const CUSTOM_RULE_IDS = ["gnu-sed-inplace-no-suffix", "mktemp-t-no-trailing-x", "gnu-sha256sum-unguarded", "sha256-bsd-flag"]

def rule-matches [rule: record, line: string]: nothing -> bool {
  if ($rule.id in $CUSTOM_RULE_IDS) {
    custom-match $rule.id $line
  } else {
    $line =~ $rule.pattern
  }
}

def is-target-file [f: string]: nothing -> bool {
  let base = ($f | path basename)
  (($base | str ends-with '.sh')
    or ($base | str starts-with 'Makefile')
    or ($base | str ends-with '.yml')
    or ($base | str ends-with '.yaml')
    or ($base | str starts-with 'Dockerfile')
    or ($base | str ends-with '.exp')
    or ($base | str ends-with '.nu'))
}

def gather-files [path: string]: nothing -> list<string> {
  let abs = ($path | path expand)
  if ($abs | path type) == "file" {
    [$abs]
  } else {
    (cd $abs; glob **/* --exclude [
        **/.git/** **/target/** **/node_modules/** **/.jj/**
        **/vendor/** **/.venv/** **/venv/** **/__pycache__/**
        **/dist/** **/build/** **/.next/** **/third_party/**
      ]
      | where {|f| ($f | path type) == "file" }
      | where {|f| is-target-file $f })
  }
}

# Returns true if this line should be checked, given running yaml run-block state.
def yaml-line-in-scope [line: string, in_run: bool, run_indent: int]: nothing -> record {
  # Detect a `run:` key (possibly with a block scalar | or >) and track indentation
  # of subsequent lines belonging to that block. A single-line `run: cmd` is always
  # in scope for that one line.
  let raw_trimmed = ($line | str trim)
  let indent = (($line | str length) - ($line | str trim --left | str length))
  # A YAML sequence item ("- run: |") prefixes the key with a list marker;
  # strip it for key detection but track the key's own column for indent math.
  let is_seq_item = ($raw_trimmed | str starts-with '- ')
  let trimmed = (if $is_seq_item { ($raw_trimmed | str substring 2.. | str trim --left) } else { $raw_trimmed })
  let key_indent = (if $is_seq_item { $indent + (($raw_trimmed | str length) - ($trimmed | str length)) } else { $indent })

  if $in_run {
    if ($trimmed == "") {
      return {scope: true, in_run: true, run_indent: $run_indent}
    }
    if $indent > $run_indent {
      return {scope: true, in_run: true, run_indent: $run_indent}
    } else {
      # block ended; fall through to re-check this line for a new `run:` key
      return (yaml-line-in-scope $line false 0)
    }
  } else {
    if ($trimmed =~ '^run:\s*[|>]') {
      return {scope: false, in_run: true, run_indent: $key_indent}
    }
    if ($trimmed =~ '^run:\s*\S') {
      return {scope: true, in_run: false, run_indent: 0}
    }
    return {scope: false, in_run: false, run_indent: 0}
  }
}

def main [
  path: string          # file or directory to scan
  --json                # emit JSON (default behavior; kept for CLI explicitness)
  --fix-suggestions     # include fix suggestions (default behavior; kept for CLI explicitness)
] {
  let all_rules = (rules)
  let files = (gather-files $path)
  mut findings = []

  for f in $files {
    let content = (try { open --raw $f } catch { "" })
    if (($content | describe) != "string") or (($content | str length) == 0) {
      continue
    }
    let file_lines = ($content | lines)
    let is_yaml = (($f | str ends-with '.yml') or ($f | str ends-with '.yaml'))
    let is_nu = ($f | str ends-with '.nu')

    mut in_run = false
    mut run_indent = 0

    for pair in ($file_lines | enumerate) {
      let idx = $pair.index
      let line = $pair.item
      let lineno = $idx + 1

      mut in_scope = true
      if $is_yaml {
        let st = (yaml-line-in-scope $line $in_run $run_indent)
        $in_scope = $st.scope
        $in_run = $st.in_run
        $run_indent = $st.run_indent
      }
      if $is_nu {
        $in_scope = ($line | str contains '^')
      }

      if not $in_scope {
        continue
      }

      for r in $all_rules {
        if (rule-matches $r $line) {
          $findings = ($findings | append {
            file: $f
            line: $lineno
            rule: $r.id
            snippet: ($line | str trim)
            fix: $r.fix
          })
        }
      }
    }
  }

  let out = {
    schema_version: "v1"
    path: $path
    findings: $findings
    count: ($findings | length)
  }
  print ($out | to json --raw)
  if ($findings | length) > 0 {
    exit 1
  }
}
