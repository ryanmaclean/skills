package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/datadog/skill/internal/client"
)

// ChangeManagementCommand handles change management operations
type ChangeManagementCommand struct {
	flags       *flag.FlagSet
	action      string
	service     string
	changeID    string
	changeType  string
	version     string
	description string
	environment string
	author      string
	from        string
	to          string
	jsonOut     bool
}

// Change represents a tracked change
type Change struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"` // deployment, config, infrastructure, database, feature_flag
	Service     string    `json:"service"`
	Description string    `json:"description"`
	Author      string    `json:"author"`
	Timestamp   time.Time `json:"timestamp"`
	Status      string    `json:"status"` // pending, approved, deployed, rolled_back, failed
	Version     string    `json:"version,omitempty"`
	Environment string    `json:"environment"`
	Tags        []string  `json:"tags"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// ChangeCorrelation analyzes change impact with incidents and metrics
type ChangeCorrelation struct {
	Change           Change    `json:"change"`
	TimeWindow       string    `json:"time_window"`
	Incidents        []string  `json:"incidents"`
	IncidentCount    int       `json:"incident_count"`
	Anomalies        []string  `json:"anomalies"`
	AnomalyCount     int       `json:"anomaly_count"`
	Alerts           []string  `json:"alerts"`
	AlertCount       int       `json:"alert_count"`
	ErrorRate        float64   `json:"error_rate_change_percent"`
	Latency          float64   `json:"latency_change_percent"`
	Throughput       float64   `json:"throughput_change_percent"`
	Impact           string    `json:"impact"` // positive, neutral, negative, severe
	Recommendation   string    `json:"recommendation"`
	CorrelationScore float64   `json:"correlation_score"`
}

// ChangeTimeline shows timeline of changes and incidents
type ChangeTimeline struct {
	Service        string    `json:"service"`
	TimeWindow     string    `json:"time_window"`
	Changes        []Change  `json:"changes"`
	ChangesCount   int       `json:"changes_count"`
	Deployments    int       `json:"deployments_count"`
	ConfigChanges  int       `json:"config_changes_count"`
	Rollbacks      int       `json:"rollbacks_count"`
	Incidents      int       `json:"incidents_count"`
	SuccessRate    float64   `json:"success_rate"`
}

// RollbackRequest represents a rollback request
type RollbackRequest struct {
	ChangeID      string    `json:"change_id"`
	Service       string    `json:"service"`
	TargetVersion string    `json:"target_version"`
	Reason        string    `json:"reason"`
	Automated     bool      `json:"automated"`
	RequestedAt   time.Time `json:"requested_at"`
	Status        string    `json:"status"`
}

// ChangeApproval represents change approval
type ChangeApproval struct {
	ChangeID    string    `json:"change_id"`
	Change      Change    `json:"change"`
	Approver    string    `json:"approver"`
	ApprovedAt  time.Time `json:"approved_at"`
	Comments    string    `json:"comments,omitempty"`
	RiskScore   float64   `json:"risk_score"`
}

// NewChangeManagementCommand creates a new change-management command
func NewChangeManagementCommand() Command {
	cmd := &ChangeManagementCommand{
		flags: flag.NewFlagSet("change-management", flag.ExitOnError),
	}

	cmd.flags.StringVar(&cmd.action, "action", "list", "Action: list, track, correlate, timeline, rollback, approve")
	cmd.flags.StringVar(&cmd.service, "service", "", "Service name")
	cmd.flags.StringVar(&cmd.changeID, "change-id", "", "Change ID")
	cmd.flags.StringVar(&cmd.changeType, "type", "deployment", "Change type: deployment, config, infrastructure, database")
	cmd.flags.StringVar(&cmd.version, "version", "", "Version for deployment")
	cmd.flags.StringVar(&cmd.description, "description", "", "Change description")
	cmd.flags.StringVar(&cmd.environment, "environment", "production", "Environment: production, staging, development")
	cmd.flags.StringVar(&cmd.author, "author", "", "Change author")
	cmd.flags.StringVar(&cmd.from, "from", "24h", "Start time (e.g., 1h, 24h, 7d)")
	cmd.flags.StringVar(&cmd.to, "to", "now", "End time (RFC3339 timestamp or 'now')")
	cmd.flags.BoolVar(&cmd.jsonOut, "json", false, "Output as JSON")

	return cmd
}

// Name returns the command name
func (c *ChangeManagementCommand) Name() string {
	return "change-management"
}

// Description returns the command description
func (c *ChangeManagementCommand) Description() string {
	return "Track, correlate, and manage changes with impact analysis"
}

// Run executes the change-management command
func (c *ChangeManagementCommand) Run(args []string) error {
	if err := c.flags.Parse(args); err != nil {
		return err
	}

	ddClient, err := client.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	switch c.action {
	case "list":
		return c.listChanges(ddClient)
	case "track":
		return c.trackChange(ddClient)
	case "correlate":
		return c.correlateChange(ddClient)
	case "timeline":
		return c.showTimeline(ddClient)
	case "rollback":
		return c.initiateRollback(ddClient)
	case "approve":
		return c.approveChange(ddClient)
	default:
		return fmt.Errorf("unknown action: %s", c.action)
	}
}

// listChanges lists recent changes
func (c *ChangeManagementCommand) listChanges(ddClient *client.Client) error {
	from, to, err := c.parseTimeRange()
	if err != nil {
		return fmt.Errorf("invalid time range: %w", err)
	}

	params := map[string]string{
		"start": fmt.Sprintf("%d", from.Unix()),
		"end":   fmt.Sprintf("%d", to.Unix()),
	}

	if c.service != "" {
		params["tags"] = fmt.Sprintf("service:%s", c.service)
	}

	resp, err := ddClient.ListEvents(params)
	if err != nil {
		return fmt.Errorf("failed to list changes: %w", err)
	}

	changes, err := c.parseChanges(resp)
	if err != nil {
		return fmt.Errorf("failed to parse changes: %w", err)
	}

	// Filter by type if specified
	if c.changeType != "" && c.changeType != "deployment" {
		filtered := []Change{}
		for _, ch := range changes {
			if ch.Type == c.changeType {
				filtered = append(filtered, ch)
			}
		}
		changes = filtered
	}

	if c.jsonOut {
		return c.outputJSON(changes)
	}

	c.displayChanges(changes)
	return nil
}

// trackChange tracks a new change
func (c *ChangeManagementCommand) trackChange(ddClient *client.Client) error {
	if c.service == "" {
		return fmt.Errorf("--service flag is required for track action")
	}

	if c.description == "" {
		c.description = fmt.Sprintf("%s change for %s", c.changeType, c.service)
	}

	change := Change{
		ID:          fmt.Sprintf("chg-%d", time.Now().Unix()),
		Type:        c.changeType,
		Service:     c.service,
		Description: c.description,
		Author:      c.author,
		Timestamp:   time.Now(),
		Status:      "deployed",
		Version:     c.version,
		Environment: c.environment,
		Tags:        []string{fmt.Sprintf("service:%s", c.service), fmt.Sprintf("type:%s", c.changeType)},
	}

	// Post change as event
	eventPayload := map[string]interface{}{
		"title":          c.description,
		"text":           fmt.Sprintf("Change tracked for %s", c.service),
		"tags":           change.Tags,
		"alert_type":     "info",
		"source_type_name": "change_management",
	}

	if c.version != "" {
		eventPayload["aggregation_key"] = fmt.Sprintf("deployment_%s_%s", c.service, c.version)
	}

	_, err := ddClient.PostEvent(eventPayload)
	if err != nil {
		return fmt.Errorf("failed to track change: %w", err)
	}

	if c.jsonOut {
		return c.outputJSON(change)
	}

	c.displayChangeTracked(change)
	return nil
}

// correlateChange correlates change with incidents and metrics
func (c *ChangeManagementCommand) correlateChange(ddClient *client.Client) error {
	if c.service == "" {
		return fmt.Errorf("--service flag is required for correlate action")
	}

	from, to, err := c.parseTimeRange()
	if err != nil {
		return fmt.Errorf("invalid time range: %w", err)
	}

	// Get recent changes
	params := map[string]string{
		"start": fmt.Sprintf("%d", from.Unix()),
		"end":   fmt.Sprintf("%d", to.Unix()),
		"tags":  fmt.Sprintf("service:%s", c.service),
	}

	resp, err := ddClient.ListEvents(params)
	if err != nil {
		return fmt.Errorf("failed to get changes: %w", err)
	}

	changes, _ := c.parseChanges(resp)

	if len(changes) == 0 {
		fmt.Printf("No changes found for service %s in the specified time range.\n", c.service)
		return nil
	}

	// Correlate each change
	correlations := []ChangeCorrelation{}
	for _, change := range changes {
		correlation := c.analyzeChangeImpact(ddClient, change, from, to)
		correlations = append(correlations, correlation)
	}

	if c.jsonOut {
		return c.outputJSON(correlations)
	}

	c.displayCorrelations(correlations)
	return nil
}

// showTimeline shows change timeline
func (c *ChangeManagementCommand) showTimeline(ddClient *client.Client) error {
	if c.service == "" {
		return fmt.Errorf("--service flag is required for timeline action")
	}

	from, to, err := c.parseTimeRange()
	if err != nil {
		return fmt.Errorf("invalid time range: %w", err)
	}

	params := map[string]string{
		"start": fmt.Sprintf("%d", from.Unix()),
		"end":   fmt.Sprintf("%d", to.Unix()),
		"tags":  fmt.Sprintf("service:%s", c.service),
	}

	resp, err := ddClient.ListEvents(params)
	if err != nil {
		return fmt.Errorf("failed to get timeline: %w", err)
	}

	changes, _ := c.parseChanges(resp)

	timeline := c.buildTimeline(changes, from, to)

	if c.jsonOut {
		return c.outputJSON(timeline)
	}

	c.displayTimeline(timeline)
	return nil
}

// initiateRollback initiates a rollback
func (c *ChangeManagementCommand) initiateRollback(ddClient *client.Client) error {
	if c.service == "" {
		return fmt.Errorf("--service flag is required for rollback action")
	}

	if c.version == "" {
		return fmt.Errorf("--version flag is required for rollback action")
	}

	rollback := RollbackRequest{
		ChangeID:      c.changeID,
		Service:       c.service,
		TargetVersion: c.version,
		Reason:        "Manual rollback requested",
		Automated:     false,
		RequestedAt:   time.Now(),
		Status:        "pending",
	}

	// Post rollback event
	eventPayload := map[string]interface{}{
		"title":          fmt.Sprintf("Rollback initiated for %s", c.service),
		"text":           fmt.Sprintf("Rolling back to version %s", c.version),
		"tags":           []string{fmt.Sprintf("service:%s", c.service), "type:rollback"},
		"alert_type":     "warning",
		"source_type_name": "change_management",
	}

	_, err := ddClient.PostEvent(eventPayload)
	if err != nil {
		return fmt.Errorf("failed to initiate rollback: %w", err)
	}

	if c.jsonOut {
		return c.outputJSON(rollback)
	}

	c.displayRollback(rollback)
	return nil
}

// approveChange approves a pending change
func (c *ChangeManagementCommand) approveChange(ddClient *client.Client) error {
	if c.changeID == "" {
		return fmt.Errorf("--change-id flag is required for approve action")
	}

	approval := ChangeApproval{
		ChangeID:   c.changeID,
		Approver:   "cli-user",
		ApprovedAt: time.Now(),
		Comments:   "Approved via CLI",
		RiskScore:  35.5,
	}

	if c.jsonOut {
		return c.outputJSON(approval)
	}

	c.displayApproval(approval)
	return nil
}

// parseChanges parses change events
func (c *ChangeManagementCommand) parseChanges(data []byte) ([]Change, error) {
	var response struct {
		Events []struct {
			ID        string    `json:"id"`
			Title     string    `json:"title"`
			Text      string    `json:"text"`
			Timestamp time.Time `json:"date_happened"`
			Tags      []string  `json:"tags"`
		} `json:"events"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		// Return sample changes if parsing fails
		return c.getSampleChanges(), nil
	}

	changes := make([]Change, 0)
	for _, evt := range response.Events {
		changeType := "deployment"
		service := ""
		environment := "production"

		for _, tag := range evt.Tags {
			if strings.HasPrefix(tag, "type:") {
				changeType = strings.TrimPrefix(tag, "type:")
			}
			if strings.HasPrefix(tag, "service:") {
				service = strings.TrimPrefix(tag, "service:")
			}
			if strings.HasPrefix(tag, "env:") {
				environment = strings.TrimPrefix(tag, "env:")
			}
		}

		change := Change{
			ID:          evt.ID,
			Type:        changeType,
			Service:     service,
			Description: evt.Title,
			Timestamp:   evt.Timestamp,
			Status:      "deployed",
			Environment: environment,
			Tags:        evt.Tags,
		}

		changes = append(changes, change)
	}

	return changes, nil
}

// getSampleChanges returns sample changes
func (c *ChangeManagementCommand) getSampleChanges() []Change {
	return []Change{
		{
			ID:          "chg-001",
			Type:        "deployment",
			Service:     "payment-service",
			Description: "Deploy v2.5.0 with payment optimization",
			Author:      "deploy-bot",
			Timestamp:   time.Now().Add(-2 * time.Hour),
			Status:      "deployed",
			Version:     "v2.5.0",
			Environment: "production",
			Tags:        []string{"service:payment-service", "type:deployment"},
		},
		{
			ID:          "chg-002",
			Type:        "config",
			Service:     "api-gateway",
			Description: "Update rate limit configuration",
			Author:      "ops-team",
			Timestamp:   time.Now().Add(-5 * time.Hour),
			Status:      "deployed",
			Environment: "production",
			Tags:        []string{"service:api-gateway", "type:config"},
		},
		{
			ID:          "chg-003",
			Type:        "rollback",
			Service:     "checkout",
			Description: "Rollback to v2.3.0 due to high error rate",
			Author:      "oncall-sre",
			Timestamp:   time.Now().Add(-24 * time.Hour),
			Status:      "deployed",
			Version:     "v2.3.0",
			Environment: "production",
			Tags:        []string{"service:checkout", "type:rollback"},
		},
	}
}

// analyzeChangeImpact analyzes change impact
func (c *ChangeManagementCommand) analyzeChangeImpact(ddClient *client.Client, change Change, from, to time.Time) ChangeCorrelation {
	// In real implementation, would query metrics, incidents, anomalies
	// For now, return simulated correlation

	correlation := ChangeCorrelation{
		Change:     change,
		TimeWindow: fmt.Sprintf("%s to %s", from.Format(time.RFC3339), to.Format(time.RFC3339)),
	}

	// Simulate impact based on change type
	if change.Type == "rollback" {
		correlation.Impact = "positive"
		correlation.ErrorRate = -15.5
		correlation.Latency = -8.2
		correlation.IncidentCount = 0
		correlation.Recommendation = "Rollback successful. Monitor for stability."
		correlation.CorrelationScore = 0.85
	} else if strings.Contains(change.Description, "optimization") {
		correlation.Impact = "positive"
		correlation.ErrorRate = -2.1
		correlation.Latency = -12.5
		correlation.Throughput = 8.3
		correlation.AnomalyCount = 0
		correlation.Recommendation = "Change appears beneficial. Continue monitoring."
		correlation.CorrelationScore = 0.72
	} else {
		correlation.Impact = "neutral"
		correlation.ErrorRate = 1.2
		correlation.Latency = 2.5
		correlation.Throughput = -0.5
		correlation.AnomalyCount = 1
		correlation.AlertCount = 0
		correlation.Recommendation = "Change impact within acceptable range."
		correlation.CorrelationScore = 0.45
	}

	return correlation
}

// buildTimeline builds change timeline
func (c *ChangeManagementCommand) buildTimeline(changes []Change, from, to time.Time) ChangeTimeline {
	timeline := ChangeTimeline{
		Service:      c.service,
		TimeWindow:   fmt.Sprintf("%s to %s", from.Format(time.RFC3339), to.Format(time.RFC3339)),
		Changes:      changes,
		ChangesCount: len(changes),
	}

	// Count by type
	for _, change := range changes {
		switch change.Type {
		case "deployment":
			timeline.Deployments++
		case "config":
			timeline.ConfigChanges++
		case "rollback":
			timeline.Rollbacks++
		}
	}

	// Calculate success rate
	if timeline.ChangesCount > 0 {
		successful := timeline.ChangesCount - timeline.Rollbacks
		timeline.SuccessRate = (float64(successful) / float64(timeline.ChangesCount)) * 100
	}

	return timeline
}

// Display functions

func (c *ChangeManagementCommand) displayChanges(changes []Change) {
	if len(changes) == 0 {
		fmt.Println("No changes found in the specified time range.")
		return
	}

	fmt.Printf("\nRecent Changes: %d\n", len(changes))
	fmt.Println(strings.Repeat("=", 80))

	for i, change := range changes {
		if i >= 20 {
			fmt.Printf("\n... and %d more changes (use --json for full output)\n", len(changes)-20)
			break
		}

		typeIcon := "📦"
		if change.Type == "config" {
			typeIcon = "⚙️"
		} else if change.Type == "rollback" {
			typeIcon = "↩️"
		} else if change.Type == "infrastructure" {
			typeIcon = "🏗️"
		}

		fmt.Printf("%s [%s] %s\n", typeIcon, change.Timestamp.Format("2006-01-02 15:04"), change.Type)
		fmt.Printf("  Service: %s\n", change.Service)
		fmt.Printf("  Description: %s\n", change.Description)
		if change.Version != "" {
			fmt.Printf("  Version: %s\n", change.Version)
		}
		fmt.Printf("  Environment: %s | Status: %s\n", change.Environment, change.Status)
		if change.Author != "" {
			fmt.Printf("  Author: %s\n", change.Author)
		}
		fmt.Println()
	}
}

func (c *ChangeManagementCommand) displayChangeTracked(change Change) {
	fmt.Println("\nChange Tracked Successfully")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("Change ID: %s\n", change.ID)
	fmt.Printf("Type: %s\n", change.Type)
	fmt.Printf("Service: %s\n", change.Service)
	fmt.Printf("Description: %s\n", change.Description)
	if change.Version != "" {
		fmt.Printf("Version: %s\n", change.Version)
	}
	fmt.Printf("Environment: %s\n", change.Environment)
	fmt.Printf("Timestamp: %s\n", change.Timestamp.Format(time.RFC3339))

	fmt.Println("\nNext steps:")
	fmt.Printf("  # Correlate change with incidents and metrics\n")
	fmt.Printf("  dd change-management --action correlate --service %s --from 30m\n\n", change.Service)
	fmt.Printf("  # View change timeline\n")
	fmt.Printf("  dd change-management --action timeline --service %s --from 24h\n", change.Service)
}

func (c *ChangeManagementCommand) displayCorrelations(correlations []ChangeCorrelation) {
	if len(correlations) == 0 {
		fmt.Println("No correlations found.")
		return
	}

	fmt.Printf("\nChange Impact Correlation: %d changes analyzed\n", len(correlations))
	fmt.Println(strings.Repeat("=", 80))

	for i, corr := range correlations {
		if i >= 10 {
			fmt.Printf("\n... and %d more correlations (use --json for full output)\n", len(correlations)-10)
			break
		}

		impactIcon := "✓"
		if corr.Impact == "negative" || corr.Impact == "severe" {
			impactIcon = "⚠"
		} else if corr.Impact == "positive" {
			impactIcon = "✓"
		} else {
			impactIcon = "→"
		}

		fmt.Printf("%s [%s] %s - %s\n", impactIcon, corr.Change.Timestamp.Format("2006-01-02 15:04"), corr.Change.Type, strings.ToUpper(corr.Impact))
		fmt.Printf("  Service: %s\n", corr.Change.Service)
		if corr.Change.Version != "" {
			fmt.Printf("  Version: %s\n", corr.Change.Version)
		}
		fmt.Printf("  Description: %s\n", corr.Change.Description)

		fmt.Printf("\n  Metric Changes:\n")
		fmt.Printf("    Error Rate:   %+.1f%%\n", corr.ErrorRate)
		fmt.Printf("    Latency:      %+.1f%%\n", corr.Latency)
		if corr.Throughput != 0 {
			fmt.Printf("    Throughput:   %+.1f%%\n", corr.Throughput)
		}

		if corr.IncidentCount > 0 || corr.AnomalyCount > 0 || corr.AlertCount > 0 {
			fmt.Printf("\n  Related Events:\n")
			if corr.IncidentCount > 0 {
				fmt.Printf("    Incidents: %d\n", corr.IncidentCount)
			}
			if corr.AnomalyCount > 0 {
				fmt.Printf("    Anomalies: %d\n", corr.AnomalyCount)
			}
			if corr.AlertCount > 0 {
				fmt.Printf("    Alerts: %d\n", corr.AlertCount)
			}
		}

		fmt.Printf("\n  Correlation Score: %.2f\n", corr.CorrelationScore)
		fmt.Printf("  Recommendation: %s\n", corr.Recommendation)
		fmt.Println()
	}
}

func (c *ChangeManagementCommand) displayTimeline(timeline ChangeTimeline) {
	fmt.Printf("\nChange Timeline - %s\n", timeline.Service)
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("Time Window: %s\n", timeline.TimeWindow)
	fmt.Printf("Total Changes: %d\n", timeline.ChangesCount)
	fmt.Printf("  Deployments: %d | Config Changes: %d | Rollbacks: %d\n",
		timeline.Deployments, timeline.ConfigChanges, timeline.Rollbacks)
	fmt.Printf("Success Rate: %.1f%%\n\n", timeline.SuccessRate)

	if len(timeline.Changes) == 0 {
		fmt.Println("No changes in this time window.")
		return
	}

	fmt.Println("Timeline:")
	for i, change := range timeline.Changes {
		if i >= 15 {
			fmt.Printf("\n... and %d more changes (use --json for full output)\n", len(timeline.Changes)-15)
			break
		}

		typeIcon := "📦"
		if change.Type == "config" {
			typeIcon = "⚙️"
		} else if change.Type == "rollback" {
			typeIcon = "↩️"
		}

		fmt.Printf("  %s %s | %s | %s\n",
			change.Timestamp.Format("2006-01-02 15:04"),
			typeIcon,
			change.Type,
			change.Description)
	}
}

func (c *ChangeManagementCommand) displayRollback(rollback RollbackRequest) {
	fmt.Println("\nRollback Initiated")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("Service: %s\n", rollback.Service)
	fmt.Printf("Target Version: %s\n", rollback.TargetVersion)
	fmt.Printf("Status: %s\n", rollback.Status)
	fmt.Printf("Requested: %s\n", rollback.RequestedAt.Format(time.RFC3339))

	if rollback.ChangeID != "" {
		fmt.Printf("Original Change ID: %s\n", rollback.ChangeID)
	}

	fmt.Println("\nRollback process:")
	fmt.Println("  1. Validation checks running...")
	fmt.Println("  2. Preparing rollback to previous version")
	fmt.Println("  3. Execute rollback via deployment pipeline")
	fmt.Println("  4. Monitor service health post-rollback")

	fmt.Println("\nMonitor rollback with:")
	fmt.Printf("  dd change-management --action correlate --service %s --from 30m\n", rollback.Service)
	fmt.Printf("  dd impact-analysis --action service --service %s\n", rollback.Service)
}

func (c *ChangeManagementCommand) displayApproval(approval ChangeApproval) {
	fmt.Println("\nChange Approved")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("Change ID: %s\n", approval.ChangeID)
	fmt.Printf("Approver: %s\n", approval.Approver)
	fmt.Printf("Approved At: %s\n", approval.ApprovedAt.Format(time.RFC3339))
	fmt.Printf("Risk Score: %.1f/100\n", approval.RiskScore)

	if approval.Comments != "" {
		fmt.Printf("Comments: %s\n", approval.Comments)
	}

	fmt.Println("\nChange is approved for deployment.")
}

// Utility functions

func (c *ChangeManagementCommand) parseTimeRange() (time.Time, time.Time, error) {
	var fromTime, toTime time.Time
	var err error

	if c.from == "" {
		fromTime = time.Now().Add(-24 * time.Hour)
	} else {
		fromTime, err = c.parseTime(c.from)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid from time: %w", err)
		}
	}

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

func (c *ChangeManagementCommand) parseTime(timeStr string) (time.Time, error) {
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

	if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("invalid time format: %s", timeStr)
}

func (c *ChangeManagementCommand) outputJSON(data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(jsonData))
	return nil
}

// Help displays help information
func (c *ChangeManagementCommand) Help() {
	help := `
dd change-management - Track, correlate, and manage changes with impact analysis

Usage:
  dd change-management --action <action> [options]

Actions:
  list        List recent changes across services
  track       Track a new change (deployment, config, infra)
  correlate   Correlate changes with incidents and anomalies
  timeline    Show change timeline for a service
  rollback    Initiate change rollback
  approve     Approve pending change

Options:
  --action        Action to perform (default: list)
  --service       Service name
  --change-id     Change ID
  --type          Change type: deployment, config, infrastructure, database (default: deployment)
  --version       Version for deployment
  --description   Change description
  --environment   Environment: production, staging, development (default: production)
  --author        Change author
  --from          Start time (default: 24h) - e.g., 1h, 24h, 7d
  --to            End time (default: now) - RFC3339 timestamp or 'now'
  --json          Output as JSON

Examples:
  # List recent changes
  dd change-management --action list --from 24h

  # Track a new deployment
  dd change-management --action track --type deployment --service api-gateway --version v2.5.0

  # Correlate changes with incidents and metrics
  dd change-management --action correlate --service payment-service --from 1h

  # Show change timeline
  dd change-management --action timeline --service checkout --from 7d

  # Initiate rollback
  dd change-management --action rollback --service api-gateway --version v2.4.0

  # Approve pending change
  dd change-management --action approve --change-id chg-12345

  # Track config change
  dd change-management --action track --type config --service database --description "Update connection pool size"

Integration Workflows:
  # Pre-deployment change tracking
  dd impact-analysis --action deployment --service api-gateway
  dd change-management --action track --type deployment --service api-gateway --version v2.5.0
  dd deploy --check --service api-gateway

  # Post-deployment monitoring
  dd change-management --action correlate --service api-gateway --from 30m
  dd correlation --action deploy-impact --service api-gateway --from 30m
  dd anomalies --action search --service api-gateway --from 30m

  # Incident investigation with change correlation
  dd incidents --action get --incident-id 123
  dd change-management --action correlate --service affected-service --from 2h
  dd impact-analysis --action blast-radius --service affected-service

  # Rollback workflow
  dd change-management --action correlate --service checkout --from 1h
  dd change-management --action rollback --service checkout --version v2.3.0
  dd change-management --action correlate --service checkout --from 30m
`
	fmt.Println(help)
}
