#!/usr/bin/env nu
# credscan.nu: report credential-shaped lines as `path:line  [kind]`.
# Never prints the matched text. Ported from the retired dd-port-work/credscan.py.
# Usage: nu credscan.nu <dir>...       scan directories
#        <text> | nu credscan.nu --from-stdin <label>

const PATS = [
  {k: "openai/anthropic sk-", p: '(?:^|[^A-Za-z0-9_-])sk-(?:proj-|ant-[a-z0-9]+-|svcacct-)?[A-Za-z0-9_-]{20,}'}
  {k: "aws AKIA/ASIA", p: '(?:^|[^A-Z0-9])(?:AKIA|ASIA)[0-9A-Z]{16}(?:[^0-9A-Z]|$)'}
  {k: "github token", p: '(?:^|[^A-Za-z0-9])(?:gh[pousr]_[A-Za-z0-9]{36,}|github_pat_[A-Za-z0-9_]{50,})'}
  {k: "gitlab glpat", p: 'glpat-[A-Za-z0-9_-]{20,}'}
  {k: "slack xox", p: 'xox[abprs]-[A-Za-z0-9-]{10,}'}
  {k: "google AIza", p: 'AIza[0-9A-Za-z_-]{35}'}
  {k: "private key", p: '-----BEGIN [A-Z ]*PRIVATE KEY-----'}
  {k: "gitea/token 40hex", p: '(?i)(?:gitea|tea|access)[_-]?token["\x27 ]*[:=]\s*["\x27]?[0-9a-f]{40}(?:[^0-9a-f]|$)'}
  {k: "oauth sk-ant-oat", p: 'sk-ant-oat[0-9A-Za-z_-]{10,}'}
]
const SKIP = [target .git .jj node_modules worktrees .worktrees .claude]

def scan-text [label: string, text: string] {
  $text | lines | enumerate | each {|row|
    let hit = ($PATS | where {|x| $row.item =~ $x.p } | get -o 0.k)
    if $hit != null { {where: $"($label):($row.index + 1)", kind: $hit} }
  } | compact
}

def main [--from-stdin, ...args: string] {
  let hits = if $from_stdin {
    scan-text ($args | get -o 0 | default "stdin") (open --raw /dev/stdin | decode utf-8)
  } else {
    $args | each {|root|
      glob $"($root)/**/*" --no-dir --exclude ($SKIP | each {|s| $"**/($s)/**" }) | each {|f|
        let b = (try { open --raw $f | first 8000000 } catch { null })
        if $b != null and not ($b | bytes index-of 0x[00] | $in >= 0 and $in < 4096) {
          scan-text $f ($b | decode utf-8)
        } else { [] }
      } | flatten
    } | flatten
  }
  $hits | each {|h| print $"($h.where)  [($h.kind)]" }
  null
}
