package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/datadog/skill/internal/client"
	"github.com/datadog/skill/internal/context"
	"github.com/datadog/skill/internal/observability"
)

type APMCommand struct {
	flags    *flag.FlagSet
	service  string
	duration string
	status   string
	limit    int
	jsonOut  bool
}

// APMResponse represents the response from Datadog APM API
type APMResponse struct {
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

// EndpointStats represents statistics for a single endpoint
type EndpointStats struct {
	ResourceName string `json:"resource_name"`
	RequestCount int64  `json:"request_count"`
	P50Ms        int64  `json:"p50_ms"`
	P95Ms        int64  `json:"p95_ms"`
	P99Ms        int64  `json:"p99_ms"`
}

// APMOutput represents the structured output
type APMOutput struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Duration  string `json:"duration"`
	Summary   *struct {
		TotalEndpoints      int   `json:"total_endpoints"`
		TotalRequests       int64 `json:"total_requests"`
		AvgP95Ms            int64 `json:"avg_p95_ms"`
		SlowEndpointsCount  int   `json:"slow_endpoints_count"`
	} `json:"summary,omitempty"`
	Endpoints []EndpointStats `json:"endpoints"`
}

func NewAPMCommand() *APMCommand {
	cmd := &APMCommand{
		flags: flag.NewFlagSet("apm", flag.ExitOnError),
	}

	cmd.flags.StringVar(&cmd.service, "service", "", "Service name (auto-detected if not provided)")
	cmd.flags.StringVar(&cmd.duration, "duration", "1h", "Time range: 1h, 24h, 7d (default: 1h)")
	cmd.flags.StringVar(&cmd.status, "status", "all", "Filter by status: error, ok, all (default: all)")
	cmd.flags.IntVar(&cmd.limit, "limit", 20, "Max endpoints to return (default: 20)")
	cmd.flags.BoolVar(&cmd.jsonOut, "json", false, "Output as JSON")

	return cmd
}

func (c *APMCommand) Name() string {
	return "apm"
}

func (c *APMCommand) Description() string {
	return "Query Datadog APM for performance analysis"
}

func (c *APMCommand) Run(args []string) error {
	// Initialize observability
	obs, err := observability.Init("query-apm", "production")
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

	obs.LogInfo(fmt.Sprintf("Querying APM for service: %s", serviceName))

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

	// Query APM
	span = obs.StartSpan("query_apm")
	obs.GetTracer().SetTag(span, "service", serviceName)
	obs.GetTracer().SetTag(span, "duration", c.duration)

	start := time.Now()
	filter := ""
	if c.status != "all" {
		filter = fmt.Sprintf("status:%s", c.status)
	}

	responseData, err := ddClient.QueryAPM(serviceName, fromTime, toTime, filter)
	apiDuration := time.Since(start).Milliseconds()
	obs.FinishSpan(span)

	if err != nil {
		obs.RecordAPICall("/api/v2/spans/analytics/aggregate", "POST", 500, float64(apiDuration), err)
		obs.LogError(fmt.Sprintf("API call failed: %s", err.Error()))

		// Check for known API validation error (400)
		errMsg := err.Error()
		if strings.Contains(errMsg, "status 400") || strings.Contains(errMsg, "validation") {
			return fmt.Errorf("failed to query APM: %w\n\n"+
				"Known Issue: APM aggregate queries have API format issues.\n"+
				"Workarounds:\n"+
				"  1. Use Datadog web UI for APM queries\n"+
				"  2. Use 'dd logs' for application logs\n"+
				"  3. Use 'dd metrics --query \"trace.*\"' for APM metrics\n"+
				"See KNOWN-ISSUES.md for details", err)
		}

		return fmt.Errorf("failed to query APM: %w\n\n"+
			"Troubleshooting:\n"+
			"  1. Check DD_API_KEY and DD_APP_KEY are set\n"+
			"  2. Verify service name with: dd context\n"+
			"  3. Check KNOWN-ISSUES.md for known bugs", err)
	}

	obs.RecordAPICall("/api/v2/spans/analytics/aggregate", "POST", 200, float64(apiDuration), nil)

	// Parse results
	span = obs.StartSpan("parse_results")
	endpoints, output, err := c.parseResults(responseData, serviceName, c.duration)
	obs.FinishSpan(span)

	if err != nil {
		obs.LogError(fmt.Sprintf("Failed to parse results: %s", err.Error()))
		return fmt.Errorf("failed to parse results: %w", err)
	}

	// Handle no data case
	if len(endpoints) == 0 {
		obs.LogWarning("No trace data found")
		obs.GetMetrics().Gauge("apm.endpoints", 0, "service:"+serviceName)

		if c.jsonOut {
			jsonData, _ := json.MarshalIndent(output, "", "  ")
			fmt.Println(string(jsonData))
		} else {
			fmt.Printf("No trace data found for service: %s\n", serviceName)
		}

		return nil
	}

	// Calculate statistics
	span = obs.StartSpan("calculate_stats")
	stats := c.calculateStats(endpoints)
	obs.FinishSpan(span)

	// Record metrics
	obs.GetMetrics().Gauge("apm.endpoints", float64(stats.TotalEndpoints), "service:"+serviceName)
	obs.GetMetrics().Gauge("apm.requests", float64(stats.TotalRequests), "service:"+serviceName)
	obs.GetMetrics().Gauge("apm.slow_endpoints", float64(stats.SlowEndpointsCount), "service:"+serviceName)
	obs.GetMetrics().Gauge("apm.avg_p95_ms", float64(stats.AvgP95Ms), "service:"+serviceName)

	// Set summary in output
	output.Summary = stats

	// Output
	if c.jsonOut {
		jsonData, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	} else {
		c.printFormatted(serviceName, c.duration, stats, endpoints)
	}

	obs.LogInfo(fmt.Sprintf("Query completed: %d endpoints", stats.TotalEndpoints))
	return nil
}

func (c *APMCommand) parseDuration(duration string) (time.Time, time.Time, error) {
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

func (c *APMCommand) parseResults(data []byte, serviceName, duration string) ([]EndpointStats, *APMOutput, error) {
	var response APMResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	output := &APMOutput{
		Status:    "ok",
		Service:   serviceName,
		Duration:  duration,
		Endpoints: []EndpointStats{},
	}

	// Check if we have data
	if len(response.Data) == 0 {
		output.Status = "no_data"
		return []EndpointStats{}, output, nil
	}

	endpoints := make([]EndpointStats, 0, len(response.Data))

	for _, bucket := range response.Data {
		resourceName := "unknown"
		if resName, ok := bucket.Attributes.By["resource_name"].(string); ok {
			resourceName = resName
		}

		computes := bucket.Attributes.Compute

		// Extract metrics - handle both float64 and int
		var requestCount int64
		var p50Ns, p95Ns, p99Ns int64

		if val, ok := computes["c0"].(float64); ok {
			requestCount = int64(val)
		}
		if val, ok := computes["c1"].(float64); ok {
			p50Ns = int64(val)
		}
		if val, ok := computes["c2"].(float64); ok {
			p95Ns = int64(val)
		}
		if val, ok := computes["c3"].(float64); ok {
			p99Ns = int64(val)
		}

		endpoint := EndpointStats{
			ResourceName: resourceName,
			RequestCount: requestCount,
			P50Ms:        p50Ns / 1_000_000, // Convert nanoseconds to milliseconds
			P95Ms:        p95Ns / 1_000_000,
			P99Ms:        p99Ns / 1_000_000,
		}

		endpoints = append(endpoints, endpoint)
	}

	// Apply limit
	if len(endpoints) > c.limit {
		endpoints = endpoints[:c.limit]
	}

	output.Endpoints = endpoints
	return endpoints, output, nil
}

func (c *APMCommand) calculateStats(endpoints []EndpointStats) *struct {
	TotalEndpoints     int   `json:"total_endpoints"`
	TotalRequests      int64 `json:"total_requests"`
	AvgP95Ms           int64 `json:"avg_p95_ms"`
	SlowEndpointsCount int   `json:"slow_endpoints_count"`
} {
	stats := &struct {
		TotalEndpoints     int   `json:"total_endpoints"`
		TotalRequests      int64 `json:"total_requests"`
		AvgP95Ms           int64 `json:"avg_p95_ms"`
		SlowEndpointsCount int   `json:"slow_endpoints_count"`
	}{
		TotalEndpoints: len(endpoints),
	}

	var totalP95 int64
	for _, endpoint := range endpoints {
		stats.TotalRequests += endpoint.RequestCount
		totalP95 += endpoint.P95Ms

		if endpoint.P95Ms > 500 {
			stats.SlowEndpointsCount++
		}
	}

	if stats.TotalEndpoints > 0 {
		stats.AvgP95Ms = totalP95 / int64(stats.TotalEndpoints)
	}

	return stats
}

func (c *APMCommand) printFormatted(serviceName, duration string, stats *struct {
	TotalEndpoints     int   `json:"total_endpoints"`
	TotalRequests      int64 `json:"total_requests"`
	AvgP95Ms           int64 `json:"avg_p95_ms"`
	SlowEndpointsCount int   `json:"slow_endpoints_count"`
}, endpoints []EndpointStats) {
	fmt.Printf("APM Analysis: %s\n", serviceName)
	fmt.Printf("Duration: %s\n", duration)
	fmt.Println()
	fmt.Printf("%d endpoints analyzed\n", stats.TotalEndpoints)
	fmt.Printf("%d requests\n", stats.TotalRequests)
	fmt.Printf("Average P95: %dms\n", stats.AvgP95Ms)

	if stats.SlowEndpointsCount > 0 {
		fmt.Println()
		fmt.Printf("%d slow endpoints (P95 > 500ms):\n", stats.SlowEndpointsCount)

		count := 0
		for _, endpoint := range endpoints {
			if endpoint.P95Ms > 500 && count < 5 {
				fmt.Printf("  - %s\n", endpoint.ResourceName)
				fmt.Printf("    P95: %dms | %d requests\n", endpoint.P95Ms, endpoint.RequestCount)
				count++
			}
		}
	} else {
		fmt.Println()
		fmt.Println("All endpoints performing well")
	}
}

func (c *APMCommand) Help() {
	fmt.Println("Usage: dd apm [options]")
	fmt.Println()
	fmt.Println("Query Datadog APM for performance analysis.")
	fmt.Println("Finds slow endpoints, errors, and performance bottlenecks.")
	fmt.Println()
	fmt.Println("Options:")
	c.flags.PrintDefaults()
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  dd apm")
	fmt.Println("  dd apm --service my-service")
	fmt.Println("  dd apm --duration 24h --status error")
	fmt.Println("  dd apm --json")
	fmt.Println("  dd apm --duration 7d --limit 50")
}
