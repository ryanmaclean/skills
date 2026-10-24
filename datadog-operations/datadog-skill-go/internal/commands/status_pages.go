package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/datadog/skill/internal/client"
)

// StatusPagesCommand manages Datadog Status Pages
type StatusPagesCommand struct {
	flags       *flag.FlagSet
	action      string
	pageID      string
	componentID string
	degradID    string
	name        string
	description string
	subdomain   string
	url         string
	componentName string
	status      string
	severity    string
	message     string
	notifySubscribers bool
	jsonOut     bool
}

// NewStatusPagesCommand creates a new status pages command
func NewStatusPagesCommand() Command {
	cmd := &StatusPagesCommand{
		flags: flag.NewFlagSet("status-pages", flag.ExitOnError),
	}

	cmd.flags.StringVar(&cmd.action, "action", "list-pages", "Action to perform")
	cmd.flags.StringVar(&cmd.pageID, "page-id", "", "Status page ID")
	cmd.flags.StringVar(&cmd.componentID, "component-id", "", "Component ID")
	cmd.flags.StringVar(&cmd.degradID, "degradation-id", "", "Degradation/incident ID")
	cmd.flags.StringVar(&cmd.name, "name", "", "Name of page/component")
	cmd.flags.StringVar(&cmd.description, "description", "", "Description")
	cmd.flags.StringVar(&cmd.subdomain, "subdomain", "", "Status page subdomain")
	cmd.flags.StringVar(&cmd.url, "url", "", "Custom domain URL")
	cmd.flags.StringVar(&cmd.componentName, "component-name", "", "Component name for degradation")
	cmd.flags.StringVar(&cmd.status, "status", "", "Status (investigating, identified, monitoring, resolved)")
	cmd.flags.StringVar(&cmd.severity, "severity", "", "Severity (critical, major, minor, maintenance)")
	cmd.flags.StringVar(&cmd.message, "message", "", "Status message or update")
	cmd.flags.BoolVar(&cmd.notifySubscribers, "notify", false, "Notify subscribers of changes")
	cmd.flags.BoolVar(&cmd.jsonOut, "json", false, "Output in JSON format")

	return cmd
}

// Name returns the command name
func (c *StatusPagesCommand) Name() string {
	return "status-pages"
}

// Description returns a short description
func (c *StatusPagesCommand) Description() string {
	return "Manage Datadog Status Pages for customer communication"
}

// Run executes the status pages command
func (c *StatusPagesCommand) Run(args []string) error {
	if err := c.flags.Parse(args); err != nil {
		return err
	}

	ddClient, err := client.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Datadog client: %w", err)
	}

	// Route to appropriate action handler
	switch c.action {
	// Page management
	case "list-pages", "list":
		return c.listPages(ddClient)
	case "create-page", "create":
		return c.createPage(ddClient)
	case "get-page", "get":
		return c.getPage(ddClient)
	case "update-page", "update":
		return c.updatePage(ddClient)
	case "delete-page", "delete":
		return c.deletePage(ddClient)

	// Component management
	case "list-components", "components":
		return c.listComponents(ddClient)
	case "create-component", "add-component":
		return c.createComponent(ddClient)
	case "get-component":
		return c.getComponent(ddClient)
	case "update-component":
		return c.updateComponent(ddClient)
	case "delete-component", "remove-component":
		return c.deleteComponent(ddClient)

	// Degradation (incident) management
	case "list-degradations", "degradations", "incidents":
		return c.listDegradations(ddClient)
	case "create-degradation", "create-incident":
		return c.createDegradation(ddClient)
	case "get-degradation", "get-incident":
		return c.getDegradation(ddClient)
	case "update-degradation", "update-incident":
		return c.updateDegradation(ddClient)
	case "resolve-degradation", "resolve-incident", "resolve":
		return c.resolveDegradation(ddClient)

	default:
		return fmt.Errorf("unknown action: %s (use --help to see available actions)", c.action)
	}
}

// Page Management Actions

func (c *StatusPagesCommand) listPages(ddClient *client.Client) error {
	resp, err := ddClient.ListStatusPages()
	if err != nil {
		return fmt.Errorf("failed to list status pages: %w", err)
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
				Subdomain   string `json:"subdomain"`
				CustomURL   string `json:"custom_url"`
				Description string `json:"description"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Data) == 0 {
		fmt.Println("No status pages found.")
		return nil
	}

	fmt.Printf("Status Pages (%d)\n", len(result.Data))
	fmt.Println(strings.Repeat("=", 80))

	for _, page := range result.Data {
		fmt.Printf("\nID: %s\n", page.ID)
		fmt.Printf("Name: %s\n", page.Attributes.Name)
		if page.Attributes.Subdomain != "" {
			fmt.Printf("Subdomain: %s\n", page.Attributes.Subdomain)
		}
		if page.Attributes.CustomURL != "" {
			fmt.Printf("Custom URL: %s\n", page.Attributes.CustomURL)
		}
		if page.Attributes.Description != "" {
			fmt.Printf("Description: %s\n", page.Attributes.Description)
		}
	}

	return nil
}

func (c *StatusPagesCommand) createPage(ddClient *client.Client) error {
	if c.name == "" {
		return fmt.Errorf("--name is required")
	}

	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "statuspages",
			"attributes": map[string]interface{}{
				"name": c.name,
			},
		},
	}

	// Add optional fields
	attrs := payload["data"].(map[string]interface{})["attributes"].(map[string]interface{})
	if c.description != "" {
		attrs["description"] = c.description
	}
	if c.subdomain != "" {
		attrs["subdomain"] = c.subdomain
	}
	if c.url != "" {
		attrs["custom_url"] = c.url
	}

	resp, err := ddClient.CreateStatusPage(payload)
	if err != nil {
		return fmt.Errorf("failed to create status page: %w", err)
	}

	if c.jsonOut {
		fmt.Println(string(resp))
		return nil
	}

	var result struct {
		Data struct {
			ID         string `json:"id"`
			Attributes struct {
				Name      string `json:"name"`
				Subdomain string `json:"subdomain"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Println("✓ Status page created successfully")
	fmt.Printf("Page ID: %s\n", result.Data.ID)
	fmt.Printf("Name: %s\n", result.Data.Attributes.Name)
	if result.Data.Attributes.Subdomain != "" {
		fmt.Printf("Subdomain: %s\n", result.Data.Attributes.Subdomain)
	}

	return nil
}

func (c *StatusPagesCommand) getPage(ddClient *client.Client) error {
	if c.pageID == "" {
		return fmt.Errorf("--page-id is required")
	}

	resp, err := ddClient.GetStatusPage(c.pageID)
	if err != nil {
		return fmt.Errorf("failed to get status page: %w", err)
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
				Subdomain   string `json:"subdomain"`
				CustomURL   string `json:"custom_url"`
				Description string `json:"description"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Printf("Status Page: %s\n", result.Data.Attributes.Name)
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("ID: %s\n", result.Data.ID)
	if result.Data.Attributes.Subdomain != "" {
		fmt.Printf("Subdomain: %s\n", result.Data.Attributes.Subdomain)
	}
	if result.Data.Attributes.CustomURL != "" {
		fmt.Printf("Custom URL: %s\n", result.Data.Attributes.CustomURL)
	}
	if result.Data.Attributes.Description != "" {
		fmt.Printf("Description: %s\n", result.Data.Attributes.Description)
	}

	return nil
}

func (c *StatusPagesCommand) updatePage(ddClient *client.Client) error {
	if c.pageID == "" {
		return fmt.Errorf("--page-id is required")
	}

	attrs := make(map[string]interface{})
	if c.name != "" {
		attrs["name"] = c.name
	}
	if c.description != "" {
		attrs["description"] = c.description
	}
	if c.subdomain != "" {
		attrs["subdomain"] = c.subdomain
	}
	if c.url != "" {
		attrs["custom_url"] = c.url
	}

	if len(attrs) == 0 {
		return fmt.Errorf("no fields to update (provide --name, --description, --subdomain, or --url)")
	}

	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type":       "statuspages",
			"id":         c.pageID,
			"attributes": attrs,
		},
	}

	resp, err := ddClient.UpdateStatusPage(c.pageID, payload)
	if err != nil {
		return fmt.Errorf("failed to update status page: %w", err)
	}

	if c.jsonOut {
		fmt.Println(string(resp))
		return nil
	}

	fmt.Println("✓ Status page updated successfully")
	return nil
}

func (c *StatusPagesCommand) deletePage(ddClient *client.Client) error {
	if c.pageID == "" {
		return fmt.Errorf("--page-id is required")
	}

	if err := ddClient.DeleteStatusPage(c.pageID); err != nil {
		return fmt.Errorf("failed to delete status page: %w", err)
	}

	fmt.Printf("✓ Status page %s deleted successfully\n", c.pageID)
	return nil
}

// Component Management Actions

func (c *StatusPagesCommand) listComponents(ddClient *client.Client) error {
	if c.pageID == "" {
		return fmt.Errorf("--page-id is required")
	}

	resp, err := ddClient.ListStatusPageComponents(c.pageID)
	if err != nil {
		return fmt.Errorf("failed to list components: %w", err)
	}

	if c.jsonOut {
		fmt.Println(string(resp))
		return nil
	}

	var result struct {
		Data []struct {
			ID         string `json:"id"`
			Attributes struct {
				Name        string `json:"name"`
				Description string `json:"description"`
				Status      string `json:"status"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Data) == 0 {
		fmt.Println("No components found for this status page.")
		return nil
	}

	fmt.Printf("Components (%d)\n", len(result.Data))
	fmt.Println(strings.Repeat("=", 80))

	for _, comp := range result.Data {
		fmt.Printf("\nID: %s\n", comp.ID)
		fmt.Printf("Name: %s\n", comp.Attributes.Name)
		if comp.Attributes.Status != "" {
			fmt.Printf("Status: %s\n", comp.Attributes.Status)
		}
		if comp.Attributes.Description != "" {
			fmt.Printf("Description: %s\n", comp.Attributes.Description)
		}
	}

	return nil
}

func (c *StatusPagesCommand) createComponent(ddClient *client.Client) error {
	if c.pageID == "" {
		return fmt.Errorf("--page-id is required")
	}
	if c.name == "" {
		return fmt.Errorf("--name is required")
	}

	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "statuspage_components",
			"attributes": map[string]interface{}{
				"name": c.name,
			},
		},
	}

	// Add optional fields
	attrs := payload["data"].(map[string]interface{})["attributes"].(map[string]interface{})
	if c.description != "" {
		attrs["description"] = c.description
	}

	resp, err := ddClient.CreateStatusPageComponent(c.pageID, payload)
	if err != nil {
		return fmt.Errorf("failed to create component: %w", err)
	}

	if c.jsonOut {
		fmt.Println(string(resp))
		return nil
	}

	var result struct {
		Data struct {
			ID         string `json:"id"`
			Attributes struct {
				Name string `json:"name"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Println("✓ Component created successfully")
	fmt.Printf("Component ID: %s\n", result.Data.ID)
	fmt.Printf("Name: %s\n", result.Data.Attributes.Name)

	return nil
}

func (c *StatusPagesCommand) getComponent(ddClient *client.Client) error {
	if c.pageID == "" {
		return fmt.Errorf("--page-id is required")
	}
	if c.componentID == "" {
		return fmt.Errorf("--component-id is required")
	}

	resp, err := ddClient.GetStatusPageComponent(c.pageID, c.componentID)
	if err != nil {
		return fmt.Errorf("failed to get component: %w", err)
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
				Description string `json:"description"`
				Status      string `json:"status"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Printf("Component: %s\n", result.Data.Attributes.Name)
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("ID: %s\n", result.Data.ID)
	if result.Data.Attributes.Status != "" {
		fmt.Printf("Status: %s\n", result.Data.Attributes.Status)
	}
	if result.Data.Attributes.Description != "" {
		fmt.Printf("Description: %s\n", result.Data.Attributes.Description)
	}

	return nil
}

func (c *StatusPagesCommand) updateComponent(ddClient *client.Client) error {
	if c.pageID == "" {
		return fmt.Errorf("--page-id is required")
	}
	if c.componentID == "" {
		return fmt.Errorf("--component-id is required")
	}

	attrs := make(map[string]interface{})
	if c.name != "" {
		attrs["name"] = c.name
	}
	if c.description != "" {
		attrs["description"] = c.description
	}
	if c.status != "" {
		attrs["status"] = c.status
	}

	if len(attrs) == 0 {
		return fmt.Errorf("no fields to update (provide --name, --description, or --status)")
	}

	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type":       "statuspage_components",
			"id":         c.componentID,
			"attributes": attrs,
		},
	}

	resp, err := ddClient.UpdateStatusPageComponent(c.pageID, c.componentID, payload)
	if err != nil {
		return fmt.Errorf("failed to update component: %w", err)
	}

	if c.jsonOut {
		fmt.Println(string(resp))
		return nil
	}

	fmt.Println("✓ Component updated successfully")
	return nil
}

func (c *StatusPagesCommand) deleteComponent(ddClient *client.Client) error {
	if c.pageID == "" {
		return fmt.Errorf("--page-id is required")
	}
	if c.componentID == "" {
		return fmt.Errorf("--component-id is required")
	}

	if err := ddClient.DeleteStatusPageComponent(c.pageID, c.componentID); err != nil {
		return fmt.Errorf("failed to delete component: %w", err)
	}

	fmt.Printf("✓ Component %s deleted successfully\n", c.componentID)
	return nil
}

// Degradation (Incident) Management Actions

func (c *StatusPagesCommand) listDegradations(ddClient *client.Client) error {
	resp, err := ddClient.ListStatusPageDegradations()
	if err != nil {
		return fmt.Errorf("failed to list degradations: %w", err)
	}

	if c.jsonOut {
		fmt.Println(string(resp))
		return nil
	}

	var result struct {
		Data []struct {
			ID         string `json:"id"`
			Attributes struct {
				Title    string `json:"title"`
				Status   string `json:"status"`
				Severity string `json:"severity"`
				Message  string `json:"message"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Data) == 0 {
		fmt.Println("No degradations/incidents found.")
		return nil
	}

	fmt.Printf("Degradations/Incidents (%d)\n", len(result.Data))
	fmt.Println(strings.Repeat("=", 80))

	for _, deg := range result.Data {
		fmt.Printf("\nID: %s\n", deg.ID)
		fmt.Printf("Title: %s\n", deg.Attributes.Title)
		if deg.Attributes.Status != "" {
			fmt.Printf("Status: %s\n", deg.Attributes.Status)
		}
		if deg.Attributes.Severity != "" {
			fmt.Printf("Severity: %s\n", deg.Attributes.Severity)
		}
		if deg.Attributes.Message != "" {
			fmt.Printf("Message: %s\n", deg.Attributes.Message)
		}
	}

	return nil
}

func (c *StatusPagesCommand) createDegradation(ddClient *client.Client) error {
	if c.pageID == "" {
		return fmt.Errorf("--page-id is required")
	}
	if c.name == "" {
		return fmt.Errorf("--name is required (incident title)")
	}

	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "statuspage_degradations",
			"attributes": map[string]interface{}{
				"title": c.name,
			},
		},
	}

	// Add optional fields
	attrs := payload["data"].(map[string]interface{})["attributes"].(map[string]interface{})
	if c.status != "" {
		attrs["status"] = c.status
	} else {
		attrs["status"] = "investigating" // Default status
	}
	if c.severity != "" {
		attrs["severity"] = c.severity
	}
	if c.message != "" {
		attrs["message"] = c.message
	}
	if c.notifySubscribers {
		attrs["notify_subscribers"] = true
	}

	resp, err := ddClient.CreateStatusPageDegradation(c.pageID, payload)
	if err != nil {
		return fmt.Errorf("failed to create degradation: %w", err)
	}

	if c.jsonOut {
		fmt.Println(string(resp))
		return nil
	}

	var result struct {
		Data struct {
			ID         string `json:"id"`
			Attributes struct {
				Title  string `json:"title"`
				Status string `json:"status"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Println("✓ Degradation/incident created successfully")
	fmt.Printf("Degradation ID: %s\n", result.Data.ID)
	fmt.Printf("Title: %s\n", result.Data.Attributes.Title)
	fmt.Printf("Status: %s\n", result.Data.Attributes.Status)

	return nil
}

func (c *StatusPagesCommand) getDegradation(ddClient *client.Client) error {
	if c.pageID == "" {
		return fmt.Errorf("--page-id is required")
	}
	if c.degradID == "" {
		return fmt.Errorf("--degradation-id is required")
	}

	resp, err := ddClient.GetStatusPageDegradation(c.pageID, c.degradID)
	if err != nil {
		return fmt.Errorf("failed to get degradation: %w", err)
	}

	if c.jsonOut {
		fmt.Println(string(resp))
		return nil
	}

	var result struct {
		Data struct {
			ID         string `json:"id"`
			Attributes struct {
				Title    string `json:"title"`
				Status   string `json:"status"`
				Severity string `json:"severity"`
				Message  string `json:"message"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Printf("Degradation: %s\n", result.Data.Attributes.Title)
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("ID: %s\n", result.Data.ID)
	if result.Data.Attributes.Status != "" {
		fmt.Printf("Status: %s\n", result.Data.Attributes.Status)
	}
	if result.Data.Attributes.Severity != "" {
		fmt.Printf("Severity: %s\n", result.Data.Attributes.Severity)
	}
	if result.Data.Attributes.Message != "" {
		fmt.Printf("Message: %s\n", result.Data.Attributes.Message)
	}

	return nil
}

func (c *StatusPagesCommand) updateDegradation(ddClient *client.Client) error {
	if c.pageID == "" {
		return fmt.Errorf("--page-id is required")
	}
	if c.degradID == "" {
		return fmt.Errorf("--degradation-id is required")
	}

	attrs := make(map[string]interface{})
	if c.name != "" {
		attrs["title"] = c.name
	}
	if c.status != "" {
		attrs["status"] = c.status
	}
	if c.severity != "" {
		attrs["severity"] = c.severity
	}
	if c.message != "" {
		attrs["message"] = c.message
	}
	if c.notifySubscribers {
		attrs["notify_subscribers"] = true
	}

	if len(attrs) == 0 {
		return fmt.Errorf("no fields to update (provide --name, --status, --severity, or --message)")
	}

	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type":       "statuspage_degradations",
			"id":         c.degradID,
			"attributes": attrs,
		},
	}

	resp, err := ddClient.UpdateStatusPageDegradation(c.pageID, c.degradID, payload)
	if err != nil {
		return fmt.Errorf("failed to update degradation: %w", err)
	}

	if c.jsonOut {
		fmt.Println(string(resp))
		return nil
	}

	fmt.Println("✓ Degradation/incident updated successfully")
	return nil
}

func (c *StatusPagesCommand) resolveDegradation(ddClient *client.Client) error {
	if c.pageID == "" {
		return fmt.Errorf("--page-id is required")
	}
	if c.degradID == "" {
		return fmt.Errorf("--degradation-id is required")
	}

	if err := ddClient.DeleteStatusPageDegradation(c.pageID, c.degradID); err != nil {
		return fmt.Errorf("failed to resolve degradation: %w", err)
	}

	fmt.Printf("✓ Degradation %s resolved successfully\n", c.degradID)
	return nil
}

// Help displays help information
func (c *StatusPagesCommand) Help() {
	help := `dd status-pages - Manage Datadog Status Pages

DESCRIPTION:
  Manage Status Pages for customer and stakeholder communication about
  service availability and incidents. Create pages, add components, and
  track service degradations.

USAGE:
  dd status-pages --action <action> [options]

PAGE MANAGEMENT ACTIONS:
  list-pages, list       List all status pages
  create-page, create    Create new status page
  get-page, get          Get status page details
  update-page, update    Update status page
  delete-page, delete    Delete status page

COMPONENT MANAGEMENT ACTIONS:
  list-components        List components on a page
  create-component       Add component to page
  get-component          Get component details
  update-component       Update component
  delete-component       Remove component from page

DEGRADATION (INCIDENT) ACTIONS:
  list-degradations      List all degradations/incidents
  create-degradation     Create new degradation/incident
  get-degradation        Get degradation details
  update-degradation     Update degradation status
  resolve-degradation    Resolve/delete degradation

COMMON OPTIONS:
  --page-id string          Status page ID
  --component-id string     Component ID
  --degradation-id string   Degradation/incident ID
  --name string             Name or title
  --description string      Description
  --subdomain string        Status page subdomain
  --url string              Custom domain URL
  --status string           Status (investigating, identified, monitoring, resolved)
  --severity string         Severity (critical, major, minor, maintenance)
  --message string          Status message or update
  --notify                  Notify subscribers of changes
  --json                    Output in JSON format

EXAMPLES:
  # List all status pages
  dd status-pages --action list-pages

  # Create new status page
  dd status-pages --action create-page --name "API Status" --subdomain api-status

  # Add component to page
  dd status-pages --action create-component --page-id abc123 --name "Authentication API"

  # List components
  dd status-pages --action list-components --page-id abc123

  # Create incident (degradation)
  dd status-pages --action create-degradation \
    --page-id abc123 \
    --name "Authentication Service Degraded" \
    --severity major \
    --status investigating \
    --notify

  # Update incident status
  dd status-pages --action update-degradation \
    --page-id abc123 \
    --degradation-id xyz789 \
    --status identified \
    --message "Root cause identified, working on fix"

  # Resolve incident
  dd status-pages --action resolve-degradation \
    --page-id abc123 \
    --degradation-id xyz789

  # Get JSON output
  dd status-pages --action list-pages --json

DEGRADATION STATUSES:
  investigating  - Team is investigating the issue
  identified     - Root cause has been identified
  monitoring     - Fix deployed, monitoring for stability
  resolved       - Issue fully resolved

SEVERITY LEVELS:
  critical       - Major service outage
  major          - Significant degradation
  minor          - Minor issues affecting some users
  maintenance    - Planned maintenance window

USE CASES:
  1. Create public status pages for customer communication
  2. Track service components and their health
  3. Report and manage service degradations
  4. Communicate incident updates to subscribers
  5. Maintain transparency during outages

AUTHENTICATION:
  Requires DD_API_KEY and DD_APP_KEY environment variables.
  Requires status_pages_settings_read/write permissions.
`
	fmt.Println(strings.TrimSpace(help))
}
