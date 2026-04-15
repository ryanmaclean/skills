---
name: translate
description: >
  Local multilingual translation via Ollama — 119 languages, no API key, no network,
  no data leaves the machine. Use when translating to/from Japanese, Chinese, Korean,
  French, Spanish, German, Portuguese, Russian, Arabic, or any other language.
  Also for localization, i18n text, and roundtrip validation.
  Qwen3:32b on Apple Silicon — faster and cheaper than cloud APIs.
---

# translate — Local Translation via Ollama

Offline-first. Qwen3 via Ollama. No data leaves the machine.

## Models (64GB unified memory)

| Model | Size | Use |
|-------|------|-----|
| `qwen3:32b` | 20GB | Default — best quality |
| `qwen3:30b` | 19GB MoE | Fast — near-same quality |
| `qwen3:14b` | 9.3GB | Acceptable quality, fastest |

```bash
ollama pull qwen3:32b   # one-time
```

## Run the script

```bash
# Translate text to a target language
nu scripts/translate.nu "Japanese" "technical-formal" "Your text here"

# From a file
nu scripts/translate.nu "Japanese" "technical-formal" \
  --file /path/to/input.txt

# Round-trip validation (EN -> JP -> EN)
nu scripts/translate.nu "Japanese" "technical-formal" \
  "Your text" --roundtrip
```

## Workflow checklist

```
Translation Progress:
- [ ] 1. Choose target language and register (see registers table below)
- [ ] 2. Run: nu scripts/translate.nu "Language" "register" "text"
- [ ] 3. Verify: does output read naturally? Check for register flattening.
- [ ] 4. For critical text: run --roundtrip and compare back-translation
- [ ] 5. For legal/published copy: use Qwen-MT API instead (see cjk.md)
```

## Registers (Japanese)

| Context | Register |
|---------|----------|
| Engineering spec, PR | `technical-formal` |
| Stakeholder report | `formal` |
| Slack / team chat | `casual` |
| Customer-facing docs | `polite` |
| Legal / contract | `keigo` |

## Thinking mode

- `/no_think` — default, faster, use for technical docs and UI strings
- `/think` — use for literary passages, ambiguous register, irony

## CJK detail

For CJK-specific system prompts, register rules, honorific mapping,
chengyu handling, and Korean verb ending nuances: see [cjk.md](cjk.md)

## Shell escaping rule

**Never** interpolate text into jq strings. Always `--arg` + pipe via `@-`.
The script handles this correctly. See [cjk.md](cjk.md) for the pattern.

## Limitations

- Qwen3:32b local vs Qwen-MT API: local strong, Qwen-MT better for legal/published
- Proper nouns: verify katakana readings for obscure new names
- No pitch accent reconstruction for TTS output
