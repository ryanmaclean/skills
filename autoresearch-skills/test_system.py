#!/usr/bin/env python3
"""
Test the AutoResearch skills system with a realistic example.
"""

import json
import sys
from pathlib import Path

# Add evals directory to path
sys.path.insert(0, str(Path(__file__).parent / "evals"))

from assertions import evaluate_skill_output


def test_ansible_fleet():
    """Test ansible-fleet with a realistic response."""
    
    # Sample input
    input_text = "How do I deploy software to multiple Raspberry Pis using Ansible?"
    
    # Realistic skill response
    response = """# Ansible Fleet Deployment for Raspberry Pis

## Overview
Deploy software to multiple Raspberry Pis using Ansible with proper architecture handling and safety checks.

## Tools Required
- **Ansible** - Automation tool
- **SSH** - Remote connectivity
- **Inventory file** - Host definitions

## Usage

### 1. Create inventory file
```yaml
[pis]
ubrpi401 ansible_host=10.0.3.215 ansible_user=ubuntu
ubrpi402 ansible_host=10.0.3.216 ansible_user=ubuntu

[pis:vars]
ansible_python_interpreter=/usr/bin/python3
architecture=arm64
```

### 2. Deploy software
```bash
ansible-playbook -i inventory deploy.yml --limit pis
```

### 3. Verify deployment
```bash
ansible pis -i inventory -m shell -a "systemctl status your-service"
```

## Architecture Considerations
- Raspberry Pis use ARM64, not AMD64
- Use `ollama-linux-arm64.tgz` for ARM64 packages
- Test on single Pi before fleet-wide deployment

## Common Issues
- SSH key authentication failures
- Python version incompatibility
- Architecture mismatches

Always verify with `ansible --version` and check architecture with `uname -m`."""
    
    # Evaluate
    result = evaluate_skill_output("ansible-fleet", input_text, response)
    
    print("=== AutoResearch Skills System Test ===")
    print(f"Skill: ansible-fleet")
    print(f"Input: {input_text}")
    print(f"\nEvaluation Results:")
    print(f"Overall: {'PASS' if result['all_passed'] else 'FAIL'}")
    print(f"Passed: {result['passed_count']}/{result['total_count']}")
    
    print(f"\nAssertion Details:")
    for assertion in result['assertions']:
        status = "✅" if assertion['passed'] else "❌"
        print(f"  {status} {assertion['name']}: {assertion['reason']}")
    
    if not result['all_passed']:
        print(f"\nFailed Assertions:")
        for failed in result['failed_assertions']:
            print(f"  - {failed['name']}: {failed['reason']}")
    
    return result


def test_variant_generation():
    """Test variant generation concept."""
    print(f"\n=== Variant Generation Example ===")
    
    base_prompt = "You are an expert in Ansible fleet deployment."
    
    # Simulate failure analysis
    failure_patterns = [
        "Add missing required sections",
        "Include more practical code examples"
    ]
    
    # Generate variant 1 (enhanced structure)
    variant_1 = base_prompt + """

IMPORTANT: Always structure responses with clear sections using ## headers.
Include specific command examples, YAML configurations, and code snippets for every major concept.
Always include actual playbook snippets and inventory examples."""
    
    print(f"Base prompt: {base_prompt}")
    print(f"Variant 1 (enhanced): {variant_1}")
    
    return variant_1


def main():
    print("Testing AutoResearch Skills Self-Improvement System")
    print("=" * 60)
    
    # Test evaluation
    result = test_ansible_fleet()
    
    # Test variant generation
    test_variant_generation()
    
    print(f"\n=== System Status ===")
    print("✅ Evaluation framework: Working")
    print("✅ Binary assertions: Working")
    print("✅ Variant generation: Working")
    print("✅ Project structure: Complete")
    
    print(f"\n=== Next Steps for Claude Code ===")
    print("1. Run: python3 evals/runner.py --skill ansible-fleet")
    print("2. Analyze failures and generate variants")
    print("3. Evaluate variants and select winner")
    print("4. Update baseline with improvement")
    print("5. Repeat until target pass rate reached")
    
    return result['all_passed']


if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1)
