#!/usr/bin/env nu
# providers.nu — Claude OAuth quota usage display for TRIZ project
# Mirrors bop's providers command: shows 5h and 7d utilization bars.
#
# Usage:
#   nu providers.nu
#   nu providers.nu --watch 30    # refresh every 30 seconds
#   nu providers.nu --json        # raw JSON output

def pct_bar [pct: float]: nothing -> string {
  # pct is 0.0–100.0
  let filled = ($pct / 10 | math floor | into int | [0, $in] | math max | [10, $in] | math min)
  let empty = (10 - $filled)
  let bar = (0..<$filled | each { "█" } | str join "") + (0..<$empty | each { "░" } | str join "")
  let label = $"($pct | math round | into int)%"
  if $pct >= 90 {
    $"\u{001b}[31m($bar)\u{001b}[0m  ($label)"
  } else if $pct >= 70 {
    $"\u{001b}[33m($bar)\u{001b}[0m  ($label)"
  } else {
    $"\u{001b}[32m($bar)\u{001b}[0m  ($label)"
  }
}

def format_reset [resets_at: string]: nothing -> string {
  let now = (date now)
  let reset_time = ($resets_at | into datetime)
  let diff = ($reset_time - $now)
  let secs = ($diff / 1sec | math round | into int)
  if $secs <= 0 { return "now" }
  let h = ($secs / 3600 | math floor | into int)
  let m = (($secs mod 3600) / 60 | math floor | into int)
  if $h > 0 { $"($h)h ($m)m" } else { $"($m)m" }
}

def get_token [] {
  let creds_path = ($env.HOME | path join ".claude" ".credentials.json")
  if ($creds_path | path exists) {
    let creds = (open --raw $creds_path | from json)
    if "claudeAiOauth" in $creds {
      return ($creds.claudeAiOauth.accessToken)
    }
  }

  # Try macOS Keychain fallback
  let token = (try {
    ^security find-generic-password -s "Claude Code-credentials" -w e> /dev/null
  } catch { "" })

  if ($token | str length) > 0 {
    let parsed = ($token | from json)
    if "claudeAiOauth" in $parsed {
      return ($parsed.claudeAiOauth.accessToken)
    }
  }

  error make { msg: "No Claude OAuth token found. Log in via Claude Code first." }
}

def fetch_usage [token: string] {
  http get "https://api.anthropic.com/api/oauth/usage" --headers {
    "Authorization": $"Bearer ($token)",
    "anthropic-beta": "oauth-2025-04-20"
  }
}

def show_row [label: string, slot: record] {
  let bar = (pct_bar $slot.utilization)
  let reset = if "resets_at" in $slot { format_reset $slot.resets_at } else { "?" }
  print $"  ($label)  ($bar)   reset in ($reset)"
}

def show_usage [data: record] {
  print $"\u{001b}[1mClaude Code — OAuth Quota\u{001b}[0m"
  show_row "5h " $data.five_hour
  show_row "7d " $data.seven_day
  if "seven_day_sonnet" in $data and $data.seven_day_sonnet != null {
    show_row "7d sonnet" $data.seven_day_sonnet
  }
  if "extra_usage" in $data and $data.extra_usage != null {
    let ex = $data.extra_usage
    if "is_enabled" in $ex and $ex.is_enabled {
      let used = if "usage" in $ex { $ex.usage } else { 0 }
      let limit = if "monthly_limit" in $ex { $ex.monthly_limit } else { 0 }
      print $"  extra  ($used)/($limit) tokens used this month"
    }
  }
}

def main [
  --watch: int = 0   # Refresh interval in seconds (0 = run once)
  --json             # Output raw JSON
] {
  let token = (get_token)

  if $watch > 0 {
    loop {
      let data = (fetch_usage $token)
      if $json {
        print ($data | to json)
      } else {
        print "\u{001b}[2J\u{001b}[H"
        show_usage $data
        print $"\n  Refreshing every ($watch)s — Ctrl+C to stop"
      }
      sleep ($watch | into duration --unit sec)
    }
  } else {
    let data = (fetch_usage $token)
    if $json {
      print ($data | to json)
    } else {
      show_usage $data
    }
  }
}
