# Skill Dependency Framework

## Overview

Skills can depend on each other conditionally with progressing context disclosure. This enables sophisticated workflows where skills:

1. **Check for dependencies** - Only proceed if required skills exist
2. **Share context progressively** - Disclose relevant information as dependencies are satisfied
3. **Fail gracefully** - Provide alternatives when dependencies are missing
4. **Chain intelligently** - Create dependency graphs with conditional execution

## Core Concepts

### Conditional Dependencies
```yaml
dependencies:
  - skill: preflight
    required: false
    alternative: "quota-gate"
    context: "quota_status"
  - skill: quota-gate  
    required: true
    context: "provider_recommendation"
```

### Progressing Context Disclosure
- **Level 0**: Basic intent (user request)
- **Level 1**: Dependency check results (what's available)
- **Level 2**: Dependency outputs (quota status, recommendations)
- **Level 3**: Execution context (chosen provider, routing decisions)

### Dependency Resolution Patterns

#### 1. **Fallback Pattern**
Try primary skill → fallback to alternative → proceed with available

#### 2. **Gate Pattern** 
Must pass dependency check → block or warn if failed

#### 3. **Chain Pattern**
Sequential execution with context passing

#### 4. **Parallel Pattern**
Multiple dependencies run in parallel → merge results

## Implementation Structure

```
/skill-dependencies/
├── dependency_resolver.py    # Core dependency engine
├── context_manager.py        # Progressive context disclosure
├── skill_registry.py         # Available skills discovery
├── examples/                 # Example dependency chains
│   ├── dispatch_chain.yaml
│   ├── security_audit.yaml
│   └── deployment_pipeline.yaml
└── schemas/                  # Validation schemas
    ├── dependency.yaml
    └── context.yaml
```

## Usage Examples

### Example 1: Smart Dispatch Chain
```yaml
name: smart-dispatch
description: Intelligent task dispatch with quota awareness

dependencies:
  - skill: preflight
    required: false
    alternative: quota-gate
    on_missing: "warn"
    context_transfer: ["quota_status", "provider_recommendation"]
  
  - skill: dispatch-nu
    required: true
    condition: "quota_status != 'blocked'"
    context_from:
      preflight: ["recommended_provider", "task_count"]
      quota-gate: ["provider_headroom"]

execution:
  mode: "sequential"
  context_disclosure: "progressive"
  fail_fast: false
```

### Example 2: Security-First Deployment
```yaml
name: secure-deploy
description: Deployment with mandatory security checks

dependencies:
  - skill: autoresearch:security
    required: true
    on_missing: "error"
    context_transfer: ["security_findings", "risk_level"]
  
  - skill: ansible-fleet
    required: true
    condition: "security_findings.critical == 0"
    context_from:
      security: ["target_hosts", "security_recommendations"]

execution:
  mode: "gate"
  context_disclosure: "full"
  fail_fast: true
```

## Context Disclosure Levels

### Level 0: Intent Disclosure
```json
{
  "intent": "deploy software to fleet",
  "scope": "ansible-fleet",
  "priority": "high"
}
```

### Level 1: Capability Disclosure  
```json
{
  "available_skills": ["preflight", "quota-gate", "ansible-fleet"],
  "missing_skills": [],
  "alternatives": {"preflight": "quota-gate"}
}
```

### Level 2: Results Disclosure
```json
{
  "preflight_result": {
    "status": "warn",
    "recommended_provider": "codex",
    "quota_status": "yellow"
  }
}
```

### Level 3: Execution Disclosure
```json
{
  "execution_plan": {
    "provider": "codex",
    "routing": "direct",
    "estimated_cost": "0.003",
    "confidence": 0.87
  }
}
```

## Dependency Resolution Algorithm

1. **Discovery**: Scan for available skills
2. **Validation**: Check dependency requirements
3. **Resolution**: Find alternatives for missing dependencies
4. **Context Building**: Assemble progressive context
5. **Execution**: Run dependency chain with context passing
6. **Adaptation**: Adjust based on intermediate results

## Error Handling Strategies

### Graceful Degradation
- Missing non-critical skills → warn and continue
- Missing critical skills → error with suggestions
- Partial failures → retry with alternatives

### Context Preservation
- Maintain context across dependency failures
- Enable retry with modified dependencies
- Preserve user intent throughout chain

## Best Practices

1. **Make dependencies optional when possible**
2. **Provide meaningful alternatives**
3. **Share only necessary context at each level**
4. **Handle missing skills gracefully**
5. **Document dependency requirements clearly**
6. **Test with missing dependencies**

## Implementation Status

- ✅ Core dependency resolution logic
- ✅ Progressive context disclosure framework
- ✅ Example dependency chains
- ⏳ Integration with existing skills
- ⏳ Visual dependency graph generator
- ⏳ Runtime dependency injection
