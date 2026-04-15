#!/usr/bin/env nu
# Lookup a TRIZ parameter or principle by name or number.
# Use when mapping a problem to the 39 parameters before running solve.nu.
# Script code never enters context; only output does.
#
# Usage:
#   nu lookup.nu "speed"          # find parameter by keyword
#   nu lookup.nu 9                # parameter by number
#   nu lookup.nu --principle 35   # explain a principle

def find_bin [] {
    let local = ($env.FILE_PWD | path join "triz")
    if ($local | path exists) { $local } else { "triz" }
}

def main [
    query?: string         # Parameter keyword or number to look up
    --principle: int = 0   # Explain a principle by ID (1-40) instead
] {
    let bin = (find_bin)

    if $principle > 0 {
        ^$bin explain $principle
        return
    }

    if ($query == null) {
        error make { msg: "Provide a parameter keyword, number, or --principle ID" }
    }

    ^$bin lookup $query
}
