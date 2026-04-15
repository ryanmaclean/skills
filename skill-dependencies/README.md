# Skill Dependencies - Conditional Execution with Progressive Context Disclosure

## Overview

This framework enables skills to depend on each other conditionally, with intelligent fallbacks and progressive context disclosure. Skills can:

- **Check for dependencies** - Only proceed if required skills exist
- **Use alternatives** - Gracefully fallback to similar skills when primaries are missing  
- **Share context progressively** - Disclose relevant information as dependencies are satisfied
- **Execute conditionally** - Run based on context conditions (e.g., quota status, security findings)
- **Fail gracefully** - Provide meaningful errors and suggestions when dependencies can't be resolved

## Quick Examples

### Example 1: Smart Dispatch with Fallbacks
```yaml
name: smart-dispatch
dependencies:
  - skill: preflight
    required: false
    alternative: quota-gate
    on_missing: "warn"
    context_transfer: ["quota_status", "recommended_provider"]
    
  - skill: dispatch-nu
    required: true
    condition: "quota_status != 'blocked'"
```

**Behavior:**
- Tries `preflight` first, falls back to `quota-gate` if missing
- Only runs `dispatch-nu` if quota isn't blocked
- Shares quota status and provider recommendation with dispatcher

### Example 2: Security-First Deployment
```yaml
name: secure-deployment
dependencies:
  - skill: autoresearch:security
    required: true
    on_missing: "error"
    context_transfer: ["security_findings", "risk_level"]
    
  - skill: ansible-fleet
    required: true
    condition: "security_findings.critical == 0"
```

**Behavior:**
- Blocks deployment if security audit is missing
- Only deploys if no critical security issues found
- Shares security findings with deployment process

## Core Concepts

### Progressive Context Disclosure

Context is revealed in levels as dependencies are satisfied:

**Level 0: Intent** - User's goal and requirements
```json
{
  "intent": "deploy software",
  "scope": "fleet deployment", 
  "priority": "critical"
}
```

**Level 1: Capabilities** - What skills are available
```json
{
  "available_skills": ["preflight", "ansible-fleet", "security"],
  "missing_skills": [],
  "alternatives": {"preflight": "quota-gate"}
}
```

**Level 2: Results** - Outputs from executed dependencies
```json
{
  "preflight": {"quota_status": "warn", "recommended_provider": "codex"},
  "security": {"security_findings": {"critical": 0, "high": 2}}
}
```

**Level 3: Execution** - Final execution plan
```json
{
  "chosen_provider": "codex",
  "routing_strategy": "direct",
  "deployment_plan": "ansible-playbook fleet.yml"
}
```

### Dependency Resolution Patterns

#### 1. **Fallback Pattern**
```yaml
- skill: primary-skill
  required: false
  alternative: fallback-skill
  on_missing: "warn"
```
*Try primary → use alternative if missing → continue*

#### 2. **Gate Pattern**
```yaml
- skill: gatekeeper
  required: true  
  condition: "risk_level != 'high'"
  on_missing: "error"
```
*Must pass gate → block if condition fails or skill missing*

#### 3. **Chain Pattern**
```yaml
- skill: step1
  context_transfer: ["output1"]
- skill: step2  
  context_from: {step1: ["output1"]}
  context_transfer: ["output2"]
```
*Sequential execution with context passing*

#### 4. **Parallel Pattern**
```yaml
execution:
  mode: "parallel"
dependencies:
  - skill: checker1
  - skill: checker2
```
*Multiple dependencies run simultaneously → merge results*

## Usage

### Basic Usage
```python
from dependency_resolver import DependencyResolver

# Initialize with skills directory
resolver = DependencyResolver(Path("/path/to/skills"))

# Load configuration
config = resolver.load_dependency_config("smart-dispatch.yaml")

# Set user intent
resolver.context_manager.set_intent({
    "action": "dispatch tasks",
    "priority": "high"
})

# Resolve and execute
execution_plan = resolver.simulate_execution(config)
```

### Configuration Format
```yaml
name: my-workflow
description: "What this workflow does"

# User intent (Level 0 context)
intent:
  action: "deploy software"
  scope: "production"
  priority: "high"

# Dependencies with conditional execution
dependencies:
  - skill: required-skill
    required: true
    on_missing: "error"  # "warn", "error", "skip"
    context_transfer: ["result1", "result2"]
    
  - skill: optional-skill
    required: false
    alternative: fallback-skill
    condition: "result1 != 'blocked'"
    context_from:
      required-skill: ["result1"]

# Execution behavior
execution:
  mode: "sequential"  # "sequential", "parallel", "gate"
  context_disclosure: "progressive"  # "progressive", "full"
  fail_fast: false
```

## Real-World Examples

### Smart Task Dispatch
```bash
# The system automatically:
# 1. Checks quota with preflight (or quota-gate fallback)
# 2. Routes to cheapest available provider
# 3. Only dispatches if quota allows
# 4. Shares quota context with dispatcher

/smart-dispatch 5 tasks
```

### Secure Deployment Pipeline  
```bash
# The system automatically:
# 1. Runs security audit (blocks if missing)
# 2. Only deploys if no critical issues
# 3. Shares security findings with ansible
# 4. Provides deployment recommendations

/secure-deploy target=pis
```

### Multi-Provider Coordination
```yaml
dependencies:
  - skill: preflight
    context_transfer: ["recommended_provider", "cost_estimate"]
  - skill: dispatch-nu
    condition: "cost_estimate < 0.01"
  - skill: ollama-deploy
    condition: "recommended_provider == 'ollama'"
```

## Error Handling

### Graceful Degradation
- **Missing optional skills**: Warn and continue with alternatives
- **Missing required skills**: Error with helpful suggestions
- **Condition failures**: Skip dependency or block based on configuration

### Context Preservation
- Maintains context across dependency failures
- Enables retry with modified dependency chains
- Preserves user intent throughout the process

## Testing

### Test Missing Dependencies
```bash
cd /Users/studio/skills-1/skill-dependencies
python3 test_missing_deps.py
```

### Test Conditional Execution
```bash
python3 dependency_resolver.py
```

## Integration with Skills

### Making Your Skill Dependency-Aware
Add this to your skill's SKILL.md frontmatter:
```yaml
---
name: your-skill
dependencies:
  - skill: optional-helper
    required: false
    alternative: fallback-helper
context_consumes: ["quota_status", "security_findings"]
context_produces: ["execution_result", "cost_estimate"]
---
```

### Dependency Injection in Skills
Skills can check for available dependencies:
```python
# Check if a dependency skill is available
if skill_registry.skill_exists("preflight"):
    quota_context = get_context_from("preflight", ["quota_status"])
    if quota_context.get("quota_status") == "blocked":
        return "Cannot proceed - quota blocked"
```

## Best Practices

1. **Make dependencies optional when possible** - Provide alternatives
2. **Share minimal necessary context** - Don't over-share sensitive data
3. **Handle missing skills gracefully** - Give users helpful error messages
4. **Use conditions intelligently** - Base decisions on actual context
5. **Document your dependencies** - Clear requirements and alternatives
6. **Test with missing dependencies** - Ensure graceful degradation

## Advanced Features

### Dynamic Dependency Resolution
The framework can discover skills at runtime and adjust dependency chains based on what's actually available.

### Context Transformation
Skills can transform context they receive before passing it to the next dependency.

### Conditional Chains
Create complex dependency graphs with multiple paths based on runtime conditions.

### Monitoring and Observability
Built-in logging of dependency resolution, context flow, and execution decisions.

## Files Structure

```
/skill-dependencies/
├── dependency_resolver.py    # Core resolution engine
├── context_manager.py        # Progressive context disclosure
├── skill_registry.py         # Skills discovery
├── examples/                 # Example configurations
│   ├── smart-dispatch.yaml
│   └── secure-deployment.yaml
├── test_missing_deps.py      # Test suite
├── DEPENDENCY_FRAMEWORK.md   # Detailed documentation
└── README.md                 # This file
```

## Contributing

When adding new dependency patterns:
1. Update the dependency resolver with new logic
2. Add test cases to `test_missing_deps.py`
3. Document the pattern in `DEPENDENCY_FRAMEWORK.md`
4. Create example configurations in `examples/`

The framework is designed to be extensible - new patterns and behaviors can be added while maintaining backward compatibility.
