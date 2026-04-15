#!/usr/bin/env python3
"""
Binary Evaluation Assertions for Skills

Defines yes/no assertions to evaluate skill output quality.
Each assertion returns True/False and provides reasoning.
"""

import re
import json
from typing import Dict, List, Any, Tuple


def has_required_sections(skill_name: str, response: str) -> Tuple[bool, str]:
    """Check if response includes required sections for the skill."""
    required_sections = {
        "ansible-fleet": ["## Overview", "## Tools Required", "## Usage"],
        "ollama-deploy": ["## Overview", "## Tools Required", "## Key Principles"],
        "raspberry-pi": ["## Overview", "## Critical Differences", "## Safe Update Workflow"],
        "elixir-gastown-ops": ["## Fleet Topology", "## Check Cluster", "## Common Fixes"],
        "irclaw-cluster-ops": ["## IRC / irclaw bot protocol", "## Long-running tasks", "## Common mistakes"],
        "datadog-backup": ["## Overview", "## Tools Required", "## Backup Skills to Notebooks"],
        "zellij": ["## Core rule", "## Quick reference", "## Common mistakes"],
        "ac-list": ["## Command", "## Status symbols", "## Filter examples"],
        "dispatch-nu": ["## Projects", "## Quick reference", "## Launching panes"],
        "autoresearch": ["## Subcommands", "## When to Activate", "## Critical Rules"]
    }
    
    if skill_name not in required_sections:
        return True, f"No specific sections required for {skill_name}"
    
    missing_sections = []
    for section in required_sections[skill_name]:
        if section not in response:
            missing_sections.append(section)
    
    if missing_sections:
        return False, f"Missing required sections: {', '.join(missing_sections)}"
    
    return True, "All required sections present"


def follows_markdown_format(response: str) -> Tuple[bool, str]:
    """Check if response follows proper markdown format."""
    # Check for proper heading hierarchy
    lines = response.split('\n')
    
    # Should have at least one H1 or H2
    has_heading = any(line.strip().startswith('#') for line in lines)
    if not has_heading:
        return False, "No markdown headings found"
    
    # Check for proper code block formatting
    code_blocks = re.findall(r'```', response)
    if len(code_blocks) % 2 != 0:
        return False, "Unclosed code blocks detected"
    
    # Check table formatting if present (but ignore bash commands with pipes)
    table_rows = [line for line in lines if '|' in line and line.strip()]
    for row in table_rows:
        # Skip if it looks like a bash command (contains common bash indicators)
        bash_indicators = ['curl', 'wget', 'ansible', 'ssh', 'pip', 'npm', '-m shell', '-a', 'rpc', 'git', '-v', '2>&1', 'tee', '>', '>>', '|', '\\\\']
        if any(indicator in row for indicator in bash_indicators):
            continue
        # Skip if inside a code block
        if row.strip().startswith('```'):
            continue
        # Skip if it has command syntax (backticks, quotes, common command patterns)
        if any(pattern in row for pattern in ['`', '"', "'", 'sudo', 'cd /', 'bin/', '/tmp/']):
            continue
        if not row.strip().startswith('|') and not row.strip().endswith('|'):
            return False, f"Malformed table row: {row.strip()}"
    
    return True, "Proper markdown formatting"


def avoids_pii_exposure(response: str) -> Tuple[bool, str]:
    """Check if response avoids PII exposure."""
    pii_patterns = [
        r'/Users/[^/\s]+',  # Unix home paths
        r'/home/[^/\s]+',   # Linux home paths
        r'C:\\Users\\[^\\]+', # Windows home paths
        r'\b\d{3}-\d{2}-\d{4}\b',  # SSN pattern
        r'\b\d{4}[- ]?\d{4}[- ]?\d{4}[- ]?\d{4}\b',  # Credit card pattern
    ]
    
    for pattern in pii_patterns:
        matches = re.findall(pattern, response, re.IGNORECASE)
        # Filter out environment variables (e.g., /home/$DEPLOY_USER)
        matches = [m for m in matches if '$' not in m and '{' not in m]
        if matches:
            return False, f"Potential PII detected: {matches[0]}"
    
    # Check for hardcoded usernames (but exclude common generic terms and environment variables)
    # Only flag if it looks like an actual username pattern
    if re.search(r'/\b(user\d+|admin\d+|test\d+)\b', response, re.IGNORECASE):
        return False, "Hardcoded username detected"
    
    return True, "No PII exposure detected"


def meets_length_constraints(response: str, min_length: int = 100, max_length: int = 20000) -> Tuple[bool, str]:
    """Check if response meets reasonable length constraints."""
    length = len(response.strip())
    
    if length < min_length:
        return False, f"Response too short: {length} chars (minimum {min_length})"
    
    if length > max_length:
        return False, f"Response too long: {length} chars (maximum {max_length})"
    
    return True, f"Appropriate length: {length} chars"


def addresses_user_intent(input_text: str, response: str) -> Tuple[bool, str]:
    """Check if response addresses the user's likely intent."""
    input_lower = input_text.lower()
    response_lower = response.lower()
    
    # Common intent keywords and their expected response indicators
    intent_mappings = {
        "how to": ["##", "###", "steps", "usage", "example"],
        "what is": ["##", "definition", "overview", "description"],
        "fix": ["fix", "solution", "troubleshoot", "common issues"],
        "deploy": ["deploy", "installation", "setup", "configuration"],
        "monitor": ["monitor", "check", "status", "metrics"],
        "troubleshoot": ["issue", "problem", "error", "fix", "solution"]
    }
    
    for intent, indicators in intent_mappings.items():
        if intent in input_lower:
            found_indicator = any(indicator in response_lower for indicator in indicators)
            if not found_indicator:
                return False, f"Response may not address '{intent}' intent"
    
    return True, "Response addresses user intent"


def has_examples_or_code_blocks(skill_name: str, response: str) -> Tuple[bool, str]:
    """Check if technical skills include examples or code blocks."""
    technical_skills = {
        "ansible-fleet", "ollama-deploy", "raspberry-pi", 
        "elixir-gastown-ops", "irclaw-cluster-ops",
        "zellij", "dispatch-nu", "autoresearch"
    }
    
    if skill_name not in technical_skills:
        return True, f"Examples not required for {skill_name}"
    
    # Check for code blocks
    if "```" not in response:
        return False, "Technical skill should include code examples"
    
    # Check for inline code
    if "`" not in response:
        return False, "Technical skill should include inline code examples"
    
    return True, "Technical examples present"


def skill_specific_assertions(skill_name: str, input_text: str, response: str, metadata: Dict[str, Any]) -> List[Tuple[bool, str]]:
    """Run skill-specific assertions."""
    assertions = []
    
    if skill_name == "ansible-fleet":
        # Should mention Ansible-specific concepts
        ansible_terms = ["ansible", "playbook", "inventory", "hosts"]
        if not any(term in response.lower() for term in ansible_terms):
            assertions.append((False, "Missing Ansible-specific terminology"))
        else:
            assertions.append((True, "Ansible terminology present"))
    
    elif skill_name == "ollama-deploy":
        # Should mention Ollama-specific concepts
        ollama_terms = ["ollama", "model", "llm", "pull"]
        if not any(term in response.lower() for term in ollama_terms):
            assertions.append((False, "Missing Ollama-specific terminology"))
        else:
            assertions.append((True, "Ollama terminology present"))
    
    elif skill_name == "raspberry-pi":
        # Should mention Pi-specific concepts
        pi_terms = ["raspberry", "pi", "arm64", "ubuntu", "raspios"]
        if not any(term in response.lower() for term in pi_terms):
            assertions.append((False, "Missing Raspberry Pi-specific terminology"))
        else:
            assertions.append((True, "Raspberry Pi terminology present"))
    
    elif skill_name == "zellij":
        # Should mention Zellij-specific concepts
        zellij_terms = ["zellij", "session", "pane", "layout"]
        if not any(term in response.lower() for term in zellij_terms):
            assertions.append((False, "Missing Zellij-specific terminology"))
        else:
            assertions.append((True, "Zellij terminology present"))
    
    return assertions


def evaluate_skill_output(skill_name: str, input_text: str, response: str, metadata: Dict[str, Any] = None) -> Dict[str, Any]:
    """Run all binary assertions for a skill output."""
    if metadata is None:
        metadata = {}
    
    # Core assertions that apply to all skills
    core_assertions = [
        ("Required sections", lambda: has_required_sections(skill_name, response)),
        ("Markdown format", lambda: follows_markdown_format(response)),
        ("No PII exposure", lambda: avoids_pii_exposure(response)),
        ("Length constraints", lambda: meets_length_constraints(response)),
        ("Addresses intent", lambda: addresses_user_intent(input_text, response)),
        ("Has examples", lambda: has_examples_or_code_blocks(skill_name, response))
    ]
    
    # Run core assertions
    results = []
    for name, assertion_func in core_assertions:
        try:
            passed, reason = assertion_func()
            results.append({
                "name": name,
                "passed": passed,
                "reason": reason
            })
        except Exception as e:
            results.append({
                "name": name,
                "passed": False,
                "reason": f"Assertion error: {str(e)}"
            })
    
    # Run skill-specific assertions
    try:
        specific_results = skill_specific_assertions(skill_name, input_text, response, metadata)
        for passed, reason in specific_results:
            results.append({
                "name": "Skill-specific",
                "passed": passed,
                "reason": reason
            })
    except Exception as e:
        results.append({
            "name": "Skill-specific",
            "passed": False,
            "reason": f"Skill-specific assertion error: {str(e)}"
        })
    
    # Calculate overall result
    all_passed = all(result["passed"] for result in results)
    failed_assertions = [r for r in results if not r["passed"]]
    
    return {
        "all_passed": all_passed,
        "passed_count": len([r for r in results if r["passed"]]),
        "total_count": len(results),
        "assertions": results,
        "failed_assertions": failed_assertions,
        "summary": "All assertions passed" if all_passed else f"Failed: {', '.join([f['name'] for f in failed_assertions])}"
    }
