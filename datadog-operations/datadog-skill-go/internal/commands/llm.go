package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/datadog/skill/internal/client"
	"github.com/datadog/skill/internal/context"
	"github.com/datadog/skill/internal/observability"
)

// LLMCommand queries Datadog LLM observability for GenAI applications
type LLMCommand struct {
	flags    *flag.FlagSet
	service  string
	duration string
	model    string
	limit    int
	jsonOut  bool
}

// ModelPricing represents pricing per 1K tokens for different LLM models
type ModelPricing struct {
	Input  float64 `json:"input"`
	Output float64 `json:"output"`
}

// Model pricing per 1K tokens (January 2026)
var modelPricing = map[string]ModelPricing{
	"gpt-4":             {Input: 0.03, Output: 0.06},
	"gpt-4-32k":         {Input: 0.06, Output: 0.12},
	"gpt-4-turbo":       {Input: 0.01, Output: 0.03},
	"gpt-3.5-turbo":     {Input: 0.0015, Output: 0.002},
	"claude-3-5-sonnet": {Input: 0.003, Output: 0.015},
	"claude-3-opus":     {Input: 0.015, Output: 0.075},
	"claude-3-sonnet":   {Input: 0.003, Output: 0.015},
	"claude-3-haiku":    {Input: 0.00025, Output: 0.00125},
	"claude-2.1":        {Input: 0.008, Output: 0.024},
	"gemini-pro":        {Input: 0.00025, Output: 0.00125},
	"gemini-ultra":      {Input: 0.005, Output: 0.015},
}

var defaultPricing = ModelPricing{Input: 0.03, Output: 0.06}

// LLMResponse represents the response from Datadog APM API for LLM queries
type LLMResponse struct {
	Data []struct {
		Type       string `json:"type"`
		ID         string `json:"id"`
		Attributes struct {
			By      map[string]interface{} `json:"by"`
			Compute map[string]interface{} `json:"compute"` // Note: compute not computes
		} `json:"attributes"`
	} `json:"data"`
	Meta struct {
		Elapsed   int    `json:"elapsed"`
		RequestID string `json:"request_id"`
		Status    string `json:"status"`
	} `json:"meta"`
}

// LLMOperation represents statistics for a single LLM operation
type LLMOperation struct {
	Operation              string  `json:"operation"`
	RequestCount           int64   `json:"request_count"`
	TotalPromptTokens      int64   `json:"total_prompt_tokens"`
	TotalCompletionTokens  int64   `json:"total_completion_tokens"`
	TotalTokens            int64   `json:"total_tokens"`
	AvgPromptTokens        int64   `json:"avg_prompt_tokens"`
	AvgCompletionTokens    int64   `json:"avg_completion_tokens"`
	P50Ms                  int64   `json:"p50_ms"`
	P95Ms                  int64   `json:"p95_ms"`
	P99Ms                  int64   `json:"p99_ms"`
	EstimatedCostUSD       float64 `json:"estimated_cost_usd"`
	CostPerRequestUSD      float64 `json:"cost_per_request_usd"`
}

// LLMModelStats represents statistics grouped by model
type LLMModelStats struct {
	Model         string  `json:"model"`
	TotalRequests int64   `json:"total_requests"`
	ErrorCount    int64   `json:"error_count"`
	ErrorRate     float64 `json:"error_rate"`
}

// LLMRecommendation represents a cost optimization recommendation
type LLMRecommendation struct {
	Type     string `json:"type"`
	Priority string `json:"priority"`
	Message  string `json:"message"`
}

// LLMOutput represents the structured output
type LLMOutput struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Model     string `json:"model_filter"`
	Duration  string `json:"duration"`
	TimeRange *struct {
		From int64 `json:"from"`
		To   int64 `json:"to"`
	} `json:"time_range,omitempty"`
	Summary *struct {
		TotalRequests          int64   `json:"total_requests"`
		TotalTokens            int64   `json:"total_tokens"`
		TotalPromptTokens      int64   `json:"total_prompt_tokens"`
		TotalCompletionTokens  int64   `json:"total_completion_tokens"`
		AvgTokensPerRequest    int64   `json:"avg_tokens_per_request"`
		TotalCostUSD           float64 `json:"total_cost_usd"`
		AvgCostPerRequestUSD   float64 `json:"avg_cost_per_request_usd"`
		AvgP95LatencyMs        int64   `json:"avg_p95_latency_ms"`
		ErrorRatePercent       float64 `json:"error_rate_percent"`
		HighCostOperations     int     `json:"high_cost_operations_count"`
		SlowOperations         int     `json:"slow_operations_count"`
	} `json:"summary,omitempty"`
	Operations            []LLMOperation      `json:"operations"`
	Models                []LLMModelStats     `json:"models,omitempty"`
	OptimizationSuggestions []LLMRecommendation `json:"optimization_suggestions,omitempty"`
	Message               string              `json:"message,omitempty"`
}

// NewLLMCommand creates a new LLM observability command
func NewLLMCommand() *LLMCommand {
	cmd := &LLMCommand{
		flags: flag.NewFlagSet("llm", flag.ExitOnError),
	}

	cmd.flags.StringVar(&cmd.service, "service", "", "LLM service name (auto-detected if not provided)")
	cmd.flags.StringVar(&cmd.duration, "duration", "1h", "Time range: 1h, 24h, 7d (default: 1h)")
	cmd.flags.StringVar(&cmd.model, "model", "", "Filter by model name (gpt-4, claude, etc.)")
	cmd.flags.IntVar(&cmd.limit, "limit", 20, "Number of operations to return (default: 20)")
	cmd.flags.BoolVar(&cmd.jsonOut, "json", false, "Output as JSON")

	return cmd
}

func (c *LLMCommand) Name() string {
	return "llm"
}

func (c *LLMCommand) Description() string {
	return "Query Datadog LLM observability for GenAI applications"
}

func (c *LLMCommand) Run(args []string) error {
	// Initialize observability
	obs, err := observability.Init("query-llm", "production")
	if err != nil {
		return fmt.Errorf("failed to init observability: %w", err)
	}
	defer obs.Shutdown(0)

	// Parse flags
	if err := c.flags.Parse(args); err != nil {
		return err
	}

	// Detect service if not provided
	serviceName := c.service
	if serviceName == "" {
		span := obs.StartSpan("detect_context")
		obs.LogInfo("Auto-detecting service name")

		ctx, err := context.DetectContext(".")
		obs.FinishSpan(span)

		if err != nil || ctx.ServiceName == "" {
			obs.LogError("Could not detect service name")
			return fmt.Errorf("could not detect service name: specify with --service or run in a git repository")
		}

		serviceName = ctx.ServiceName
		obs.LogInfo(fmt.Sprintf("Auto-detected service: %s", serviceName))
	}

	obs.LogInfo(fmt.Sprintf("Querying LLM observability for service: %s", serviceName))

	// Parse duration to time range
	span := obs.StartSpan("parse_duration")
	fromTime, toTime, err := c.parseDuration(c.duration)
	obs.FinishSpan(span)

	if err != nil {
		obs.LogError(fmt.Sprintf("Invalid duration: %s", err.Error()))
		return fmt.Errorf("invalid duration: %w", err)
	}

	// Create Datadog client
	span = obs.StartSpan("create_client")
	ddClient, err := client.NewClient()
	obs.FinishSpan(span)

	if err != nil {
		obs.LogError(fmt.Sprintf("Failed to create Datadog client: %s", err.Error()))
		return fmt.Errorf("failed to create Datadog client: %w", err)
	}

	// Query 1: Token usage and latency by operation
	span = obs.StartSpan("query_token_usage")
	obs.GetTracer().SetTag(span, "service", serviceName)
	obs.GetTracer().SetTag(span, "duration", c.duration)
	if c.model != "" {
		obs.GetTracer().SetTag(span, "model", c.model)
	}

	start := time.Now()
	tokenData, err := c.queryTokenUsage(ddClient, serviceName, fromTime, toTime)
	apiDuration := time.Since(start).Milliseconds()
	obs.FinishSpan(span)

	if err != nil {
		obs.RecordAPICall("/api/v2/spans/analytics/aggregate", "POST", 500, float64(apiDuration), err)
		obs.LogError(fmt.Sprintf("Token usage query failed: %s", err.Error()))

		// Check for known API validation error (400)
		errMsg := err.Error()
		if strings.Contains(errMsg, "status 400") || strings.Contains(errMsg, "validation") {
			return fmt.Errorf("failed to query token usage: %w\n\n"+
				"Known Issue: LLM aggregate queries have API format issues.\n"+
				"Workarounds:\n"+
				"  1. Use Datadog web UI for LLM Observability\n"+
				"  2. View LLM metrics in dashboards\n"+
				"  3. Use custom metrics for LLM monitoring\n"+
				"See KNOWN-ISSUES.md for details", err)
		}

		return fmt.Errorf("failed to query token usage: %w\n\n"+
			"Troubleshooting:\n"+
			"  1. Check DD_API_KEY and DD_APP_KEY are set\n"+
			"  2. Verify LLM instrumentation is enabled\n"+
			"  3. Check KNOWN-ISSUES.md for known bugs", err)
	}

	obs.RecordAPICall("/api/v2/spans/analytics/aggregate", "POST", 200, float64(apiDuration), nil)

	// Parse token usage results
	span = obs.StartSpan("parse_token_results")
	operations, output, err := c.parseTokenResults(tokenData, serviceName, c.duration)
	obs.FinishSpan(span)

	if err != nil {
		obs.LogError(fmt.Sprintf("Failed to parse token results: %s", err.Error()))
		return fmt.Errorf("failed to parse token results: %w", err)
	}

	// Handle no data case
	if len(operations) == 0 {
		obs.LogWarning("No LLM trace data found")
		obs.GetMetrics().Gauge("llm.operations", 0, "service:"+serviceName)

		output.Message = "No LLM spans found. Ensure LLM instrumentation is enabled and spans are tagged with llm.* attributes."
		if c.jsonOut {
			jsonData, _ := json.MarshalIndent(output, "", "  ")
			fmt.Println(string(jsonData))
		} else {
			fmt.Printf("No LLM trace data found for service: %s\n", serviceName)
			fmt.Println(output.Message)
		}

		return nil
	}

	// Query 2: Error rates by model
	span = obs.StartSpan("query_error_rates")
	start = time.Now()
	errorData, err := c.queryErrorRates(ddClient, serviceName, fromTime, toTime)
	apiDuration = time.Since(start).Milliseconds()
	obs.FinishSpan(span)

	if err != nil {
		obs.RecordAPICall("/api/v2/spans/analytics/aggregate", "POST", 500, float64(apiDuration), err)
		obs.LogWarning(fmt.Sprintf("Error rate query failed: %s", err.Error()))
		// Continue without error data
	} else {
		obs.RecordAPICall("/api/v2/spans/analytics/aggregate", "POST", 200, float64(apiDuration), nil)

		// Parse error results
		span = obs.StartSpan("parse_error_results")
		models := c.parseErrorResults(errorData)
		obs.FinishSpan(span)
		output.Models = models
	}

	// Calculate summary statistics
	span = obs.StartSpan("calculate_summary")
	summary := c.calculateSummary(operations, output.Models)
	obs.FinishSpan(span)
	output.Summary = summary

	// Set time range
	output.TimeRange = &struct {
		From int64 `json:"from"`
		To   int64 `json:"to"`
	}{
		From: fromTime.Unix(),
		To:   toTime.Unix(),
	}

	// Generate recommendations
	span = obs.StartSpan("generate_recommendations")
	recommendations := c.generateRecommendations(summary)
	obs.FinishSpan(span)
	output.OptimizationSuggestions = recommendations

	// Record metrics
	obs.GetMetrics().Gauge("llm.operations", float64(len(operations)), "service:"+serviceName)
	obs.GetMetrics().Gauge("llm.requests", float64(summary.TotalRequests), "service:"+serviceName)
	obs.GetMetrics().Gauge("llm.tokens", float64(summary.TotalTokens), "service:"+serviceName)
	obs.GetMetrics().Gauge("llm.cost_usd", summary.TotalCostUSD, "service:"+serviceName)
	obs.GetMetrics().Gauge("llm.error_rate", summary.ErrorRatePercent, "service:"+serviceName)

	// Output
	if c.jsonOut {
		jsonData, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	} else {
		c.printFormatted(serviceName, c.duration, summary, operations, recommendations)
	}

	obs.LogInfo(fmt.Sprintf("Query completed: %d operations, %d tokens, $%.2f cost", len(operations), summary.TotalTokens, summary.TotalCostUSD))
	return nil
}

func (c *LLMCommand) parseDuration(duration string) (time.Time, time.Time, error) {
	var d time.Duration
	var err error

	if strings.HasSuffix(duration, "h") {
		hours := strings.TrimSuffix(duration, "h")
		var h int
		_, err = fmt.Sscanf(hours, "%d", &h)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid duration format: %s", duration)
		}
		d = time.Duration(h) * time.Hour
	} else if strings.HasSuffix(duration, "d") {
		days := strings.TrimSuffix(duration, "d")
		var day int
		_, err = fmt.Sscanf(days, "%d", &day)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid duration format: %s", duration)
		}
		d = time.Duration(day) * 24 * time.Hour
	} else {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid duration format: %s (use format like '1h', '24h', '7d')", duration)
	}

	toTime := time.Now()
	fromTime := toTime.Add(-d)

	return fromTime, toTime, nil
}

func (c *LLMCommand) queryTokenUsage(ddClient *client.Client, serviceName string, fromTime, toTime time.Time) ([]byte, error) {
	query := fmt.Sprintf("service:%s", serviceName)
	if c.model != "" {
		query += fmt.Sprintf(" llm.model:%s", c.model)
	}

	// API v2 requires data.attributes wrapper - same format as APM
	type DataAttributes struct {
		Filter  map[string]interface{}   `json:"filter"`
		Compute []map[string]interface{} `json:"compute"`
		GroupBy []map[string]interface{} `json:"group_by"`
	}

	type AggregateData struct {
		Type       string         `json:"type"`
		Attributes DataAttributes `json:"attributes"`
	}

	type AggregateRequest struct {
		Data AggregateData `json:"data"`
	}

	payload := AggregateRequest{
		Data: AggregateData{
			Type: "aggregate_request",
			Attributes: DataAttributes{
				Filter: map[string]interface{}{
					"from":  fromTime.Format(time.RFC3339),
					"to":    toTime.Format(time.RFC3339),
					"query": query,
				},
				Compute: []map[string]interface{}{
					{"aggregation": "count", "type": "total"},
					{"aggregation": "sum", "metric": "llm.tokens.prompt", "type": "total"},
					{"aggregation": "sum", "metric": "llm.tokens.completion", "type": "total"},
					{"aggregation": "sum", "metric": "llm.tokens.total", "type": "total"},
					{"aggregation": "avg", "metric": "llm.tokens.prompt", "type": "total"},
					{"aggregation": "avg", "metric": "llm.tokens.completion", "type": "total"},
					{"aggregation": "median", "metric": "@duration", "type": "total"}, // was pc50
					{"aggregation": "pc95", "metric": "@duration", "type": "total"},
					{"aggregation": "pc99", "metric": "@duration", "type": "total"},
				},
				GroupBy: []map[string]interface{}{
					{
						"facet": "resource_name",
						"limit": c.limit,
						// Note: sort removed - "aggregation" field in sort causes API validation error
					},
				},
			},
		},
	}

	return ddClient.QueryLLMSpans(&payload)
}

func (c *LLMCommand) queryErrorRates(ddClient *client.Client, serviceName string, fromTime, toTime time.Time) ([]byte, error) {
	query := fmt.Sprintf("service:%s", serviceName)
	if c.model != "" {
		query += fmt.Sprintf(" llm.model:%s", c.model)
	}

	// API v2 requires data.attributes wrapper - same format as APM
	type DataAttributes struct {
		Filter  map[string]interface{}   `json:"filter"`
		Compute []map[string]interface{} `json:"compute"`
		GroupBy []map[string]interface{} `json:"group_by"`
	}

	type AggregateData struct {
		Type       string         `json:"type"`
		Attributes DataAttributes `json:"attributes"`
	}

	type AggregateRequest struct {
		Data AggregateData `json:"data"`
	}

	payload := AggregateRequest{
		Data: AggregateData{
			Type: "aggregate_request",
			Attributes: DataAttributes{
				Filter: map[string]interface{}{
					"from":  fromTime.Format(time.RFC3339),
					"to":    toTime.Format(time.RFC3339),
					"query": query,
				},
				Compute: []map[string]interface{}{
					{"aggregation": "count", "type": "total"},
					{"aggregation": "cardinality", "metric": "trace_id", "type": "total"},
				},
				GroupBy: []map[string]interface{}{
					{"facet": "llm.model", "limit": 20},
					{"facet": "status", "limit": 10},
				},
			},
		},
	}

	return ddClient.QueryLLMSpans(&payload)
}

func (c *LLMCommand) parseTokenResults(data []byte, serviceName, duration string) ([]LLMOperation, *LLMOutput, error) {
	var response LLMResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	output := &LLMOutput{
		Status:     "ok",
		Service:    serviceName,
		Model:      c.model,
		Duration:   duration,
		Operations: []LLMOperation{},
	}

	// Check if we have data
	if len(response.Data) == 0 {
		output.Status = "no_data"
		return []LLMOperation{}, output, nil
	}

	operations := make([]LLMOperation, 0, len(response.Data))

	for _, bucket := range response.Data {
		operation := "unknown"
		if resName, ok := bucket.Attributes.By["resource_name"].(string); ok {
			operation = resName
		}

		computes := bucket.Attributes.Compute

		// Extract metrics
		var requestCount int64
		var totalPrompt, totalCompletion, totalTokens int64
		var avgPrompt, avgCompletion int64
		var p50Ns, p95Ns, p99Ns int64

		if val, ok := computes["c0"].(float64); ok {
			requestCount = int64(val)
		}
		if val, ok := computes["c1"].(float64); ok {
			totalPrompt = int64(val)
		}
		if val, ok := computes["c2"].(float64); ok {
			totalCompletion = int64(val)
		}
		if val, ok := computes["c3"].(float64); ok {
			totalTokens = int64(val)
		}
		if val, ok := computes["c4"].(float64); ok {
			avgPrompt = int64(val)
		}
		if val, ok := computes["c5"].(float64); ok {
			avgCompletion = int64(val)
		}
		if val, ok := computes["c6"].(float64); ok {
			p50Ns = int64(val)
		}
		if val, ok := computes["c7"].(float64); ok {
			p95Ns = int64(val)
		}
		if val, ok := computes["c8"].(float64); ok {
			p99Ns = int64(val)
		}

		// Calculate cost
		estimatedCost := c.calculateCost(float64(totalPrompt), float64(totalCompletion), c.model)
		costPerRequest := 0.0
		if requestCount > 0 {
			costPerRequest = estimatedCost / float64(requestCount)
		}

		op := LLMOperation{
			Operation:             operation,
			RequestCount:          requestCount,
			TotalPromptTokens:     totalPrompt,
			TotalCompletionTokens: totalCompletion,
			TotalTokens:           totalTokens,
			AvgPromptTokens:       avgPrompt,
			AvgCompletionTokens:   avgCompletion,
			P50Ms:                 p50Ns / 1_000_000,
			P95Ms:                 p95Ns / 1_000_000,
			P99Ms:                 p99Ns / 1_000_000,
			EstimatedCostUSD:      roundFloat(estimatedCost, 2),
			CostPerRequestUSD:     roundFloat(costPerRequest, 3),
		}

		operations = append(operations, op)
	}

	output.Operations = operations
	return operations, output, nil
}

func (c *LLMCommand) parseErrorResults(data []byte) []LLMModelStats {
	var response LLMResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return []LLMModelStats{}
	}

	// Group by model
	modelsMap := make(map[string]*LLMModelStats)

	for _, bucket := range response.Data {
		modelName := "unknown"
		if model, ok := bucket.Attributes.By["llm_model"].(string); ok {
			modelName = model
		}

		status := "ok"
		if s, ok := bucket.Attributes.By["status"].(string); ok {
			status = s
		}

		var count int64
		if val, ok := bucket.Attributes.Compute["c0"].(float64); ok {
			count = int64(val)
		}

		if _, exists := modelsMap[modelName]; !exists {
			modelsMap[modelName] = &LLMModelStats{
				Model: modelName,
			}
		}

		modelsMap[modelName].TotalRequests += count
		if status == "error" {
			modelsMap[modelName].ErrorCount += count
		}
	}

	// Calculate error rates and convert to slice
	models := make([]LLMModelStats, 0, len(modelsMap))
	for _, stats := range modelsMap {
		if stats.TotalRequests > 0 {
			stats.ErrorRate = roundFloat((float64(stats.ErrorCount)/float64(stats.TotalRequests))*100, 1)
		}
		models = append(models, *stats)
	}

	// Sort by total requests descending
	sort.Slice(models, func(i, j int) bool {
		return models[i].TotalRequests > models[j].TotalRequests
	})

	return models
}

func (c *LLMCommand) calculateCost(promptTokens, completionTokens float64, model string) float64 {
	pricing := defaultPricing
	if p, ok := modelPricing[model]; ok {
		pricing = p
	}

	promptCost := (promptTokens / 1000) * pricing.Input
	completionCost := (completionTokens / 1000) * pricing.Output

	return promptCost + completionCost
}

func (c *LLMCommand) calculateSummary(operations []LLMOperation, models []LLMModelStats) *struct {
	TotalRequests         int64   `json:"total_requests"`
	TotalTokens           int64   `json:"total_tokens"`
	TotalPromptTokens     int64   `json:"total_prompt_tokens"`
	TotalCompletionTokens int64   `json:"total_completion_tokens"`
	AvgTokensPerRequest   int64   `json:"avg_tokens_per_request"`
	TotalCostUSD          float64 `json:"total_cost_usd"`
	AvgCostPerRequestUSD  float64 `json:"avg_cost_per_request_usd"`
	AvgP95LatencyMs       int64   `json:"avg_p95_latency_ms"`
	ErrorRatePercent      float64 `json:"error_rate_percent"`
	HighCostOperations    int     `json:"high_cost_operations_count"`
	SlowOperations        int     `json:"slow_operations_count"`
} {
	summary := &struct {
		TotalRequests         int64   `json:"total_requests"`
		TotalTokens           int64   `json:"total_tokens"`
		TotalPromptTokens     int64   `json:"total_prompt_tokens"`
		TotalCompletionTokens int64   `json:"total_completion_tokens"`
		AvgTokensPerRequest   int64   `json:"avg_tokens_per_request"`
		TotalCostUSD          float64 `json:"total_cost_usd"`
		AvgCostPerRequestUSD  float64 `json:"avg_cost_per_request_usd"`
		AvgP95LatencyMs       int64   `json:"avg_p95_latency_ms"`
		ErrorRatePercent      float64 `json:"error_rate_percent"`
		HighCostOperations    int     `json:"high_cost_operations_count"`
		SlowOperations        int     `json:"slow_operations_count"`
	}{}

	// Calculate totals from operations
	var totalP95 int64
	for _, op := range operations {
		summary.TotalRequests += op.RequestCount
		summary.TotalTokens += op.TotalTokens
		summary.TotalPromptTokens += op.TotalPromptTokens
		summary.TotalCompletionTokens += op.TotalCompletionTokens
		summary.TotalCostUSD += op.EstimatedCostUSD
		totalP95 += op.P95Ms

		if op.EstimatedCostUSD > 1.0 {
			summary.HighCostOperations++
		}
		if op.P95Ms > 2000 {
			summary.SlowOperations++
		}
	}

	// Calculate averages
	if summary.TotalRequests > 0 {
		summary.AvgTokensPerRequest = summary.TotalTokens / summary.TotalRequests
		summary.AvgCostPerRequestUSD = roundFloat(summary.TotalCostUSD/float64(summary.TotalRequests), 3)
	}

	if len(operations) > 0 {
		summary.AvgP95LatencyMs = totalP95 / int64(len(operations))
	}

	// Calculate error rate from models
	var totalErrors int64
	var totalModelRequests int64
	for _, model := range models {
		totalErrors += model.ErrorCount
		totalModelRequests += model.TotalRequests
	}

	if totalModelRequests > 0 {
		summary.ErrorRatePercent = roundFloat((float64(totalErrors)/float64(totalModelRequests))*100, 2)
	}

	summary.TotalCostUSD = roundFloat(summary.TotalCostUSD, 2)

	return summary
}

func (c *LLMCommand) generateRecommendations(summary *struct {
	TotalRequests         int64   `json:"total_requests"`
	TotalTokens           int64   `json:"total_tokens"`
	TotalPromptTokens     int64   `json:"total_prompt_tokens"`
	TotalCompletionTokens int64   `json:"total_completion_tokens"`
	AvgTokensPerRequest   int64   `json:"avg_tokens_per_request"`
	TotalCostUSD          float64 `json:"total_cost_usd"`
	AvgCostPerRequestUSD  float64 `json:"avg_cost_per_request_usd"`
	AvgP95LatencyMs       int64   `json:"avg_p95_latency_ms"`
	ErrorRatePercent      float64 `json:"error_rate_percent"`
	HighCostOperations    int     `json:"high_cost_operations_count"`
	SlowOperations        int     `json:"slow_operations_count"`
}) []LLMRecommendation {
	var recommendations []LLMRecommendation

	// High token usage
	if summary.AvgTokensPerRequest > 4000 {
		recommendations = append(recommendations, LLMRecommendation{
			Type:     "high_token_usage",
			Priority: "high",
			Message:  fmt.Sprintf("Average tokens per request is %d. Consider prompt optimization or response truncation.", summary.AvgTokensPerRequest),
		})
	}

	// High latency
	if summary.AvgP95LatencyMs > 3000 {
		recommendations = append(recommendations, LLMRecommendation{
			Type:     "high_latency",
			Priority: "medium",
			Message:  fmt.Sprintf("Average P95 latency is %dms. Consider using faster models for non-critical operations.", summary.AvgP95LatencyMs),
		})
	}

	// High cost
	if summary.TotalCostUSD > 100 {
		recommendations = append(recommendations, LLMRecommendation{
			Type:     "high_cost",
			Priority: "high",
			Message:  fmt.Sprintf("Total cost is $%.2f. Review if all operations require premium models.", summary.TotalCostUSD),
		})
	}

	// High error rate
	if summary.ErrorRatePercent > 5 {
		recommendations = append(recommendations, LLMRecommendation{
			Type:     "high_error_rate",
			Priority: "critical",
			Message:  fmt.Sprintf("Error rate is %.1f%%. Investigate failed requests to avoid wasted token costs.", summary.ErrorRatePercent),
		})
	}

	// Slow operations
	if summary.SlowOperations > 0 {
		recommendations = append(recommendations, LLMRecommendation{
			Type:     "slow_operations",
			Priority: "medium",
			Message:  fmt.Sprintf("%d slow operations detected (P95 > 2000ms). Review and optimize.", summary.SlowOperations),
		})
	}

	return recommendations
}

func (c *LLMCommand) printFormatted(serviceName, duration string, summary *struct {
	TotalRequests         int64   `json:"total_requests"`
	TotalTokens           int64   `json:"total_tokens"`
	TotalPromptTokens     int64   `json:"total_prompt_tokens"`
	TotalCompletionTokens int64   `json:"total_completion_tokens"`
	AvgTokensPerRequest   int64   `json:"avg_tokens_per_request"`
	TotalCostUSD          float64 `json:"total_cost_usd"`
	AvgCostPerRequestUSD  float64 `json:"avg_cost_per_request_usd"`
	AvgP95LatencyMs       int64   `json:"avg_p95_latency_ms"`
	ErrorRatePercent      float64 `json:"error_rate_percent"`
	HighCostOperations    int     `json:"high_cost_operations_count"`
	SlowOperations        int     `json:"slow_operations_count"`
}, operations []LLMOperation, recommendations []LLMRecommendation) {
	fmt.Printf("LLM Observability Analysis: %s\n", serviceName)
	fmt.Printf("Duration: %s\n", duration)
	if c.model != "" {
		fmt.Printf("Model Filter: %s\n", c.model)
	}
	fmt.Println()

	fmt.Println("Token Usage:")
	fmt.Printf("  Total tokens: %s\n", formatNumber(summary.TotalTokens))
	fmt.Printf("  Prompt tokens: %s\n", formatNumber(summary.TotalPromptTokens))
	fmt.Printf("  Completion tokens: %s\n", formatNumber(summary.TotalCompletionTokens))
	fmt.Printf("  Avg per request: %s\n", formatNumber(summary.AvgTokensPerRequest))
	fmt.Println()

	fmt.Println("Cost Analysis:")
	fmt.Printf("  Total cost: $%.2f\n", summary.TotalCostUSD)
	fmt.Printf("  Avg per request: $%.3f\n", summary.AvgCostPerRequestUSD)
	if summary.HighCostOperations > 0 {
		fmt.Printf("  High-cost operations: %d\n", summary.HighCostOperations)
	}
	fmt.Println()

	fmt.Println("Performance:")
	fmt.Printf("  Total requests: %s\n", formatNumber(summary.TotalRequests))
	fmt.Printf("  Avg P95 latency: %dms\n", summary.AvgP95LatencyMs)
	if summary.SlowOperations > 0 {
		fmt.Printf("  Slow operations: %d\n", summary.SlowOperations)
	}
	if summary.ErrorRatePercent > 0 {
		fmt.Printf("  Error rate: %.2f%%\n", summary.ErrorRatePercent)
	}
	fmt.Println()

	// Show top operations by cost
	if len(operations) > 0 {
		fmt.Println("Top Operations by Cost:")
		// Sort by cost
		sortedOps := make([]LLMOperation, len(operations))
		copy(sortedOps, operations)
		sort.Slice(sortedOps, func(i, j int) bool {
			return sortedOps[i].EstimatedCostUSD > sortedOps[j].EstimatedCostUSD
		})

		displayCount := 5
		if len(sortedOps) < displayCount {
			displayCount = len(sortedOps)
		}

		for i := 0; i < displayCount; i++ {
			op := sortedOps[i]
			fmt.Printf("  %d. %s\n", i+1, op.Operation)
			fmt.Printf("     Cost: $%.2f | Tokens: %s | Requests: %s\n",
				op.EstimatedCostUSD,
				formatNumber(op.TotalTokens),
				formatNumber(op.RequestCount))
		}
		fmt.Println()
	}

	// Show recommendations
	if len(recommendations) > 0 {
		fmt.Println("Optimization Recommendations:")
		for _, rec := range recommendations {
			fmt.Printf("  [%s] %s\n", strings.ToUpper(rec.Priority), rec.Message)
		}
	} else {
		fmt.Println("All metrics within acceptable ranges")
	}
}

func (c *LLMCommand) Help() {
	fmt.Println("Usage: dd llm [options]")
	fmt.Println()
	fmt.Println("Query Datadog LLM observability for GenAI applications.")
	fmt.Println("Analyzes token usage, costs, latency, and error rates for LLM operations.")
	fmt.Println()
	fmt.Println("Options:")
	c.flags.PrintDefaults()
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  dd llm")
	fmt.Println("  dd llm --service my-ai-service")
	fmt.Println("  dd llm --duration 24h --model gpt-4")
	fmt.Println("  dd llm --json")
	fmt.Println("  dd llm --duration 7d --limit 50")
}
