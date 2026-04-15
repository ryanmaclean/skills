# Better Evaluation Frameworks Research

## Overview
Research into state-of-the-art AI evaluation frameworks and methodologies to improve the current autoresearch evaluation system.

## Key Findings

### 1. Anthropic's "Demystifying evals for AI agents"

**Core Concepts:**
- **Types of Graders:**
  - Code-based: Deterministic, fast, reliable
  - Model-based: LLM-as-judge, flexible, captures nuance
  - Human: Gold standard, expensive, used for calibration

- **Capability vs Regression Evals:**
  - Capability evals: "What can this agent do well?" - start low, hill-climb
  - Regression evals: "Does it still handle what it used to?" - near 100% pass rate

- **Roadmap for Great Evals:**
  1. Start early (20-50 tasks from real failures)
  2. Start with manual checks you already do
  3. Write unambiguous tasks with reference solutions
  4. Build balanced problem sets (test both positive and negative cases)
  5. Build robust eval harness with stable environment
  6. Design graders thoughtfully (grade outcomes, not paths)
  7. Make graders resistant to bypasses

**Key Insights:**
- Grade outcomes, not implementation paths (agents find creative solutions)
- Use partial credit for multi-component tasks
- Calibrate LLM-as-judge with human experts
- Watch for subtle failure modes (rigid grading, ambiguous specs)
- Make graders cheat-resistant

### 2. AWS's "Evaluating AI agents: Real-world lessons"

**Core Concepts:**
- **Holistic Multi-Dimensional Evaluation:**
  - Quality: Reasoning coherence, tool selection, task completion
  - Performance: Latency, throughput, resource utilization
  - Responsibility: Safety, toxicity, bias, hallucination detection
  - Cost: Model inference, tool invocation, data processing, human effort

- **Evaluation Framework Components:**
  - Automated AI agent evaluation workflow (4 steps)
  - AI agent evaluation library (3 layers)
  - Human-in-the-loop (HITL) as critical component
  - Continuous evaluation in production

**Key Insights:**
- Extend beyond accuracy to comprehensive assessment
- Use case-specific evaluation metrics
- HITL essential for high-stakes decisions
- Continuous production monitoring to detect degradation
- Track key metrics through operational dashboards

### 3. Modern AI Evaluation Tools (Galileo Survey)

**Top Platforms:**
- **Galileo:** ChainPoll methodology, multi-model consensus, near-human accuracy
- **MLflow 3.0:** LLM-as-a-judge evaluators, hallucination detection, production monitoring
- **Weights & Biases (Weave):** Developer-friendly, LLM-as-a-judge, custom metrics
- **Google Vertex AI:** Enterprise-scale, integrated serving, vendor lock-in concerns
- **Opik by Comet:** Open-source, agent reliability, automated prompt optimization
- **DeepEval (Confident AI):** GenAI-native, unit testing framework, no ground truth needed

**Common Features:**
- LLM-as-a-judge evaluation
- Hallucination detection
- Real-time production monitoring
- Automated evaluation metrics
- Custom evaluation capabilities
- Integration with popular frameworks

### 4. Ai2 Evaluation Frameworks

**Specialized Frameworks:**
- **DataDecide:** 1,050 models, 30k checkpoints, 10 benchmarks for pretraining data decisions
- **Ai2 Safety Toolkit:** WildTeaming (red-teaming), WildJailbreak (safety training), WildGuard (moderation)
- **Paloma:** Cross-domain benchmark across 585 different domains

## Comparison with Current Autoresearch System

### Current System Strengths:
- Binary assertions provide clear pass/fail criteria
- Covers multiple dimensions (markdown, PII, length, intent, examples)
- Successfully improved 10 skills to 100% pass rate
- Simple and interpretable

### Current System Limitations:
- Only uses code-based graders (no LLM-as-judge)
- No capability vs regression eval distinction
- No partial credit for multi-component tasks
- No human-in-the-loop calibration
- No production monitoring
- Limited to skill file content evaluation (not actual agent performance)
- No evaluation of reasoning, tool usage, or decision-making quality
- No continuous monitoring or drift detection

## Recommended Improvements

### High Priority:
1. **Add LLM-as-Judge Graders**
   - Implement model-based evaluation for nuanced assessment
   - Use for evaluating skill quality, completeness, and effectiveness
   - Calibrate with human expert review initially

2. **Distinguish Capability vs Regression Evals**
   - Create separate test suites for improvement tracking vs regression prevention
   - Use capability evals for skill improvement (start low, hill-climb)
   - Use regression evals to prevent backsliding (near 100% pass rate)

3. **Add Partial Credit Scoring**
   - Implement weighted scoring for multi-component skills
   - Reward partial completion of complex tasks
   - Better reflect real-world performance

### Medium Priority:
4. **Implement Human-in-the-Loop Calibration**
   - Periodic human review of evaluation results
   - Use human feedback to calibrate LLM-as-judge graders
   - Create golden test datasets from expert review

5. **Add Production Monitoring**
   - Track skill performance over time
   - Detect performance degradation or drift
   - Implement alerting for significant changes

6. **Expand Evaluation Dimensions**
   - Add reasoning quality assessment
   - Evaluate tool usage appropriateness
   - Measure decision-making quality
   - Assess safety and responsibility

### Lower Priority:
7. **Integrate with Modern Platforms**
   - Consider integrating with tools like DeepEval or Opik
   - Leverage existing evaluation frameworks where appropriate
   - Maintain flexibility for custom evaluation needs

## Implementation Roadmap

### Phase 1: Enhanced Grading (Week 1-2)
- Add LLM-as-judge capability to assertion system
- Implement partial credit scoring
- Create separate capability and regression test suites

### Phase 2: Human Calibration (Week 3-4)
- Implement human review workflow
- Calibrate LLM-as-judge with expert feedback
- Create golden test datasets

### Phase 3: Production Monitoring (Week 5-6)
- Add performance tracking over time
- Implement drift detection
- Create monitoring dashboards

### Phase 4: Advanced Features (Week 7-8)
- Expand evaluation dimensions
- Add reasoning and decision-making assessment
- Integrate with modern evaluation platforms

## Conclusion

The current autoresearch evaluation system is effective for basic skill assessment but lacks the sophistication of state-of-the-art evaluation frameworks. Key improvements include adding LLM-as-judge graders, distinguishing capability vs regression evals, implementing partial credit, adding human calibration, and expanding evaluation dimensions. These improvements would make the system more aligned with industry best practices and better suited for continuous skill improvement.
