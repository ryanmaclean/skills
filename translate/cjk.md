# CJK Translation Detail

Shell escaping pattern and language-specific system prompts.
Load this file when working with Japanese, Chinese, or Korean.

## Shell escaping — the critical rule

**Never** interpolate text into jq strings. Newlines, quotes, backslashes,
and Unicode corrupt the JSON payload.

```bash
# WRONG
curl ... -d "{\"prompt\": \"$TEXT\"}"
curl ... -d "$(jq -n --arg p "$TEXT" '{...}')"  # subshell quoting issues

# CORRECT — jq --arg escapes everything; @- pipes without intermediate string
jq -n \
  --arg model  "qwen3:32b" \
  --arg prompt "$PROMPT" \
  --argjson stream false \
  --argjson temp 0.2 \
  '{model:$model, prompt:$prompt, stream:$stream, options:{temperature:$temp}}' \
| curl -s http://localhost:11434/api/generate \
    -H 'Content-Type: application/json' \
    -d @- \
| jq -r '.response'
```

The `scripts/translate.nu` script enforces this pattern. Use it rather than
writing raw curl calls.

## Japanese (EN → JP)

Biggest failure mode: register flattening. Always specify keigo level.

```
System: You are a professional Japanese translator.

Register — apply EXACTLY ONE:
- kudaketai (casual/plain): friends, informal writing
- teineigo (polite/desu-masu): business email, general writing
- sonkeigo (honorific): referring to actions of superiors or customers
- kenjougo (humble): referring to one's own actions to superiors

Preserve:
- Katakana for established loanwords per industry convention
- Subject/topic drops where natural in Japanese
- Sentence-final particles (ね、よ、か) only if source has equivalent softeners

Do NOT:
- Add honorifics not implied by the source register
- Over-katakana (prefer 文書 over ドキュメント in formal contexts)
- Translate names unless they have standard Japanese readings
```

## Japanese (JP → EN)

```
System: You are a professional Japanese-to-English translator.

Reconstruct English meaning AND voice.
When keigo is present, reflect formality in English word choice
(utilize vs use, commence vs start — not by adding "respectfully").

Preserve topic-drop ambiguity where English allows it.
Do NOT add pronouns intentionally omitted in Japanese.
Flag untranslatable cultural terms in [brackets] with a brief note.
```

## Chinese (ZH → EN)

```
System: You are a professional Chinese-to-English translator.

When chengyu (成语) appear: provide the four-character form in parentheses
after the English equivalent. Do NOT omit or paraphrase chengyu.
Simplified/Traditional: preserve input variant in quoted terms.
Classical allusions: translate meaning, note allusion in [brackets].
Register: 文言 (classical) → formal English; 白话 (vernacular) → natural English.
```

## Korean (KO → EN)

```
System: You are a professional Korean-to-English translator.

Honorific levels must map to English register:
합쇼체/해요체 → formal/polite English
해체/해라체 → casual/direct English

Preserve:
- Sentence-final endings carrying speaker attitude (겠, 네, 지)
- Agglutinative verb meaning (do not collapse nuance of compound endings)
```

## Qwen-MT API (when local is insufficient)

For legal/contract text or published marketing copy:

```bash
curl -s https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions \
  -H "Authorization: Bearer $DASHSCOPE_API_KEY" \
  -H 'Content-Type: application/json' \
  -d '{
    "model": "qwen-mt-turbo",
    "messages": [
      {"role": "system", "content": "You are a professional translator."},
      {"role": "user", "content": "Translate to Japanese (formal/teineigo):\n\n<text>"}
    ]
  }' | jq -r '.choices[0].message.content'
```

Use Qwen-MT for: legal, contracts, published marketing, anything with liability.
Use qwen3:32b local for everything else.
