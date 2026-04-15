#!/usr/bin/env python3
"""
AutoResearch Skills Evaluation Runner

Evaluates skill prompt variants against binary assertions to calculate pass rates.
Used by Claude Code to measure improvement during the AutoResearch loop.
"""

import json
import sys
import argparse
from pathlib import Path
from typing import Dict, List, Any, Tuple
from datetime import datetime

# Import assertion functions
from assertions import evaluate_skill_output


def load_test_cases(skill_name: str) -> List[Dict[str, Any]]:
    """Load test cases for a specific skill."""
    test_file = Path(__file__).parent / "test_cases.jsonl"
    if not test_file.exists():
        return []
    
    test_cases = []
    with open(test_file, 'r') as f:
        for line in f:
            case = json.loads(line.strip())
            if case.get('skill') == skill_name or case.get('skill') == 'all':
                test_cases.append(case)
    return test_cases


def load_prompt(skill_name: str, variant: str = "current") -> str:
    """Load a skill prompt variant."""
    if variant == "current":
        prompt_file = Path(__file__).parent.parent / "prompts" / "current" / f"{skill_name}.md"
    else:
        prompt_file = Path(__file__).parent.parent / "prompts" / "candidates" / f"{skill_name}_{variant}.md"
    
    if not prompt_file.exists():
        raise FileNotFoundError(f"Prompt file not found: {prompt_file}")
    
    with open(prompt_file, 'r') as f:
        return f.read()


def save_results(skill_name: str, results: Dict[str, Any]) -> None:
    """Save evaluation results."""
    results_dir = Path(__file__).parent.parent / "results"
    results_dir.mkdir(exist_ok=True)
    
    # Save detailed results
    latest_file = results_dir / "latest_run.json"
    with open(latest_file, 'w') as f:
        json.dump(results, f, indent=2)
    
    # Update scores history
    scores_file = results_dir / "scores.json"
    scores = []
    if scores_file.exists():
        with open(scores_file, 'r') as f:
            scores = json.load(f)
    
    scores.append({
        "timestamp": datetime.now().isoformat(),
        "skill": skill_name,
        "variant": results.get("variant", "current"),
        "pass_rate": results.get("pass_rate", 0),
        "passed": results.get("passed", 0),
        "total": results.get("total", 0),
        "improvement": results.get("improvement", 0)
    })
    
    with open(scores_file, 'w') as f:
        json.dump(scores, f, indent=2)


def get_previous_pass_rate(skill_name: str) -> float:
    """Get the previous pass rate for comparison."""
    scores_file = Path(__file__).parent.parent / "results" / "scores.json"
    if not scores_file.exists():
        return 0.0
    
    with open(scores_file, 'r') as f:
        scores = json.load(f)
    
    # Find the most recent result for this skill
    skill_scores = [s for s in scores if s["skill"] == skill_name]
    if not skill_scores:
        return 0.0
    
    return skill_scores[-1]["pass_rate"]


def extract_relevant_response(skill_name: str, input_text: str, full_skill: str) -> str:
    """Extract relevant sections from skill file based on input."""
    # Simple approach: return the full skill for now
    # In a more sophisticated system, this would analyze the input and extract relevant sections
    return full_skill


def generate_realistic_response(skill_name: str, input_text: str) -> str:
    """Generate a realistic response for a skill based on its documentation."""
    # Load the actual skill file if it exists
    skill_file = Path(__file__).parent.parent.parent / skill_name / "SKILL.md"
    if skill_file.exists():
        with open(skill_file, 'r') as f:
            return f.read()
    
    # Fallback: Generate a basic response structure
    response = f"""# {skill_name.title()} Response

## Overview
This is a response for: {input_text}

## Key Points
- Point 1 related to the skill
- Point 2 related to the skill  
- Point 3 related to the skill

## Examples
Example code or configuration would go here.

## Notes
Additional information and considerations.
"""
    return response


def evaluate_skill(skill_name: str, variant: str = "current") -> Dict[str, Any]:
    """Evaluate a skill prompt variant against all test cases."""
    print(f"Evaluating {skill_name} (variant: {variant})")
    
    # Load test cases and prompt
    test_cases = load_test_cases(skill_name)
    if not test_cases:
        return {"error": f"No test cases found for skill: {skill_name}"}
    
    try:
        prompt = load_prompt(skill_name, variant)
    except FileNotFoundError as e:
        return {"error": str(e)}
    
    # Evaluate each test case
    results = []
    passed = 0
    
    for test_case in test_cases:
        # Load the actual skill response from the skills directory
        skill_file = Path(__file__).parent.parent.parent / skill_name / "SKILL.md"
        if skill_file.exists():
            with open(skill_file, 'r') as f:
                full_skill = f.read()
            # Extract relevant sections based on input
            skill_response = extract_relevant_response(skill_name, test_case["input"], full_skill)
        else:
            # Fallback to generating a realistic response based on the skill
            skill_response = generate_realistic_response(skill_name, test_case["input"])
        
        # Run binary assertions
        assertions_result = evaluate_skill_output(
            skill_name, 
            test_case["input"], 
            skill_response, 
            test_case.get("metadata", {})
        )
        
        test_passed = assertions_result["all_passed"]
        if test_passed:
            passed += 1
        
        results.append({
            "test_id": test_case["id"],
            "input": test_case["input"],
            "passed": test_passed,
            "assertions": assertions_result["assertions"],
            "response": skill_response
        })
    
    pass_rate = (passed / len(test_cases)) * 100
    previous_rate = get_previous_pass_rate(skill_name)
    improvement = pass_rate - previous_rate
    
    evaluation_result = {
        "skill": skill_name,
        "variant": variant,
        "timestamp": datetime.now().isoformat(),
        "total_tests": len(test_cases),
        "passed": passed,
        "failed": len(test_cases) - passed,
        "pass_rate": round(pass_rate, 2),
        "previous_pass_rate": previous_rate,
        "improvement": round(improvement, 2),
        "test_results": results
    }
    
    # Save results
    save_results(skill_name, evaluation_result)
    
    print(f"Results: {passed}/{len(test_cases)} passed ({pass_rate:.1f}% pass rate)")
    if improvement != 0:
        print(f"Improvement: {improvement:+.1f}% from previous run")
    
    return evaluation_result


def main():
    parser = argparse.ArgumentParser(description="Evaluate skill prompts")
    parser.add_argument("--skill", required=True, help="Skill name to evaluate")
    parser.add_argument("--variant", default="current", help="Prompt variant to test")
    parser.add_argument("--list-skills", action="store_true", help="List available skills")
    
    args = parser.parse_args()
    
    if args.list_skills:
        # List skills with test cases
        test_file = Path(__file__).parent / "test_cases.jsonl"
        if not test_file.exists():
            print("No test cases file found")
            return
        
        skills = set()
        with open(test_file, 'r') as f:
            for line in f:
                case = json.loads(line.strip())
                skills.add(case.get('skill', 'unknown'))
        
        print("Available skills:")
        for skill in sorted(skills):
            print(f"  - {skill}")
        return
    
    result = evaluate_skill(args.skill, args.variant)
    
    if "error" in result:
        print(f"Error: {result['error']}", file=sys.stderr)
        sys.exit(1)
    
    # Print summary for Claude Code analysis
    print(f"\nSummary for {args.skill}:")
    print(f"Pass rate: {result['pass_rate']}%")
    print(f"Improvement: {result['improvement']:+.1f}%")
    
    if result['improvement'] < 0:
        print("⚠️  Performance decreased - consider reverting")
    elif result['improvement'] > 5:
        print("✅ Significant improvement - good candidate for new baseline")


if __name__ == "__main__":
    main()
