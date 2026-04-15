#!/usr/bin/env nu
# tiger-team.nu — spawn specialist Claude agents attacking ONE problem from different angles
#
# Usage:
#   nu tiger-team.nu --problem "Why won't BEAM cluster federate"
#   nu tiger-team.nu --problem "..." --roles [attacker pragmatist]
#   nu tiger-team.nu --dry-run --problem "..."

# Role persona definitions
def role-persona [role: string, problem: string] {
  let personas = {
    attacker: $"You are the ATTACKER on a tiger team. Your job is to find where the approach breaks, stress-test every assumption, and identify failure modes. Be adversarial — poke holes, find edge cases, identify what could go wrong. The problem: ($problem)"
    defender: $"You are the DEFENDER on a tiger team. Your job is to find what's solid, identify invariants that must be preserved, and build the safest path forward. Be conservative — what can we rely on, what must not change, what's the safe approach. The problem: ($problem)"
    architect: $"You are the ARCHITECT on a tiger team. Your job is to look at the structural and systemic angle — interfaces, coupling, long-term implications, and how this fits into the broader system. Think about root causes, not symptoms. The problem: ($problem)"
    pragmatist: $"You are the PRAGMATIST on a tiger team. Your job is to find the fastest working solution right now. Minimal viable fix. Unblock immediately. Skip elegance, ship it. The problem: ($problem)"
  }
  $personas | get $role
}

def main [
  --problem: string = ""        # The problem statement all specialists attack
  --roles: list<string> = [attacker defender architect pragmatist]  # Which specialists to spawn
  --tab: string = ""            # Zellij tab name (default: tiger-<timestamp>)
  --mode: string = "bypassPermissions"  # Claude permission mode
  --effort: string = ""         # effort level flag (low/high/normal)
  --workdir: string = ""        # Working directory for agents (default: cwd)
  --dry-run                     # Print commands without running
] {
  if $problem == "" {
    error make { msg: "No problem provided. Use --problem \"description\"." }
  }

  let valid_roles = [attacker defender architect pragmatist]
  for r in $roles {
    if $r not-in $valid_roles {
      error make { msg: $"Unknown role '($r)'. Valid: ($valid_roles | str join ', ')" }
    }
  }

  let n = ($roles | length)
  let tab_name = if $tab == "" { $"tiger-(date now | format date '%H%M%S')" } else { $tab }
  let cwd = if $workdir == "" { $env.PWD } else { $workdir }

  # ── Build agent command ──────────────────────────────────────────────────
  let effort_flag = if $effort == "" { "" } else { $"--effort ($effort)" }

  def agent_cmd [task: string] {
    let safe_task = ($task | str replace --all "'" "'\"'\"'")
    if $effort_flag == "" {
      $"claude --dangerously-skip-permissions --permission-mode ($mode) -p '($safe_task)'"
    } else {
      $"claude --dangerously-skip-permissions --permission-mode ($mode) ($effort_flag) -p '($safe_task)'"
    }
  }

  # ── Build per-role prompts ─────────────────────────────────────────────
  let role_prompts = ($roles | each { |r| role-persona $r $problem })

  let roles_str = ($roles | str join ', ')
  print $"Tiger team: ($n) specialist\(s\) [($roles_str)] on: ($problem)"

  if $dry_run {
    print $"Tab: ($tab_name) | Session: <dry-run>"
    print "\n[DRY RUN] Commands that would be run:\n"
    for i in 0..<$n {
      let role = ($roles | get $i)
      print $"  Pane ($i + 1) [($role)]: (agent_cmd ($role_prompts | get $i))"
    }
    return
  }

  # ── Zellij session (skipped in dry-run) ────────────────────────────────
  let session = if "ZELLIJ_SESSION_NAME" in $env {
    $env.ZELLIJ_SESSION_NAME
  } else {
    # Only count active (non-EXITED) sessions — EXITED sessions are scrollback logs, never kill them
    let sessions = try {
      ^zellij list-sessions --no-formatting e> /dev/null
        | lines
        | where { |l| ($l | str trim) != "" and not ($l | str contains "EXITED") }
        | each { |l| $l | split row " " | first }
    } catch { [] }
    if ($sessions | length) == 1 {
      $sessions | first | str trim
    } else if ($sessions | length) > 1 {
      error make { msg: $"Multiple active Zellij sessions: ($sessions | str join ', '). Run inside Zellij or set ZELLIJ_SESSION_NAME=<name> before calling this script." }
    } else {
      error make { msg: "No active Zellij session found. Start one with: zellij --session <name>" }
    }
  }

  print $"Tab: ($tab_name) | Session: ($session)"

  # ── Create tab ───────────────────────────────────────────────────────────
  ^zellij --session $session action new-tab --name $tab_name
  sleep 200ms

  # ── First pane ──────────────────────────────────────────────────────────
  let first_cmd = (agent_cmd ($role_prompts | get 0))
  ^zellij --session $session action write-chars $"cd ($cwd) && ($first_cmd)\n"
  sleep 100ms

  # ── Remaining panes ────────────────────────────────────────────────────
  for i in 1..<$n {
    let direction = if ($n <= 2) {
      "right"
    } else if ($i mod 2 == 1) {
      "right"
    } else {
      "down"
    }

    ^zellij --session $session action new-pane --direction $direction
    sleep 150ms
    let cmd = (agent_cmd ($role_prompts | get $i))
    ^zellij --session $session action write-chars $"cd ($cwd) && ($cmd)\n"
    sleep 100ms
  }

  print $"\n($n) specialists started in tab '($tab_name)'"
  print "  Read all panes, then synthesize: convergence = high confidence, divergence = your call"
}
