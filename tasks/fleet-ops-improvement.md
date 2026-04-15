# Task: Improve Fleet Operations Skills

## Skills to Improve
- ansible-fleet
- ollama-deploy  
- raspberry-pi
- elixir-gastown-ops
- irclaw-cluster-ops

## Improvement Focus
1. **Expand test coverage** with edge cases and error scenarios:
   - Add test cases for playbook failures, network timeouts, partial deployments
   - Add test cases for service failures, model download failures
   - Add test cases for OS upgrade failures, Python upgrade issues
   - Add test cases for cluster disconnection, deployment failures
   - Add test cases for bot security policy violations, long-running task failures

2. **Deepen prompt files** with domain expertise:
   - Add real-world patterns and best practices for fleet deployments
   - Add common pitfalls and anti-patterns
   - Add production troubleshooting guides
   - Add performance optimization patterns

3. **Add integration tests** between related skills:
   - Test ansible-fleet + ollama-deploy coordination
   - Test elixir-gastown-ops + irclaw-cluster-ops integration
   - Test raspberry-pi + ansible-fleet workflows

## Output
- Update test_cases.jsonl with new test cases (5-10 per skill)
- Update prompt files in autoresearch-skills/prompts/current/ with deeper expertise
- Run evals/runner.py for each skill to verify improvements
- Document any integration issues found

## Constraints
- Work in /Users/studio/skills-1 directory
- Use existing eval framework (evals/runner.py)
- Maintain 100% pass rate on existing tests while adding new ones
- Focus on production-critical scenarios
