#!/usr/bin/env nu
# TRIZ contradiction oracle — calls the triz Rust binary.
# Script code never enters context; only output does.
#
# Usage:
#   nu solve.nu "speed" "reliability"
#   nu solve.nu "speed" "reliability" --context "distributed cache"
#   nu solve.nu "speed" "reliability" --json

def find_bin [] {
    # Look for binary alongside this script first, then fall back to PATH
    let local = ($env.FILE_PWD | path join "triz")
    if ($local | path exists) { $local } else { "triz" }
}

def main [
    improving: string,          # Parameter being improved
    worsening: string,          # Parameter worsened as a result
    --context: string = "",     # Optional situation context
    --json                      # Output JSON instead of text
] {
    let bin = (find_bin)

    if ($context | is-empty) {
        if $json {
            ^$bin solve $improving $worsening --output json
        } else {
            ^$bin solve $improving $worsening
        }
    } else {
        if $json {
            ^$bin solve $improving $worsening -c $context --output json
        } else {
            ^$bin solve $improving $worsening -c $context
        }
    }

    # Exit code propagates:
    # 0 = principles found
    # 1 = empty matrix cell (no recommendation for this pair)
    # 2 = error (binary not found, bad params, etc.)
}
