package commands

import (
	"encoding/json"
	"flag"
	"fmt"

	"github.com/datadog/skill/internal/client"
	"github.com/datadog/skill/internal/observability"
)

// UsersCommand manages Datadog users for access control and administration
type UsersCommand struct {
	flags    *flag.FlagSet
	action   string
	userID   string
	email    string
	name     string
	disabled bool
	filter   string
	jsonOut  bool
}

// User represents a parsed user from Datadog API
type User struct {
	ID       string   `json:"id"`
	Email    string   `json:"email"`
	Name     string   `json:"name,omitempty"`
	Handle   string   `json:"handle,omitempty"`
	Verified bool     `json:"verified,omitempty"`
	Disabled bool     `json:"disabled,omitempty"`
	Roles    []string `json:"roles,omitempty"`
	Title    string   `json:"title,omitempty"`
}

// UsersResponse represents the formatted users response
type UsersResponse struct {
	Status     string `json:"status"`
	TotalUsers int    `json:"total_users"`
	Users      []User `json:"users,omitempty"`
}

// NewUsersCommand creates a new users command
func NewUsersCommand() *UsersCommand {
	cmd := &UsersCommand{
		flags: flag.NewFlagSet("users", flag.ExitOnError),
	}

	cmd.flags.StringVar(&cmd.action, "action", "list", "Action: list, get, update, disable")
	cmd.flags.StringVar(&cmd.userID, "user-id", "", "User ID (for get/update/disable)")
	cmd.flags.StringVar(&cmd.email, "email", "", "User email")
	cmd.flags.StringVar(&cmd.name, "name", "", "User name")
	cmd.flags.BoolVar(&cmd.disabled, "disabled", false, "User disabled status")
	cmd.flags.StringVar(&cmd.filter, "filter", "", "Filter by email, name, or status")
	cmd.flags.BoolVar(&cmd.jsonOut, "json", false, "Output as JSON")

	return cmd
}

// Name returns the command name
func (c *UsersCommand) Name() string {
	return "users"
}

// Description returns the command description
func (c *UsersCommand) Description() string {
	return "Manage Datadog users for access control and administration"
}

// Run executes the users command
func (c *UsersCommand) Run(args []string) error {
	// Initialize observability
	obs, err := observability.Init("manage-users", "production")
	if err != nil {
		return fmt.Errorf("failed to init observability: %w", err)
	}
	defer obs.Shutdown(0)

	// Parse flags
	if err := c.flags.Parse(args); err != nil {
		return err
	}

	// Start tracing
	span := obs.StartSpan("users.manage")
	defer obs.FinishSpan(span)

	obs.LogInfo(fmt.Sprintf("Managing users with action: %s", c.action))

	// Initialize Datadog client
	ddClient, err := client.NewClient()
	if err != nil {
		return fmt.Errorf("failed to initialize Datadog client: %w", err)
	}

	// Execute action
	switch c.action {
	case "list", "list-users":
		return c.listUsers(ddClient, obs)
	case "get", "get-user":
		return c.getUser(ddClient, obs)
	case "update", "update-user":
		return c.updateUser(ddClient, obs)
	case "disable", "disable-user":
		return c.disableUser(ddClient, obs)
	default:
		return fmt.Errorf("unknown action: %s (use: list, get, update, disable)", c.action)
	}
}

// listUsers lists all users
func (c *UsersCommand) listUsers(ddClient *client.Client, obs *observability.Observability) error {
	obs.LogInfo("Listing users")

	// Query users
	data, err := ddClient.ListUsers(c.filter)
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}

	// Parse and display results
	return c.parseAndDisplayList(data, obs)
}

// getUser gets a specific user
func (c *UsersCommand) getUser(ddClient *client.Client, obs *observability.Observability) error {
	if c.userID == "" {
		return fmt.Errorf("--user-id is required for get action")
	}

	obs.LogInfo(fmt.Sprintf("Getting user: %s", c.userID))

	// Get user
	data, err := ddClient.GetUser(c.userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Parse and display result
	return c.parseAndDisplaySingle(data, obs)
}

// updateUser updates an existing user
func (c *UsersCommand) updateUser(ddClient *client.Client, obs *observability.Observability) error {
	if c.userID == "" {
		return fmt.Errorf("--user-id is required for update action")
	}

	obs.LogInfo(fmt.Sprintf("Updating user: %s", c.userID))

	// Build user payload
	payload := c.buildUserPayload()

	// Update user
	data, err := ddClient.UpdateUser(c.userID, payload)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	// Parse and display result
	return c.parseAndDisplaySingle(data, obs)
}

// disableUser disables a user
func (c *UsersCommand) disableUser(ddClient *client.Client, obs *observability.Observability) error {
	if c.userID == "" {
		return fmt.Errorf("--user-id is required for disable action")
	}

	obs.LogInfo(fmt.Sprintf("Disabling user: %s", c.userID))

	// Disable user
	err := ddClient.DisableUser(c.userID)
	if err != nil {
		return fmt.Errorf("failed to disable user: %w", err)
	}

	fmt.Printf("✓ User %s disabled successfully\n", c.userID)
	return nil
}

// buildUserPayload builds the API payload for update
func (c *UsersCommand) buildUserPayload() map[string]interface{} {
	attributes := make(map[string]interface{})

	if c.name != "" {
		attributes["name"] = c.name
	}
	if c.email != "" {
		attributes["email"] = c.email
	}
	if c.disabled {
		attributes["disabled"] = c.disabled
	}

	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type":       "users",
			"id":         c.userID,
			"attributes": attributes,
		},
	}

	return payload
}

// parseAndDisplayList parses and displays list of users
func (c *UsersCommand) parseAndDisplayList(data []byte, obs *observability.Observability) error {
	// Parse JSON response
	var apiResponse struct {
		Data []struct {
			ID         string `json:"id"`
			Type       string `json:"type"`
			Attributes struct {
				Name     string `json:"name"`
				Email    string `json:"email"`
				Handle   string `json:"handle"`
				Verified bool   `json:"verified"`
				Disabled bool   `json:"disabled"`
				Title    string `json:"title"`
			} `json:"attributes"`
			Relationships struct {
				Roles struct {
					Data []struct {
						ID   string `json:"id"`
						Type string `json:"type"`
					} `json:"data"`
				} `json:"roles"`
			} `json:"relationships"`
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
	response := UsersResponse{
		Status:     "success",
		TotalUsers: apiResponse.Meta.Page.TotalCount,
		Users:      make([]User, 0),
	}

	// Parse users
	for _, item := range apiResponse.Data {
		user := User{
			ID:       item.ID,
			Email:    item.Attributes.Email,
			Name:     item.Attributes.Name,
			Handle:   item.Attributes.Handle,
			Verified: item.Attributes.Verified,
			Disabled: item.Attributes.Disabled,
			Title:    item.Attributes.Title,
			Roles:    make([]string, 0),
		}

		// Extract role IDs
		for _, role := range item.Relationships.Roles.Data {
			user.Roles = append(user.Roles, role.ID)
		}

		response.Users = append(response.Users, user)
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

// parseAndDisplaySingle parses and displays a single user
func (c *UsersCommand) parseAndDisplaySingle(data []byte, obs *observability.Observability) error {
	// Parse JSON response
	var apiResponse struct {
		Data struct {
			ID         string `json:"id"`
			Type       string `json:"type"`
			Attributes struct {
				Name     string `json:"name"`
				Email    string `json:"email"`
				Handle   string `json:"handle"`
				Verified bool   `json:"verified"`
				Disabled bool   `json:"disabled"`
				Title    string `json:"title"`
			} `json:"attributes"`
			Relationships struct {
				Roles struct {
					Data []struct {
						ID   string `json:"id"`
						Type string `json:"type"`
					} `json:"data"`
				} `json:"roles"`
			} `json:"relationships"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &apiResponse); err != nil {
		return fmt.Errorf("failed to parse API response: %w", err)
	}

	// Build user object
	user := User{
		ID:       apiResponse.Data.ID,
		Email:    apiResponse.Data.Attributes.Email,
		Name:     apiResponse.Data.Attributes.Name,
		Handle:   apiResponse.Data.Attributes.Handle,
		Verified: apiResponse.Data.Attributes.Verified,
		Disabled: apiResponse.Data.Attributes.Disabled,
		Title:    apiResponse.Data.Attributes.Title,
		Roles:    make([]string, 0),
	}

	// Extract role IDs
	for _, role := range apiResponse.Data.Relationships.Roles.Data {
		user.Roles = append(user.Roles, role.ID)
	}

	// Output result
	if c.jsonOut {
		output, err := json.MarshalIndent(user, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(output))
	} else {
		c.displayFormattedSingle(user)
	}

	return nil
}

// displayFormattedList displays formatted list output
func (c *UsersCommand) displayFormattedList(response UsersResponse) {
	fmt.Println("Users Summary")
	fmt.Println("=============")
	fmt.Printf("Total users: %d\n", response.TotalUsers)

	if len(response.Users) > 0 {
		fmt.Println("\nUsers:")
		for i, user := range response.Users {
			if i >= 20 {
				fmt.Printf("\n... and %d more (use --json for full list)\n", len(response.Users)-20)
				break
			}

			status := "active"
			if user.Disabled {
				status = "disabled"
			} else if !user.Verified {
				status = "pending"
			}

			fmt.Printf("\n[%s] %s\n", user.Handle, user.Email)
			if user.Name != "" {
				fmt.Printf("  Name: %s\n", user.Name)
			}
			fmt.Printf("  Status: %s\n", status)
			if user.Title != "" {
				fmt.Printf("  Title: %s\n", user.Title)
			}
			if len(user.Roles) > 0 {
				fmt.Printf("  Roles: %d assigned\n", len(user.Roles))
			}
		}
	} else {
		fmt.Println("\nNo users found.")
	}
}

// displayFormattedSingle displays formatted single user output
func (c *UsersCommand) displayFormattedSingle(user User) {
	fmt.Println("User Details")
	fmt.Println("============")
	fmt.Printf("ID: %s\n", user.ID)
	fmt.Printf("Email: %s\n", user.Email)
	fmt.Printf("Handle: %s\n", user.Handle)

	if user.Name != "" {
		fmt.Printf("Name: %s\n", user.Name)
	}

	if user.Title != "" {
		fmt.Printf("Title: %s\n", user.Title)
	}

	status := "active"
	if user.Disabled {
		status = "disabled"
	} else if !user.Verified {
		status = "pending verification"
	}
	fmt.Printf("Status: %s\n", status)

	if len(user.Roles) > 0 {
		fmt.Printf("\nRoles (%d):\n", len(user.Roles))
		for _, roleID := range user.Roles {
			fmt.Printf("  - %s\n", roleID)
		}
	}
}

// Help displays help information
func (c *UsersCommand) Help() {
	help := `dd users - Manage Datadog Users

DESCRIPTION:
  Manage Datadog users for access control and administration. View user details,
  update user information, and manage user access status.

USAGE:
  dd users --action <action> [options]

ACTIONS:
  list              List all users
  get               Get a specific user
  update            Update an existing user
  disable           Disable a user account

EXAMPLES:
  # List all users
  dd users --action list

  # Filter users by email or name
  dd users --action list --filter "engineer"

  # Get specific user
  dd users --action get --user-id abc123-def456

  # Update user name
  dd users --action update \
    --user-id abc123-def456 \
    --name "John Doe"

  # Disable user account
  dd users --action disable --user-id abc123-def456

  # Get JSON output
  dd users --action list --json

OPTIONS:
  --action          Action to perform (list, get, update, disable)
  --user-id         User ID (required for get/update/disable)
  --email           User email
  --name            User name
  --disabled        User disabled status
  --filter          Filter by email, name, or status
  --json            Output as JSON

NOTES:
  - User creation is managed through Datadog UI or invitation system
  - Disable action requires administrator privileges
  - Updates may require specific permissions
  - User handles are unique identifiers within the organization
  - Roles are managed separately via roles command

For more information: https://docs.datadoghq.com/api/latest/users/
`
	fmt.Println(help)
}
