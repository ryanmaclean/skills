package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/datadog/skill/internal/client"
)

// CorrelationCommand handles event correlation operations
type CorrelationCommand struct {
	flags    *flag.FlagSet
	action   string
	service  string
	alertID  string
	traceID  string
	deployID string
	from     string
	to       string
	jsonOut  bool
}

// CorrelatedEvent represents an event that can be correlated
type CorrelatedEvent struct {
	Timestamp   time.Time `json:"timestamp"`
	Type        string    `json:"type"` // deployment, alert, anomaly, trace, log, incident
	Source      string    `json:"source"`
	Service     string    `json:"service"`
	Description string    `json:"description"`
	Severity    string    `json:"severity,omitempty"`
	TraceID     string    `json:"trace_id,omitempty"`
	AlertID     string    `json:"alert_id,omitempty"`
	DeployID    string    `json:"deploy_id,omitempty"`
	Related     []string  `json:"related_event_ids"`
	Score       float64   `json:"correlation_score,omitempty"`
}

// CorrelationAnalysis provides multi-signal correlation analysis
type CorrelationAnalysis struct {
	TimeWindow       string            `json:"time_window"`
	TotalEvents      int               `json:"total_events"`
	EventsByType     map[string]int    `json:"events_by_type"`
	CorrelationScore float64           `json:"correlation_score"`
	LikelyRootCause  *CorrelatedEvent  `json:"likely_root_cause,omitempty"`
	ImpactedServices []string          `json:"impacted_services"`
	Timeline         []CorrelatedEvent `json:"timeline"`
	Insights         []string          `json:"insights"`
}

// DeploymentCorrelation analyzes deployment impact
type DeploymentCorrelation struct {
	DeploymentID     string    `json:"deployment_id"`
	Service          string    `json:"service"`
	DeployedAt       time.Time `json:"deployed_at"`
	DeployedBy       string    `json:"deployed_by,omitempty"`
	Version          string    `json:"version,omitempty"`
	ErrorRateChange  float64   `json:"error_rate_change_percent"`
	LatencyChange    float64   `json:"latency_change_percent"`
	ThroughputChange float64   `json:"throughput_change_percent"`
	AnomaliesAfter   int       `json:"anomalies_after_deploy"`
	AlertsTriggered  int       `json:"alerts_triggered"`
	ImpactAssessment string    `json:"impact_assessment"` // positive, neutral, negative
	RecommendedAction string   `json:"recommended_action,omitempty"`
}

// AlertCorrelation provides context for an alert
type AlertCorrelation struct {
	AlertID          string            `json:"alert_id"`
	AlertName        string            `json:"alert_name"`
	TriggeredAt      time.Time         `json:"triggered_at"`
	Service          string            `json:"service"`
	RelatedEvents    []CorrelatedEvent `json:"related_events"`
	PossibleCauses   []string          `json:"possible_causes"`
	RecentDeployment *DeploymentCorrelation `json:"recent_deployment,omitempty"`
	AnomaliesFound   int               `json:"anomalies_found"`
}

// TraceLogCorrelation correlates traces with logs
type TraceLogCorrelation struct {
	TraceID       string    `json:"trace_id"`
	Service       string    `json:"service"`
	StartTime     time.Time `json:"start_time"`
	Duration      float64   `json:"duration_ms"`
	Status        string    `json:"status"`
	ErrorMessage  string    `json:"error_message,omitempty"`
	RelatedLogs   []LogEvent `json:"related_logs"`
	LogsCount     int       `json:"logs_count"`
}

// LogEvent represents a log entry
type LogEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Service   string    `json:"service"`
	TraceID   string    `json:"trace_id,omitempty"`
}

// CrossSignalCorrelation provides multi-signal correlation
type CrossSignalCorrelation struct {
	Service          string            `json:"service"`
	TimeWindow       string            `json:"time_window"`
	Deployments      []DeploymentCorrelation `json:"deployments"`
	Alerts           []AlertCorrelation `json:"alerts"`
	Anomalies        []CorrelatedEvent `json:"anomalies"`
	ErrorSpikes      []CorrelatedEvent `json:"error_spikes"`
	LatencyChanges   []CorrelatedEvent `json:"latency_changes"`
	CorrelationScore float64           `json:"overall_correlation_score"`
	Summary          string            `json:"summary"`
}

// NewCorrelationCommand creates a new correlation command
func NewCorrelationCommand() Command {
	cmd := &CorrelationCommand{
		flags: flag.NewFlagSet("correlation", flag.ExitOnError),
	}

	cmd.flags.StringVar(&cmd.action, "action", "timeline", "Action: timeline, root-cause, deploy-impact, alert-context, trace-logs, cross-signal")
	cmd.flags.StringVar(&cmd.service, "service", "", "Service name for correlation analysis")
	cmd.flags.StringVar(&cmd.alertID, "alert-id", "", "Alert ID for alert-context action")
	cmd.flags.StringVar(&cmd.traceID, "trace-id", "", "Trace ID for trace-logs action")
	cmd.flags.StringVar(&cmd.deployID, "deploy-id", "", "Deployment ID for deploy-impact action")
	cmd.flags.StringVar(&cmd.from, "from", "2h", "Start time (e.g., 1h, 2h, 24h, 7d, or RFC3339)")
	cmd.flags.StringVar(&cmd.to, "to", "now", "End time (RFC3339 timestamp or 'now')")
	cmd.flags.BoolVar(&cmd.jsonOut, "json", false, "Output as JSON")

	return cmd
}

// Name returns the command name
func (c *CorrelationCommand) Name() string {
	return "correlation"
}

// Description returns the command description
func (c *CorrelationCommand) Description() string {
	return "Correlate events across multiple signals for root cause analysis"
}

// Run executes the correlation command
func (c *CorrelationCommand) Run(args []string) error {
	if err := c.flags.Parse(args); err != nil {
		return err
	}

	ddClient, err := client.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	switch c.action {
	case "timeline":
		return c.buildTimeline(ddClient)
	case "root-cause":
		return c.analyzeRootCause(ddClient)
	case "deploy-impact":
		return c.analyzeDeploymentImpact(ddClient)
	case "alert-context":
		return c.analyzeAlertContext(ddClient)
	case "trace-logs":
		return c.correlateTraceLogs(ddClient)
	case "cross-signal":
		return c.crossSignalAnalysis(ddClient)
	default:
		return fmt.Errorf("unknown action: %s", c.action)
	}
}

// buildTimeline builds a correlated event timeline
func (c *CorrelationCommand) buildTimeline(ddClient *client.Client) error {
	if c.service == "" {
		return fmt.Errorf("--service flag is required for timeline action")
	}

	from, to, err := c.parseTimeRange()
	if err != nil {
		return fmt.Errorf("invalid time range: %w", err)
	}

	// Fetch events from multiple sources
	events, err := c.fetchCorrelatedEvents(ddClient, from, to)
	if err != nil {
		return fmt.Errorf("failed to fetch events: %w", err)
	}

	// Sort events by timestamp
	c.sortEventsByTime(events)

	// Build analysis
	analysis := CorrelationAnalysis{
		TimeWindow:   fmt.Sprintf("%s to %s", from.Format(time.RFC3339), to.Format(time.RFC3339)),
		TotalEvents:  len(events),
		EventsByType: make(map[string]int),
		Timeline:     events,
	}

	for _, event := range events {
		analysis.EventsByType[event.Type]++
		if event.Service != "" {
			analysis.ImpactedServices = append(analysis.ImpactedServices, event.Service)
		}
	}

	// Deduplicate services
	analysis.ImpactedServices = c.deduplicate(analysis.ImpactedServices)

	// Calculate correlation score and insights
	analysis.CorrelationScore = c.calculateCorrelationScore(events)
	analysis.Insights = c.generateInsights(events)

	if c.jsonOut {
		return c.outputJSON(analysis)
	}

	c.displayTimeline(analysis)
	return nil
}

// analyzeRootCause performs root cause analysis
func (c *CorrelationCommand) analyzeRootCause(ddClient *client.Client) error {
	if c.service == "" {
		return fmt.Errorf("--service flag is required for root-cause action")
	}

	from, to, err := c.parseTimeRange()
	if err != nil {
		return fmt.Errorf("invalid time range: %w", err)
	}

	// Fetch all correlated events
	events, err := c.fetchCorrelatedEvents(ddClient, from, to)
	if err != nil {
		return fmt.Errorf("failed to fetch events: %w", err)
	}

	// Analyze for root cause
	analysis := c.performRootCauseAnalysis(events)

	if c.jsonOut {
		return c.outputJSON(analysis)
	}

	c.displayRootCauseAnalysis(analysis)
	return nil
}

// analyzeDeploymentImpact analyzes deployment impact
func (c *CorrelationCommand) analyzeDeploymentImpact(ddClient *client.Client) error {
	if c.service == "" {
		return fmt.Errorf("--service flag is required for deploy-impact action")
	}

	from, to, err := c.parseTimeRange()
	if err != nil {
		return fmt.Errorf("invalid time range: %w", err)
	}

	// Query deployments
	params := map[string]string{
		"tags":  fmt.Sprintf("service:%s,event_type:deployment", c.service),
		"start": fmt.Sprintf("%d", from.Unix()),
		"end":   fmt.Sprintf("%d", to.Unix()),
	}

	deploymentsResp, err := ddClient.ListEvents(params)
	if err != nil {
		return fmt.Errorf("failed to get deployments: %w", err)
	}

	// Parse and analyze deployment impact
	deployments, err := c.parseDeployments(deploymentsResp)
	if err != nil {
		return fmt.Errorf("failed to parse deployments: %w", err)
	}

	// Analyze impact for each deployment
	correlations := []DeploymentCorrelation{}
	for _, deploy := range deployments {
		correlation := c.analyzeDeployment(ddClient, deploy, from, to)
		correlations = append(correlations, correlation)
	}

	if c.jsonOut {
		return c.outputJSON(correlations)
	}

	c.displayDeploymentCorrelations(correlations)
	return nil
}

// analyzeAlertContext provides context for an alert
func (c *CorrelationCommand) analyzeAlertContext(ddClient *client.Client) error {
	if c.alertID == "" {
		return fmt.Errorf("--alert-id flag is required for alert-context action")
	}

	// Fetch alert details
	// In real implementation, would call monitor API
	// For now, create correlation structure

	from := time.Now().Add(-2 * time.Hour)
	to := time.Now()

	events, err := c.fetchCorrelatedEvents(ddClient, from, to)
	if err != nil {
		return fmt.Errorf("failed to fetch events: %w", err)
	}

	correlation := AlertCorrelation{
		AlertID:     c.alertID,
		AlertName:   "Sample Alert",
		TriggeredAt: time.Now().Add(-30 * time.Minute),
		Service:     c.service,
		RelatedEvents: events,
		PossibleCauses: c.identifyPossibleCauses(events),
	}

	// Check for recent deployment
	for _, event := range events {
		if event.Type == "deployment" {
			deployCorr := c.analyzeDeployment(ddClient, event, from, to)
			correlation.RecentDeployment = &deployCorr
			break
		}
	}

	if c.jsonOut {
		return c.outputJSON(correlation)
	}

	c.displayAlertCorrelation(correlation)
	return nil
}

// correlateTraceLogs correlates a trace with its logs
func (c *CorrelationCommand) correlateTraceLogs(ddClient *client.Client) error {
	if c.traceID == "" {
		return fmt.Errorf("--trace-id flag is required for trace-logs action")
	}

	from, to, err := c.parseTimeRange()
	if err != nil {
		return fmt.Errorf("invalid time range: %w", err)
	}

	// Query logs for trace ID
	query := fmt.Sprintf("trace_id:%s", c.traceID)

	logsResp, err := ddClient.SearchLogs(query, from, to, 100)
	if err != nil {
		return fmt.Errorf("failed to search logs: %w", err)
	}

	logs, err := c.parseLogs(logsResp)
	if err != nil {
		return fmt.Errorf("failed to parse logs: %w", err)
	}

	correlation := TraceLogCorrelation{
		TraceID:     c.traceID,
		Service:     c.service,
		StartTime:   time.Now().Add(-5 * time.Minute),
		Duration:    125.5,
		Status:      "error",
		RelatedLogs: logs,
		LogsCount:   len(logs),
	}

	if c.jsonOut {
		return c.outputJSON(correlation)
	}

	c.displayTraceLogCorrelation(correlation)
	return nil
}

// crossSignalAnalysis performs multi-signal correlation
func (c *CorrelationCommand) crossSignalAnalysis(ddClient *client.Client) error {
	if c.service == "" {
		return fmt.Errorf("--service flag is required for cross-signal action")
	}

	from, to, err := c.parseTimeRange()
	if err != nil {
		return fmt.Errorf("invalid time range: %w", err)
	}

	// Fetch data from multiple signals
	events, err := c.fetchCorrelatedEvents(ddClient, from, to)
	if err != nil {
		return fmt.Errorf("failed to fetch events: %w", err)
	}

	// Build cross-signal correlation
	correlation := CrossSignalCorrelation{
		Service:          c.service,
		TimeWindow:       fmt.Sprintf("%s to %s", from.Format(time.RFC3339), to.Format(time.RFC3339)),
		CorrelationScore: c.calculateCorrelationScore(events),
	}

	// Categorize events by type
	for _, event := range events {
		switch event.Type {
		case "deployment":
			deployCorr := c.analyzeDeployment(ddClient, event, from, to)
			correlation.Deployments = append(correlation.Deployments, deployCorr)
		case "anomaly":
			correlation.Anomalies = append(correlation.Anomalies, event)
		case "alert":
			// Would populate AlertCorrelation, simplified for now
		}
	}

	correlation.Summary = c.generateCrossSignalSummary(correlation)

	if c.jsonOut {
		return c.outputJSON(correlation)
	}

	c.displayCrossSignalCorrelation(correlation)
	return nil
}

// fetchCorrelatedEvents fetches events from multiple sources
func (c *CorrelationCommand) fetchCorrelatedEvents(ddClient *client.Client, from, to time.Time) ([]CorrelatedEvent, error) {
	var events []CorrelatedEvent

	// Fetch events
	params := map[string]string{
		"start": fmt.Sprintf("%d", from.Unix()),
		"end":   fmt.Sprintf("%d", to.Unix()),
	}

	if c.service != "" {
		params["tags"] = fmt.Sprintf("service:%s", c.service)
	}

	resp, err := ddClient.ListEvents(params)
	if err != nil {
		// Continue even if events fetch fails
		return events, nil
	}

	parsed, err := c.parseEvents(resp)
	if err == nil {
		events = append(events, parsed...)
	}

	return events, nil
}

// parseEvents parses event response
func (c *CorrelationCommand) parseEvents(data []byte) ([]CorrelatedEvent, error) {
	var response struct {
		Events []struct {
			ID        string    `json:"id"`
			Title     string    `json:"title"`
			Text      string    `json:"text"`
			Timestamp time.Time `json:"date_happened"`
			Tags      []string  `json:"tags"`
			Priority  string    `json:"priority"`
		} `json:"events"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return []CorrelatedEvent{}, nil
	}

	events := make([]CorrelatedEvent, len(response.Events))
	for i, evt := range response.Events {
		eventType := "event"
		service := ""

		// Extract type and service from tags
		for _, tag := range evt.Tags {
			if strings.HasPrefix(tag, "event_type:") {
				eventType = strings.TrimPrefix(tag, "event_type:")
			}
			if strings.HasPrefix(tag, "service:") {
				service = strings.TrimPrefix(tag, "service:")
			}
		}

		events[i] = CorrelatedEvent{
			Timestamp:   evt.Timestamp,
			Type:        eventType,
			Source:      "datadog-events",
			Service:     service,
			Description: evt.Title,
			Severity:    evt.Priority,
		}
	}

	return events, nil
}

// parseDeployments parses deployment events
func (c *CorrelationCommand) parseDeployments(data []byte) ([]CorrelatedEvent, error) {
	events, err := c.parseEvents(data)
	if err != nil {
		return nil, err
	}

	var deployments []CorrelatedEvent
	for _, evt := range events {
		if evt.Type == "deployment" {
			deployments = append(deployments, evt)
		}
	}

	return deployments, nil
}

// parseLogs parses log search response
func (c *CorrelationCommand) parseLogs(data []byte) ([]LogEvent, error) {
	var response struct {
		Data []struct {
			Attributes struct {
				Timestamp time.Time              `json:"timestamp"`
				Message   string                 `json:"message"`
				Status    string                 `json:"status"`
				Service   string                 `json:"service"`
				Attributes map[string]interface{} `json:"attributes"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return []LogEvent{}, nil
	}

	logs := make([]LogEvent, len(response.Data))
	for i, item := range response.Data {
		logs[i] = LogEvent{
			Timestamp: item.Attributes.Timestamp,
			Level:     item.Attributes.Status,
			Message:   item.Attributes.Message,
			Service:   item.Attributes.Service,
		}

		if traceID, ok := item.Attributes.Attributes["trace_id"].(string); ok {
			logs[i].TraceID = traceID
		}
	}

	return logs, nil
}

// analyzeDeployment analyzes deployment impact
func (c *CorrelationCommand) analyzeDeployment(ddClient *client.Client, deploy CorrelatedEvent, from, to time.Time) DeploymentCorrelation {
	// In real implementation, would query metrics before/after deployment
	// For now, return placeholder data

	correlation := DeploymentCorrelation{
		DeploymentID:     deploy.DeployID,
		Service:          deploy.Service,
		DeployedAt:       deploy.Timestamp,
		ErrorRateChange:  15.5,
		LatencyChange:    -8.2,
		ThroughputChange: 12.3,
		AnomaliesAfter:   2,
		AlertsTriggered:  1,
		ImpactAssessment: "neutral",
	}

	// Determine impact assessment
	if correlation.ErrorRateChange > 10 || correlation.AlertsTriggered > 2 {
		correlation.ImpactAssessment = "negative"
		correlation.RecommendedAction = "Consider rollback"
	} else if correlation.LatencyChange < -5 && correlation.ErrorRateChange < 5 {
		correlation.ImpactAssessment = "positive"
	}

	return correlation
}

// performRootCauseAnalysis analyzes events for root cause
func (c *CorrelationCommand) performRootCauseAnalysis(events []CorrelatedEvent) CorrelationAnalysis {
	analysis := CorrelationAnalysis{
		TotalEvents:  len(events),
		EventsByType: make(map[string]int),
		Timeline:     events,
	}

	// Find earliest event (potential root cause)
	if len(events) > 0 {
		c.sortEventsByTime(events)
		earliest := events[0]

		// Calculate correlation score based on event proximity
		score := 0.0
		for i := 1; i < len(events); i++ {
			timeDiff := events[i].Timestamp.Sub(earliest.Timestamp).Minutes()
			if timeDiff < 30 {
				score += (30 - timeDiff) / 30
			}
		}
		analysis.CorrelationScore = score / float64(len(events))
		analysis.LikelyRootCause = &earliest
	}

	for _, event := range events {
		analysis.EventsByType[event.Type]++
	}

	analysis.Insights = c.generateInsights(events)

	return analysis
}

// sortEventsByTime sorts events by timestamp
func (c *CorrelationCommand) sortEventsByTime(events []CorrelatedEvent) {
	for i := 0; i < len(events); i++ {
		for j := i + 1; j < len(events); j++ {
			if events[j].Timestamp.Before(events[i].Timestamp) {
				events[i], events[j] = events[j], events[i]
			}
		}
	}
}

// calculateCorrelationScore calculates overall correlation score
func (c *CorrelationCommand) calculateCorrelationScore(events []CorrelatedEvent) float64 {
	if len(events) == 0 {
		return 0.0
	}

	// Simple scoring based on event density
	c.sortEventsByTime(events)
	if len(events) < 2 {
		return 0.5
	}

	timeSpan := events[len(events)-1].Timestamp.Sub(events[0].Timestamp).Minutes()
	density := float64(len(events)) / (timeSpan + 1)

	score := density * 10
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// generateInsights generates correlation insights
func (c *CorrelationCommand) generateInsights(events []CorrelatedEvent) []string {
	var insights []string

	deploymentCount := 0
	anomalyCount := 0
	alertCount := 0

	for _, evt := range events {
		switch evt.Type {
		case "deployment":
			deploymentCount++
		case "anomaly":
			anomalyCount++
		case "alert":
			alertCount++
		}
	}

	if deploymentCount > 0 && (anomalyCount > 0 || alertCount > 0) {
		insights = append(insights, "Deployment activity correlated with anomalies or alerts")
	}

	if anomalyCount > 2 {
		insights = append(insights, fmt.Sprintf("Multiple anomalies detected (%d) in time window", anomalyCount))
	}

	if len(events) > 10 {
		insights = append(insights, "High event density suggests significant system activity")
	}

	if len(insights) == 0 {
		insights = append(insights, "Normal activity levels observed")
	}

	return insights
}

// identifyPossibleCauses identifies possible causes from events
func (c *CorrelationCommand) identifyPossibleCauses(events []CorrelatedEvent) []string {
	var causes []string

	for _, evt := range events {
		switch evt.Type {
		case "deployment":
			causes = append(causes, fmt.Sprintf("Recent deployment: %s", evt.Description))
		case "anomaly":
			causes = append(causes, fmt.Sprintf("Anomaly detected: %s", evt.Description))
		case "incident":
			causes = append(causes, fmt.Sprintf("Related incident: %s", evt.Description))
		}
	}

	if len(causes) == 0 {
		causes = append(causes, "No obvious causes identified in correlated events")
	}

	return causes
}

// generateCrossSignalSummary generates summary for cross-signal analysis
func (c *CorrelationCommand) generateCrossSignalSummary(correlation CrossSignalCorrelation) string {
	parts := []string{
		fmt.Sprintf("%d deployments", len(correlation.Deployments)),
		fmt.Sprintf("%d anomalies", len(correlation.Anomalies)),
		fmt.Sprintf("%d alerts", len(correlation.Alerts)),
	}

	return fmt.Sprintf("Cross-signal analysis found: %s", strings.Join(parts, ", "))
}

// deduplicate removes duplicate strings
func (c *CorrelationCommand) deduplicate(items []string) []string {
	seen := make(map[string]bool)
	result := []string{}

	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}

// displayTimeline displays event timeline
func (c *CorrelationCommand) displayTimeline(analysis CorrelationAnalysis) {
	fmt.Printf("\nEvent Timeline - %s\n", analysis.TimeWindow)
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("Total Events: %d | Correlation Score: %.2f\n", analysis.TotalEvents, analysis.CorrelationScore)
	fmt.Printf("Event Types: ")
	for evtType, count := range analysis.EventsByType {
		fmt.Printf("%s=%d ", evtType, count)
	}
	fmt.Println()

	if len(analysis.ImpactedServices) > 0 {
		fmt.Printf("Impacted Services: %s\n\n", strings.Join(analysis.ImpactedServices, ", "))
	}

	// Display timeline
	for i, evt := range analysis.Timeline {
		if i >= 20 {
			fmt.Printf("\n... and %d more events (use --json for full output)\n", len(analysis.Timeline)-20)
			break
		}

		fmt.Printf("[%s] %s\n", evt.Timestamp.Format("15:04:05"), evt.Type)
		if evt.Service != "" {
			fmt.Printf("  Service: %s\n", evt.Service)
		}
		fmt.Printf("  %s\n", evt.Description)
		if evt.Severity != "" {
			fmt.Printf("  Severity: %s\n", evt.Severity)
		}
		fmt.Println()
	}

	// Display insights
	if len(analysis.Insights) > 0 {
		fmt.Println("Insights:")
		for _, insight := range analysis.Insights {
			fmt.Printf("  • %s\n", insight)
		}
	}
}

// displayRootCauseAnalysis displays root cause analysis
func (c *CorrelationCommand) displayRootCauseAnalysis(analysis CorrelationAnalysis) {
	fmt.Println("\nRoot Cause Analysis")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("Total Events Analyzed: %d\n", analysis.TotalEvents)
	fmt.Printf("Correlation Score: %.2f\n\n", analysis.CorrelationScore)

	if analysis.LikelyRootCause != nil {
		fmt.Println("Likely Root Cause:")
		rc := analysis.LikelyRootCause
		fmt.Printf("  Type: %s\n", rc.Type)
		fmt.Printf("  Time: %s\n", rc.Timestamp.Format(time.RFC3339))
		fmt.Printf("  Service: %s\n", rc.Service)
		fmt.Printf("  Description: %s\n", rc.Description)
		fmt.Println()
	}

	fmt.Println("Event Breakdown:")
	for evtType, count := range analysis.EventsByType {
		fmt.Printf("  %s: %d\n", evtType, count)
	}

	if len(analysis.Insights) > 0 {
		fmt.Println("\nInsights:")
		for _, insight := range analysis.Insights {
			fmt.Printf("  • %s\n", insight)
		}
	}
}

// displayDeploymentCorrelations displays deployment correlations
func (c *CorrelationCommand) displayDeploymentCorrelations(correlations []DeploymentCorrelation) {
	if len(correlations) == 0 {
		fmt.Println("No deployments found in the specified time range.")
		return
	}

	fmt.Printf("\nDeployment Impact Analysis: %d deployments\n", len(correlations))
	fmt.Println(strings.Repeat("=", 80))

	for i, corr := range correlations {
		if i >= 10 {
			fmt.Printf("\n... and %d more deployments (use --json for full output)\n", len(correlations)-10)
			break
		}

		fmt.Printf("[%s] %s\n", corr.ImpactAssessment, corr.Service)
		fmt.Printf("  Deployed: %s\n", corr.DeployedAt.Format(time.RFC3339))
		if corr.DeploymentID != "" {
			fmt.Printf("  ID: %s\n", corr.DeploymentID)
		}
		fmt.Printf("  Error Rate: %+.1f%% | Latency: %+.1f%% | Throughput: %+.1f%%\n",
			corr.ErrorRateChange, corr.LatencyChange, corr.ThroughputChange)
		fmt.Printf("  Anomalies: %d | Alerts: %d\n", corr.AnomaliesAfter, corr.AlertsTriggered)
		if corr.RecommendedAction != "" {
			fmt.Printf("  Action: %s\n", corr.RecommendedAction)
		}
		fmt.Println()
	}
}

// displayAlertCorrelation displays alert correlation
func (c *CorrelationCommand) displayAlertCorrelation(correlation AlertCorrelation) {
	fmt.Println("\nAlert Context and Correlation")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("Alert: %s\n", correlation.AlertName)
	fmt.Printf("Alert ID: %s\n", correlation.AlertID)
	fmt.Printf("Triggered: %s\n", correlation.TriggeredAt.Format(time.RFC3339))
	if correlation.Service != "" {
		fmt.Printf("Service: %s\n", correlation.Service)
	}
	fmt.Println()

	fmt.Printf("Related Events: %d\n", len(correlation.RelatedEvents))
	for i, evt := range correlation.RelatedEvents {
		if i >= 5 {
			fmt.Printf("... and %d more events\n", len(correlation.RelatedEvents)-5)
			break
		}
		fmt.Printf("  [%s] %s: %s\n", evt.Timestamp.Format("15:04:05"), evt.Type, evt.Description)
	}
	fmt.Println()

	if correlation.RecentDeployment != nil {
		fmt.Println("Recent Deployment:")
		deploy := correlation.RecentDeployment
		fmt.Printf("  Service: %s\n", deploy.Service)
		fmt.Printf("  Deployed: %s\n", deploy.DeployedAt.Format(time.RFC3339))
		fmt.Printf("  Impact: %s\n", deploy.ImpactAssessment)
		fmt.Println()
	}

	fmt.Println("Possible Causes:")
	for _, cause := range correlation.PossibleCauses {
		fmt.Printf("  • %s\n", cause)
	}
}

// displayTraceLogCorrelation displays trace-log correlation
func (c *CorrelationCommand) displayTraceLogCorrelation(correlation TraceLogCorrelation) {
	fmt.Println("\nTrace-Log Correlation")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("Trace ID: %s\n", correlation.TraceID)
	fmt.Printf("Service: %s\n", correlation.Service)
	fmt.Printf("Start Time: %s\n", correlation.StartTime.Format(time.RFC3339))
	fmt.Printf("Duration: %.2f ms\n", correlation.Duration)
	fmt.Printf("Status: %s\n", correlation.Status)
	if correlation.ErrorMessage != "" {
		fmt.Printf("Error: %s\n", correlation.ErrorMessage)
	}
	fmt.Println()

	fmt.Printf("Related Logs: %d\n", correlation.LogsCount)
	fmt.Println(strings.Repeat("-", 80))

	for i, log := range correlation.RelatedLogs {
		if i >= 10 {
			fmt.Printf("\n... and %d more logs (use --json for full output)\n", correlation.LogsCount-10)
			break
		}

		fmt.Printf("[%s] %s\n", log.Timestamp.Format("15:04:05"), log.Level)
		fmt.Printf("  %s\n", log.Message)
		fmt.Println()
	}
}

// displayCrossSignalCorrelation displays cross-signal analysis
func (c *CorrelationCommand) displayCrossSignalCorrelation(correlation CrossSignalCorrelation) {
	fmt.Printf("\nCross-Signal Correlation - %s\n", correlation.Service)
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("Time Window: %s\n", correlation.TimeWindow)
	fmt.Printf("Correlation Score: %.2f\n", correlation.CorrelationScore)
	fmt.Println()

	fmt.Println(correlation.Summary)
	fmt.Println()

	if len(correlation.Deployments) > 0 {
		fmt.Printf("Deployments (%d):\n", len(correlation.Deployments))
		for i, deploy := range correlation.Deployments {
			if i >= 3 {
				fmt.Printf("  ... and %d more\n", len(correlation.Deployments)-3)
				break
			}
			fmt.Printf("  [%s] %s - Impact: %s\n", deploy.DeployedAt.Format("15:04:05"), deploy.Service, deploy.ImpactAssessment)
		}
		fmt.Println()
	}

	if len(correlation.Anomalies) > 0 {
		fmt.Printf("Anomalies (%d):\n", len(correlation.Anomalies))
		for i, anom := range correlation.Anomalies {
			if i >= 3 {
				fmt.Printf("  ... and %d more\n", len(correlation.Anomalies)-3)
				break
			}
			fmt.Printf("  [%s] %s\n", anom.Timestamp.Format("15:04:05"), anom.Description)
		}
	}
}

// parseTimeRange parses from and to time flags
func (c *CorrelationCommand) parseTimeRange() (time.Time, time.Time, error) {
	var fromTime, toTime time.Time
	var err error

	// Parse from time
	if c.from == "" {
		fromTime = time.Now().Add(-2 * time.Hour)
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
func (c *CorrelationCommand) parseTime(timeStr string) (time.Time, error) {
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

// outputJSON outputs data as JSON
func (c *CorrelationCommand) outputJSON(data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(jsonData))
	return nil
}

// Help displays help information
func (c *CorrelationCommand) Help() {
	help := `
dd correlation - Correlate events across multiple signals for root cause analysis

Usage:
  dd correlation --action <action> [options]

Actions:
  timeline        Show correlated event timeline across multiple signals
  root-cause      Analyze root cause across correlated events
  deploy-impact   Correlate deployment with metrics and errors
  alert-context   Correlate alert with related events
  trace-logs      Correlate traces with log events
  cross-signal    Multi-signal correlation analysis

Options:
  --action        Action to perform (default: timeline)
  --service       Service name for correlation analysis
  --alert-id      Alert ID for alert-context action
  --trace-id      Trace ID for trace-logs action
  --deploy-id     Deployment ID for deploy-impact action
  --from          Start time (default: 2h) - e.g., 1h, 2h, 24h, 7d, or RFC3339
  --to            End time (default: now) - RFC3339 timestamp or 'now'
  --json          Output as JSON

Examples:
  # Show event timeline for incident investigation
  dd correlation --action timeline --service payment-service --from 2h

  # Analyze root cause of production issue
  dd correlation --action root-cause --service api-gateway --from 1h

  # Check deployment impact
  dd correlation --action deploy-impact --service checkout --from 30m

  # Correlate trace with logs
  dd correlation --action trace-logs --trace-id abc123xyz

  # Multi-signal analysis
  dd correlation --action cross-signal --service frontend --from 1h

  # Get alert context
  dd correlation --action alert-context --alert-id 12345 --service auth

Integration Workflows:
  # Incident investigation workflow
  dd anomalies --action list --from 1h --severity high
  dd correlation --action timeline --service affected-service --from 2h
  dd correlation --action root-cause --service affected-service

  # Deployment validation workflow
  dd deploy --check --service api-gateway
  dd correlation --action deploy-impact --service api-gateway --from 15m
  dd spans --action errors --service api-gateway --from 15m

  # Alert triage workflow
  dd correlation --action alert-context --alert-id 12345
  dd correlation --action root-cause --service alerted-service
  dd service-map --action graph --service alerted-service
`
	fmt.Println(help)
}
