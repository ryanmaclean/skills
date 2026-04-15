# Skills AutoResearch Priority List

## High Priority (Start Here)
These skills have high usage impact and clear success metrics:

1. **ansible-fleet** - Core infrastructure deployment
   - Current baseline: TBD
   - Target: 85% pass rate
   - Test cases: 3
   - Focus areas: Architecture handling, troubleshooting, best practices

2. **ollama-deploy** - LLM deployment across fleet
   - Current baseline: TBD
   - Target: 80% pass rate
   - Test cases: 3
   - Focus areas: ARM64 support, model management, service troubleshooting

3. **raspberry-pi** - Pi fleet management
   - Current baseline: TBD
   - Target: 85% pass rate
   - Test cases: 3
   - Focus areas: OS upgrades, Python compatibility, architecture differences

## Medium Priority
Important skills with moderate complexity:

4. **elixir-gastown-ops** - BEAM cluster operations
   - Target: 75% pass rate
   - Test cases: 3
   - Focus areas: Cluster health, deployment, connectivity issues

5. **irclaw-cluster-ops** - IRC bot fleet management
   - Target: 75% pass rate
   - Test cases: 3
   - Focus areas: Bot communication, security policy, long-running tasks

6. **zellij** - Terminal multiplexer workflows
   - Target: 80% pass rate
   - Test cases: 3
   - Focus areas: Session management, scripting, troubleshooting

## Lower Priority
Specialized skills and utilities:

7. **datadog-backup** - Skills backup/restore
   - Target: 70% pass rate
   - Test cases: 2
   - Focus areas: Backup process, restore workflow

8. **ac-list** - Auto-Claude status checking
   - Target: 75% pass rate
   - Test cases: 2
   - Focus areas: Status queries, filtering

9. **dispatch-nu** - Spec dispatching
   - Target: 70% pass rate
   - Test cases: 2
   - Focus areas: Wave execution, setup

10. **autoresearch** - Self-improvement system
    - Target: 80% pass rate
    - Test cases: 3
    - Focus areas: Loop setup, metrics, security auditing

## Improvement Strategy

### Phase 1: Baseline Establishment
- Run initial evaluation on all high-priority skills
- Document current pass rates
- Identify common failure patterns

### Phase 2: Iterative Improvement
- Focus on one skill at a time
- Run 5-10 iterations per skill
- Target 10-15% improvement per skill

### Phase 3: Refinement
- Re-evaluate improved skills
- Address remaining edge cases
- Optimize for consistency

## Success Metrics

### Individual Skill Targets
- High priority: 80-85% pass rate
- Medium priority: 70-80% pass rate  
- Lower priority: 65-75% pass rate

### System-wide Targets
- Average pass rate across all skills: 75%+
- Zero skills below 60% pass rate
- Consistent improvement trajectory

### Quality Gates
- All skills must pass PII exposure checks
- All skills must follow markdown format
- All skills must include examples for technical content

## AutoResearch Loop Configuration

### Standard Loop Parameters
- Variants per iteration: 3
- Maximum iterations per skill: 20
- Stop conditions: 90% pass rate OR 5 iterations without improvement
- Evaluation timeout: 30 seconds per test case

### Claude Code Instructions
- Focus analysis on failure patterns
- Generate targeted variants based on specific assertion failures
- Maintain skill-specific terminology and context
- Preserve successful patterns between variants

## Monitoring and Maintenance

### Daily Checks
- Review pass rate trends
- Identify skills needing attention
- Check for assertion drift

### Weekly Reviews
- Evaluate overall system performance
- Update test cases for new edge cases
- Refine assertion logic

### Monthly Maintenance
- Add new skills to the pipeline
- Retire obsolete skills
- Update priority rankings based on usage patterns
