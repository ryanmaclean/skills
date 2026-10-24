package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"time"

	"github.com/datadog/skill/internal/client"
	"github.com/datadog/skill/internal/observability"
)

// APIKeysCommand manages Datadog API keys for primary authentication
type APIKeysCommand struct {
	flags   *flag.FlagSet
	action  string
	keyID   string
	name    string
	jsonOut bool
}

// APIKey represents a parsed API key from Datadog API
type APIKey struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Key       string `json:"key,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	CreatedBy string `json:"created_by,omitempty"`
}

// APIKeysResponse represents the formatted API keys response
type APIKeysResponse struct {
	Status    string   `json:"status"`
	TotalKeys int      `json:"total_keys"`
	Keys      []APIKey `json:"keys,omitempty"`
}

// NewAPIKeysCommand creates a new API keys command
func NewAPIKeysCommand() *APIKeysCommand {
	cmd := &APIKeysCommand{
		flags: flag.NewFlagSet("api-keys", flag.ExitOnError),
	}

	cmd.flags.StringVar(&cmd.action, "action", "list", "Action: list, create, get, update, delete")
	cmd.flags.StringVar(&cmd.keyID, "key-id", "", "API key ID (for get/update/delete)")
	cmd.flags.StringVar(&cmd.name, "name", "", "API key name (required for create/update)")
	cmd.flags.BoolVar(&cmd.jsonOut, "json", false, "Output as JSON")

	return cmd
}

// Name returns the command name
func (c *APIKeysCommand) Name() string {
	return "api-keys"
}

// Description returns the command description
func (c *APIKeysCommand) Description() string {
	return "Manage Datadog API keys for primary authentication"
}

// Run executes the API keys command
func (c *APIKeysCommand) Run(args []string) error {
	// Initialize observability
	obs, err := observability.Init("manage-api-keys", "production")
	if err != nil {
		return fmt.Errorf("failed to init observability: %w", err)
	}
	defer obs.Shutdown(0)

	// Parse flags
	if err := c.flags.Parse(args); err != nil {
		return err
	}

	// Start tracing
	span := obs.StartSpan("api_keys.manage")
	defer obs.FinishSpan(span)

	obs.LogInfo(fmt.Sprintf("Managing API keys with action: %s", c.action))

	// Initialize Datadog client
	ddClient, err := client.NewClient()
	if err != nil {
		return fmt.Errorf("failed to initialize Datadog client: %w", err)
	}

	// Execute action
	switch c.action {
	case "list", "list-keys":
		return c.listAPIKeys(ddClient, obs)
	case "create", "create-key":
		return c.createAPIKey(ddClient, obs)
	case "get", "get-key":
		return c.getAPIKey(ddClient, obs)
	case "update", "update-key":
		return c.updateAPIKey(ddClient, obs)
	case "delete", "delete-key":
		return c.deleteAPIKey(ddClient, obs)
	default:
		return fmt.Errorf("unknown action: %s (use: list, create, get, update, delete)", c.action)
	}
}

// listAPIKeys lists all API keys
func (c *APIKeysCommand) listAPIKeys(ddClient *client.Client, obs *observability.Observability) error {
	obs.LogInfo("Listing API keys")

	// Query API keys
	data, err := ddClient.ListAPIKeys()
	if err != nil {
		return fmt.Errorf("failed to list API keys: %w", err)
	}

	// Parse and display results
	return c.parseAndDisplayList(data, obs)
}

// createAPIKey creates a new API key
func (c *APIKeysCommand) createAPIKey(ddClient *client.Client, obs *observability.Observability) error {
	obs.LogInfo("Creating API key")

	// Validate required fields
	if c.name == "" {
		return fmt.Errorf("--name is required for creating API key")
	}

	// Build API key payload
	payload := c.buildAPIKeyPayload()

	// Create API key
	data, err := ddClient.CreateAPIKey(payload)
	if err != nil {
		return fmt.Errorf("failed to create API key: %w", err)
	}

	// Parse and display result (includes actual key value - shown only once!)
	return c.parseAndDisplaySingle(data, obs, true)
}

// getAPIKey gets a specific API key
func (c *APIKeysCommand) getAPIKey(ddClient *client.Client, obs *observability.Observability) error {
	if c.keyID == "" {
		return fmt.Errorf("--key-id is required for get action")
	}

	obs.LogInfo(fmt.Sprintf("Getting API key: %s", c.keyID))

	// Get API key
	data, err := ddClient.GetAPIKey(c.keyID)
	if err != nil {
		return fmt.Errorf("failed to get API key: %w", err)
	}

	// Parse and display result (key value not included in get response)
	return c.parseAndDisplaySingle(data, obs, false)
}

// updateAPIKey updates an existing API key
func (c *APIKeysCommand) updateAPIKey(ddClient *client.Client, obs *observability.Observability) error {
	if c.keyID == "" {
		return fmt.Errorf("--key-id is required for update action")
	}

	if c.name == "" {
		return fmt.Errorf("--name is required for update action")
	}

	obs.LogInfo(fmt.Sprintf("Updating API key: %s", c.keyID))

	// Build API key payload
	payload := c.buildAPIKeyPayload()

	// Update API key
	data, err := ddClient.UpdateAPIKey(c.keyID, payload)
	if err != nil {
		return fmt.Errorf("failed to update API key: %w", err)
	}

	// Parse and display result
	return c.parseAndDisplaySingle(data, obs, false)
}

// deleteAPIKey deletes an API key
func (c *APIKeysCommand) deleteAPIKey(ddClient *client.Client, obs *observability.Observability) error {
	if c.keyID == "" {
		return fmt.Errorf("--key-id is required for delete action")
	}

	obs.LogInfo(fmt.Sprintf("Deleting API key: %s", c.keyID))

	// Delete API key
	err := ddClient.DeleteAPIKey(c.keyID)
	if err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}

	fmt.Printf("✓ API key %s deleted successfully\n", c.keyID)
	fmt.Println("⚠️  WARNING: Any services using this key will immediately lose access!")
	return nil
}

// buildAPIKeyPayload builds the API payload for create/update
func (c *APIKeysCommand) buildAPIKeyPayload() map[string]interface{} {
	attributes := make(map[string]interface{})

	if c.name != "" {
		attributes["name"] = c.name
	}

	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type":       "api_keys",
			"attributes": attributes,
		},
	}

	return payload
}

// parseAndDisplayList parses and displays list of API keys
func (c *APIKeysCommand) parseAndDisplayList(data []byte, obs *observability.Observability) error {
	// Parse JSON response
	var apiResponse struct {
		Data []struct {
			ID         string `json:"id"`
			Type       string `json:"type"`
			Attributes struct {
				Name      string `json:"name"`
				CreatedAt string `json:"created_at"`
				Last4     string `json:"last4"`
			} `json:"attributes"`
			Relationships struct {
				CreatedBy struct {
					Data struct {
						ID   string `json:"id"`
						Type string `json:"type"`
					} `json:"data"`
				} `json:"created_by"`
			} `json:"relationships"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &apiResponse); err != nil {
		return fmt.Errorf("failed to parse API response: %w", err)
	}

	// Build response
	response := APIKeysResponse{
		Status:    "success",
		TotalKeys: len(apiResponse.Data),
		Keys:      make([]APIKey, 0),
	}

	// Parse API keys
	for _, item := range apiResponse.Data {
		key := APIKey{
			ID:        item.ID,
			Name:      item.Attributes.Name,
			CreatedAt: item.Attributes.CreatedAt,
			CreatedBy: item.Relationships.CreatedBy.Data.ID,
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

// parseAndDisplaySingle parses and displays a single API key
func (c *APIKeysCommand) parseAndDisplaySingle(data []byte, obs *observability.Observability, showKey bool) error {
	// Parse JSON response
	var apiResponse struct {
		Data struct {
			ID         string `json:"id"`
			Type       string `json:"type"`
			Attributes struct {
				Name      string `json:"name"`
				Key       string `json:"key"`
				CreatedAt string `json:"created_at"`
				Last4     string `json:"last4"`
			} `json:"attributes"`
			Relationships struct {
				CreatedBy struct {
					Data struct {
						ID   string `json:"id"`
						Type string `json:"type"`
					} `json:"data"`
				} `json:"created_by"`
			} `json:"relationships"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &apiResponse); err != nil {
		return fmt.Errorf("failed to parse API response: %w", err)
	}

	// Build API key object
	key := APIKey{
		ID:        apiResponse.Data.ID,
		Name:      apiResponse.Data.Attributes.Name,
		CreatedAt: apiResponse.Data.Attributes.CreatedAt,
		CreatedBy: apiResponse.Data.Relationships.CreatedBy.Data.ID,
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
func (c *APIKeysCommand) displayFormattedList(response APIKeysResponse) {
	fmt.Println("API Keys Summary")
	fmt.Println("================")
	fmt.Printf("Total API keys: %d\n", response.TotalKeys)

	if len(response.Keys) > 0 {
		fmt.Println("\nAPI Keys:")
		for i, key := range response.Keys {
			if i >= 20 {
				fmt.Printf("\n... and %d more (use --json for full list)\n", len(response.Keys)-20)
				break
			}

			fmt.Printf("\n[%s] %s\n", key.ID, key.Name)
			if key.CreatedBy != "" {
				fmt.Printf("  Created by: %s\n", key.CreatedBy)
			}
			if key.CreatedAt != "" {
				// Format timestamp
				if t, err := time.Parse(time.RFC3339, key.CreatedAt); err == nil {
					fmt.Printf("  Created: %s\n", t.Format("2006-01-02 15:04:05 MST"))
				} else {
					fmt.Printf("  Created: %s\n", key.CreatedAt)
				}
			}
		}
	} else {
		fmt.Println("\nNo API keys found.")
	}

	fmt.Println("\n⚠️  Security Note: API keys provide full account access. Rotate regularly and delete unused keys.")
}

// displayFormattedSingle displays formatted single API key output
func (c *APIKeysCommand) displayFormattedSingle(key APIKey, showKey bool) {
	fmt.Println("API Key Details")
	fmt.Println("===============")
	fmt.Printf("ID: %s\n", key.ID)
	fmt.Printf("Name: %s\n", key.Name)

	// Show actual key value only on creation (one time only!)
	if showKey && key.Key != "" {
		fmt.Println()
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("⚠️  CRITICAL: Save this API key securely - it will NEVER be shown again!")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Printf("\nAPI Key: %s\n", key.Key)
		fmt.Println()
		fmt.Println("This key provides FULL access to your Datadog account.")
		fmt.Println("Store it in a secure location (password manager, secrets vault).")
		fmt.Printf("Set as environment variable: export DD_API_KEY=\"%s\"\n", key.Key)
		fmt.Println()
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()
	}

	if key.CreatedBy != "" {
		fmt.Printf("Created by: %s\n", key.CreatedBy)
	}

	if key.CreatedAt != "" {
		// Format timestamp
		if t, err := time.Parse(time.RFC3339, key.CreatedAt); err == nil {
			fmt.Printf("Created: %s\n", t.Format("2006-01-02 15:04:05 MST"))
		} else {
			fmt.Printf("Created: %s\n", key.CreatedAt)
		}
	}

	if !showKey {
		fmt.Println("\n⚠️  Security Note: API key value cannot be retrieved after creation.")
		fmt.Println("If lost, create a new key and delete this one.")
	}
}

// Help displays help information
func (c *APIKeysCommand) Help() {
	help := `dd api-keys - Manage Datadog API Keys

DESCRIPTION:
  Manage Datadog API keys for primary authentication. API keys are required for
  all Datadog API requests and provide full access to your account. They are
  different from application keys and should be treated as highly sensitive.

USAGE:
  dd api-keys --action <action> [options]

ACTIONS:
  list              List all API keys
  create            Create a new API key
  get               Get a specific API key
  update            Update an API key name
  delete            Delete an API key

EXAMPLES:
  # List all API keys
  dd api-keys --action list

  # Create new API key
  dd api-keys --action create --name "Production API Key"

  # Create key for CI/CD
  dd api-keys --action create --name "CI Pipeline - GitHub Actions"

  # Get specific API key
  dd api-keys --action get --key-id abc123-def456

  # Update API key name
  dd api-keys --action update \
    --key-id abc123-def456 \
    --name "Updated Key Name"

  # Delete API key
  dd api-keys --action delete --key-id abc123-def456

  # Get JSON output
  dd api-keys --action list --json

OPTIONS:
  --action          Action to perform (list, create, get, update, delete)
  --key-id          API key ID (required for get/update/delete)
  --name            API key name (required for create/update)
  --json            Output as JSON

API KEY USAGE:
  # Set as environment variable
  export DD_API_KEY="your-api-key-here"

  # Use with dd CLI
  dd metrics --action query --metric "system.cpu.idle"

  # Use with Datadog Agent
  # In datadog.yaml:
  api_key: your-api-key-here

SECURITY NOTES:
  - API keys provide FULL access to your Datadog account
  - The actual key value is shown ONLY during creation
  - Store keys securely in password managers or secrets vaults
  - Never commit API keys to source control
  - Rotate keys regularly (every 90 days recommended)
  - Delete unused keys immediately
  - Use separate keys for different environments (dev, staging, prod)
  - Monitor key usage through audit logs
  - If a key is compromised, delete it immediately and create a new one

KEY ROTATION WORKFLOW:
  1. Create new API key with descriptive name
  2. Update services to use new key
  3. Verify services are working with new key
  4. Delete old API key
  5. Monitor for any services still using old key (they will fail)

DIFFERENCE FROM APPLICATION KEYS:
  - API keys: Primary authentication, required for all API requests
  - Application keys: Secondary authentication, used alongside API keys
  - API keys provide full account access
  - Application keys can be scoped to specific permissions

For more information: https://docs.datadoghq.com/api/latest/authentication/
`
	fmt.Println(help)
}
