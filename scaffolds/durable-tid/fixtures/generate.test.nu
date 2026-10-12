#!/usr/bin/env nu
# SPDX-License-Identifier: BSD-2-Clause
use std/assert
use generate.nu [le-hex fixture-records]
const HERE = path self | path dirname

def main [scratch: path] {
    if ($scratch | path exists) { error make {msg: 'scratch must not already exist'} }
    mkdir $scratch
    let out = ($scratch | path join 'generated')
    mkdir $out
    let child = (^$nu.current-exe --no-config-file ($HERE | path join 'generate.nu') --output $out | complete)
    assert equal $child.exit_code 0
    assert equal $child.stderr ''
    assert equal ($child.stdout | str trim) 'generated: genesis-submit, committed-persistent, duplicate-submit-rejected-shape'
    let names = ['genesis-submit' 'committed-persistent' 'duplicate-submit-rejected-shape']
    mut cases = []
    for name in $names {
        for ext in ['hex' 'json'] {
            let file = ($name + '.' + $ext)
            let expected = (open --raw ($HERE | path join $file))
            let actual = (open --raw ($out | path join $file))
            assert equal $actual $expected
            $cases ++= [$'exact-($file)']
        }
    }
    assert equal (ls $out | length) 6
    let second = (^$nu.current-exe --no-config-file ($HERE | path join 'generate.nu') --output $out | complete)
    assert equal $second.exit_code 0
    assert equal $second.stderr ''
    for name in $names {
        for ext in ['hex' 'json'] {
            let file = ($name + '.' + $ext)
            assert equal (open --raw ($out | path join $file)) (open --raw ($HERE | path join $file))
        }
    }
    $cases ++= ['repeat-exact-six']
    for row in [
        {n: 0, width: 4, hex: '00000000'}
        {n: 255, width: 4, hex: 'ff000000'}
        {n: 256, width: 4, hex: '00010000'}
        {n: 16909060, width: 4, hex: '04030201'}
        {n: 4294967295, width: 4, hex: 'ffffffff'}
        {n: 9223372036854775807, width: 8, hex: 'ffffffffffffff7f'}
    ] {
        assert equal (le-hex $row.n $row.width) $row.hex
        $cases ++= [$'integer-($row.n)-($row.width)']
    }
    for row in [{n: -1, width: 4} {n: 4294967296, width: 4} {n: 1, width: 3}] {
        let rejected = (try { le-hex $row.n $row.width | ignore; false } catch { true })
        assert $rejected
        $cases ++= [$'reject-integer-($row.n)-($row.width)']
    }
    let records = (fixture-records)
    assert equal (($records | get 1).json | from json | get content_hash) '6ef95ff683fb8812a070c65f24b9d0dbc158bed5a60df7f3c432a70cf97c582f'
    assert equal (($records | get 0).hex | decode hex | bytes length) 104
    assert equal (($records | get 1).hex | decode hex | bytes length) 108
    $cases ++= ['known-content-hash-and-wire-lengths']
    let missing = (^$nu.current-exe --no-config-file ($HERE | path join 'generate.nu') --output ($scratch | path join 'absent') | complete)
    assert ($missing.exit_code != 0)
    assert (not ($scratch | path join 'absent' | path exists))
    $cases ++= ['missing-output-rejected']
    {ok: true, cases: $cases, count: ($cases | length), canonical_files: 6, legacy_generator_executed: false} | to json
}
