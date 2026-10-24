package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/datadog/skill/internal/client"
)

// EventsCommand manages Datadog events
type EventsCommand struct {
	flags      *flag.FlagSet
	action     string
	eventID    string
	title      string
	text       string
	tags       string
	priority   string
	alertType  string
	aggregKey  string
	sourceType string
	start      string
	end        string
	sources    string
	query      string
	limit      int
	jsonOut    bool
}

// EventData represents a Datadog event
type EventData struct {
	ID           int64    `json:"id"`
	Title        string   `json:"title"`
	Text         string   `json:"text"`
	Tags         []string `json:"tags,omitempty"`
	Priority     string   `json:"priority,omitempty"`
	AlertType    string   `json:"alert_type,omitempty"`
	AggregKey    string   `json:"aggregation_key,omitempty"`
	SourceType   string   `json:"source_type,omitempty"`
	DateHappened int64    `json:"date_happened,omitempty"`
	DeviceName   string   `json:"device_name,omitempty"`
	Host         string   `json:"host,omitempty"`
	URL          string   `json:"url,omitempty"`
}

// EventsListResponse represents the list events API response
type EventsListResponse struct {
	Status string      `json:"status"`
	Events []EventData `json:"events,omitempty"`
}

// EventResponse represents a single event API response
type EventResponse struct {
	Status string    `json:"status"`
	Event  EventData `json:"event,omitempty"`
}

// NewEventsCommand creates a new events command instance
func NewEventsCommand() *EventsCommand {
	cmd := &EventsCommand{
		flags: flag.NewFlagSet("events", flag.ExitOnError),
	}

	cmd.flags.StringVar(&cmd.action, "action", "list", "Action to perform (list, get, post, query)")
	cmd.flags.StringVar(&cmd.eventID, "event-id", "", "Event ID for get action")
	cmd.flags.StringVar(&cmd.title, "title", "", "Event title (required for post)")
	cmd.flags.StringVar(&cmd.text, "text", "", "Event description (required for post)")
	cmd.flags.StringVar(&cmd.tags, "tags", "", "Comma-separated tags")
	cmd.flags.StringVar(&cmd.priority, "priority", "normal", "Event priority (normal, low)")
	cmd.flags.StringVar(&cmd.alertType, "alert-type", "info", "Alert type (error, warning, info, success)")
	cmd.flags.StringVar(&cmd.aggregKey, "aggregation-key", "", "Aggregation key for grouping")
	cmd.flags.StringVar(&cmd.sourceType, "source-type", "cli", "Source type name")
	cmd.flags.StringVar(&cmd.start, "start", "", "Start time (RFC3339 or Unix timestamp)")
	cmd.flags.StringVar(&cmd.end, "end", "", "End time (RFC3339 or Unix timestamp)")
	cmd.flags.StringVar(&cmd.sources, "sources", "", "Comma-separated source names")
	cmd.flags.StringVar(&cmd.query, "query", "", "Search query string")
	cmd.flags.IntVar(&cmd.limit, "limit", 100, "Maximum events to return")
	cmd.flags.BoolVar(&cmd.jsonOut, "json", false, "Output as JSON")

	return cmd
}

func (c *EventsCommand) Name() string {
	return "events"
}

func (c *EventsCommand) Description() string {
	return "Manage Datadog events"
}

func (c *EventsCommand) Run(args []string) error {
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

	ddClient, err := client.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Datadog client: %w", err)
	}

	switch c.action {
	case "list":
		return c.listEvents(ddClient)
	case "get":
		return c.getEvent(ddClient)
	case "post":
		return c.postEvent(ddClient)
	case "query":
		return c.queryEvents(ddClient)
	default:
		return fmt.Errorf("unknown action: %s (valid: list, get, post, query)", c.action)
	}
}

func (c *EventsCommand) listEvents(ddClient *client.Client) error {
	// Build query parameters
	params := make(map[string]string)

	// Time range (default: last 24 hours)
	if c.start == "" {
		params["start"] = strconv.FormatInt(time.Now().Add(-24*time.Hour).Unix(), 10)
	} else {
		startTime, err := c.parseTime(c.start)
		if err != nil {
			return fmt.Errorf("invalid start time: %w", err)
		}
		params["start"] = strconv.FormatInt(startTime.Unix(), 10)
	}

	if c.end == "" {
		params["end"] = strconv.FormatInt(time.Now().Unix(), 10)
	} else {
		endTime, err := c.parseTime(c.end)
		if err != nil {
			return fmt.Errorf("invalid end time: %w", err)
		}
		params["end"] = strconv.FormatInt(endTime.Unix(), 10)
	}

	// Add optional filters
	if c.priority != "" {
		params["priority"] = c.priority
	}
	if c.sources != "" {
		params["sources"] = c.sources
	}
	if c.tags != "" {
		params["tags"] = c.tags
	}

	data, err := ddClient.ListEvents(params)
	if err != nil {
		return fmt.Errorf("failed to list events: %w", err)
	}

	// Parse response
	var apiResp struct {
		Events []EventData `json:"events"`
	}
	if err := json.Unmarshal(data, &apiResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Limit results
	events := apiResp.Events
	if len(events) > c.limit {
		events = events[:c.limit]
	}

	// Format output
	if c.jsonOut {
		response := EventsListResponse{
			Status: "success",
			Events: events,
		}
		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}

	// Human-readable output
	fmt.Printf("Events (%d results)\n", len(events))
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println()

	for i, event := range events {
		if i > 0 {
			fmt.Println()
		}

		fmt.Printf("ID: %d\n", event.ID)
		fmt.Printf("Title: %s\n", event.Title)
		if event.Text != "" {
			fmt.Printf("Text: %s\n", event.Text)
		}
		if len(event.Tags) > 0 {
			fmt.Printf("Tags: %s\n", strings.Join(event.Tags, ", "))
		}
		if event.Priority != "" {
			fmt.Printf("Priority: %s\n", event.Priority)
		}
		if event.AlertType != "" {
			fmt.Printf("Alert Type: %s\n", event.AlertType)
		}
		if event.DateHappened > 0 {
			timestamp := time.Unix(event.DateHappened, 0)
			fmt.Printf("Time: %s\n", timestamp.Format("2006-01-02 15:04:05 MST"))
		}
		if event.Host != "" {
			fmt.Printf("Host: %s\n", event.Host)
		}
		if event.SourceType != "" {
			fmt.Printf("Source: %s\n", event.SourceType)
		}
	}

	if len(events) == 0 {
		fmt.Println("No events found in the specified time range.")
	}

	return nil
}

func (c *EventsCommand) getEvent(ddClient *client.Client) error {
	if c.eventID == "" {
		return fmt.Errorf("--event-id is required for get action")
	}

	data, err := ddClient.GetEvent(c.eventID)
	if err != nil {
		return fmt.Errorf("failed to get event: %w", err)
	}

	// Parse response
	var apiResp struct {
		Event EventData `json:"event"`
	}
	if err := json.Unmarshal(data, &apiResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	event := apiResp.Event

	// Format output
	if c.jsonOut {
		response := EventResponse{
			Status: "success",
			Event:  event,
		}
		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}

	// Human-readable output
	fmt.Println("Event Details")
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println()

	fmt.Printf("ID: %d\n", event.ID)
	fmt.Printf("Title: %s\n", event.Title)
	if event.Text != "" {
		fmt.Printf("Text: %s\n", event.Text)
	}
	if len(event.Tags) > 0 {
		fmt.Printf("Tags: %s\n", strings.Join(event.Tags, ", "))
	}
	if event.Priority != "" {
		fmt.Printf("Priority: %s\n", event.Priority)
	}
	if event.AlertType != "" {
		fmt.Printf("Alert Type: %s\n", event.AlertType)
	}
	if event.DateHappened > 0 {
		timestamp := time.Unix(event.DateHappened, 0)
		fmt.Printf("Time: %s\n", timestamp.Format("2006-01-02 15:04:05 MST"))
	}
	if event.Host != "" {
		fmt.Printf("Host: %s\n", event.Host)
	}
	if event.DeviceName != "" {
		fmt.Printf("Device: %s\n", event.DeviceName)
	}
	if event.SourceType != "" {
		fmt.Printf("Source Type: %s\n", event.SourceType)
	}
	if event.AggregKey != "" {
		fmt.Printf("Aggregation Key: %s\n", event.AggregKey)
	}
	if event.URL != "" {
		fmt.Printf("URL: %s\n", event.URL)
	}

	return nil
}

func (c *EventsCommand) postEvent(ddClient *client.Client) error {
	if c.title == "" {
		return fmt.Errorf("--title is required for post action")
	}
	if c.text == "" {
		return fmt.Errorf("--text is required for post action")
	}

	// Build event payload
	payload := map[string]interface{}{
		"title":      c.title,
		"text":       c.text,
		"priority":   c.priority,
		"alert_type": c.alertType,
	}

	// Add optional fields
	if c.tags != "" {
		payload["tags"] = strings.Split(c.tags, ",")
	}
	if c.aggregKey != "" {
		payload["aggregation_key"] = c.aggregKey
	}
	if c.sourceType != "" {
		payload["source_type_name"] = c.sourceType
	}

	// Post event
	data, err := ddClient.PostEvent(payload)
	if err != nil {
		return fmt.Errorf("failed to post event: %w", err)
	}

	// Parse response
	var apiResp struct {
		Status string    `json:"status"`
		Event  EventData `json:"event"`
	}
	if err := json.Unmarshal(data, &apiResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Format output
	if c.jsonOut {
		response := EventResponse{
			Status: "success",
			Event:  apiResp.Event,
		}
		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}

	// Human-readable output
	fmt.Println("Event Posted Successfully")
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println()

	fmt.Printf("✅ Event ID: %d\n", apiResp.Event.ID)
	fmt.Printf("Title: %s\n", c.title)
	fmt.Printf("Priority: %s\n", c.priority)
	fmt.Printf("Alert Type: %s\n", c.alertType)
	if c.tags != "" {
		fmt.Printf("Tags: %s\n", c.tags)
	}
	fmt.Println()
	fmt.Println("Event successfully posted to Datadog event stream.")

	return nil
}

func (c *EventsCommand) queryEvents(ddClient *client.Client) error {
	if c.query == "" {
		return fmt.Errorf("--query is required for query action")
	}

	// Build query parameters
	params := make(map[string]string)
	params["filter[query]"] = c.query

	// Time range (default: last 24 hours)
	if c.start == "" {
		params["filter[from]"] = time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
	} else {
		startTime, err := c.parseTime(c.start)
		if err != nil {
			return fmt.Errorf("invalid start time: %w", err)
		}
		params["filter[from]"] = startTime.Format(time.RFC3339)
	}

	if c.end == "" {
		params["filter[to]"] = time.Now().Format(time.RFC3339)
	} else {
		endTime, err := c.parseTime(c.end)
		if err != nil {
			return fmt.Errorf("invalid end time: %w", err)
		}
		params["filter[to]"] = endTime.Format(time.RFC3339)
	}

	data, err := ddClient.QueryEventsV2(params)
	if err != nil {
		return fmt.Errorf("failed to query events: %w", err)
	}

	// Parse response
	var apiResp struct {
		Data []struct {
			Attributes struct {
				Attributes map[string]interface{} `json:"attributes"`
				Message    string                 `json:"message"`
				Tags       []string               `json:"tags"`
				Timestamp  string                 `json:"timestamp"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &apiResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Format output
	if c.jsonOut {
		jsonData, err := json.MarshalIndent(apiResp, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}

	// Human-readable output
	fmt.Printf("Event Query Results (%d events)\n", len(apiResp.Data))
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println()

	for i, item := range apiResp.Data {
		if i > 0 {
			fmt.Println()
		}

		fmt.Printf("Message: %s\n", item.Attributes.Message)
		if len(item.Attributes.Tags) > 0 {
			fmt.Printf("Tags: %s\n", strings.Join(item.Attributes.Tags, ", "))
		}
		if item.Attributes.Timestamp != "" {
			fmt.Printf("Time: %s\n", item.Attributes.Timestamp)
		}
	}

	if len(apiResp.Data) == 0 {
		fmt.Println("No events found matching the query.")
	}

	return nil
}

func (c *EventsCommand) parseTime(timeStr string) (time.Time, error) {
	// Try Unix timestamp first
	if timestamp, err := strconv.ParseInt(timeStr, 10, 64); err == nil {
		return time.Unix(timestamp, 0), nil
	}

	// Try RFC3339
	if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
		return t, nil
	}

	// Try relative time (e.g., "1h", "24h", "7d")
	if strings.HasSuffix(timeStr, "h") {
		hours, err := strconv.Atoi(strings.TrimSuffix(timeStr, "h"))
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid time format: %s", timeStr)
		}
		return time.Now().Add(-time.Duration(hours) * time.Hour), nil
	}
	if strings.HasSuffix(timeStr, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(timeStr, "d"))
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid time format: %s", timeStr)
		}
		return time.Now().Add(-time.Duration(days) * 24 * time.Hour), nil
	}

	return time.Time{}, fmt.Errorf("invalid time format: %s (use RFC3339, Unix timestamp, or relative like '24h' or '7d')", timeStr)
}

func (c *EventsCommand) Help() {
	helpText := `dd events - Manage Datadog Events

DESCRIPTION:
  Query and post Datadog events for deployments, alerts, configuration changes,
  and custom notifications. Events provide context for metrics, traces, and logs.

USAGE:
  dd events --action <action> [options]

ACTIONS:
  list             List recent events with filtering
  get              Get specific event details
  post             Post custom event
  query            Advanced event search (v2 API)

EXAMPLES:
  # List events from last 24 hours (default)
  dd events --action list

  # List events from last 7 days
  dd events --action list --start 7d

  # List events with priority filter
  dd events --action list --priority normal

  # List events with tag filter
  dd events --action list --tags "env:prod,service:api"

  # Get specific event
  dd events --action get --event-id 1234567890

  # Post deployment event
  dd events --action post \
    --title "Deploy production v1.2.3" \
    --text "Deployed version 1.2.3 to production" \
    --tags "deploy,production,v1.2.3" \
    --priority normal \
    --alert-type info

  # Post error event
  dd events --action post \
    --title "Database migration failed" \
    --text "Migration script failed with exit code 1" \
    --tags "database,migration,error" \
    --priority normal \
    --alert-type error

  # Query events with search
  dd events --action query \
    --query "deploy AND production" \
    --start 7d

  # Get JSON output
  dd events --action list --json

OPTIONS:
  --action          Action to perform (list, get, post, query)
  --event-id        Event ID for get action
  --title           Event title (required for post)
  --text            Event description (required for post)
  --tags            Comma-separated tags
  --priority        Event priority (normal, low) - default: normal
  --alert-type      Alert type (error, warning, info, success) - default: info
  --aggregation-key Aggregation key for grouping related events
  --source-type     Source type name (default: cli)
  --start           Start time (RFC3339, Unix timestamp, or relative like '24h', '7d')
  --end             End time (RFC3339, Unix timestamp, or relative like '24h', '7d')
  --sources         Comma-separated source names to filter
  --query           Search query string (for query action)
  --limit           Maximum events to return (default: 100)
  --json            Output as JSON

TIME FORMATS:
  Relative:  1h (1 hour ago)
             24h (24 hours ago)
             7d (7 days ago)

  Absolute:  2026-01-23T10:00:00Z (RFC3339)
             1737626400 (Unix timestamp)

PRIORITY LEVELS:
  normal:    Standard event priority
  low:       Low priority event

ALERT TYPES:
  error:     Error condition
  warning:   Warning condition
  info:      Informational message (default)
  success:   Success notification

USE CASES:
  Deployment Tracking:
    - Post deployment events from CI/CD pipelines
    - Correlate deployments with error spikes
    - Track rollback events

  Incident Context:
    - Query events during incident timeframe
    - Link events to incident timeline
    - Post incident resolution events

  Configuration Changes:
    - Track infrastructure changes
    - Document database migrations
    - Record feature flag changes

  Custom Notifications:
    - Business metric milestones
    - Scheduled maintenance windows
    - System state changes

  Event Correlation:
    - View events alongside metrics in dashboards
    - Correlate events with APM traces
    - Link events to log patterns

INTEGRATION WITH OTHER COMMANDS:
  Deployment Workflow:
    1. Check safety: dd error-budgets --action status
    2. Post event:   dd events --action post --title "Deploy v1.2.3"
    3. Monitor:      dd health --service api --since-deploy
    4. Check alerts: dd events --action list --tags "alert:true"

  Incident Response:
    1. Query events: dd events --action list --start 2h --priority normal
    2. Check health: dd health --service api
    3. View logs:    dd logs --service api --since 2h
    4. Post update:  dd events --action post --title "Incident resolved"

  Monitoring:
    1. List events:  dd events --action list --tags "env:prod"
    2. Query alerts: dd security --action list-signals
    3. Check SLOs:   dd slos --action get --slo-id "availability"

EVENT STREAM:
  Events appear in:
  - Event Stream (Datadog UI)
  - Dashboards (Event Overlay)
  - Monitors (Event Context)
  - APM (Deployment Tracking)
  - Notebooks (Investigation Timeline)

BEST PRACTICES:
  - Tag events consistently for filtering
  - Use descriptive titles and text
  - Post deployment events for correlation
  - Set appropriate priority and alert type
  - Use aggregation keys for related events
  - Query events to understand system behavior
  - Limit time ranges for faster queries

TAGS FORMAT:
  Comma-separated without spaces:
    ✅ "deploy,production,v1.2.3"
    ✅ "env:prod,service:api,team:platform"
    ❌ "deploy, production, v1.2.3" (spaces not recommended)

NOTES:
  - Events are retained according to your Datadog plan
  - Maximum text length: 4000 characters
  - Maximum 100 tags per event
  - Use --aggregation-key to group related events
  - Query action uses v2 API with advanced search
  - List/get/post actions use v1 API

For more information: https://docs.datadoghq.com/api/latest/events/
`

	fmt.Println(helpText)
}
