package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/datadog/skill/internal/client"
	"github.com/datadog/skill/internal/observability"
)

// DashboardsCommand manages Datadog dashboards
type DashboardsCommand struct {
	flags     *flag.FlagSet
	operation string
	id        string
	title     string
	file      string
	json      bool
}

// DashboardSummary represents a summarized dashboard
type DashboardSummary struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	LayoutType  string   `json:"layout_type,omitempty"`
	Author      string   `json:"author,omitempty"`
	URL         string   `json:"url,omitempty"`
	IsReadOnly  bool     `json:"is_read_only"`
	Tags        []string `json:"tags,omitempty"`
	CreatedAt   string   `json:"created_at,omitempty"`
	ModifiedAt  string   `json:"modified_at,omitempty"`
}

// DashboardsResponse represents the formatted dashboards response
type DashboardsResponse struct {
	Status     string             `json:"status"`
	Operation  string             `json:"operation"`
	Total      int                `json:"total,omitempty"`
	Dashboards []DashboardSummary `json:"dashboards,omitempty"`
	Dashboard  interface{}        `json:"dashboard,omitempty"`
	Message    string             `json:"message,omitempty"`
}

// NewDashboardsCommand creates a new dashboards command
func NewDashboardsCommand() *DashboardsCommand {
	cmd := &DashboardsCommand{
		flags: flag.NewFlagSet("dashboards", flag.ExitOnError),
	}

	cmd.flags.StringVar(&cmd.operation, "operation", "list", "Operation: list, get, create, update, delete")
	cmd.flags.StringVar(&cmd.id, "id", "", "Dashboard ID (required for get, update, delete)")
	cmd.flags.StringVar(&cmd.title, "title", "", "Filter dashboards by title (for list)")
	cmd.flags.StringVar(&cmd.file, "file", "", "JSON file path for create/update operations (use '-' for stdin)")
	cmd.flags.BoolVar(&cmd.json, "json", false, "Output as JSON")

	return cmd
}

// Name returns the command name
func (c *DashboardsCommand) Name() string {
	return "dashboards"
}

// Description returns the command description
func (c *DashboardsCommand) Description() string {
	return "Manage Datadog dashboards (list, get, create, update, delete)"
}

// Run executes the dashboards command
func (c *DashboardsCommand) Run(args []string) error {
	// Initialize observability
	obs, err := observability.Init("dd-cli-dashboards", "production")
	if err != nil {
		return fmt.Errorf("failed to init observability: %w", err)
	}
	defer obs.Shutdown(0)

	// Parse flags
	if err := c.flags.Parse(args); err != nil {
		return err
	}

	// Start tracing
	span := obs.StartSpan("dashboards." + c.operation)
	defer obs.FinishSpan(span)

	obs.LogInfo(fmt.Sprintf("Starting dashboard operation: %s", c.operation))
	obs.GetTracer().SetTag(span, "operation", c.operation)

	// Validate operation
	validOps := map[string]bool{"list": true, "get": true, "create": true, "update": true, "delete": true}
	if !validOps[c.operation] {
		obs.LogError(fmt.Sprintf("Invalid operation: %s", c.operation))
		return fmt.Errorf("invalid operation: %s. Must be one of: list, get, create, update, delete", c.operation)
	}

	// Validate required parameters
	if (c.operation == "get" || c.operation == "update" || c.operation == "delete") && c.id == "" {
		obs.LogError("Dashboard ID is required for " + c.operation)
		return fmt.Errorf("--id is required for %s operation", c.operation)
	}

	if (c.operation == "create" || c.operation == "update") && c.file == "" {
		obs.LogError("File is required for " + c.operation)
		return fmt.Errorf("--file is required for %s operation", c.operation)
	}

	// Create Datadog client
	clientSpan := obs.StartSpan("dashboards.create_client")
	ddClient, err := client.NewClient()
	if err != nil {
		obs.LogError("Failed to create Datadog client: " + err.Error())
		obs.FinishSpan(clientSpan)
		return fmt.Errorf("failed to create Datadog client: %w", err)
	}
	obs.FinishSpan(clientSpan)

	// Execute operation
	var response *DashboardsResponse
	switch c.operation {
	case "list":
		response, err = c.listDashboards(obs, ddClient)
	case "get":
		response, err = c.getDashboard(obs, ddClient)
	case "create":
		response, err = c.createDashboard(obs, ddClient)
	case "update":
		response, err = c.updateDashboard(obs, ddClient)
	case "delete":
		response, err = c.deleteDashboard(obs, ddClient)
	}

	if err != nil {
		obs.LogError(fmt.Sprintf("Operation failed: %s", err.Error()))
		return err
	}

	// Record metrics
	obs.GetMetrics().Count("dashboards.operation", 1,
		"operation:"+c.operation,
		"status:"+response.Status,
	)

	// Output results
	if c.json {
		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	} else {
		c.printFormatted(response)
	}

	obs.LogInfo(fmt.Sprintf("Dashboard operation completed: %s", c.operation))
	return nil
}

// listDashboards lists all dashboards
func (c *DashboardsCommand) listDashboards(obs *observability.Observability, ddClient *client.Client) (*DashboardsResponse, error) {
	span := obs.StartSpan("dashboards.api_list")
	defer obs.FinishSpan(span)

	start := time.Now()
	rawData, err := ddClient.ListDashboards()
	apiDuration := time.Since(start).Milliseconds()

	if err != nil {
		obs.RecordAPICall("/api/v1/dashboard", "GET", 500, float64(apiDuration), err)
		obs.GetTracer().SetError(span, err)
		return nil, fmt.Errorf("failed to list dashboards: %w", err)
	}

	obs.RecordAPICall("/api/v1/dashboard", "GET", 200, float64(apiDuration), nil)

	// Parse response
	var apiResponse struct {
		Dashboards []map[string]interface{} `json:"dashboards"`
	}

	if err := json.Unmarshal(rawData, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to summaries
	summaries := make([]DashboardSummary, 0, len(apiResponse.Dashboards))
	for _, d := range apiResponse.Dashboards {
		summary := DashboardSummary{
			ID:          getDashboardString(d, "id", ""),
			Title:       getDashboardString(d, "title", ""),
			Description: getDashboardString(d, "description", ""),
			LayoutType:  getDashboardString(d, "layout_type", ""),
			Author:      getDashboardString(d, "author_handle", ""),
			URL:         getDashboardString(d, "url", ""),
			IsReadOnly:  getDashboardBool(d, "is_read_only", false),
			CreatedAt:   getDashboardString(d, "created_at", ""),
			ModifiedAt:  getDashboardString(d, "modified_at", ""),
		}

		// Extract tags
		if tags, ok := d["tags"].([]interface{}); ok {
			for _, tag := range tags {
				if tagStr, ok := tag.(string); ok {
					summary.Tags = append(summary.Tags, tagStr)
				}
			}
		}

		// Filter by title if specified
		if c.title == "" || strings.Contains(strings.ToLower(summary.Title), strings.ToLower(c.title)) {
			summaries = append(summaries, summary)
		}
	}

	obs.LogInfo(fmt.Sprintf("Found %d dashboards", len(summaries)))
	obs.GetMetrics().Gauge("dashboards.total", float64(len(summaries)))

	return &DashboardsResponse{
		Status:     "ok",
		Operation:  "list",
		Total:      len(summaries),
		Dashboards: summaries,
		Message:    fmt.Sprintf("Found %d dashboards", len(summaries)),
	}, nil
}

// getDashboard retrieves a specific dashboard
func (c *DashboardsCommand) getDashboard(obs *observability.Observability, ddClient *client.Client) (*DashboardsResponse, error) {
	span := obs.StartSpan("dashboards.api_get")
	obs.GetTracer().SetTag(span, "dashboard.id", c.id)
	defer obs.FinishSpan(span)

	start := time.Now()
	rawData, err := ddClient.GetDashboard(c.id)
	apiDuration := time.Since(start).Milliseconds()

	if err != nil {
		obs.RecordAPICall("/api/v1/dashboard/"+c.id, "GET", 500, float64(apiDuration), err)
		obs.GetTracer().SetError(span, err)
		return nil, fmt.Errorf("failed to get dashboard: %w", err)
	}

	obs.RecordAPICall("/api/v1/dashboard/"+c.id, "GET", 200, float64(apiDuration), nil)

	// Parse as generic interface for full details
	var dashboard map[string]interface{}
	if err := json.Unmarshal(rawData, &dashboard); err != nil {
		return nil, fmt.Errorf("failed to parse dashboard: %w", err)
	}

	obs.LogInfo(fmt.Sprintf("Retrieved dashboard: %s", c.id))

	return &DashboardsResponse{
		Status:    "ok",
		Operation: "get",
		Dashboard: dashboard,
		Message:   fmt.Sprintf("Retrieved dashboard: %s", getDashboardString(dashboard, "title", c.id)),
	}, nil
}

// createDashboard creates a new dashboard
func (c *DashboardsCommand) createDashboard(obs *observability.Observability, ddClient *client.Client) (*DashboardsResponse, error) {
	// Read payload from file or stdin
	parseSpan := obs.StartSpan("dashboards.parse_payload")
	payload, err := c.readPayload()
	obs.FinishSpan(parseSpan)

	if err != nil {
		return nil, fmt.Errorf("failed to read payload: %w", err)
	}

	// Validate payload
	if payload["title"] == nil {
		return nil, fmt.Errorf("dashboard title is required in payload")
	}
	if payload["layout_type"] == nil {
		return nil, fmt.Errorf("dashboard layout_type is required in payload")
	}
	if payload["widgets"] == nil {
		return nil, fmt.Errorf("dashboard widgets are required in payload")
	}

	span := obs.StartSpan("dashboards.api_create")
	defer obs.FinishSpan(span)

	start := time.Now()
	rawData, err := ddClient.CreateDashboard(payload)
	apiDuration := time.Since(start).Milliseconds()

	if err != nil {
		obs.RecordAPICall("/api/v1/dashboard", "POST", 500, float64(apiDuration), err)
		obs.GetTracer().SetError(span, err)
		return nil, fmt.Errorf("failed to create dashboard: %w", err)
	}

	obs.RecordAPICall("/api/v1/dashboard", "POST", 201, float64(apiDuration), nil)

	// Parse response
	var dashboard map[string]interface{}
	if err := json.Unmarshal(rawData, &dashboard); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	dashboardID := getDashboardString(dashboard, "id", "")
	dashboardTitle := getDashboardString(dashboard, "title", "")

	obs.LogInfo(fmt.Sprintf("Created dashboard: %s (%s)", dashboardTitle, dashboardID))
	obs.GetMetrics().Count("dashboards.created", 1)

	return &DashboardsResponse{
		Status:    "ok",
		Operation: "create",
		Dashboard: dashboard,
		Message:   fmt.Sprintf("Created dashboard: %s (ID: %s)", dashboardTitle, dashboardID),
	}, nil
}

// updateDashboard updates an existing dashboard
func (c *DashboardsCommand) updateDashboard(obs *observability.Observability, ddClient *client.Client) (*DashboardsResponse, error) {
	// Read payload from file or stdin
	parseSpan := obs.StartSpan("dashboards.parse_payload")
	payload, err := c.readPayload()
	obs.FinishSpan(parseSpan)

	if err != nil {
		return nil, fmt.Errorf("failed to read payload: %w", err)
	}

	span := obs.StartSpan("dashboards.api_update")
	obs.GetTracer().SetTag(span, "dashboard.id", c.id)
	defer obs.FinishSpan(span)

	start := time.Now()
	rawData, err := ddClient.UpdateDashboard(c.id, payload)
	apiDuration := time.Since(start).Milliseconds()

	if err != nil {
		obs.RecordAPICall("/api/v1/dashboard/"+c.id, "PUT", 500, float64(apiDuration), err)
		obs.GetTracer().SetError(span, err)
		return nil, fmt.Errorf("failed to update dashboard: %w", err)
	}

	obs.RecordAPICall("/api/v1/dashboard/"+c.id, "PUT", 200, float64(apiDuration), nil)

	// Parse response
	var dashboard map[string]interface{}
	if err := json.Unmarshal(rawData, &dashboard); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	dashboardTitle := getDashboardString(dashboard, "title", "")

	obs.LogInfo(fmt.Sprintf("Updated dashboard: %s (%s)", dashboardTitle, c.id))
	obs.GetMetrics().Count("dashboards.updated", 1)

	return &DashboardsResponse{
		Status:    "ok",
		Operation: "update",
		Dashboard: dashboard,
		Message:   fmt.Sprintf("Updated dashboard: %s (ID: %s)", dashboardTitle, c.id),
	}, nil
}

// deleteDashboard deletes a dashboard
func (c *DashboardsCommand) deleteDashboard(obs *observability.Observability, ddClient *client.Client) (*DashboardsResponse, error) {
	span := obs.StartSpan("dashboards.api_delete")
	obs.GetTracer().SetTag(span, "dashboard.id", c.id)
	defer obs.FinishSpan(span)

	start := time.Now()
	err := ddClient.DeleteDashboard(c.id)
	apiDuration := time.Since(start).Milliseconds()

	if err != nil {
		obs.RecordAPICall("/api/v1/dashboard/"+c.id, "DELETE", 500, float64(apiDuration), err)
		obs.GetTracer().SetError(span, err)
		return nil, fmt.Errorf("failed to delete dashboard: %w", err)
	}

	obs.RecordAPICall("/api/v1/dashboard/"+c.id, "DELETE", 204, float64(apiDuration), nil)

	obs.LogInfo(fmt.Sprintf("Deleted dashboard: %s", c.id))
	obs.GetMetrics().Count("dashboards.deleted", 1)

	return &DashboardsResponse{
		Status:    "ok",
		Operation: "delete",
		Message:   fmt.Sprintf("Deleted dashboard: %s", c.id),
	}, nil
}

// readPayload reads JSON payload from file or stdin
func (c *DashboardsCommand) readPayload() (map[string]interface{}, error) {
	var reader io.Reader
	var err error

	if c.file == "-" {
		reader = os.Stdin
	} else {
		file, err := os.Open(c.file)
		if err != nil {
			return nil, fmt.Errorf("failed to open file: %w", err)
		}
		defer file.Close()
		reader = file
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read payload: %w", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return payload, nil
}

// printFormatted prints the response in conversational format
func (c *DashboardsCommand) printFormatted(response *DashboardsResponse) {
	fmt.Printf("Dashboard Operation: %s\n", response.Operation)
	fmt.Printf("Status: %s\n", response.Status)
	fmt.Println()

	switch response.Operation {
	case "list":
		fmt.Printf("Total Dashboards: %d\n", response.Total)
		if len(response.Dashboards) > 0 {
			fmt.Println()
			fmt.Println("Dashboards:")
			for i, d := range response.Dashboards {
				if i >= 20 {
					fmt.Printf("\n... and %d more dashboards\n", response.Total-20)
					break
				}
				fmt.Printf("\n%d. %s\n", i+1, d.Title)
				fmt.Printf("   ID: %s\n", d.ID)
				if d.LayoutType != "" {
					fmt.Printf("   Layout: %s\n", d.LayoutType)
				}
				if d.Author != "" {
					fmt.Printf("   Author: %s\n", d.Author)
				}
				if len(d.Tags) > 0 {
					fmt.Printf("   Tags: %s\n", strings.Join(d.Tags, ", "))
				}
				if d.Description != "" {
					desc := d.Description
					if len(desc) > 100 {
						desc = desc[:100] + "..."
					}
					fmt.Printf("   Description: %s\n", desc)
				}
				if d.URL != "" {
					fmt.Printf("   URL: %s\n", d.URL)
				}
			}
		}

	case "get":
		if dashboard, ok := response.Dashboard.(map[string]interface{}); ok {
			fmt.Printf("Title: %s\n", getDashboardString(dashboard, "title", ""))
			fmt.Printf("ID: %s\n", getDashboardString(dashboard, "id", ""))
			fmt.Printf("Layout Type: %s\n", getDashboardString(dashboard, "layout_type", ""))

			if desc := getDashboardString(dashboard, "description", ""); desc != "" {
				fmt.Printf("Description: %s\n", desc)
			}

			if author := getDashboardString(dashboard, "author_handle", ""); author != "" {
				fmt.Printf("Author: %s\n", author)
			}

			if url := getDashboardString(dashboard, "url", ""); url != "" {
				fmt.Printf("URL: %s\n", url)
			}

			// Widget count
			if widgets, ok := dashboard["widgets"].([]interface{}); ok {
				fmt.Printf("Widgets: %d\n", len(widgets))
			}

			// Tags
			if tags, ok := dashboard["tags"].([]interface{}); ok && len(tags) > 0 {
				tagStrings := make([]string, 0, len(tags))
				for _, tag := range tags {
					if tagStr, ok := tag.(string); ok {
						tagStrings = append(tagStrings, tagStr)
					}
				}
				if len(tagStrings) > 0 {
					fmt.Printf("Tags: %s\n", strings.Join(tagStrings, ", "))
				}
			}

			fmt.Println()
			fmt.Println("Use --json flag for full dashboard definition")
		}

	case "create", "update":
		if dashboard, ok := response.Dashboard.(map[string]interface{}); ok {
			fmt.Printf("Dashboard ID: %s\n", getDashboardString(dashboard, "id", ""))
			fmt.Printf("Title: %s\n", getDashboardString(dashboard, "title", ""))
			if url := getDashboardString(dashboard, "url", ""); url != "" {
				fmt.Printf("URL: %s\n", url)
			}
		}
		fmt.Printf("\n%s\n", response.Message)

	case "delete":
		fmt.Println(response.Message)
	}
}

// Help prints the help message
func (c *DashboardsCommand) Help() {
	fmt.Println("Usage: dd dashboards [options]")
	fmt.Println()
	fmt.Println("Manage Datadog dashboards (list, get, create, update, delete)")
	fmt.Println()
	fmt.Println("Options:")
	c.flags.PrintDefaults()
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # List all dashboards")
	fmt.Println("  dd dashboards --operation list")
	fmt.Println()
	fmt.Println("  # List dashboards with title filter")
	fmt.Println("  dd dashboards --operation list --title 'Production'")
	fmt.Println()
	fmt.Println("  # Get a specific dashboard")
	fmt.Println("  dd dashboards --operation get --id abc-123-def")
	fmt.Println()
	fmt.Println("  # Create a dashboard from a file")
	fmt.Println("  dd dashboards --operation create --file dashboard.json")
	fmt.Println()
	fmt.Println("  # Create a dashboard from stdin")
	fmt.Println("  cat dashboard.json | dd dashboards --operation create --file -")
	fmt.Println()
	fmt.Println("  # Update a dashboard")
	fmt.Println("  dd dashboards --operation update --id abc-123-def --file dashboard.json")
	fmt.Println()
	fmt.Println("  # Delete a dashboard")
	fmt.Println("  dd dashboards --operation delete --id abc-123-def")
	fmt.Println()
	fmt.Println("  # Output as JSON")
	fmt.Println("  dd dashboards --operation list --json")
	fmt.Println()
	fmt.Println("Dashboard JSON format for create/update:")
	fmt.Println("  {")
	fmt.Println("    \"title\": \"My Dashboard\",")
	fmt.Println("    \"description\": \"Dashboard description\",")
	fmt.Println("    \"layout_type\": \"ordered\",")
	fmt.Println("    \"widgets\": [")
	fmt.Println("      {")
	fmt.Println("        \"definition\": {")
	fmt.Println("          \"type\": \"timeseries\",")
	fmt.Println("          \"requests\": [...]")
	fmt.Println("        }")
	fmt.Println("      }")
	fmt.Println("    ]")
	fmt.Println("  }")
}

// Helper functions

func getDashboardString(m map[string]interface{}, key, defaultVal string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultVal
}

func getDashboardBool(m map[string]interface{}, key string, defaultVal bool) bool {
	if val, ok := m[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return defaultVal
}
