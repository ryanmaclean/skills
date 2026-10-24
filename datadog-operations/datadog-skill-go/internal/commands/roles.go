package commands

import (
	"encoding/json"
	"flag"
	"fmt"

	"github.com/datadog/skill/internal/client"
	"github.com/datadog/skill/internal/observability"
)

// RolesCommand manages Datadog roles for access control
type RolesCommand struct {
	flags      *flag.FlagSet
	action     string
	roleID     string
	name       string
	permID     string
	userID     string
	jsonOut    bool
}

// Role represents a parsed role from Datadog API
type Role struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	UserCount   int      `json:"user_count,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}

// Permission represents a parsed permission from Datadog API
type Permission struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name,omitempty"`
	Description string `json:"description,omitempty"`
	Group       string `json:"group,omitempty"`
}

// RolesResponse represents the formatted roles response
type RolesResponse struct {
	Status     string `json:"status"`
	TotalRoles int    `json:"total_roles"`
	Roles      []Role `json:"roles,omitempty"`
}

// PermissionsResponse represents the formatted permissions response
type PermissionsResponse struct {
	Status          string       `json:"status"`
	TotalPermissions int         `json:"total_permissions"`
	Permissions     []Permission `json:"permissions,omitempty"`
}

// NewRolesCommand creates a new roles command
func NewRolesCommand() *RolesCommand {
	cmd := &RolesCommand{
		flags: flag.NewFlagSet("roles", flag.ExitOnError),
	}

	cmd.flags.StringVar(&cmd.action, "action", "list", "Action: list, get, create, delete, list-permissions, add-permission, remove-permission")
	cmd.flags.StringVar(&cmd.roleID, "role-id", "", "Role ID (for get/delete/permissions)")
	cmd.flags.StringVar(&cmd.name, "name", "", "Role name")
	cmd.flags.StringVar(&cmd.permID, "permission-id", "", "Permission ID (for add/remove-permission)")
	cmd.flags.StringVar(&cmd.userID, "user-id", "", "User ID (for add-user)")
	cmd.flags.BoolVar(&cmd.jsonOut, "json", false, "Output as JSON")

	return cmd
}

// Name returns the command name
func (c *RolesCommand) Name() string {
	return "roles"
}

// Description returns the command description
func (c *RolesCommand) Description() string {
	return "Manage Datadog roles for access control"
}

// Run executes the roles command
func (c *RolesCommand) Run(args []string) error {
	// Initialize observability
	obs, err := observability.Init("manage-roles", "production")
	if err != nil {
		return fmt.Errorf("failed to init observability: %w", err)
	}
	defer obs.Shutdown(0)

	// Parse flags
	if err := c.flags.Parse(args); err != nil {
		return err
	}

	// Start tracing
	span := obs.StartSpan("roles.manage")
	defer obs.FinishSpan(span)

	obs.LogInfo(fmt.Sprintf("Managing roles with action: %s", c.action))

	// Initialize Datadog client
	ddClient, err := client.NewClient()
	if err != nil {
		return fmt.Errorf("failed to initialize Datadog client: %w", err)
	}

	// Execute action
	switch c.action {
	case "list", "list-roles":
		return c.listRoles(ddClient, obs)
	case "get", "get-role":
		return c.getRole(ddClient, obs)
	case "create", "create-role":
		return c.createRole(ddClient, obs)
	case "delete", "delete-role":
		return c.deleteRole(ddClient, obs)
	case "list-permissions", "permissions":
		return c.listPermissions(ddClient, obs)
	case "add-permission":
		return c.addPermission(ddClient, obs)
	case "remove-permission":
		return c.removePermission(ddClient, obs)
	default:
		return fmt.Errorf("unknown action: %s (use: list, get, create, delete, list-permissions, add-permission, remove-permission)", c.action)
	}
}

// listRoles lists all roles
func (c *RolesCommand) listRoles(ddClient *client.Client, obs *observability.Observability) error {
	obs.LogInfo("Listing roles")

	// Query roles
	data, err := ddClient.ListRoles()
	if err != nil {
		return fmt.Errorf("failed to list roles: %w", err)
	}

	// Parse and display results
	return c.parseAndDisplayList(data, obs)
}

// getRole gets a specific role
func (c *RolesCommand) getRole(ddClient *client.Client, obs *observability.Observability) error {
	if c.roleID == "" {
		return fmt.Errorf("--role-id is required for get action")
	}

	obs.LogInfo(fmt.Sprintf("Getting role: %s", c.roleID))

	// Get role
	data, err := ddClient.GetRole(c.roleID)
	if err != nil {
		return fmt.Errorf("failed to get role: %w", err)
	}

	// Parse and display result
	return c.parseAndDisplaySingle(data, obs)
}

// createRole creates a new role
func (c *RolesCommand) createRole(ddClient *client.Client, obs *observability.Observability) error {
	if c.name == "" {
		return fmt.Errorf("--name is required for creating role")
	}

	obs.LogInfo("Creating role")

	// Build role payload
	payload := c.buildRolePayload()

	// Create role
	data, err := ddClient.CreateRole(payload)
	if err != nil {
		return fmt.Errorf("failed to create role: %w", err)
	}

	// Parse and display result
	return c.parseAndDisplaySingle(data, obs)
}

// deleteRole deletes a role
func (c *RolesCommand) deleteRole(ddClient *client.Client, obs *observability.Observability) error {
	if c.roleID == "" {
		return fmt.Errorf("--role-id is required for delete action")
	}

	obs.LogInfo(fmt.Sprintf("Deleting role: %s", c.roleID))

	// Delete role
	err := ddClient.DeleteRole(c.roleID)
	if err != nil {
		return fmt.Errorf("failed to delete role: %w", err)
	}

	fmt.Printf("✓ Role %s deleted successfully\n", c.roleID)
	return nil
}

// listPermissions lists permissions for a role or all permissions
func (c *RolesCommand) listPermissions(ddClient *client.Client, obs *observability.Observability) error {
	obs.LogInfo("Listing permissions")

	var data []byte
	var err error

	if c.roleID != "" {
		// List permissions for specific role
		obs.LogInfo(fmt.Sprintf("Listing permissions for role: %s", c.roleID))
		data, err = ddClient.ListRolePermissions(c.roleID)
	} else {
		// List all available permissions
		obs.LogInfo("Listing all available permissions")
		data, err = ddClient.ListAllPermissions()
	}

	if err != nil {
		return fmt.Errorf("failed to list permissions: %w", err)
	}

	// Parse and display permissions
	return c.parseAndDisplayPermissions(data, obs)
}

// addPermission adds a permission to a role
func (c *RolesCommand) addPermission(ddClient *client.Client, obs *observability.Observability) error {
	if c.roleID == "" {
		return fmt.Errorf("--role-id is required for add-permission action")
	}
	if c.permID == "" {
		return fmt.Errorf("--permission-id is required for add-permission action")
	}

	obs.LogInfo(fmt.Sprintf("Adding permission %s to role %s", c.permID, c.roleID))

	// Add permission
	err := ddClient.AddRolePermission(c.roleID, c.permID)
	if err != nil {
		return fmt.Errorf("failed to add permission: %w", err)
	}

	fmt.Printf("✓ Permission %s added to role %s successfully\n", c.permID, c.roleID)
	return nil
}

// removePermission removes a permission from a role
func (c *RolesCommand) removePermission(ddClient *client.Client, obs *observability.Observability) error {
	if c.roleID == "" {
		return fmt.Errorf("--role-id is required for remove-permission action")
	}
	if c.permID == "" {
		return fmt.Errorf("--permission-id is required for remove-permission action")
	}

	obs.LogInfo(fmt.Sprintf("Removing permission %s from role %s", c.permID, c.roleID))

	// Remove permission
	err := ddClient.RemoveRolePermission(c.roleID, c.permID)
	if err != nil {
		return fmt.Errorf("failed to remove permission: %w", err)
	}

	fmt.Printf("✓ Permission %s removed from role %s successfully\n", c.permID, c.roleID)
	return nil
}

// buildRolePayload builds the API payload for create
func (c *RolesCommand) buildRolePayload() map[string]interface{} {
	attributes := make(map[string]interface{})

	if c.name != "" {
		attributes["name"] = c.name
	}

	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type":       "roles",
			"attributes": attributes,
		},
	}

	return payload
}

// parseAndDisplayList parses and displays list of roles
func (c *RolesCommand) parseAndDisplayList(data []byte, obs *observability.Observability) error {
	// Parse JSON response
	var apiResponse struct {
		Data []struct {
			ID         string `json:"id"`
			Type       string `json:"type"`
			Attributes struct {
				Name      string `json:"name"`
				UserCount int    `json:"user_count"`
			} `json:"attributes"`
			Relationships struct {
				Permissions struct {
					Data []struct {
						ID   string `json:"id"`
						Type string `json:"type"`
					} `json:"data"`
				} `json:"permissions"`
			} `json:"relationships"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &apiResponse); err != nil {
		return fmt.Errorf("failed to parse API response: %w", err)
	}

	// Build response
	response := RolesResponse{
		Status:     "success",
		TotalRoles: len(apiResponse.Data),
		Roles:      make([]Role, 0),
	}

	// Parse roles
	for _, item := range apiResponse.Data {
		role := Role{
			ID:          item.ID,
			Name:        item.Attributes.Name,
			UserCount:   item.Attributes.UserCount,
			Permissions: make([]string, 0),
		}

		// Extract permission IDs
		for _, perm := range item.Relationships.Permissions.Data {
			role.Permissions = append(role.Permissions, perm.ID)
		}

		response.Roles = append(response.Roles, role)
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

// parseAndDisplaySingle parses and displays a single role
func (c *RolesCommand) parseAndDisplaySingle(data []byte, obs *observability.Observability) error {
	// Parse JSON response
	var apiResponse struct {
		Data struct {
			ID         string `json:"id"`
			Type       string `json:"type"`
			Attributes struct {
				Name      string `json:"name"`
				UserCount int    `json:"user_count"`
			} `json:"attributes"`
			Relationships struct {
				Permissions struct {
					Data []struct {
						ID   string `json:"id"`
						Type string `json:"type"`
					} `json:"data"`
				} `json:"permissions"`
			} `json:"relationships"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &apiResponse); err != nil {
		return fmt.Errorf("failed to parse API response: %w", err)
	}

	// Build role object
	role := Role{
		ID:          apiResponse.Data.ID,
		Name:        apiResponse.Data.Attributes.Name,
		UserCount:   apiResponse.Data.Attributes.UserCount,
		Permissions: make([]string, 0),
	}

	// Extract permission IDs
	for _, perm := range apiResponse.Data.Relationships.Permissions.Data {
		role.Permissions = append(role.Permissions, perm.ID)
	}

	// Output result
	if c.jsonOut {
		output, err := json.MarshalIndent(role, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(output))
	} else {
		c.displayFormattedSingle(role)
	}

	return nil
}

// parseAndDisplayPermissions parses and displays permissions
func (c *RolesCommand) parseAndDisplayPermissions(data []byte, obs *observability.Observability) error {
	// Parse JSON response
	var apiResponse struct {
		Data []struct {
			ID         string `json:"id"`
			Type       string `json:"type"`
			Attributes struct {
				Name        string `json:"name"`
				DisplayName string `json:"display_name"`
				Description string `json:"description"`
				GroupName   string `json:"group_name"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &apiResponse); err != nil {
		return fmt.Errorf("failed to parse API response: %w", err)
	}

	// Build response
	response := PermissionsResponse{
		Status:          "success",
		TotalPermissions: len(apiResponse.Data),
		Permissions:     make([]Permission, 0),
	}

	// Parse permissions
	for _, item := range apiResponse.Data {
		perm := Permission{
			ID:          item.ID,
			Name:        item.Attributes.Name,
			DisplayName: item.Attributes.DisplayName,
			Description: item.Attributes.Description,
			Group:       item.Attributes.GroupName,
		}

		response.Permissions = append(response.Permissions, perm)
	}

	// Output results
	if c.jsonOut {
		output, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(output))
	} else {
		c.displayFormattedPermissions(response)
	}

	return nil
}

// displayFormattedList displays formatted list output
func (c *RolesCommand) displayFormattedList(response RolesResponse) {
	fmt.Println("Roles Summary")
	fmt.Println("=============")
	fmt.Printf("Total roles: %d\n", response.TotalRoles)

	if len(response.Roles) > 0 {
		fmt.Println("\nRoles:")
		for _, role := range response.Roles {
			fmt.Printf("\n%s\n", role.Name)
			fmt.Printf("  ID: %s\n", role.ID)
			fmt.Printf("  Users: %d\n", role.UserCount)
			fmt.Printf("  Permissions: %d\n", len(role.Permissions))
		}
	} else {
		fmt.Println("\nNo roles found.")
	}
}

// displayFormattedSingle displays formatted single role output
func (c *RolesCommand) displayFormattedSingle(role Role) {
	fmt.Println("Role Details")
	fmt.Println("============")
	fmt.Printf("ID: %s\n", role.ID)
	fmt.Printf("Name: %s\n", role.Name)
	fmt.Printf("Users: %d\n", role.UserCount)

	if len(role.Permissions) > 0 {
		fmt.Printf("\nPermissions (%d):\n", len(role.Permissions))
		for i, permID := range role.Permissions {
			if i >= 20 {
				fmt.Printf("... and %d more (use --json for full list)\n", len(role.Permissions)-20)
				break
			}
			fmt.Printf("  - %s\n", permID)
		}
	}
}

// displayFormattedPermissions displays formatted permissions output
func (c *RolesCommand) displayFormattedPermissions(response PermissionsResponse) {
	fmt.Println("Permissions Summary")
	fmt.Println("===================")
	fmt.Printf("Total permissions: %d\n", response.TotalPermissions)

	if len(response.Permissions) > 0 {
		fmt.Println("\nPermissions:")
		for i, perm := range response.Permissions {
			if i >= 30 {
				fmt.Printf("\n... and %d more (use --json for full list)\n", len(response.Permissions)-30)
				break
			}

			fmt.Printf("\n%s\n", perm.DisplayName)
			fmt.Printf("  ID: %s\n", perm.ID)
			if perm.Group != "" {
				fmt.Printf("  Group: %s\n", perm.Group)
			}
			if perm.Description != "" {
				// Truncate long descriptions
				desc := perm.Description
				if len(desc) > 80 {
					desc = desc[:77] + "..."
				}
				fmt.Printf("  Description: %s\n", desc)
			}
		}
	} else {
		fmt.Println("\nNo permissions found.")
	}
}

// Help displays help information
func (c *RolesCommand) Help() {
	help := `dd roles - Manage Datadog Roles

DESCRIPTION:
  Manage Datadog roles for access control. Roles define sets of permissions
  that can be assigned to users for fine-grained access control.

USAGE:
  dd roles --action <action> [options]

ACTIONS:
  list                  List all roles
  get                   Get a specific role
  create                Create a new role
  delete                Delete a role
  list-permissions      List permissions (all or for specific role)
  add-permission        Add a permission to a role
  remove-permission     Remove a permission from a role

EXAMPLES:
  # List all roles
  dd roles --action list

  # Get specific role with permissions
  dd roles --action get --role-id abc123-def456

  # Create new role
  dd roles --action create --name "DevOps Engineer"

  # Delete role
  dd roles --action delete --role-id abc123-def456

  # List all available permissions
  dd roles --action list-permissions

  # List permissions for specific role
  dd roles --action list-permissions --role-id abc123-def456

  # Add permission to role
  dd roles --action add-permission \
    --role-id abc123-def456 \
    --permission-id logs_read_data

  # Remove permission from role
  dd roles --action remove-permission \
    --role-id abc123-def456 \
    --permission-id logs_write_exclusion_filters

  # Get JSON output
  dd roles --action list --json

OPTIONS:
  --action          Action to perform
  --role-id         Role ID (required for get/delete/permissions)
  --name            Role name (required for create)
  --permission-id   Permission ID (for add/remove-permission)
  --user-id         User ID (for add-user)
  --json            Output as JSON

NOTES:
  - Roles require organization admin privileges to manage
  - Default roles (Admin, Standard, Read Only) cannot be deleted
  - Permission IDs can be found using list-permissions action
  - Users can be assigned multiple roles
  - Permissions are additive across all assigned roles

For more information: https://docs.datadoghq.com/api/latest/roles/
`
	fmt.Println(help)
}
