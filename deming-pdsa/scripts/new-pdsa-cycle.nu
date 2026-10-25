#!/usr/bin/env nu

def slugify [s: string] {
  $s
  | str downcase
  | str replace -ar '[^a-z0-9]+' '-'
  | str trim -c '-'
}

def main [
  --title (-t): string,
  --owner (-o): string = "unassigned",
  --dir: string = "plans/pdsa"
] {
  let ts = (date now | format date "%Y%m%dT%H%M%S")
  let slug = (slugify $title)
  let id = $"($ts)-($slug)"

  mkdir $dir
  let path = ([$dir $"($id).md"] | path join)

  let body = $"# PDSA Cycle: ($title)

- cycle_id: ($id)
- owner: ($owner)
- created_at: (date now)
- status: PLAN

## PLAN
- problem_statement:
- hypothesis:
- primary_metric:
- baseline:
- target:
- guardrails:
- timebox:
- stop_line_conditions:

<claims>
- claim:
  probe:
  expect:
</claims>

## DO
- actions:
- commands:
- artifacts:

## STUDY
- metric_result:
- baseline_vs_result:
- proveryay_verdicts:
- unexpected_findings:

## ACT
- decision: ADOPT|ADAPT|ABANDON
- standard_work_update:
- next_cycle_trigger:
"

  $body | save -f $path

  let std_path = ([$dir "STANDARD-WORK.md"] | path join)
  if not ($std_path | path exists) {
    "# Standard Work\n" | save -f $std_path
  }

  print ({cycle_id: $id, path: $path, standard_work: $std_path} | to json --raw)
}
