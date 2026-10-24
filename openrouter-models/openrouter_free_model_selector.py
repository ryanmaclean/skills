#!/usr/bin/env python3
"""OpenRouter Free Model Selector - Tests and selects best available free model."""

import json
import os
import sys
import time
from typing import Optional
import urllib.request
import urllib.error

OPENROUTER_API_KEY = os.getenv(
    "OPENROUTER_API_KEY",
    ""
)

# Preferred free models in priority order (best for coding tasks first)
PREFERRED_FREE_MODELS = [
    "deepseek/deepseek-r1-0528:free",
    "qwen/qwen3-coder:free",
    "meta-llama/llama-3.3-70b-instruct:free",
    "mistralai/mistral-small-3.1-24b-instruct:free",
    "google/gemma-3-27b-it:free",
    "qwen/qwen3-4b:free",
    "openai/gpt-oss-120b:free",
    "nvidia/nemotron-nano-9b-v2:free",
]

# Fallback paid models (cheap)
FALLBACK_PAID_MODELS = [
    "anthropic/claude-3-haiku",
    "openai/gpt-4o-mini",
    "google/gemini-flash-1.5",
]


def test_model(model_id: str, timeout: int = 30) -> tuple[bool, float, str]:
    """Test if a model responds correctly. Returns (success, latency_ms, response)."""
    url = "https://openrouter.ai/api/v1/chat/completions"
    headers = {
        "Authorization": f"Bearer {OPENROUTER_API_KEY}",
        "Content-Type": "application/json",
        "HTTP-Referer": "https://github.com/gastown",
        "X-Title": "GasTown Model Selector",
    }
    data = json.dumps({
        "model": model_id,
        "messages": [{"role": "user", "content": "Reply with exactly: OK"}],
        "max_tokens": 10,
        "temperature": 0,
    }).encode()

    req = urllib.request.Request(url, data=data, headers=headers, method="POST")

    start = time.time()
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            result = json.loads(resp.read().decode())
            latency = (time.time() - start) * 1000
            content = result.get("choices", [{}])[0].get("message", {}).get("content", "")
            return True, latency, content.strip()
    except urllib.error.HTTPError as e:
        return False, 0, f"HTTP {e.code}: {e.reason}"
    except urllib.error.URLError as e:
        return False, 0, f"URL Error: {e.reason}"
    except Exception as e:
        return False, 0, str(e)


def get_available_free_models() -> list[str]:
    """Fetch current list of free models from OpenRouter."""
    url = "https://openrouter.ai/api/v1/models"
    headers = {"Authorization": f"Bearer {OPENROUTER_API_KEY}"}
    req = urllib.request.Request(url, headers=headers)

    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            data = json.loads(resp.read().decode())
            free_models = []
            for model in data.get("data", []):
                pricing = model.get("pricing", {})
                prompt_cost = float(pricing.get("prompt", 1))
                if prompt_cost == 0:
                    free_models.append(model["id"])
            return free_models
    except Exception as e:
        print(f"Warning: Could not fetch models: {e}", file=sys.stderr)
        return []


def select_best_model(verbose: bool = False) -> Optional[str]:
    """Test models and return the best working one."""
    available = set(get_available_free_models())

    # Test preferred free models first
    for model in PREFERRED_FREE_MODELS:
        if model not in available:
            if verbose:
                print(f"  {model}: not available", file=sys.stderr)
            continue

        success, latency, response = test_model(model)
        if success:
            if verbose:
                print(f"  {model}: OK ({latency:.0f}ms) -> '{response}'", file=sys.stderr)
            return model
        else:
            if verbose:
                print(f"  {model}: FAILED - {response}", file=sys.stderr)

    # Try any other available free model
    for model in available:
        if model in PREFERRED_FREE_MODELS:
            continue
        success, latency, response = test_model(model)
        if success:
            if verbose:
                print(f"  {model}: OK ({latency:.0f}ms)", file=sys.stderr)
            return model

    # Fall back to paid models
    if verbose:
        print("  No free models working, trying paid fallbacks...", file=sys.stderr)
    for model in FALLBACK_PAID_MODELS:
        success, latency, response = test_model(model)
        if success:
            if verbose:
                print(f"  {model}: OK (paid, {latency:.0f}ms)", file=sys.stderr)
            return model

    return None


def main():
    import argparse
    parser = argparse.ArgumentParser(description="Select best available OpenRouter model")
    parser.add_argument("-v", "--verbose", action="store_true", help="Show test results")
    parser.add_argument("--test", metavar="MODEL", help="Test a specific model")
    parser.add_argument("--list", action="store_true", help="List all free models")
    parser.add_argument("--json", action="store_true", help="Output as JSON")
    args = parser.parse_args()

    if args.list:
        models = get_available_free_models()
        if args.json:
            print(json.dumps(models, indent=2))
        else:
            for m in sorted(models):
                print(m)
        return 0

    if args.test:
        success, latency, response = test_model(args.test)
        if args.json:
            print(json.dumps({"model": args.test, "success": success, "latency_ms": latency, "response": response}))
        else:
            status = "OK" if success else "FAILED"
            print(f"{args.test}: {status} ({latency:.0f}ms) -> {response}")
        return 0 if success else 1

    if args.verbose:
        print("Testing models...", file=sys.stderr)

    model = select_best_model(verbose=args.verbose)

    if model:
        if args.json:
            print(json.dumps({"selected_model": model}))
        else:
            print(model)
        return 0
    else:
        print("ERROR: No working models found", file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
