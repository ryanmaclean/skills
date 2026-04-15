#!/usr/bin/env python3
"""
Test dependency resolution with missing dependencies and fallbacks.
"""

import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).parent))

from dependency_resolver import DependencyResolver

def test_missing_dependencies():
    """Test behavior when dependencies are missing."""
    skills_dir = Path(__file__).parent.parent
    resolver = DependencyResolver(skills_dir)
    
    # Configuration with missing dependencies
    config = {
        "name": "test-missing",
        "dependencies": [
            {
                "skill": "nonexistent-skill",
                "required": True,
                "on_missing": "error",
                "context_transfer": ["some_data"]
            },
            {
                "skill": "another-missing",
                "required": False,
                "alternative": "preflight",
                "on_missing": "warn",
                "context_transfer": ["quota_status"]
            },
            {
                "skill": "preflight",
                "required": True,
                "context_transfer": ["recommended_provider"]
            }
        ],
        "execution": {
            "mode": "sequential",
            "fail_fast": True
        }
    }
    
    # Set intent
    resolver.context_manager.set_intent({
        "intent": "test missing dependencies",
        "scope": "testing",
        "priority": "medium"
    })
    
    # Simulate execution
    execution_plan = resolver.simulate_execution(config)
    
    print("=== Missing Dependencies Test ===")
    print(f"Can execute: {execution_plan['can_execute']}")
    print(f"Dependencies: {len(execution_plan['dependencies'])}")
    
    print("\n=== Dependency Results ===")
    for dep in execution_plan['dependencies']:
        status_symbol = "✓" if dep['status'] == 'available' else "⚠" if dep['status'] == 'alternative_found' else "✗"
        print(f"{status_symbol} {dep['requested_skill']}", end="")
        if dep['actual_skill'] != dep['requested_skill']:
            print(f" → {dep['actual_skill']}", end="")
        print(f" ({dep['status']}, required: {dep['required']})")
    
    print(f"\nWarnings: {len(execution_plan['warnings'])}")
    for warning in execution_plan['warnings']:
        print(f"  ⚠ {warning}")
    
    print(f"\nErrors: {len(execution_plan['errors'])}")
    for error in execution_plan['errors']:
        print(f"  ✗ {error}")
    
    return execution_plan['can_execute']

def test_conditional_execution():
    """Test conditional execution based on context."""
    skills_dir = Path(__file__).parent.parent
    resolver = DependencyResolver(skills_dir)
    
    # Configuration with conditions
    config = {
        "name": "test-conditional",
        "dependencies": [
            {
                "skill": "preflight",
                "required": False,
                "context_transfer": ["quota_status", "recommended_provider"]
            },
            {
                "skill": "dispatch-nu",
                "required": True,
                "condition": "quota_status != 'blocked'",  # Should pass
                "context_transfer": ["dispatch_result"]
            },
            {
                "skill": "ansible-fleet",
                "required": False,
                "condition": "recommended_provider == 'codex'",  # Should pass based on simulated context
                "context_transfer": ["deployment_ready"]
            }
        ]
    }
    
    resolver.context_manager.set_intent({
        "intent": "test conditional execution",
        "scope": "testing"
    })
    
    execution_plan = resolver.simulate_execution(config)
    
    print("\n=== Conditional Execution Test ===")
    print(f"Can execute: {execution_plan['can_execute']}")
    
    print("\n=== Context Flow ===")
    for flow in execution_plan['context_flow']:
        print(f"📋 {flow['skill']}: {flow['context_keys']}")
    
    print("\n=== Context Levels ===")
    for level in range(4):
        context = resolver.context_manager.get_context_for_level(level)
        if context:
            print(f"\nLevel {level}:")
            for key, value in context.items():
                print(f"  {key}: {value}")

if __name__ == "__main__":
    can_execute_1 = test_missing_dependencies()
    test_conditional_execution()
    
    print(f"\n=== Summary ===")
    print(f"Missing deps test passed: {can_execute_1}")
    print("Conditional execution test: Completed")
