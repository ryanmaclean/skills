package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"time"

	"github.com/datadog/skill/internal/client"
	"github.com/datadog/skill/internal/observability"
)

// SLOCorrectionsCommand manages SLO corrections for accurate reporting
type SLOCorrectionsCommand struct {
	flags       *flag.FlagSet
	action      string
	correctionID string
	sloID       string
	category    string
	description string
	start       string
	end         string
	duration    int
	jsonOut     bool
}

// SLOCorrection represents a parsed SLO correction from Datadog API
type SLOCorrection struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Start       int64  `json:"start"`
	End         int64  `json:"end"`
	Duration    int    `json:"duration"`
	SLOIDs      []string `json:"slo_ids,omitempty"`
	CreatedAt   int64  `json:"created_at,omitempty"`
	ModifiedAt  int64  `json:"modified_at,omitempty"`
	Creator     string `json:"creator,omitempty"`
	Modifier    string `json:"modifier,omitempty"`
}

// SLOCorrectionsResponse represents the formatted SLO corrections response
type SLOCorrectionsResponse struct {
	Status           string          `json:"status"`
	TotalCorrections int             `json:"total_corrections"`
	Corrections      []SLOCorrection `json:"corrections,omitempty"`
}

// NewSLOCorrectionsCommand creates a new SLO corrections command
func NewSLOCorrectionsCommand() *SLOCorrectionsCommand {
	cmd := &SLOCorrectionsCommand{
		flags: flag.NewFlagSet("slo-corrections", flag.ExitOnError),
	}

	cmd.flags.StringVar(&cmd.action, "action", "list", "Action: list, create, get, update, delete")
	cmd.flags.StringVar(&cmd.correctionID, "correction-id", "", "Correction ID (for get/update/delete)")
	cmd.flags.StringVar(&cmd.sloID, "slo-id", "", "SLO ID to apply correction to")
	cmd.flags.StringVar(&cmd.category, "category", "", "Category: Scheduled Maintenance, Outside Business Hours, Deployment, Other")
	cmd.flags.StringVar(&cmd.description, "description", "", "Description of why correction is needed")
	cmd.flags.StringVar(&cmd.start, "start", "", "Start time (Unix timestamp or RFC3339)")
	cmd.flags.StringVar(&cmd.end, "end", "", "End time (Unix timestamp or RFC3339)")
	cmd.flags.IntVar(&cmd.duration, "duration", 0, "Duration in seconds (alternative to end time)")
	cmd.flags.BoolVar(&cmd.jsonOut, "json", false, "Output as JSON")

	return cmd
}

// Name returns the command name
func (c *SLOCorrectionsCommand) Name() string {
	return "slo-corrections"
}

// Description returns the command description
func (c *SLOCorrectionsCommand) Description() string {
	return "Manage SLO corrections for accurate reporting"
}

// Run executes the SLO corrections command
func (c *SLOCorrectionsCommand) Run(args []string) error {
	// Initialize observability
	obs, err := observability.Init("manage-slo-corrections", "production")
	if err != nil {
		return fmt.Errorf("failed to init observability: %w", err)
	}
	defer obs.Shutdown(0)

	// Parse flags
	if err := c.flags.Parse(args); err != nil {
		return err
	}

	// Start tracing
	span := obs.StartSpan("slo_corrections.manage")
	defer obs.FinishSpan(span)

	obs.LogInfo(fmt.Sprintf("Managing SLO corrections with action: %s", c.action))

	// Initialize Datadog client
	ddClient, err := client.NewClient()
	if err != nil {
		return fmt.Errorf("failed to initialize Datadog client: %w", err)
	}

	// Execute action
	switch c.action {
	case "list", "list-corrections":
		return c.listCorrections(ddClient, obs)
	case "create", "create-correction":
		return c.createCorrection(ddClient, obs)
	case "get", "get-correction":
		return c.getCorrection(ddClient, obs)
	case "update", "update-correction":
		return c.updateCorrection(ddClient, obs)
	case "delete", "delete-correction":
		return c.deleteCorrection(ddClient, obs)
	default:
		return fmt.Errorf("unknown action: %s (use: list, create, get, update, delete)", c.action)
	}
}

// listCorrections lists all SLO corrections
func (c *SLOCorrectionsCommand) listCorrections(ddClient *client.Client, obs *observability.Observability) error {
	obs.LogInfo("Listing SLO corrections")

	// Query corrections
	data, err := ddClient.ListSLOCorrections()
	if err != nil {
		return fmt.Errorf("failed to list SLO corrections: %w", err)
	}

	// Parse and display results
	return c.parseAndDisplayList(data, obs)
}

// createCorrection creates a new SLO correction
func (c *SLOCorrectionsCommand) createCorrection(ddClient *client.Client, obs *observability.Observability) error {
	obs.LogInfo("Creating SLO correction")

	// Validate required fields
	if c.category == "" {
		return fmt.Errorf("--category is required for creating correction")
	}
	if c.description == "" {
		return fmt.Errorf("--description is required for creating correction")
	}
	if c.start == "" {
		return fmt.Errorf("--start is required for creating correction")
	}
	if c.end == "" && c.duration == 0 {
		return fmt.Errorf("either --end or --duration is required for creating correction")
	}

	// Build correction payload
	payload, err := c.buildCorrectionPayload()
	if err != nil {
		return err
	}

	// Create correction
	data, err := ddClient.CreateSLOCorrection(payload)
	if err != nil {
		return fmt.Errorf("failed to create SLO correction: %w", err)
	}

	// Parse and display result
	return c.parseAndDisplaySingle(data, obs)
}

// getCorrection gets a specific SLO correction
func (c *SLOCorrectionsCommand) getCorrection(ddClient *client.Client, obs *observability.Observability) error {
	if c.correctionID == "" {
		return fmt.Errorf("--correction-id is required for get action")
	}

	obs.LogInfo(fmt.Sprintf("Getting SLO correction: %s", c.correctionID))

	// Get correction
	data, err := ddClient.GetSLOCorrection(c.correctionID)
	if err != nil {
		return fmt.Errorf("failed to get SLO correction: %w", err)
	}

	// Parse and display result
	return c.parseAndDisplaySingle(data, obs)
}

// updateCorrection updates an existing SLO correction
func (c *SLOCorrectionsCommand) updateCorrection(ddClient *client.Client, obs *observability.Observability) error {
	if c.correctionID == "" {
		return fmt.Errorf("--correction-id is required for update action")
	}

	obs.LogInfo(fmt.Sprintf("Updating SLO correction: %s", c.correctionID))

	// Build correction payload
	payload, err := c.buildCorrectionPayload()
	if err != nil {
		return err
	}

	// Update correction
	data, err := ddClient.UpdateSLOCorrection(c.correctionID, payload)
	if err != nil {
		return fmt.Errorf("failed to update SLO correction: %w", err)
	}

	// Parse and display result
	return c.parseAndDisplaySingle(data, obs)
}

// deleteCorrection deletes an SLO correction
func (c *SLOCorrectionsCommand) deleteCorrection(ddClient *client.Client, obs *observability.Observability) error {
	if c.correctionID == "" {
		return fmt.Errorf("--correction-id is required for delete action")
	}

	obs.LogInfo(fmt.Sprintf("Deleting SLO correction: %s", c.correctionID))

	// Delete correction
	err := ddClient.DeleteSLOCorrection(c.correctionID)
	if err != nil {
		return fmt.Errorf("failed to delete SLO correction: %w", err)
	}

	fmt.Printf("✓ SLO correction %s deleted successfully\n", c.correctionID)
	return nil
}

// buildCorrectionPayload builds the API payload for create/update
func (c *SLOCorrectionsCommand) buildCorrectionPayload() (map[string]interface{}, error) {
	attributes := make(map[string]interface{})

	// Category
	if c.category != "" {
		attributes["category"] = c.category
	}

	// Description
	if c.description != "" {
		attributes["description"] = c.description
	}

	// Parse start time
	if c.start != "" {
		startTime, err := parseTime(c.start)
		if err != nil {
			return nil, fmt.Errorf("invalid start time: %w", err)
		}
		attributes["start"] = startTime.Unix()

		// Calculate end time
		if c.end != "" {
			endTime, err := parseTime(c.end)
			if err != nil {
				return nil, fmt.Errorf("invalid end time: %w", err)
			}
			attributes["end"] = endTime.Unix()
			attributes["duration"] = int(endTime.Unix() - startTime.Unix())
		} else if c.duration > 0 {
			attributes["duration"] = c.duration
			attributes["end"] = startTime.Unix() + int64(c.duration)
		}
	}

	// SLO ID(s)
	if c.sloID != "" {
		attributes["slo_id"] = c.sloID
	}

	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type":       "correction",
			"attributes": attributes,
		},
	}

	return payload, nil
}

// parseTime parses time from Unix timestamp or RFC3339
func parseTime(s string) (time.Time, error) {
	// Try Unix timestamp first
	var timestamp int64
	_, err := fmt.Sscanf(s, "%d", &timestamp)
	if err == nil {
		return time.Unix(timestamp, 0), nil
	}

	// Try RFC3339
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("time must be Unix timestamp or RFC3339 format")
	}
	return t, nil
}

// parseAndDisplayList parses and displays list of SLO corrections
func (c *SLOCorrectionsCommand) parseAndDisplayList(data []byte, obs *observability.Observability) error {
	// Parse JSON response
	var apiResponse struct {
		Data []struct {
			ID         string `json:"id"`
			Type       string `json:"type"`
			Attributes struct {
				Category    string   `json:"category"`
				Description string   `json:"description"`
				Start       int64    `json:"start"`
				End         int64    `json:"end"`
				Duration    int      `json:"duration"`
				SLOIDs      []string `json:"slo_ids"`
				CreatedAt   int64    `json:"created_at"`
				ModifiedAt  int64    `json:"modified_at"`
				Creator     struct {
					Email string `json:"email"`
				} `json:"creator"`
				Modifier struct {
					Email string `json:"email"`
				} `json:"modifier"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &apiResponse); err != nil {
		return fmt.Errorf("failed to parse API response: %w", err)
	}

	// Build response
	response := SLOCorrectionsResponse{
		Status:           "success",
		TotalCorrections: len(apiResponse.Data),
		Corrections:      make([]SLOCorrection, 0),
	}

	// Parse corrections
	for _, item := range apiResponse.Data {
		correction := SLOCorrection{
			ID:          item.ID,
			Type:        item.Type,
			Category:    item.Attributes.Category,
			Description: item.Attributes.Description,
			Start:       item.Attributes.Start,
			End:         item.Attributes.End,
			Duration:    item.Attributes.Duration,
			SLOIDs:      item.Attributes.SLOIDs,
			CreatedAt:   item.Attributes.CreatedAt,
			ModifiedAt:  item.Attributes.ModifiedAt,
			Creator:     item.Attributes.Creator.Email,
			Modifier:    item.Attributes.Modifier.Email,
		}

		response.Corrections = append(response.Corrections, correction)
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

// parseAndDisplaySingle parses and displays a single SLO correction
func (c *SLOCorrectionsCommand) parseAndDisplaySingle(data []byte, obs *observability.Observability) error {
	// Parse JSON response
	var apiResponse struct {
		Data struct {
			ID         string `json:"id"`
			Type       string `json:"type"`
			Attributes struct {
				Category    string   `json:"category"`
				Description string   `json:"description"`
				Start       int64    `json:"start"`
				End         int64    `json:"end"`
				Duration    int      `json:"duration"`
				SLOIDs      []string `json:"slo_ids"`
				CreatedAt   int64    `json:"created_at"`
				ModifiedAt  int64    `json:"modified_at"`
				Creator     struct {
					Email string `json:"email"`
				} `json:"creator"`
				Modifier struct {
					Email string `json:"email"`
				} `json:"modifier"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &apiResponse); err != nil {
		return fmt.Errorf("failed to parse API response: %w", err)
	}

	// Build correction object
	correction := SLOCorrection{
		ID:          apiResponse.Data.ID,
		Type:        apiResponse.Data.Type,
		Category:    apiResponse.Data.Attributes.Category,
		Description: apiResponse.Data.Attributes.Description,
		Start:       apiResponse.Data.Attributes.Start,
		End:         apiResponse.Data.Attributes.End,
		Duration:    apiResponse.Data.Attributes.Duration,
		SLOIDs:      apiResponse.Data.Attributes.SLOIDs,
		CreatedAt:   apiResponse.Data.Attributes.CreatedAt,
		ModifiedAt:  apiResponse.Data.Attributes.ModifiedAt,
		Creator:     apiResponse.Data.Attributes.Creator.Email,
		Modifier:    apiResponse.Data.Attributes.Modifier.Email,
	}

	// Output result
	if c.jsonOut {
		output, err := json.MarshalIndent(correction, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(output))
	} else {
		c.displayFormattedSingle(correction)
	}

	return nil
}

// displayFormattedList displays formatted list output
func (c *SLOCorrectionsCommand) displayFormattedList(response SLOCorrectionsResponse) {
	fmt.Println("SLO Corrections")
	fmt.Println("===============")
	fmt.Printf("Total corrections: %d\n", response.TotalCorrections)

	if len(response.Corrections) > 0 {
		fmt.Println("\nCorrections:")
		for i, correction := range response.Corrections {
			if i >= 20 {
				fmt.Printf("\n... and %d more (use --json for full list)\n", len(response.Corrections)-20)
				break
			}

			fmt.Printf("\n[%s] %s\n", correction.ID, correction.Category)
			fmt.Printf("  Description: %s\n", correction.Description)
			fmt.Printf("  Start: %s\n", time.Unix(correction.Start, 0).Format("2006-01-02 15:04:05 MST"))
			fmt.Printf("  End: %s\n", time.Unix(correction.End, 0).Format("2006-01-02 15:04:05 MST"))
			fmt.Printf("  Duration: %d seconds (%.2f hours)\n", correction.Duration, float64(correction.Duration)/3600)
			if len(correction.SLOIDs) > 0 {
				fmt.Printf("  SLOs affected: %d\n", len(correction.SLOIDs))
			}
			if correction.Creator != "" {
				fmt.Printf("  Created by: %s\n", correction.Creator)
			}
		}
	} else {
		fmt.Println("\nNo SLO corrections found.")
	}
}

// displayFormattedSingle displays formatted single correction output
func (c *SLOCorrectionsCommand) displayFormattedSingle(correction SLOCorrection) {
	fmt.Println("SLO Correction Details")
	fmt.Println("======================")
	fmt.Printf("ID: %s\n", correction.ID)
	fmt.Printf("Category: %s\n", correction.Category)
	fmt.Printf("Description: %s\n", correction.Description)
	fmt.Println()
	fmt.Printf("Start: %s\n", time.Unix(correction.Start, 0).Format("2006-01-02 15:04:05 MST"))
	fmt.Printf("End: %s\n", time.Unix(correction.End, 0).Format("2006-01-02 15:04:05 MST"))
	fmt.Printf("Duration: %d seconds (%.2f hours)\n", correction.Duration, float64(correction.Duration)/3600)
	fmt.Println()

	if len(correction.SLOIDs) > 0 {
		fmt.Printf("SLOs affected (%d):\n", len(correction.SLOIDs))
		for _, sloID := range correction.SLOIDs {
			fmt.Printf("  - %s\n", sloID)
		}
		fmt.Println()
	}

	if correction.Creator != "" {
		fmt.Printf("Created by: %s at %s\n", correction.Creator, time.Unix(correction.CreatedAt, 0).Format("2006-01-02 15:04:05 MST"))
	}
	if correction.Modifier != "" {
		fmt.Printf("Modified by: %s at %s\n", correction.Modifier, time.Unix(correction.ModifiedAt, 0).Format("2006-01-02 15:04:05 MST"))
	}
}

// Help displays help information
func (c *SLOCorrectionsCommand) Help() {
	help := `dd slo-corrections - Manage SLO Corrections

DESCRIPTION:
  Manage SLO corrections to exclude time periods from SLO calculations. Use
  corrections when issues were outside your control (third-party outages,
  scheduled maintenance, etc.) to maintain accurate SLO reporting.

USAGE:
  dd slo-corrections --action <action> [options]

ACTIONS:
  list              List all SLO corrections
  create            Create a new SLO correction
  get               Get a specific SLO correction
  update            Update an existing SLO correction
  delete            Delete an SLO correction

EXAMPLES:
  # List all corrections
  dd slo-corrections --action list

  # Create correction for third-party outage
  dd slo-corrections --action create \
    --category "Other" \
    --description "AWS S3 outage in us-east-1" \
    --start 1674145200 \
    --end 1674152400 \
    --slo-id "abc123"

  # Create correction with duration
  dd slo-corrections --action create \
    --category "Scheduled Maintenance" \
    --description "Database migration" \
    --start "2026-01-25T02:00:00Z" \
    --duration 7200 \
    --slo-id "def456"

  # Get specific correction
  dd slo-corrections --action get --correction-id "correction-123"

  # Update correction description
  dd slo-corrections --action update \
    --correction-id "correction-123" \
    --description "Updated description"

  # Delete correction
  dd slo-corrections --action delete --correction-id "correction-123"

  # Get JSON output
  dd slo-corrections --action list --json

OPTIONS:
  --action          Action to perform (list, create, get, update, delete)
  --correction-id   Correction ID (required for get/update/delete)
  --slo-id          SLO ID to apply correction to
  --category        Category (Scheduled Maintenance, Outside Business Hours, Deployment, Other)
  --description     Description of why correction is needed
  --start           Start time (Unix timestamp or RFC3339)
  --end             End time (Unix timestamp or RFC3339)
  --duration        Duration in seconds (alternative to end time)
  --json            Output as JSON

CORRECTION CATEGORIES:
  Scheduled Maintenance
    - Planned downtime for upgrades
    - Database migrations
    - Infrastructure changes

  Outside Business Hours
    - Non-critical maintenance windows
    - Updates during off-peak times

  Deployment
    - Planned deployment windows
    - Release-related downtime

  Other
    - Third-party service outages
    - DDoS attacks
    - Infrastructure provider issues
    - Unexpected external dependencies

TIME FORMATS:
  Unix timestamp: 1674145200
  RFC3339:       2026-01-25T02:00:00Z

  Use 'date -u +%s' to get current Unix timestamp
  Use 'date -u +%Y-%m-%dT%H:%M:%SZ' for RFC3339

WHEN TO USE CORRECTIONS:
  ✓ Third-party service outages affecting your SLO
  ✓ Scheduled maintenance communicated to customers
  ✓ DDoS attacks or security incidents
  ✓ Infrastructure provider issues
  ✓ Data pipeline delays from external systems

WHEN NOT TO USE CORRECTIONS:
  ✗ Issues caused by your own code or infrastructure
  ✗ Bugs in your application
  ✗ Capacity issues you could have prevented
  ✗ Regular maintenance that wasn't communicated

SRE WORKFLOW:
  1. Incident occurs
  2. Determine if issue was outside your control
  3. Communicate with stakeholders
  4. Create SLO correction with clear description
  5. Document in incident post-mortem
  6. Review corrections quarterly for patterns

NOTES:
  - Corrections affect SLO calculations retroactively
  - All corrections are audited (creator, modifier tracked)
  - Corrections can span multiple SLOs
  - Use clear descriptions for audit trail
  - Review corrections regularly to ensure accuracy
  - Overuse of corrections can indicate systemic issues

For more information: https://docs.datadoghq.com/api/latest/service-level-objectives/
`
	fmt.Println(help)
}
