# Task: Improve Coordination Skills

## Skills to Improve
- delegate
- zellij-team
- tiger-team
- council

## Improvement Focus
1. **Expand test coverage** with complex scenarios:
   - Add test cases for agent spawning failures
   - Add test cases for team coordination edge cases
   - Add test cases for adversarial review scenarios
   - Add test cases for consensus failure scenarios

2. **Deepen prompt files** with coordination patterns:
   - Add decision matrices for when to use each coordination method
   - Add failure handling strategies
   - Add escalation procedures
   - Add best practices for agent team management

3. **Add integration tests** between coordination skills:
   - Test delegate spawning zellij-team agents
   - Test council spawning tiger-team for sub-problems
   - Test coordination skill + fleet operations integration

## Output
- Update test_cases.jsonl with new test cases (3-5 per skill)
- Update prompt files in autoresearch-skills/prompts/current/ with coordination patterns
- Run evals/runner.py for each skill to verify improvements
- Document any coordination issues found

## Constraints
- Work in /Users/studio/skills-1 directory
- Use existing eval framework (evals/runner.py)
- Maintain 100% pass rate on existing tests while adding new ones
- Focus on real-world coordination scenarios
