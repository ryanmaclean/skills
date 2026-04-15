#!/usr/bin/env nu
# FMEA — failure mode analysis for a proposed resolution.
# Run after `solve.nu` to validate the recommended principle.
# Script code never enters context; only output does.
#
# Usage:
#   nu fmea.nu 35                              # principle ID only
#   nu fmea.nu 35 "cache the result"           # principle + resolution text
#   nu fmea.nu 35 "cache the result" -c "CDN"  # with context
#   nu fmea.nu 35 --json                        # machine-readable output
#
# Or pipe from solve:
#   nu solve.nu "speed" "reliability" --json | nu fmea.nu --stdin

def find_bin [] {
    let local = ($env.FILE_PWD | path join "triz")
    if ($local | path exists) { $local } else { "triz" }
}

def main [
    principle?: int        # TRIZ principle ID (1-40)
    resolution?: string    # Resolution text (optional)
    --context: string = ""
    --json
    --stdin                # Read principle/resolution from solve --json output on stdin
] {
    let out_flag = if $json { ["--output" "json"] } else { [] }

    let bin = (find_bin)

    if $stdin {
        let input = $in | from json
        let p = $input.recommended_principle.id
        let r = ($input.resolution? | default "")
        if ($r | is-empty) {
            ^$bin fmea --principle $p ...$out_flag
        } else {
            ^$bin fmea --principle $p --resolution $r ...$out_flag
        }
        return
    }

    if ($principle == null) {
        error make { msg: "Provide principle ID (1-40) or use --stdin with solve --json output" }
    }

    let args = (
        ["--principle" ($principle | into string)]
        | if ($resolution != null) { append ["--resolution" $resolution] } else { $in }
        | if (not ($context | is-empty)) { append ["-c" $context] } else { $in }
        | append $out_flag
    )

    ^$bin fmea ...$args
}
