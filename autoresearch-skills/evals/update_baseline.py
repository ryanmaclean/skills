#!/usr/bin/env python3
"""
Update baseline prompt with the best performing variant.

Used by Claude Code to promote a winning variant to the current baseline.
"""

import json
import argparse
import shutil
from pathlib import Path
from datetime import datetime


def load_latest_results() -> Dict[str, Any]:
    """Load the latest evaluation results."""
    results_file = Path(__file__).parent.parent / "results" / "latest_run.json"
    
    if not results_file.exists():
        raise FileNotFoundError("No latest results found. Run evaluation first.")
    
    with open(results_file, 'r') as f:
        return json.load(f)


def load_variant(skill_name: str, variant_id: str) -> str:
    """Load a specific variant."""
    variant_file = Path(__file__).parent.parent / "prompts" / "candidates" / f"{skill_name}_{variant_id}.md"
    
    if not variant_file.exists():
        raise FileNotFoundError(f"Variant not found: {variant_file}")
    
    with open(variant_file, 'r') as f:
        return f.read()


def save_to_history(skill_name: str, prompt: str, score: float) -> None:
    """Save the old baseline to history with score in filename."""
    history_dir = Path(__file__).parent.parent / "prompts" / "history"
    history_dir.mkdir(exist_ok=True)
    
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    history_file = history_dir / f"{skill_name}_{score:.1f}_{timestamp}.md"
    
    with open(history_file, 'w') as f:
        f.write(prompt)
    
    print(f"Saved previous baseline to history: {history_file.name}")


def update_baseline(skill_name: str, variant_id: str) -> None:
    """Update the current baseline with the specified variant."""
    # Load current baseline for history
    current_file = Path(__file__).parent.parent / "prompts" / "current" / f"{skill_name}.md"
    old_prompt = ""
    if current_file.exists():
        with open(current_file, 'r') as f:
            old_prompt = f.read()
    
    # Load variant
    new_prompt = load_variant(skill_name, variant_id)
    
    # Get latest score for history
    results = load_latest_results()
    score = results.get("pass_rate", 0.0)
    
    # Save old to history
    if old_prompt:
        save_to_history(skill_name, old_prompt, score)
    
    # Update baseline
    with open(current_file, 'w') as f:
        f.write(new_prompt)
    
    print(f"Updated baseline for {skill_name} with variant {variant_id}")
    print(f"New baseline score: {score:.1f}% pass rate")


def main():
    parser = argparse.ArgumentParser(description="Update baseline with winning variant")
    parser.add_argument("--skill", required=True, help="Skill name to update")
    parser.add_argument("--variant", required=True, help="Variant ID to promote")
    
    args = parser.parse_args()
    
    try:
        update_baseline(args.skill, args.variant)
        print("Baseline updated successfully!")
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    import sys
    sys.exit(main())
