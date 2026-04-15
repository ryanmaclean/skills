#!/usr/bin/env nu
# Install the triz binary into this skill's scripts/ directory.
# Run once after building the triz project with `cargo build --release`.
#
# Usage:
#   nu setup.nu                              # auto-detect project from git root
#   nu setup.nu --project /path/to/triz       # explicit project path

def main [
    --project: string = ""  # Path to triz Rust project (default: auto-detect)
] {
    let proj = if ($project | is-empty) {
        # Walk up from FILE_PWD looking for Cargo.toml with name = "triz"
        let git_root = (^git -C $env.FILE_PWD rev-parse --show-toplevel | str trim)
        $git_root
    } else {
        $project
    }

    let cargo_toml = ($proj | path join "Cargo.toml")
    if not ($cargo_toml | path exists) {
        error make { msg: $"No Cargo.toml found at ($proj) — pass --project /path/to/triz" }
    }

    print $"[setup] building triz (release) in ($proj) …"
    ^cargo build --release --manifest-path $cargo_toml

    let bin_src = ($proj | path join "target/release/triz")
    if not ($bin_src | path exists) {
        error make { msg: $"Build succeeded but binary not found at ($bin_src)" }
    }

    let bin_dst = ($env.FILE_PWD | path join "triz")
    ^cp $bin_src $bin_dst
    ^chmod +x $bin_dst

    print $"[setup] triz binary installed → ($bin_dst)"
    print "[setup] scripts/solve.nu, whys.nu, lookup.nu, fmea.nu will use it automatically."
}
