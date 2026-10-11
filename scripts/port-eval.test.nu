#!/usr/bin/env nu
# Tests for scripts/port-eval.nu. Run: nu scripts/port-eval.test.nu
# Exits non-zero if any test fails.

use std/assert
use port-eval-exact.nu [exact-from-json exact-to-json number? number-text number-integer? number-compare number-rounded-ms]

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

def expect-critical-error [r: record, needle: string] {
    assert equal $r.code 2 $"expected exit 2, got ($r.code): ($r.out | to json -r)"
    assert equal $r.out.verdict "critical_fail"
    assert ($r.out.errors | any {|e| $e | str contains $needle })
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

def test_critical_check_is_required [] {
    let d = (tmpdir)
    let p = (write-json $d spec.json (open $SPEC | reject safety.critical_check))
    expect-error (tool validate $p) "safety.critical_check is required"
    rm -rf $d
}

def test_min_soak_is_required [] {
    let d = (tmpdir)
    let p = (write-json $d spec.json (open $SPEC | reject safety.intake.min_soak_seconds))
    expect-error (tool validate $p) "safety.intake.min_soak_seconds is required"
    rm -rf $d
}

def test_every_arm_needs_an_applicable_gate [] {
    let d = (tmpdir)
    let spec = (open $SPEC)
    let gates = ($spec.gates | each {|g| $g | upsert applies_to_arch ["amd64"] })
    let p = (write-json $d spec.json ($spec | upsert gates $gates))
    expect-error (tool validate $p) "has no applicable gates"
    # `tool` parses stdout as JSON; both paths must refuse before gate scoring.
    let checked = (tool check $p $RESULT)
    assert equal $checked.code 1
    assert equal $checked.out.verdict "inconsistent"
    assert ($checked.out.errors | any {|e| $e | str contains "has no applicable gates" })
    let projected = (tool bench $p $RESULT)
    assert equal $projected.code 1
    assert equal $projected.out.verdict "refused"
    assert ($projected.out.errors | any {|e| $e | str contains "has no applicable gates" })
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

def test_critical_method_is_required [] {
    let d = (tmpdir)
    let p = (write-json $d result.json (open $RESULT | reject critical.method))
    let r = (tool check $SPEC $p)
    assert equal $r.code 2
    assert equal $r.out.verdict "critical_fail"
    assert ($r.out.errors | any {|e| $e | str contains "result.critical.method is required" })
    rm -rf $d
}

def test_critical_checked_at_is_required [] {
    let d = (tmpdir)
    let p = (write-json $d result.json (open $RESULT | reject critical.checked_at))
    expect-critical-error (tool check $SPEC $p) "result.critical.checked_at is required"
    rm -rf $d
}

def test_critical_checked_at_format_is_checked [] {
    let d = (tmpdir)
    let p = (write-json $d result.json (open $RESULT | upsert critical.checked_at "yesterday"))
    expect-critical-error (tool check $SPEC $p) "result.critical.checked_at"
    rm -rf $d
}

def test_critical_checked_at_type_is_checked [] {
    let d = (tmpdir)
    let p = (write-json $d result.json (open $RESULT | upsert critical.checked_at 42))
    expect-critical-error (tool check $SPEC $p) "result.critical.checked_at"
    rm -rf $d
}

def test_critical_boolean_is_required [] {
    let d = (tmpdir)
    let p = (write-json $d result.json (open $RESULT | reject critical.any_agent_down))
    expect-critical-error (tool check $SPEC $p) "result.critical.any_agent_down is required"
    rm -rf $d
}

def test_critical_agent_down_must_be_boolean [] {
    let d = (tmpdir)
    let p = (write-json $d result.json (open $RESULT | upsert critical.any_agent_down "false"))
    expect-critical-error (tool check $SPEC $p) "result.critical.any_agent_down: wrong type; expected boolean"
    rm -rf $d
}

def test_critical_eval_binary_must_be_boolean [] {
    let d = (tmpdir)
    let p = (write-json $d result.json (open $RESULT | upsert critical.any_host_left_on_eval_binary 0))
    expect-critical-error (tool check $SPEC $p) "result.critical.any_host_left_on_eval_binary: wrong type; expected boolean"
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


# Schema regression cases are SOURCE ONLY until run on an admitted remote host.
def test_schema_missing_upstream [] {
    let d = (tmpdir)
    let p = (write-json $d spec.json (open $SPEC | reject task.upstream))
    expect-error (tool validate $p) "spec.task.upstream is required"
    expect-error (tool new-result $p) "spec.task.upstream is required"
    rm -rf $d
}

def test_schema_unknown_fields [] {
    let d = (tmpdir)
    let s = (write-json $d spec.json (open $SPEC | upsert task.unexpected "x"))
    expect-error (tool validate $s) "additional property"
    let r = (write-json $d result.json (open $RESULT | upsert judge.unexpected "x"))
    expect-error (tool check $SPEC $r) "additional property"
    assert equal (tool bench $SPEC $r).out.verdict "refused"
    rm -rf $d
}

def test_schema_wrong_nested_types [] {
    let d = (tmpdir)
    let s = (write-json $d spec.json (open $SPEC | upsert task.starting_points "wrong"))
    expect-error (tool validate $s) "wrong type"
    let r = (write-json $d result.json (open $RESULT | upsert arms.opus-xhigh-amd64.gates_verified.G1 1))
    expect-error (tool check $SPEC $r) "wrong type"
    rm -rf $d
}

def test_schema_wrong_root_types [] {
    let d = (tmpdir)
    let p = (write-json $d array.json [])
    expect-error (tool validate $p) "wrong type"
    expect-error (tool check $SPEC $p) "wrong type"
    rm -rf $d
}

def test_schema_array_and_numeric_constraints [] {
    let d = (tmpdir)
    let duplicate = (write-json $d duplicate.json (open $SPEC | upsert task.target_arches ["amd64" "amd64"]))
    expect-error (tool validate $duplicate) "unique"
    let empty = (write-json $d empty.json (open $SPEC | upsert task.starting_points []))
    expect-error (tool validate $empty) "too few items"
    let number = (write-json $d number.json (open $SPEC | upsert budget.max_arms 1.5))
    expect-error (tool validate $number) "integer"
    let negative = (write-json $d negative.json (open $RESULT | upsert arms.opus-xhigh-amd64.tool_calls -1))
    expect-error (tool check $SPEC $negative) "below minimum"
    rm -rf $d
}

def test_schema_patterns_and_open_maps [] {
    let d = (tmpdir)
    let p = (write-json $d digest.json (open $RESULT | upsert arms.opus-xhigh-amd64.package_sha256 "bad"))
    expect-error (tool check $SPEC $p) "pattern"
    let note = (write-json $d note.json (open $RESULT | upsert arms.opus-xhigh-amd64.score_notes.extra 123))
    expect-error (tool check $SPEC $note) "wrong type"
    rm -rf $d
}

def test_schema_calendar_dates [] {
    let d = (tmpdir)
    for stamp in ["2026-02-29T00:00:00Z" "2026-04-31T00:00:00Z" "2026-01-01T24:00:00Z" "2026-01-01T00:00:00+24:00"] {
        let p = (write-json $d invalid.json (open $RESULT | upsert judged_at $stamp))
        expect-error (tool check $SPEC $p) "RFC3339"
    }
    let p = (write-json $d critical.json (open $RESULT | upsert critical.checked_at "2026-02-31T00:00:00Z"))
    expect-critical-error (tool check $SPEC $p) "RFC3339"
    rm -rf $d
}

def test_schema_rfc3339_forms [] {
    let d = (tmpdir)
    for stamp in ["2024-02-29t00:00:00.123z" "1990-12-31T23:59:60Z" "1990-12-31T15:59:60-08:00" "1991-01-01T00:59:60+01:00" "2026-01-01T00:00:00-00:00"] {
        let p = (write-json $d valid.json (open $RESULT | upsert judged_at $stamp | upsert critical.checked_at $stamp))
        assert equal (tool check $SPEC $p).code 0
    }
    let p = (write-json $d invalid.json (open $RESULT | upsert judged_at "1990-12-31T12:00:60Z"))
    expect-error (tool check $SPEC $p) "RFC3339"
    rm -rf $d
}

def test_invalid_json_is_json_error [] {
    let d = (tmpdir)
    let p = ($d | path join broken.json)
    for raw in ['{' '{"ignored":true,}' '{"ignored":/*not JSON*/true}'] {
        $raw | save -f $p
        expect-error (tool validate $p) "decode JSON"
    }
    rm -rf $d
}

# Required conformance gate, not a claimed passing regression. A JSON decoder
# that rounds this noninteger to 4 must not permit schema integer acceptance.
def test_integer_lexeme_precision [] {
    let d = (tmpdir)
    let before = (open --raw $SPEC)
    let raw = ($before | str replace '"max_arms": 4' '"max_arms": 4.0000000000000000000000001')
    assert ($raw != $before)
    let p = ($d | path join precision.json)
    $raw | save $p
    expect-error (tool validate $p) "integer"
    rm -rf $d
}


# Raw fixture construction deliberately occurs after ordinary JSON encoding;
# neither the fixture author nor the CLI may round the tested numeric token.
def raw-number [document: any, field: cell-path, token: string] {
    $document | upsert $field '__NUMERIC_LEXEME__' | to json --raw
        | str replace '"__NUMERIC_LEXEME__"' $token
}
def write-raw [dir: string, text: string] {
    let path = ($dir | path join exact.json)
    $text | save -f $path
    $path
}
def tool-exact [...args: string] {
    let r = (^nu $TOOL ...$args | complete)
    {code: $r.exit_code, out: (exact-from-json $r.stdout), raw: $r.stdout, stderr: $r.stderr}
}
def test_fractional_rounding_rejected_by_all_commands [] {
    let d = (tmpdir)
    for token in ['4.0000000000000000000000001' '3.9999999999999999999999999' '4.0000000000000000001e0'] {
        let p = (write-raw $d (raw-number (open $SPEC) budget.max_arms $token))
        expect-error (tool validate $p) 'integer'
        expect-error (tool new-result $p) 'integer'
        expect-error (tool check $p $RESULT) 'integer'
        expect-error (tool bench $p $RESULT) 'integer'
    }
    rm -rf $d
}
def test_equivalent_integer_lexemes_remain_valid [] {
    let d = (tmpdir)
    for token in ['4.0' '40e-1' '0.4e1' '400000000000000000000000e-23'] {
        let p = (write-raw $d (raw-number (open $SPEC) budget.max_arms $token))
        assert equal (tool validate $p).code 0
        assert equal (tool new-result $p).code 0
        assert equal (tool check $p $RESULT).code 0
        assert equal (tool bench $p $RESULT).code 0
    }
    rm -rf $d
}
def test_huge_budget_integers_do_not_impose_machine_bounds [] {
    let d = (tmpdir)
    for token in ['9223372036854775808' '1e10000' '1e99999999999999999999999999999999999999999999999999'] {
        let p = (write-raw $d (raw-number (open $SPEC) budget.max_arms $token))
        assert equal (tool validate $p).code 0
        assert equal (tool new-result $p).code 0
        assert equal (tool check $p $RESULT).code 0
        assert equal (tool bench $p $RESULT).code 0
    }
    rm -rf $d
}
def test_telemetry_huge_integer_projection_is_exact [] {
    let d = (tmpdir)
    for field in [tokens.output tokens.input_incl_cache tool_calls human_interventions] {
        let path = ([arms opus-xhigh-amd64] ++ ($field | split row '.') | into cell-path)
        let p = (write-raw $d (raw-number (open $RESULT) $path '922337203685477580812345678901234567890'))
        assert equal (tool check $SPEC $p).code 0
        let projected = (tool-exact bench $SPEC $p)
        assert equal $projected.code 0
        let key = match $field { 'tokens.output' => 'output_tokens', 'tokens.input_incl_cache' => 'input_tokens_incl_cache', _ => $field }
        let value = ($projected.out | where tags.arm == opus-xhigh-amd64 | first | get metrics | get $key)
        assert equal (number-text $value) '922337203685477580812345678901234567890'
    }
    rm -rf $d
}
def test_negative_tiny_numbers_and_fractional_counts_fail_exactly [] {
    let d = (tmpdir)
    let negative = (write-raw $d (raw-number (open $RESULT) arms.opus-xhigh-amd64.wall_clock_minutes '-1e-999999999999999999999999999999999999'))
    expect-error (tool check $SPEC $negative) 'below minimum'
    let fractional = (write-raw $d (raw-number (open $RESULT) arms.opus-xhigh-amd64.tool_calls '1e-999999999999999999999999999999999999'))
    expect-error (tool check $SPEC $fractional) 'integer'
    rm -rf $d
}
def test_signed_zero_and_exponent_cancellation [] {
    let d = (tmpdir)
    for token in ['-0' '-0.000e-99999999999999999999999' '10000000000000000000e-19'] {
        let p = (write-raw $d (raw-number (open $RESULT) arms.opus-xhigh-amd64.tool_calls $token))
        assert equal (tool check $SPEC $p).code 0
        let projected = (tool-exact bench $SPEC $p)
        assert equal $projected.code 0
        let value = ($projected.out | where tags.arm == opus-xhigh-amd64 | first | get metrics.tool_calls)
        assert equal (number-compare $value (exact-from-json $token)) 0
    }
    rm -rf $d
}
def test_huge_scores_and_weights_are_consistency_errors [] {
    let d = (tmpdir)
    let p = (write-raw $d (raw-number (open $RESULT) arms.opus-xhigh-amd64.scores.verified_gates '1e999999999999999999999'))
    expect-error (tool check $SPEC $p) 'outside 0..'
    expect-error (tool bench $SPEC $p) 'outside 0..'
    let p = (write-raw $d (raw-number (open $SPEC) rubric.categories.0.weight '1e999999999999999999999'))
    expect-error (tool validate $p) 'sum to 100'
    rm -rf $d
}
def test_wall_projection_decimal_rounding_boundaries [] {
    let d = (tmpdir)
    let cases = [
        {token: '0.0000249999999999999999999999999999', expected: '1'}
        {token: '0.000025', expected: '2'}
        {token: '0.0000250000000000000000000000000001', expected: '2'}
        {token: '1e-99999999999999999999999', expected: '0'}
        {token: '1e10000', expected: '6e10004'}
        {token: '0.0000083333333333333333333333333333', expected: '0'}
        {token: '0.0000083333333333333333333333333334', expected: '1'}
    ]
    for case in $cases {
        let p = (write-raw $d (raw-number (open $RESULT) arms.opus-xhigh-amd64.wall_clock_minutes $case.token))
        assert equal (tool check $SPEC $p).code 0
        let projected = (tool-exact bench $SPEC $p)
        assert equal $projected.code 0
        let value = ($projected.out | where tags.arm == opus-xhigh-amd64 | first | get metrics.wall_clock_ms)
        assert equal (number-compare $value (exact-from-json $case.expected)) 0
        assert (number-integer? $value)
    }
    rm -rf $d
}
def test_huge_exponents_are_compared_without_expansion [] {
    let a = (exact-from-json '1e99999999999999999999999999999999999999999999999999')
    let b = (exact-from-json '10e99999999999999999999999999999999999999999999999998')
    assert equal (number-compare $a $b) 0
    assert (number-integer? $a)
    let tiny = (exact-from-json '1e-99999999999999999999999999999999999999999999999999')
    assert ((number-compare $tiny 0) > 0)
    assert (not (number-integer? $tiny))
    assert equal (number-text (number-rounded-ms $tiny)) '0'
}
def test_number_parser_strict_grammar_and_unforgeable_type [] {
    let d = (tmpdir)
    for token in ['01' '+1' '.5' '1.' '1e' '1e+' 'NaN' 'Infinity' '--1'] {
        let p = (write-raw $d (raw-number (open $SPEC) budget.max_arms $token))
        expect-error (tool validate $p) 'decode JSON'
    }
    assert (not (number? (exact-from-json '"4e10000"')))
    assert (not (number? (exact-from-json '{"kind":"number","value":"4e10000"}')))
    let invalid_binary_rejected = (try { exact-to-json ('1,true' | encode utf-8); false } catch { true })
    assert $invalid_binary_rejected
    rm -rf $d
}
def test_exact_json_preserves_nested_strings_nulls_and_numbers [] {
    let raw = '{"s":"quote\" slash\\ newline\n \uD83D\uDE00","a":[null,[],{},"123.456",-0,1e9999],"true":false}'
    let value = (exact-from-json $raw)
    let again = (exact-from-json (exact-to-json $value))
    assert equal $again.s $value.s
    assert equal ($again.a | length) 6
    assert equal ($again.a | get 0) null
    assert equal ($again.a | get 1 | length) 0
    assert equal ($again.a | get 2 | columns | length) 0
    assert equal ($again.a | get 3) '123.456'
    assert equal (number-compare ($again.a | get 4) 0) 0
    assert equal (number-text ($again.a | get 5)) '1e9999'
    assert equal $again.true false
}

def test_explicit_critical_flag_dominates_invalid_spec_and_missing_sibling [] {
    let d = (tmpdir)
    let spec = (write-json $d spec.json (open $SPEC | reject task.upstream))
    # Both shapes deliberately omit the sibling flag and observation fields.
    # Even malformed spec/result shape cannot downgrade a separately explicit true.
    for critical in [{any_agent_down: true} {any_host_left_on_eval_binary: true}] {
        let result = (write-json $d result.json (open $RESULT | upsert critical $critical))
        expect-critical-error (tool check $spec $result) 'critical safety check failed'
        let bench = (tool bench $spec $result)
        assert equal $bench.code 1
        assert equal $bench.out.verdict 'refused'
        assert ($bench.out.errors | any {|e| $e | str contains 'critical safety check failed' })
    }
    # Absence alone is not an explicit true; existing invalid-spec precedence remains.
    let result = (write-json $d result.json (open $RESULT | upsert critical {any_agent_down: false}))
    expect-error (tool check $spec $result) 'spec:'
    rm -rf $d
}

def main [] {
    let tests = [
        { name: "test_explicit_critical_flag_dominates_invalid_spec_and_missing_sibling", run: {|| test_explicit_critical_flag_dominates_invalid_spec_and_missing_sibling } }
        { name: "test_fractional_rounding_rejected_by_all_commands", run: {|| test_fractional_rounding_rejected_by_all_commands } }
        { name: "test_equivalent_integer_lexemes_remain_valid", run: {|| test_equivalent_integer_lexemes_remain_valid } }
        { name: "test_huge_budget_integers_do_not_impose_machine_bounds", run: {|| test_huge_budget_integers_do_not_impose_machine_bounds } }
        { name: "test_telemetry_huge_integer_projection_is_exact", run: {|| test_telemetry_huge_integer_projection_is_exact } }
        { name: "test_negative_tiny_numbers_and_fractional_counts_fail_exactly", run: {|| test_negative_tiny_numbers_and_fractional_counts_fail_exactly } }
        { name: "test_signed_zero_and_exponent_cancellation", run: {|| test_signed_zero_and_exponent_cancellation } }
        { name: "test_huge_scores_and_weights_are_consistency_errors", run: {|| test_huge_scores_and_weights_are_consistency_errors } }
        { name: "test_wall_projection_decimal_rounding_boundaries", run: {|| test_wall_projection_decimal_rounding_boundaries } }
        { name: "test_huge_exponents_are_compared_without_expansion", run: {|| test_huge_exponents_are_compared_without_expansion } }
        { name: "test_number_parser_strict_grammar_and_unforgeable_type", run: {|| test_number_parser_strict_grammar_and_unforgeable_type } }
        { name: "test_exact_json_preserves_nested_strings_nulls_and_numbers", run: {|| test_exact_json_preserves_nested_strings_nulls_and_numbers } }
        { name: "test_integer_lexeme_precision", run: {|| test_integer_lexeme_precision } }
        { name: "test_schema_missing_upstream", run: {|| test_schema_missing_upstream } }
        { name: "test_schema_unknown_fields", run: {|| test_schema_unknown_fields } }
        { name: "test_schema_wrong_nested_types", run: {|| test_schema_wrong_nested_types } }
        { name: "test_schema_wrong_root_types", run: {|| test_schema_wrong_root_types } }
        { name: "test_schema_array_and_numeric_constraints", run: {|| test_schema_array_and_numeric_constraints } }
        { name: "test_schema_patterns_and_open_maps", run: {|| test_schema_patterns_and_open_maps } }
        { name: "test_schema_calendar_dates", run: {|| test_schema_calendar_dates } }
        { name: "test_schema_rfc3339_forms", run: {|| test_schema_rfc3339_forms } }
        { name: "test_invalid_json_is_json_error", run: {|| test_invalid_json_is_json_error } }
        { name: "test_example_spec_valid", run: {|| test_example_spec_valid } }
        { name: "test_example_result_consistent", run: {|| test_example_result_consistent } }
        { name: "test_rubric_must_sum_to_100", run: {|| test_rubric_must_sum_to_100 } }
        { name: "test_safety_critical_gate_needs_watchdog", run: {|| test_safety_critical_gate_needs_watchdog } }
        { name: "test_low_soak_is_warning_not_error", run: {|| test_low_soak_is_warning_not_error } }
        { name: "test_gate_arch_must_be_target_arch", run: {|| test_gate_arch_must_be_target_arch } }
        { name: "test_critical_check_is_required", run: {|| test_critical_check_is_required } }
        { name: "test_min_soak_is_required", run: {|| test_min_soak_is_required } }
        { name: "test_every_arm_needs_an_applicable_gate", run: {|| test_every_arm_needs_an_applicable_gate } }
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
        { name: "test_critical_method_is_required", run: {|| test_critical_method_is_required } }
        { name: "test_critical_checked_at_is_required", run: {|| test_critical_checked_at_is_required } }
        { name: "test_critical_checked_at_format_is_checked", run: {|| test_critical_checked_at_format_is_checked } }
        { name: "test_critical_checked_at_type_is_checked", run: {|| test_critical_checked_at_type_is_checked } }
        { name: "test_critical_boolean_is_required", run: {|| test_critical_boolean_is_required } }
        { name: "test_critical_agent_down_must_be_boolean", run: {|| test_critical_agent_down_must_be_boolean } }
        { name: "test_critical_eval_binary_must_be_boolean", run: {|| test_critical_eval_binary_must_be_boolean } }
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
