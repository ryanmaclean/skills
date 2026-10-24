package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"time"

	"github.com/datadog/skill/internal/client"
	"github.com/datadog/skill/internal/observability"
)

// ApplicationKeysCommand manages Datadog application keys for API authentication
type ApplicationKeysCommand struct {
	flags     *flag.FlagSet
	action    string
	keyID     string
	name      string
	scopes    string
	ownerID   string
	filterAll bool
	jsonOut   bool
}

// ApplicationKey represents a parsed application key from Datadog API
type ApplicationKey struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Hash      string   `json:"hash,omitempty"`
	Key       string   `json:"key,omitempty"`
	Owner     string   `json:"owner,omitempty"`
	CreatedAt string   `json:"created_at,omitempty"`
	Scopes    []string `json:"scopes,omitempty"`
}

// ApplicationKeysResponse represents the formatted application keys response
type ApplicationKeysResponse struct {
	Status    string           `json:"status"`
	TotalKeys int              `json:"total_keys"`
	Keys      []ApplicationKey `json:"keys,omitempty"`
}

// NewApplicationKeysCommand creates a new application keys command
func NewApplicationKeysCommand() *ApplicationKeysCommand {
	cmd := &ApplicationKeysCommand{
		flags: flag.NewFlagSet("application-keys", flag.ExitOnError),
	}

	cmd.flags.StringVar(&cmd.action, "action", "list", "Action: list, create, get, update, delete")
	cmd.flags.StringVar(&cmd.keyID, "key-id", "", "Application key ID (for get/update/delete)")
	cmd.flags.StringVar(&cmd.name, "name", "", "Application key name")
	cmd.flags.StringVar(&cmd.scopes, "scopes", "", "Comma-separated list of scopes")
	cmd.flags.StringVar(&cmd.ownerID, "owner-id", "", "Owner user ID (service account)")
	cmd.flags.BoolVar(&cmd.filterAll, "all", false, "List all org keys (admin only)")
	cmd.flags.BoolVar(&cmd.jsonOut, "json", false, "Output as JSON")

	return cmd
}

// Name returns the command name
func (c *ApplicationKeysCommand) Name() string {
	return "application-keys"
}

// Description returns the command description
func (c *ApplicationKeysCommand) Description() string {
	return "Manage Datadog application keys for API authentication"
}

// Run executes the application keys command
func (c *ApplicationKeysCommand) Run(args []string) error {
	// Initialize observability
	obs, err := observability.Init("manage-application-keys", "production")
	if err != nil {
		return fmt.Errorf("failed to init observability: %w", err)
	}
	defer obs.Shutdown(0)

	// Parse flags
	if err := c.flags.Parse(args); err != nil {
		return err
	}

	// Start tracing
	span := obs.StartSpan("application_keys.manage")
	defer obs.FinishSpan(span)

	obs.LogInfo(fmt.Sprintf("Managing application keys with action: %s", c.action))

	// Initialize Datadog client
	ddClient, err := client.NewClient()
	if err != nil {
		return fmt.Errorf("failed to initialize Datadog client: %w", err)
	}

	// Execute action
	switch c.action {
	case "list", "list-keys":
		return c.listApplicationKeys(ddClient, obs)
	case "create", "create-key":
		return c.createApplicationKey(ddClient, obs)
	case "get", "get-key":
		return c.getApplicationKey(ddClient, obs)
	case "update", "update-key":
		return c.updateApplicationKey(ddClient, obs)
	case "delete", "delete-key":
		return c.deleteApplicationKey(ddClient, obs)
	default:
		return fmt.Errorf("unknown action: %s (use: list, create, get, update, delete)", c.action)
	}
}

// listApplicationKeys lists all application keys
func (c *ApplicationKeysCommand) listApplicationKeys(ddClient *client.Client, obs *observability.Observability) error {
	obs.LogInfo("Listing application keys")

	// Query application keys
	data, err := ddClient.ListApplicationKeys(c.filterAll)
	if err != nil {
		return fmt.Errorf("failed to list application keys: %w", err)
	}

	// Parse and display results
	return c.parseAndDisplayList(data, obs)
}

// createApplicationKey creates a new application key
func (c *ApplicationKeysCommand) createApplicationKey(ddClient *client.Client, obs *observability.Observability) error {
	obs.LogInfo("Creating application key")

	// Validate required fields
	if c.name == "" {
		return fmt.Errorf("--name is required for creating application key")
	}

	// Build application key payload
	payload := c.buildApplicationKeyPayload()

	// Create application key
	data, err := ddClient.CreateApplicationKey(payload)
	if err != nil {
		return fmt.Errorf("failed to create application key: %w", err)
	}

	// Parse and display result (includes actual key value - shown only once!)
	return c.parseAndDisplaySingle(data, obs, true)
}

// getApplicationKey gets a specific application key
func (c *ApplicationKeysCommand) getApplicationKey(ddClient *client.Client, obs *observability.Observability) error {
	if c.keyID == "" {
		return fmt.Errorf("--key-id is required for get action")
	}

	obs.LogInfo(fmt.Sprintf("Getting application key: %s", c.keyID))

	// Get application key
	data, err := ddClient.GetApplicationKey(c.keyID)
	if err != nil {
		return fmt.Errorf("failed to get application key: %w", err)
	}

	// Parse and display result (key value not included in get response)
	return c.parseAndDisplaySingle(data, obs, false)
}

// updateApplicationKey updates an existing application key
func (c *ApplicationKeysCommand) updateApplicationKey(ddClient *client.Client, obs *observability.Observability) error {
	if c.keyID == "" {
		return fmt.Errorf("--key-id is required for update action")
	}

	obs.LogInfo(fmt.Sprintf("Updating application key: %s", c.keyID))

	// Build application key payload
	payload := c.buildApplicationKeyPayload()

	// Update application key
	data, err := ddClient.UpdateApplicationKey(c.keyID, payload)
	if err != nil {
		return fmt.Errorf("failed to update application key: %w", err)
	}

	// Parse and display result
	return c.parseAndDisplaySingle(data, obs, false)
}

// deleteApplicationKey deletes an application key
func (c *ApplicationKeysCommand) deleteApplicationKey(ddClient *client.Client, obs *observability.Observability) error {
	if c.keyID == "" {
		return fmt.Errorf("--key-id is required for delete action")
	}

	obs.LogInfo(fmt.Sprintf("Deleting application key: %s", c.keyID))

	// Delete application key
	err := ddClient.DeleteApplicationKey(c.keyID)
	if err != nil {
		return fmt.Errorf("failed to delete application key: %w", err)
	}

	fmt.Printf("✓ Application key %s deleted successfully\n", c.keyID)
	return nil
}

// buildApplicationKeyPayload builds the API payload for create/update
func (c *ApplicationKeysCommand) buildApplicationKeyPayload() map[string]interface{} {
	attributes := make(map[string]interface{})

	if c.name != "" {
		attributes["name"] = c.name
	}

	// Parse scopes if provided
	if c.scopes != "" {
		// Split comma-separated scopes
		scopesList := []string{}
		for _, scope := range splitScopes(c.scopes) {
			if scope != "" {
				scopesList = append(scopesList, scope)
			}
		}
		if len(scopesList) > 0 {
			attributes["scopes"] = scopesList
		}
	}

	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type":       "application_keys",
			"attributes": attributes,
		},
	}

	// Add owner relationship if specified (for service accounts)
	if c.ownerID != "" {
		payload["data"].(map[string]interface{})["relationships"] = map[string]interface{}{
			"owned_by": map[string]interface{}{
				"data": map[string]interface{}{
					"type": "users",
					"id":   c.ownerID,
				},
			},
		}
	}

	return payload
}

// parseAndDisplayList parses and displays list of application keys
func (c *ApplicationKeysCommand) parseAndDisplayList(data []byte, obs *observability.Observability) error {
	// Parse JSON response
	var apiResponse struct {
		Data []struct {
			ID         string `json:"id"`
			Type       string `json:"type"`
			Attributes struct {
				Name      string   `json:"name"`
				Hash      string   `json:"hash"`
				CreatedAt string   `json:"created_at"`
				Scopes    []string `json:"scopes"`
			} `json:"attributes"`
			Relationships struct {
				OwnedBy struct {
					Data struct {
						ID   string `json:"id"`
						Type string `json:"type"`
					} `json:"data"`
				} `json:"owned_by"`
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
	response := ApplicationKeysResponse{
		Status:    "success",
		TotalKeys: len(apiResponse.Data),
		Keys:      make([]ApplicationKey, 0),
	}

	// Parse application keys
	for _, item := range apiResponse.Data {
		key := ApplicationKey{
			ID:        item.ID,
			Name:      item.Attributes.Name,
			Hash:      item.Attributes.Hash,
			CreatedAt: item.Attributes.CreatedAt,
			Scopes:    item.Attributes.Scopes,
			Owner:     item.Relationships.OwnedBy.Data.ID,
		}

		response.Keys = append(response.Keys, key)
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

// parseAndDisplaySingle parses and displays a single application key
func (c *ApplicationKeysCommand) parseAndDisplaySingle(data []byte, obs *observability.Observability, showKey bool) error {
	// Parse JSON response
	var apiResponse struct {
		Data struct {
			ID         string `json:"id"`
			Type       string `json:"type"`
			Attributes struct {
				Name      string   `json:"name"`
				Hash      string   `json:"hash"`
				Key       string   `json:"key"`
				CreatedAt string   `json:"created_at"`
				Scopes    []string `json:"scopes"`
			} `json:"attributes"`
			Relationships struct {
				OwnedBy struct {
					Data struct {
						ID   string `json:"id"`
						Type string `json:"type"`
					} `json:"data"`
				} `json:"owned_by"`
			} `json:"relationships"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &apiResponse); err != nil {
		return fmt.Errorf("failed to parse API response: %w", err)
	}

	// Build application key object
	key := ApplicationKey{
		ID:        apiResponse.Data.ID,
		Name:      apiResponse.Data.Attributes.Name,
		Hash:      apiResponse.Data.Attributes.Hash,
		CreatedAt: apiResponse.Data.Attributes.CreatedAt,
		Scopes:    apiResponse.Data.Attributes.Scopes,
		Owner:     apiResponse.Data.Relationships.OwnedBy.Data.ID,
	}

	// Include actual key value only for create response
	if showKey && apiResponse.Data.Attributes.Key != "" {
		key.Key = apiResponse.Data.Attributes.Key
	}

	// Output result
	if c.jsonOut {
		output, err := json.MarshalIndent(key, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(output))
	} else {
		c.displayFormattedSingle(key, showKey)
	}

	return nil
}

// displayFormattedList displays formatted list output
func (c *ApplicationKeysCommand) displayFormattedList(response ApplicationKeysResponse) {
	fmt.Println("Application Keys Summary")
	fmt.Println("========================")
	fmt.Printf("Total application keys: %d\n", response.TotalKeys)

	if len(response.Keys) > 0 {
		fmt.Println("\nApplication Keys:")
		for i, key := range response.Keys {
			if i >= 20 {
				fmt.Printf("\n... and %d more (use --json for full list)\n", len(response.Keys)-20)
				break
			}

			fmt.Printf("\n[%s] %s\n", key.ID, key.Name)
			fmt.Printf("  Hash: %s\n", key.Hash)
			if key.Owner != "" {
				fmt.Printf("  Owner: %s\n", key.Owner)
			}
			if key.CreatedAt != "" {
				// Format timestamp
				if t, err := time.Parse(time.RFC3339, key.CreatedAt); err == nil {
					fmt.Printf("  Created: %s\n", t.Format("2006-01-02 15:04:05 MST"))
				} else {
					fmt.Printf("  Created: %s\n", key.CreatedAt)
				}
			}
			if len(key.Scopes) > 0 {
				fmt.Printf("  Scopes: %d configured\n", len(key.Scopes))
			}
		}
	} else {
		fmt.Println("\nNo application keys found.")
	}
}

// displayFormattedSingle displays formatted single application key output
func (c *ApplicationKeysCommand) displayFormattedSingle(key ApplicationKey, showKey bool) {
	fmt.Println("Application Key Details")
	fmt.Println("=======================")
	fmt.Printf("ID: %s\n", key.ID)
	fmt.Printf("Name: %s\n", key.Name)
	fmt.Printf("Hash: %s\n", key.Hash)

	// Show actual key value only on creation (one time only!)
	if showKey && key.Key != "" {
		fmt.Println()
		fmt.Println("⚠️  IMPORTANT: Save this key securely - it will not be shown again!")
		fmt.Printf("Key: %s\n", key.Key)
		fmt.Println()
	}

	if key.Owner != "" {
		fmt.Printf("Owner: %s\n", key.Owner)
	}

	if key.CreatedAt != "" {
		// Format timestamp
		if t, err := time.Parse(time.RFC3339, key.CreatedAt); err == nil {
			fmt.Printf("Created: %s\n", t.Format("2006-01-02 15:04:05 MST"))
		} else {
			fmt.Printf("Created: %s\n", key.CreatedAt)
		}
	}

	if len(key.Scopes) > 0 {
		fmt.Printf("\nScopes (%d):\n", len(key.Scopes))
		for _, scope := range key.Scopes {
			fmt.Printf("  - %s\n", scope)
		}
	} else {
		fmt.Println("\nScopes: None (full access)")
	}
}

// splitScopes splits a comma-separated list of scopes
func splitScopes(scopes string) []string {
	result := []string{}
	current := ""
	for _, ch := range scopes {
		if ch == ',' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else if ch != ' ' && ch != '\t' {
			current += string(ch)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

// Help displays help information
func (c *ApplicationKeysCommand) Help() {
	help := `dd application-keys - Manage Datadog Application Keys

DESCRIPTION:
  Manage Datadog application keys for API authentication. Application keys are
  used alongside API keys to authenticate API requests. They can be scoped to
  specific permissions and assigned to users or service accounts.

USAGE:
  dd application-keys --action <action> [options]

ACTIONS:
  list              List all application keys
  create            Create a new application key
  get               Get a specific application key
  update            Update an existing application key
  delete            Delete an application key

EXAMPLES:
  # List all application keys (current user)
  dd application-keys --action list

  # List all org keys (admin only)
  dd application-keys --action list --all

  # Create new application key
  dd application-keys --action create \
    --name "CI/CD Pipeline Key"

  # Create scoped application key
  dd application-keys --action create \
    --name "Read-Only Metrics Key" \
    --scopes "metrics_read,timeseries_query"

  # Create key for service account
  dd application-keys --action create \
    --name "Automation Key" \
    --owner-id "service-account-uuid"

  # Get specific application key
  dd application-keys --action get --key-id abc123-def456

  # Update application key name
  dd application-keys --action update \
    --key-id abc123-def456 \
    --name "Updated Key Name"

  # Delete application key
  dd application-keys --action delete --key-id abc123-def456

  # Get JSON output
  dd application-keys --action list --json

OPTIONS:
  --action          Action to perform (list, create, get, update, delete)
  --key-id          Application key ID (required for get/update/delete)
  --name            Application key name (required for create)
  --scopes          Comma-separated list of scopes (optional, defaults to full access)
  --owner-id        Owner user ID (for service accounts)
  --all             List all org keys instead of just current user keys (admin only)
  --json            Output as JSON

COMMON SCOPES:
  - metrics_read            Read metrics data
  - timeseries_query        Query time series
  - logs_read_data          Read log data
  - apm_read                Read APM data
  - dashboards_read         Read dashboards
  - dashboards_write        Write dashboards
  - monitors_read           Read monitors
  - monitors_write          Write monitors

NOTES:
  - Application keys are required for API authentication (alongside API key)
  - The actual key value is shown ONLY during creation - save it securely
  - Scoped keys provide principle of least privilege access
  - Keys can be owned by users or service accounts
  - Use service account keys for CI/CD and automation
  - Rotate keys regularly for security
  - Delete unused keys to reduce attack surface

For more information: https://docs.datadoghq.com/api/latest/key-management/
`
	fmt.Println(help)
}
