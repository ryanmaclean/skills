#!/usr/bin/env nu
# Fixture-based positive/negative test for every rule in ../lint.nu.
# Run: nu tests/lint-test.nu   (from the skill directory, or with an absolute path)
#
# For each rule id we write one fixture file containing a POSITIVE line
# (must be flagged) and a NEGATIVE line (must NOT be flagged), run the lint
# against it, and assert the finding set matches expectations exactly.

const SELF_PATH = (path self)

def lint-path [] {
  ($SELF_PATH | path dirname | path join '..' 'lint.nu' | path expand)
}

# {id, positive, negative, ext}
def cases [] {
  [
    {id: "gnu-sed-inplace-no-suffix", positive: "sed -i -e 's/a/b/' file.txt", negative: "sed -i '' -e 's/a/b/' file.txt", ext: "sh"}
    {id: "mktemp-suffix-after-x-run", positive: "mktemp /tmp/name.XXXXXX.tmp", negative: "mktemp /tmp/name.XXXXXX", ext: "sh"}
    {id: "mktemp-t-no-trailing-x", positive: "mktemp -d -t NAME", negative: "mktemp -d -t NAME.XXXXXX", ext: "sh"}
    {id: "gnu-stat-dash-c", positive: "stat -c %s file", negative: "stat -f '%z' file", ext: "sh"}
    {id: "bsd-stat-dash-f-on-non-bsd", positive: "stat -f '%z' file", negative: "stat -c %s file", ext: "sh"}
    {id: "gnu-date-dash-d", positive: "date -d '1 day ago'", negative: "date -j -f '%Y' 2026 +%s", ext: "sh"}
    {id: "gnu-realpath-dash-m", positive: "realpath -m /nonexistent", negative: "realpath /existing/path", ext: "sh"}
    {id: "timeout-no-macos-fallback", positive: "timeout 30 sleep 5", negative: "gtimeout 30 sleep 5", ext: "sh"}
    {id: "gnu-nproc", positive: "echo $(nproc)", negative: "getconf _NPROCESSORS_ONLN", ext: "sh"}
    {id: "gnu-sha256sum-unguarded", positive: "sha256sum file.txt", negative: "sha256sum file.txt || shasum -a 256 file.txt", ext: "sh"}
    {id: "gnu-cp-reflink", positive: "cp --reflink=auto a b", negative: "cp -R a b", ext: "sh"}
    {id: "bash-shebang-hardcoded", positive: "#!/bin/bash", negative: "#!/usr/bin/env bash", ext: "sh"}
    {id: "grep-perl-regex", positive: "grep -P 'foo' file", negative: "grep -E 'foo' file", ext: "sh"}
    {id: "find-printf", positive: "find . -printf '%p\\n'", negative: "find . -print0", ext: "sh"}
    {id: "gnu-tar-long-opts", positive: "tar --transform 's/a/b/' -cf out.tar .", negative: "tar -s /a/b/ -cf out.tar .", ext: "sh"}
    {id: "du-gnu-only-flags", positive: "du -sb .", negative: "du -d 1 .", ext: "sh"}
    {id: "df-gnu-vs-bsd-flags", positive: "df -g /", negative: "df -k /", ext: "sh"}
    {id: "nushell-external-literal-redirect", positive: "^git log 2>/dev/null", negative: "^git log e> /dev/null", ext: "nu"}
    {id: "env-dash-u-not-posix", positive: "env -u FOO cmd", negative: "hide-env FOO", ext: "sh"}
    {id: "which-not-posix", positive: "which foo", negative: "command -v foo", ext: "sh"}
    {id: "sha256-bsd-flag", positive: "sha256 -q file.txt", negative: "sha256 -q file.txt || echo fallback shasum", ext: "sh"}
  ]
}

def run-lint [file: string] {
  let result = (do { ^nu (lint-path) $file } | complete)
  ($result.stdout | from json)
}

def main [] {
  let tmp = (mktemp -d)
  mut failures = []
  mut total = 0

  for c in (cases) {
    $total += 1
    # positive fixture: must be flagged with this rule id
    let pos_file = ($tmp | path join $"pos_($c.id).($c.ext)")
    (if $c.ext == "nu" { $"def main [] {\n  ($c.positive)\n}\n" } else { $"#!/bin/sh\n($c.positive)\n" }) | save -f $pos_file
    let pos_out = (run-lint $pos_file)
    let pos_rules = ($pos_out.findings | get rule)
    if not ($c.id in $pos_rules) {
      $failures = ($failures | append $"POSITIVE MISS  [($c.id)]  line '($c.positive)' was not flagged. Findings: ($pos_rules)")
    }

    # negative fixture: must NOT be flagged with this rule id
    let neg_file = ($tmp | path join $"neg_($c.id).($c.ext)")
    (if $c.ext == "nu" { $"def main [] {\n  ($c.negative)\n}\n" } else { $"#!/bin/sh\n($c.negative)\n" }) | save -f $neg_file
    let neg_out = (run-lint $neg_file)
    let neg_rules = ($neg_out.findings | get rule)
    if ($c.id in $neg_rules) {
      $failures = ($failures | append $"NEGATIVE FALSE-POSITIVE  [($c.id)]  line '($c.negative)' was incorrectly flagged.")
    }
  }

  # YAML run-block scoping test
  let yml_file = ($tmp | path join "scope.yml")
  "jobs:\n  build:\n    steps:\n      - run: |\n          stat -c %s file.txt\n      - env:\n          FOO: \"stat -c %s should not match\"\n" | save -f $yml_file
  let yml_out = (run-lint $yml_file)
  $total += 1
  if ($yml_out.count != 1) {
    $failures = ($failures | append $"YAML SCOPING  expected exactly 1 finding \(inside run: block\), got ($yml_out.count)")
  }

  # Directory-recursion smoke test
  $total += 1
  let dir_out = (run-lint $tmp)
  let ncases = (cases | length)
  if ($dir_out.count < $ncases) {
    $failures = ($failures | append $"DIRECTORY SCAN  expected at least ($ncases) findings scanning ($tmp), got ($dir_out.count)")
  }

  rm -rf $tmp

  print $"Ran ($total) checks across ($ncases) rules."
  if ($failures | length) > 0 {
    print $"($failures | length) FAILURE\(S\):"
    for f in $failures { print $"  - ($f)" }
    exit 1
  } else {
    print "All bsd-portability lint rules verified (positive + negative) on nu (version)."
    exit 0
  }
}
