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

// DowntimesCommand manages Datadog monitor downtimes and scheduled maintenance windows
type DowntimesCommand struct {
	flags        *flag.FlagSet
	action       string
	downtimeID   string
	message      string
	scope        string
	monitorID    int
	monitorTags  string
	start        string
	end          string
	duration     string
	rrule        string
	timezone     string
	currentOnly  bool
	jsonOut      bool
}

// Downtime represents a parsed downtime from Datadog API
type Downtime struct {
	ID         string                 `json:"id"`
	Message    string                 `json:"message,omitempty"`
	Scope      string                 `json:"scope,omitempty"`
	Start      string                 `json:"start,omitempty"`
	End        string                 `json:"end,omitempty"`
	Timezone   string                 `json:"timezone,omitempty"`
	Status     string                 `json:"status,omitempty"`
	Recurring  bool                   `json:"recurring"`
	MonitorID  interface{}            `json:"monitor_id,omitempty"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// DowntimesResponse represents the formatted downtimes response
type DowntimesResponse struct {
	Status         string                 `json:"status"`
	TotalDowntimes int                    `json:"total_downtimes"`
	ActiveCount    int                    `json:"active_count"`
	ScheduledCount int                    `json:"scheduled_count"`
	Summary        map[string]int         `json:"summary"`
	Downtimes      []Downtime             `json:"downtimes,omitempty"`
	RawData        map[string]interface{} `json:"raw_data,omitempty"`
}

// NewDowntimesCommand creates a new downtimes command
func NewDowntimesCommand() *DowntimesCommand {
	cmd := &DowntimesCommand{
		flags: flag.NewFlagSet("downtimes", flag.ExitOnError),
	}

	cmd.flags.StringVar(&cmd.action, "action", "list", "Action: list, create, get, update, cancel")
	cmd.flags.StringVar(&cmd.downtimeID, "downtime-id", "", "Downtime ID (for get/update/cancel)")
	cmd.flags.StringVar(&cmd.message, "message", "", "Downtime message/description")
	cmd.flags.StringVar(&cmd.scope, "scope", "", "Scope filter (e.g., 'env:prod', 'service:api')")
	cmd.flags.IntVar(&cmd.monitorID, "monitor-id", 0, "Specific monitor ID to mute")
	cmd.flags.StringVar(&cmd.monitorTags, "monitor-tags", "", "Monitor tags (comma-separated, or '*' for all)")
	cmd.flags.StringVar(&cmd.start, "start", "", "Start time (RFC3339, e.g., '2024-01-15T10:00:00Z')")
	cmd.flags.StringVar(&cmd.end, "end", "", "End time (RFC3339, for one-time downtime)")
	cmd.flags.StringVar(&cmd.duration, "duration", "", "Duration (for recurring, e.g., '2h', '30m')")
	cmd.flags.StringVar(&cmd.rrule, "rrule", "", "Recurrence rule (e.g., 'FREQ=DAILY;INTERVAL=1')")
	cmd.flags.StringVar(&cmd.timezone, "timezone", "UTC", "Timezone (e.g., 'America/New_York')")
	cmd.flags.BoolVar(&cmd.currentOnly, "current-only", false, "Show only active downtimes")
	cmd.flags.BoolVar(&cmd.jsonOut, "json", false, "Output as JSON")

	return cmd
}

// Name returns the command name
func (c *DowntimesCommand) Name() string {
	return "downtimes"
}

// Description returns the command description
func (c *DowntimesCommand) Description() string {
	return "Manage Datadog monitor downtimes and scheduled maintenance windows"
}

// Run executes the downtimes command
func (c *DowntimesCommand) Run(args []string) error {
	// Initialize observability
	obs, err := observability.Init("manage-downtimes", "production")
	if err != nil {
		return fmt.Errorf("failed to init observability: %w", err)
	}
	defer obs.Shutdown(0)

	// Parse flags
	if err := c.flags.Parse(args); err != nil {
		return err
	}

	// Start tracing
	span := obs.StartSpan("downtimes.manage")
	defer obs.FinishSpan(span)

	obs.LogInfo(fmt.Sprintf("Managing downtimes with action: %s", c.action))

	// Initialize Datadog client
	ddClient, err := client.NewClient()
	if err != nil {
		return fmt.Errorf("failed to initialize Datadog client: %w", err)
	}

	// Execute action
	switch c.action {
	case "list", "list-downtimes":
		return c.listDowntimes(ddClient, obs)
	case "create", "create-downtime":
		return c.createDowntime(ddClient, obs)
	case "get", "get-downtime":
		return c.getDowntime(ddClient, obs)
	case "update", "update-downtime":
		return c.updateDowntime(ddClient, obs)
	case "cancel", "cancel-downtime", "delete":
		return c.cancelDowntime(ddClient, obs)
	default:
		return fmt.Errorf("unknown action: %s (use: list, create, get, update, cancel)", c.action)
	}
}

// listDowntimes lists all downtimes
func (c *DowntimesCommand) listDowntimes(ddClient *client.Client, obs *observability.Observability) error {
	obs.LogInfo("Listing downtimes")

	// Query downtimes
	data, err := ddClient.ListDowntimes(c.currentOnly)
	if err != nil {
			return fmt.Errorf("failed to list downtimes: %w", err)
	}

	// Parse and display results
	return c.parseAndDisplayList(data, obs)
}

// createDowntime creates a new downtime
func (c *DowntimesCommand) createDowntime(ddClient *client.Client, obs *observability.Observability) error {
	obs.LogInfo("Creating downtime")

	// Validate required fields
	if c.start == "" {
		return fmt.Errorf("--start is required for creating downtime")
	}

	// Build downtime payload
	payload, err := c.buildDowntimePayload()
	if err != nil {
		return fmt.Errorf("failed to build downtime payload: %w", err)
	}

	// Create downtime
	data, err := ddClient.CreateDowntime(payload)
	if err != nil {
		return fmt.Errorf("failed to create downtime: %w", err)
	}

	// Parse and display result
	return c.parseAndDisplaySingle(data, obs)
}

// getDowntime gets a specific downtime
func (c *DowntimesCommand) getDowntime(ddClient *client.Client, obs *observability.Observability) error {
	if c.downtimeID == "" {
		return fmt.Errorf("--downtime-id is required for get action")
	}

	obs.LogInfo(fmt.Sprintf("Getting downtime: %s", c.downtimeID))

	// Get downtime
	data, err := ddClient.GetDowntime(c.downtimeID)
	if err != nil {
		return fmt.Errorf("failed to get downtime: %w", err)
	}

	// Parse and display result
	return c.parseAndDisplaySingle(data, obs)
}

// updateDowntime updates an existing downtime
func (c *DowntimesCommand) updateDowntime(ddClient *client.Client, obs *observability.Observability) error {
	if c.downtimeID == "" {
		return fmt.Errorf("--downtime-id is required for update action")
	}

	obs.LogInfo(fmt.Sprintf("Updating downtime: %s", c.downtimeID))

	// Build downtime payload
	payload, err := c.buildDowntimePayload()
	if err != nil {
		return fmt.Errorf("failed to build downtime payload: %w", err)
	}

	// Update downtime
	data, err := ddClient.UpdateDowntime(c.downtimeID, payload)
	if err != nil {
		return fmt.Errorf("failed to update downtime: %w", err)
	}

	// Parse and display result
	return c.parseAndDisplaySingle(data, obs)
}

// cancelDowntime cancels/deletes a downtime
func (c *DowntimesCommand) cancelDowntime(ddClient *client.Client, obs *observability.Observability) error {
	if c.downtimeID == "" {
		return fmt.Errorf("--downtime-id is required for cancel action")
	}

	obs.LogInfo(fmt.Sprintf("Canceling downtime: %s", c.downtimeID))

	// Cancel downtime
	err := ddClient.CancelDowntime(c.downtimeID)
	if err != nil {
		return fmt.Errorf("failed to cancel downtime: %w", err)
	}

	fmt.Printf("✓ Downtime %s canceled successfully\n", c.downtimeID)
	return nil
}

// buildDowntimePayload builds the API payload for create/update
func (c *DowntimesCommand) buildDowntimePayload() (map[string]interface{}, error) {
	// Build attributes
	attributes := make(map[string]interface{})

	// Message
	if c.message != "" {
		attributes["message"] = c.message
	}

	// Scope
	if c.scope != "" {
		attributes["scope"] = c.scope
	}

	// Monitor identifier
	monitorIdentifier := make(map[string]interface{})
	if c.monitorID > 0 {
		monitorIdentifier["monitor_id"] = c.monitorID
	} else if c.monitorTags != "" {
		tags := strings.Split(c.monitorTags, ",")
		for i := range tags {
			tags[i] = strings.TrimSpace(tags[i])
		}
		monitorIdentifier["monitor_tags"] = tags
	}
	if len(monitorIdentifier) > 0 {
		attributes["monitor_identifier"] = monitorIdentifier
	}

	// Schedule
	schedule, err := c.buildSchedule()
	if err != nil {
		return nil, err
	}
	attributes["schedule"] = schedule

	// Display timezone
	if c.timezone != "" {
		attributes["display_timezone"] = c.timezone
	}

	// Build payload
	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type":       "downtime",
			"attributes": attributes,
		},
	}

	return payload, nil
}

// buildSchedule builds the schedule object
func (c *DowntimesCommand) buildSchedule() (map[string]interface{}, error) {
	schedule := make(map[string]interface{})

	// Check if recurring
	if c.rrule != "" {
		// Recurring downtime
		recurrence := make(map[string]interface{})

		// RRULE
		recurrence["rrule"] = c.rrule

		// Start time (without timezone for recurring)
		if c.start != "" {
			// Parse and format start time
			startTime, err := time.Parse(time.RFC3339, c.start)
			if err != nil {
				return nil, fmt.Errorf("invalid start time format: %w", err)
			}
			recurrence["start"] = startTime.Format("2006-01-02T15:04:05")
		}

		// Duration
		if c.duration != "" {
			recurrence["duration"] = c.duration
		}

		// Timezone for recurring
		if c.timezone != "" {
			schedule["timezone"] = c.timezone
		}

		schedule["recurrences"] = []interface{}{recurrence}
	} else {
		// One-time downtime
		if c.start != "" {
			schedule["start"] = c.start
		}
		if c.end != "" {
			schedule["end"] = c.end
		}
	}

	return schedule, nil
}

// parseAndDisplayList parses and displays list of downtimes
func (c *DowntimesCommand) parseAndDisplayList(data []byte, obs *observability.Observability) error {
	// Parse JSON response
	var apiResponse struct {
		Data []struct {
			ID         string `json:"id"`
			Type       string `json:"type"`
			Attributes struct {
				Message          string                 `json:"message"`
				Scope            string                 `json:"scope"`
				MonitorIdentifier map[string]interface{} `json:"monitor_identifier"`
				Schedule         map[string]interface{} `json:"schedule"`
				DisplayTimezone  string                 `json:"display_timezone"`
				Status           string                 `json:"status"`
			} `json:"attributes"`
		} `json:"data"`
		Meta struct {
			Page struct {
				TotalCount int `json:"total_count"`
			} `json:"page"`
		} `json:"meta"`
	}

	if err := json.Unmarshal(data, &apiResponse); err != nil {
		return fmt.Errorf("failed to parse API response: %w", err)
	}

	// Build response
	response := DowntimesResponse{
		Status:         "success",
		TotalDowntimes: apiResponse.Meta.Page.TotalCount,
		Summary:        make(map[string]int),
		Downtimes:      make([]Downtime, 0),
	}

	// Parse downtimes
	for _, item := range apiResponse.Data {
		downtime := Downtime{
			ID:      item.ID,
			Message: item.Attributes.Message,
			Scope:   item.Attributes.Scope,
			Timezone: item.Attributes.DisplayTimezone,
			Status:  item.Attributes.Status,
			Attributes: map[string]interface{}{
				"schedule":           item.Attributes.Schedule,
				"monitor_identifier": item.Attributes.MonitorIdentifier,
			},
		}

		// Check if recurring
		if recurrences, ok := item.Attributes.Schedule["recurrences"]; ok && recurrences != nil {
			downtime.Recurring = true
		}

		// Extract start/end times
		if start, ok := item.Attributes.Schedule["start"].(string); ok {
			downtime.Start = start
		}
		if end, ok := item.Attributes.Schedule["end"].(string); ok {
			downtime.End = end
		}

		// Update counts
		response.Summary[item.Attributes.Status]++
		if item.Attributes.Status == "active" {
			response.ActiveCount++
		} else if item.Attributes.Status == "scheduled" {
			response.ScheduledCount++
		}

		response.Downtimes = append(response.Downtimes, downtime)
	}

	// Output results
	if c.jsonOut {
		output, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(output))
	} else {
		c.displayFormattedList(response)
	}

	return nil
}

// parseAndDisplaySingle parses and displays a single downtime
func (c *DowntimesCommand) parseAndDisplaySingle(data []byte, obs *observability.Observability) error {
	// Parse JSON response
	var apiResponse struct {
		Data struct {
			ID         string `json:"id"`
			Type       string `json:"type"`
			Attributes struct {
				Message          string                 `json:"message"`
				Scope            string                 `json:"scope"`
				MonitorIdentifier map[string]interface{} `json:"monitor_identifier"`
				Schedule         map[string]interface{} `json:"schedule"`
				DisplayTimezone  string                 `json:"display_timezone"`
				Status           string                 `json:"status"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &apiResponse); err != nil {
		return fmt.Errorf("failed to parse API response: %w", err)
	}

	// Build downtime object
	downtime := Downtime{
		ID:       apiResponse.Data.ID,
		Message:  apiResponse.Data.Attributes.Message,
		Scope:    apiResponse.Data.Attributes.Scope,
		Timezone: apiResponse.Data.Attributes.DisplayTimezone,
		Status:   apiResponse.Data.Attributes.Status,
		Attributes: map[string]interface{}{
			"schedule":           apiResponse.Data.Attributes.Schedule,
			"monitor_identifier": apiResponse.Data.Attributes.MonitorIdentifier,
		},
	}

	// Check if recurring
	if schedule, ok := apiResponse.Data.Attributes.Schedule["recurrences"]; ok && schedule != nil {
		downtime.Recurring = true
	}

	// Output result
	if c.jsonOut {
		output, err := json.MarshalIndent(downtime, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(output))
	} else {
		c.displayFormattedSingle(downtime)
	}

	return nil
}

// displayFormattedList displays formatted list output
func (c *DowntimesCommand) displayFormattedList(response DowntimesResponse) {
	fmt.Println("Downtime Summary")
	fmt.Println("================")
	fmt.Printf("Total downtimes: %d\n", response.TotalDowntimes)
	fmt.Printf("  Active: %d\n", response.ActiveCount)
	fmt.Printf("  Scheduled: %d\n", response.ScheduledCount)

	// Status breakdown
	if len(response.Summary) > 0 {
		fmt.Println("\nStatus breakdown:")
		for status, count := range response.Summary {
			fmt.Printf("  %s: %d\n", status, count)
		}
	}

	// Recent downtimes
	if len(response.Downtimes) > 0 {
		fmt.Println("\nRecent Downtimes:")
		for i, dt := range response.Downtimes {
			if i >= 10 {
				fmt.Printf("\n... and %d more (use --json for full list)\n", len(response.Downtimes)-10)
				break
			}

			fmt.Printf("\n[%s] %s\n", dt.ID, dt.Status)
			if dt.Message != "" {
				fmt.Printf("  Message: %s\n", dt.Message)
			}
			if dt.Scope != "" {
				fmt.Printf("  Scope: %s\n", dt.Scope)
			}
			if dt.Recurring {
				fmt.Printf("  Type: Recurring\n")
			} else {
				fmt.Printf("  Type: One-time\n")
				if dt.Start != "" {
					fmt.Printf("  Start: %s\n", dt.Start)
				}
				if dt.End != "" {
					fmt.Printf("  End: %s\n", dt.End)
				}
			}
		}
	}
}

// displayFormattedSingle displays formatted single downtime output
func (c *DowntimesCommand) displayFormattedSingle(downtime Downtime) {
	fmt.Println("Downtime Details")
	fmt.Println("================")
	fmt.Printf("ID: %s\n", downtime.ID)
	fmt.Printf("Status: %s\n", downtime.Status)

	if downtime.Message != "" {
		fmt.Printf("Message: %s\n", downtime.Message)
	}

	if downtime.Scope != "" {
		fmt.Printf("Scope: %s\n", downtime.Scope)
	}

	if downtime.Timezone != "" {
		fmt.Printf("Timezone: %s\n", downtime.Timezone)
	}

	if downtime.Recurring {
		fmt.Printf("Type: Recurring\n")
	} else {
		fmt.Printf("Type: One-time\n")
	}

	// Display schedule details
	if schedule, ok := downtime.Attributes["schedule"].(map[string]interface{}); ok {
		fmt.Println("\nSchedule:")
		if recurrences, ok := schedule["recurrences"].([]interface{}); ok && len(recurrences) > 0 {
			if rec, ok := recurrences[0].(map[string]interface{}); ok {
				if rrule, ok := rec["rrule"].(string); ok {
					fmt.Printf("  Recurrence: %s\n", rrule)
				}
				if duration, ok := rec["duration"].(string); ok {
					fmt.Printf("  Duration: %s\n", duration)
				}
				if start, ok := rec["start"].(string); ok {
					fmt.Printf("  Start: %s\n", start)
				}
			}
		} else {
			if start, ok := schedule["start"].(string); ok {
				fmt.Printf("  Start: %s\n", start)
			}
			if end, ok := schedule["end"].(string); ok {
				fmt.Printf("  End: %s\n", end)
			}
		}
	}

	// Display monitor info
	if monitorID, ok := downtime.Attributes["monitor_identifier"].(map[string]interface{}); ok {
		fmt.Println("\nMonitor:")
		if id, ok := monitorID["monitor_id"]; ok {
			fmt.Printf("  ID: %v\n", id)
		}
		if tags, ok := monitorID["monitor_tags"].([]interface{}); ok {
			fmt.Printf("  Tags: %v\n", tags)
		}
	}
}

// Help displays help information
func (c *DowntimesCommand) Help() {
	help := `dd downtimes - Manage Datadog Monitor Downtimes

DESCRIPTION:
  Manage monitor downtimes and scheduled maintenance windows. Create downtimes
  to suppress monitor alerts during planned maintenance, deployments, or testing.

USAGE:
  dd downtimes --action <action> [options]

ACTIONS:
  list              List all downtimes
  create            Create a new downtime
  get               Get a specific downtime
  update            Update an existing downtime
  cancel            Cancel/delete a downtime

EXAMPLES:
  # List all active downtimes
  dd downtimes --action list

  # List only currently active downtimes
  dd downtimes --action list --current-only

  # Create one-time downtime for specific monitor
  dd downtimes --action create \
    --monitor-id 123456 \
    --start "2024-01-15T10:00:00Z" \
    --end "2024-01-15T12:00:00Z" \
    --message "Scheduled maintenance"

  # Create downtime for all monitors with specific scope
  dd downtimes --action create \
    --monitor-tags "*" \
    --scope "env:prod" \
    --start "2024-01-15T22:00:00Z" \
    --end "2024-01-16T02:00:00Z" \
    --message "Production deployment window"

  # Create recurring daily downtime
  dd downtimes --action create \
    --monitor-tags "service:database" \
    --start "2024-01-15T03:00:00" \
    --duration "2h" \
    --rrule "FREQ=DAILY;INTERVAL=1" \
    --timezone "America/New_York" \
    --message "Daily backup window"

  # Create weekly recurring downtime (every Monday at 2 AM)
  dd downtimes --action create \
    --monitor-tags "service:api" \
    --start "2024-01-15T02:00:00" \
    --duration "1h" \
    --rrule "FREQ=WEEKLY;BYDAY=MO;INTERVAL=1" \
    --timezone "UTC" \
    --message "Weekly maintenance"

  # Get specific downtime
  dd downtimes --action get --downtime-id abc123-def456

  # Cancel downtime
  dd downtimes --action cancel --downtime-id abc123-def456

OPTIONS:
  --action          Action to perform (list, create, get, update, cancel)
  --downtime-id     Downtime ID (required for get/update/cancel)
  --message         Downtime message/description
  --scope           Scope filter (e.g., 'env:prod', 'service:api')
  --monitor-id      Specific monitor ID to mute
  --monitor-tags    Monitor tags (comma-separated, or '*' for all)
  --start           Start time (RFC3339, e.g., '2024-01-15T10:00:00Z')
  --end             End time (RFC3339, for one-time downtime)
  --duration        Duration (for recurring, e.g., '2h', '30m')
  --rrule           Recurrence rule (e.g., 'FREQ=DAILY;INTERVAL=1')
  --timezone        Timezone (default: UTC, e.g., 'America/New_York')
  --current-only    Show only active downtimes
  --json            Output as JSON

RRULE FORMAT:
  Recurrence rules follow iCalendar standard (RFC 5545):
    FREQ          - DAILY, WEEKLY, MONTHLY, YEARLY
    INTERVAL      - Recurrence step (default: 1)
    BYDAY         - Days of week (MO, TU, WE, TH, FR, SA, SU)
    BYMONTHDAY    - Day of month (1-31)
    BYSETPOS      - Position in recurrence set

  Examples:
    Daily:              FREQ=DAILY;INTERVAL=1
    Weekly (Monday):    FREQ=WEEKLY;BYDAY=MO;INTERVAL=1
    Bi-weekly:          FREQ=WEEKLY;INTERVAL=2
    Monthly (3rd Wed):  FREQ=MONTHLY;BYSETPOS=3;BYDAY=WE;INTERVAL=1

NOTES:
  - Start time must be in RFC3339 format (ISO 8601)
  - One-time downtimes require --start and --end
  - Recurring downtimes require --start, --duration, and --rrule
  - Use --monitor-tags "*" to mute all monitors for a given scope
  - Timezone defaults to UTC for display purposes
  - Canceled downtimes are retained for ~2 days before deletion

For more information: https://docs.datadoghq.com/api/latest/downtimes/
`
	fmt.Println(help)
}
