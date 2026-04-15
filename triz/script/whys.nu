#!/usr/bin/env nu
# 5 Whys drill — interactive contradiction articulation.
# Guides from symptom to a ready-to-run `triz solve` command.
# Script code never enters context; only output does.
#
# Usage:
#   nu whys.nu                          # interactive — prompts for symptom
#   nu whys.nu --context "payment svc"  # with situational context
#   nu whys.nu --json                   # output JSON with solve_cmd field
#   echo "slow deploys" | nu whys.nu    # piped symptom

def find_bin [] {
    let local = ($env.FILE_PWD | path join "triz")
    if ($local | path exists) { $local } else { "triz" }
}

def main [
    --context: string = ""
    --json
] {
    let bin = (find_bin)

    if $json {
        if ($context | is-empty) {
            ^$bin whys --output json
        } else {
            ^$bin whys --output json -c $context
        }
    } else {
        if ($context | is-empty) {
            ^$bin whys
        } else {
            ^$bin whys -c $context
        }
    }
}
