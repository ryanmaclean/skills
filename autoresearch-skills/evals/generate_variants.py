#!/usr/bin/env python3
"""
Generate improved prompt variants based on failure analysis.

Used by Claude Code during AutoResearch loop to create targeted improvements.
"""

import json
import argparse
from pathlib import Path
from datetime import datetime
from typing import List, Dict, Any


def load_latest_results(skill_name: str) -> Dict[str, Any]:
    """Load the latest evaluation results for analysis."""
    results_file = Path(__file__).parent.parent / "results" / "latest_run.json"
    
    if not results_file.exists():
        return {}
    
    with open(results_file, 'r') as f:
        results = json.load(f)
    
    # Filter results for the specific skill
    if results.get("skill") != skill_name:
        return {}
    
    return results


def load_current_prompt(skill_name: str) -> str:
    """Load the current prompt for a skill."""
    prompt_file = Path(__file__).parent.parent / "prompts" / "current" / f"{skill_name}.md"
    
    if not prompt_file.exists():
        raise FileNotFoundError(f"Current prompt not found for skill: {skill_name}")
    
    with open(prompt_file, 'r') as f:
        return f.read()


def save_variant(skill_name: str, variant_id: str, prompt: str) -> None:
    """Save a generated variant."""
    candidates_dir = Path(__file__).parent.parent / "prompts" / "candidates"
    candidates_dir.mkdir(exist_ok=True)
    
    variant_file = candidates_dir / f"{skill_name}_{variant_id}.md"
    with open(variant_file, 'w') as f:
        f.write(prompt)


def analyze_failures(results: Dict[str, Any]) -> List[str]:
    """Analyze failed assertions to identify improvement areas."""
    if not results or "test_results" not in results:
        return ["No previous results available - generate general improvements"]
    
    failure_patterns = {}
    
    for test_result in results["test_results"]:
        if not test_result["passed"]:
            for assertion in test_result["assertions"]:
                if not assertion["passed"]:
                    assertion_name = assertion["name"]
                    if assertion_name not in failure_patterns:
                        failure_patterns[assertion_name] = []
                    failure_patterns[assertion_name].append({
                        "test_id": test_result["test_id"],
                        "reason": assertion["reason"]
                    })
    
    # Generate improvement suggestions based on failure patterns
    suggestions = []
    
    if "Required sections" in failure_patterns:
        suggestions.append("Add missing required sections with proper structure")
    
    if "Has examples" in failure_patterns:
        suggestions.append("Include more practical code examples and command samples")
    
    if "Addresses intent" in failure_patterns:
        suggestions.append("Better align content with user's specific questions and needs")
    
    if "Markdown format" in failure_patterns:
        suggestions.append("Fix markdown formatting issues (tables, code blocks, headings)")
    
    if "No PII exposure" in failure_patterns:
        suggestions.append("Remove any hardcoded paths, usernames, or sensitive information")
    
    if "Length constraints" in failure_patterns:
        suggestions.append("Adjust content length to be more concise or more comprehensive")
    
    if "Skill-specific" in failure_patterns:
        suggestions.append("Include more domain-specific terminology and concepts")
    
    if not suggestions:
        suggestions.append("General refinement: improve clarity, add examples, enhance structure")
    
    return suggestions


def generate_variant_1(base_prompt: str, improvement_areas: List[str], skill_name: str) -> str:
    """Generate variant 1: Enhanced structure and examples."""
    variant = base_prompt
    
    # Add emphasis on structure
    if "Add missing required sections" in improvement_areas:
        variant += "\n\nIMPORTANT: Always structure responses with clear sections using ## headers."
    
    # Add emphasis on examples
    if "Include more practical code examples" in improvement_areas:
        variant += "\n\nInclude specific command examples, YAML configurations, and code snippets for every major concept."
    
    # Add skill-specific enhancements
    if skill_name == "ansible-fleet":
        variant += "\n\nAlways include actual playbook snippets and inventory examples."
    elif skill_name == "ollama-deploy":
        variant += "\n\nProvide specific Ollama commands and model management examples."
    elif skill_name == "raspberry-pi":
        variant += "\n\nAddress ARM64-specific considerations and Pi hardware differences."
    
    return variant


def generate_variant_2(base_prompt: str, improvement_areas: List[str], skill_name: str) -> str:
    """Generate variant 2: Better user intent alignment."""
    variant = base_prompt
    
    # Add emphasis on user needs
    if "Better align content with user's specific questions" in improvement_areas:
        variant += "\n\nCarefully analyze the user's specific question and provide targeted, relevant information. Avoid generic responses."
    
    # Add emphasis on practical solutions
    variant += "\n\nFocus on solving real problems with actionable steps and troubleshooting guidance."
    
    # Add safety and verification
    variant += "\n\nAlways include verification steps and safety checks for deployments."
    
    return variant


def generate_variant_3(base_prompt: str, improvement_areas: List[str], skill_name: str) -> str:
    """Generate variant 3: Comprehensive coverage."""
    variant = base_prompt
    
    # Add comprehensive coverage instruction
    variant += "\n\nProvide comprehensive coverage including: overview, prerequisites, step-by-step instructions, examples, common issues, and troubleshooting."
    
    # Add formatting emphasis
    if "Fix markdown formatting issues" in improvement_areas:
        variant += "\n\nUse proper markdown formatting with tables, code blocks with language specifiers, and clear heading hierarchy."
    
    # Add domain depth
    variant += f"\n\nDemonstrate deep expertise in {skill_name.replace('-', ' ')} with advanced insights and best practices."
    
    return variant


def main():
    parser = argparse.ArgumentParser(description="Generate improved prompt variants")
    parser.add_argument("--skill", required=True, help="Skill name to generate variants for")
    parser.add_argument("--count", type=int, default=3, help="Number of variants to generate")
    parser.add_argument("--analysis", action="store_true", help="Show failure analysis")
    
    args = parser.parse_args()
    
    # Load current prompt and results
    try:
        base_prompt = load_current_prompt(args.skill)
        results = load_latest_results(args.skill)
    except FileNotFoundError as e:
        print(f"Error: {e}", file=sys.stderr)
        return 1
    
    # Analyze failures
    improvement_areas = analyze_failures(results)
    
    if args.analysis:
        print(f"Failure analysis for {args.skill}:")
        for area in improvement_areas:
            print(f"  - {area}")
        print()
    
    # Generate variants
    variants = [
        ("structure", generate_variant_1(base_prompt, improvement_areas, args.skill)),
        ("alignment", generate_variant_2(base_prompt, improvement_areas, args.skill)),
        ("comprehensive", generate_variant_3(base_prompt, improvement_areas, args.skill))
    ]
    
    # Save variants
    for i, (variant_name, variant_content) in enumerate(variants[:args.count]):
        variant_id = f"{variant_name}_{datetime.now().strftime('%H%M%S')}"
        save_variant(args.skill, variant_id, variant_content)
        print(f"Generated variant {i+1}: {variant_id}")
    
    print(f"\nGenerated {min(args.count, 3)} variants for {args.skill}")
    print("Next step: Evaluate each variant using:")
    for i, (variant_name, _) in enumerate(variants[:args.count]):
        variant_id = f"{variant_name}_{datetime.now().strftime('%H%M%S')}"
        print(f"  python evals/runner.py --skill {args.skill} --variant {variant_id}")


if __name__ == "__main__":
    import sys
    main()
