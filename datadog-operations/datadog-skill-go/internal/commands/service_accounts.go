package commands

import (
	"encoding/json"
	"flag"
	"fmt"

	"github.com/datadog/skill/internal/client"
	"github.com/datadog/skill/internal/observability"
)

// ServiceAccountsCommand manages Datadog service accounts for automation and API access
type ServiceAccountsCommand struct {
	flags       *flag.FlagSet
	action      string
	accountID   string
	name        string
	email       string
	disabled    bool
	filter      string
	filterAppID string
	jsonOut     bool
}

// ServiceAccount represents a parsed service account from Datadog API
type ServiceAccount struct {
	ID         string   `json:"id"`
	Name       string   `json:"name,omitempty"`
	Email      string   `json:"email"`
	Handle     string   `json:"handle,omitempty"`
	Disabled   bool     `json:"disabled,omitempty"`
	Verified   bool     `json:"verified,omitempty"`
	Roles      []string `json:"roles,omitempty"`
	CreatedAt  string   `json:"created_at,omitempty"`
	Status     string   `json:"status,omitempty"`
	ServiceAccount bool `json:"service_account"`
}

// ServiceAccountsResponse represents the formatted service accounts response
type ServiceAccountsResponse struct {
	Status        string           `json:"status"`
	TotalAccounts int              `json:"total_accounts"`
	Accounts      []ServiceAccount `json:"accounts,omitempty"`
}

// NewServiceAccountsCommand creates a new service accounts command
func NewServiceAccountsCommand() *ServiceAccountsCommand {
	cmd := &ServiceAccountsCommand{
		flags: flag.NewFlagSet("service-accounts", flag.ExitOnError),
	}

	cmd.flags.StringVar(&cmd.action, "action", "list", "Action: list, create, get, update, delete")
	cmd.flags.StringVar(&cmd.accountID, "account-id", "", "Service account ID (for get/update/delete)")
	cmd.flags.StringVar(&cmd.name, "name", "", "Service account name")
	cmd.flags.StringVar(&cmd.email, "email", "", "Service account email")
	cmd.flags.BoolVar(&cmd.disabled, "disabled", false, "Service account disabled status")
	cmd.flags.StringVar(&cmd.filter, "filter", "", "Filter by name or email")
	cmd.flags.StringVar(&cmd.filterAppID, "filter-app-id", "", "Filter by application ID")
	cmd.flags.BoolVar(&cmd.jsonOut, "json", false, "Output as JSON")

	return cmd
}

// Name returns the command name
func (c *ServiceAccountsCommand) Name() string {
	return "service-accounts"
}

// Description returns the command description
func (c *ServiceAccountsCommand) Description() string {
	return "Manage Datadog service accounts for automation and API access"
}

// Run executes the service accounts command
func (c *ServiceAccountsCommand) Run(args []string) error {
	// Initialize observability
	obs, err := observability.Init("manage-service-accounts", "production")
	if err != nil {
		return fmt.Errorf("failed to init observability: %w", err)
	}
	defer obs.Shutdown(0)

	// Parse flags
	if err := c.flags.Parse(args); err != nil {
		return err
	}

	// Start tracing
	span := obs.StartSpan("service_accounts.manage")
	defer obs.FinishSpan(span)

	obs.LogInfo(fmt.Sprintf("Managing service accounts with action: %s", c.action))

	// Initialize Datadog client
	ddClient, err := client.NewClient()
	if err != nil {
		return fmt.Errorf("failed to initialize Datadog client: %w", err)
	}

	// Execute action
	switch c.action {
	case "list", "list-accounts":
		return c.listServiceAccounts(ddClient, obs)
	case "create", "create-account":
		return c.createServiceAccount(ddClient, obs)
	case "get", "get-account":
		return c.getServiceAccount(ddClient, obs)
	case "update", "update-account":
		return c.updateServiceAccount(ddClient, obs)
	case "delete", "delete-account":
		return c.deleteServiceAccount(ddClient, obs)
	default:
		return fmt.Errorf("unknown action: %s (use: list, create, get, update, delete)", c.action)
	}
}

// listServiceAccounts lists all service accounts
func (c *ServiceAccountsCommand) listServiceAccounts(ddClient *client.Client, obs *observability.Observability) error {
	obs.LogInfo("Listing service accounts")

	// Query service accounts
	data, err := ddClient.ListServiceAccounts(c.filter, c.filterAppID)
	if err != nil {
		return fmt.Errorf("failed to list service accounts: %w", err)
	}

	// Parse and display results
	return c.parseAndDisplayList(data, obs)
}

// createServiceAccount creates a new service account
func (c *ServiceAccountsCommand) createServiceAccount(ddClient *client.Client, obs *observability.Observability) error {
	obs.LogInfo("Creating service account")

	// Validate required fields
	if c.email == "" {
		return fmt.Errorf("--email is required for creating service account")
	}

	// Build service account payload
	payload := c.buildServiceAccountPayload()

	// Create service account
	data, err := ddClient.CreateServiceAccount(payload)
	if err != nil {
		return fmt.Errorf("failed to create service account: %w", err)
	}

	// Parse and display result
	return c.parseAndDisplaySingle(data, obs)
}

// getServiceAccount gets a specific service account
func (c *ServiceAccountsCommand) getServiceAccount(ddClient *client.Client, obs *observability.Observability) error {
	if c.accountID == "" {
		return fmt.Errorf("--account-id is required for get action")
	}

	obs.LogInfo(fmt.Sprintf("Getting service account: %s", c.accountID))

	// Get service account
	data, err := ddClient.GetServiceAccount(c.accountID)
	if err != nil {
		return fmt.Errorf("failed to get service account: %w", err)
	}

	// Parse and display result
	return c.parseAndDisplaySingle(data, obs)
}

// updateServiceAccount updates an existing service account
func (c *ServiceAccountsCommand) updateServiceAccount(ddClient *client.Client, obs *observability.Observability) error {
	if c.accountID == "" {
		return fmt.Errorf("--account-id is required for update action")
	}

	obs.LogInfo(fmt.Sprintf("Updating service account: %s", c.accountID))

	// Build service account payload
	payload := c.buildServiceAccountPayload()

	// Update service account
	data, err := ddClient.UpdateServiceAccount(c.accountID, payload)
	if err != nil {
		return fmt.Errorf("failed to update service account: %w", err)
	}

	// Parse and display result
	return c.parseAndDisplaySingle(data, obs)
}

// deleteServiceAccount deletes a service account
func (c *ServiceAccountsCommand) deleteServiceAccount(ddClient *client.Client, obs *observability.Observability) error {
	if c.accountID == "" {
		return fmt.Errorf("--account-id is required for delete action")
	}

	obs.LogInfo(fmt.Sprintf("Deleting service account: %s", c.accountID))

	// Delete service account
	err := ddClient.DeleteServiceAccount(c.accountID)
	if err != nil {
		return fmt.Errorf("failed to delete service account: %w", err)
	}

	fmt.Printf("✓ Service account %s deleted successfully\n", c.accountID)
	return nil
}

// buildServiceAccountPayload builds the API payload for create/update
func (c *ServiceAccountsCommand) buildServiceAccountPayload() map[string]interface{} {
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

	// Mark as service account
	attributes["service_account"] = true

	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type":       "users",
			"attributes": attributes,
		},
	}

	return payload
}

// parseAndDisplayList parses and displays list of service accounts
func (c *ServiceAccountsCommand) parseAndDisplayList(data []byte, obs *observability.Observability) error {
	// Parse JSON response
	var apiResponse struct {
		Data []struct {
			ID         string `json:"id"`
			Type       string `json:"type"`
			Attributes struct {
				Name           string `json:"name"`
				Email          string `json:"email"`
				Handle         string `json:"handle"`
				Verified       bool   `json:"verified"`
				Disabled       bool   `json:"disabled"`
				ServiceAccount bool   `json:"service_account"`
				CreatedAt      string `json:"created_at"`
				Status         string `json:"status"`
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
	response := ServiceAccountsResponse{
		Status:        "success",
		TotalAccounts: 0,
		Accounts:      make([]ServiceAccount, 0),
	}

	// Parse service accounts (filter only service accounts)
	for _, item := range apiResponse.Data {
		// Only include actual service accounts
		if !item.Attributes.ServiceAccount {
			continue
		}

		account := ServiceAccount{
			ID:             item.ID,
			Email:          item.Attributes.Email,
			Name:           item.Attributes.Name,
			Handle:         item.Attributes.Handle,
			Verified:       item.Attributes.Verified,
			Disabled:       item.Attributes.Disabled,
			ServiceAccount: item.Attributes.ServiceAccount,
			CreatedAt:      item.Attributes.CreatedAt,
			Status:         item.Attributes.Status,
			Roles:          make([]string, 0),
		}

		// Extract role IDs
		for _, role := range item.Relationships.Roles.Data {
			account.Roles = append(account.Roles, role.ID)
		}

		response.Accounts = append(response.Accounts, account)
	}

	response.TotalAccounts = len(response.Accounts)

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

// parseAndDisplaySingle parses and displays a single service account
func (c *ServiceAccountsCommand) parseAndDisplaySingle(data []byte, obs *observability.Observability) error {
	// Parse JSON response
	var apiResponse struct {
		Data struct {
			ID         string `json:"id"`
			Type       string `json:"type"`
			Attributes struct {
				Name           string `json:"name"`
				Email          string `json:"email"`
				Handle         string `json:"handle"`
				Verified       bool   `json:"verified"`
				Disabled       bool   `json:"disabled"`
				ServiceAccount bool   `json:"service_account"`
				CreatedAt      string `json:"created_at"`
				Status         string `json:"status"`
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

	// Build service account object
	account := ServiceAccount{
		ID:             apiResponse.Data.ID,
		Email:          apiResponse.Data.Attributes.Email,
		Name:           apiResponse.Data.Attributes.Name,
		Handle:         apiResponse.Data.Attributes.Handle,
		Verified:       apiResponse.Data.Attributes.Verified,
		Disabled:       apiResponse.Data.Attributes.Disabled,
		ServiceAccount: apiResponse.Data.Attributes.ServiceAccount,
		CreatedAt:      apiResponse.Data.Attributes.CreatedAt,
		Status:         apiResponse.Data.Attributes.Status,
		Roles:          make([]string, 0),
	}

	// Extract role IDs
	for _, role := range apiResponse.Data.Relationships.Roles.Data {
		account.Roles = append(account.Roles, role.ID)
	}

	// Output result
	if c.jsonOut {
		output, err := json.MarshalIndent(account, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(output))
	} else {
		c.displayFormattedSingle(account)
	}

	return nil
}

// displayFormattedList displays formatted list output
func (c *ServiceAccountsCommand) displayFormattedList(response ServiceAccountsResponse) {
	fmt.Println("Service Accounts Summary")
	fmt.Println("========================")
	fmt.Printf("Total service accounts: %d\n", response.TotalAccounts)

	if len(response.Accounts) > 0 {
		fmt.Println("\nService Accounts:")
		for i, account := range response.Accounts {
			if i >= 20 {
				fmt.Printf("\n... and %d more (use --json for full list)\n", len(response.Accounts)-20)
				break
			}

			status := "active"
			if account.Disabled {
				status = "disabled"
			} else if !account.Verified {
				status = "pending"
			}

			fmt.Printf("\n[%s] %s\n", account.Handle, account.Email)
			if account.Name != "" {
				fmt.Printf("  Name: %s\n", account.Name)
			}
			fmt.Printf("  Status: %s\n", status)
			if account.CreatedAt != "" {
				fmt.Printf("  Created: %s\n", account.CreatedAt)
			}
			if len(account.Roles) > 0 {
				fmt.Printf("  Roles: %d assigned\n", len(account.Roles))
			}
		}
	} else {
		fmt.Println("\nNo service accounts found.")
	}
}

// displayFormattedSingle displays formatted single service account output
func (c *ServiceAccountsCommand) displayFormattedSingle(account ServiceAccount) {
	fmt.Println("Service Account Details")
	fmt.Println("=======================")
	fmt.Printf("ID: %s\n", account.ID)
	fmt.Printf("Email: %s\n", account.Email)
	fmt.Printf("Handle: %s\n", account.Handle)

	if account.Name != "" {
		fmt.Printf("Name: %s\n", account.Name)
	}

	status := "active"
	if account.Disabled {
		status = "disabled"
	} else if !account.Verified {
		status = "pending verification"
	}
	fmt.Printf("Status: %s\n", status)

	if account.CreatedAt != "" {
		fmt.Printf("Created: %s\n", account.CreatedAt)
	}

	fmt.Printf("Service Account: %t\n", account.ServiceAccount)

	if len(account.Roles) > 0 {
		fmt.Printf("\nRoles (%d):\n", len(account.Roles))
		for _, roleID := range account.Roles {
			fmt.Printf("  - %s\n", roleID)
		}
	}
}

// Help displays help information
func (c *ServiceAccountsCommand) Help() {
	help := `dd service-accounts - Manage Datadog Service Accounts

DESCRIPTION:
  Manage Datadog service accounts for automation and API access. Service accounts
  are non-human users designed for CI/CD pipelines, automation scripts, and
  programmatic access to Datadog APIs.

USAGE:
  dd service-accounts --action <action> [options]

ACTIONS:
  list              List all service accounts
  create            Create a new service account
  get               Get a specific service account
  update            Update an existing service account
  delete            Delete a service account

EXAMPLES:
  # List all service accounts
  dd service-accounts --action list

  # Filter service accounts by name or email
  dd service-accounts --action list --filter "ci-cd"

  # Create new service account
  dd service-accounts --action create \
    --email "ci-pipeline@example.com" \
    --name "CI/CD Pipeline Account"

  # Get specific service account
  dd service-accounts --action get --account-id abc123-def456

  # Update service account name
  dd service-accounts --action update \
    --account-id abc123-def456 \
    --name "Updated Pipeline Account"

  # Disable service account
  dd service-accounts --action update \
    --account-id abc123-def456 \
    --disabled

  # Delete service account
  dd service-accounts --action delete --account-id abc123-def456

  # Get JSON output
  dd service-accounts --action list --json

OPTIONS:
  --action          Action to perform (list, create, get, update, delete)
  --account-id      Service account ID (required for get/update/delete)
  --email           Service account email (required for create)
  --name            Service account name
  --disabled        Service account disabled status
  --filter          Filter by name or email
  --filter-app-id   Filter by application ID
  --json            Output as JSON

NOTES:
  - Service accounts are designed for automation and programmatic access
  - Use service accounts instead of regular user accounts for CI/CD pipelines
  - Service accounts require application keys for API authentication
  - Roles can be assigned to control service account permissions
  - Service accounts appear in audit logs for compliance tracking
  - Disabled service accounts cannot authenticate to Datadog APIs

For more information: https://docs.datadoghq.com/api/latest/service-accounts/
`
	fmt.Println(help)
}
