#!/usr/bin/env nu
# zellij-team.nu — spawn parallel Claude agents in Zellij panes
#
# Usage:
#   nu zellij-team.nu --tasks ["task A" "task B" "task C"]
#   nu zellij-team.nu --files [task1.md task2.md]
#   nu zellij-team.nu --tab "my-team" --mode default --tasks ["..."]

def main [
  --tasks: list<string> = []   # Task prompts (inline strings)
  --files: list<string> = []   # Task files (read content as prompt)
  --tab: string = ""           # Zellij tab name (default: team-<epoch>)
  --mode: string = "bypassPermissions"  # Claude permission mode
  --effort: string = ""        # effort level flag (low/high/normal)
  --workdir: string = ""       # Working directory for agents (default: cwd)
  --dry-run                    # Print commands without running
] {
  # ── Resolve tasks ────────────────────────────────────────────────────────
  mut all_tasks: list<string> = $tasks
  for f in $files {
    let content = (open --raw $f | str trim)
    $all_tasks = ($all_tasks | append $content)
  }

  if ($all_tasks | length) == 0 {
    error make { msg: "No tasks provided. Use --tasks or --files." }
  }

  let n = ($all_tasks | length)
  let tab_name = if $tab == "" { $"team-(date now | format date '%H%M%S')" } else { $tab }
  let cwd = if $workdir == "" { $env.PWD } else { $workdir }

  # ── Zellij session ───────────────────────────────────────────────────────
  let session = if "ZELLIJ_SESSION_NAME" in $env {
    $env.ZELLIJ_SESSION_NAME
  } else {
    let sessions = (^zellij list-sessions --no-formatting 2>/dev/null
      | lines
      | where { |l| ($l | str trim) != "" and not ($l | str contains "EXITED") }
      | each { |l| $l | split row " " | first })
    if ($sessions | length) == 1 {
      $sessions | first | str trim
    } else if ($sessions | length) > 1 {
      error make { msg: $"Multiple Zellij sessions: ($sessions | str join ', '). Set ZELLIJ_SESSION_NAME." }
    } else {
      error make { msg: "No Zellij session running. Start one first." }
    }
  }

  # ── Build agent command ──────────────────────────────────────────────────
  let effort_flag = if $effort == "" { "" } else { $"--effort ($effort)" }

  def agent_cmd [task: string] {
    # Escape single quotes in task for shell safety
    let safe_task = ($task | str replace --all "'" "'\"'\"'")
    if $effort_flag == "" {
      $"claude --dangerously-skip-permissions --permission-mode ($mode) -p '($safe_task)'"
    } else {
      $"claude --dangerously-skip-permissions --permission-mode ($mode) ($effort_flag) -p '($safe_task)'"
    }
  }

  # ── Layout plan ──────────────────────────────────────────────────────────
  # Strategy: open new tab, then add panes via zellij action
  # First pane is free (tab itself), subsequent panes split

  print $"Spawning ($n) agent(s) in Zellij tab '($tab_name)' (session: ($session))"

  if $dry_run {
    print "\n[DRY RUN] Commands that would be run:\n"
    for i in 0..<$n {
      print $"  Pane ($i + 1): (agent_cmd ($all_tasks | get $i))"
    }
    return
  }

  # ── Create tab ───────────────────────────────────────────────────────────
  ^zellij --session $session action new-tab --name $tab_name
  sleep 200ms

  # ── First pane: run first agent ──────────────────────────────────────────
  let first_cmd = (agent_cmd ($all_tasks | get 0))
  ^zellij --session $session action write-chars $"cd ($cwd) && ($first_cmd)\n"
  sleep 100ms

  # ── Remaining panes ──────────────────────────────────────────────────────
  for i in 1..<$n {
    # Alternate split direction for rough grid layout
    let direction = if ($n <= 2) {
      "right"           # 2 agents → side by side
    } else if ($i mod 2 == 1) {
      "right"           # odd panes → split right
    } else {
      "down"            # even panes → split down
    }

    ^zellij --session $session action new-pane --direction $direction
    sleep 150ms
    let cmd = (agent_cmd ($all_tasks | get $i))
    ^zellij --session $session action write-chars $"cd ($cwd) && ($cmd)\n"
    sleep 100ms
  }

  print $"\n✓ ($n) agents started in tab '($tab_name)'"
  print "  Use Alt+[ / Alt+] to navigate, Alt+arrow to switch panes"
  print "  ctrl-c in any pane to kill that agent"
}
