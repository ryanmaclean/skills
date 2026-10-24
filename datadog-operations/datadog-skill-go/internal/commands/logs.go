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

// LogsCommand searches and analyzes Datadog logs
type LogsCommand struct {
	flags    *flag.FlagSet
	query    string
	service  string
	duration string
	status   string
	limit    int
	json     bool
}

// LogEntry represents a parsed log entry from Datadog API
type LogEntry struct {
	Timestamp   string                 `json:"timestamp"`
	Status      string                 `json:"status"`
	Service     string                 `json:"service"`
	Host        string                 `json:"host"`
	Message     string                 `json:"message"`
	TraceID     string                 `json:"trace_id,omitempty"`
	SpanID      string                 `json:"span_id,omitempty"`
	ContainerID string                 `json:"container_id,omitempty"`
	Attributes  map[string]interface{} `json:"attributes,omitempty"`
}

// ErrorPattern represents a grouped error pattern
type ErrorPattern struct {
	Message  string   `json:"message"`
	Count    int      `json:"count"`
	Services []string `json:"services"`
	Hosts    []string `json:"hosts"`
}

// LogsResponse represents the formatted logs response
type LogsResponse struct {
	Status        string                 `json:"status"`
	TotalLogs     int                    `json:"total_logs"`
	Query         string                 `json:"query"`
	Duration      string                 `json:"duration"`
	Summary       map[string]int         `json:"summary"`
	ErrorPatterns []ErrorPattern         `json:"error_patterns,omitempty"`
	Services      map[string]int         `json:"services"`
	Hosts         map[string]int         `json:"hosts"`
	TraceIDsCount int                    `json:"trace_ids_count"`
	RecentLogs    []LogEntry             `json:"recent_logs,omitempty"`
	RawData       map[string]interface{} `json:"raw_data,omitempty"`
}

// NewLogsCommand creates a new logs command
func NewLogsCommand() *LogsCommand {
	cmd := &LogsCommand{
		flags: flag.NewFlagSet("logs", flag.ExitOnError),
	}

	cmd.flags.StringVar(&cmd.query, "query", "", "Log search query (Datadog query syntax)")
	cmd.flags.StringVar(&cmd.service, "service", "", "Filter by service (auto-detect if not provided)")
	cmd.flags.StringVar(&cmd.duration, "duration", "24h", "Time range: 1h, 24h, 7d, 30d")
	cmd.flags.StringVar(&cmd.status, "status", "", "Filter by status: error, warn, info")
	cmd.flags.IntVar(&cmd.limit, "limit", 100, "Number of results (max: 1000)")
	cmd.flags.BoolVar(&cmd.json, "json", false, "Output as JSON")

	return cmd
}

// Name returns the command name
func (c *LogsCommand) Name() string {
	return "logs"
}

// Description returns the command description
func (c *LogsCommand) Description() string {
	return "Search and analyze Datadog logs for error patterns"
}

// Run executes the logs command
func (c *LogsCommand) Run(args []string) error {
	// Initialize observability
	obs, err := observability.Init("dd-cli", "production")
	if err != nil {
		return fmt.Errorf("failed to init observability: %w", err)
	}
	defer obs.Shutdown(0)

	// Parse flags
	if err := c.flags.Parse(args); err != nil {
		return err
	}

	// Start tracing
	span := obs.StartSpan("logs.search")
	defer obs.FinishSpan(span)

	obs.LogInfo("Starting log search")

	// Validate limit
	if c.limit > 1000 {
		obs.LogError("Limit cannot exceed 1000")
		return fmt.Errorf("limit cannot exceed 1000")
	}

	// Auto-detect service if needed
	service := c.service
	if service == "" && c.query == "" {
		detectSpan := obs.StartSpan("logs.detect_context")
		obs.LogInfo("Auto-detecting service context")

		ctx, err := context.DetectContext(".")
		if err != nil {
			obs.LogWarning("Failed to detect context, proceeding without service filter")
		} else {
			service = ctx.ServiceName
			obs.LogInfo(fmt.Sprintf("Auto-detected service: %s", service))
			obs.GetTracer().SetTag(detectSpan, "service.name", service)
			obs.GetTracer().SetTag(detectSpan, "detection.method", ctx.DetectionMethod)
		}
		obs.FinishSpan(detectSpan)
	}

	// Build query
	buildSpan := obs.StartSpan("logs.build_query")
	searchQuery := c.buildQuery(service)
	obs.LogInfo(fmt.Sprintf("Search query: %s", searchQuery))
	obs.GetTracer().SetTag(buildSpan, "search.query", searchQuery)
	obs.FinishSpan(buildSpan)

	// Parse duration
	fromTime, toTime, err := c.parseDuration()
	if err != nil {
		obs.LogError("Failed to parse duration: " + err.Error())
		return fmt.Errorf("failed to parse duration: %w", err)
	}

	// Create Datadog client
	clientSpan := obs.StartSpan("logs.create_client")
	ddClient, err := client.NewClient()
	if err != nil {
		obs.LogError("Failed to create Datadog client: " + err.Error())
		return fmt.Errorf("failed to create Datadog client: %w", err)
	}
	obs.FinishSpan(clientSpan)

	// Search logs
	searchSpan := obs.StartSpan("logs.api_search")
	obs.GetTracer().SetTag(searchSpan, "query", searchQuery)
	obs.GetTracer().SetTag(searchSpan, "duration", c.duration)
	obs.GetTracer().SetTag(searchSpan, "limit", c.limit)

	start := time.Now()
	rawData, err := ddClient.SearchLogs(searchQuery, fromTime, toTime, c.limit)
	apiDuration := time.Since(start).Milliseconds()

	if err != nil {
		obs.LogError("API call failed: " + err.Error())
		obs.RecordAPICall("/api/v2/logs/events/search", "POST", 500, float64(apiDuration), err)
		obs.GetTracer().SetError(searchSpan, err)
		obs.FinishSpan(searchSpan)
		return fmt.Errorf("failed to search logs: %w", err)
	}

	obs.RecordAPICall("/api/v2/logs/events/search", "POST", 200, float64(apiDuration), nil)
	obs.FinishSpan(searchSpan)

	// Parse and analyze results
	parseSpan := obs.StartSpan("logs.parse_results")
	response, err := c.parseResponse(rawData, searchQuery)
	if err != nil {
		obs.LogError("Failed to parse response: " + err.Error())
		obs.FinishSpan(parseSpan)
		return fmt.Errorf("failed to parse response: %w", err)
	}
	obs.FinishSpan(parseSpan)

	// Record metrics
	obs.LogInfo(fmt.Sprintf("Found %d log entries", response.TotalLogs))
	obs.GetMetrics().Gauge("logs.total", float64(response.TotalLogs))
	obs.GetMetrics().Gauge("logs.errors", float64(response.Summary["error"]))
	obs.GetMetrics().Gauge("logs.warnings", float64(response.Summary["warn"]))
	obs.GetMetrics().Count("logs.search.count", 1,
		"service:"+service,
		"status:"+response.Status,
	)

	// Output
	if c.json {
		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	} else {
		c.printFormatted(response)
	}

	obs.LogInfo("Log search complete")
	return nil
}

// buildQuery constructs the log search query
func (c *LogsCommand) buildQuery(service string) string {
	var queryParts []string

	if service != "" {
		queryParts = append(queryParts, fmt.Sprintf("service:%s", service))
	}

	if c.status != "" {
		queryParts = append(queryParts, fmt.Sprintf("status:%s", c.status))
	}

	if c.query != "" {
		queryParts = append(queryParts, fmt.Sprintf("(%s)", c.query))
	}

	// Default to error if no query specified
	if len(queryParts) == 0 {
		queryParts = append(queryParts, "status:error")
	}

	return strings.Join(queryParts, " AND ")
}

// parseDuration parses the duration string into time range
func (c *LogsCommand) parseDuration() (time.Time, time.Time, error) {
	toTime := time.Now()
	var duration time.Duration

	if strings.HasSuffix(c.duration, "h") {
		hours := strings.TrimSuffix(c.duration, "h")
		var h int
		if _, err := fmt.Sscanf(hours, "%d", &h); err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid duration: %s", c.duration)
		}
		duration = time.Duration(h) * time.Hour
	} else if strings.HasSuffix(c.duration, "d") {
		days := strings.TrimSuffix(c.duration, "d")
		var d int
		if _, err := fmt.Sscanf(days, "%d", &d); err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid duration: %s", c.duration)
		}
		duration = time.Duration(d) * 24 * time.Hour
	} else {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid duration format: %s. Use format like '1h', '24h', '7d', '30d'", c.duration)
	}

	fromTime := toTime.Add(-duration)
	return fromTime, toTime, nil
}

// parseResponse parses the raw API response into structured format
func (c *LogsCommand) parseResponse(rawData []byte, searchQuery string) (*LogsResponse, error) {
	var apiResponse map[string]interface{}
	if err := json.Unmarshal(rawData, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	response := &LogsResponse{
		Query:      searchQuery,
		Duration:   c.duration,
		Summary:    make(map[string]int),
		Services:   make(map[string]int),
		Hosts:      make(map[string]int),
		RecentLogs: []LogEntry{},
		RawData:    apiResponse,
	}

	// Extract logs from API response
	logs, ok := apiResponse["data"].([]interface{})
	if !ok {
		response.TotalLogs = 0
		response.Status = "ok"
		return response, nil
	}

	response.TotalLogs = len(logs)

	if response.TotalLogs == 0 {
		response.Status = "ok"
		return response, nil
	}

	// Parse logs and collect statistics
	var errorMessages []map[string]interface{}
	traceIDs := make(map[string]bool)

	for i, logItem := range logs {
		logMap, ok := logItem.(map[string]interface{})
		if !ok {
			continue
		}

		attrs, ok := logMap["attributes"].(map[string]interface{})
		if !ok {
			continue
		}

		status := getString(attrs, "status", "unknown")
		service := getString(attrs, "service", "unknown")
		host := getString(attrs, "host", "unknown")
		message := getString(attrs, "message", "No message")
		timestamp := getString(attrs, "timestamp", "")

		// Count by status
		response.Summary[status]++
		response.Services[service]++
		response.Hosts[host]++

		// Extract trace information
		if attrsNested, ok := attrs["attributes"].(map[string]interface{}); ok {
			if ddInfo, ok := attrsNested["dd"].(map[string]interface{}); ok {
				if traceID := getString(ddInfo, "trace_id", ""); traceID != "" {
					traceIDs[traceID] = true
				}
			}
		}

		// Collect error messages for pattern analysis
		if status == "error" {
			errorMessages = append(errorMessages, map[string]interface{}{
				"message": truncate(message, 200),
				"service": service,
				"host":    host,
			})
		}

		// Keep first 10 logs for detailed output
		if i < 10 {
			entry := LogEntry{
				Timestamp: timestamp,
				Status:    status,
				Service:   service,
				Host:      host,
				Message:   truncate(message, 500),
			}

			// Extract trace correlation
			if attrsNested, ok := attrs["attributes"].(map[string]interface{}); ok {
				if ddInfo, ok := attrsNested["dd"].(map[string]interface{}); ok {
					entry.TraceID = getString(ddInfo, "trace_id", "")
					entry.SpanID = getString(ddInfo, "span_id", "")
				}
				entry.ContainerID = getString(attrsNested, "container_id", "")
			}

			response.RecentLogs = append(response.RecentLogs, entry)
		}
	}

	response.TraceIDsCount = len(traceIDs)

	// Analyze error patterns
	response.ErrorPatterns = analyzeErrorPatterns(errorMessages, 10)

	// Determine overall status
	errorCount := response.Summary["error"]
	warnCount := response.Summary["warn"]

	if errorCount > 0 {
		response.Status = "error"
	} else if warnCount > 0 {
		response.Status = "warning"
	} else {
		response.Status = "ok"
	}

	return response, nil
}

// analyzeErrorPatterns groups error messages by frequency
func analyzeErrorPatterns(errorMessages []map[string]interface{}, maxPatterns int) []ErrorPattern {
	if len(errorMessages) == 0 {
		return []ErrorPattern{}
	}

	// Count messages
	messageCounts := make(map[string]int)
	messageDetails := make(map[string]map[string]map[string]bool)

	for _, errMsg := range errorMessages {
		message := errMsg["message"].(string)
		service := errMsg["service"].(string)
		host := errMsg["host"].(string)

		messageCounts[message]++

		if messageDetails[message] == nil {
			messageDetails[message] = map[string]map[string]bool{
				"services": {},
				"hosts":    {},
			}
		}
		messageDetails[message]["services"][service] = true
		messageDetails[message]["hosts"][host] = true
	}

	// Sort by count
	type messageCount struct {
		message string
		count   int
	}
	var sorted []messageCount
	for msg, count := range messageCounts {
		sorted = append(sorted, messageCount{msg, count})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].count > sorted[j].count
	})

	// Build patterns
	var patterns []ErrorPattern
	for i, mc := range sorted {
		if i >= maxPatterns {
			break
		}

		details := messageDetails[mc.message]
		var services []string
		var hosts []string

		for svc := range details["services"] {
			services = append(services, svc)
		}
		for host := range details["hosts"] {
			hosts = append(hosts, host)
		}

		sort.Strings(services)
		sort.Strings(hosts)

		patterns = append(patterns, ErrorPattern{
			Message:  mc.message,
			Count:    mc.count,
			Services: services,
			Hosts:    hosts,
		})
	}

	return patterns
}

// printFormatted prints the logs response in a conversational format
func (c *LogsCommand) printFormatted(response *LogsResponse) {
	fmt.Println("Log Search Results")
	fmt.Printf("Query: %s\n", response.Query)
	fmt.Printf("Duration: %s\n", response.Duration)
	fmt.Println()
	fmt.Printf("Found %d log entries\n", response.TotalLogs)

	if response.TotalLogs == 0 {
		return
	}

	fmt.Println()
	fmt.Println("Status breakdown:")
	fmt.Printf("  Errors: %d\n", response.Summary["error"])
	fmt.Printf("  Warnings: %d\n", response.Summary["warn"])
	fmt.Printf("  Info: %d\n", response.Summary["info"])

	if len(response.ErrorPatterns) > 0 {
		fmt.Println()
		fmt.Printf("Top error patterns (%d):\n", len(response.ErrorPatterns))
		for i, pattern := range response.ErrorPatterns {
			if i >= 5 {
				break
			}
			fmt.Printf("  %d. %s...\n", i+1, truncate(pattern.Message, 80))
			fmt.Printf("     Count: %d | Services: %s\n",
				pattern.Count,
				strings.Join(limitSlice(pattern.Services, 3), ", "))
		}
	}

	if len(response.Services) > 0 {
		fmt.Println()
		fmt.Println("Top services:")

		// Sort services by count
		type serviceCount struct {
			name  string
			count int
		}
		var services []serviceCount
		for name, count := range response.Services {
			services = append(services, serviceCount{name, count})
		}
		sort.Slice(services, func(i, j int) bool {
			return services[i].count > services[j].count
		})

		for i, svc := range services {
			if i >= 5 {
				break
			}
			fmt.Printf("  %s: %d\n", svc.name, svc.count)
		}
	}

	if response.TraceIDsCount > 0 {
		fmt.Println()
		fmt.Printf("Found %d logs with trace IDs\n", response.TraceIDsCount)
	}
}

// Help prints the help message
func (c *LogsCommand) Help() {
	fmt.Println("Usage: dd logs [options]")
	fmt.Println()
	fmt.Println("Search and analyze Datadog logs for error patterns")
	fmt.Println()
	fmt.Println("Options:")
	c.flags.PrintDefaults()
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  dd logs")
	fmt.Println("  dd logs --query 'error AND database'")
	fmt.Println("  dd logs --service my-api --status error --duration 1h")
	fmt.Println("  dd logs --json")
}

// Helper functions

func getString(m map[string]interface{}, key, defaultVal string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultVal
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

func limitSlice(s []string, max int) []string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
