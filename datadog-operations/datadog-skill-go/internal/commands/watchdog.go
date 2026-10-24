package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/datadog/skill/internal/client"
	"github.com/datadog/skill/internal/context"
	"github.com/datadog/skill/internal/observability"
)

// WatchdogCommand queries Datadog Watchdog for anomaly detection
type WatchdogCommand struct {
	flags    *flag.FlagSet
	service  string
	duration string
	category string
	limit    int
	json     bool
}

// WatchdogAnomaly represents a parsed Watchdog anomaly
type WatchdogAnomaly struct {
	ID              string   `json:"id"`
	Type            string   `json:"type"`
	Timestamp       string   `json:"timestamp"`
	Title           string   `json:"title"`
	Message         string   `json:"message"`
	Tags            []string `json:"tags"`
	Priority        string   `json:"priority"`
	Service         string   `json:"service,omitempty"`
	Resource        string   `json:"resource,omitempty"`
	AnomalyCategory string   `json:"anomaly_category"`
	Severity        string   `json:"severity"`
}

// WatchdogResponse represents the formatted watchdog response
type WatchdogResponse struct {
	Status           string                 `json:"status"`
	TotalAnomalies   int                    `json:"total_anomalies"`
	Duration         string                 `json:"duration"`
	Service          string                 `json:"service"`
	Query            string                 `json:"query"`
	Summary          *WatchdogSummary       `json:"summary"`
	AffectedServices map[string]int         `json:"affected_services"`
	Anomalies        []WatchdogAnomaly      `json:"anomalies"`
	RawData          map[string]interface{} `json:"raw_data,omitempty"`
}

// WatchdogSummary contains anomaly statistics
type WatchdogSummary struct {
	LatencySpikes      int `json:"latency_spikes"`
	ErrorRateIncreases int `json:"error_rate_increases"`
	TrafficDrops       int `json:"traffic_drops"`
	Other              int `json:"other"`
}

// NewWatchdogCommand creates a new watchdog command
func NewWatchdogCommand() *WatchdogCommand {
	cmd := &WatchdogCommand{
		flags: flag.NewFlagSet("watchdog", flag.ExitOnError),
	}

	cmd.flags.StringVar(&cmd.service, "service", "", "Filter by service (auto-detect if not provided)")
	cmd.flags.StringVar(&cmd.duration, "duration", "24h", "Time range: 1h, 24h, 7d, 30d")
	cmd.flags.StringVar(&cmd.category, "category", "all", "Filter by category: apm, infrastructure, logs, all")
	cmd.flags.IntVar(&cmd.limit, "limit", 20, "Number of results (default: 20)")
	cmd.flags.BoolVar(&cmd.json, "json", false, "Output as JSON")

	return cmd
}

// Name returns the command name
func (c *WatchdogCommand) Name() string {
	return "watchdog"
}

// Description returns the command description
func (c *WatchdogCommand) Description() string {
	return "Query Datadog Watchdog for automated anomaly detection"
}

// Run executes the watchdog command
func (c *WatchdogCommand) Run(args []string) error {
	// Initialize observability
	obs, err := observability.Init("query-watchdog", "production")
	if err != nil {
		return fmt.Errorf("failed to init observability: %w", err)
	}
	defer obs.Shutdown(0)

	// Parse flags
	if err := c.flags.Parse(args); err != nil {
		return err
	}

	// Start tracing
	span := obs.StartSpan("watchdog.query")
	defer obs.FinishSpan(span)

	obs.LogInfo("Starting Watchdog anomaly query")

	// Auto-detect service if needed
	service := c.service
	if service == "" {
		detectSpan := obs.StartSpan("watchdog.detect_context")
		obs.LogInfo("Auto-detecting service context")

		ctx, err := context.DetectContext(".")
		if err != nil {
			obs.LogWarning("Failed to detect context, querying all services")
		} else {
			service = ctx.ServiceName
			obs.LogInfo(fmt.Sprintf("Auto-detected service: %s", service))
			obs.GetTracer().SetTag(detectSpan, "service.name", service)
			obs.GetTracer().SetTag(detectSpan, "detection.method", ctx.DetectionMethod)
		}
		obs.FinishSpan(detectSpan)
	}

	// Build query
	buildSpan := obs.StartSpan("watchdog.build_query")
	query := c.buildQuery(service)
	obs.LogInfo(fmt.Sprintf("Watchdog query: %s", query))
	obs.GetTracer().SetTag(buildSpan, "query", query)
	obs.FinishSpan(buildSpan)

	// Parse duration
	parseSpan := obs.StartSpan("watchdog.parse_duration")
	fromTime, toTime, err := c.parseDuration()
	if err != nil {
		obs.LogError("Failed to parse duration: " + err.Error())
		obs.FinishSpan(parseSpan)
		return fmt.Errorf("failed to parse duration: %w", err)
	}
	obs.GetTracer().SetTag(parseSpan, "from", fromTime.Format(time.RFC3339))
	obs.GetTracer().SetTag(parseSpan, "to", toTime.Format(time.RFC3339))
	obs.FinishSpan(parseSpan)

	// Create Datadog client
	clientSpan := obs.StartSpan("watchdog.create_client")
	ddClient, err := client.NewClient()
	if err != nil {
		obs.LogError("Failed to create Datadog client: " + err.Error())
		obs.FinishSpan(clientSpan)
		return fmt.Errorf("failed to create Datadog client: %w", err)
	}
	obs.FinishSpan(clientSpan)

	// Query Watchdog alerts
	querySpan := obs.StartSpan("watchdog.api_query")
	obs.GetTracer().SetTag(querySpan, "service", service)
	obs.GetTracer().SetTag(querySpan, "duration", c.duration)
	obs.GetTracer().SetTag(querySpan, "category", c.category)

	start := time.Now()
	rawData, err := ddClient.WatchdogAlerts(query, fromTime, toTime, c.limit)
	apiDuration := time.Since(start).Milliseconds()

	if err != nil {
		obs.LogError("API call failed: " + err.Error())
		obs.RecordAPICall("/api/v2/events/search", "POST", 500, float64(apiDuration), err)
		obs.GetTracer().SetError(querySpan, err)
		obs.FinishSpan(querySpan)
		return fmt.Errorf("failed to query Watchdog alerts: %w", err)
	}

	obs.RecordAPICall("/api/v2/events/search", "POST", 200, float64(apiDuration), nil)
	obs.FinishSpan(querySpan)

	// Parse and analyze results
	analyzeSpan := obs.StartSpan("watchdog.parse_results")
	response, err := c.parseResponse(rawData, service, query)
	if err != nil {
		obs.LogError("Failed to parse response: " + err.Error())
		obs.FinishSpan(analyzeSpan)
		return fmt.Errorf("failed to parse response: %w", err)
	}
	obs.FinishSpan(analyzeSpan)

	// Record metrics
	obs.LogInfo(fmt.Sprintf("Found %d Watchdog anomalies", response.TotalAnomalies))
	obs.GetMetrics().Gauge("watchdog.anomalies.total", float64(response.TotalAnomalies))
	obs.GetMetrics().Gauge("watchdog.anomalies.latency_spikes", float64(response.Summary.LatencySpikes))
	obs.GetMetrics().Gauge("watchdog.anomalies.error_rate_increases", float64(response.Summary.ErrorRateIncreases))
	obs.GetMetrics().Gauge("watchdog.anomalies.traffic_drops", float64(response.Summary.TrafficDrops))
	obs.GetMetrics().Count("watchdog.query.count", 1,
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

	obs.LogInfo("Watchdog anomaly query complete")
	return nil
}

// buildQuery builds the Watchdog query string
func (c *WatchdogCommand) buildQuery(service string) string {
	queryParts := []string{"source:watchdog"}

	if service != "" {
		queryParts = append(queryParts, fmt.Sprintf("service:%s", service))
	}

	// Add category filters
	switch c.category {
	case "apm":
		queryParts = append(queryParts, "(latency OR p99 OR response_time OR error OR error_rate OR errors OR hits OR traffic OR request_rate OR throughput)")
	case "infrastructure":
		queryParts = append(queryParts, "(cpu OR memory OR disk OR network)")
	case "logs":
		queryParts = append(queryParts, "(log OR logs)")
		// "all" means no additional filter
	}

	return strings.Join(queryParts, " ")
}

// parseDuration parses the duration string into time range
func (c *WatchdogCommand) parseDuration() (time.Time, time.Time, error) {
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
func (c *WatchdogCommand) parseResponse(rawData []byte, service, query string) (*WatchdogResponse, error) {
	var apiResponse map[string]interface{}
	if err := json.Unmarshal(rawData, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	response := &WatchdogResponse{
		Duration:         c.duration,
		Service:          service,
		Query:            query,
		Summary:          &WatchdogSummary{},
		AffectedServices: make(map[string]int),
		Anomalies:        []WatchdogAnomaly{},
		RawData:          apiResponse,
	}

	// Extract events from API response
	events, ok := apiResponse["data"].([]interface{})
	if !ok {
		response.TotalAnomalies = 0
		response.Status = "ok"
		return response, nil
	}

	response.TotalAnomalies = len(events)

	if response.TotalAnomalies == 0 {
		response.Status = "ok"
		return response, nil
	}

	// Parse events and collect statistics
	for _, eventItem := range events {
		eventMap, ok := eventItem.(map[string]interface{})
		if !ok {
			continue
		}

		attrs, ok := eventMap["attributes"].(map[string]interface{})
		if !ok {
			continue
		}

		eventAttrs, ok := attrs["attributes"].(map[string]interface{})
		if !ok {
			eventAttrs = make(map[string]interface{})
		}

		eventID := getWatchdogString(eventMap, "id", "")
		eventType := getWatchdogString(eventMap, "type", "")
		timestamp := getWatchdogString(attrs, "timestamp", "")
		title := getWatchdogString(eventAttrs, "title", "")
		message := getWatchdogString(eventAttrs, "message", "")
		priority := getWatchdogString(eventAttrs, "priority", "normal")

		// Extract first line of message
		if message != "" {
			lines := strings.Split(message, "\n")
			message = lines[0]
		}

		// Extract tags
		tags := []string{}
		if tagsArr, ok := attrs["tags"].([]interface{}); ok {
			for _, tag := range tagsArr {
				if tagStr, ok := tag.(string); ok {
					tags = append(tags, tagStr)
				}
			}
		}

		// Extract service and resource from tags
		serviceTag := extractTagValue(tags, "service:")
		resourceTag := extractTagValue(tags, "resource_name:")

		// Categorize anomaly
		anomalyCategory := c.categorizeAnomaly(title)

		// Update category counts
		switch anomalyCategory {
		case "latency_spike":
			response.Summary.LatencySpikes++
		case "error_rate_increase":
			response.Summary.ErrorRateIncreases++
		case "traffic_drop":
			response.Summary.TrafficDrops++
		default:
			response.Summary.Other++
		}

		// Count affected services
		if serviceTag != "" {
			response.AffectedServices[serviceTag]++
		}

		// Map priority to severity
		severity := c.mapPriorityToSeverity(priority)

		anomaly := WatchdogAnomaly{
			ID:              eventID,
			Type:            eventType,
			Timestamp:       timestamp,
			Title:           title,
			Message:         message,
			Tags:            tags,
			Priority:        priority,
			Service:         serviceTag,
			Resource:        resourceTag,
			AnomalyCategory: anomalyCategory,
			Severity:        severity,
		}

		response.Anomalies = append(response.Anomalies, anomaly)
	}

	// Determine overall status
	if response.Summary.ErrorRateIncreases > 0 {
		response.Status = "critical"
	} else if response.Summary.LatencySpikes > 3 || response.Summary.TrafficDrops > 3 {
		response.Status = "warning"
	} else {
		response.Status = "ok"
	}

	return response, nil
}

// categorizeAnomaly categorizes anomaly based on title
func (c *WatchdogCommand) categorizeAnomaly(title string) string {
	titleLower := strings.ToLower(title)

	latencyPattern := regexp.MustCompile(`latency|p99|response.?time`)
	errorPattern := regexp.MustCompile(`error|failure`)
	trafficPattern := regexp.MustCompile(`hits|traffic|request|throughput|drop`)

	if latencyPattern.MatchString(titleLower) {
		return "latency_spike"
	} else if errorPattern.MatchString(titleLower) {
		return "error_rate_increase"
	} else if trafficPattern.MatchString(titleLower) {
		return "traffic_drop"
	}

	return "other"
}

// mapPriorityToSeverity maps Datadog priority to severity level
func (c *WatchdogCommand) mapPriorityToSeverity(priority string) string {
	switch priority {
	case "low":
		return "low"
	case "normal":
		return "medium"
	default:
		return "high"
	}
}

// printFormatted prints the watchdog response in a conversational format
func (c *WatchdogCommand) printFormatted(response *WatchdogResponse) {
	fmt.Println("Watchdog Anomaly Detection")
	fmt.Printf("Duration: %s\n", response.Duration)
	if response.Service != "" {
		fmt.Printf("Service: %s\n", response.Service)
	}
	fmt.Println()
	fmt.Printf("Found %d anomalies\n", response.TotalAnomalies)

	if response.TotalAnomalies == 0 {
		fmt.Println()
		fmt.Println("No Watchdog anomalies found for the specified criteria")
		return
	}

	fmt.Println()
	fmt.Println("Anomaly type breakdown:")
	fmt.Printf("  Latency spikes: %d\n", response.Summary.LatencySpikes)
	fmt.Printf("  Error rate increases: %d\n", response.Summary.ErrorRateIncreases)
	fmt.Printf("  Traffic drops: %d\n", response.Summary.TrafficDrops)
	fmt.Printf("  Other: %d\n", response.Summary.Other)

	if response.Summary.ErrorRateIncreases > 0 {
		fmt.Println()
		fmt.Printf("CRITICAL: %d error rate increases detected\n", response.Summary.ErrorRateIncreases)
	}

	if len(response.AffectedServices) > 0 {
		fmt.Println()
		fmt.Println("Affected services:")

		// Sort services by count
		type serviceCount struct {
			name  string
			count int
		}
		var services []serviceCount
		for name, count := range response.AffectedServices {
			services = append(services, serviceCount{name, count})
		}
		sort.Slice(services, func(i, j int) bool {
			return services[i].count > services[j].count
		})

		displayCount := 5
		if len(services) < displayCount {
			displayCount = len(services)
		}

		for i := 0; i < displayCount; i++ {
			fmt.Printf("  %s: %d anomalies\n", services[i].name, services[i].count)
		}
	}

	// Show recent anomalies
	if len(response.Anomalies) > 0 {
		fmt.Println()
		fmt.Println("Recent anomalies:")

		displayCount := 5
		if len(response.Anomalies) < displayCount {
			displayCount = len(response.Anomalies)
		}

		for i := 0; i < displayCount; i++ {
			anomaly := response.Anomalies[i]
			fmt.Printf("  [%s] %s\n", strings.ToUpper(anomaly.Severity), anomaly.Title)
			if anomaly.Service != "" {
				fmt.Printf("    Service: %s\n", anomaly.Service)
			}
			if anomaly.Message != "" && anomaly.Message != anomaly.Title {
				fmt.Printf("    %s\n", anomaly.Message)
			}
		}
	}

	// Show recommendations
	if response.Summary.ErrorRateIncreases > 0 || response.Summary.LatencySpikes > 0 {
		fmt.Println()
		fmt.Println("Recommended actions:")
		if response.Summary.ErrorRateIncreases > 0 {
			fmt.Println("  - Investigate error rate spikes in affected services")
			fmt.Println("  - Check recent deployments and rollback if necessary")
			fmt.Println("  - Review error logs for patterns")
		}
		if response.Summary.LatencySpikes > 0 {
			fmt.Println("  - Analyze latency patterns in APM traces")
			fmt.Println("  - Check database and external service performance")
			fmt.Println("  - Review resource utilization metrics")
		}
		if response.Summary.TrafficDrops > 0 {
			fmt.Println("  - Verify traffic routing and load balancer configuration")
			fmt.Println("  - Check for upstream service issues")
		}
	}
}

// Help prints the help message
func (c *WatchdogCommand) Help() {
	fmt.Println("Usage: dd watchdog [options]")
	fmt.Println()
	fmt.Println("Query Datadog Watchdog for automated anomaly detection")
	fmt.Println()
	fmt.Println("Options:")
	c.flags.PrintDefaults()
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  dd watchdog")
	fmt.Println("  dd watchdog --service my-api --duration 1h")
	fmt.Println("  dd watchdog --category apm --duration 7d")
	fmt.Println("  dd watchdog --json")
	fmt.Println("  dd watchdog --duration 30d --limit 50")
}

// Helper functions

func getWatchdogString(m map[string]interface{}, key, defaultVal string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultVal
}
