package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/datadog/skill/internal/client"
	"github.com/datadog/skill/internal/observability"
)

// IncidentsCommand manages Datadog incidents
type IncidentsCommand struct {
	flags *flag.FlagSet
}

// IncidentData represents a single incident
type IncidentData struct {
	ID                  string     `json:"id"`
	Title               string     `json:"title"`
	State               string     `json:"state"`
	Severity            string     `json:"severity"`
	CustomerImpacted    bool       `json:"customer_impacted"`
	Created             time.Time  `json:"created"`
	Modified            time.Time  `json:"modified"`
	Resolved            *time.Time `json:"resolved,omitempty"`
	CustomerImpactScope string     `json:"customer_impact_scope,omitempty"`
}

// IncidentsOutput represents the structured output
type IncidentsOutput struct {
	Status         string            `json:"status"`
	TotalIncidents int               `json:"total_incidents,omitempty"`
	Summary        *IncidentsSummary `json:"summary,omitempty"`
	Incidents      []IncidentData    `json:"incidents,omitempty"`
	Incident       *IncidentData     `json:"incident,omitempty"`
	Message        string            `json:"message,omitempty"`
}

// IncidentsSummary contains summary statistics
type IncidentsSummary struct {
	Active   int `json:"active"`
	Stable   int `json:"stable"`
	Resolved int `json:"resolved"`
}

// IncidentAPIResponse represents the API response structure
type IncidentAPIResponse struct {
	Data []struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		Attributes struct {
			Title               string  `json:"title"`
			State               string  `json:"state"`
			Severity            string  `json:"severity"`
			CustomerImpacted    bool    `json:"customer_impacted"`
			Created             string  `json:"created"`
			Modified            string  `json:"modified"`
			Resolved            *string `json:"resolved"`
			CustomerImpactScope string  `json:"customer_impact_scope"`
		} `json:"attributes"`
	} `json:"data"`
}

// SingleIncidentAPIResponse represents a single incident API response
type SingleIncidentAPIResponse struct {
	Data struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		Attributes struct {
			Title               string  `json:"title"`
			State               string  `json:"state"`
			Severity            string  `json:"severity"`
			CustomerImpacted    bool    `json:"customer_impacted"`
			Created             string  `json:"created"`
			Modified            string  `json:"modified"`
			Resolved            *string `json:"resolved"`
			CustomerImpactScope string  `json:"customer_impact_scope"`
		} `json:"attributes"`
	} `json:"data"`
}

// NewIncidentsCommand creates a new incidents command
func NewIncidentsCommand() *IncidentsCommand {
	cmd := &IncidentsCommand{
		flags: flag.NewFlagSet("incidents", flag.ContinueOnError),
	}

	return cmd
}

// Name returns the command name
func (c *IncidentsCommand) Name() string {
	return "incidents"
}

// Description returns the command description
func (c *IncidentsCommand) Description() string {
	return "Manage Datadog incidents - create, update, list, and close"
}

// Run executes the incidents command
func (c *IncidentsCommand) Run(args []string) error {
	// Initialize observability
	obs, err := observability.Init("manage-incidents", "production")
	if err != nil {
		return fmt.Errorf("failed to init observability: %w", err)
	}
	defer obs.Shutdown(0)

	if len(args) == 0 {
		c.Help()
		return fmt.Errorf("subcommand required: list, create, update, or close")
	}

	subcommand := args[0]
	subArgs := args[1:]

	obs.LogInfo(fmt.Sprintf("Executing incidents subcommand: %s", subcommand))

	// Create Datadog client
	span := obs.StartSpan("create_client")
	ddClient, err := client.NewClient()
	obs.FinishSpan(span)

	if err != nil {
		obs.LogError(fmt.Sprintf("Failed to create Datadog client: %s", err.Error()))
		return fmt.Errorf("failed to create Datadog client: %w", err)
	}

	// Route to subcommand
	switch subcommand {
	case "list":
		return c.runList(subArgs, ddClient, obs)
	case "create":
		return c.runCreate(subArgs, ddClient, obs)
	case "update":
		return c.runUpdate(subArgs, ddClient, obs)
	case "close":
		return c.runClose(subArgs, ddClient, obs)
	default:
		c.Help()
		return fmt.Errorf("unknown subcommand: %s", subcommand)
	}
}

// runList executes the list subcommand
func (c *IncidentsCommand) runList(args []string, ddClient *client.Client, obs *observability.Observability) error {
	listFlags := flag.NewFlagSet("list", flag.ExitOnError)
	service := listFlags.String("service", "", "Filter by service")
	status := listFlags.String("status", "", "Filter by status: active, stable, resolved")
	limit := listFlags.Int("limit", 100, "Max incidents to return")
	jsonOut := listFlags.Bool("json", false, "Output as JSON")

	if err := listFlags.Parse(args); err != nil {
		return err
	}

	obs.LogInfo(fmt.Sprintf("Listing incidents (status=%s)", *status))

	// Query incidents
	span := obs.StartSpan("query_incidents")
	if *status != "" {
		obs.GetTracer().SetTag(span, "status", *status)
	}
	if *service != "" {
		obs.GetTracer().SetTag(span, "service", *service)
	}

	start := time.Now()
	responseData, err := ddClient.ListIncidents(*status, *service, *limit)
	apiDuration := time.Since(start).Milliseconds()
	obs.FinishSpan(span)

	if err != nil {
		obs.RecordAPICall("/api/v2/incidents", "GET", 500, float64(apiDuration), err)
		obs.LogError(fmt.Sprintf("API call failed: %s", err.Error()))
		return fmt.Errorf("failed to list incidents: %w", err)
	}

	obs.RecordAPICall("/api/v2/incidents", "GET", 200, float64(apiDuration), nil)

	// Parse results
	span = obs.StartSpan("process_results")
	output, err := c.parseListResults(responseData)
	obs.FinishSpan(span)

	if err != nil {
		obs.LogError(fmt.Sprintf("Failed to parse results: %s", err.Error()))
		return fmt.Errorf("failed to parse results: %w", err)
	}

	// Record metrics
	obs.GetMetrics().Gauge("incidents.total", float64(output.TotalIncidents))
	obs.GetMetrics().Gauge("incidents.active", float64(output.Summary.Active), "state:active")
	obs.GetMetrics().Gauge("incidents.stable", float64(output.Summary.Stable), "state:stable")
	obs.GetMetrics().Gauge("incidents.resolved", float64(output.Summary.Resolved), "state:resolved")

	// Output
	if *jsonOut {
		jsonData, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	} else {
		c.printListFormatted(output)
	}

	obs.LogInfo(fmt.Sprintf("Listed %d incidents", output.TotalIncidents))
	return nil
}

// runCreate executes the create subcommand
func (c *IncidentsCommand) runCreate(args []string, ddClient *client.Client, obs *observability.Observability) error {
	createFlags := flag.NewFlagSet("create", flag.ExitOnError)
	title := createFlags.String("title", "", "Incident title (required)")
	severity := createFlags.String("severity", "UNKNOWN", "Severity: SEV-1, SEV-2, SEV-3, SEV-4, SEV-5, UNKNOWN")
	service := createFlags.String("service", "", "Service name (required)")
	message := createFlags.String("message", "", "Initial message")
	jsonOut := createFlags.Bool("json", false, "Output as JSON")

	if err := createFlags.Parse(args); err != nil {
		return err
	}

	if *title == "" {
		return fmt.Errorf("--title is required")
	}
	if *service == "" {
		return fmt.Errorf("--service is required")
	}

	obs.LogInfo(fmt.Sprintf("Creating incident: %s", *title))

	// Build fields
	fields := map[string]interface{}{
		"service": map[string]interface{}{
			"type":  "textbox",
			"value": *service,
		},
	}

	// Create incident
	span := obs.StartSpan("create_incident")
	obs.GetTracer().SetTag(span, "title", *title)
	obs.GetTracer().SetTag(span, "severity", *severity)

	start := time.Now()
	responseData, err := ddClient.CreateIncidentFull(*title, *severity, false, fields)
	apiDuration := time.Since(start).Milliseconds()
	obs.FinishSpan(span)

	if err != nil {
		obs.RecordAPICall("/api/v2/incidents", "POST", 500, float64(apiDuration), err)
		obs.LogError(fmt.Sprintf("API call failed: %s", err.Error()))
		return fmt.Errorf("failed to create incident: %w", err)
	}

	obs.RecordAPICall("/api/v2/incidents", "POST", 200, float64(apiDuration), nil)

	// Parse result
	span = obs.StartSpan("parse_result")
	output, err := c.parseCreateResult(responseData)
	obs.FinishSpan(span)

	if err != nil {
		obs.LogError(fmt.Sprintf("Failed to parse result: %s", err.Error()))
		return fmt.Errorf("failed to parse result: %w", err)
	}

	// Add timeline entry if message provided
	if *message != "" && output.Incident != nil {
		_ = ddClient.AddIncidentTimeline(output.Incident.ID, *message)
	}

	obs.GetMetrics().Count("incident.created", 1, fmt.Sprintf("severity:%s", *severity))

	// Output
	if *jsonOut {
		jsonData, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	} else {
		c.printCreateFormatted(output)
	}

	obs.LogInfo(fmt.Sprintf("Incident created: %s", output.Incident.ID))
	return nil
}

// runUpdate executes the update subcommand
func (c *IncidentsCommand) runUpdate(args []string, ddClient *client.Client, obs *observability.Observability) error {
	updateFlags := flag.NewFlagSet("update", flag.ExitOnError)
	id := updateFlags.String("id", "", "Incident ID (required)")
	status := updateFlags.String("status", "", "New status: active, stable, resolved (required)")
	message := updateFlags.String("message", "", "Update message")
	jsonOut := updateFlags.Bool("json", false, "Output as JSON")

	if err := updateFlags.Parse(args); err != nil {
		return err
	}

	if *id == "" {
		return fmt.Errorf("--id is required")
	}
	if *status == "" {
		return fmt.Errorf("--status is required")
	}

	obs.LogInfo(fmt.Sprintf("Updating incident %s to status: %s", *id, *status))

	// Update incident
	span := obs.StartSpan("update_incident")
	obs.GetTracer().SetTag(span, "incident_id", *id)
	obs.GetTracer().SetTag(span, "new_status", *status)

	start := time.Now()
	responseData, err := ddClient.UpdateIncident(*id, *status)
	apiDuration := time.Since(start).Milliseconds()
	obs.FinishSpan(span)

	if err != nil {
		obs.RecordAPICall(fmt.Sprintf("/api/v2/incidents/%s", *id), "PATCH", 500, float64(apiDuration), err)
		obs.LogError(fmt.Sprintf("API call failed: %s", err.Error()))
		return fmt.Errorf("failed to update incident: %w", err)
	}

	obs.RecordAPICall(fmt.Sprintf("/api/v2/incidents/%s", *id), "PATCH", 200, float64(apiDuration), nil)

	// Parse result
	span = obs.StartSpan("parse_result")
	output, err := c.parseUpdateResult(responseData)
	obs.FinishSpan(span)

	if err != nil {
		obs.LogError(fmt.Sprintf("Failed to parse result: %s", err.Error()))
		return fmt.Errorf("failed to parse result: %w", err)
	}

	// Add timeline entry if message provided
	if *message != "" && output.Incident != nil {
		_ = ddClient.AddIncidentTimeline(output.Incident.ID, *message)
	}

	obs.GetMetrics().Count("incident.updated", 1, fmt.Sprintf("new_status:%s", *status))

	// Output
	if *jsonOut {
		jsonData, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	} else {
		c.printUpdateFormatted(output)
	}

	obs.LogInfo(fmt.Sprintf("Incident updated: %s", output.Incident.ID))
	return nil
}

// runClose executes the close subcommand
func (c *IncidentsCommand) runClose(args []string, ddClient *client.Client, obs *observability.Observability) error {
	closeFlags := flag.NewFlagSet("close", flag.ExitOnError)
	id := closeFlags.String("id", "", "Incident ID (required)")
	message := closeFlags.String("message", "", "Closure message")
	jsonOut := closeFlags.Bool("json", false, "Output as JSON")

	if err := closeFlags.Parse(args); err != nil {
		return err
	}

	if *id == "" {
		return fmt.Errorf("--id is required")
	}

	obs.LogInfo(fmt.Sprintf("Closing incident: %s", *id))

	// Close incident (set status to resolved)
	span := obs.StartSpan("close_incident")
	obs.GetTracer().SetTag(span, "incident_id", *id)

	start := time.Now()
	responseData, err := ddClient.UpdateIncident(*id, "resolved")
	apiDuration := time.Since(start).Milliseconds()
	obs.FinishSpan(span)

	if err != nil {
		obs.RecordAPICall(fmt.Sprintf("/api/v2/incidents/%s", *id), "PATCH", 500, float64(apiDuration), err)
		obs.LogError(fmt.Sprintf("API call failed: %s", err.Error()))
		return fmt.Errorf("failed to close incident: %w", err)
	}

	obs.RecordAPICall(fmt.Sprintf("/api/v2/incidents/%s", *id), "PATCH", 200, float64(apiDuration), nil)

	// Parse result
	span = obs.StartSpan("parse_result")
	output, err := c.parseUpdateResult(responseData)
	obs.FinishSpan(span)

	if err != nil {
		obs.LogError(fmt.Sprintf("Failed to parse result: %s", err.Error()))
		return fmt.Errorf("failed to parse result: %w", err)
	}

	// Add timeline entry if message provided
	if *message != "" && output.Incident != nil {
		_ = ddClient.AddIncidentTimeline(output.Incident.ID, *message)
	}

	obs.GetMetrics().Count("incident.closed", 1)

	// Output
	if *jsonOut {
		jsonData, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	} else {
		c.printCloseFormatted(output)
	}

	obs.LogInfo(fmt.Sprintf("Incident closed: %s", output.Incident.ID))
	return nil
}

// parseListResults parses the list incidents API response
func (c *IncidentsCommand) parseListResults(data []byte) (*IncidentsOutput, error) {
	var response IncidentAPIResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	output := &IncidentsOutput{
		Status:         "ok",
		TotalIncidents: len(response.Data),
		Incidents:      []IncidentData{},
		Summary: &IncidentsSummary{
			Active:   0,
			Stable:   0,
			Resolved: 0,
		},
	}

	for _, item := range response.Data {
		created, _ := time.Parse(time.RFC3339, item.Attributes.Created)
		modified, _ := time.Parse(time.RFC3339, item.Attributes.Modified)

		var resolved *time.Time
		if item.Attributes.Resolved != nil {
			t, _ := time.Parse(time.RFC3339, *item.Attributes.Resolved)
			resolved = &t
		}

		incident := IncidentData{
			ID:                  item.ID,
			Title:               item.Attributes.Title,
			State:               item.Attributes.State,
			Severity:            item.Attributes.Severity,
			CustomerImpacted:    item.Attributes.CustomerImpacted,
			Created:             created,
			Modified:            modified,
			Resolved:            resolved,
			CustomerImpactScope: item.Attributes.CustomerImpactScope,
		}

		output.Incidents = append(output.Incidents, incident)

		// Update summary
		switch strings.ToLower(incident.State) {
		case "active":
			output.Summary.Active++
		case "stable":
			output.Summary.Stable++
		case "resolved":
			output.Summary.Resolved++
		}
	}

	return output, nil
}

// parseCreateResult parses the create incident API response
func (c *IncidentsCommand) parseCreateResult(data []byte) (*IncidentsOutput, error) {
	var response SingleIncidentAPIResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	created, _ := time.Parse(time.RFC3339, response.Data.Attributes.Created)
	modified, _ := time.Parse(time.RFC3339, response.Data.Attributes.Modified)

	incident := &IncidentData{
		ID:               response.Data.ID,
		Title:            response.Data.Attributes.Title,
		State:            response.Data.Attributes.State,
		Severity:         response.Data.Attributes.Severity,
		CustomerImpacted: response.Data.Attributes.CustomerImpacted,
		Created:          created,
		Modified:         modified,
	}

	return &IncidentsOutput{
		Status:   "created",
		Incident: incident,
		Message:  "Incident created successfully",
	}, nil
}

// parseUpdateResult parses the update incident API response
func (c *IncidentsCommand) parseUpdateResult(data []byte) (*IncidentsOutput, error) {
	var response SingleIncidentAPIResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	created, _ := time.Parse(time.RFC3339, response.Data.Attributes.Created)
	modified, _ := time.Parse(time.RFC3339, response.Data.Attributes.Modified)

	var resolved *time.Time
	if response.Data.Attributes.Resolved != nil {
		t, _ := time.Parse(time.RFC3339, *response.Data.Attributes.Resolved)
		resolved = &t
	}

	incident := &IncidentData{
		ID:               response.Data.ID,
		Title:            response.Data.Attributes.Title,
		State:            response.Data.Attributes.State,
		Severity:         response.Data.Attributes.Severity,
		CustomerImpacted: response.Data.Attributes.CustomerImpacted,
		Created:          created,
		Modified:         modified,
		Resolved:         resolved,
	}

	return &IncidentsOutput{
		Status:   "updated",
		Incident: incident,
		Message:  "Incident updated successfully",
	}, nil
}

// printListFormatted prints the list results in a conversational format
func (c *IncidentsCommand) printListFormatted(output *IncidentsOutput) {
	fmt.Println("Incident Summary")
	fmt.Println()
	fmt.Printf("Total incidents: %d\n", output.TotalIncidents)
	fmt.Printf("  Active: %d\n", output.Summary.Active)
	fmt.Printf("  Stable: %d\n", output.Summary.Stable)
	fmt.Printf("  Resolved: %d\n", output.Summary.Resolved)

	if len(output.Incidents) > 0 {
		fmt.Println()
		fmt.Println("Recent incidents:")

		count := 0
		for _, inc := range output.Incidents {
			if count >= 10 {
				break
			}

			stateSymbol := map[string]string{
				"active":   "ACTIVE",
				"stable":   "STABLE",
				"resolved": "RESOLVED",
			}[strings.ToLower(inc.State)]

			if stateSymbol == "" {
				stateSymbol = "UNKNOWN"
			}

			fmt.Printf("  [%s] %s\n", inc.ID, inc.Title)
			fmt.Printf("      State: %s | Severity: %s\n", stateSymbol, inc.Severity)
			count++
		}
	}
}

// printCreateFormatted prints the create result in a conversational format
func (c *IncidentsCommand) printCreateFormatted(output *IncidentsOutput) {
	fmt.Println("Incident created successfully")
	fmt.Println()
	fmt.Printf("ID: %s\n", output.Incident.ID)
	fmt.Printf("Title: %s\n", output.Incident.Title)
	fmt.Printf("Severity: %s\n", output.Incident.Severity)
	fmt.Printf("State: %s\n", output.Incident.State)
}

// printUpdateFormatted prints the update result in a conversational format
func (c *IncidentsCommand) printUpdateFormatted(output *IncidentsOutput) {
	fmt.Println("Incident updated successfully")
	fmt.Println()
	fmt.Printf("ID: %s\n", output.Incident.ID)
	fmt.Printf("Title: %s\n", output.Incident.Title)
	fmt.Printf("New State: %s\n", output.Incident.State)
}

// printCloseFormatted prints the close result in a conversational format
func (c *IncidentsCommand) printCloseFormatted(output *IncidentsOutput) {
	fmt.Println("Incident closed successfully")
	fmt.Println()
	fmt.Printf("ID: %s\n", output.Incident.ID)
	fmt.Printf("Title: %s\n", output.Incident.Title)
	fmt.Printf("State: %s\n", output.Incident.State)
	if output.Incident.Resolved != nil {
		fmt.Printf("Resolved: %s\n", output.Incident.Resolved.Format(time.RFC3339))
	}
}

// Help prints the help message
func (c *IncidentsCommand) Help() {
	fmt.Println("Usage: dd incidents <subcommand> [options]")
	fmt.Println()
	fmt.Println("Manage Datadog incidents - create, update, list, and close")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  list    List incidents")
	fmt.Println("  create  Create a new incident")
	fmt.Println("  update  Update an incident")
	fmt.Println("  close   Close an incident")
	fmt.Println()
	fmt.Println("List options:")
	fmt.Println("  --service string   Filter by service")
	fmt.Println("  --status string    Filter by status: active, stable, resolved")
	fmt.Println("  --limit int        Max incidents to return (default 100)")
	fmt.Println("  --json             Output as JSON")
	fmt.Println()
	fmt.Println("Create options:")
	fmt.Println("  --title string     Incident title (required)")
	fmt.Println("  --severity string  Severity: SEV-1, SEV-2, SEV-3, SEV-4, SEV-5, UNKNOWN (default UNKNOWN)")
	fmt.Println("  --service string   Service name (required)")
	fmt.Println("  --message string   Initial message")
	fmt.Println("  --json             Output as JSON")
	fmt.Println()
	fmt.Println("Update options:")
	fmt.Println("  --id string        Incident ID (required)")
	fmt.Println("  --status string    New status: active, stable, resolved (required)")
	fmt.Println("  --message string   Update message")
	fmt.Println("  --json             Output as JSON")
	fmt.Println()
	fmt.Println("Close options:")
	fmt.Println("  --id string        Incident ID (required)")
	fmt.Println("  --message string   Closure message")
	fmt.Println("  --json             Output as JSON")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  dd incidents list --status active")
	fmt.Println("  dd incidents create --title \"Payment API Down\" --service payment-api --severity SEV-1")
	fmt.Println("  dd incidents update --id abc123 --status stable")
	fmt.Println("  dd incidents close --id abc123 --message \"Issue resolved\"")
}
