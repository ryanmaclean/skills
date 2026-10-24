---
name: openrouter-models
description: "Find and test free OpenRouter models. Lists available free models, tests their availability, and selects the best working model for code generation tasks. Use this before spawning agents to select cost-effective models."
author: gastown
version: 1.0.0
---

# OpenRouter Free Model Selector

Find and test free models on OpenRouter to minimize costs while maintaining quality.

## What This Skill Does

- **List Free Models**: Fetch all models with $0 pricing from OpenRouter
- **Test Models**: Verify a model responds correctly before use
- **Auto-Select**: Find the best available free model for coding tasks
- **Fallback**: Gracefully fall back to cheap paid models if needed

## Prerequisites

Set the OpenRouter API key:

```bash
export OPENROUTER_API_KEY="sk-or-v1-..."
```

Or the skill uses the default free-tier key.

## Usage

### List All Free Models
```bash
python3 scripts/openrouter_free_model_selector.py --list
```

### Test a Specific Model
```bash
python3 scripts/openrouter_free_model_selector.py --test "qwen/qwen3-coder:free"
```

### Auto-Select Best Model (Verbose)
```bash
python3 scripts/openrouter_free_model_selector.py -v
```

### Get Model ID Only (for scripting)
```bash
MODEL=$(python3 scripts/openrouter_free_model_selector.py)
echo "Using: $MODEL"
```

### JSON Output
```bash
python3 scripts/openrouter_free_model_selector.py --list --json
python3 scripts/openrouter_free_model_selector.py --json
```

## Preferred Free Models (Priority Order)

1. `deepseek/deepseek-r1-0528:free` - DeepSeek R1 (best reasoning)
2. `qwen/qwen3-coder:free` - Qwen Coder (optimized for code)
3. `meta-llama/llama-3.3-70b-instruct:free` - Llama 3.3 70B
4. `mistralai/mistral-small-3.1-24b-instruct:free` - Mistral Small
5. `google/gemma-3-27b-it:free` - Gemma 3 27B

## Integration with Gas Town

To use with polecats:
```bash
# Select best free model
MODEL=$(python3 scripts/openrouter_free_model_selector.py)

# Configure for agent use
export OPENROUTER_MODEL="$MODEL"
```

## Fallback Behavior

If no free models work, falls back to:
1. `anthropic/claude-3-haiku` (cheap)
2. `openai/gpt-4o-mini` (cheap)
3. `google/gemini-flash-1.5` (cheap)
