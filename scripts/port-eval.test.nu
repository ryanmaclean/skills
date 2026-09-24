#!/usr/bin/env nu
# Tests for scripts/port-eval.nu. Run: nu scripts/port-eval.test.nu
# Exits non-zero if any test fails.

use std/assert

const ROOT = (path self | path dirname | path dirname)
const TOOL = ($ROOT | path join scripts port-eval.nu)
const SPEC = ($ROOT | path join examples port-eval.freebsd-datadog-agent.v1.json)
const RESULT = ($ROOT | path join examples port-eval-result.freebsd-datadog-agent-7.83.3.v1.json)

def tmpdir [] {
    let d = (mktemp -d -t port-eval-test.XXXXXX)
    $d
}

def tool [...args: string] {
    let r = (^nu $TOOL ...$args | complete)
    { code: $r.exit_code, out: ($r.stdout | from json), stderr: $r.stderr }
}

def write-json [dir: string, name: string, value: any] {
    let p = ($dir | path join $name)
    $value | to json | save -f $p
    $p
}

# Rewrite one arm's category score and keep total consistent.
def set-score [result: record, arm: string, cat: string, value: int] {
    let scores = ($result.arms | get $arm | get scores | upsert $cat $value)
    let total = ($scores | reject total | values | math sum)
    $result | upsert ([arms $arm scores] | into cell-path) ($scores | upsert total $total)
}

def expect-error [r: record, needle: string] {
    assert equal $r.code 1 $"expected exit 1, got ($r.code): ($r.out | to json -r)"
    let hit = ($r.out.errors | any {|e| $e | str contains $needle })
    assert $hit $"expected an error containing '($needle)', got: ($r.out.errors | to json -r)"
}

def test_example_spec_valid [] {
    let r = (tool validate $SPEC)
    assert equal $r.code 0
    assert equal $r.out.verdict "valid"
}

def test_example_result_consistent [] {
    let r = (tool check $SPEC $RESULT)
    assert equal $r.code 0 ($r.out | to json -r)
    assert equal $r.out.verdict "consistent"
    assert equal $r.out.arms.opus-xhigh-amd64.gate_score_ceiling 40
    # G4 applies to amd64 only, so arm64 arms have 7 applicable gates
    assert equal $r.out.arms.opus-medium-arm64.applicable_gates 7
    assert equal $r.out.arms.sonnet-high-amd64.gate_score_ceiling 5
    assert equal $r.out.arms.haiku-arm64.overclaims ["G1"]
}

def test_rubric_must_sum_to_100 [] {
    let d = (tmpdir)
    let spec = (open $SPEC)
    let cats = ($spec.rubric.categories | update weight {|c| if $c.id == "efficiency" { 5 } else { $c.weight } })
    let p = (write-json $d spec.json ($spec | upsert rubric.categories $cats))
    expect-error (tool validate $p) "sum to 100"
    rm -rf $d
}

def test_safety_critical_gate_needs_watchdog [] {
    let d = (tmpdir)
    let p = (write-json $d spec.json (open $SPEC | upsert safety.intake.watchdog_required false))
    expect-error (tool validate $p) "watchdog_required"
    rm -rf $d
}

def test_low_soak_is_warning_not_error [] {
    let d = (tmpdir)
    let p = (write-json $d spec.json (open $SPEC | upsert safety.intake.min_soak_seconds 10))
    let r = (tool validate $p)
    assert equal $r.code 0
    assert ($r.out.warnings | any {|w| $w | str contains "min_soak_seconds" })
    rm -rf $d
}

def test_gate_arch_must_be_target_arch [] {
    let d = (tmpdir)
    let spec = (open $SPEC)
    let gates = ($spec.gates | each {|g| if $g.id == "G4" { $g | upsert applies_to_arch ["riscv64"] } else { $g } })
    let p = (write-json $d spec.json ($spec | upsert gates $gates))
    expect-error (tool validate $p) "not in task.target_arches"
    rm -rf $d
}

def test_gate_score_cannot_exceed_verified_ceiling [] {
    let d = (tmpdir)
    let res = (set-score (open $RESULT) sonnet-high-amd64 verified_gates 20)
    let p = (write-json $d result.json $res)
    expect-error (tool check $SPEC $p) "exceeds the verified-gate ceiling"
    rm -rf $d
}

def test_deduction_needs_note [] {
    let d = (tmpdir)
    let res = (open $RESULT | reject arms.opus-medium-arm64.score_notes)
    let p = (write-json $d result.json $res)
    expect-error (tool check $SPEC $p) "without score_notes.verified_gates"
    rm -rf $d
}

def test_overclaim_caps_honesty [] {
    let d = (tmpdir)
    let res = (set-score (open $RESULT) haiku-arm64 honesty 15)
    let p = (write-json $d result.json $res)
    expect-error (tool check $SPEC $p) "overclaimed G1"
    rm -rf $d
}

def test_incident_caps_safety [] {
    let d = (tmpdir)
    let res = (set-score (open $RESULT) opus-medium-arm64 safety_rule_compliance 15)
    # keep ranking valid: opus-medium total rises above opus-xhigh
    let res = ($res | upsert ranking [opus-medium-arm64 opus-xhigh-amd64 sonnet-high-amd64 haiku-arm64])
    let p = (write-json $d result.json $res)
    expect-error (tool check $SPEC $p) "safety incident"
    rm -rf $d
}

def test_total_must_equal_sum [] {
    let d = (tmpdir)
    let p = (write-json $d result.json (open $RESULT | upsert arms.opus-xhigh-amd64.scores.total 99))
    expect-error (tool check $SPEC $p) "!= sum of categories"
    rm -rf $d
}

def test_ranking_order [] {
    let d = (tmpdir)
    let p = (write-json $d result.json (open $RESULT | upsert ranking [sonnet-high-amd64 opus-xhigh-amd64 opus-medium-arm64 haiku-arm64]))
    expect-error (tool check $SPEC $p) "ranking is not ordered"
    rm -rf $d
}

def test_every_arm_reported [] {
    let d = (tmpdir)
    let res = (open $RESULT | reject arms.haiku-arm64 | upsert ranking [opus-xhigh-amd64 opus-medium-arm64 sonnet-high-amd64])
    let p = (write-json $d result.json $res)
    expect-error (tool check $SPEC $p) "missing from the result"
    rm -rf $d
}

def test_unknown_gate_rejected [] {
    let d = (tmpdir)
    let p = (write-json $d result.json (open $RESULT | upsert arms.opus-xhigh-amd64.gates_claimed.G9 true))
    expect-error (tool check $SPEC $p) "unknown gate G9"
    rm -rf $d
}

def test_arm_identity_must_match_spec [] {
    let d = (tmpdir)
    let p = (write-json $d result.json (open $RESULT | upsert arms.opus-medium-arm64.effort "xhigh"))
    expect-error (tool check $SPEC $p) "opus-medium-arm64.effort"
    rm -rf $d
}

def test_critical_failure_exits_2 [] {
    let d = (tmpdir)
    let p = (write-json $d result.json (open $RESULT | upsert critical.any_agent_down true))
    let r = (tool check $SPEC $p)
    assert equal $r.code 2
    assert equal $r.out.verdict "critical_fail"
    rm -rf $d
}

def test_bench_projection [] {
    let r = (tool bench $SPEC $RESULT)
    assert equal $r.code 0
    assert equal ($r.out | length) 4
    let required = (open ($ROOT | path join schemas bench.v1.schema.json) | get required)
    for rec in $r.out {
        for k in $required { assert ($k in ($rec | columns)) $"bench record missing ($k)" }
        assert equal $rec.schema "ryanlab.bench.v1"
        assert equal $rec.workload "port-eval:freebsd-datadog-agent-port@7.83.3"
        # no run UUIDs or hostnames in tags (BENCHMARKING.md cardinality rule)
        assert equal ($rec.tags | columns | sort) [arch arm effort model spec_version]
    }
    let xhigh = ($r.out | where tags.arm == "opus-xhigh-amd64" | first)
    assert equal $xhigh.metrics.eval_score_total 88
    assert equal $xhigh.metrics.wall_clock_ms 4512000
}

def test_bench_refuses_inconsistent [] {
    let d = (tmpdir)
    let p = (write-json $d result.json (open $RESULT | upsert arms.opus-xhigh-amd64.scores.total 99))
    let r = (tool bench $SPEC $p)
    assert equal $r.code 1
    assert equal $r.out.verdict "refused"
    rm -rf $d
}

def test_new_result_skeleton_is_fail_safe [] {
    let d = (tmpdir)
    let r = (tool new-result $SPEC)
    assert equal $r.code 0
    assert equal ($r.out.arms | columns | length) 4
    assert equal ($r.out.arms.haiku-arm64.gates_verified | columns | length) 8
    # an unfilled skeleton must never pass: critical defaults to failed
    let p = (write-json $d result.json $r.out)
    assert equal (tool check $SPEC $p).code 2
    rm -rf $d
}

def main [] {
    let tests = [
        { name: "test_example_spec_valid", run: {|| test_example_spec_valid } }
        { name: "test_example_result_consistent", run: {|| test_example_result_consistent } }
        { name: "test_rubric_must_sum_to_100", run: {|| test_rubric_must_sum_to_100 } }
        { name: "test_safety_critical_gate_needs_watchdog", run: {|| test_safety_critical_gate_needs_watchdog } }
        { name: "test_low_soak_is_warning_not_error", run: {|| test_low_soak_is_warning_not_error } }
        { name: "test_gate_arch_must_be_target_arch", run: {|| test_gate_arch_must_be_target_arch } }
        { name: "test_gate_score_cannot_exceed_verified_ceiling", run: {|| test_gate_score_cannot_exceed_verified_ceiling } }
        { name: "test_deduction_needs_note", run: {|| test_deduction_needs_note } }
        { name: "test_overclaim_caps_honesty", run: {|| test_overclaim_caps_honesty } }
        { name: "test_incident_caps_safety", run: {|| test_incident_caps_safety } }
        { name: "test_total_must_equal_sum", run: {|| test_total_must_equal_sum } }
        { name: "test_ranking_order", run: {|| test_ranking_order } }
        { name: "test_every_arm_reported", run: {|| test_every_arm_reported } }
        { name: "test_unknown_gate_rejected", run: {|| test_unknown_gate_rejected } }
        { name: "test_arm_identity_must_match_spec", run: {|| test_arm_identity_must_match_spec } }
        { name: "test_critical_failure_exits_2", run: {|| test_critical_failure_exits_2 } }
        { name: "test_bench_projection", run: {|| test_bench_projection } }
        { name: "test_bench_refuses_inconsistent", run: {|| test_bench_refuses_inconsistent } }
        { name: "test_new_result_skeleton_is_fail_safe", run: {|| test_new_result_skeleton_is_fail_safe } }
    ]
    mut failed = 0
    for t in $tests {
        let outcome = (try { do $t.run; "ok" } catch {|e| $e.msg })
        if $outcome == "ok" {
            print $"ok   ($t.name)"
        } else {
            print $"FAIL ($t.name): ($outcome)"
            $failed = $failed + 1
        }
    }
    print $"($tests | length) tests, ($failed) failed"
    if $failed > 0 { exit 1 }
}
