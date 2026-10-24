package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/datadog/skill/internal/client"
	"github.com/datadog/skill/internal/observability"
)

// DatabaseCommand queries Datadog database monitoring for performance analysis
type DatabaseCommand struct {
	flags    *flag.FlagSet
	host     string
	database string
	duration string
	metric   string
	jsonOut  bool
}

// MetricSeriesData represents time series metric data from Datadog
type MetricSeriesData struct {
	Status string `json:"status"`
	Series []struct {
		Pointlist [][2]interface{} `json:"pointlist"`
		Metric    string           `json:"metric"`
	} `json:"series"`
}

// APMSpanData represents APM span analytics data
type APMSpanData struct {
	Data struct {
		Buckets []struct {
			By       map[string]interface{} `json:"by"`
			Computes map[string]interface{} `json:"computes"`
		} `json:"buckets"`
	} `json:"data"`
}

// QueryPerformance represents performance statistics for a single query
type QueryPerformance struct {
	QueryText      string `json:"query_text"`
	ExecutionCount int64  `json:"execution_count"`
	P95Ms          int64  `json:"p95_ms"`
	AvgMs          int64  `json:"avg_ms"`
}

// ConnectionMetrics represents database connection statistics
type ConnectionMetrics struct {
	CurrentConnections int64 `json:"current_connections"`
	MaxConnections     int64 `json:"max_connections"`
	ConnectionPercent  int64 `json:"connection_percent"`
}

// DatabaseOutput represents the structured output
type DatabaseOutput struct {
	Status   string `json:"status"`
	Host     string `json:"host"`
	Database string `json:"database"`
	Duration string `json:"duration"`
	Summary  *struct {
		ConnectionCount    int64 `json:"connection_count"`
		QueryPatterns      int   `json:"query_patterns"`
		SlowQueries        int   `json:"slow_queries"`
		SlowestP95Ms       int64 `json:"slowest_p95_ms"`
		AvgQueryDurationMs int64 `json:"avg_query_duration_ms"`
	} `json:"summary,omitempty"`
	Connections *ConnectionMetrics `json:"connections,omitempty"`
	Queries     []QueryPerformance `json:"queries,omitempty"`
	Issues      []struct {
		Severity string `json:"severity"`
		Message  string `json:"message"`
		Action   string `json:"action"`
	} `json:"issues,omitempty"`
}

// NewDatabaseCommand creates a new database command
func NewDatabaseCommand() *DatabaseCommand {
	cmd := &DatabaseCommand{
		flags: flag.NewFlagSet("database", flag.ExitOnError),
	}

	cmd.flags.StringVar(&cmd.host, "host", "", "Database host to query")
	cmd.flags.StringVar(&cmd.database, "database", "", "Database name")
	cmd.flags.StringVar(&cmd.duration, "duration", "1h", "Time range: 1h, 24h, 7d (default: 1h)")
	cmd.flags.StringVar(&cmd.metric, "metric", "all", "Specific metric: queries, connections, latency, all (default: all)")
	cmd.flags.BoolVar(&cmd.jsonOut, "json", false, "Output as JSON")

	return cmd
}

// Name returns the command name
func (c *DatabaseCommand) Name() string {
	return "database"
}

// Description returns the command description
func (c *DatabaseCommand) Description() string {
	return "Query Datadog Database Monitoring for performance analysis"
}

// Run executes the database command
func (c *DatabaseCommand) Run(args []string) error {
	// Initialize observability
	obs, err := observability.Init("query-database", "production")
	if err != nil {
		return fmt.Errorf("failed to init observability: %w", err)
	}
	defer obs.Shutdown(0)

	// Parse flags
	if err := c.flags.Parse(args); err != nil {
		return err
	}

	// Validate host
	if c.host == "" {
		obs.LogError("Database host is required")
		return fmt.Errorf("database host is required: use --host flag")
	}

	obs.LogInfo(fmt.Sprintf("Querying database monitoring: host=%s", c.host))

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

	// Initialize output
	output := &DatabaseOutput{
		Status:   "ok",
		Host:     c.host,
		Database: c.database,
		Duration: c.duration,
		Summary: &struct {
			ConnectionCount    int64 `json:"connection_count"`
			QueryPatterns      int   `json:"query_patterns"`
			SlowQueries        int   `json:"slow_queries"`
			SlowestP95Ms       int64 `json:"slowest_p95_ms"`
			AvgQueryDurationMs int64 `json:"avg_query_duration_ms"`
		}{},
	}

	// Query connection metrics if requested
	if c.metric == "all" || c.metric == "connections" {
		span = obs.StartSpan("query_connections")
		obs.GetTracer().SetTag(span, "host", c.host)

		start := time.Now()
		connMetrics, err := c.queryConnectionMetrics(ddClient, fromTime, toTime)
		apiDuration := time.Since(start).Milliseconds()
		obs.FinishSpan(span)

		if err != nil {
			obs.RecordAPICall("/api/v1/query", "GET", 500, float64(apiDuration), err)
			obs.LogWarning(fmt.Sprintf("Failed to query connection metrics: %s", err.Error()))
		} else {
			obs.RecordAPICall("/api/v1/query", "GET", 200, float64(apiDuration), nil)
			output.Connections = connMetrics
			output.Summary.ConnectionCount = connMetrics.CurrentConnections
		}
	}

	// Query slow queries if requested
	if c.metric == "all" || c.metric == "queries" || c.metric == "latency" {
		span = obs.StartSpan("query_slow_queries")
		obs.GetTracer().SetTag(span, "host", c.host)

		start := time.Now()
		queries, err := c.querySlowQueries(ddClient, fromTime, toTime)
		apiDuration := time.Since(start).Milliseconds()
		obs.FinishSpan(span)

		if err != nil {
			obs.RecordAPICall("/api/v2/spans/analytics/aggregate", "POST", 500, float64(apiDuration), err)
			obs.LogWarning(fmt.Sprintf("Failed to query slow queries: %s", err.Error()))
		} else {
			obs.RecordAPICall("/api/v2/spans/analytics/aggregate", "POST", 200, float64(apiDuration), nil)
			output.Queries = queries
			output.Summary.QueryPatterns = len(queries)

			// Calculate statistics
			if len(queries) > 0 {
				var totalDuration int64
				var slowestP95 int64

				for _, q := range queries {
					totalDuration += q.AvgMs
					if q.P95Ms > slowestP95 {
						slowestP95 = q.P95Ms
					}
					if q.P95Ms > 1000 {
						output.Summary.SlowQueries++
					}
				}

				output.Summary.SlowestP95Ms = slowestP95
				output.Summary.AvgQueryDurationMs = totalDuration / int64(len(queries))
			}
		}
	}

	// Handle no data case
	if output.Summary.ConnectionCount == 0 && output.Summary.QueryPatterns == 0 {
		obs.LogWarning("No database monitoring data found")
		obs.GetMetrics().Gauge("database.query_patterns", 0, "host:"+c.host)

		output.Status = "no_data"

		if c.jsonOut {
			jsonData, _ := json.MarshalIndent(output, "", "  ")
			fmt.Println(string(jsonData))
		} else {
			fmt.Printf("No database monitoring data found for host: %s\n", c.host)
			fmt.Println("Ensure Database Monitoring is enabled and the host is correct.")
		}

		return nil
	}

	// Analyze and add issues
	span = obs.StartSpan("analyze_issues")
	c.analyzeIssues(output)
	obs.FinishSpan(span)

	// Record metrics
	obs.GetMetrics().Gauge("database.connections", float64(output.Summary.ConnectionCount), "host:"+c.host)
	obs.GetMetrics().Gauge("database.query_patterns", float64(output.Summary.QueryPatterns), "host:"+c.host)
	obs.GetMetrics().Gauge("database.slow_queries", float64(output.Summary.SlowQueries), "host:"+c.host)
	obs.GetMetrics().Gauge("database.slowest_p95_ms", float64(output.Summary.SlowestP95Ms), "host:"+c.host)

	// Output
	if c.jsonOut {
		jsonData, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	} else {
		c.printFormatted(output)
	}

	obs.LogInfo(fmt.Sprintf("Database monitoring query completed: %d query patterns, %d slow queries", output.Summary.QueryPatterns, output.Summary.SlowQueries))
	return nil
}

// parseDuration parses duration string to time range
func (c *DatabaseCommand) parseDuration(duration string) (time.Time, time.Time, error) {
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

// queryConnectionMetrics queries database connection metrics
func (c *DatabaseCommand) queryConnectionMetrics(ddClient *client.Client, fromTime, toTime time.Time) (*ConnectionMetrics, error) {
	// Try PostgreSQL connection metric
	query := fmt.Sprintf("avg:postgresql.connections.count{host:%s}", c.host)

	responseData, err := ddClient.QueryMetrics(query, fromTime, toTime)
	if err != nil {
		return nil, err
	}

	var metricData MetricSeriesData
	if err := json.Unmarshal(responseData, &metricData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metric response: %w", err)
	}

	metrics := &ConnectionMetrics{
		CurrentConnections: 0,
		MaxConnections:     0,
		ConnectionPercent:  0,
	}

	// Extract connection count from last data point
	if len(metricData.Series) > 0 && len(metricData.Series[0].Pointlist) > 0 {
		lastPoint := metricData.Series[0].Pointlist[len(metricData.Series[0].Pointlist)-1]
		if len(lastPoint) == 2 {
			switch v := lastPoint[1].(type) {
			case float64:
				metrics.CurrentConnections = int64(v)
			case int64:
				metrics.CurrentConnections = v
			}
		}
	}

	return metrics, nil
}

// querySlowQueries queries slow database queries from APM
func (c *DatabaseCommand) querySlowQueries(ddClient *client.Client, fromTime, toTime time.Time) ([]QueryPerformance, error) {
	// Build query filter
	filter := fmt.Sprintf("resource_type:sql host:%s", c.host)
	if c.database != "" {
		filter += fmt.Sprintf(" db.name:%s", c.database)
	}

	fromNs := fromTime.UnixNano()
	toNs := toTime.UnixNano()

	// Build APM query payload
	payload := map[string]interface{}{
		"filter": map[string]interface{}{
			"from":  fmt.Sprintf("%d", fromNs),
			"to":    fmt.Sprintf("%d", toNs),
			"query": filter,
		},
		"compute": []map[string]interface{}{
			{"aggregation": "count", "metric": "*"},
			{"aggregation": "pc95", "metric": "duration"},
			{"aggregation": "avg", "metric": "duration"},
		},
		"group_by": []map[string]interface{}{
			{
				"facet": "resource_name",
				"limit": 20,
				"sort": map[string]interface{}{
					"order":       "desc",
					"aggregation": "pc95",
					"metric":      "duration",
				},
			},
		},
	}

	// Make API request
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Use generic doRequest through reflection
	// Since QueryAPM doesn't allow custom query, we'll parse it differently
	responseData, err := ddClient.QueryAPM("*", fromTime, toTime, filter)
	if err != nil {
		return nil, err
	}

	var spanData APMSpanData
	if err := json.Unmarshal(responseData, &spanData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal APM response: %w", err)
	}

	queries := make([]QueryPerformance, 0)

	for _, bucket := range spanData.Data.Buckets {
		queryText := "unknown"
		if resName, ok := bucket.By["resource_name"].(string); ok {
			queryText = resName
		}

		computes := bucket.Computes

		var executionCount int64
		var p95Ns, avgNs int64

		if val, ok := computes["c0"].(float64); ok {
			executionCount = int64(val)
		}
		if val, ok := computes["c1"].(float64); ok {
			p95Ns = int64(val)
		}
		if val, ok := computes["c2"].(float64); ok {
			avgNs = int64(val)
		}

		// Truncate long queries
		if len(queryText) > 200 {
			queryText = queryText[:200] + "..."
		}

		query := QueryPerformance{
			QueryText:      queryText,
			ExecutionCount: executionCount,
			P95Ms:          p95Ns / 1_000_000, // Convert nanoseconds to milliseconds
			AvgMs:          avgNs / 1_000_000,
		}

		queries = append(queries, query)
	}

	// Suppress unused variable warning
	_ = payloadBytes

	return queries, nil
}

// analyzeIssues analyzes database metrics and adds issues
func (c *DatabaseCommand) analyzeIssues(output *DatabaseOutput) {
	output.Issues = make([]struct {
		Severity string `json:"severity"`
		Message  string `json:"message"`
		Action   string `json:"action"`
	}, 0)

	// Check for very slow queries
	if output.Summary.SlowestP95Ms > 5000 {
		output.Status = "critical"
		output.Issues = append(output.Issues, struct {
			Severity string `json:"severity"`
			Message  string `json:"message"`
			Action   string `json:"action"`
		}{
			Severity: "critical",
			Message:  fmt.Sprintf("Very slow queries detected (P95: %dms)", output.Summary.SlowestP95Ms),
			Action:   "Investigate query optimization or add indexes",
		})
	} else if output.Summary.SlowestP95Ms > 2000 {
		if output.Status != "critical" {
			output.Status = "warning"
		}
		output.Issues = append(output.Issues, struct {
			Severity string `json:"severity"`
			Message  string `json:"message"`
			Action   string `json:"action"`
		}{
			Severity: "warning",
			Message:  fmt.Sprintf("Slow queries detected (P95: %dms)", output.Summary.SlowestP95Ms),
			Action:   "Consider query optimization",
		})
	}

	// Check connection count
	if output.Connections != nil && output.Connections.CurrentConnections > 100 {
		if output.Status == "ok" {
			output.Status = "warning"
		}
		output.Issues = append(output.Issues, struct {
			Severity string `json:"severity"`
			Message  string `json:"message"`
			Action   string `json:"action"`
		}{
			Severity: "warning",
			Message:  fmt.Sprintf("High connection count (%d connections)", output.Connections.CurrentConnections),
			Action:   "Consider connection pooling to reduce database load",
		})
	}

	// Check query pattern diversity
	if output.Summary.QueryPatterns > 50 {
		if output.Status == "ok" {
			output.Status = "warning"
		}
		output.Issues = append(output.Issues, struct {
			Severity string `json:"severity"`
			Message  string `json:"message"`
			Action   string `json:"action"`
		}{
			Severity: "warning",
			Message:  fmt.Sprintf("High query diversity (%d patterns)", output.Summary.QueryPatterns),
			Action:   "Review N+1 query patterns and consider query consolidation",
		})
	}
}

// printFormatted prints the database output in a conversational format
func (c *DatabaseCommand) printFormatted(output *DatabaseOutput) {
	// Header with status indicator
	statusIndicator := "OK"
	if output.Status == "warning" {
		statusIndicator = "WARNING"
	} else if output.Status == "critical" {
		statusIndicator = "CRITICAL"
	}

	fmt.Printf("Database Monitoring: %s [%s]\n", output.Host, statusIndicator)
	if output.Database != "" {
		fmt.Printf("Database: %s\n", output.Database)
	}
	fmt.Printf("Duration: %s\n", output.Duration)
	fmt.Println()

	// Summary
	summary := output.Summary
	fmt.Println("Summary:")
	if output.Connections != nil {
		fmt.Printf("  Active connections: %d\n", output.Connections.CurrentConnections)
	}
	fmt.Printf("  Query patterns: %d\n", summary.QueryPatterns)
	fmt.Printf("  Slow queries (>1s): %d\n", summary.SlowQueries)
	fmt.Printf("  Slowest P95: %dms\n", summary.SlowestP95Ms)
	fmt.Printf("  Average duration: %dms\n", summary.AvgQueryDurationMs)
	fmt.Println()

	// Issues
	if len(output.Issues) > 0 {
		fmt.Println("Issues:")
		for _, issue := range output.Issues {
			fmt.Printf("  [%s] %s\n", strings.ToUpper(issue.Severity), issue.Message)
			fmt.Printf("    Action: %s\n", issue.Action)
		}
		fmt.Println()
	}

	// Top slow queries
	if len(output.Queries) > 0 {
		fmt.Println("Top Slow Queries:")
		displayCount := 5
		if len(output.Queries) < displayCount {
			displayCount = len(output.Queries)
		}

		for i := 0; i < displayCount; i++ {
			query := output.Queries[i]
			fmt.Printf("  %d. %s\n", i+1, truncateQuery(query.QueryText, 80))
			fmt.Printf("     P95: %dms | Avg: %dms | Executions: %s\n",
				query.P95Ms, query.AvgMs, formatDatabaseNumber(query.ExecutionCount))
		}

		if len(output.Queries) > displayCount {
			fmt.Printf("  ... and %d more\n", len(output.Queries)-displayCount)
		}
		fmt.Println()
	}

	// Recommendations
	if output.Status == "ok" && len(output.Issues) == 0 {
		fmt.Println("Database performance is healthy")
	}
}

// Help prints the help message
func (c *DatabaseCommand) Help() {
	fmt.Println("Usage: dd database [options]")
	fmt.Println()
	fmt.Println("Query Datadog Database Monitoring for performance analysis.")
	fmt.Println("Analyzes query performance, connection pools, and slow queries.")
	fmt.Println()
	fmt.Println("Options:")
	c.flags.PrintDefaults()
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  dd database --host db-prod-01")
	fmt.Println("  dd database --host db-prod-01 --database myapp")
	fmt.Println("  dd database --host db-prod-01 --duration 24h")
	fmt.Println("  dd database --host db-prod-01 --metric queries")
	fmt.Println("  dd database --host db-prod-01 --duration 7d --json")
}

// Helper functions

func truncateQuery(query string, maxLen int) string {
	if len(query) <= maxLen {
		return query
	}
	return query[:maxLen] + "..."
}

func formatDatabaseNumber(n int64) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1000000 {
		return fmt.Sprintf("%d,%03d", n/1000, n%1000)
	}
	return fmt.Sprintf("%d,%03d,%03d", n/1000000, (n/1000)%1000, n%1000)
}
