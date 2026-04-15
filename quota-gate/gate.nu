#!/usr/bin/env nu
# gate.nu — pre-delegation quota check
# Usage: nu gate.nu [--warn_at 70] [--block_at 90]

def bar [pct: int] {
    let filled = ($pct / 10)
    let empty  = (10 - $filled)
    let f = (0..$filled | each { "█" } | str join "")
    let e = (0..$empty  | each { "░" } | str join "")
    $f + $e
}

def main [
    --warn_at: int = 70
    --block_at: int = 90
] {
    let bop_bin = (
        try { (^which bop | str trim) } catch {
            # fall back to known dev build locations
            let candidates = [
                $"($env.HOME)/bop/target/debug/bop"
                $"($env.HOME)/bop/target/release/bop"
                "/usr/local/bin/bop"
            ]
            $candidates | where { |p| ($p | path exists) } | first
        }
    )
    let raw = (do { ^$bop_bin providers --json } | complete)
    if $raw.exit_code != 0 {
        print "✗ bop providers failed"; exit 1
    }
    let providers = ($raw.stdout | from json)

    print ""
    print "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    print "  QUOTA GATE — pre-delegation check"
    print "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    mut any_blocked = false
    mut any_warned  = false
    mut best_name: string = ""
    mut best_pct: int = 101

    for p in $providers {
        let pct   = ($p.primary_pct | default 0)
        let label = ($p.primary_label | default "primary")

        let reset = if $p.reset_at != null {
            try {
                let dt    = ($p.reset_at | into datetime)
                let secs  = ($dt - (date now) | into int) / 1_000_000_000
                if $secs < 0 { "reset due" } else {
                    $"in ($secs / 3600)h ($secs mod 3600 / 60)m"
                }
            } catch { "?" }
        } else { "—" }

        let symbol = if $pct >= $block_at {
            $any_blocked = true; "✗"
        } else if $pct >= $warn_at {
            $any_warned = true; "⚠"
        } else {
            if $pct < $best_pct {
                $best_name = $p.display_name
                $best_pct  = $pct
            }
            "✓"
        }

        let err = if $p.error != null { $"  \(($p.error))" } else { "" }
        print $"  ($p.display_name)  ($label)  (bar $pct)  ($pct)%  ($symbol)  ($reset)($err)"
    }

    print "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    let verdict = if $any_blocked and $best_name == "" {
        "  ✗  HOLD — all providers saturated"
    } else if $any_blocked {
        $"  ⚠  CAUTION — use ($best_name) \(($best_pct)% used)"
    } else if $any_warned {
        $"  ⚠  GO with care — ($best_name) has most headroom \(($best_pct)%)"
    } else {
        $"  ✓  GO — ($best_name) has most headroom \(($best_pct)% used)"
    }

    print $verdict
    print "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    print ""

    if $any_blocked and $best_name == "" { exit 1 }
}
