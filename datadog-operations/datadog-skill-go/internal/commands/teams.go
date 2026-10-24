package commands

import (
	"encoding/json"
	"flag"
	"fmt"

	"github.com/datadog/skill/internal/client"
	"github.com/datadog/skill/internal/observability"
)

// TeamsCommand manages Datadog teams for organization and access control
type TeamsCommand struct {
	flags       *flag.FlagSet
	action      string
	teamID      string
	name        string
	handle      string
	description string
	keyword     string
	filterMe    bool
	sortBy      string
	jsonOut     bool
}

// Team represents a parsed team from Datadog API
type Team struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Handle      string `json:"handle"`
	Description string `json:"description,omitempty"`
	UserCount   int    `json:"user_count,omitempty"`
	LinkCount   int    `json:"link_count,omitempty"`
}

// TeamsResponse represents the formatted teams response
type TeamsResponse struct {
	Status     string `json:"status"`
	TotalTeams int    `json:"total_teams"`
	Teams      []Team `json:"teams,omitempty"`
}

// NewTeamsCommand creates a new teams command
func NewTeamsCommand() *TeamsCommand {
	cmd := &TeamsCommand{
		flags: flag.NewFlagSet("teams", flag.ExitOnError),
	}

	cmd.flags.StringVar(&cmd.action, "action", "list", "Action: list, create, get, update, delete")
	cmd.flags.StringVar(&cmd.teamID, "team-id", "", "Team ID (for get/update/delete)")
	cmd.flags.StringVar(&cmd.name, "name", "", "Team name")
	cmd.flags.StringVar(&cmd.handle, "handle", "", "Team handle (unique identifier)")
	cmd.flags.StringVar(&cmd.description, "description", "", "Team description")
	cmd.flags.StringVar(&cmd.keyword, "keyword", "", "Search by keyword")
	cmd.flags.BoolVar(&cmd.filterMe, "filter-me", false, "Filter to teams current user belongs to")
	cmd.flags.StringVar(&cmd.sortBy, "sort", "name", "Sort by: name, -name, user_count, -user_count")
	cmd.flags.BoolVar(&cmd.jsonOut, "json", false, "Output as JSON")

	return cmd
}

// Name returns the command name
func (c *TeamsCommand) Name() string {
	return "teams"
}

// Description returns the command description
func (c *TeamsCommand) Description() string {
	return "Manage Datadog teams for organization and access control"
}

// Run executes the teams command
func (c *TeamsCommand) Run(args []string) error {
	// Initialize observability
	obs, err := observability.Init("manage-teams", "production")
	if err != nil {
		return fmt.Errorf("failed to init observability: %w", err)
	}
	defer obs.Shutdown(0)

	// Parse flags
	if err := c.flags.Parse(args); err != nil {
		return err
	}

	// Start tracing
	span := obs.StartSpan("teams.manage")
	defer obs.FinishSpan(span)

	obs.LogInfo(fmt.Sprintf("Managing teams with action: %s", c.action))

	// Initialize Datadog client
	ddClient, err := client.NewClient()
	if err != nil {
		return fmt.Errorf("failed to initialize Datadog client: %w", err)
	}

	// Execute action
	switch c.action {
	case "list", "list-teams":
		return c.listTeams(ddClient, obs)
	case "create", "create-team":
		return c.createTeam(ddClient, obs)
	case "get", "get-team":
		return c.getTeam(ddClient, obs)
	case "update", "update-team":
		return c.updateTeam(ddClient, obs)
	case "delete", "delete-team":
		return c.deleteTeam(ddClient, obs)
	default:
		return fmt.Errorf("unknown action: %s (use: list, create, get, update, delete)", c.action)
	}
}

// listTeams lists all teams
func (c *TeamsCommand) listTeams(ddClient *client.Client, obs *observability.Observability) error {
	obs.LogInfo("Listing teams")

	// Query teams
	data, err := ddClient.ListTeams(c.keyword, c.filterMe, c.sortBy)
	if err != nil {
		return fmt.Errorf("failed to list teams: %w", err)
	}

	// Parse and display results
	return c.parseAndDisplayList(data, obs)
}

// createTeam creates a new team
func (c *TeamsCommand) createTeam(ddClient *client.Client, obs *observability.Observability) error {
	obs.LogInfo("Creating team")

	// Validate required fields
	if c.handle == "" {
		return fmt.Errorf("--handle is required for creating team")
	}

	// Build team payload
	payload := c.buildTeamPayload()

	// Create team
	data, err := ddClient.CreateTeam(payload)
	if err != nil {
		return fmt.Errorf("failed to create team: %w", err)
	}

	// Parse and display result
	return c.parseAndDisplaySingle(data, obs)
}

// getTeam gets a specific team
func (c *TeamsCommand) getTeam(ddClient *client.Client, obs *observability.Observability) error {
	if c.teamID == "" {
		return fmt.Errorf("--team-id is required for get action")
	}

	obs.LogInfo(fmt.Sprintf("Getting team: %s", c.teamID))

	// Get team
	data, err := ddClient.GetTeam(c.teamID)
	if err != nil {
		return fmt.Errorf("failed to get team: %w", err)
	}

	// Parse and display result
	return c.parseAndDisplaySingle(data, obs)
}

// updateTeam updates an existing team
func (c *TeamsCommand) updateTeam(ddClient *client.Client, obs *observability.Observability) error {
	if c.teamID == "" {
		return fmt.Errorf("--team-id is required for update action")
	}

	obs.LogInfo(fmt.Sprintf("Updating team: %s", c.teamID))

	// Build team payload
	payload := c.buildTeamPayload()

	// Update team
	data, err := ddClient.UpdateTeam(c.teamID, payload)
	if err != nil {
		return fmt.Errorf("failed to update team: %w", err)
	}

	// Parse and display result
	return c.parseAndDisplaySingle(data, obs)
}

// deleteTeam deletes a team
func (c *TeamsCommand) deleteTeam(ddClient *client.Client, obs *observability.Observability) error {
	if c.teamID == "" {
		return fmt.Errorf("--team-id is required for delete action")
	}

	obs.LogInfo(fmt.Sprintf("Deleting team: %s", c.teamID))

	// Delete team
	err := ddClient.DeleteTeam(c.teamID)
	if err != nil {
		return fmt.Errorf("failed to delete team: %w", err)
	}

	fmt.Printf("✓ Team %s deleted successfully\n", c.teamID)
	return nil
}

// buildTeamPayload builds the API payload for create/update
func (c *TeamsCommand) buildTeamPayload() map[string]interface{} {
	attributes := make(map[string]interface{})

	if c.handle != "" {
		attributes["handle"] = c.handle
	}
	if c.name != "" {
		attributes["name"] = c.name
	}
	if c.description != "" {
		attributes["description"] = c.description
	}

	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type":       "team",
			"attributes": attributes,
		},
	}

	return payload
}

// parseAndDisplayList parses and displays list of teams
func (c *TeamsCommand) parseAndDisplayList(data []byte, obs *observability.Observability) error {
	// Parse JSON response
	var apiResponse struct {
		Data []struct {
			ID         string `json:"id"`
			Type       string `json:"type"`
			Attributes struct {
				Handle      string `json:"handle"`
				Name        string `json:"name"`
				Description string `json:"description"`
				UserCount   int    `json:"user_count"`
				LinkCount   int    `json:"link_count"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &apiResponse); err != nil {
		return fmt.Errorf("failed to parse API response: %w", err)
	}

	// Build response
	response := TeamsResponse{
		Status:     "success",
		TotalTeams: len(apiResponse.Data),
		Teams:      make([]Team, 0),
	}

	// Parse teams
	for _, item := range apiResponse.Data {
		team := Team{
			ID:          item.ID,
			Name:        item.Attributes.Name,
			Handle:      item.Attributes.Handle,
			Description: item.Attributes.Description,
			UserCount:   item.Attributes.UserCount,
			LinkCount:   item.Attributes.LinkCount,
		}

		response.Teams = append(response.Teams, team)
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

// parseAndDisplaySingle parses and displays a single team
func (c *TeamsCommand) parseAndDisplaySingle(data []byte, obs *observability.Observability) error {
	// Parse JSON response
	var apiResponse struct {
		Data struct {
			ID         string `json:"id"`
			Type       string `json:"type"`
			Attributes struct {
				Handle      string `json:"handle"`
				Name        string `json:"name"`
				Description string `json:"description"`
				UserCount   int    `json:"user_count"`
				LinkCount   int    `json:"link_count"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &apiResponse); err != nil {
		return fmt.Errorf("failed to parse API response: %w", err)
	}

	// Build team object
	team := Team{
		ID:          apiResponse.Data.ID,
		Name:        apiResponse.Data.Attributes.Name,
		Handle:      apiResponse.Data.Attributes.Handle,
		Description: apiResponse.Data.Attributes.Description,
		UserCount:   apiResponse.Data.Attributes.UserCount,
		LinkCount:   apiResponse.Data.Attributes.LinkCount,
	}

	// Output result
	if c.jsonOut {
		output, err := json.MarshalIndent(team, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(output))
	} else {
		c.displayFormattedSingle(team)
	}

	return nil
}

// displayFormattedList displays formatted list output
func (c *TeamsCommand) displayFormattedList(response TeamsResponse) {
	fmt.Println("Teams Summary")
	fmt.Println("=============")
	fmt.Printf("Total teams: %d\n", response.TotalTeams)

	if len(response.Teams) > 0 {
		fmt.Println("\nTeams:")
		for i, team := range response.Teams {
			if i >= 20 {
				fmt.Printf("\n... and %d more (use --json for full list)\n", len(response.Teams)-20)
				break
			}

			fmt.Printf("\n[@%s] %s\n", team.Handle, team.Name)
			fmt.Printf("  ID: %s\n", team.ID)
			if team.Description != "" {
				// Truncate long descriptions
				desc := team.Description
				if len(desc) > 100 {
					desc = desc[:97] + "..."
				}
				fmt.Printf("  Description: %s\n", desc)
			}
			fmt.Printf("  Members: %d\n", team.UserCount)
			if team.LinkCount > 0 {
				fmt.Printf("  Links: %d\n", team.LinkCount)
			}
		}
	} else {
		fmt.Println("\nNo teams found.")
	}
}

// displayFormattedSingle displays formatted single team output
func (c *TeamsCommand) displayFormattedSingle(team Team) {
	fmt.Println("Team Details")
	fmt.Println("============")
	fmt.Printf("ID: %s\n", team.ID)
	fmt.Printf("Handle: @%s\n", team.Handle)
	fmt.Printf("Name: %s\n", team.Name)

	if team.Description != "" {
		fmt.Printf("Description: %s\n", team.Description)
	}

	fmt.Printf("Members: %d\n", team.UserCount)

	if team.LinkCount > 0 {
		fmt.Printf("Links: %d\n", team.LinkCount)
	}
}

// Help displays help information
func (c *TeamsCommand) Help() {
	help := `dd teams - Manage Datadog Teams

DESCRIPTION:
  Manage Datadog teams for organization and access control. Teams help organize
  users, assign ownership, and control access to resources.

USAGE:
  dd teams --action <action> [options]

ACTIONS:
  list              List all teams
  create            Create a new team
  get               Get a specific team
  update            Update an existing team
  delete            Delete a team

EXAMPLES:
  # List all teams
  dd teams --action list

  # Search teams by keyword
  dd teams --action list --keyword "backend"

  # Filter to teams current user belongs to
  dd teams --action list --filter-me

  # Sort by user count (descending)
  dd teams --action list --sort -user_count

  # Create new team
  dd teams --action create \
    --handle "backend-team" \
    --name "Backend Engineering" \
    --description "Backend services and APIs"

  # Get specific team
  dd teams --action get --team-id abc123-def456

  # Update team
  dd teams --action update \
    --team-id abc123-def456 \
    --name "Updated Team Name" \
    --description "Updated description"

  # Delete team
  dd teams --action delete --team-id abc123-def456

  # Get JSON output
  dd teams --action list --json

OPTIONS:
  --action          Action to perform (list, create, get, update, delete)
  --team-id         Team ID (required for get/update/delete)
  --name            Team name
  --handle          Team handle (unique identifier, required for create)
  --description     Team description
  --keyword         Search by keyword
  --filter-me       Filter to teams current user belongs to
  --sort            Sort by: name, -name, user_count, -user_count (default: name)
  --json            Output as JSON

NOTES:
  - Team handles must be unique across organization
  - Use handles for tagging and ownership assignment
  - Teams can be assigned to monitors, dashboards, and services
  - Team members inherit access to team resources

For more information: https://docs.datadoghq.com/api/latest/teams/
`
	fmt.Println(help)
}
