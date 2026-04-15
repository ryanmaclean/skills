#!/usr/bin/env nu
# Local translation via Ollama + Qwen3.
# Script code never enters context; only output does.
#
# Shell escaping rule enforced here:
#   - Text passed via jq --arg (handles all escaping)
#   - Payload piped to curl via @- (no subshell quoting)
#   - Never: -d "{\"prompt\": \"$TEXT\"}"

def build-payload [model: string, prompt: string]: nothing -> string {
    jq -rn --arg model $model --arg prompt $prompt \
        --argjson stream false --argjson temp 0.2 \
        '{model: $model, prompt: $prompt, stream: $stream, options: {temperature: $temp}}'
}

def call-ollama [payload: string]: nothing -> string {
    $payload
    | curl -s http://localhost:11434/api/generate
        -H 'Content-Type: application/json'
        -d @-
    | jq -r '.response'
}

def make-prompt [
    target_lang: string
    register: string
    text: string
    think: bool
]: nothing -> string {
    let mode = if $think { "/think" } else { "/no_think" }
    $"($mode)\n\nTranslate to ($target_lang). Register: ($register).
Preserve technical terms, CLI flags, and code identifiers untranslated.
Output ONLY the translation.\n\nTEXT:\n($text)"
}

def main [
    target_lang: string      # e.g. "Japanese", "Chinese", "Korean", "French"
    register: string         # e.g. "technical-formal", "casual", "polite", "keigo"
    text?: string            # Text to translate (or use --file)
    --file: string = ""      # Read input from file instead
    --model: string = "qwen3:32b"
    --think                  # Use /think mode (slower; for ambiguous/literary text)
    --roundtrip              # Run forward + back translation for validation
] {
    # Resolve input text
    let input = if ($file | is-empty) {
        if ($text == null) {
            error make { msg: "Provide text argument or --file path" }
        }
        $text
    } else {
        open --raw $file
    }

    let prompt = make-prompt $target_lang $register $input $think
    let payload = build-payload $model $prompt
    let forward = call-ollama $payload

    if not $roundtrip {
        print $forward
        return
    }

    # Round-trip: translate back to English
    let back_prompt = make-prompt "English" "natural" $forward false
    let back_payload = build-payload $model $back_prompt
    let backward = call-ollama $back_payload

    print "=== ORIGINAL ==="
    print $input
    print $"\n=== ($target_lang) ==="
    print $forward
    print "\n=== BACK TO ENGLISH ==="
    print $backward
}
