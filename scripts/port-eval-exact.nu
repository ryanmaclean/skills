# SPDX-License-Identifier: MIT
# JSON numbers retain their lexical bytes as Nu binary values. JSON syntax
# cannot construct a binary value, so input objects/strings cannot impersonate
# this internal number representation. No binary marker is emitted in JSON.

const NUMBER = '^-?(0|[1-9][0-9]*)(\.[0-9]+)?([eE][+-]?[0-9]+)?$'

export def number? [value: any] {
    let kind = ($value | describe)
    $kind == 'binary' or $kind == 'int'
}
export def number-text [value: any] {
    let kind = ($value | describe)
    let text = if $kind == 'binary' { $value | decode utf-8 } else if $kind == 'int' { $value | into string } else {
        error make {msg: 'exact number required; machine floats are not admitted'}
    }
    if not ($text =~ $NUMBER) { error make {msg: 'invalid internal numeric lexeme'} }
    $text
}
def unsigned [digits: string] {
    let trimmed = ($digits | str replace -r '^0+' '')
    if $trimmed == '' { '0' } else { $trimmed }
}
def unsigned-cmp [left: string, right: string] {
    let a = (unsigned $left)
    let b = (unsigned $right)
    if ($a | str length) < ($b | str length) { -1 } else if ($a | str length) > ($b | str length) { 1 } else if $a == $b { 0 } else if $a < $b { -1 } else { 1 }
}
def unsigned-add [left: string, right: string] {
    let a = ($left | split chars | reverse)
    let b = ($right | split chars | reverse)
    let count = ([($a | length) ($b | length)] | math max)
    mut result = []
    mut carry = 0
    for i in 0..<$count {
        let total = (($a | get -o $i | default '0' | into int) + ($b | get -o $i | default '0' | into int) + $carry)
        $result = ($result ++ [($total mod 10 | into string)])
        $carry = ($total // 10)
    }
    if $carry > 0 { $result = ($result ++ [($carry | into string)]) }
    $result | reverse | str join
}
# Precondition left >= right, verified by signed-add.
def unsigned-sub [left: string, right: string] {
    let a = ($left | split chars | reverse)
    let b = ($right | split chars | reverse)
    mut result = []
    mut borrow = 0
    for i in 0..<($a | length) {
        let digit = (($a | get $i | into int) - ($b | get -o $i | default '0' | into int) - $borrow)
        $result = ($result ++ [(if $digit < 0 { $digit + 10 } else { $digit } | into string)])
        $borrow = (if $digit < 0 { 1 } else { 0 })
    }
    unsigned ($result | reverse | str join)
}
def signed [text: string] {
    let negative = ($text | str starts-with '-')
    let magnitude = (unsigned ($text | str replace -r '^[+-]' ''))
    {negative: ($negative and $magnitude != '0'), magnitude: $magnitude}
}
def signed-text [negative: bool, magnitude: string] {
    let digits = (unsigned $magnitude)
    if $negative and $digits != '0' { '-' + $digits } else { $digits }
}
def signed-add [left: string, right: string] {
    let a = (signed $left)
    let b = (signed $right)
    if $a.negative == $b.negative { return (signed-text $a.negative (unsigned-add $a.magnitude $b.magnitude)) }
    let order = (unsigned-cmp $a.magnitude $b.magnitude)
    if $order == 0 { '0' } else if $order > 0 { signed-text $a.negative (unsigned-sub $a.magnitude $b.magnitude) } else { signed-text $b.negative (unsigned-sub $b.magnitude $a.magnitude) }
}
def signed-cmp [left: string, right: string] {
    let a = (signed $left)
    let b = (signed $right)
    if $a.negative != $b.negative { return (if $a.negative { -1 } else { 1 }) }
    let order = (unsigned-cmp $a.magnitude $b.magnitude)
    if $a.negative { 0 - $order } else { $order }
}
def parts [value: any] {
    let text = (number-text $value)
    let p = ($text | parse -r '^(?<sign>-?)(?<whole>0|[1-9][0-9]*)(\.(?<fraction>[0-9]+))?([eE](?<exponent>[+-]?[0-9]+))?$' | first)
    let fraction = ($p | get -o fraction | default '')
    let exponent_text = ($p | get -o exponent | default '')
    let exponent = if $exponent_text == '' { '0' } else { $exponent_text }
    let digits = (unsigned ($p.whole + $fraction))
    if $digits == '0' { return {negative: false, coefficient: '0', exponent: '0'} }
    let coefficient = ($digits | str replace -r '0+$' '')
    let removed = (($digits | str length) - ($coefficient | str length))
    let shift = ($removed - ($fraction | str length))
    {negative: ($p.sign == '-'), coefficient: $coefficient, exponent: (signed-add $exponent ($shift | into string))}
}
export def number-integer? [value: any] {
    if not (number? $value) { return false }
    let p = (parts $value)
    $p.coefficient == '0' or (signed-cmp $p.exponent '0') >= 0
}
export def number-compare [left: any, right: any] {
    let a = (parts $left)
    let b = (parts $right)
    if $a.coefficient == '0' and $b.coefficient == '0' { return 0 }
    if $a.negative != $b.negative { return (if $a.negative { -1 } else { 1 }) }
    mut order = 0
    if $a.coefficient == '0' { $order = -1 } else if $b.coefficient == '0' { $order = 1 } else {
        let ad = (signed-add $a.exponent ($a.coefficient | str length | into string))
        let bd = (signed-add $b.exponent ($b.coefficient | str length | into string))
        $order = (signed-cmp $ad $bd)
        if $order == 0 {
            let ac = ($a.coefficient | split chars)
            let bc = ($b.coefficient | split chars)
            let count = ([($ac | length) ($bc | length)] | math max)
            for i in 0..<$count {
                let x = ($ac | get -o $i | default '0' | into int)
                let y = ($bc | get -o $i | default '0' | into int)
                if $x != $y { $order = (if $x < $y { -1 } else { 1 }); break }
            }
        }
    }
    if $a.negative { 0 - $order } else { $order }
}
# Conversion is allowed only after a caller-proven small bound, never to decide
# whether an unbounded input satisfies the schema's integer/minimum assertion.
export def number-small [value: any, maximum: int] {
    if not (number-integer? $value) or (number-compare $value 0) < 0 or (number-compare $value $maximum) > 0 {
        error make {msg: 'number is outside the proven small integer computation range'}
    }
    let p = (parts $value)
    mut text = $p.coefficient
    let places = ($p.exponent | into int)
    for _ in 0..<$places { $text = $text + '0' }
    $text | into int
}
def multiply-small [digits: string, multiplier: int] {
    mut carry = 0
    mut result = []
    for digit in ($digits | split chars | reverse) {
        let product = (($digit | into int) * $multiplier + $carry)
        $result = ($result ++ [($product mod 10 | into string)])
        $carry = ($product // 10)
    }
    while $carry > 0 {
        $result = ($result ++ [($carry mod 10 | into string)])
        $carry = ($carry // 10)
    }
    unsigned ($result | reverse | str join)
}
export def number-times-small [value: any, multiplier: int] {
    if $multiplier < 0 or $multiplier > 60000 { error make {msg: 'internal multiplier outside its declared source range'} }
    let p = (parts $value)
    let digits = (multiply-small $p.coefficient $multiplier)
    let sign = if $p.negative and $digits != '0' { '-' } else { '' }
    ($sign + $digits + 'e' + $p.exponent) | encode utf-8
}
# The existing wall-clock projection rounds nonnegative minutes*60000 to the
# nearest integer, ties upward. Decimal arithmetic handles this without float
# overflow/underflow and retains huge positive exponents in JSON notation.
export def number-rounded-ms [value: any] {
    if (number-compare $value 0) < 0 { error make {msg: 'negative wall time'} }
    let scaled = (number-times-small $value 60000)
    let p = (parts $scaled)
    if (signed-cmp $p.exponent '0') >= 0 { return $scaled }
    let count = ($p.coefficient | str length)
    if (signed-cmp $p.exponent ((0 - $count) | into string)) < 0 { return ('0' | encode utf-8) }
    # The comparison above proves -count <= exponent < 0 before conversion.
    let cut = ($count + ($p.exponent | into int))
    let digits = ($p.coefficient | split chars)
    let whole = (unsigned ($digits | take $cut | str join))
    let rounded = if ($digits | get $cut | into int) >= 5 { unsigned-add $whole '1' } else { $whole }
    $rounded | encode utf-8
}

# Only string leaves use Nu's JSON decoder: no numeric token reaches a bounded
# decoder, including schema-valid huge counts and huge/tiny exponents.
def skip-space [chars: list<string>, start: int] {
    mut pos = $start
    while $pos < ($chars | length) and ($chars | get $pos) in [' ' "\t" "\r" "\n"] { $pos = $pos + 1 }
    $pos
}
def json-string [chars: list<string>, start: int] {
    mut pos = ($start + 1)
    mut escaped = false
    while $pos < ($chars | length) {
        let char = ($chars | get $pos)
        if not $escaped and $char == '"' {
            let token = ($chars | skip $start | take ($pos - $start + 1) | str join)
            return {value: ($token | from json --strict), next: ($pos + 1)}
        }
        if $escaped { $escaped = false } else if $char == '\' { $escaped = true }
        $pos = $pos + 1
    }
    error make {msg: 'unterminated JSON string'}
}
def json-node [chars: list<string>, start: int] {
    mut pos = (skip-space $chars $start)
    if $pos >= ($chars | length) { error make {msg: 'missing JSON value'} }
    let char = ($chars | get $pos)
    if $char == '"' { return (json-string $chars $pos) }
    if $char == '{' {
        mut object = {}
        $pos = (skip-space $chars ($pos + 1))
        if ($chars | get -o $pos) == '}' { return {value: $object, next: ($pos + 1)} }
        loop {
            if ($chars | get -o $pos) != '"' { error make {msg: 'JSON object key must be a string'} }
            let key = (json-string $chars $pos)
            $pos = (skip-space $chars $key.next)
            if ($chars | get -o $pos) != ':' { error make {msg: 'missing JSON colon'} }
            let node = (json-node $chars ($pos + 1))
            # Match the underlying object model: later duplicate keys replace
            # earlier values. This is not a duplicate-key security guarantee.
            $object = ($object | upsert $key.value $node.value)
            $pos = (skip-space $chars $node.next)
            if ($chars | get -o $pos) == '}' { return {value: $object, next: ($pos + 1)} }
            if ($chars | get -o $pos) != ',' { error make {msg: 'missing JSON object separator'} }
            $pos = (skip-space $chars ($pos + 1))
        }
    }
    if $char == '[' {
        mut items = []
        $pos = (skip-space $chars ($pos + 1))
        if ($chars | get -o $pos) == ']' { return {value: $items, next: ($pos + 1)} }
        loop {
            let node = (json-node $chars $pos)
            $items = ($items ++ [$node.value])
            $pos = (skip-space $chars $node.next)
            if ($chars | get -o $pos) == ']' { return {value: $items, next: ($pos + 1)} }
            if ($chars | get -o $pos) != ',' { error make {msg: 'missing JSON array separator'} }
            $pos = (skip-space $chars ($pos + 1))
        }
    }
    let begin = $pos
    while $pos < ($chars | length) and ($chars | get $pos) not-in [',' ']' '}' ' ' "\t" "\r" "\n"] { $pos = $pos + 1 }
    let token = ($chars | skip $begin | take ($pos - $begin) | str join)
    let value = match $token {
        'true' => true
        'false' => false
        'null' => null
        _ => {
            if not ($token =~ $NUMBER) { error make {msg: 'invalid JSON literal or number'} }
            $token | encode utf-8
        }
    }
    {value: $value, next: $pos}
}
export def exact-from-json [text: string] {
    let chars = ($text | split chars)
    let node = (json-node $chars 0)
    if (skip-space $chars $node.next) != ($chars | length) { error make {msg: 'trailing JSON content'} }
    $node.value
}
export def exact-to-json [value: any] {
    let kind = ($value | describe)
    if $kind == 'binary' { return (number-text $value) }
    if ($kind | str starts-with 'record') {
        let fields = ($value | columns | each {|key|
            ($key | to json --raw) + ':' + (exact-to-json ($value | get $key))
        })
        return ('{' + ($fields | str join ',') + '}')
    }
    if ($kind | str starts-with 'list') or ($kind | str starts-with 'table') {
        return ('[' + ($value | each {|item| exact-to-json $item } | str join ',') + ']')
    }
    if $kind not-in [int string bool nothing] { error make {msg: 'unsupported internal JSON value; refusing lossy serialization'} }
    $value | to json --raw
}
