#!/usr/bin/env nu
# a-team.nu — role-based project team coordination in Zellij panes
#
# Usage:
#   nu a-team.nu --phase 1 --brief "project description"
#   nu a-team.nu --phase 2 --brief "project description" --specialist "ansible"
#   nu a-team.nu --phase 3 --brief "project description"
#   nu a-team.nu --phase 4 --brief "project description"

def main [
  --brief: string = ""            # Project brief / description
  --specialist: string = "general" # Specialist domain (ansible, security, elixir, etc.)
  --phase: int = 1                # Phase to run: 1=PM, 2=Engineer+Specialist, 3=Reviewer, 4=PM synthesis
  --mode: string = "bypassPermissions"  # Claude permission mode
  --effort: string = ""           # effort level flag (low/high/normal)
  --workdir: string = ""          # Working directory for agents (default: cwd)
  --dry-run                       # Print commands without running
] {
  if $brief == "" {
    error make { msg: "No brief provided. Use --brief \"project description\"." }
  }

  let cwd = if $workdir == "" { $env.PWD } else { $workdir }

  # ── Output directory ──────────────────────────────────────────────────────
  let brief_hash = ($brief | hash md5 | str substring 0..8)
  let outdir = $"/tmp/a-team-($brief_hash)"
  mkdir $outdir

  # ── Zellij session (skip for dry-run) ───────────────────────────────────────
  let session = if $dry_run {
    "DRY-RUN"
  } else if "ZELLIJ_SESSION_NAME" in $env {
    $env.ZELLIJ_SESSION_NAME
  } else {
    let sessions = (do { ^zellij list-sessions --no-formatting } | complete | get stdout
      | lines | where { |l| ($l | str trim) != "" })
    if ($sessions | length) == 1 {
      $sessions | first | str trim
    } else if ($sessions | length) > 1 {
      error make { msg: $"Multiple Zellij sessions: ($sessions | str join ', '). Set ZELLIJ_SESSION_NAME." }
    } else {
      error make { msg: "No Zellij session running. Start one first." }
    }
  }

  # ── Helper: build claude command ───────────────────────────────────────────
  let effort_flag = if $effort == "" { "" } else { $" --effort ($effort)" }

  def agent_cmd [role: string, prompt: string] {
    let safe_prompt = ($prompt | str replace --all "'" "'\"'\"'")
    $"claude --dangerously-skip-permissions --permission-mode ($mode)($effort_flag) -p '($safe_prompt)'"
  }

  # ── Role prompts ───────────────────────────────────────────────────────────
  let pm_plan_prompt = $"You are the PROJECT MANAGER for this a-team project.

PROJECT BRIEF: ($brief)

Your job:
1. Break down this project into discrete, assignable tasks
2. For each task, assign a role: engineer or specialist \(domain: ($specialist)\)
3. Identify dependencies and ordering
4. Write the plan to ($outdir)/phase1-pm-plan.md

Output format in the file:
# A-Team Plan
## Brief
\(restate brief\)
## Tasks
- [ ] Task 1 — [engineer] description
- [ ] Task 2 — [specialist:($specialist)] description
...
## Dependencies
\(any ordering notes\)
## Review Criteria
\(what the reviewer should check\)

Write the plan file, then print a summary."

  let engineer_prompt = $"You are the ENGINEER on this a-team project.

PROJECT BRIEF: ($brief)

Read the PM's plan at ($outdir)/phase1-pm-plan.md for your assigned tasks.
Execute all tasks marked [engineer].
Write your output and results to ($outdir)/phase2-engineer.md.
Include: what you did, files changed, commands run, any issues encountered."

  let specialist_prompt = $"You are the SPECIALIST \(domain: ($specialist)\) on this a-team project.

PROJECT BRIEF: ($brief)

Read the PM's plan at ($outdir)/phase1-pm-plan.md for your assigned tasks.
Execute all tasks marked [specialist:($specialist)].
Also review the engineer's approach from your domain expertise perspective.
Write your output and findings to ($outdir)/phase2-specialist.md.
Include: domain-specific recommendations, issues found, tasks completed."

  let reviewer_prompt = $"You are the REVIEWER on this a-team project.

PROJECT BRIEF: ($brief)

Review all output from previous phases:
- PM plan: ($outdir)/phase1-pm-plan.md
- Engineer output: ($outdir)/phase2-engineer.md
- Specialist output: ($outdir)/phase2-specialist.md

Check:
1. All tasks from the PM plan are addressed
2. Code quality and correctness
3. Domain-specific concerns flagged by specialist are resolved
4. No security issues, no missing error handling at boundaries

Write your review to ($outdir)/phase3-review.md with:
- PASS/FAIL for each task
- Issues found \(if any\)
- Recommended fixes"

  let synthesis_prompt = $"You are the PROJECT MANAGER doing final synthesis for this a-team project.

PROJECT BRIEF: ($brief)

Read all phase outputs:
- Your plan: ($outdir)/phase1-pm-plan.md
- Engineer: ($outdir)/phase2-engineer.md
- Specialist: ($outdir)/phase2-specialist.md
- Reviewer: ($outdir)/phase3-review.md

Write a final project summary to ($outdir)/phase4-summary.md:
1. What was delivered
2. Outstanding issues from review
3. Next steps / follow-ups
4. Ship/no-ship recommendation"

  # ── Phase execution ────────────────────────────────────────────────────────
  let tab_name = $"a-team-p($phase)"

  if $phase == 1 {
    # PM runs solo — blocking via --print mode (writes to file itself)
    let cmd = (agent_cmd "pm" $pm_plan_prompt)
    print $"Phase 1: PM breakdown → ($outdir)/phase1-pm-plan.md"
    if $dry_run {
      print "\n[DRY RUN] Would run PM agent:\n"
      print $"  ($cmd)\n"
      print $"Output dir: ($outdir)"
      return
    }
    ^zellij --session $session action new-tab --name $tab_name
    sleep 200ms
    ^zellij --session $session action write-chars $"cd ($cwd) && ($cmd)\n"
    print $"\nPM agent started in tab '($tab_name)'"
    print $"When done, review ($outdir)/phase1-pm-plan.md then run --phase 2"

  } else if $phase == 2 {
    # Engineer + Specialist in parallel panes
    let eng_cmd = (agent_cmd "engineer" $engineer_prompt)
    let spec_cmd = (agent_cmd "specialist" $specialist_prompt)
    print $"Phase 2: Engineer + Specialist\(($specialist)\) in parallel"
    if $dry_run {
      print "\n[DRY RUN] Would run in parallel panes:\n"
      print $"  Engineer: ($eng_cmd)\n"
      print $"  Specialist: ($spec_cmd)\n"
      print $"Output dir: ($outdir)"
      return
    }
    ^zellij --session $session action new-tab --name $tab_name
    sleep 200ms
    ^zellij --session $session action write-chars $"cd ($cwd) && ($eng_cmd)\n"
    sleep 100ms
    ^zellij --session $session action new-pane --direction right
    sleep 150ms
    ^zellij --session $session action write-chars $"cd ($cwd) && ($spec_cmd)\n"
    print $"\nEngineer + Specialist started in tab '($tab_name)'"
    print $"When both done, run --phase 3"

  } else if $phase == 3 {
    # Reviewer runs solo
    let cmd = (agent_cmd "reviewer" $reviewer_prompt)
    print $"Phase 3: Reviewer checking all output"
    if $dry_run {
      print "\n[DRY RUN] Would run Reviewer agent:\n"
      print $"  ($cmd)\n"
      print $"Output dir: ($outdir)"
      return
    }
    ^zellij --session $session action new-tab --name $tab_name
    sleep 200ms
    ^zellij --session $session action write-chars $"cd ($cwd) && ($cmd)\n"
    print $"\nReviewer started in tab '($tab_name)'"
    print $"When done, review ($outdir)/phase3-review.md then run --phase 4"

  } else if $phase == 4 {
    # PM synthesis
    let cmd = (agent_cmd "pm" $synthesis_prompt)
    print $"Phase 4: PM final synthesis"
    if $dry_run {
      print "\n[DRY RUN] Would run PM synthesis agent:\n"
      print $"  ($cmd)\n"
      print $"Output dir: ($outdir)"
      return
    }
    ^zellij --session $session action new-tab --name $tab_name
    sleep 200ms
    ^zellij --session $session action write-chars $"cd ($cwd) && ($cmd)\n"
    print $"\nPM synthesis started in tab '($tab_name)'"
    print $"Final summary: ($outdir)/phase4-summary.md"

  } else {
    error make { msg: "Invalid phase. Use --phase 1, 2, 3, or 4." }
  }
}
