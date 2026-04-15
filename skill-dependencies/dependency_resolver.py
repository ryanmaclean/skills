#!/usr/bin/env python3
"""
Skill Dependency Resolver

Handles conditional skill dependencies with progressive context disclosure.
Supports fallback patterns, gating, and intelligent dependency resolution.
"""

import json
import yaml
from pathlib import Path
from typing import Dict, List, Any, Optional, Tuple
from dataclasses import dataclass
from enum import Enum


class DependencyStatus(Enum):
    AVAILABLE = "available"
    MISSING = "missing" 
    ALTERNATIVE_FOUND = "alternative_found"
    FAILED = "failed"


@dataclass
class Dependency:
    skill: str
    required: bool = True
    alternative: Optional[str] = None
    on_missing: str = "warn"  # "warn", "error", "skip"
    context_transfer: List[str] = None
    condition: Optional[str] = None


@dataclass
class DependencyResult:
    dependency: Dependency
    status: DependencyStatus
    actual_skill: Optional[str] = None
    result: Optional[Dict[str, Any]] = None
    context: Optional[Dict[str, Any]] = None


class SkillRegistry:
    """Discovers available skills in the repository."""
    
    def __init__(self, skills_dir: Path):
        self.skills_dir = skills_dir
        self._cache = None
    
    def discover_skills(self) -> set:
        """Scan for available SKILL.md files."""
        if self._cache is not None:
            return self._cache
        
        skills = set()
        for skill_file in self.skills_dir.rglob("SKILL.md"):
            # Extract skill name from directory structure
            relative_path = skill_file.relative_to(self.skills_dir)
            if len(relative_path.parts) > 1:
                skill_name = relative_path.parts[0]
                skills.add(skill_name)
        
        self._cache = skills
        return skills
    
    def skill_exists(self, skill_name: str) -> bool:
        """Check if a skill exists."""
        return skill_name in self.discover_skills()
    
    def get_skill_path(self, skill_name: str) -> Optional[Path]:
        """Get the path to a skill's SKILL.md file."""
        if not self.skill_exists(skill_name):
            return None
        
        skill_file = self.skills_dir / skill_name / "SKILL.md"
        if skill_file.exists():
            return skill_file
        
        # Search subdirectories
        for skill_file in self.skills_dir.rglob("SKILL.md"):
            if skill_file.parent.name == skill_name:
                return skill_file
        
        return None


class ContextManager:
    """Manages progressive context disclosure."""
    
    def __init__(self):
        self.context = {
            "level_0": {},  # Intent
            "level_1": {},  # Capabilities  
            "level_2": {},  # Results
            "level_3": {}   # Execution
        }
    
    def set_intent(self, intent: Dict[str, Any]):
        """Set Level 0 context (user intent)."""
        self.context["level_0"] = intent
    
    def add_capabilities(self, capabilities: Dict[str, Any]):
        """Set Level 1 context (available capabilities)."""
        self.context["level_1"] = capabilities
    
    def add_results(self, skill_name: str, results: Dict[str, Any]):
        """Add Level 2 context (skill execution results)."""
        self.context["level_2"][skill_name] = results
    
    def add_execution(self, execution: Dict[str, Any]):
        """Set Level 3 context (execution plan)."""
        self.context["level_3"] = execution
    
    def get_context_for_level(self, level: int) -> Dict[str, Any]:
        """Get context up to specified level."""
        result = {}
        for i in range(min(level + 1, 4)):
            result.update(self.context[f"level_{i}"])
        return result
    
    def get_relevant_context(self, skill_name: str, context_keys: List[str]) -> Dict[str, Any]:
        """Extract context relevant to a specific skill."""
        relevant = {}
        
        for key in context_keys:
            # Search through all levels for the key
            for level in range(4):
                level_context = self.context[f"level_{level}"]
                if key in level_context:
                    relevant[key] = level_context[key]
                    break
        
        return relevant


class DependencyResolver:
    """Main dependency resolution engine."""
    
    def __init__(self, skills_dir: Path):
        self.registry = SkillRegistry(skills_dir)
        self.context_manager = ContextManager()
        self.execution_log = []
    
    def load_dependency_config(self, config_path: Path) -> Dict[str, Any]:
        """Load dependency configuration from YAML file."""
        with open(config_path, 'r') as f:
            return yaml.safe_load(f)
    
    def parse_dependencies(self, config: Dict[str, Any]) -> List[Dependency]:
        """Parse dependencies from configuration."""
        dependencies = []
        
        for dep_config in config.get("dependencies", []):
            dependency = Dependency(
                skill=dep_config["skill"],
                required=dep_config.get("required", True),
                alternative=dep_config.get("alternative"),
                on_missing=dep_config.get("on_missing", "warn"),
                context_transfer=dep_config.get("context_transfer", []),
                condition=dep_config.get("condition")
            )
            dependencies.append(dependency)
        
        return dependencies
    
    def resolve_single_dependency(self, dependency: Dependency) -> DependencyResult:
        """Resolve a single dependency with fallback logic."""
        # Check if primary skill exists
        if self.registry.skill_exists(dependency.skill):
            return DependencyResult(
                dependency=dependency,
                status=DependencyStatus.AVAILABLE,
                actual_skill=dependency.skill
            )
        
        # Try alternative if specified
        if dependency.alternative and self.registry.skill_exists(dependency.alternative):
            return DependencyResult(
                dependency=dependency,
                status=DependencyStatus.ALTERNATIVE_FOUND,
                actual_skill=dependency.alternative
            )
        
        # Handle missing dependency
        status = DependencyStatus.MISSING
        if dependency.required and dependency.on_missing == "error":
            status = DependencyStatus.FAILED
        
        return DependencyResult(
            dependency=dependency,
            status=status,
            actual_skill=None
        )
    
    def evaluate_condition(self, condition: str, context: Dict[str, Any]) -> bool:
        """Evaluate a condition string against context."""
        # Simple condition evaluation - can be extended with proper expression parsing
        if not condition:
            return True
        
        # Handle basic conditions like "quota_status != 'blocked'"
        try:
            # This is a simplified evaluator - in production, use a proper expression parser
            if "!=" in condition:
                key, value = condition.split("!=", 1)
                key = key.strip()
                value = value.strip().strip("'\"")
                
                # Navigate context to find the key
                context_value = self._get_nested_value(context, key.strip())
                return str(context_value) != value
            
            elif "==" in condition:
                key, value = condition.split("==", 1)
                key = key.strip()
                value = value.strip().strip("'\"")
                
                context_value = self._get_nested_value(context, key.strip())
                return str(context_value) == value
            
        except Exception:
            # If evaluation fails, default to allowing execution
            return True
        
        return True
    
    def _get_nested_value(self, data: Dict[str, Any], key: str) -> Any:
        """Get nested value from dict using dot notation."""
        keys = key.split('.')
        current = data
        
        for k in keys:
            if isinstance(current, dict) and k in current:
                current = current[k]
            else:
                return None
        
        return current
    
    def resolve_dependencies(self, config: Dict[str, Any]) -> Tuple[List[DependencyResult], bool]:
        """Resolve all dependencies for a configuration."""
        dependencies = self.parse_dependencies(config)
        results = []
        can_proceed = True
        
        # Set Level 1 context (capabilities)
        available_skills = self.registry.discover_skills()
        alternatives = {}
        
        for dep in dependencies:
            if dep.alternative:
                alternatives[dep.skill] = dep.alternative
        
        self.context_manager.add_capabilities({
            "available_skills": list(available_skills),
            "missing_skills": [],
            "alternatives": alternatives
        })
        
        # Resolve each dependency
        for dependency in dependencies:
            result = self.resolve_single_dependency(dependency)
            results.append(result)
            
            # Check if we can proceed
            if result.status == DependencyStatus.FAILED:
                can_proceed = False
            elif result.status == DependencyStatus.MISSING and dependency.required:
                if dependency.on_missing == "error":
                    can_proceed = False
        
        return results, can_proceed
    
    def simulate_execution(self, config: Dict[str, Any]) -> Dict[str, Any]:
        """Simulate execution with context passing."""
        results, can_proceed = self.resolve_dependencies(config)
        
        execution_plan = {
            "can_execute": can_proceed,
            "dependencies": [],
            "context_flow": [],
            "warnings": [],
            "errors": []
        }
        
        # Build dependency chain
        for result in results:
            dep_info = {
                "requested_skill": result.dependency.skill,
                "actual_skill": result.actual_skill,
                "status": result.status.value,
                "required": result.dependency.required
            }
            execution_plan["dependencies"].append(dep_info)
            
            # Add warnings/errors
            if result.status == DependencyStatus.MISSING:
                if result.dependency.required:
                    execution_plan["errors"].append(
                        f"Required skill '{result.dependency.skill}' is missing"
                    )
                else:
                    execution_plan["warnings"].append(
                        f"Optional skill '{result.dependency.skill}' is missing"
                    )
            
            elif result.status == DependencyStatus.ALTERNATIVE_FOUND:
                execution_plan["warnings"].append(
                    f"Using alternative '{result.dependency.alternative}' instead of '{result.dependency.skill}'"
                )
        
        # Simulate context flow
        if can_proceed:
            for result in results:
                if result.actual_skill:
                    # Simulate skill execution and context generation
                    simulated_context = self._simulate_skill_context(
                        result.actual_skill, 
                        result.dependency.context_transfer
                    )
                    
                    self.context_manager.add_results(result.actual_skill, simulated_context)
                    
                    execution_plan["context_flow"].append({
                        "skill": result.actual_skill,
                        "context_keys": result.dependency.context_transfer,
                        "generated_context": list(simulated_context.keys())
                    })
        
        # Set Level 3 context (execution plan)
        self.context_manager.add_execution({
            "execution_mode": config.get("execution", {}).get("mode", "sequential"),
            "fail_fast": config.get("execution", {}).get("fail_fast", True),
            "dependencies_satisfied": can_proceed
        })
        
        return execution_plan
    
    def _simulate_skill_context(self, skill_name: str, context_keys: List[str]) -> Dict[str, Any]:
        """Simulate context generation for a skill."""
        simulated_context = {}
        
        # Simulate different skills generating different context
        if skill_name == "preflight":
            simulated_context = {
                "quota_status": "warn",
                "recommended_provider": "codex",
                "task_capacity": 5,
                "cost_estimate": 0.003
            }
        elif skill_name == "quota-gate":
            simulated_context = {
                "quota_status": "go", 
                "provider_headroom": {"codex": 0.79, "claude": 0.18},
                "recommendation": "proceed with codex"
            }
        elif skill_name == "autoresearch:security":
            simulated_context = {
                "security_findings": {"critical": 0, "high": 2, "medium": 5},
                "risk_level": "medium",
                "scan_duration": "3m 42s"
            }
        elif skill_name == "ansible-fleet":
            simulated_context = {
                "target_hosts": 12,
                "playbook_ready": True,
                "estimated_duration": "8m 15s"
            }
        
        # Filter to only requested context keys
        if context_keys:
            simulated_context = {
                key: simulated_context.get(key) 
                for key in context_keys 
                if key in simulated_context
            }
        
        return simulated_context
    
    def get_full_context(self, max_level: int = 3) -> Dict[str, Any]:
        """Get complete context up to specified level."""
        return self.context_manager.get_context_for_level(max_level)


def main():
    """Example usage of the dependency resolver."""
    skills_dir = Path(__file__).parent.parent
    resolver = DependencyResolver(skills_dir)
    
    # Example configuration
    config = {
        "name": "smart-dispatch",
        "dependencies": [
            {
                "skill": "preflight",
                "required": False,
                "alternative": "quota-gate",
                "on_missing": "warn",
                "context_transfer": ["quota_status", "recommended_provider"]
            },
            {
                "skill": "dispatch-nu", 
                "required": True,
                "condition": "quota_status != 'blocked'",
                "context_from": {
                    "preflight": ["recommended_provider"],
                    "quota-gate": ["provider_headroom"]
                }
            }
        ],
        "execution": {
            "mode": "sequential",
            "fail_fast": False
        }
    }
    
    # Set user intent
    resolver.context_manager.set_intent({
        "intent": "dispatch tasks efficiently",
        "scope": "multi-agent coordination",
        "priority": "high"
    })
    
    # Simulate execution
    execution_plan = resolver.simulate_execution(config)
    
    print("=== Dependency Resolution Results ===")
    print(f"Can execute: {execution_plan['can_execute']}")
    print(f"Dependencies: {len(execution_plan['dependencies'])}")
    print(f"Warnings: {len(execution_plan['warnings'])}")
    print(f"Errors: {len(execution_plan['errors'])}")
    
    print("\n=== Execution Plan ===")
    for dep in execution_plan['dependencies']:
        status_symbol = "✓" if dep['status'] == 'available' else "⚠" if dep['status'] == 'alternative_found' else "✗"
        print(f"{status_symbol} {dep['requested_skill']}", end="")
        if dep['actual_skill'] != dep['requested_skill']:
            print(f" → {dep['actual_skill']}", end="")
        print(f" ({dep['status']})")
    
    print("\n=== Context Flow ===")
    for flow in execution_plan['context_flow']:
        print(f"📋 {flow['skill']}: {flow['context_keys']}")
    
    print("\n=== Full Context ===")
    full_context = resolver.get_full_context()
    for level_name, context in full_context.items():
        if isinstance(context, dict) and context:
            print(f"\n{level_name.upper()}:")
            for key, value in context.items():
                print(f"  {key}: {value}")
        elif context:
            print(f"\n{level_name.upper()}: {context}")


if __name__ == "__main__":
    main()
