# SPDX-License-Identifier: MIT
# Exact vocabulary used by the committed port-eval v1 schemas.
# Fail closed if a later schema introduces an unsupported assertion.
# This module validates document shape; it cannot authenticate observations.

use port-eval-exact.nu [number? number-integer? number-compare]

const KEYWORDS = ['$schema' '$id' title description default type const required properties additionalProperties items minItems uniqueItems minProperties minimum pattern format]

def object? [value: any] { $value | describe | str starts-with 'record' }
def array? [value: any] {
    let kind = ($value | describe)
    ($kind | str starts-with 'list') or ($kind | str starts-with 'table')
}

def matches-type [value: any, wanted: string] {
    let kind = ($value | describe)
    match $wanted {
        'object' => (object? $value)
        'array' => (array? $value)
        'string' => ($kind == 'string')
        'boolean' => ($kind == 'bool')
        'null' => ($value == null)
        'number' => (number? $value)
        'integer' => (number-integer? $value)
        _ => false
    }
}

def vocabulary-errors [schema: record, location: string] {
    mut errors = []
    for key in ($schema | columns) {
        if $key not-in $KEYWORDS { $errors = ($errors | append $'($location): unsupported schema keyword ($key)') }
    }
    if ($schema | get -o format) != null and $schema.format != 'date-time' {
        $errors = ($errors | append $'($location): unsupported schema format')
    }
    let types = ($schema | get -o type)
    if $types != null {
        let names = if (array? $types) { $types } else { [$types] }
        for name in $names {
            if $name not-in ['object' 'array' 'string' 'boolean' 'null' 'number' 'integer'] {
                $errors = ($errors | append $'($location): unsupported schema type')
            }
        }
    }
    let properties = ($schema | get -o properties | default {})
    for key in ($properties | columns) {
        $errors = ($errors ++ (vocabulary-errors ($properties | get $key) $'($location)/properties/($key)'))
    }
    for key in [items additionalProperties] {
        let child = ($schema | get -o $key)
        if (object? $child) {
            $errors = ($errors ++ (vocabulary-errors $child $'($location)/($key)'))
        }
    }
    $errors
}

def rfc3339? [value: string] {
    let parts = ($value | parse -r '^(?<year>[0-9]{4})-(?<month>0[1-9]|1[0-2])-(?<day>0[1-9]|[12][0-9]|3[01])[Tt](?<hour>[01][0-9]|2[0-3]):(?<minute>[0-5][0-9]):(?<second>[0-5][0-9]|60)(\.[0-9]+)?(?<offset>[Zz]|[+-]([01][0-9]|2[0-3]):[0-5][0-9])$')
    if ($parts | is-empty) { return false }
    let p = ($parts | first)
    let year = ($p.year | into int)
    let month = ($p.month | into int)
    let day = ($p.day | into int)
    let leap = (($year mod 4) == 0 and (($year mod 100) != 0 or ($year mod 400) == 0))
    let days = if $month == 2 { if $leap { 29 } else { 28 } } else if $month in [4 6 9 11] { 30 } else { 31 }
    if $day > $days { return false }
    if $p.second != '60' { return true }
    # A positive leap-second spelling must correspond to UTC month-end 23:59.
    # This checks format placement, not an observation's temporal provenance.
    let offset = if $p.offset in ['Z' 'z'] { 0 } else {
        let o = ($p.offset | parse -r '^(?<sign>[+-])(?<hours>[0-9]{2}):(?<minutes>[0-9]{2})$' | first)
        let minutes = (($o.hours | into int) * 60 + ($o.minutes | into int))
        if $o.sign == '-' { 0 - $minutes } else { $minutes }
    }
    let utc_minute = (($p.hour | into int) * 60 + ($p.minute | into int) - $offset)
    ($utc_minute == 1439 and $day == $days) or ($utc_minute == -1 and $day == 1)
}

def node-errors [value: any, schema: record, location: string] {
    mut errors = []
    let types = ($schema | get -o type)
    if $types != null {
        let names = if (array? $types) { $types } else { [$types] }
        if not ($names | any {|name| matches-type $value $name }) {
            return [$'($location): wrong type; expected ($names | str join " or ")']
        }
    }
    if 'const' in ($schema | columns) and $value != $schema.const {
        $errors = ($errors | append $'($location): does not match const')
    }
    if (object? $value) {
        let keys = ($value | columns)
        for key in ($schema | get -o required | default []) {
            if $key not-in $keys { $errors = ($errors | append $'($location).($key) is required') }
        }
        let count = ($schema | get -o minProperties)
        if $count != null and (number-compare ($keys | length) $count) < 0 {
            $errors = ($errors | append $'($location): too few properties')
        }
        let properties = ($schema | get -o properties | default {})
        let declared = ($properties | columns)
        let additional = ($schema | get -o additionalProperties)
        for key in $keys {
            if $key in $declared {
                $errors = ($errors ++ (node-errors ($value | get $key) ($properties | get $key) $'($location).($key)'))
            } else if $additional == false {
                $errors = ($errors | append $'($location).($key): additional property is not allowed')
            } else if (object? $additional) {
                $errors = ($errors ++ (node-errors ($value | get $key) $additional $'($location).($key)'))
            }
        }
    }
    if (array? $value) {
        let count = ($schema | get -o minItems)
        if $count != null and (number-compare ($value | length) $count) < 0 {
            $errors = ($errors | append $'($location): too few items')
        }
        if ($schema | get -o uniqueItems | default false) and ($value | uniq | length) != ($value | length) {
            $errors = ($errors | append $'($location): items must be unique')
        }
        let items = ($schema | get -o items)
        if $items != null {
            for item in ($value | enumerate) {
                $errors = ($errors ++ (node-errors $item.item $items $'($location)[($item.index)]'))
            }
        }
    }
    let kind = ($value | describe)
    if (number? $value) {
        let minimum = ($schema | get -o minimum)
        if $minimum != null and (number-compare $value $minimum) < 0 { $errors = ($errors | append $'($location): below minimum') }
    }
    if $kind == 'string' {
        let pattern = ($schema | get -o pattern)
        if $pattern != null and not ($value =~ $pattern) { $errors = ($errors | append $'($location): does not match pattern') }
        if ($schema | get -o format) == 'date-time' and not (rfc3339? $value) {
            $errors = ($errors | append $'($location): must be a calendar-valid RFC3339 date-time')
        }
    }
    $errors
}

export def schema-errors [value: any, schema: record, location: string] {
    if ($schema | get -o '$schema') != 'https://json-schema.org/draft/2020-12/schema' {
        return ['$schema: unsupported or missing schema dialect']
    }
    let unsupported = (vocabulary-errors $schema '$schema')
    if ($unsupported | is-not-empty) { return $unsupported }
    node-errors $value $schema $location
}
