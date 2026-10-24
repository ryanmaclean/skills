package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/datadog/skill/internal/client"
)

// AnomaliesCommand handles anomaly detection operations
type AnomaliesCommand struct {
	flags    *flag.FlagSet
	action   string
	service  string
	metric   string
	severity string
	from     string
	to       string
	status   string
	anomType string
	jsonOut  bool
}

// Anomaly represents a detected anomaly
type Anomaly struct {
	ID            string    `json:"id"`
	Type          string    `json:"type"` // metric, log, trace, infrastructure
	Severity      string    `json:"severity"`
	Service       string    `json:"service,omitempty"`
	Metric        string    `json:"metric,omitempty"`
	Description   string    `json:"description"`
	DetectedAt    time.Time `json:"detected_at"`
	StartTime     time.Time `json:"start_time"`
	EndTime       time.Time `json:"end_time,omitempty"`
	Impact        string    `json:"impact"`
	Confidence    float64   `json:"confidence"`
	Baseline      float64   `json:"baseline,omitempty"`
	ObservedValue float64   `json:"observed_value,omitempty"`
	Deviation     float64   `json:"deviation_percent,omitempty"`
	Status        string    `json:"status"` // active, resolved
}

// AnomalyAnalytics provides aggregated anomaly statistics
type AnomalyAnalytics struct {
	TotalAnomalies    int            `json:"total_anomalies"`
	BySeverity        map[string]int `json:"by_severity"`
	ByType            map[string]int `json:"by_type"`
	ByService         map[string]int `json:"by_service"`
	TopImpacted       []string       `json:"top_impacted_services"`
	ActiveAnomalies   int            `json:"active_anomalies"`
	ResolvedAnomalies int            `json:"resolved_anomalies"`
}

// MetricAnomaly represents a metric anomaly detection result
type MetricAnomaly struct {
	Metric        string    `json:"metric"`
	Timestamp     time.Time `json:"timestamp"`
	Value         float64   `json:"value"`
	Baseline      float64   `json:"baseline"`
	UpperBound    float64   `json:"upper_bound"`
	LowerBound    float64   `json:"lower_bound"`
	Deviation     float64   `json:"deviation_percent"`
	IsAnomaly     bool      `json:"is_anomaly"`
	Severity      string    `json:"severity"`
	ConfidenceLevel float64 `json:"confidence"`
}

// LogAnomaly represents log pattern anomalies
type LogAnomaly struct {
	Pattern       string    `json:"pattern"`
	Service       string    `json:"service"`
	DetectedAt    time.Time `json:"detected_at"`
	Count         int64     `json:"count"`
	BaselineCount int64     `json:"baseline_count"`
	Increase      float64   `json:"increase_percent"`
	Severity      string    `json:"severity"`
	Example       string    `json:"example_message"`
}

// TraceAnomaly represents APM trace anomalies
type TraceAnomaly struct {
	Service       string    `json:"service"`
	Operation     string    `json:"operation"`
	DetectedAt    time.Time `json:"detected_at"`
	AnomalyType   string    `json:"anomaly_type"` // latency, error_rate, throughput
	Metric        string    `json:"metric"`
	CurrentValue  float64   `json:"current_value"`
	BaselineValue float64   `json:"baseline_value"`
	Deviation     float64   `json:"deviation_percent"`
	Severity      string    `json:"severity"`
}

// InfrastructureAnomaly represents infrastructure anomalies
type InfrastructureAnomaly struct {
	Host          string    `json:"host"`
	Metric        string    `json:"metric"` // cpu, memory, disk, network
	DetectedAt    time.Time `json:"detected_at"`
	CurrentValue  float64   `json:"current_value"`
	BaselineValue float64   `json:"baseline_value"`
	Threshold     float64   `json:"threshold"`
	Severity      string    `json:"severity"`
	Impact        string    `json:"impact"`
}

// NewAnomaliesCommand creates a new anomalies command
func NewAnomaliesCommand() Command {
	cmd := &AnomaliesCommand{
		flags: flag.NewFlagSet("anomalies", flag.ExitOnError),
	}

	cmd.flags.StringVar(&cmd.action, "action", "list", "Action to perform: list, metrics, logs, traces, infrastructure, search")
	cmd.flags.StringVar(&cmd.service, "service", "", "Filter by service name")
	cmd.flags.StringVar(&cmd.metric, "metric", "", "Metric name for metric anomaly detection")
	cmd.flags.StringVar(&cmd.severity, "severity", "", "Filter by severity: low, medium, high, critical")
	cmd.flags.StringVar(&cmd.from, "from", "1h", "Start time (e.g., 1h, 24h, 7d, or RFC3339 timestamp)")
	cmd.flags.StringVar(&cmd.to, "to", "now", "End time (RFC3339 timestamp or 'now')")
	cmd.flags.StringVar(&cmd.status, "status", "", "Filter by status: active, resolved")
	cmd.flags.StringVar(&cmd.anomType, "type", "", "Filter by anomaly type: metric, log, trace, infrastructure")
	cmd.flags.BoolVar(&cmd.jsonOut, "json", false, "Output as JSON")

	return cmd
}

// Name returns the command name
func (c *AnomaliesCommand) Name() string {
	return "anomalies"
}

// Description returns the command description
func (c *AnomaliesCommand) Description() string {
	return "Detect anomalies across metrics, logs, traces, and infrastructure"
}

// Run executes the anomalies command
func (c *AnomaliesCommand) Run(args []string) error {
	if err := c.flags.Parse(args); err != nil {
		return err
	}

	ddClient, err := client.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	switch c.action {
	case "list":
		return c.listAnomalies(ddClient)
	case "metrics":
		return c.detectMetricAnomalies(ddClient)
	case "logs":
		return c.detectLogAnomalies(ddClient)
	case "traces":
		return c.detectTraceAnomalies(ddClient)
	case "infrastructure":
		return c.detectInfrastructureAnomalies(ddClient)
	case "search":
		return c.searchAnomalies(ddClient)
	default:
		return fmt.Errorf("unknown action: %s", c.action)
	}
}

// listAnomalies lists recent anomalies from Watchdog
func (c *AnomaliesCommand) listAnomalies(ddClient *client.Client) error {
	from, to, err := c.parseTimeRange()
	if err != nil {
		return fmt.Errorf("invalid time range: %w", err)
	}

	params := map[string]string{
		"from": fmt.Sprintf("%d", from.Unix()),
		"to":   fmt.Sprintf("%d", to.Unix()),
	}

	if c.service != "" {
		params["service"] = c.service
	}

	resp, err := ddClient.GetWatchdogAnomalies(params)
	if err != nil {
		return fmt.Errorf("failed to get anomalies: %w", err)
	}

	anomalies, err := c.parseAnomalies(resp)
	if err != nil {
		return fmt.Errorf("failed to parse anomalies: %w", err)
	}

	// Filter by severity if specified
	if c.severity != "" {
		filtered := []Anomaly{}
		for _, a := range anomalies {
			if strings.EqualFold(a.Severity, c.severity) {
				filtered = append(filtered, a)
			}
		}
		anomalies = filtered
	}

	// Filter by status if specified
	if c.status != "" {
		filtered := []Anomaly{}
		for _, a := range anomalies {
			if strings.EqualFold(a.Status, c.status) {
				filtered = append(filtered, a)
			}
		}
		anomalies = filtered
	}

	// Filter by type if specified
	if c.anomType != "" {
		filtered := []Anomaly{}
		for _, a := range anomalies {
			if strings.EqualFold(a.Type, c.anomType) {
				filtered = append(filtered, a)
			}
		}
		anomalies = filtered
	}

	if c.jsonOut {
		jsonData, err := json.MarshalIndent(anomalies, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}

	c.displayAnomalies(anomalies)
	return nil
}

// detectMetricAnomalies detects anomalies in metrics
func (c *AnomaliesCommand) detectMetricAnomalies(ddClient *client.Client) error {
	if c.metric == "" {
		return fmt.Errorf("--metric flag is required for metric anomaly detection")
	}

	from, to, err := c.parseTimeRange()
	if err != nil {
		return fmt.Errorf("invalid time range: %w", err)
	}

	query := map[string]interface{}{
		"metric": c.metric,
		"from":   from.Unix(),
		"to":     to.Unix(),
	}

	if c.service != "" {
		query["service"] = c.service
	}

	resp, err := ddClient.DetectMetricAnomalies(query)
	if err != nil {
		return fmt.Errorf("failed to detect metric anomalies: %w", err)
	}

	anomalies, err := c.parseMetricAnomalies(resp)
	if err != nil {
		return fmt.Errorf("failed to parse metric anomalies: %w", err)
	}

	if c.jsonOut {
		jsonData, err := json.MarshalIndent(anomalies, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}

	c.displayMetricAnomalies(anomalies)
	return nil
}

// detectLogAnomalies detects anomalies in log patterns
func (c *AnomaliesCommand) detectLogAnomalies(ddClient *client.Client) error {
	from, to, err := c.parseTimeRange()
	if err != nil {
		return fmt.Errorf("invalid time range: %w", err)
	}

	query := map[string]interface{}{
		"from": from.Unix() * 1000,
		"to":   to.Unix() * 1000,
	}

	if c.service != "" {
		query["filter"] = map[string]interface{}{
			"query": fmt.Sprintf("service:%s", c.service),
		}
	}

	resp, err := ddClient.DetectLogAnomalies(query)
	if err != nil {
		return fmt.Errorf("failed to detect log anomalies: %w", err)
	}

	anomalies, err := c.parseLogAnomalies(resp)
	if err != nil {
		return fmt.Errorf("failed to parse log anomalies: %w", err)
	}

	if c.jsonOut {
		jsonData, err := json.MarshalIndent(anomalies, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}

	c.displayLogAnomalies(anomalies)
	return nil
}

// detectTraceAnomalies detects anomalies in APM traces
func (c *AnomaliesCommand) detectTraceAnomalies(ddClient *client.Client) error {
	from, to, err := c.parseTimeRange()
	if err != nil {
		return fmt.Errorf("invalid time range: %w", err)
	}

	query := map[string]interface{}{
		"from": from.Unix() * 1000,
		"to":   to.Unix() * 1000,
	}

	if c.service != "" {
		query["filter"] = map[string]interface{}{
			"query": fmt.Sprintf("service:%s", c.service),
		}
	}

	// Detect latency anomalies
	query["compute"] = map[string]interface{}{
		"aggregation": "avg",
		"metric":      "@duration",
	}

	resp, err := ddClient.AggregateSpans(query)
	if err != nil {
		return fmt.Errorf("failed to detect trace anomalies: %w", err)
	}

	anomalies, err := c.parseTraceAnomalies(resp)
	if err != nil {
		return fmt.Errorf("failed to parse trace anomalies: %w", err)
	}

	if c.jsonOut {
		jsonData, err := json.MarshalIndent(anomalies, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}

	c.displayTraceAnomalies(anomalies)
	return nil
}

// detectInfrastructureAnomalies detects infrastructure anomalies
func (c *AnomaliesCommand) detectInfrastructureAnomalies(ddClient *client.Client) error {
	from, to, err := c.parseTimeRange()
	if err != nil {
		return fmt.Errorf("invalid time range: %w", err)
	}

	// Query for infrastructure metrics
	metrics := []string{
		"system.cpu.user",
		"system.mem.used",
		"system.disk.used",
		"system.net.bytes_rcvd",
	}

	var allAnomalies []InfrastructureAnomaly

	for _, metric := range metrics {
		query := map[string]interface{}{
			"metric": metric,
			"from":   from.Unix(),
			"to":     to.Unix(),
		}

		resp, err := ddClient.DetectMetricAnomalies(query)
		if err != nil {
			continue // Skip failed queries
		}

		anomalies, err := c.parseInfrastructureAnomalies(resp, metric)
		if err != nil {
			continue
		}

		allAnomalies = append(allAnomalies, anomalies...)
	}

	if c.jsonOut {
		jsonData, err := json.MarshalIndent(allAnomalies, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}

	c.displayInfrastructureAnomalies(allAnomalies)
	return nil
}

// searchAnomalies searches anomalies with filters
func (c *AnomaliesCommand) searchAnomalies(ddClient *client.Client) error {
	// Search is essentially the same as list with all filters applied
	return c.listAnomalies(ddClient)
}

// parseAnomalies parses Watchdog anomalies response
func (c *AnomaliesCommand) parseAnomalies(data []byte) ([]Anomaly, error) {
	// In a real implementation, parse actual Datadog API response
	// For now, create sample data structure

	var response struct {
		Data []struct {
			ID         string `json:"id"`
			Type       string `json:"type"`
			Attributes struct {
				Title       string    `json:"title"`
				Description string    `json:"description"`
				Severity    string    `json:"severity"`
				Status      string    `json:"status"`
				Service     string    `json:"service"`
				StartTime   time.Time `json:"start_time"`
				EndTime     time.Time `json:"end_time,omitempty"`
				Impact      string    `json:"impact"`
				Confidence  float64   `json:"confidence"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		// Return empty if parsing fails (API might return different structure)
		return []Anomaly{}, nil
	}

	anomalies := make([]Anomaly, len(response.Data))
	for i, item := range response.Data {
		anomalies[i] = Anomaly{
			ID:          item.ID,
			Type:        item.Type,
			Severity:    item.Attributes.Severity,
			Service:     item.Attributes.Service,
			Description: item.Attributes.Description,
			DetectedAt:  item.Attributes.StartTime,
			StartTime:   item.Attributes.StartTime,
			EndTime:     item.Attributes.EndTime,
			Impact:      item.Attributes.Impact,
			Confidence:  item.Attributes.Confidence,
			Status:      item.Attributes.Status,
		}
	}

	return anomalies, nil
}

// parseMetricAnomalies parses metric anomaly detection response
func (c *AnomaliesCommand) parseMetricAnomalies(data []byte) ([]MetricAnomaly, error) {
	// In a real implementation, parse actual anomaly detection response
	// For now, return placeholder structure
	var anomalies []MetricAnomaly

	// NOTE: Placeholder for actual API response parsing
	// Real implementation would analyze metric data points and detect deviations
	// Currently returns empty results (mock data mode)

	return anomalies, nil
}

// parseLogAnomalies parses log anomaly detection response
func (c *AnomaliesCommand) parseLogAnomalies(data []byte) ([]LogAnomaly, error) {
	// In a real implementation, parse log pattern analysis
	var anomalies []LogAnomaly

	// NOTE: Placeholder for actual API response parsing
	// Real implementation would detect log pattern changes and spikes
	// Currently returns empty results (mock data mode)

	return anomalies, nil
}

// parseTraceAnomalies parses trace anomaly detection response
func (c *AnomaliesCommand) parseTraceAnomalies(data []byte) ([]TraceAnomaly, error) {
	// In a real implementation, parse trace analytics for anomalies
	var anomalies []TraceAnomaly

	// NOTE: Placeholder for actual API response parsing
	// Real implementation would detect latency and error rate anomalies
	// Currently returns empty results (mock data mode)

	return anomalies, nil
}

// parseInfrastructureAnomalies parses infrastructure anomaly detection response
func (c *AnomaliesCommand) parseInfrastructureAnomalies(data []byte, metric string) ([]InfrastructureAnomaly, error) {
	// In a real implementation, parse infrastructure metric anomalies
	var anomalies []InfrastructureAnomaly

	// NOTE: Placeholder for actual API response parsing
	// Real implementation would detect CPU, memory, disk, network anomalies
	// Currently returns empty results (mock data mode)

	return anomalies, nil
}

// displayAnomalies displays anomalies in human-readable format
func (c *AnomaliesCommand) displayAnomalies(anomalies []Anomaly) {
	if len(anomalies) == 0 {
		fmt.Println("No anomalies detected in the specified time range.")
		return
	}

	fmt.Printf("\nAnomalies Detected: %d\n", len(anomalies))
	fmt.Println(strings.Repeat("=", 80))

	// Calculate analytics
	analytics := c.calculateAnalytics(anomalies)

	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Active: %d | Resolved: %d\n", analytics.ActiveAnomalies, analytics.ResolvedAnomalies)
	fmt.Printf("  By Severity: ")
	for severity, count := range analytics.BySeverity {
		fmt.Printf("%s=%d ", severity, count)
	}
	fmt.Println()

	fmt.Printf("  By Type: ")
	for anomType, count := range analytics.ByType {
		fmt.Printf("%s=%d ", anomType, count)
	}
	fmt.Println()

	// Display individual anomalies
	for i, anom := range anomalies {
		if i >= 20 {
			fmt.Printf("\n... and %d more anomalies (use --json for full output)\n", len(anomalies)-20)
			break
		}

		fmt.Printf("[%s] %s - %s\n", anom.Severity, anom.Type, anom.Status)
		if anom.Service != "" {
			fmt.Printf("  Service: %s\n", anom.Service)
		}
		if anom.Metric != "" {
			fmt.Printf("  Metric: %s\n", anom.Metric)
		}
		fmt.Printf("  Description: %s\n", anom.Description)
		fmt.Printf("  Detected: %s\n", anom.DetectedAt.Format(time.RFC3339))
		fmt.Printf("  Impact: %s\n", anom.Impact)
		fmt.Printf("  Confidence: %.1f%%\n", anom.Confidence*100)

		if anom.Deviation != 0 {
			fmt.Printf("  Deviation: %.1f%% from baseline\n", anom.Deviation)
		}

		fmt.Println()
	}
}

// displayMetricAnomalies displays metric anomalies
func (c *AnomaliesCommand) displayMetricAnomalies(anomalies []MetricAnomaly) {
	if len(anomalies) == 0 {
		fmt.Printf("No anomalies detected for metric: %s\n", c.metric)
		return
	}

	fmt.Printf("\nMetric Anomalies for %s: %d\n", c.metric, len(anomalies))
	fmt.Println(strings.Repeat("=", 80))

	for i, anom := range anomalies {
		if i >= 10 {
			fmt.Printf("\n... and %d more anomalies (use --json for full output)\n", len(anomalies)-10)
			break
		}

		fmt.Printf("[%s] %s\n", anom.Severity, anom.Timestamp.Format(time.RFC3339))
		fmt.Printf("  Value: %.2f (Baseline: %.2f)\n", anom.Value, anom.Baseline)
		fmt.Printf("  Bounds: [%.2f - %.2f]\n", anom.LowerBound, anom.UpperBound)
		fmt.Printf("  Deviation: %.1f%%\n", anom.Deviation)
		fmt.Printf("  Confidence: %.1f%%\n", anom.ConfidenceLevel*100)
		fmt.Println()
	}
}

// displayLogAnomalies displays log anomalies
func (c *AnomaliesCommand) displayLogAnomalies(anomalies []LogAnomaly) {
	if len(anomalies) == 0 {
		fmt.Println("No log pattern anomalies detected.")
		return
	}

	fmt.Printf("\nLog Pattern Anomalies: %d\n", len(anomalies))
	fmt.Println(strings.Repeat("=", 80))

	for i, anom := range anomalies {
		if i >= 10 {
			fmt.Printf("\n... and %d more anomalies (use --json for full output)\n", len(anomalies)-10)
			break
		}

		fmt.Printf("[%s] %s\n", anom.Severity, anom.Service)
		fmt.Printf("  Pattern: %s\n", anom.Pattern)
		fmt.Printf("  Count: %d (Baseline: %d)\n", anom.Count, anom.BaselineCount)
		fmt.Printf("  Increase: %.1f%%\n", anom.Increase)
		fmt.Printf("  Detected: %s\n", anom.DetectedAt.Format(time.RFC3339))
		if anom.Example != "" {
			fmt.Printf("  Example: %s\n", anom.Example)
		}
		fmt.Println()
	}
}

// displayTraceAnomalies displays trace anomalies
func (c *AnomaliesCommand) displayTraceAnomalies(anomalies []TraceAnomaly) {
	if len(anomalies) == 0 {
		fmt.Println("No trace anomalies detected.")
		return
	}

	fmt.Printf("\nTrace Anomalies: %d\n", len(anomalies))
	fmt.Println(strings.Repeat("=", 80))

	for i, anom := range anomalies {
		if i >= 10 {
			fmt.Printf("\n... and %d more anomalies (use --json for full output)\n", len(anomalies)-10)
			break
		}

		fmt.Printf("[%s] %s - %s\n", anom.Severity, anom.Service, anom.AnomalyType)
		fmt.Printf("  Operation: %s\n", anom.Operation)
		fmt.Printf("  Metric: %s\n", anom.Metric)
		fmt.Printf("  Current: %.2f | Baseline: %.2f\n", anom.CurrentValue, anom.BaselineValue)
		fmt.Printf("  Deviation: %.1f%%\n", anom.Deviation)
		fmt.Printf("  Detected: %s\n", anom.DetectedAt.Format(time.RFC3339))
		fmt.Println()
	}
}

// displayInfrastructureAnomalies displays infrastructure anomalies
func (c *AnomaliesCommand) displayInfrastructureAnomalies(anomalies []InfrastructureAnomaly) {
	if len(anomalies) == 0 {
		fmt.Println("No infrastructure anomalies detected.")
		return
	}

	fmt.Printf("\nInfrastructure Anomalies: %d\n", len(anomalies))
	fmt.Println(strings.Repeat("=", 80))

	for i, anom := range anomalies {
		if i >= 10 {
			fmt.Printf("\n... and %d more anomalies (use --json for full output)\n", len(anomalies)-10)
			break
		}

		fmt.Printf("[%s] %s - %s\n", anom.Severity, anom.Host, anom.Metric)
		fmt.Printf("  Current: %.2f | Baseline: %.2f | Threshold: %.2f\n",
			anom.CurrentValue, anom.BaselineValue, anom.Threshold)
		fmt.Printf("  Impact: %s\n", anom.Impact)
		fmt.Printf("  Detected: %s\n", anom.DetectedAt.Format(time.RFC3339))
		fmt.Println()
	}
}

// calculateAnalytics calculates anomaly analytics
func (c *AnomaliesCommand) calculateAnalytics(anomalies []Anomaly) AnomalyAnalytics {
	analytics := AnomalyAnalytics{
		TotalAnomalies: len(anomalies),
		BySeverity:     make(map[string]int),
		ByType:         make(map[string]int),
		ByService:      make(map[string]int),
	}

	for _, anom := range anomalies {
		analytics.BySeverity[anom.Severity]++
		analytics.ByType[anom.Type]++

		if anom.Service != "" {
			analytics.ByService[anom.Service]++
		}

		if anom.Status == "active" {
			analytics.ActiveAnomalies++
		} else {
			analytics.ResolvedAnomalies++
		}
	}

	// Find top impacted services
	type serviceCount struct {
		service string
		count   int
	}
	var services []serviceCount
	for svc, count := range analytics.ByService {
		services = append(services, serviceCount{svc, count})
	}

	// Simple sort by count (top 5)
	for i := 0; i < len(services) && i < 5; i++ {
		for j := i + 1; j < len(services); j++ {
			if services[j].count > services[i].count {
				services[i], services[j] = services[j], services[i]
			}
		}
		analytics.TopImpacted = append(analytics.TopImpacted, services[i].service)
	}

	return analytics
}

// parseTimeRange parses from and to time flags
func (c *AnomaliesCommand) parseTimeRange() (time.Time, time.Time, error) {
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

// parseTime parses a time string (relative or RFC3339)
func (c *AnomaliesCommand) parseTime(timeStr string) (time.Time, error) {
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
	if strings.HasSuffix(timeStr, "m") && !strings.Contains(timeStr, "-") {
		mins := strings.TrimSuffix(timeStr, "m")
		var m int
		fmt.Sscanf(mins, "%d", &m)
		return time.Now().Add(-time.Duration(m) * time.Minute), nil
	}

	// Try RFC3339
	if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("invalid time format: %s (use '1h', '24h', '7d' or RFC3339)", timeStr)
}

// Help displays help information
func (c *AnomaliesCommand) Help() {
	help := `
dd anomalies - Detect anomalies across metrics, logs, traces, and infrastructure

Usage:
  dd anomalies --action <action> [options]

Actions:
  list             List recent anomalies detected by Watchdog
  metrics          Detect metric anomalies (requires --metric)
  logs             Detect log pattern anomalies
  traces           Detect APM trace anomalies (latency, error rate)
  infrastructure   Detect infrastructure anomalies (CPU, memory, disk)
  search           Search anomalies with filters (alias for list)

Options:
  --action         Action to perform (default: list)
  --service        Filter by service name
  --metric         Metric name for metric anomaly detection
  --severity       Filter by severity: low, medium, high, critical
  --from           Start time (default: 1h) - e.g., 1h, 24h, 7d, or RFC3339
  --to             End time (default: now) - RFC3339 timestamp or 'now'
  --status         Filter by status: active, resolved
  --type           Filter by type: metric, log, trace, infrastructure
  --json           Output as JSON

Examples:
  # List all recent anomalies
  dd anomalies --action list --from 24h

  # Find critical anomalies for a service
  dd anomalies --action search --service payment-service --severity critical

  # Detect metric anomalies
  dd anomalies --action metrics --metric system.cpu.user --from 1h

  # Detect trace anomalies
  dd anomalies --action traces --service api-gateway --from 2h

  # Detect infrastructure anomalies
  dd anomalies --action infrastructure --from 30m

  # List only active anomalies
  dd anomalies --action list --status active --from 7d

Integration Workflows:
  # Anomaly investigation
  dd anomalies --action list --severity high
  dd spans --action latency --service affected-service
  dd service-map --action graph --service affected-service

  # Service health check
  dd anomalies --action search --service checkout --status active
  dd health --service checkout
  dd monitors --action list --service checkout
`
	fmt.Println(help)
}
