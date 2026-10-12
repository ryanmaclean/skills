#!/usr/bin/env nu
# SPDX-License-Identifier: BSD-2-Clause
# Exact RECORD-V0 fixture generation. Builtins only; no shell/OpenSSL helpers.

export def le-hex [value: int, width: int] {
    if $value < 0 or $width not-in [4 8] { error make {msg: 'invalid fixture integer or width'} }
    let digits = ('0123456789abcdef' | split chars)
    mut remaining = $value
    mut output = ''
    for _ in 0..<$width {
        let byte = ($remaining mod 256)
        let high = ($digits | get ($byte // 16))
        let low = ($digits | get ($byte mod 16))
        $output = $output + $high + $low
        $remaining = ($remaining // 256)
    }
    if $remaining != 0 { error make {msg: 'fixture integer overflows field'} }
    $output
}

export def fixture-records [] {
    let zero4 = '00000000'
    let zero6 = '000000000000'
    let zero7 = '00000000000000'
    let zero8 = '0000000000000000'
    let zero32 = '0000000000000000000000000000000000000000000000000000000000000000'
    let domain = (('durable-tid/record/v0' | encode utf-8 | encode hex | str lowercase) + '000000')
    let rows = [
        {name: 'genesis-submit', request: '112233445566778899aabbccddeeff00', tid: 0, code: '0100', operation: 'submit', status: 'PROPOSED', payload: '', hashed: false}
        {name: 'committed-persistent', request: '212233445566778899aabbccddeeff01', tid: 1, code: '0403', operation: 'durable_ack', status: 'PERSISTENT', payload: 'deadbeef', hashed: true}
        {name: 'duplicate-submit-rejected-shape', request: '112233445566778899aabbccddeeff00', tid: 0, code: '0200', operation: 'duplicate_submit', status: 'PROPOSED', payload: '', hashed: false}
    ]
    mut result = []
    for row in $rows {
        let prefix = ('00' + $zero7 + $zero8 + $row.request + (le-hex 1 8) + (le-hex $row.tid 8) + $zero8 + $row.code + $zero6)
        let length = ($row.payload | decode hex | bytes length)
        let suffix = ((le-hex $length 4) + $zero4)
        let zeroed = ($prefix + $zero32 + $suffix)
        if ($zeroed | decode hex | bytes length) != 104 { error make {msg: 'fixture header size mismatch'} }
        let digest = if $row.hashed { ($domain + $zeroed + $row.payload) | decode hex | hash sha256 } else { null }
        let wire_hash = if $row.hashed { $digest } else { $zero32 }
        let hex = ($prefix + $wire_hash + $suffix + $row.payload)
        let projection = {
            name: $row.name, schema_version: 0, epoch: 0, request_id: $row.request,
            object_id: 1, tid: $row.tid, parent_tid: 0, operation: $row.operation,
            status: $row.status, content_hash: $digest, payload_hex: $row.payload,
            hex_file: ($row.name + '.hex')
        }
        $result ++= [{name: $row.name, hex: $hex, json: (($projection | to json --indent 2) + (char nl))}]
    }
    $result
}

export def generate-fixtures [output: path] {
    if not ($output | path exists) or ($output | path type) != 'dir' { error make {msg: 'output must be an existing directory'} }
    for row in (fixture-records) {
        $row.hex | save --raw --force ($output | path join ($row.name + '.hex'))
        $row.json | save --raw --force ($output | path join ($row.name + '.json'))
    }
}

def main [--output: path = '.'] {
    generate-fixtures $output
    print 'generated: genesis-submit, committed-persistent, duplicate-submit-rejected-shape'
}
