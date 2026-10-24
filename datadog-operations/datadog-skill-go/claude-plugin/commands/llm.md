---
description: "Query LLM Observability for GenAI applications - monitor AI agents, trace LLM calls, track costs and performance"
argument-hint: "[APPLICATION] [--duration TIMERANGE] [--model MODEL] [--limit N]"
---

# Datadog LLM Observability

Query LLM Observability to monitor GenAI applications, AI agents, LLM calls, costs, latency, and quality metrics.

## What is LLM Observability?

Datadog LLM Observability provides end-to-end visibility into your AI applications:
- **Trace LLM calls** across OpenAI, Anthropic, AWS Bedrock, LangChain, and more
- **Monitor AI agents** with detailed step-by-step execution traces
- **Track costs** per model, user, session, or application
- **Measure quality** with evaluations, experiments, and prompt testing
- **Debug issues** with input/output capture and error tracking

**Official Documentation**: https://docs.datadoghq.com/llm_observability/

## Usage

```bash
# Query all LLM activity
dd llm

# Query specific application
dd llm my-chatbot

# Filter by time range
dd llm my-chatbot --duration 24h

# Filter by model
dd llm --model gpt-4 --duration 1h
```

## Key Metrics

**Performance**:
- Request latency (p50, p95, p99)
- Token generation speed
- Time to first token (TTFB)
- Total request duration

**Cost**:
- Token usage (input + output)
- Cost per request
- Cost by model, user, or application
- Daily/monthly spend trends

**Quality**:
- Error rates
- Prompt evaluation scores
- Output quality metrics
- User feedback ratings

**Volume**:
- Requests per second
- Tokens per second
- Active users
- Session counts

## Use Cases

### 1. Monitor AI Agent Performance
Track multi-step agent execution:
```bash
dd llm my-agent --duration 1h
```

**Provides**:
- Agent workflow traces
- Step-by-step latency
- Tool/function call performance
- Intermediate outputs

### 2. Optimize LLM Costs
Analyze spending patterns:
```bash
dd llm --duration 7d
```

**Insights**:
- Most expensive models
- High-cost users or sessions
- Token usage trends
- Cost optimization opportunities

### 3. Debug Production Issues
Investigate errors or slow responses:
```bash
dd llm my-app --duration 15m
```

**Debug Info**:
- Full request/response traces
- Error messages and stack traces
- Prompt and completion content
- Model parameters used

### 4. Compare Model Performance
Evaluate different models:
```bash
dd llm --model gpt-4 --duration 24h
dd llm --model claude-3-opus --duration 24h
```

**Comparison**:
- Latency differences
- Cost per request
- Error rates
- Quality scores

### 5. Track A/B Experiments
Monitor prompt or model experiments:
```bash
dd llm my-app --duration 7d
```

**Experiment Data**:
- Variant performance
- Quality metrics by variant
- Cost impact
- User engagement

## AI Agent Monitoring

**New in 2025**: Datadog now provides specialized AI Agent monitoring with:
- **Agent Console**: Visual workflow monitoring
- **Step-by-step traces**: See every agent decision
- **ROI tracking**: Measure agent business impact
- **Security scanning**: Check for prompt injection, data leakage
- **Compliance monitoring**: Track sensitive data handling

**Learn More**: https://www.datadoghq.com/product/ai/bits-ai-agents/

## Why Use the CLI?

**Advantages over Web UI**:
1. **Faster access** - 3ms startup vs loading dashboard
2. **Scriptable** - Automate monitoring in CI/CD
3. **Git context** - Auto-detects application from repo
4. **Offline access** - Query cached data without internet
5. **Programmatic** - Integrate with other tools

**Unique CLI Features**:
- Multi-signal correlation (LLM + APM + logs)
- Smart defaults (auto-detect app, timerange)
- Batch queries across multiple apps
- Export to CSV/JSON for analysis

## Supported Frameworks

The CLI queries data from LLM Observability integrations:

**LLM Providers**:
- OpenAI (GPT-3.5, GPT-4, etc.)
- Anthropic (Claude)
- AWS Bedrock
- Azure OpenAI
- Google Vertex AI
- Cohere
- Hugging Face

**Frameworks**:
- LangChain
- LlamaIndex
- Haystack
- Custom integrations

**Languages**:
- Python (dd-trace-py)
- Node.js (dd-trace-js)
- Java, Go, Ruby (via SDK)

## Output Format

Returns JSON with:
```json
{
  "application": "my-chatbot",
  "timerange": "1h",
  "metrics": {
    "requests": 1234,
    "avg_latency_ms": 850,
    "p99_latency_ms": 2100,
    "error_rate": 0.02,
    "total_tokens": 456789,
    "total_cost_usd": 12.34
  },
  "top_models": [
    {"model": "gpt-4-turbo", "requests": 800, "cost": 9.50},
    {"model": "gpt-3.5-turbo", "requests": 434, "cost": 2.84}
  ],
  "traces": [...]
}
```

## Example Prompts for Claude Code

> "Show me LLM Observability data for my chatbot application"
> "What's the cost of LLM usage in the last 24 hours?"
> "Query LLM traces with errors from the last hour"
> "How is my AI agent performing?"
> "Compare GPT-4 vs Claude cost and latency"

## Best Practices

**From Datadog's LLM Observability Guide**:

1. **Tag Everything**: Use consistent tags for application, environment, user, session
2. **Enable Sampling**: Sample high-volume applications (keep 100% of errors)
3. **Capture Metadata**: Store model parameters, prompt templates, versions
4. **Set Budgets**: Configure cost alerts per application
5. **Run Evaluations**: Test prompts before production deployment

**Security**:
- Use Sensitive Data Scanner to redact PII
- Monitor for prompt injection attempts
- Track data retention and compliance
- Audit model access patterns

**Performance**:
- Monitor cache hit rates
- Track streaming vs batch latency
- Measure time to first token
- Optimize prompt length

**Cost**:
- Set per-user spending limits
- Use cheaper models for simple tasks
- Implement caching strategies
- Monitor token usage trends

## Advanced Features

### Experiments
Run controlled A/B tests:
```bash
dd llm my-app --duration 7d
```

### Evaluations
Check quality metrics:
```bash
dd llm my-app --duration 24h
```

### Custom Metrics
Query custom tags:
```bash
dd llm my-app --duration 1h
```

## Troubleshooting

**No data showing?**
- Verify LLM Observability SDK is installed
- Check instrumentation is enabled
- Confirm API keys are set
- Review ingestion status in Datadog UI

**High costs?**
- Review token usage by model
- Check for duplicate requests
- Verify caching is enabled
- Consider cheaper models for simpler tasks

**Slow performance?**
- Check model latency vs API latency
- Review prompt length
- Verify streaming is enabled
- Check network connectivity

## Related Commands

- `dd apm` - Query APM traces (for full-stack visibility)
- `dd logs` - Search application logs
- `dd metrics` - Query custom metrics
- `dd cost` - Analyze Datadog usage costs

## Environment Variables

Required:
- `DD_API_KEY`: Datadog API key
- `DD_APP_KEY`: Datadog application key
- `DD_SITE`: Datadog site (default: datadoghq.com)

## Learn More

**Official Resources**:
- [LLM Observability Docs](https://docs.datadoghq.com/llm_observability/)
- [LLM Observability Product Page](https://www.datadoghq.com/product/llm-observability/)
- [Bits AI Agents](https://www.datadoghq.com/product/ai/bits-ai-agents/)
- [OpenAI Agents Guide](https://www.datadoghq.com/blog/openai-agents-llm-observability/)
- [LLM Observability GitHub Examples](https://github.com/DataDog/llm-observability)

**Integration Guides**:
- [Python SDK](https://docs.datadoghq.com/llm_observability/instrumentation/sdk/)
- [Quickstart Guide](https://docs.datadoghq.com/llm_observability/quickstart/)
- [Best Practices](https://docs.datadoghq.com/llm_observability/best_practices/)

## Notes

- LLM Observability is Datadog's newest capability (2024-2025)
- Specialized for GenAI/AI agent monitoring
- Complements APM with LLM-specific metrics
- Essential for production AI applications
- Supports all major LLM providers and frameworks

## Recent Updates

**✅ FIXED (Jan 22, 2026 - Iteration 25):**
LLM command now fully functional. Previous API validation errors have been resolved by fixing the request format to match Datadog's API v2 specifications. All LLM Observability features are now working.
