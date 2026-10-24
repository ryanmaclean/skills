package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/datadog/skill/internal/client"
)

// OnCallCommand manages Datadog On-Call scheduling and rotations
type OnCallCommand struct {
	flags       *flag.FlagSet
	action      string
	scheduleID  string
	teamID      string
	name        string
	timezone    string
	description string
	startDate   string
	endDate     string
	rotation    string
	members     string
	jsonOut     bool
}

// NewOnCallCommand creates a new on-call command
func NewOnCallCommand() Command {
	cmd := &OnCallCommand{
		flags: flag.NewFlagSet("on-call", flag.ExitOnError),
	}

	cmd.flags.StringVar(&cmd.action, "action", "list", "Action to perform")
	cmd.flags.StringVar(&cmd.scheduleID, "schedule-id", "", "Schedule ID")
	cmd.flags.StringVar(&cmd.teamID, "team-id", "", "Team ID")
	cmd.flags.StringVar(&cmd.name, "name", "", "Schedule name")
	cmd.flags.StringVar(&cmd.timezone, "timezone", "UTC", "Timezone (e.g., UTC, America/New_York)")
	cmd.flags.StringVar(&cmd.description, "description", "", "Schedule description")
	cmd.flags.StringVar(&cmd.startDate, "start", "", "Start date (ISO 8601 format)")
	cmd.flags.StringVar(&cmd.endDate, "end", "", "End date (ISO 8601 format)")
	cmd.flags.StringVar(&cmd.rotation, "rotation", "weekly", "Rotation type (daily, weekly, biweekly, monthly)")
	cmd.flags.StringVar(&cmd.members, "members", "", "Comma-separated user IDs for rotation")
	cmd.flags.BoolVar(&cmd.jsonOut, "json", false, "Output in JSON format")

	return cmd
}

// Name returns the command name
func (c *OnCallCommand) Name() string {
	return "on-call"
}

// Description returns a short description
func (c *OnCallCommand) Description() string {
	return "Manage Datadog On-Call scheduling and rotations"
}

// Run executes the on-call command
func (c *OnCallCommand) Run(args []string) error {
	if err := c.flags.Parse(args); err != nil {
		return err
	}

	ddClient, err := client.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Datadog client: %w", err)
	}

	// Route to appropriate action handler
	switch c.action {
	case "list", "list-schedules":
		return c.listSchedules(ddClient)
	case "create", "create-schedule":
		return c.createSchedule(ddClient)
	case "get", "get-schedule":
		return c.getSchedule(ddClient)
	case "update", "update-schedule":
		return c.updateSchedule(ddClient)
	case "delete", "delete-schedule":
		return c.deleteSchedule(ddClient)
	case "who", "who-is-on-call":
		return c.whoIsOnCall(ddClient)
	default:
		return fmt.Errorf("unknown action: %s (use --help to see available actions)", c.action)
	}
}

// Schedule Management Actions

func (c *OnCallCommand) listSchedules(ddClient *client.Client) error {
	resp, err := ddClient.ListOnCallSchedules()
	if err != nil {
		return fmt.Errorf("failed to list on-call schedules: %w", err)
	}

	if c.jsonOut {
		fmt.Println(string(resp))
		return nil
	}

	var result struct {
		Data []struct {
			ID         string `json:"id"`
			Type       string `json:"type"`
			Attributes struct {
				Name        string `json:"name"`
				Timezone    string `json:"timezone"`
				Description string `json:"description"`
			} `json:"attributes"`
			Relationships struct {
				Teams struct {
					Data []struct {
						ID   string `json:"id"`
						Type string `json:"type"`
					} `json:"data"`
				} `json:"teams"`
			} `json:"relationships"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Data) == 0 {
		fmt.Println("No on-call schedules found.")
		fmt.Println("\nTip: Create a schedule with:")
		fmt.Println("  dd on-call --action create --name \"My Schedule\" --team-id <team-id>")
		return nil
	}

	fmt.Printf("On-Call Schedules (%d)\n", len(result.Data))
	fmt.Println(strings.Repeat("=", 80))

	for _, schedule := range result.Data {
		fmt.Printf("\nID: %s\n", schedule.ID)
		fmt.Printf("Name: %s\n", schedule.Attributes.Name)
		if schedule.Attributes.Timezone != "" {
			fmt.Printf("Timezone: %s\n", schedule.Attributes.Timezone)
		}
		if schedule.Attributes.Description != "" {
			fmt.Printf("Description: %s\n", schedule.Attributes.Description)
		}
		if len(schedule.Relationships.Teams.Data) > 0 {
			fmt.Printf("Teams: %d associated\n", len(schedule.Relationships.Teams.Data))
		}
	}

	return nil
}

func (c *OnCallCommand) createSchedule(ddClient *client.Client) error {
	if c.name == "" {
		return fmt.Errorf("--name is required")
	}

	// Build schedule layers (rotation configuration)
	layers := []map[string]interface{}{}

	if c.members != "" {
		memberIDs := strings.Split(c.members, ",")
		users := []map[string]interface{}{}
		for _, memberID := range memberIDs {
			users = append(users, map[string]interface{}{
				"id":   strings.TrimSpace(memberID),
				"type": "users",
			})
		}

		// Determine rotation length based on type
		rotationLength := c.getRotationLength()

		// Create layer with rotation
		layer := map[string]interface{}{
			"type": "on_call_schedule_layers",
			"attributes": map[string]interface{}{
				"name":     fmt.Sprintf("%s Rotation", c.rotation),
				"start":    c.getStartDate(),
				"timezone": c.timezone,
				"rotation": map[string]interface{}{
					"type":   "rolling",
					"length": rotationLength,
					"users":  users,
				},
			},
		}

		layers = append(layers, layer)
	}

	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "on_call_schedules",
			"attributes": map[string]interface{}{
				"name":     c.name,
				"timezone": c.timezone,
			},
		},
	}

	// Add optional fields
	attrs := payload["data"].(map[string]interface{})["attributes"].(map[string]interface{})
	if c.description != "" {
		attrs["description"] = c.description
	}
	if len(layers) > 0 {
		attrs["layers"] = layers
	}

	// Add team relationship if provided
	if c.teamID != "" {
		payload["data"].(map[string]interface{})["relationships"] = map[string]interface{}{
			"teams": map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"id":   c.teamID,
						"type": "teams",
					},
				},
			},
		}
	}

	resp, err := ddClient.CreateOnCallSchedule(payload)
	if err != nil {
		return fmt.Errorf("failed to create on-call schedule: %w", err)
	}

	if c.jsonOut {
		fmt.Println(string(resp))
		return nil
	}

	var result struct {
		Data struct {
			ID         string `json:"id"`
			Attributes struct {
				Name     string `json:"name"`
				Timezone string `json:"timezone"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Println("✓ On-call schedule created successfully")
	fmt.Printf("Schedule ID: %s\n", result.Data.ID)
	fmt.Printf("Name: %s\n", result.Data.Attributes.Name)
	fmt.Printf("Timezone: %s\n", result.Data.Attributes.Timezone)

	return nil
}

func (c *OnCallCommand) getSchedule(ddClient *client.Client) error {
	if c.scheduleID == "" {
		return fmt.Errorf("--schedule-id is required")
	}

	resp, err := ddClient.GetOnCallSchedule(c.scheduleID)
	if err != nil {
		return fmt.Errorf("failed to get on-call schedule: %w", err)
	}

	if c.jsonOut {
		fmt.Println(string(resp))
		return nil
	}

	var result struct {
		Data struct {
			ID         string `json:"id"`
			Attributes struct {
				Name        string `json:"name"`
				Timezone    string `json:"timezone"`
				Description string `json:"description"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Printf("On-Call Schedule: %s\n", result.Data.Attributes.Name)
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("ID: %s\n", result.Data.ID)
	fmt.Printf("Timezone: %s\n", result.Data.Attributes.Timezone)
	if result.Data.Attributes.Description != "" {
		fmt.Printf("Description: %s\n", result.Data.Attributes.Description)
	}

	return nil
}

func (c *OnCallCommand) updateSchedule(ddClient *client.Client) error {
	if c.scheduleID == "" {
		return fmt.Errorf("--schedule-id is required")
	}

	attrs := make(map[string]interface{})
	if c.name != "" {
		attrs["name"] = c.name
	}
	if c.description != "" {
		attrs["description"] = c.description
	}
	if c.timezone != "" && c.timezone != "UTC" {
		attrs["timezone"] = c.timezone
	}

	if len(attrs) == 0 {
		return fmt.Errorf("no fields to update (provide --name, --description, or --timezone)")
	}

	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type":       "on_call_schedules",
			"id":         c.scheduleID,
			"attributes": attrs,
		},
	}

	resp, err := ddClient.UpdateOnCallSchedule(c.scheduleID, payload)
	if err != nil {
		return fmt.Errorf("failed to update on-call schedule: %w", err)
	}

	if c.jsonOut {
		fmt.Println(string(resp))
		return nil
	}

	fmt.Println("✓ On-call schedule updated successfully")
	return nil
}

func (c *OnCallCommand) deleteSchedule(ddClient *client.Client) error {
	if c.scheduleID == "" {
		return fmt.Errorf("--schedule-id is required")
	}

	if err := ddClient.DeleteOnCallSchedule(c.scheduleID); err != nil {
		return fmt.Errorf("failed to delete on-call schedule: %w", err)
	}

	fmt.Printf("✓ On-call schedule %s deleted successfully\n", c.scheduleID)
	return nil
}

func (c *OnCallCommand) whoIsOnCall(ddClient *client.Client) error {
	if c.scheduleID == "" {
		// List all schedules and show who's on call for each
		resp, err := ddClient.ListOnCallSchedules()
		if err != nil {
			return fmt.Errorf("failed to list schedules: %w", err)
		}

		var result struct {
			Data []struct {
				ID         string `json:"id"`
				Attributes struct {
					Name string `json:"name"`
				} `json:"attributes"`
			} `json:"data"`
		}

		if err := json.Unmarshal(resp, &result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if len(result.Data) == 0 {
			fmt.Println("No on-call schedules configured.")
			return nil
		}

		fmt.Println("Current On-Call Status")
		fmt.Println(strings.Repeat("=", 80))

		for _, schedule := range result.Data {
			fmt.Printf("\nSchedule: %s\n", schedule.Attributes.Name)
			fmt.Printf("ID: %s\n", schedule.ID)
			fmt.Println("Status: Check schedule details for current on-call rotation")
		}

		return nil
	}

	// Get specific schedule details
	resp, err := ddClient.GetOnCallSchedule(c.scheduleID)
	if err != nil {
		return fmt.Errorf("failed to get schedule: %w", err)
	}

	if c.jsonOut {
		fmt.Println(string(resp))
		return nil
	}

	var result struct {
		Data struct {
			ID         string `json:"id"`
			Attributes struct {
				Name     string `json:"name"`
				Timezone string `json:"timezone"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Printf("On-Call Schedule: %s\n", result.Data.Attributes.Name)
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("Schedule ID: %s\n", result.Data.ID)
	fmt.Printf("Timezone: %s\n", result.Data.Attributes.Timezone)
	fmt.Println("\nNote: Use --json flag to see complete schedule details including current rotation")

	return nil
}

// Helper functions

func (c *OnCallCommand) getRotationLength() int {
	switch c.rotation {
	case "daily":
		return 1
	case "weekly":
		return 7
	case "biweekly":
		return 14
	case "monthly":
		return 30
	default:
		return 7 // default to weekly
	}
}

func (c *OnCallCommand) getStartDate() string {
	if c.startDate != "" {
		return c.startDate
	}
	// Default to current time in ISO 8601 format
	return time.Now().UTC().Format(time.RFC3339)
}

// Help displays help information
func (c *OnCallCommand) Help() {
	help := `dd on-call - Manage Datadog On-Call Scheduling

DESCRIPTION:
  Manage on-call schedules and rotations for team coverage. Create schedules,
  configure rotations, and track who's currently on-call.

USAGE:
  dd on-call --action <action> [options]

ACTIONS:
  list, list-schedules       List all on-call schedules
  create, create-schedule    Create new on-call schedule
  get, get-schedule          Get schedule details
  update, update-schedule    Update schedule
  delete, delete-schedule    Delete schedule
  who, who-is-on-call        Show who is currently on-call

OPTIONS:
  --schedule-id string    Schedule ID
  --team-id string        Team ID to associate with schedule
  --name string           Schedule name
  --timezone string       Timezone (default: UTC)
                          Examples: UTC, America/New_York, Europe/London
  --description string    Schedule description
  --start string          Start date (ISO 8601 format)
  --rotation string       Rotation type (default: weekly)
                          Options: daily, weekly, biweekly, monthly
  --members string        Comma-separated user IDs for rotation
  --json                  Output in JSON format

EXAMPLES:
  # List all on-call schedules
  dd on-call --action list

  # Create new schedule with weekly rotation
  dd on-call --action create \
    --name "Backend Team Schedule" \
    --team-id team-123 \
    --rotation weekly \
    --members "user1,user2,user3" \
    --timezone "America/New_York"

  # Create daily rotation schedule
  dd on-call --action create \
    --name "Daily Rotation" \
    --rotation daily \
    --members "user1,user2"

  # Get schedule details
  dd on-call --action get --schedule-id schedule-123

  # Update schedule
  dd on-call --action update \
    --schedule-id schedule-123 \
    --name "Updated Schedule Name"

  # Delete schedule
  dd on-call --action delete --schedule-id schedule-123

  # Check who is currently on-call
  dd on-call --action who

  # Check who is on-call for specific schedule
  dd on-call --action who --schedule-id schedule-123

  # Get JSON output
  dd on-call --action list --json

ROTATION TYPES:
  daily      - 24-hour rotations
  weekly     - 7-day rotations (default)
  biweekly   - 14-day rotations
  monthly    - 30-day rotations

TIMEZONES:
  Common timezones:
    UTC                - Coordinated Universal Time
    America/New_York   - Eastern Time (US)
    America/Los_Angeles - Pacific Time (US)
    America/Chicago    - Central Time (US)
    Europe/London      - UK Time
    Europe/Paris       - Central European Time
    Asia/Tokyo         - Japan Time

USE CASES:
  1. Create rotating on-call schedules for teams
  2. Manage 24/7 coverage with rotation handoffs
  3. Track who is currently on-call
  4. Configure timezone-aware schedules
  5. Set up daily, weekly, or monthly rotations

INTEGRATION:
  On-call schedules integrate with:
  - Incidents - Route to on-call team members
  - Monitors - Alert on-call responders
  - Status Pages - Show on-call contacts

AUTHENTICATION:
  Requires DD_API_KEY and DD_APP_KEY environment variables.
  Requires on_call_read and on_call_write permissions.
`
	fmt.Println(strings.TrimSpace(help))
}
