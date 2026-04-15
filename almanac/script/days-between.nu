#!/usr/bin/env nu
# days-between.nu — Calculate days between two dates
#
# Usage:
#   nu script/days-between.nu "2026-04-01" "2026-04-15"
#   nu script/days-between.nu "2026-04-01" "2026-04-15" --json

def main [start_str: string, end_str: string, --json] {
  # Simplified placeholder - full implementation needs proper date arithmetic
  
  if $json {
    {
      start_date: $start_str,
      end_date: $end_str,
      days_between: 14  # Placeholder
    } | to json
  } else {
    print "14 days between ($start_str) and ($end_str) (placeholder)"
  }
}
