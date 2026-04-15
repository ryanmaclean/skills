# AutoResearch Skills Self-Improvement System

A complete implementation of Andrej Karpathy's AutoResearch pattern applied to Claude Code skills, enabling autonomous improvement through binary evaluation assertions and iterative prompt refinement.

## Overview

This system allows skills to automatically improve themselves overnight by:
1. **Evaluating** current performance against binary assertions
2. **Analyzing** failure patterns to identify improvement areas  
3. **Generating** targeted prompt variants
4. **Selecting** the best performing variant as the new baseline
5. **Repeating** until target pass rates are achieved

## Quick Start

```bash
# Test the system
python3 test_system.py

# Evaluate a skill
python3 evals/runner.py --skill ansible-fleet

# Generate improvement variants
python3 evals/generate_variants.py --skill ansible-fleet --count 3

# Evaluate each variant
python3 evals/runner.py --skill ansible-fleet --variant structure_143000
python3 evals/runner.py --skill ansible-fleet --variant alignment_143000
python3 evals/runner.py --skill ansible-fleet --variant comprehensive_143000

# Update baseline with winner
python3 evals/update_baseline.py --skill ansible-fleet --variant structure_143000
```

## Project Structure

```
/autoresearch-skills/
├── AGENT_INSTRUCTIONS.md    # Instructions for Claude Code
├── skills_to_improve.md     # Priority list and targets
├── README.md                # This file
├── test_system.py           # System validation
│
├── prompts/                 # Prompt management
│   ├── current/             # Active skill prompts
│   ├── candidates/          # Variants under evaluation
│   └── history/             # Past versions with scores
│
├── evals/                   # Evaluation framework
│   ├── runner.py            # Main evaluation script
│   ├── assertions.py        # Binary eval functions
│   ├── generate_variants.py # Variant generation
│   ├── update_baseline.py   # Baseline management
│   └── test_cases.jsonl     # Test inputs
│
└── results/                 # Results tracking
    ├── scores.json          # Pass rate history
    └── latest_run.json      # Detailed results
```

## Binary Evaluation Assertions

Each skill is evaluated against 7 binary assertions:

### Core Assertions (All Skills)
1. **Required Sections** - Has proper skill-specific structure
2. **Markdown Format** - Follows proper markdown formatting
3. **No PII Exposure** - Avoids hardcoded paths/usernames
4. **Length Constraints** - Appropriate content length (100-5000 chars)
5. **Addresses Intent** - Responds to user's specific question
6. **Has Examples** - Includes code examples for technical skills

### Skill-Specific Assertions
- **ansible-fleet**: Ansible terminology present
- **ollama-deploy**: Ollama terminology present  
- **raspberry-pi**: Pi-specific terminology present
- **zellij**: Zellij terminology present

## Supported Skills

### High Priority (27 test cases)
- **ansible-fleet** - Infrastructure deployment (3 tests)
- **ollama-deploy** - LLM deployment (3 tests)
- **raspberry-pi** - Pi fleet management (3 tests)

### Medium Priority  
- **elixir-gastown-ops** - BEAM cluster operations (3 tests)
- **irclaw-cluster-ops** - IRC bot fleet (3 tests)
- **zellij** - Terminal multiplexer (3 tests)

### Lower Priority
- **datadog-backup** - Skills backup/restore (2 tests)
- **ac-list** - Auto-Claude status (2 tests)
- **dispatch-nu** - Spec dispatching (2 tests)
- **autoresearch** - Self-improvement system (3 tests)

## AutoResearch Loop

### Standard Configuration
- **Variants per iteration**: 3
- **Maximum iterations**: 20 per skill
- **Stop conditions**: 90% pass rate OR 5 iterations without improvement
- **Evaluation timeout**: 30 seconds per test case

### Claude Code Integration
The system is designed to work with Claude Code's `/loop` command:

```bash
# Run 10 improvement cycles overnight
/loop 10 /autoresearch
```

Claude Code will:
1. Read `AGENT_INSTRUCTIONS.md` for guidance
2. Analyze latest results from `results/latest_run.json`
3. Generate targeted variants based on failure patterns
4. Evaluate each variant using `runner.py`
5. Select and promote the winner using `update_baseline.py`
6. Repeat until loop completes

## Success Metrics

### Individual Skill Targets
- **High priority**: 80-85% pass rate
- **Medium priority**: 70-80% pass rate
- **Lower priority**: 65-75% pass rate

### System-wide Targets
- **Average pass rate**: 75%+ across all skills
- **Minimum pass rate**: 60% for any skill
- **PII compliance**: 100% (no failures allowed)

## Usage Examples

### Evaluate Current Performance
```bash
# Check all skills
python3 evals/runner.py --list-skills

# Evaluate specific skill
python3 evals/runner.py --skill ansible-fleet

# Check detailed results
cat results/latest_run.json | jq '.pass_rate, .improvement'
```

### Run Improvement Cycle
```bash
# Analyze failures and generate variants
python3 evals/generate_variants.py --skill ansible-fleet --analysis

# Evaluate each variant
for variant in structure_143000 alignment_143000 comprehensive_143000; do
    python3 evals/runner.py --skill ansible-fleet --variant $variant
done

# Update with best performer
python3 evals/update_baseline.py --skill ansible-fleet --variant structure_143000
```

### Track Progress
```bash
# View pass rate history
cat results/scores.json | jq '.[] | {skill, pass_rate, timestamp}'

# Plot improvement trends (requires jq and terminal plotting)
cat results/scores.json | jq -r '.[] | "\(.skill): \(.pass_rate)%"' | sort
```

## Adding New Skills

### 1. Create Test Cases
Add to `evals/test_cases.jsonl`:
```json
{"id": "tc_028", "skill": "new-skill", "input": "Your test question", "metadata": {"category": "test"}}
```

### 2. Define Required Sections
Update `assertions.py` in `has_required_sections()`:
```python
required_sections["new-skill"] = ["## Overview", "## Usage", "## Examples"]
```

### 3. Add Skill-Specific Assertions
Update `skill_specific_assertions()` in `assertions.py`:
```python
elif skill_name == "new-skill":
    # Your custom assertions here
```

### 4. Create Initial Prompt
Create `prompts/current/new-skill.md` with baseline prompt.

### 5. Update Priority List
Add to `skills_to_improve.md` with targets and test case count.

## Troubleshooting

### Low Pass Rates
- Check `results/latest_run.json` for failed assertions
- Run `python3 evals/generate_variants.py --analysis` to see improvement areas
- Ensure test cases cover diverse scenarios

### Variant Generation Issues
- Verify current prompt exists in `prompts/current/`
- Check previous results exist in `results/latest_run.json`
- Review failure analysis for targeted improvements

### Baseline Update Problems
- Ensure variant was evaluated successfully
- Check permissions on `prompts/current/` directory
- Verify variant ID matches generated file

## Integration with Claude Code

### Nightly Improvement Run
```bash
# Set up in Claude Code project
/loop 15 /autoresearch
```

### Manual Improvement Session
```bash
# Quick 5-iteration improvement
/loop 5 /autoresearch
```

### Targeted Skill Focus
```bash
# Focus on specific skill
/autoresearch
Skill: ansible-fleet
Target: 85% pass rate
```

## Results and Monitoring

### Daily Check
```bash
# Check overnight improvements
python3 evals/runner.py --list-skills | while read skill; do
    echo "=== $skill ==="
    python3 evals/runner.py --skill $skill | grep "pass rate"
done
```

### Weekly Review
```bash
# Generate improvement report
cat results/scores.json | jq -r '
  group_by(.skill) | 
  map({
    skill: .[0].skill,
    initial: .[0].pass_rate,
    latest: .[-1].pass_rate,
    improvement: (.[-1].pass_rate - .[0].pass_rate),
    iterations: length
  }) | 
  sort_by(.improvement) | 
  reverse[]
'
```

## Contributing

When adding new skills or assertions:
1. Maintain binary (True/False) evaluation
2. Ensure deterministic, repeatable results
3. Add comprehensive test cases
4. Document skill-specific requirements
5. Update this README with examples

## License

This AutoResearch implementation follows the same license as the skills repository.
