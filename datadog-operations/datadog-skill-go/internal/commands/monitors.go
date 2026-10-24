package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/datadog/skill/internal/client"
	"github.com/datadog/skill/internal/observability"
)

// MonitorsCommand manages Datadog monitors
type MonitorsCommand struct{}

// MonitorData represents a single monitor
type MonitorData struct {
	ID      int      `json:"id"`
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Query   string   `json:"query"`
	State   string   `json:"state"`
	Tags    []string `json:"tags"`
	Message string   `json:"message"`
}

// MonitorsListOutput represents the list output structure
type MonitorsListOutput struct {
	Total    int                    `json:"total"`
	Summary  *MonitorsSummary       `json:"summary"`
	Monitors []MonitorData          `json:"monitors"`
	RawData  map[string]interface{} `json:"raw_data,omitempty"`
}

// MonitorsSummary contains summary statistics
type MonitorsSummary struct {
	Alert int `json:"alert"`
	Warn  int `json:"warn"`
	OK    int `json:"ok"`
}

// MonitorCreateOutput represents the create output structure
type MonitorCreateOutput struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Query   string `json:"query"`
	Message string `json:"message"`
	Created string `json:"created"`
	Status  string `json:"status"`
}

// MonitorMuteOutput represents the mute/unmute output structure
type MonitorMuteOutput struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

// MonitorDeleteOutput represents the delete output structure
type MonitorDeleteOutput struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
}

// MonitorCreator represents the creator of a monitor
type MonitorCreator struct {
	Email  string `json:"email"`
	Handle string `json:"handle"`
	Name   string `json:"name"`
	ID     int64  `json:"id"`
}

// MonitorAPIResponse represents the API response for a monitor
type MonitorAPIResponse struct {
	ID              int             `json:"id"`
	Name            string          `json:"name"`
	Type            string          `json:"type"`
	Query           string          `json:"query"`
	OverallState    string          `json:"overall_state"`
	Tags            []string        `json:"tags"`
	Message         string          `json:"message"`
	Created         string          `json:"created"`
	CreatedAt       int64           `json:"created_at"`
	Creator         *MonitorCreator `json:"creator"` // Changed from string to object
	Modified        string          `json:"modified"`
	Priority        *int            `json:"priority"`
	RestrictedRoles []string        `json:"restricted_roles,omitempty"`
}

// NewMonitorsCommand creates a new monitors command
func NewMonitorsCommand() *MonitorsCommand {
	return &MonitorsCommand{}
}

// Name returns the command name
func (c *MonitorsCommand) Name() string {
	return "monitors"
}

// Description returns the command description
func (c *MonitorsCommand) Description() string {
	return "Manage Datadog monitors - list, create, mute, unmute, delete"
}

// Run executes the monitors command
func (c *MonitorsCommand) Run(args []string) error {
	// Check for subcommand
	if len(args) == 0 {
		c.Help()
		return fmt.Errorf("subcommand required: list, create, mute, unmute, or delete")
	}

	subcommand := args[0]
	subArgs := args[1:]

	switch subcommand {
	case "list":
		return c.runList(subArgs)
	case "create":
		return c.runCreate(subArgs)
	case "mute":
		return c.runMute(subArgs)
	case "unmute":
		return c.runUnmute(subArgs)
	case "delete":
		return c.runDelete(subArgs)
	default:
		c.Help()
		return fmt.Errorf("unknown subcommand: %s", subcommand)
	}
}

// runList lists monitors
func (c *MonitorsCommand) runList(args []string) error {
	// Initialize observability
	obs, err := observability.Init("manage-monitors-list", "production")
	if err != nil {
		return fmt.Errorf("failed to init observability: %w", err)
	}
	defer obs.Shutdown(0)

	// Parse flags
	flags := flag.NewFlagSet("list", flag.ExitOnError)
	service := flags.String("service", "", "Filter by service tag")
	tags := flags.String("tags", "", "Filter by tags (comma-separated)")
	status := flags.String("status", "", "Filter by status (alert/warn/ok)")
	jsonOut := flags.Bool("json", false, "Output as JSON")

	if err := flags.Parse(args); err != nil {
		return err
	}

	obs.LogInfo("Listing monitors")

	// Build tags filter
	var tagsList []string
	if *service != "" {
		tagsList = append(tagsList, fmt.Sprintf("service:%s", *service))
	}
	if *tags != "" {
		for _, tag := range strings.Split(*tags, ",") {
			tagsList = append(tagsList, strings.TrimSpace(tag))
		}
	}

	// Create Datadog client
	span := obs.StartSpan("create_client")
	ddClient, err := client.NewClient()
	obs.FinishSpan(span)

	if err != nil {
		obs.LogError(fmt.Sprintf("Failed to create Datadog client: %s", err.Error()))
		return fmt.Errorf("failed to create Datadog client: %w", err)
	}

	// Query monitors
	span = obs.StartSpan("list_monitors")
	if len(tagsList) > 0 {
		obs.GetTracer().SetTag(span, "tags", strings.Join(tagsList, ","))
	}

	start := time.Now()
	responseData, err := ddClient.GetMonitors(tagsList, nil)
	apiDuration := time.Since(start).Milliseconds()
	obs.FinishSpan(span)

	if err != nil {
		obs.RecordAPICall("/api/v1/monitor", "GET", 500, float64(apiDuration), err)
		obs.LogError(fmt.Sprintf("API call failed: %s", err.Error()))
		return fmt.Errorf("failed to list monitors: %w", err)
	}

	obs.RecordAPICall("/api/v1/monitor", "GET", 200, float64(apiDuration), nil)

	// Parse results
	span = obs.StartSpan("process_results")
	output, err := c.parseListResults(responseData, *status)
	obs.FinishSpan(span)

	if err != nil {
		obs.LogError(fmt.Sprintf("Failed to parse results: %s", err.Error()))
		return fmt.Errorf("failed to parse results: %w", err)
	}

	// Record metrics
	obs.GetMetrics().Gauge("monitors.total", float64(output.Total))
	obs.GetMetrics().Gauge("monitors.alert", float64(output.Summary.Alert))
	obs.GetMetrics().Gauge("monitors.warn", float64(output.Summary.Warn))
	obs.GetMetrics().Gauge("monitors.ok", float64(output.Summary.OK))

	obs.LogInfo(fmt.Sprintf("Listed %d monitors", output.Total))

	// Output
	if *jsonOut {
		jsonData, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	} else {
		c.printListFormatted(*service, output)
	}

	return nil
}

// runCreate creates a monitor
func (c *MonitorsCommand) runCreate(args []string) error {
	// Initialize observability
	obs, err := observability.Init("manage-monitors-create", "production")
	if err != nil {
		return fmt.Errorf("failed to init observability: %w", err)
	}
	defer obs.Shutdown(0)

	// Parse flags
	flags := flag.NewFlagSet("create", flag.ExitOnError)
	name := flags.String("name", "", "Monitor name")
	query := flags.String("query", "", "Monitor query")
	message := flags.String("message", "", "Alert message")
	monitorType := flags.String("type", "metric alert", "Monitor type (metric/service/apm)")
	jsonOut := flags.Bool("json", false, "Output as JSON")

	if err := flags.Parse(args); err != nil {
		return err
	}

	// Validate required flags
	if *name == "" || *query == "" || *message == "" {
		return fmt.Errorf("--name, --query, and --message are required")
	}

	obs.LogInfo(fmt.Sprintf("Creating monitor: %s", *name))

	// Build payload
	payload := map[string]interface{}{
		"name":    *name,
		"type":    *monitorType,
		"query":   *query,
		"message": *message,
		"tags":    []string{},
		"options": map[string]interface{}{
			"notify_no_data":    true,
			"no_data_timeframe": 20,
		},
	}

	// Create Datadog client
	span := obs.StartSpan("create_client")
	ddClient, err := client.NewClient()
	obs.FinishSpan(span)

	if err != nil {
		obs.LogError(fmt.Sprintf("Failed to create Datadog client: %s", err.Error()))
		return fmt.Errorf("failed to create Datadog client: %w", err)
	}

	// Create monitor
	span = obs.StartSpan("create_monitor")
	obs.GetTracer().SetTag(span, "name", *name)
	obs.GetTracer().SetTag(span, "type", *monitorType)

	start := time.Now()
	responseData, err := ddClient.CreateMonitor(payload)
	apiDuration := time.Since(start).Milliseconds()
	obs.FinishSpan(span)

	if err != nil {
		obs.RecordAPICall("/api/v1/monitor", "POST", 500, float64(apiDuration), err)
		obs.LogError(fmt.Sprintf("API call failed: %s", err.Error()))
		return fmt.Errorf("failed to create monitor: %w", err)
	}

	obs.RecordAPICall("/api/v1/monitor", "POST", 200, float64(apiDuration), nil)

	// Parse response
	var response MonitorAPIResponse
	if err := json.Unmarshal(responseData, &response); err != nil {
		obs.LogError(fmt.Sprintf("Failed to parse response: %s", err.Error()))
		return fmt.Errorf("failed to parse response: %w", err)
	}

	obs.GetMetrics().Count("monitor.created", 1, fmt.Sprintf("type:%s", *monitorType))
	obs.LogInfo(fmt.Sprintf("Monitor created: %d", response.ID))

	output := MonitorCreateOutput{
		ID:      response.ID,
		Name:    response.Name,
		Type:    response.Type,
		Query:   response.Query,
		Message: response.Message,
		Created: response.Created,
		Status:  "created",
	}

	// Output
	if *jsonOut {
		jsonData, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	} else {
		fmt.Println("Monitor created successfully")
		fmt.Println()
		fmt.Printf("ID: %d\n", output.ID)
		fmt.Printf("Name: %s\n", output.Name)
		fmt.Printf("Type: %s\n", output.Type)
	}

	return nil
}

// runMute mutes a monitor
func (c *MonitorsCommand) runMute(args []string) error {
	// Initialize observability
	obs, err := observability.Init("manage-monitors-mute", "production")
	if err != nil {
		return fmt.Errorf("failed to init observability: %w", err)
	}
	defer obs.Shutdown(0)

	// Parse flags
	flags := flag.NewFlagSet("mute", flag.ExitOnError)
	id := flags.Int("id", 0, "Monitor ID")
	duration := flags.Int("duration", 0, "Duration in hours")
	jsonOut := flags.Bool("json", false, "Output as JSON")

	if err := flags.Parse(args); err != nil {
		return err
	}

	// Validate required flags
	if *id == 0 {
		return fmt.Errorf("--id is required")
	}

	obs.LogInfo(fmt.Sprintf("Muting monitor: %d", *id))

	// Build payload
	payload := map[string]interface{}{
		"scope": "*",
	}
	if *duration > 0 {
		endTime := time.Now().Add(time.Duration(*duration) * time.Hour).Unix()
		payload["end"] = endTime
	}

	// Create Datadog client
	span := obs.StartSpan("create_client")
	ddClient, err := client.NewClient()
	obs.FinishSpan(span)

	if err != nil {
		obs.LogError(fmt.Sprintf("Failed to create Datadog client: %s", err.Error()))
		return fmt.Errorf("failed to create Datadog client: %w", err)
	}

	// Mute monitor
	span = obs.StartSpan("mute_monitor")
	obs.GetTracer().SetTag(span, "monitor_id", strconv.Itoa(*id))
	obs.GetTracer().SetTag(span, "duration", strconv.Itoa(*duration))

	start := time.Now()
	responseData, err := ddClient.MuteMonitor(strconv.Itoa(*id), payload)
	apiDuration := time.Since(start).Milliseconds()
	obs.FinishSpan(span)

	if err != nil {
		obs.RecordAPICall(fmt.Sprintf("/api/v1/monitor/%d/mute", *id), "POST", 500, float64(apiDuration), err)
		obs.LogError(fmt.Sprintf("API call failed: %s", err.Error()))
		return fmt.Errorf("failed to mute monitor: %w", err)
	}

	obs.RecordAPICall(fmt.Sprintf("/api/v1/monitor/%d/mute", *id), "POST", 200, float64(apiDuration), nil)

	// Parse response
	var response MonitorAPIResponse
	if err := json.Unmarshal(responseData, &response); err != nil {
		obs.LogError(fmt.Sprintf("Failed to parse response: %s", err.Error()))
		return fmt.Errorf("failed to parse response: %w", err)
	}

	obs.GetMetrics().Count("monitor.muted", 1)
	obs.LogInfo(fmt.Sprintf("Monitor muted: %d", response.ID))

	output := MonitorMuteOutput{
		ID:     response.ID,
		Name:   response.Name,
		Status: "muted",
	}

	// Output
	if *jsonOut {
		jsonData, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	} else {
		fmt.Println("Monitor muted successfully")
		fmt.Println()
		fmt.Printf("ID: %d\n", output.ID)
		fmt.Printf("Name: %s\n", output.Name)
		if *duration > 0 {
			fmt.Printf("Duration: %d hours\n", *duration)
		}
	}

	return nil
}

// runUnmute unmutes a monitor
func (c *MonitorsCommand) runUnmute(args []string) error {
	// Initialize observability
	obs, err := observability.Init("manage-monitors-unmute", "production")
	if err != nil {
		return fmt.Errorf("failed to init observability: %w", err)
	}
	defer obs.Shutdown(0)

	// Parse flags
	flags := flag.NewFlagSet("unmute", flag.ExitOnError)
	id := flags.Int("id", 0, "Monitor ID")
	jsonOut := flags.Bool("json", false, "Output as JSON")

	if err := flags.Parse(args); err != nil {
		return err
	}

	// Validate required flags
	if *id == 0 {
		return fmt.Errorf("--id is required")
	}

	obs.LogInfo(fmt.Sprintf("Unmuting monitor: %d", *id))

	// Build payload
	payload := map[string]interface{}{
		"scope": "*",
	}

	// Create Datadog client
	span := obs.StartSpan("create_client")
	ddClient, err := client.NewClient()
	obs.FinishSpan(span)

	if err != nil {
		obs.LogError(fmt.Sprintf("Failed to create Datadog client: %s", err.Error()))
		return fmt.Errorf("failed to create Datadog client: %w", err)
	}

	// Unmute monitor
	span = obs.StartSpan("unmute_monitor")
	obs.GetTracer().SetTag(span, "monitor_id", strconv.Itoa(*id))

	start := time.Now()
	responseData, err := ddClient.UnmuteMonitor(strconv.Itoa(*id), payload)
	apiDuration := time.Since(start).Milliseconds()
	obs.FinishSpan(span)

	if err != nil {
		obs.RecordAPICall(fmt.Sprintf("/api/v1/monitor/%d/unmute", *id), "POST", 500, float64(apiDuration), err)
		obs.LogError(fmt.Sprintf("API call failed: %s", err.Error()))
		return fmt.Errorf("failed to unmute monitor: %w", err)
	}

	obs.RecordAPICall(fmt.Sprintf("/api/v1/monitor/%d/unmute", *id), "POST", 200, float64(apiDuration), nil)

	// Parse response
	var response MonitorAPIResponse
	if err := json.Unmarshal(responseData, &response); err != nil {
		obs.LogError(fmt.Sprintf("Failed to parse response: %s", err.Error()))
		return fmt.Errorf("failed to parse response: %w", err)
	}

	obs.GetMetrics().Count("monitor.unmuted", 1)
	obs.LogInfo(fmt.Sprintf("Monitor unmuted: %d", response.ID))

	output := MonitorMuteOutput{
		ID:     response.ID,
		Name:   response.Name,
		Status: "unmuted",
	}

	// Output
	if *jsonOut {
		jsonData, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	} else {
		fmt.Println("Monitor unmuted successfully")
		fmt.Println()
		fmt.Printf("ID: %d\n", output.ID)
		fmt.Printf("Name: %s\n", output.Name)
	}

	return nil
}

// runDelete deletes a monitor
func (c *MonitorsCommand) runDelete(args []string) error {
	// Initialize observability
	obs, err := observability.Init("manage-monitors-delete", "production")
	if err != nil {
		return fmt.Errorf("failed to init observability: %w", err)
	}
	defer obs.Shutdown(0)

	// Parse flags
	flags := flag.NewFlagSet("delete", flag.ExitOnError)
	id := flags.Int("id", 0, "Monitor ID")
	jsonOut := flags.Bool("json", false, "Output as JSON")

	if err := flags.Parse(args); err != nil {
		return err
	}

	// Validate required flags
	if *id == 0 {
		return fmt.Errorf("--id is required")
	}

	obs.LogInfo(fmt.Sprintf("Deleting monitor: %d", *id))

	// Create Datadog client
	span := obs.StartSpan("create_client")
	ddClient, err := client.NewClient()
	obs.FinishSpan(span)

	if err != nil {
		obs.LogError(fmt.Sprintf("Failed to create Datadog client: %s", err.Error()))
		return fmt.Errorf("failed to create Datadog client: %w", err)
	}

	// Delete monitor
	span = obs.StartSpan("delete_monitor")
	obs.GetTracer().SetTag(span, "monitor_id", strconv.Itoa(*id))

	start := time.Now()
	_, err = ddClient.DeleteMonitor(strconv.Itoa(*id))
	apiDuration := time.Since(start).Milliseconds()
	obs.FinishSpan(span)

	if err != nil {
		obs.RecordAPICall(fmt.Sprintf("/api/v1/monitor/%d", *id), "DELETE", 500, float64(apiDuration), err)
		obs.LogError(fmt.Sprintf("API call failed: %s", err.Error()))
		return fmt.Errorf("failed to delete monitor: %w", err)
	}

	obs.RecordAPICall(fmt.Sprintf("/api/v1/monitor/%d", *id), "DELETE", 200, float64(apiDuration), nil)
	obs.GetMetrics().Count("monitor.deleted", 1)
	obs.LogInfo(fmt.Sprintf("Monitor deleted: %d", *id))

	output := MonitorDeleteOutput{
		ID:     *id,
		Status: "deleted",
	}

	// Output
	if *jsonOut {
		jsonData, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	} else {
		fmt.Println("Monitor deleted successfully")
		fmt.Println()
		fmt.Printf("ID: %d\n", output.ID)
	}

	return nil
}

// parseListResults parses the monitor list API response
func (c *MonitorsCommand) parseListResults(data []byte, statusFilter string) (*MonitorsListOutput, error) {
	var response []MonitorAPIResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	output := &MonitorsListOutput{
		Summary: &MonitorsSummary{
			Alert: 0,
			Warn:  0,
			OK:    0,
		},
		Monitors: []MonitorData{},
	}

	// Process each monitor
	for _, m := range response {
		state := m.OverallState

		// Apply status filter
		if statusFilter != "" {
			normalizedState := strings.ToLower(state)
			normalizedFilter := strings.ToLower(statusFilter)

			// Map "ok" to match both "OK" and "No Data"
			if normalizedFilter == "ok" && (normalizedState != "ok" && normalizedState != "no data") {
				continue
			} else if normalizedFilter != "ok" && normalizedState != normalizedFilter {
				continue
			}
		}

		monitor := MonitorData{
			ID:      m.ID,
			Name:    m.Name,
			Type:    m.Type,
			Query:   m.Query,
			State:   state,
			Tags:    m.Tags,
			Message: m.Message,
		}
		output.Monitors = append(output.Monitors, monitor)

		// Update summary
		switch strings.ToLower(state) {
		case "alert":
			output.Summary.Alert++
		case "warn":
			output.Summary.Warn++
		case "ok", "no data":
			output.Summary.OK++
		}
	}

	output.Total = len(output.Monitors)

	// Sort monitors by state (alert first, then warn, then ok)
	sort.Slice(output.Monitors, func(i, j int) bool {
		stateOrder := map[string]int{"Alert": 0, "Warn": 1, "OK": 2, "No Data": 2}
		iOrder := stateOrder[output.Monitors[i].State]
		jOrder := stateOrder[output.Monitors[j].State]
		if iOrder != jOrder {
			return iOrder < jOrder
		}
		return output.Monitors[i].Name < output.Monitors[j].Name
	})

	return output, nil
}

// printListFormatted prints the monitor list in conversational format
func (c *MonitorsCommand) printListFormatted(serviceName string, output *MonitorsListOutput) {
	fmt.Println("Monitor Summary")
	if serviceName != "" {
		fmt.Printf("Service: %s\n", serviceName)
	}
	fmt.Println()
	fmt.Printf("Total monitors: %d\n", output.Total)
	fmt.Printf("  Alert: %d\n", output.Summary.Alert)
	fmt.Printf("  Warn: %d\n", output.Summary.Warn)
	fmt.Printf("  OK/No Data: %d\n", output.Summary.OK)

	if len(output.Monitors) > 0 {
		fmt.Println()
		fmt.Println("Monitors:")
		for i, m := range output.Monitors {
			if i >= 10 {
				fmt.Printf("\n... and %d more monitors (use --json to see all)\n", len(output.Monitors)-10)
				break
			}
			stateSymbol := map[string]string{
				"Alert":   "✗",
				"Warn":    "⚠",
				"OK":      "✓",
				"No Data": "⚪",
			}[m.State]
			if stateSymbol == "" {
				stateSymbol = "?"
			}

			fmt.Printf("  %s [%d] %s\n", stateSymbol, m.ID, m.Name)
			fmt.Printf("      State: %s | Type: %s\n", m.State, m.Type)
		}
	}
}

// Help prints the help message
func (c *MonitorsCommand) Help() {
	fmt.Println("Usage: dd monitors <subcommand> [options]")
	fmt.Println()
	fmt.Println("Manage Datadog monitors - list, create, mute, unmute, and delete monitors.")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  list       List monitors")
	fmt.Println("  create     Create a new monitor")
	fmt.Println("  mute       Mute a monitor")
	fmt.Println("  unmute     Unmute a monitor")
	fmt.Println("  delete     Delete a monitor")
	fmt.Println()
	fmt.Println("List Options:")
	fmt.Println("  --service string")
	fmt.Println("        Filter by service tag")
	fmt.Println("  --tags string")
	fmt.Println("        Filter by tags (comma-separated)")
	fmt.Println("  --status string")
	fmt.Println("        Filter by status (alert/warn/ok)")
	fmt.Println("  --json")
	fmt.Println("        Output as JSON")
	fmt.Println()
	fmt.Println("Create Options:")
	fmt.Println("  --name string")
	fmt.Println("        Monitor name (required)")
	fmt.Println("  --query string")
	fmt.Println("        Monitor query (required)")
	fmt.Println("  --message string")
	fmt.Println("        Alert message (required)")
	fmt.Println("  --type string")
	fmt.Println("        Monitor type (default: metric alert)")
	fmt.Println("  --json")
	fmt.Println("        Output as JSON")
	fmt.Println()
	fmt.Println("Mute Options:")
	fmt.Println("  --id int")
	fmt.Println("        Monitor ID (required)")
	fmt.Println("  --duration int")
	fmt.Println("        Duration in hours (optional)")
	fmt.Println("  --json")
	fmt.Println("        Output as JSON")
	fmt.Println()
	fmt.Println("Unmute Options:")
	fmt.Println("  --id int")
	fmt.Println("        Monitor ID (required)")
	fmt.Println("  --json")
	fmt.Println("        Output as JSON")
	fmt.Println()
	fmt.Println("Delete Options:")
	fmt.Println("  --id int")
	fmt.Println("        Monitor ID (required)")
	fmt.Println("  --json")
	fmt.Println("        Output as JSON")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # List all monitors")
	fmt.Println("  dd monitors list")
	fmt.Println()
	fmt.Println("  # List monitors for a service")
	fmt.Println("  dd monitors list --service payment-api")
	fmt.Println()
	fmt.Println("  # Create error rate monitor")
	fmt.Println("  dd monitors create --name \"High Error Rate\" \\")
	fmt.Println("    --query \"avg(last_5m):sum:trace.express.request.errors{service:my-service}.as_count() > 10\" \\")
	fmt.Println("    --message \"Error rate is high @slack-alerts\"")
	fmt.Println()
	fmt.Println("  # Mute monitor for 2 hours")
	fmt.Println("  dd monitors mute --id 12345 --duration 2")
	fmt.Println()
	fmt.Println("  # Unmute monitor")
	fmt.Println("  dd monitors unmute --id 12345")
	fmt.Println()
	fmt.Println("  # Delete monitor")
	fmt.Println("  dd monitors delete --id 12345")
}
