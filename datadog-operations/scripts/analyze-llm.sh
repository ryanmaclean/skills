#!/bin/bash
set -e

# Analyze LLM observability data for GenAI applications
# Query token usage, costs, latency, and errors for LLM-powered services

# Check required environment variables
if [ -z "$DD_API_KEY" ] || [ -z "$DD_APP_KEY" ]; then
    echo "[ERROR] DD_API_KEY and DD_APP_KEY must be set" >&2
    exit 1
fi

DD_SITE=${DD_SITE:-datadoghq.com}

# Parse arguments
SERVICE=""
MODEL=""
DURATION="24h"
OUTPUT_FORMAT="json"

while [[ $# -gt 0 ]]; do
    case $1 in
        --service)
            SERVICE="$2"
            shift 2
            ;;
        --model)
            MODEL="$2"
            shift 2
            ;;
        --duration)
            DURATION="$2"
            shift 2
            ;;
        --format)
            OUTPUT_FORMAT="$2"
            shift 2
            ;;
        *)
            echo "[ERROR] Unknown argument: $1" >&2
            echo "Usage: $0 --service SERVICE [--model MODEL] [--duration DURATION] [--format json|summary]" >&2
            exit 1
            ;;
    esac
done

if [ -z "$SERVICE" ]; then
    echo "[ERROR] --service is required" >&2
    exit 1
fi

# Model pricing per 1K tokens (input/output)
# Prices as of January 2026
# Bash 3.2 compatible function-based pricing
get_model_pricing() {
    local model_name="$1"
    local token_type="$2"  # input or output

    case "$model_name" in
        gpt-4)
            [ "$token_type" = "input" ] && echo "0.03" || echo "0.06"
            ;;
        gpt-4-32k)
            [ "$token_type" = "input" ] && echo "0.06" || echo "0.12"
            ;;
        gpt-4-turbo)
            [ "$token_type" = "input" ] && echo "0.01" || echo "0.03"
            ;;
        gpt-3.5-turbo)
            [ "$token_type" = "input" ] && echo "0.0015" || echo "0.002"
            ;;
        claude-3-5-sonnet)
            [ "$token_type" = "input" ] && echo "0.003" || echo "0.015"
            ;;
        claude-3-opus)
            [ "$token_type" = "input" ] && echo "0.015" || echo "0.075"
            ;;
        claude-3-sonnet)
            [ "$token_type" = "input" ] && echo "0.003" || echo "0.015"
            ;;
        claude-3-haiku)
            [ "$token_type" = "input" ] && echo "0.00025" || echo "0.00125"
            ;;
        claude-2.1)
            [ "$token_type" = "input" ] && echo "0.008" || echo "0.024"
            ;;
        gemini-pro)
            [ "$token_type" = "input" ] && echo "0.00025" || echo "0.00125"
            ;;
        gemini-ultra)
            [ "$token_type" = "input" ] && echo "0.005" || echo "0.015"
            ;;
        *)
            # Default pricing for unknown models
            [ "$token_type" = "input" ] && echo "0.03" || echo "0.06"
            ;;
    esac
}

# Convert duration to timestamps
case $DURATION in
    1h)
        FROM_TS=$(($(date +%s) - 3600))
        ;;
    24h)
        FROM_TS=$(($(date +%s) - 86400))
        ;;
    7d)
        FROM_TS=$(($(date +%s) - 604800))
        ;;
    30d)
        FROM_TS=$(($(date +%s) - 2592000))
        ;;
    *)
        echo "[ERROR] Invalid duration. Use: 1h, 24h, 7d, or 30d" >&2
        exit 1
        ;;
esac

TO_TS=$(date +%s)
FROM_NS=${FROM_TS}000000000
TO_NS=${TO_TS}000000000

# Build query
QUERY="service:$SERVICE"
if [ -n "$MODEL" ]; then
    QUERY="$QUERY llm.model:$MODEL"
fi

echo "[INFO] Analyzing LLM observability data..." >&2
echo "[INFO] Service: $SERVICE" >&2
echo "[INFO] Duration: $DURATION" >&2
if [ -n "$MODEL" ]; then
    echo "[INFO] Model filter: $MODEL" >&2
fi
echo "" >&2

# Query 1: Token usage and latency by operation
echo "[INFO] Querying token usage and latency..." >&2
TOKEN_RESPONSE=$(curl -s -X POST "https://api.${DD_SITE}/api/v2/spans/analytics/aggregate" \
  -H "DD-API-KEY: ${DD_API_KEY}" \
  -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
  -H "Content-Type: application/json" \
  -d "{
    \"data\": {
      \"type\": \"aggregate_request\",
      \"attributes\": {
        \"filter\": {
          \"from\": \"now-${DURATION}\",
          \"to\": \"now\",
          \"query\": \"${QUERY}\"
        },
        \"compute\": [
          {\"aggregation\": \"count\"},
          {\"aggregation\": \"sum\", \"metric\": \"@llm.tokens.prompt\"},
          {\"aggregation\": \"sum\", \"metric\": \"@llm.tokens.completion\"},
          {\"aggregation\": \"sum\", \"metric\": \"@llm.tokens.total\"},
          {\"aggregation\": \"avg\", \"metric\": \"@llm.tokens.prompt\"},
          {\"aggregation\": \"avg\", \"metric\": \"@llm.tokens.completion\"},
          {\"aggregation\": \"avg\", \"metric\": \"@duration\"},
          {\"aggregation\": \"max\", \"metric\": \"@duration\"}
        ],
        \"group_by\": [
          {\"facet\": \"resource_name\", \"limit\": 20}
        ]
      }
    }
  }")

# Check for API errors
if echo "$TOKEN_RESPONSE" | jq -e '.errors' > /dev/null 2>&1; then
    echo "[ERROR] API error:" >&2
    echo "$TOKEN_RESPONSE" | jq '.errors' >&2
    exit 1
fi

# Check if data exists (v2 API format: .data is array of items)
if ! echo "$TOKEN_RESPONSE" | jq -e '.data' > /dev/null 2>&1 || [ "$(echo "$TOKEN_RESPONSE" | jq '.data | length')" -eq 0 ]; then
    echo "[WARN] No LLM trace data found" >&2
    cat <<EOF
{
  "status": "no_data",
  "service": "$SERVICE",
  "model_filter": "${MODEL:-all}",
  "duration": "$DURATION",
  "message": "No LLM spans found. Ensure LLM instrumentation is enabled and spans are tagged with llm.* attributes."
}
EOF
    exit 0
fi

# Query 2: Error rates by model
echo "[INFO] Querying error rates..." >&2
ERROR_RESPONSE=$(curl -s -X POST "https://api.${DD_SITE}/api/v2/spans/analytics/aggregate" \
  -H "DD-API-KEY: ${DD_API_KEY}" \
  -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
  -H "Content-Type: application/json" \
  -d "{
    \"data\": {
      \"type\": \"aggregate_request\",
      \"attributes\": {
        \"filter\": {
          \"from\": \"now-${DURATION}\",
          \"to\": \"now\",
          \"query\": \"${QUERY}\"
        },
        \"compute\": [
          {\"aggregation\": \"count\"},
          {\"aggregation\": \"cardinality\", \"metric\": \"@trace_id\"}
        ],
        \"group_by\": [
          {\"facet\": \"@llm.model\", \"limit\": 20},
          {\"facet\": \"status\", \"limit\": 10}
        ]
      }
    }
  }")

# Parse token usage data (v2 API format: .data[].attributes.by and .data[].attributes.compute)
OPERATIONS=$(echo "$TOKEN_RESPONSE" | jq -c '[.data[] | {
  operation: .attributes.by.resource_name,
  request_count: .attributes.compute.c0,
  total_prompt_tokens: (.attributes.compute.c1 // 0 | floor),
  total_completion_tokens: (.attributes.compute.c2 // 0 | floor),
  total_tokens: (.attributes.compute.c3 // 0 | floor),
  avg_prompt_tokens: (.attributes.compute.c4 // 0 | floor),
  avg_completion_tokens: (.attributes.compute.c5 // 0 | floor),
  avg_ms: ((.attributes.compute.c6 // 0) / 1000000 | floor),
  max_ms: ((.attributes.compute.c7 // 0) / 1000000 | floor)
}]')

# Parse error data by model (v2 API format)
MODELS_DATA=$(echo "$ERROR_RESPONSE" | jq -c '[
  .data | group_by(.attributes.by["@llm.model"])[] | {
    model: (.[0].attributes.by["@llm.model"] // "unknown"),
    total_requests: ([.[].attributes.compute.c0] | add // 0),
    error_count: ([.[] | select(.attributes.by.status == "error") | .attributes.compute.c0] | add // 0),
    error_rate: (([.[] | select(.attributes.by.status == "error") | .attributes.compute.c0] | add // 0) / ([.[].attributes.compute.c0] | add // 1) * 100 | floor)
  }
]')

# Calculate costs for each operation
# Uses default GPT-4 pricing (0.03 input, 0.06 output per 1K tokens)
OPERATIONS_WITH_COST=$(echo "$OPERATIONS" | jq -c '[.[] | . + {
    estimated_cost_usd: (
      if .total_prompt_tokens > 0 and .total_completion_tokens > 0 then
        (.total_prompt_tokens / 1000 * 0.03) + (.total_completion_tokens / 1000 * 0.06)
      else
        0
      end | (. * 100 | floor) / 100
    ),
    cost_per_request_usd: (
      if .request_count > 0 and .total_tokens > 0 then
        ((.total_prompt_tokens / 1000 * 0.03) + (.total_completion_tokens / 1000 * 0.06)) / .request_count
      else
        0
      end | (. * 1000 | floor) / 1000
    )
  }]')

# Calculate summary statistics
TOTAL_REQUESTS=$(echo "$OPERATIONS_WITH_COST" | jq '[.[].request_count] | add // 0')
TOTAL_TOKENS=$(echo "$OPERATIONS_WITH_COST" | jq '[.[].total_tokens] | add // 0')
TOTAL_PROMPT_TOKENS=$(echo "$OPERATIONS_WITH_COST" | jq '[.[].total_prompt_tokens] | add // 0')
TOTAL_COMPLETION_TOKENS=$(echo "$OPERATIONS_WITH_COST" | jq '[.[].total_completion_tokens] | add // 0')
TOTAL_COST=$(echo "$OPERATIONS_WITH_COST" | jq '[.[].estimated_cost_usd] | add // 0 | (. * 100 | floor) / 100')
AVG_COST_PER_REQUEST=$(echo "$OPERATIONS_WITH_COST" | jq '[.[].cost_per_request_usd] | add / length // 0 | (. * 1000 | floor) / 1000')
AVG_LATENCY=$(echo "$OPERATIONS_WITH_COST" | jq '[.[].avg_ms] | add / length // 0 | floor')
AVG_TOKENS_PER_REQUEST=$(echo "$OPERATIONS_WITH_COST" | jq '([.[].total_tokens] | add // 0) / ([.[].request_count] | add // 1) | floor')

# Identify high-cost operations
HIGH_COST_OPS=$(echo "$OPERATIONS_WITH_COST" | jq '[.[] | select(.estimated_cost_usd > 1)] | length')
SLOW_OPS=$(echo "$OPERATIONS_WITH_COST" | jq '[.[] | select(.avg_ms > 2000)] | length')

# Calculate error rate
TOTAL_ERRORS=$(echo "$MODELS_DATA" | jq '[.[].error_count] | add // 0')
OVERALL_ERROR_RATE=$(echo "$TOTAL_REQUESTS $TOTAL_ERRORS" | awk '{if ($1 > 0) printf "%.2f", ($2 / $1) * 100; else print 0}')

# Print summary to stderr
echo "" >&2
echo "[SUMMARY] LLM Performance Analysis" >&2
echo "Total requests: $TOTAL_REQUESTS" >&2
echo "Total tokens: $TOTAL_TOKENS (prompt: $TOTAL_PROMPT_TOKENS, completion: $TOTAL_COMPLETION_TOKENS)" >&2
echo "Average tokens per request: $AVG_TOKENS_PER_REQUEST" >&2
echo "Estimated total cost: \$${TOTAL_COST}" >&2
echo "Average cost per request: \$${AVG_COST_PER_REQUEST}" >&2
echo "Average latency: ${AVG_LATENCY}ms" >&2
echo "Overall error rate: ${OVERALL_ERROR_RATE}%" >&2
echo "" >&2

if [ "$HIGH_COST_OPS" -gt 0 ]; then
    echo "[WARN] Found $HIGH_COST_OPS high-cost operations (>\$1)" >&2
fi

if [ "$SLOW_OPS" -gt 0 ]; then
    echo "[WARN] Found $SLOW_OPS slow operations (P95 > 2000ms)" >&2
fi

# Top 5 most expensive operations
echo "[INFO] Top 5 most expensive operations:" >&2
echo "$OPERATIONS_WITH_COST" | jq -r '.[:5] | .[] | "  \(.operation): $\(.estimated_cost_usd) (\(.request_count) requests)"' >&2
echo "" >&2

# Cost optimization opportunities
OPTIMIZATION_SUGGESTIONS=()

# Check for high token usage
if [ "$AVG_TOKENS_PER_REQUEST" -gt 4000 ]; then
    OPTIMIZATION_SUGGESTIONS+=("{\"type\": \"high_token_usage\", \"message\": \"Average tokens per request is ${AVG_TOKENS_PER_REQUEST}. Consider prompt optimization or response truncation.\"}")
fi

# Check for slow operations
if [ "$AVG_LATENCY" -gt 3000 ]; then
    OPTIMIZATION_SUGGESTIONS+=("{\"type\": \"high_latency\", \"message\": \"Average latency is ${AVG_LATENCY}ms. Consider using faster models for non-critical operations.\"}")
fi

# Check for high costs
if (( $(echo "$TOTAL_COST > 100" | bc -l) )); then
    OPTIMIZATION_SUGGESTIONS+=("{\"type\": \"high_cost\", \"message\": \"Total cost for ${DURATION} is \$${TOTAL_COST}. Review if all operations require premium models.\"}")
fi

# Check error rate
if (( $(echo "$OVERALL_ERROR_RATE > 5" | bc -l) )); then
    OPTIMIZATION_SUGGESTIONS+=("{\"type\": \"high_error_rate\", \"message\": \"Error rate is ${OVERALL_ERROR_RATE}%. Investigate failed requests to avoid wasted token costs.\"}")
fi

SUGGESTIONS_JSON=$(printf '%s\n' "${OPTIMIZATION_SUGGESTIONS[@]}" | jq -s '.')

# Output format
if [ "$OUTPUT_FORMAT" = "summary" ]; then
    # Human-readable summary
    echo "LLM Observability Analysis - $SERVICE"
    echo "Duration: $DURATION"
    echo ""
    echo "Token Usage:"
    echo "  Total tokens: $TOTAL_TOKENS"
    echo "  Prompt tokens: $TOTAL_PROMPT_TOKENS"
    echo "  Completion tokens: $TOTAL_COMPLETION_TOKENS"
    echo "  Avg per request: $AVG_TOKENS_PER_REQUEST"
    echo ""
    echo "Cost Analysis:"
    echo "  Total cost: \$${TOTAL_COST}"
    echo "  Avg per request: \$${AVG_COST_PER_REQUEST}"
    echo "  High-cost ops: $HIGH_COST_OPS"
    echo ""
    echo "Performance:"
    echo "  Total requests: $TOTAL_REQUESTS"
    echo "  Avg latency: ${AVG_LATENCY}ms"
    echo "  Slow operations: $SLOW_OPS"
    echo "  Error rate: ${OVERALL_ERROR_RATE}%"
else
    # Structured JSON output
    cat <<EOF
{
  "status": "ok",
  "service": "$SERVICE",
  "model_filter": "${MODEL:-all}",
  "duration": "$DURATION",
  "time_range": {
    "from": $FROM_TS,
    "to": $TO_TS
  },
  "summary": {
    "total_requests": $TOTAL_REQUESTS,
    "total_tokens": $TOTAL_TOKENS,
    "total_prompt_tokens": $TOTAL_PROMPT_TOKENS,
    "total_completion_tokens": $TOTAL_COMPLETION_TOKENS,
    "avg_tokens_per_request": $AVG_TOKENS_PER_REQUEST,
    "total_cost_usd": $TOTAL_COST,
    "avg_cost_per_request_usd": $AVG_COST_PER_REQUEST,
    "avg_latency_ms": $AVG_LATENCY,
    "error_rate_percent": $OVERALL_ERROR_RATE,
    "high_cost_operations_count": $HIGH_COST_OPS,
    "slow_operations_count": $SLOW_OPS
  },
  "operations": $OPERATIONS_WITH_COST,
  "models": $MODELS_DATA,
  "optimization_suggestions": $SUGGESTIONS_JSON
}
EOF
fi
