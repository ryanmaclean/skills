package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/datadog/skill/internal/client"
)

// SpansCommand queries APM spans for trace analysis
type SpansCommand struct {
	flags     *flag.FlagSet
	action    string
	service   string
	operation string
	resource  string
	env       string
	status    string
	from      string
	to        string
	groupBy   string
	limit     int
	jsonOut   bool
}

// SpanData represents a single span
type SpanData struct {
	TraceID   string                 `json:"trace_id"`
	SpanID    string                 `json:"span_id"`
	Service   string                 `json:"service"`
	Operation string                 `json:"operation"`
	Resource  string                 `json:"resource"`
	Duration  int64                  `json:"duration_ns"`
	Status    string                 `json:"status"`
	Error     int                    `json:"error"`
	StartTime int64                  `json:"start"`
	Tags      map[string]interface{} `json:"tags,omitempty"`
}

// SpanAnalytics represents aggregated span analytics
type SpanAnalytics struct {
	Service     string  `json:"service"`
	Operation   string  `json:"operation"`
	TotalSpans  int64   `json:"total_spans"`
	ErrorRate   float64 `json:"error_rate"`
	P50         float64 `json:"p50_ms"`
	P75         float64 `json:"p75_ms"`
	P90         float64 `json:"p90_ms"`
	P95         float64 `json:"p95_ms"`
	P99         float64 `json:"p99_ms"`
	AvgDuration float64 `json:"avg_duration_ms"`
	Throughput  float64 `json:"throughput_per_sec"`
}

// SpanBreakdown represents span metrics grouped by dimension
type SpanBreakdown struct {
	Dimension string        `json:"dimension"`
	Groups    []BreakdownGroup `json:"groups"`
}

// BreakdownGroup represents a single group in breakdown
type BreakdownGroup struct {
	Name       string  `json:"name"`
	Count      int64   `json:"count"`
	ErrorCount int64   `json:"error_count"`
	ErrorRate  float64 `json:"error_rate"`
	AvgLatency float64 `json:"avg_latency_ms"`
}

// SpansResponse represents the spans command response
type SpansResponse struct {
	Status    string        `json:"status"`
	Spans     []SpanData    `json:"spans,omitempty"`
	Analytics *SpanAnalytics `json:"analytics,omitempty"`
	Breakdown *SpanBreakdown `json:"breakdown,omitempty"`
}

// NewSpansCommand creates a new spans command instance
func NewSpansCommand() *SpansCommand {
	cmd := &SpansCommand{
		flags: flag.NewFlagSet("spans", flag.ExitOnError),
	}

	cmd.flags.StringVar(&cmd.action, "action", "analytics", "Action to perform (search, analytics, list, breakdown, latency, errors)")
	cmd.flags.StringVar(&cmd.service, "service", "", "Service name (required)")
	cmd.flags.StringVar(&cmd.operation, "operation", "", "Operation name")
	cmd.flags.StringVar(&cmd.resource, "resource", "", "Resource name")
	cmd.flags.StringVar(&cmd.env, "env", "prod", "Environment")
	cmd.flags.StringVar(&cmd.status, "status", "", "Status filter (ok, error)")
	cmd.flags.StringVar(&cmd.from, "from", "1h", "Start time (relative like '1h', '24h' or RFC3339)")
	cmd.flags.StringVar(&cmd.to, "to", "now", "End time (relative or RFC3339)")
	cmd.flags.StringVar(&cmd.groupBy, "group-by", "", "Group by dimension (resource, status, error.type)")
	cmd.flags.IntVar(&cmd.limit, "limit", 100, "Maximum spans to return")
	cmd.flags.BoolVar(&cmd.jsonOut, "json", false, "Output as JSON")

	return cmd
}

func (c *SpansCommand) Name() string {
	return "spans"
}

func (c *SpansCommand) Description() string {
	return "Query APM spans for trace analysis"
}

func (c *SpansCommand) Run(args []string) error {
	if err := c.flags.Parse(args); err != nil {
		return err
	}

	// Check for --help flag
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			c.Help()
			return nil
		}
	}

	if c.service == "" {
		return fmt.Errorf("--service is required")
	}

	ddClient, err := client.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Datadog client: %w", err)
	}

	switch c.action {
	case "search":
		return c.searchSpans(ddClient)
	case "analytics":
		return c.spanAnalytics(ddClient)
	case "list":
		return c.listSpans(ddClient)
	case "breakdown":
		return c.spanBreakdown(ddClient)
	case "latency":
		return c.latencyAnalysis(ddClient)
	case "errors":
		return c.errorAnalysis(ddClient)
	default:
		return fmt.Errorf("unknown action: %s (valid: search, analytics, list, breakdown, latency, errors)", c.action)
	}
}

func (c *SpansCommand) searchSpans(ddClient *client.Client) error {
	fromTime, toTime, err := c.parseTimeRange()
	if err != nil {
		return err
	}

	// Build search query
	query := c.buildSearchQuery()

	payload := map[string]interface{}{
		"filter": map[string]interface{}{
			"query": query,
			"from":  fromTime.Format(time.RFC3339),
			"to":    toTime.Format(time.RFC3339),
		},
		"page": map[string]interface{}{
			"limit": c.limit,
		},
		"sort": "-duration",
	}

	data, err := ddClient.SearchSpans(payload)
	if err != nil {
		return fmt.Errorf("failed to search spans: %w", err)
	}

	// Parse response (simplified structure)
	var apiResp struct {
		Data []struct {
			Attributes SpanData `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &apiResp); err != nil {
		// Try alternate format
		return c.displaySearchResults(data, fromTime, toTime)
	}

	spans := make([]SpanData, len(apiResp.Data))
	for i, item := range apiResp.Data {
		spans[i] = item.Attributes
	}

	return c.displaySpans(spans, "search")
}

func (c *SpansCommand) spanAnalytics(ddClient *client.Client) error {
	fromTime, toTime, err := c.parseTimeRange()
	if err != nil {
		return err
	}

	// Build aggregation query
	query := c.buildSearchQuery()

	payload := map[string]interface{}{
		"filter": map[string]interface{}{
			"query": query,
			"from":  fromTime.Format(time.RFC3339),
			"to":    toTime.Format(time.RFC3339),
		},
		"compute": []map[string]interface{}{
			{
				"aggregation": "count",
				"metric":      "*",
			},
			{
				"aggregation": "avg",
				"metric":      "duration",
			},
			{
				"aggregation": "pc50",
				"metric":      "duration",
			},
			{
				"aggregation": "pc75",
				"metric":      "duration",
			},
			{
				"aggregation": "pc90",
				"metric":      "duration",
			},
			{
				"aggregation": "pc95",
				"metric":      "duration",
			},
			{
				"aggregation": "pc99",
				"metric":      "duration",
			},
		},
	}

	data, err := ddClient.AggregateSpans(payload)
	if err != nil {
		return fmt.Errorf("failed to aggregate spans: %w", err)
	}

	return c.displayAnalytics(data, fromTime, toTime)
}

func (c *SpansCommand) listSpans(ddClient *client.Client) error {
	// Similar to search but with different display
	return c.searchSpans(ddClient)
}

func (c *SpansCommand) spanBreakdown(ddClient *client.Client) error {
	if c.groupBy == "" {
		c.groupBy = "resource"
	}

	fromTime, toTime, err := c.parseTimeRange()
	if err != nil {
		return err
	}

	query := c.buildSearchQuery()

	payload := map[string]interface{}{
		"filter": map[string]interface{}{
			"query": query,
			"from":  fromTime.Format(time.RFC3339),
			"to":    toTime.Format(time.RFC3339),
		},
		"compute": []map[string]interface{}{
			{
				"aggregation": "count",
				"metric":      "*",
			},
			{
				"aggregation": "avg",
				"metric":      "duration",
			},
		},
		"group_by": []map[string]interface{}{
			{
				"facet": c.groupBy,
				"limit": 20,
				"sort": map[string]interface{}{
					"aggregation": "count",
					"order":       "desc",
				},
			},
		},
	}

	data, err := ddClient.AggregateSpans(payload)
	if err != nil {
		return fmt.Errorf("failed to aggregate spans: %w", err)
	}

	return c.displayBreakdown(data, c.groupBy, fromTime, toTime)
}

func (c *SpansCommand) latencyAnalysis(ddClient *client.Client) error {
	// Use analytics with focus on latency percentiles
	c.action = "analytics"
	return c.spanAnalytics(ddClient)
}

func (c *SpansCommand) errorAnalysis(ddClient *client.Client) error {
	// Set status filter to errors
	originalStatus := c.status
	c.status = "error"

	err := c.spanBreakdown(ddClient)

	c.status = originalStatus
	return err
}

func (c *SpansCommand) buildSearchQuery() string {
	var parts []string

	parts = append(parts, fmt.Sprintf("service:%s", c.service))

	if c.env != "" {
		parts = append(parts, fmt.Sprintf("env:%s", c.env))
	}

	if c.operation != "" {
		parts = append(parts, fmt.Sprintf("operation_name:%s", c.operation))
	}

	if c.resource != "" {
		parts = append(parts, fmt.Sprintf("resource_name:%s", c.resource))
	}

	if c.status != "" {
		if c.status == "error" {
			parts = append(parts, "status:error")
		} else {
			parts = append(parts, "status:ok")
		}
	}

	return strings.Join(parts, " ")
}

func (c *SpansCommand) parseTimeRange() (time.Time, time.Time, error) {
	var fromTime, toTime time.Time
	var err error

	// Parse from time
	if c.from == "" {
		fromTime = time.Now().Add(-1 * time.Hour)
	} else {
		fromTime, err = c.parseTime(c.from)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid from time: %w", err)
		}
	}

	// Parse to time
	if c.to == "" || c.to == "now" {
		toTime = time.Now()
	} else {
		toTime, err = c.parseTime(c.to)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid to time: %w", err)
		}
	}

	return fromTime, toTime, nil
}

func (c *SpansCommand) parseTime(timeStr string) (time.Time, error) {
	// Try relative time first (e.g., "1h", "24h", "7d")
	if strings.HasSuffix(timeStr, "h") {
		hours := strings.TrimSuffix(timeStr, "h")
		var h int
		fmt.Sscanf(hours, "%d", &h)
		return time.Now().Add(-time.Duration(h) * time.Hour), nil
	}
	if strings.HasSuffix(timeStr, "d") {
		days := strings.TrimSuffix(timeStr, "d")
		var d int
		fmt.Sscanf(days, "%d", &d)
		return time.Now().Add(-time.Duration(d) * 24 * time.Hour), nil
	}

	// Try RFC3339
	if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("invalid time format: %s (use '1h', '24h', '7d' or RFC3339)", timeStr)
}

func (c *SpansCommand) displaySpans(spans []SpanData, displayType string) error {
	if c.jsonOut {
		response := SpansResponse{
			Status: "success",
			Spans:  spans,
		}
		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}

	// Human-readable output
	fmt.Printf("Spans: %s (%d results)\n", c.service, len(spans))
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println()

	if len(spans) == 0 {
		fmt.Println("No spans found matching the criteria.")
		return nil
	}

	for i, span := range spans {
		if i > 0 {
			fmt.Println()
		}

		durationMs := float64(span.Duration) / 1000000.0

		fmt.Printf("Trace ID: %s\n", span.TraceID)
		fmt.Printf("Span ID: %s\n", span.SpanID)
		fmt.Printf("Service: %s\n", span.Service)
		if span.Operation != "" {
			fmt.Printf("Operation: %s\n", span.Operation)
		}
		if span.Resource != "" {
			fmt.Printf("Resource: %s\n", span.Resource)
		}
		fmt.Printf("Duration: %.2fms\n", durationMs)
		fmt.Printf("Status: %s\n", span.Status)
		if span.Error > 0 {
			fmt.Println("Error: Yes")
		}

		if i >= 9 {
			remaining := len(spans) - i - 1
			if remaining > 0 {
				fmt.Printf("\n... and %d more spans (use --limit to see more)\n", remaining)
			}
			break
		}
	}

	return nil
}

func (c *SpansCommand) displaySearchResults(data []byte, fromTime, toTime time.Time) error {
	// Display raw analytics when search structure is complex
	fmt.Printf("Span Search Results: %s\n", c.service)
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println()
	fmt.Printf("Time Range: %s to %s\n", fromTime.Format("2006-01-02 15:04:05"), toTime.Format("2006-01-02 15:04:05"))
	fmt.Println()

	if c.jsonOut {
		fmt.Println(string(data))
		return nil
	}

	fmt.Println("Note: Use --json for full response details")
	return nil
}

func (c *SpansCommand) displayAnalytics(data []byte, fromTime, toTime time.Time) error {
	// Parse aggregation response
	var apiResp struct {
		Data struct {
			Attributes struct {
				Aggregates []struct {
					Value float64 `json:"value"`
				} `json:"aggregates"`
			} `json:"attributes"`
		} `json:"data"`
	}

	// For display purposes, create synthetic analytics
	analytics := SpanAnalytics{
		Service:   c.service,
		Operation: c.operation,
	}

	if err := json.Unmarshal(data, &apiResp); err == nil {
		if len(apiResp.Data.Attributes.Aggregates) >= 7 {
			analytics.TotalSpans = int64(apiResp.Data.Attributes.Aggregates[0].Value)
			analytics.AvgDuration = apiResp.Data.Attributes.Aggregates[1].Value / 1000000.0
			analytics.P50 = apiResp.Data.Attributes.Aggregates[2].Value / 1000000.0
			analytics.P75 = apiResp.Data.Attributes.Aggregates[3].Value / 1000000.0
			analytics.P90 = apiResp.Data.Attributes.Aggregates[4].Value / 1000000.0
			analytics.P95 = apiResp.Data.Attributes.Aggregates[5].Value / 1000000.0
			analytics.P99 = apiResp.Data.Attributes.Aggregates[6].Value / 1000000.0

			duration := toTime.Sub(fromTime).Seconds()
			if duration > 0 {
				analytics.Throughput = float64(analytics.TotalSpans) / duration
			}
		}
	}

	if c.jsonOut {
		response := SpansResponse{
			Status:    "success",
			Analytics: &analytics,
		}
		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}

	// Human-readable output
	fmt.Printf("Span Analytics: %s\n", c.service)
	if c.operation != "" {
		fmt.Printf("Operation: %s\n", c.operation)
	}
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println()
	fmt.Printf("Time Range: %s to %s\n", fromTime.Format("2006-01-02 15:04:05"), toTime.Format("2006-01-02 15:04:05"))
	fmt.Println()

	if analytics.TotalSpans > 0 {
		fmt.Println("Latency Distribution:")
		fmt.Printf("  p50: %.2fms\n", analytics.P50)
		fmt.Printf("  p75: %.2fms\n", analytics.P75)
		fmt.Printf("  p90: %.2fms\n", analytics.P90)
		fmt.Printf("  p95: %.2fms\n", analytics.P95)
		fmt.Printf("  p99: %.2fms\n", analytics.P99)
		fmt.Println()
		fmt.Printf("Total Spans: %d\n", analytics.TotalSpans)
		if analytics.ErrorRate > 0 {
			fmt.Printf("Error Rate: %.2f%%\n", analytics.ErrorRate)
		}
		fmt.Printf("Throughput: %.2f spans/sec\n", analytics.Throughput)
	} else {
		fmt.Println("No span data available for the specified time range.")
		fmt.Println()
		fmt.Println("Note: Use --json to see raw API response")
	}

	return nil
}

func (c *SpansCommand) displayBreakdown(data []byte, dimension string, fromTime, toTime time.Time) error {
	// Parse grouped response
	var apiResp struct {
		Data struct {
			Attributes struct {
				By []struct {
					Buckets []struct {
						By struct {
							Dimension string `json:"dimension"`
						} `json:"by"`
						Computes struct {
							Count struct {
								Value float64 `json:"value"`
							} `json:"c0"`
							AvgDuration struct {
								Value float64 `json:"value"`
							} `json:"c1"`
						} `json:"computes"`
					} `json:"buckets"`
				} `json:"by"`
			} `json:"attributes"`
		} `json:"data"`
	}

	breakdown := SpanBreakdown{
		Dimension: dimension,
		Groups:    []BreakdownGroup{},
	}

	if err := json.Unmarshal(data, &apiResp); err == nil {
		if len(apiResp.Data.Attributes.By) > 0 {
			for _, bucket := range apiResp.Data.Attributes.By[0].Buckets {
				group := BreakdownGroup{
					Name:       bucket.By.Dimension,
					Count:      int64(bucket.Computes.Count.Value),
					AvgLatency: bucket.Computes.AvgDuration.Value / 1000000.0,
				}
				breakdown.Groups = append(breakdown.Groups, group)
			}
		}
	}

	if c.jsonOut {
		response := SpansResponse{
			Status:    "success",
			Breakdown: &breakdown,
		}
		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}

	// Human-readable output
	fmt.Printf("Span Breakdown: %s\n", c.service)
	fmt.Printf("Grouped by: %s\n", dimension)
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println()
	fmt.Printf("Time Range: %s to %s\n", fromTime.Format("2006-01-02 15:04:05"), toTime.Format("2006-01-02 15:04:05"))
	fmt.Println()

	if len(breakdown.Groups) == 0 {
		fmt.Println("No breakdown data available.")
		fmt.Println()
		fmt.Println("Note: Use --json to see raw API response")
		return nil
	}

	// Sort by count descending
	sort.Slice(breakdown.Groups, func(i, j int) bool {
		return breakdown.Groups[i].Count > breakdown.Groups[j].Count
	})

	var totalSpans int64
	for _, group := range breakdown.Groups {
		totalSpans += group.Count
	}

	fmt.Printf("Total Spans: %d\n", totalSpans)
	fmt.Println()

	for i, group := range breakdown.Groups {
		percentage := float64(group.Count) / float64(totalSpans) * 100.0
		fmt.Printf("%d. %s\n", i+1, group.Name)
		fmt.Printf("   Count: %d (%.1f%%)\n", group.Count, percentage)
		fmt.Printf("   Avg Latency: %.2fms\n", group.AvgLatency)
		if group.ErrorRate > 0 {
			fmt.Printf("   Error Rate: %.2f%%\n", group.ErrorRate)
		}
		fmt.Println()

		if i >= 9 {
			remaining := len(breakdown.Groups) - i - 1
			if remaining > 0 {
				fmt.Printf("... and %d more groups\n", remaining)
			}
			break
		}
	}

	return nil
}

func (c *SpansCommand) Help() {
	helpText := `dd spans - Query APM Spans for Trace Analysis

DESCRIPTION:
  Query APM spans for detailed distributed trace analysis, latency investigation,
  and service call patterns. Analyze span-level metrics including percentile
  latencies, error rates, and resource breakdowns.

USAGE:
  dd spans --service <service> --action <action> [options]

ACTIONS:
  search           Search spans with filtering
  analytics        Aggregate span analytics (default)
  list             List recent spans with details
  breakdown        Breakdown spans by dimension
  latency          Analyze latency distribution
  errors           Find spans with errors

EXAMPLES:
  # Analyze span latencies for service (last hour)
  dd spans --service payment-service --action analytics

  # Analyze specific operation
  dd spans --service payment-service \
    --action analytics \
    --operation checkout.process_payment

  # Breakdown by resource
  dd spans --service payment-service \
    --action breakdown \
    --group-by resource

  # Find error spans
  dd spans --service payment-service \
    --action errors \
    --from 24h

  # Search spans with filters
  dd spans --service payment-service \
    --action search \
    --operation checkout.process_payment \
    --status error \
    --from 1h

  # Analyze latency for specific time range
  dd spans --service api-gateway \
    --action latency \
    --from "2026-01-23T10:00:00Z" \
    --to "2026-01-23T11:00:00Z"

  # Get JSON output
  dd spans --service payment-service --action analytics --json

OPTIONS:
  --service         Service name (required)
  --action          Action to perform (search, analytics, list, breakdown, latency, errors)
  --operation       Operation name filter
  --resource        Resource name filter
  --env             Environment (default: prod)
  --status          Status filter (ok, error)
  --from            Start time (default: 1h) - relative like '1h', '24h' or RFC3339
  --to              End time (default: now) - relative or RFC3339
  --group-by        Group by dimension (resource, status, error.type) for breakdown
  --limit           Maximum spans to return (default: 100)
  --json            Output as JSON

TIME FORMATS:
  Relative:  1h (1 hour ago)
             24h (24 hours ago)
             7d (7 days ago)

  Absolute:  2026-01-23T10:00:00Z (RFC3339)

LATENCY PERCENTILES:
  p50:  Median latency (50th percentile)
  p75:  75th percentile latency
  p90:  90th percentile latency
  p95:  95th percentile latency
  p99:  99th percentile latency (tail latency)

GROUP BY DIMENSIONS:
  resource:       Group by resource name (endpoint, query, etc.)
  status:         Group by span status (ok, error)
  error.type:     Group by error type
  http.status_code: Group by HTTP status code
  db.statement:   Group by database query pattern

USE CASES:
  Latency Investigation:
    - Identify slow spans in distributed traces
    - Find latency regressions after deployments
    - Analyze p95/p99 tail latencies
    - Compare latencies across time periods

  Error Analysis:
    - Find error patterns across services
    - Identify specific error types
    - Analyze error rates by resource
    - Correlate errors with latency spikes

  Performance Debugging:
    - Identify slow resources (endpoints, queries)
    - Find bottlenecks in service calls
    - Analyze database query performance
    - Compare operation performance

  Resource Optimization:
    - Find slowest database queries
    - Identify expensive API calls
    - Optimize high-latency operations
    - Reduce error-prone resources

  Service Monitoring:
    - Monitor span throughput
    - Track error rate trends
    - Analyze service call patterns
    - Identify sampling coverage

INTEGRATION WITH OTHER COMMANDS:
  Performance Investigation:
    1. Check health:      dd health --service payment-service
    2. Analyze spans:     dd spans --service payment-service --action analytics
    3. Breakdown:         dd spans --service payment-service --action breakdown
    4. Check deps:        dd service-map --action dependencies --service payment-service

  Error Investigation:
    1. Find errors:       dd spans --service api --action errors --from 1h
    2. Breakdown:         dd spans --service api --action breakdown --status error
    3. Check logs:        dd logs --service api --severity error --since 1h
    4. Post event:        dd events --action post --title "Investigating errors"

  Deployment Analysis:
    1. Post deploy event: dd events --action post --title "Deploy v1.2.3"
    2. Check health:      dd health --service api --since-deploy
    3. Compare latency:   dd spans --service api --action analytics
    4. Check errors:      dd spans --service api --action errors

BEST PRACTICES:
  Query Optimization:
    - Use specific service and operation filters
    - Limit time ranges to reduce query load
    - Use breakdown for aggregated views
    - Sample large result sets

  Latency Analysis:
    - Focus on p95/p99 for tail latency issues
    - Compare percentiles across time periods
    - Analyze slow resources first
    - Consider span sampling effects

  Error Analysis:
    - Group errors by type for patterns
    - Correlate errors with latency
    - Check error rates by resource
    - Link to log analysis

  Performance:
    - Start with analytics for overview
    - Use breakdown to identify hotspots
    - Search for specific slow spans
    - Cross-reference with metrics

NOTES:
  - Spans are sampled; not all traces are captured
  - Latency values are in milliseconds (ms)
  - Duration internally stored in nanoseconds
  - Error rate calculated from span error flag
  - Throughput calculated as spans per second
  - Large result sets may be paginated

SPAN ATTRIBUTES:
  service:       Service that generated the span
  operation:     Operation name (e.g., "http.request", "db.query")
  resource:      Resource name (e.g., "/api/users", "SELECT users")
  duration:      Span duration in nanoseconds
  status:        Span status (ok, error)
  error:         Error flag (0 = no error, 1 = error)
  trace_id:      Distributed trace ID
  span_id:       Unique span ID

For more information: https://docs.datadoghq.com/tracing/
`

	fmt.Println(helpText)
}
