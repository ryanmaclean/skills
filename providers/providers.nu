#!/usr/bin/env nu
# providers.nu — Unified AI quota monitor: Claude, Codex, Gemini
#
# Usage:
#   nu providers.nu               # show all providers
#   nu providers.nu --claude      # only Claude
#   nu providers.nu --codex       # only Codex
#   nu providers.nu --gemini      # only Gemini
#   nu providers.nu --watch 30    # refresh every 30s
#   nu providers.nu --json        # raw JSON for all providers

# ─── rendering ──────────────────────────────────────────────────────────────

def pct_bar [pct: float]: nothing -> string {
  let clamped = ($pct | [0.0, $in] | math max | [100.0, $in] | math min)
  let filled = ($clamped / 10 | math floor | into int)
  let empty = (10 - $filled)
  let bar = (0..<$filled | each { "█" } | str join "") + (0..<$empty | each { "░" } | str join "")
  let label = $"($clamped | math round | into int)%"
  if $clamped >= 90 {
    $"\u{001b}[31m($bar)\u{001b}[0m  ($label)"
  } else if $clamped >= 70 {
    $"\u{001b}[33m($bar)\u{001b}[0m  ($label)"
  } else {
    $"\u{001b}[32m($bar)\u{001b}[0m  ($label)"
  }
}

def format_reset [resets_at: string]: nothing -> string {
  let diff = ($resets_at | into datetime) - (date now)
  let secs = ($diff / 1sec | math round | into int)
  if $secs <= 0 { return "now" }
  let h = ($secs / 3600 | math floor | into int)
  let m = (($secs mod 3600) / 60 | math floor | into int)
  if $h > 0 { $"($h)h ($m)m" } else { $"($m)m" }
}

def print_section [title: string] {
  print $"\u{001b}[1m($title)\u{001b}[0m"
}

def print_row [label: string, pct: float, reset_at: string] {
  let bar = (pct_bar $pct)
  let reset = (format_reset $reset_at)
  print $"  ($label)  ($bar)   reset in ($reset)"
}

# ─── Claude ─────────────────────────────────────────────────────────────────

def claude_token [] {
  # Try credentials file first
  let path = ($env.HOME | path join ".claude" ".credentials.json")
  if ($path | path exists) {
    let creds = (open --raw $path | from json)
    if "claudeAiOauth" in $creds {
      return $creds.claudeAiOauth.accessToken
    }
  }
  # Fallback: macOS Keychain
  let raw = (try { ^security find-generic-password -s "Claude Code-credentials" -w e> /dev/null | str trim } catch { "" })
  if ($raw | str length) > 0 {
    let parsed = (try { $raw | from json } catch { null })
    if $parsed != null and "claudeAiOauth" in $parsed {
      return $parsed.claudeAiOauth.accessToken
    }
  }
  error make { msg: "No Claude credentials (checked ~/.claude/.credentials.json and Keychain)" }
}

def fetch_claude [] {
  let token = (claude_token)
  http get "https://api.anthropic.com/api/oauth/usage" --headers {
    "Authorization": $"Bearer ($token)",
    "anthropic-beta": "oauth-2025-04-20"
  }
}

def show_claude [data: record] {
  print_section "Claude Code"
  print_row "5h       " $data.five_hour.utilization $data.five_hour.resets_at
  print_row "7d       " $data.seven_day.utilization $data.seven_day.resets_at
  if "seven_day_sonnet" in $data and $data.seven_day_sonnet != null {
    print_row "7d sonnet" $data.seven_day_sonnet.utilization $data.seven_day_sonnet.resets_at
  }
  if "seven_day_opus" in $data and $data.seven_day_opus != null {
    print_row "7d opus  " $data.seven_day_opus.utilization $data.seven_day_opus.resets_at
  }
  if "extra_usage" in $data and $data.extra_usage != null {
    let ex = $data.extra_usage
    if "is_enabled" in $ex and $ex.is_enabled {
      let used = if "usage" in $ex { $ex.usage } else { 0 }
      let limit = if "monthly_limit" in $ex { $ex.monthly_limit } else { 0 }
      print $"  extra    ($used)/($limit) tokens this month"
    }
  }
}

# ─── Codex ──────────────────────────────────────────────────────────────────

def codex_token [] {
  let path = ($env.HOME | path join ".codex" "auth.json")
  if not ($path | path exists) {
    error make { msg: "No Codex credentials at ~/.codex/auth.json" }
  }
  let d = (open --raw $path | from json)
  # Support both flat {access_token} and nested {tokens: {access_token}}
  if "access_token" in $d { $d.access_token } else { $d.tokens.access_token }
}

def fetch_codex [] {
  let token = (codex_token)
  http get "https://chatgpt.com/backend-api/wham/usage" --headers {
    "Authorization": $"Bearer ($token)",
    "Content-Type": "application/json"
  }
}

def format_reset_secs [secs: int]: nothing -> string {
  if $secs <= 0 { return "now" }
  let h = ($secs / 3600 | math floor | into int)
  let m = (($secs mod 3600) / 60 | math floor | into int)
  if $h > 0 { $"($h)h ($m)m" } else { $"($m)m" }
}

def show_codex [data: record] {
  print_section "Codex CLI"
  # Response shape: rate_limit.primary_window / secondary_window
  let rl = if "rate_limit" in $data { $data.rate_limit } else { null }
  if $rl == null {
    print "  no rate limit data returned"
    return
  }
  if "primary_window" in $rl and $rl.primary_window != null {
    let w = $rl.primary_window
    let pct = if "used_percent" in $w { $w.used_percent | into float } else { 0.0 }
    let reset = if "reset_after_seconds" in $w { format_reset_secs $w.reset_after_seconds } else { "?" }
    print $"  session  (pct_bar $pct)   reset in ($reset)"
  }
  if "secondary_window" in $rl and $rl.secondary_window != null {
    let w = $rl.secondary_window
    let pct = if "used_percent" in $w { $w.used_percent | into float } else { 0.0 }
    let reset = if "reset_after_seconds" in $w { format_reset_secs $w.reset_after_seconds } else { "?" }
    print $"  weekly   (pct_bar $pct)   reset in ($reset)"
  }
}

# ─── Gemini ─────────────────────────────────────────────────────────────────

def gemini_token [] {
  let path = ($env.HOME | path join ".gemini" "oauth_creds.json")
  if not ($path | path exists) {
    error make { msg: "No Gemini credentials at ~/.gemini/oauth_creds.json" }
  }
  let creds = (open --raw $path | from json)

  # Check if token is expired (expiry_date may be seconds or milliseconds)
  if "expiry_date" in $creds and $creds.expiry_date != null {
    let raw = $creds.expiry_date
    # If > 1e12 it's milliseconds, divide by 1000 to get seconds
    let epoch_secs = if $raw > 1000000000000.0 { $raw / 1000 } else { $raw }
    let now_secs = (date now | into int) / 1000000000
    if $epoch_secs < $now_secs {
      # Token expired — try to refresh
      if "refresh_token" in $creds and $creds.refresh_token != null {
        let refreshed = (try_refresh_gemini $creds.refresh_token)
        if $refreshed != null { return $refreshed }
      }
      print "  ⚠ Gemini token expired — run `gemini` once to refresh"
    }
  }
  $creds.access_token
}

def try_refresh_gemini [refresh_token: string] {
  # Extract client_id/secret from gemini binary's oauth2.js
  let gemini_bin = (try { ^which gemini e> /dev/null | str trim } catch { "" })
  if ($gemini_bin | is-empty) { return null }

  # Resolve symlinks to find real path
  let real_bin = (try { ^readlink -f $gemini_bin | str trim } catch { $gemini_bin })
  let bin_dir = ($real_bin | path dirname)

  # Walk up looking for node_modules/@google/gemini-cli-core/dist/src/code_assist/oauth2.js
  let oauth2_js = (
    [0, 1, 2, 3, 4, 5, 6, 7, 8, 9] | each { |n|
      let dir = (1..$n | reduce -f $bin_dir { |_, acc| $acc | path dirname })
      let candidates = [
        ($dir | path join "node_modules" "@google" "gemini-cli-core" "dist" "src" "code_assist" "oauth2.js"),
        ($dir | path join "lib" "node_modules" "@google" "gemini-cli-core" "dist" "src" "code_assist" "oauth2.js"),
      ]
      $candidates | where { |p| $p | path exists } | first
    } | flatten | first
  )

  if ($oauth2_js | is-empty) { return null }

  let source = (open --raw $oauth2_js)
  let client_id = (try { $source | parse --regex "OAUTH_CLIENT_ID\\s*=\\s*[\"']([^\"']+)[\"']" | get capture0.0 } catch { "" })
  let client_secret = (try { $source | parse --regex "OAUTH_CLIENT_SECRET\\s*=\\s*[\"']([^\"']+)[\"']" | get capture0.0 } catch { "" })

  if ($client_id | is-empty) or ($client_secret | is-empty) { return null }

  let refresh_body = { "client_id": $client_id, "client_secret": $client_secret, "refresh_token": $refresh_token, "grant_type": "refresh_token" }
  let resp = (try { http post "https://oauth2.googleapis.com/token" $refresh_body } catch { null })

  if $resp == null or not ("access_token" in $resp) { return null }

  # Save refreshed token back to disk
  let path = ($env.HOME | path join ".gemini" "oauth_creds.json")
  let existing = (open --raw $path | from json)
  let updated = ($existing | upsert access_token $resp.access_token)
  $updated | to json | save --force $path

  $resp.access_token
}

def fetch_gemini [] {
  let token = (gemini_token)
  let auth = { "Authorization": $"Bearer ($token)" }

  # Step 1: loadCodeAssist → get project
  let project = (try {
    let body = { "metadata": { "ideType": "GEMINI_CLI", "pluginType": "GEMINI" } }
    let resp = (http post "https://cloudcode-pa.googleapis.com/v1internal:loadCodeAssist" --headers $auth $body)
    if "cloudaicompanionProject" in $resp { $resp.cloudaicompanionProject } else { "" }
  } catch { "" })

  # Step 2: retrieveUserQuota
  let body = { "project": $project }
  http post "https://cloudcode-pa.googleapis.com/v1internal:retrieveUserQuota" --headers $auth $body
}

def show_gemini [data: record] {
  print_section "Gemini CLI"

  let entries = if "quotas" in $data { $data.quotas } else { [] }
  if ($entries | is-empty) {
    print "  no quota data returned"
    return
  }

  # Group by model type, find minimum remainingFraction per group
  let pro_entries = ($entries | where { |e| "modelId" in $e and ($e.modelId | str downcase | str contains "pro") })
  let flash_entries = ($entries | where { |e| "modelId" in $e and ($e.modelId | str downcase | str contains "flash") })

  let earliest_reset = (
    $entries
    | where { |e| "resetTime" in $e and $e.resetTime != null }
    | each { |e| $e.resetTime }
    | sort
    | first
  )

  if not ($pro_entries | is-empty) {
    let min_remaining = ($pro_entries | where { |e| "remainingFraction" in $e } | each { |e| $e.remainingFraction } | math min)
    let used_pct = ((1 - $min_remaining) * 100)
    let reset = if ($earliest_reset | is-empty) { "?" } else { format_reset $earliest_reset }
    print $"  Pro      (pct_bar $used_pct)   reset in ($reset)"
  }
  if not ($flash_entries | is-empty) {
    let min_remaining = ($flash_entries | where { |e| "remainingFraction" in $e } | each { |e| $e.remainingFraction } | math min)
    let used_pct = ((1 - $min_remaining) * 100)
    let reset = if ($earliest_reset | is-empty) { "?" } else { format_reset $earliest_reset }
    print $"  Flash    (pct_bar $used_pct)   reset in ($reset)"
  }
}

# ─── main ────────────────────────────────────────────────────────────────────

def main [
  --claude          # Show only Claude
  --codex           # Show only Codex
  --gemini          # Show only Gemini
  --watch: int = 0  # Auto-refresh interval in seconds
  --json            # Raw JSON output
] {
  let show_all = (not $claude) and (not $codex) and (not $gemini)
  let show_claude = $show_all or $claude
  let show_codex  = $show_all or $codex
  let show_gemini = $show_all or $gemini

  let run = {
    if $json {
      let out = {}
      let out = if $show_claude { $out | insert claude (try { fetch_claude } catch { |e| {error: $e.msg} }) } else { $out }
      let out = if $show_codex  { $out | insert codex  (try { fetch_codex  } catch { |e| {error: $e.msg} }) } else { $out }
      let out = if $show_gemini { $out | insert gemini (try { fetch_gemini } catch { |e| {error: $e.msg} }) } else { $out }
      print ($out | to json --indent 2)
      return
    }

    if $show_claude {
      try { show_claude (fetch_claude) } catch { |e| print $"\u{001b}[1mClaude Code\u{001b}[0m\n  ✗ ($e.msg)" }
    }
    if $show_codex {
      if $show_claude { print "" }
      try { show_codex (fetch_codex) } catch { |e| print $"\u{001b}[1mCodex CLI\u{001b}[0m\n  ✗ ($e.msg)" }
    }
    if $show_gemini {
      if $show_claude or $show_codex { print "" }
      try { show_gemini (fetch_gemini) } catch { |e| print $"\u{001b}[1mGemini CLI\u{001b}[0m\n  ✗ ($e.msg)" }
    }
  }

  if $watch > 0 {
    loop {
      print "\u{001b}[2J\u{001b}[H"
      do $run
      print $"\n  Refreshing every ($watch)s — Ctrl+C to stop"
      sleep ($watch | into duration --unit sec)
    }
  } else {
    do $run
  }
}
