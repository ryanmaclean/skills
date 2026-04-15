# AutoResearch Instructions for Skills Self-Improvement

## Your Mission
You are an autonomous agent improving AI skills in this repository using the AutoResearch loop. Your goal is to systematically increase pass rates on binary eval assertions through iterative prompt refinement.

## The AutoResearch Loop
1. **Analyze** current results and failure patterns
2. **Generate** 3 improved prompt variants based on failures
3. **Evaluate** each variant against all test cases
4. **Select** the highest-scoring variant as the new baseline
5. **Repeat** until interrupted or pass rate target is reached

## Project Structure
```
/autoresearch-skills/
├── prompts/
│   ├── current/           # Active skill prompts
│   ├── candidates/        # Variants under evaluation
│   └── history/           # Past versions with scores in filenames
├── evals/
│   ├── test_cases.jsonl   # Test inputs for each skill
│   ├── assertions.py      # Binary eval functions
│   └── runner.py          # Evaluation harness
├── results/
│   ├── scores.json        # Pass rate history
│   └── latest_run.json    # Detailed results for analysis
└── skills_to_improve.md   # Priority list of skills to work on
```

## Critical Rules
- **Mechanical verification only** - Use pass rates, not subjective judgment
- **One change per cycle** - Generate focused variants, not random variations
- **Read before writing** - Analyze failure patterns before generating variants
- **Automatic rollback** - If a variant performs worse, revert immediately
- **Track everything** - Log all iterations with pass rates and reasoning

## Binary Eval Assertions
Each skill has specific yes/no assertions:
- Response includes required sections? (True/False)
- Follows specified format? (True/False)
- Avoids prohibited content? (True/False)
- Meets length constraints? (True/False)
- Addresses user intent? (True/False)

## Success Metrics
- **Primary**: Overall pass rate percentage
- **Secondary**: Improvement per iteration
- **Target**: 75-85% pass rate for most skills
- **Stop condition**: Plateau for 5 consecutive iterations OR 90% pass rate

## Commands to Use
```bash
# Run evaluation cycle
python evals/runner.py --skill <skill_name>

# Generate new variants (after analysis)
python evals/generate_variants.py --skill <skill_name> --count 3

# Update baseline with winner
python evals/update_baseline.py --skill <skill_name> --variant <variant_id>
```

## When to Stop
- Pass rate reaches 90% OR
- No improvement for 5 consecutive iterations OR
- Manual interruption

## Focus Skills Priority
1. High-usage skills (ansible-fleet, ollama-deploy, raspberry-pi)
2. Complex skills with multiple outputs (elixir-gastown-ops, irclaw-cluster-ops)
3. Skills with low current pass rates
4. New skills without established baselines

Remember: Every iteration must improve the metric. If something doesn't work, analyze why and try a different approach. Never make random changes - always base variants on failure pattern analysis.
