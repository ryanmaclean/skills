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

// SyntheticsCommand manages Datadog synthetic tests
type SyntheticsCommand struct{}

// SyntheticTestData represents a single synthetic test
type SyntheticTestData struct {
	PublicID string   `json:"public_id"`
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Status   string   `json:"status"`
	Tags     []string `json:"tags"`
	Locations []string `json:"locations"`
	Message  string   `json:"message"`
	URL      string   `json:"url,omitempty"`
}

// SyntheticsListOutput represents the list output structure
type SyntheticsListOutput struct {
	Total     int                    `json:"total"`
	Summary   *SyntheticsSummary     `json:"summary"`
	Tests     []SyntheticTestData    `json:"tests"`
	RawData   map[string]interface{} `json:"raw_data,omitempty"`
}

// SyntheticsSummary contains summary statistics
type SyntheticsSummary struct {
	API       int `json:"api"`
	Browser   int `json:"browser"`
	Live      int `json:"live"`
	Paused    int `json:"paused"`
}

// SyntheticTestAPIResponse represents the API response for a synthetic test
type SyntheticTestAPIResponse struct {
	PublicID  string   `json:"public_id"`
	Name      string   `json:"name"`
	Type      string   `json:"type"`
	Status    string   `json:"status"`
	Tags      []string `json:"tags"`
	Locations []string `json:"locations"`
	Message   string   `json:"message"`
	Config    struct {
		Request struct {
			URL string `json:"url"`
		} `json:"request"`
		Assertions []map[string]interface{} `json:"assertions"`
	} `json:"config"`
	Options struct {
		TickEvery int64 `json:"tick_every"`
	} `json:"options"`
	MonitorID int64  `json:"monitor_id"`
	CreatedAt string `json:"created_at"`
	ModifiedAt string `json:"modified_at"`
}

// SyntheticTestCreateOutput represents the create output structure
type SyntheticTestCreateOutput struct {
	PublicID  string `json:"public_id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Status    string `json:"status"`
}

// SyntheticTestOperationOutput represents the operation output structure
type SyntheticTestOperationOutput struct {
	PublicID  string `json:"public_id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	Operation string `json:"operation"`
}

// SyntheticTestDeleteOutput represents the delete output structure
type SyntheticTestDeleteOutput struct {
	PublicID string `json:"public_id"`
	Status   string `json:"status"`
}

// SyntheticResultsOutput represents the results output structure
type SyntheticResultsOutput struct {
	TestID      string                  `json:"test_id"`
	TestName    string                  `json:"test_name"`
	TotalResults int                    `json:"total_results"`
	PassRate    float64                 `json:"pass_rate"`
	FailRate    float64                 `json:"fail_rate"`
	AvgDuration float64                 `json:"avg_duration_ms"`
	Results     []SyntheticResultData   `json:"results"`
}

// SyntheticResultData represents a single test result
type SyntheticResultData struct {
	ResultID  string  `json:"result_id"`
	CheckTime string  `json:"check_time"`
	Status    int     `json:"status"`
	Duration  float64 `json:"duration_ms"`
	Location  string  `json:"location"`
	Error     string  `json:"error,omitempty"`
}

// NewSyntheticsCommand creates a new synthetics command
func NewSyntheticsCommand() *SyntheticsCommand {
	return &SyntheticsCommand{}
}

// Name returns the command name
func (c *SyntheticsCommand) Name() string {
	return "synthetics"
}

// Description returns the command description
func (c *SyntheticsCommand) Description() string {
	return "Manage Datadog synthetic tests - list, get, results, create, update, delete, pause, resume"
}

// Run executes the synthetics command
func (c *SyntheticsCommand) Run(args []string) error {
	// Check for subcommand
	if len(args) == 0 {
		c.Help()
		return fmt.Errorf("subcommand required: list, get, results, create, update, delete, pause, or resume")
	}

	subcommand := args[0]
	subArgs := args[1:]

	switch subcommand {
	case "list":
		return c.runList(subArgs)
	case "get":
		return c.runGet(subArgs)
	case "results":
		return c.runResults(subArgs)
	case "create":
		return c.runCreate(subArgs)
	case "update":
		return c.runUpdate(subArgs)
	case "delete":
		return c.runDelete(subArgs)
	case "pause":
		return c.runPause(subArgs)
	case "resume":
		return c.runResume(subArgs)
	default:
		c.Help()
		return fmt.Errorf("unknown subcommand: %s", subcommand)
	}
}

// runList lists synthetic tests
func (c *SyntheticsCommand) runList(args []string) error {
	// Initialize observability
	obs, err := observability.Init("manage-synthetics-list", "production")
	if err != nil {
		return fmt.Errorf("failed to init observability: %w", err)
	}
	defer obs.Shutdown(0)

	// Parse flags
	flags := flag.NewFlagSet("list", flag.ExitOnError)
	testType := flags.String("type", "", "Filter by test type (api/browser)")
	tags := flags.String("tags", "", "Filter by tags (comma-separated)")
	jsonOut := flags.Bool("json", false, "Output as JSON")

	if err := flags.Parse(args); err != nil {
		return err
	}

	obs.LogInfo("Listing synthetic tests")

	// Create Datadog client
	span := obs.StartSpan("create_client")
	ddClient, err := client.NewClient()
	obs.FinishSpan(span)

	if err != nil {
		obs.LogError(fmt.Sprintf("Failed to create Datadog client: %s", err.Error()))
		return fmt.Errorf("failed to create Datadog client: %w", err)
	}

	// Query synthetic tests
	span = obs.StartSpan("list_synthetic_tests")
	if *testType != "" {
		obs.GetTracer().SetTag(span, "test_type", *testType)
	}

	start := time.Now()
	responseData, err := ddClient.ListSyntheticTests(*testType)
	apiDuration := time.Since(start).Milliseconds()
	obs.FinishSpan(span)

	if err != nil {
		obs.RecordAPICall("/api/v1/synthetics/tests", "GET", 500, float64(apiDuration), err)
		obs.LogError(fmt.Sprintf("API call failed: %s", err.Error()))
		return fmt.Errorf("failed to list synthetic tests: %w", err)
	}

	obs.RecordAPICall("/api/v1/synthetics/tests", "GET", 200, float64(apiDuration), nil)

	// Parse results
	span = obs.StartSpan("process_results")
	output, err := c.parseListResults(responseData, *tags)
	obs.FinishSpan(span)

	if err != nil {
		obs.LogError(fmt.Sprintf("Failed to parse results: %s", err.Error()))
		return fmt.Errorf("failed to parse results: %w", err)
	}

	// Record metrics
	obs.GetMetrics().Gauge("synthetics.total", float64(output.Total))
	obs.GetMetrics().Gauge("synthetics.api", float64(output.Summary.API))
	obs.GetMetrics().Gauge("synthetics.browser", float64(output.Summary.Browser))
	obs.GetMetrics().Gauge("synthetics.paused", float64(output.Summary.Paused))

	obs.LogInfo(fmt.Sprintf("Listed %d synthetic tests", output.Total))

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

	return nil
}

// runGet retrieves a specific synthetic test
func (c *SyntheticsCommand) runGet(args []string) error {
	// Initialize observability
	obs, err := observability.Init("manage-synthetics-get", "production")
	if err != nil {
		return fmt.Errorf("failed to init observability: %w", err)
	}
	defer obs.Shutdown(0)

	// Parse flags
	flags := flag.NewFlagSet("get", flag.ExitOnError)
	id := flags.String("id", "", "Test public ID")
	jsonOut := flags.Bool("json", false, "Output as JSON")

	if err := flags.Parse(args); err != nil {
		return err
	}

	// Validate required flags
	if *id == "" {
		return fmt.Errorf("--id is required")
	}

	obs.LogInfo(fmt.Sprintf("Getting synthetic test: %s", *id))

	// Create Datadog client
	span := obs.StartSpan("create_client")
	ddClient, err := client.NewClient()
	obs.FinishSpan(span)

	if err != nil {
		obs.LogError(fmt.Sprintf("Failed to create Datadog client: %s", err.Error()))
		return fmt.Errorf("failed to create Datadog client: %w", err)
	}

	// Get synthetic test
	span = obs.StartSpan("get_synthetic_test")
	obs.GetTracer().SetTag(span, "test_id", *id)

	start := time.Now()
	responseData, err := ddClient.GetSyntheticTest(*id)
	apiDuration := time.Since(start).Milliseconds()
	obs.FinishSpan(span)

	if err != nil {
		obs.RecordAPICall(fmt.Sprintf("/api/v1/synthetics/tests/%s", *id), "GET", 500, float64(apiDuration), err)
		obs.LogError(fmt.Sprintf("API call failed: %s", err.Error()))
		return fmt.Errorf("failed to get synthetic test: %w", err)
	}

	obs.RecordAPICall(fmt.Sprintf("/api/v1/synthetics/tests/%s", *id), "GET", 200, float64(apiDuration), nil)

	// Parse response
	var response SyntheticTestAPIResponse
	if err := json.Unmarshal(responseData, &response); err != nil {
		obs.LogError(fmt.Sprintf("Failed to parse response: %s", err.Error()))
		return fmt.Errorf("failed to parse response: %w", err)
	}

	obs.LogInfo(fmt.Sprintf("Retrieved synthetic test: %s", response.Name))

	// Output
	if *jsonOut {
		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	} else {
		c.printTestDetails(response)
	}

	return nil
}

// runResults retrieves test results
func (c *SyntheticsCommand) runResults(args []string) error {
	// Initialize observability
	obs, err := observability.Init("manage-synthetics-results", "production")
	if err != nil {
		return fmt.Errorf("failed to init observability: %w", err)
	}
	defer obs.Shutdown(0)

	// Parse flags
	flags := flag.NewFlagSet("results", flag.ExitOnError)
	id := flags.String("id", "", "Test public ID")
	hours := flags.Int("hours", 24, "Hours to look back (default: 24)")
	jsonOut := flags.Bool("json", false, "Output as JSON")

	if err := flags.Parse(args); err != nil {
		return err
	}

	// Validate required flags
	if *id == "" {
		return fmt.Errorf("--id is required")
	}

	obs.LogInfo(fmt.Sprintf("Getting synthetic test results: %s", *id))

	// Create Datadog client
	span := obs.StartSpan("create_client")
	ddClient, err := client.NewClient()
	obs.FinishSpan(span)

	if err != nil {
		obs.LogError(fmt.Sprintf("Failed to create Datadog client: %s", err.Error()))
		return fmt.Errorf("failed to create Datadog client: %w", err)
	}

	// Get test details first for name
	testData, _ := ddClient.GetSyntheticTest(*id)
	var test SyntheticTestAPIResponse
	testName := *id
	if testData != nil {
		json.Unmarshal(testData, &test)
		testName = test.Name
	}

	// Query test results
	span = obs.StartSpan("get_synthetic_results")
	obs.GetTracer().SetTag(span, "test_id", *id)
	obs.GetTracer().SetTag(span, "hours", strconv.Itoa(*hours))

	toTime := time.Now()
	fromTime := toTime.Add(-time.Duration(*hours) * time.Hour)

	start := time.Now()
	responseData, err := ddClient.GetSyntheticResults(*id, fromTime, toTime)
	apiDuration := time.Since(start).Milliseconds()
	obs.FinishSpan(span)

	if err != nil {
		obs.RecordAPICall(fmt.Sprintf("/api/v1/synthetics/tests/%s/results", *id), "GET", 500, float64(apiDuration), err)
		obs.LogError(fmt.Sprintf("API call failed: %s", err.Error()))
		return fmt.Errorf("failed to get synthetic test results: %w", err)
	}

	obs.RecordAPICall(fmt.Sprintf("/api/v1/synthetics/tests/%s/results", *id), "GET", 200, float64(apiDuration), nil)

	// Parse results
	span = obs.StartSpan("process_results")
	output, err := c.parseResults(responseData, *id, testName)
	obs.FinishSpan(span)

	if err != nil {
		obs.LogError(fmt.Sprintf("Failed to parse results: %s", err.Error()))
		return fmt.Errorf("failed to parse results: %w", err)
	}

	// Record metrics
	obs.GetMetrics().Gauge("synthetics.results.total", float64(output.TotalResults))
	obs.GetMetrics().Gauge("synthetics.results.pass_rate", output.PassRate)
	obs.GetMetrics().Gauge("synthetics.results.avg_duration", output.AvgDuration)

	obs.LogInfo(fmt.Sprintf("Retrieved %d test results, pass rate: %.2f%%", output.TotalResults, output.PassRate))

	// Output
	if *jsonOut {
		jsonData, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	} else {
		c.printResultsFormatted(output, *hours)
	}

	return nil
}

// runCreate creates a synthetic test
func (c *SyntheticsCommand) runCreate(args []string) error {
	// Initialize observability
	obs, err := observability.Init("manage-synthetics-create", "production")
	if err != nil {
		return fmt.Errorf("failed to init observability: %w", err)
	}
	defer obs.Shutdown(0)

	// Parse flags
	flags := flag.NewFlagSet("create", flag.ExitOnError)
	name := flags.String("name", "", "Test name")
	url := flags.String("url", "", "URL to test")
	testType := flags.String("type", "api", "Test type (api/browser)")
	message := flags.String("message", "", "Alert message")
	locations := flags.String("locations", "aws:us-east-1", "Test locations (comma-separated)")
	jsonOut := flags.Bool("json", false, "Output as JSON")

	if err := flags.Parse(args); err != nil {
		return err
	}

	// Validate required flags
	if *name == "" || *url == "" {
		return fmt.Errorf("--name and --url are required")
	}

	obs.LogInfo(fmt.Sprintf("Creating synthetic test: %s", *name))

	// Build payload
	locationsList := strings.Split(*locations, ",")
	for i := range locationsList {
		locationsList[i] = strings.TrimSpace(locationsList[i])
	}

	payload := map[string]interface{}{
		"name":    *name,
		"type":    *testType,
		"status":  "live",
		"tags":    []string{},
		"locations": locationsList,
		"message": *message,
		"config": map[string]interface{}{
			"request": map[string]interface{}{
				"url":    *url,
				"method": "GET",
			},
			"assertions": []map[string]interface{}{
				{
					"type":     "statusCode",
					"operator": "is",
					"target":   200,
				},
			},
		},
		"options": map[string]interface{}{
			"tick_every":      60,
			"follow_redirects": true,
			"min_failure_duration": 0,
			"min_location_failed": 1,
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

	// Create synthetic test
	span = obs.StartSpan("create_synthetic_test")
	obs.GetTracer().SetTag(span, "name", *name)
	obs.GetTracer().SetTag(span, "type", *testType)

	start := time.Now()
	responseData, err := ddClient.CreateSyntheticTest(payload)
	apiDuration := time.Since(start).Milliseconds()
	obs.FinishSpan(span)

	if err != nil {
		obs.RecordAPICall("/api/v1/synthetics/tests", "POST", 500, float64(apiDuration), err)
		obs.LogError(fmt.Sprintf("API call failed: %s", err.Error()))
		return fmt.Errorf("failed to create synthetic test: %w", err)
	}

	obs.RecordAPICall("/api/v1/synthetics/tests", "POST", 200, float64(apiDuration), nil)

	// Parse response
	var response SyntheticTestAPIResponse
	if err := json.Unmarshal(responseData, &response); err != nil {
		obs.LogError(fmt.Sprintf("Failed to parse response: %s", err.Error()))
		return fmt.Errorf("failed to parse response: %w", err)
	}

	obs.GetMetrics().Count("synthetic.created", 1, fmt.Sprintf("type:%s", *testType))
	obs.LogInfo(fmt.Sprintf("Synthetic test created: %s", response.PublicID))

	output := SyntheticTestCreateOutput{
		PublicID: response.PublicID,
		Name:     response.Name,
		Type:     response.Type,
		Status:   "created",
	}

	// Output
	if *jsonOut {
		jsonData, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	} else {
		fmt.Println("Synthetic test created successfully")
		fmt.Println()
		fmt.Printf("ID: %s\n", output.PublicID)
		fmt.Printf("Name: %s\n", output.Name)
		fmt.Printf("Type: %s\n", output.Type)
	}

	return nil
}

// runUpdate updates a synthetic test
func (c *SyntheticsCommand) runUpdate(args []string) error {
	// Initialize observability
	obs, err := observability.Init("manage-synthetics-update", "production")
	if err != nil {
		return fmt.Errorf("failed to init observability: %w", err)
	}
	defer obs.Shutdown(0)

	// Parse flags
	flags := flag.NewFlagSet("update", flag.ExitOnError)
	id := flags.String("id", "", "Test public ID")
	name := flags.String("name", "", "New test name")
	url := flags.String("url", "", "New URL to test")
	message := flags.String("message", "", "New alert message")
	jsonOut := flags.Bool("json", false, "Output as JSON")

	if err := flags.Parse(args); err != nil {
		return err
	}

	// Validate required flags
	if *id == "" {
		return fmt.Errorf("--id is required")
	}

	obs.LogInfo(fmt.Sprintf("Updating synthetic test: %s", *id))

	// Build payload with only provided fields
	payload := make(map[string]interface{})
	if *name != "" {
		payload["name"] = *name
	}
	if *message != "" {
		payload["message"] = *message
	}
	if *url != "" {
		payload["config"] = map[string]interface{}{
			"request": map[string]interface{}{
				"url": *url,
			},
		}
	}

	if len(payload) == 0 {
		return fmt.Errorf("at least one of --name, --url, or --message must be provided")
	}

	// Create Datadog client
	span := obs.StartSpan("create_client")
	ddClient, err := client.NewClient()
	obs.FinishSpan(span)

	if err != nil {
		obs.LogError(fmt.Sprintf("Failed to create Datadog client: %s", err.Error()))
		return fmt.Errorf("failed to create Datadog client: %w", err)
	}

	// Update synthetic test
	span = obs.StartSpan("update_synthetic_test")
	obs.GetTracer().SetTag(span, "test_id", *id)

	start := time.Now()
	responseData, err := ddClient.UpdateSyntheticTest(*id, payload)
	apiDuration := time.Since(start).Milliseconds()
	obs.FinishSpan(span)

	if err != nil {
		obs.RecordAPICall(fmt.Sprintf("/api/v1/synthetics/tests/%s", *id), "PUT", 500, float64(apiDuration), err)
		obs.LogError(fmt.Sprintf("API call failed: %s", err.Error()))
		return fmt.Errorf("failed to update synthetic test: %w", err)
	}

	obs.RecordAPICall(fmt.Sprintf("/api/v1/synthetics/tests/%s", *id), "PUT", 200, float64(apiDuration), nil)

	// Parse response
	var response SyntheticTestAPIResponse
	if err := json.Unmarshal(responseData, &response); err != nil {
		obs.LogError(fmt.Sprintf("Failed to parse response: %s", err.Error()))
		return fmt.Errorf("failed to parse response: %w", err)
	}

	obs.GetMetrics().Count("synthetic.updated", 1)
	obs.LogInfo(fmt.Sprintf("Synthetic test updated: %s", response.PublicID))

	output := SyntheticTestOperationOutput{
		PublicID:  response.PublicID,
		Name:      response.Name,
		Status:    "updated",
		Operation: "update",
	}

	// Output
	if *jsonOut {
		jsonData, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	} else {
		fmt.Println("Synthetic test updated successfully")
		fmt.Println()
		fmt.Printf("ID: %s\n", output.PublicID)
		fmt.Printf("Name: %s\n", output.Name)
	}

	return nil
}

// runDelete deletes a synthetic test
func (c *SyntheticsCommand) runDelete(args []string) error {
	// Initialize observability
	obs, err := observability.Init("manage-synthetics-delete", "production")
	if err != nil {
		return fmt.Errorf("failed to init observability: %w", err)
	}
	defer obs.Shutdown(0)

	// Parse flags
	flags := flag.NewFlagSet("delete", flag.ExitOnError)
	id := flags.String("id", "", "Test public ID")
	jsonOut := flags.Bool("json", false, "Output as JSON")

	if err := flags.Parse(args); err != nil {
		return err
	}

	// Validate required flags
	if *id == "" {
		return fmt.Errorf("--id is required")
	}

	obs.LogInfo(fmt.Sprintf("Deleting synthetic test: %s", *id))

	// Create Datadog client
	span := obs.StartSpan("create_client")
	ddClient, err := client.NewClient()
	obs.FinishSpan(span)

	if err != nil {
		obs.LogError(fmt.Sprintf("Failed to create Datadog client: %s", err.Error()))
		return fmt.Errorf("failed to create Datadog client: %w", err)
	}

	// Delete synthetic test
	span = obs.StartSpan("delete_synthetic_test")
	obs.GetTracer().SetTag(span, "test_id", *id)

	start := time.Now()
	err = ddClient.DeleteSyntheticTest(*id)
	apiDuration := time.Since(start).Milliseconds()
	obs.FinishSpan(span)

	if err != nil {
		obs.RecordAPICall(fmt.Sprintf("/api/v1/synthetics/tests/%s", *id), "DELETE", 500, float64(apiDuration), err)
		obs.LogError(fmt.Sprintf("API call failed: %s", err.Error()))
		return fmt.Errorf("failed to delete synthetic test: %w", err)
	}

	obs.RecordAPICall(fmt.Sprintf("/api/v1/synthetics/tests/%s", *id), "DELETE", 200, float64(apiDuration), nil)
	obs.GetMetrics().Count("synthetic.deleted", 1)
	obs.LogInfo(fmt.Sprintf("Synthetic test deleted: %s", *id))

	output := SyntheticTestDeleteOutput{
		PublicID: *id,
		Status:   "deleted",
	}

	// Output
	if *jsonOut {
		jsonData, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	} else {
		fmt.Println("Synthetic test deleted successfully")
		fmt.Println()
		fmt.Printf("ID: %s\n", output.PublicID)
	}

	return nil
}

// runPause pauses a synthetic test
func (c *SyntheticsCommand) runPause(args []string) error {
	// Initialize observability
	obs, err := observability.Init("manage-synthetics-pause", "production")
	if err != nil {
		return fmt.Errorf("failed to init observability: %w", err)
	}
	defer obs.Shutdown(0)

	// Parse flags
	flags := flag.NewFlagSet("pause", flag.ExitOnError)
	id := flags.String("id", "", "Test public ID")
	jsonOut := flags.Bool("json", false, "Output as JSON")

	if err := flags.Parse(args); err != nil {
		return err
	}

	// Validate required flags
	if *id == "" {
		return fmt.Errorf("--id is required")
	}

	obs.LogInfo(fmt.Sprintf("Pausing synthetic test: %s", *id))

	// Create Datadog client
	span := obs.StartSpan("create_client")
	ddClient, err := client.NewClient()
	obs.FinishSpan(span)

	if err != nil {
		obs.LogError(fmt.Sprintf("Failed to create Datadog client: %s", err.Error()))
		return fmt.Errorf("failed to create Datadog client: %w", err)
	}

	// Pause synthetic test
	span = obs.StartSpan("pause_synthetic_test")
	obs.GetTracer().SetTag(span, "test_id", *id)

	start := time.Now()
	err = ddClient.PauseSyntheticTest(*id)
	apiDuration := time.Since(start).Milliseconds()
	obs.FinishSpan(span)

	if err != nil {
		obs.RecordAPICall(fmt.Sprintf("/api/v1/synthetics/tests/%s/status", *id), "PUT", 500, float64(apiDuration), err)
		obs.LogError(fmt.Sprintf("API call failed: %s", err.Error()))
		return fmt.Errorf("failed to pause synthetic test: %w", err)
	}

	obs.RecordAPICall(fmt.Sprintf("/api/v1/synthetics/tests/%s/status", *id), "PUT", 200, float64(apiDuration), nil)
	obs.GetMetrics().Count("synthetic.paused", 1)
	obs.LogInfo(fmt.Sprintf("Synthetic test paused: %s", *id))

	output := SyntheticTestOperationOutput{
		PublicID:  *id,
		Status:    "paused",
		Operation: "pause",
	}

	// Output
	if *jsonOut {
		jsonData, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	} else {
		fmt.Println("Synthetic test paused successfully")
		fmt.Println()
		fmt.Printf("ID: %s\n", output.PublicID)
	}

	return nil
}

// runResume resumes a synthetic test
func (c *SyntheticsCommand) runResume(args []string) error {
	// Initialize observability
	obs, err := observability.Init("manage-synthetics-resume", "production")
	if err != nil {
		return fmt.Errorf("failed to init observability: %w", err)
	}
	defer obs.Shutdown(0)

	// Parse flags
	flags := flag.NewFlagSet("resume", flag.ExitOnError)
	id := flags.String("id", "", "Test public ID")
	jsonOut := flags.Bool("json", false, "Output as JSON")

	if err := flags.Parse(args); err != nil {
		return err
	}

	// Validate required flags
	if *id == "" {
		return fmt.Errorf("--id is required")
	}

	obs.LogInfo(fmt.Sprintf("Resuming synthetic test: %s", *id))

	// Create Datadog client
	span := obs.StartSpan("create_client")
	ddClient, err := client.NewClient()
	obs.FinishSpan(span)

	if err != nil {
		obs.LogError(fmt.Sprintf("Failed to create Datadog client: %s", err.Error()))
		return fmt.Errorf("failed to create Datadog client: %w", err)
	}

	// Resume synthetic test
	span = obs.StartSpan("resume_synthetic_test")
	obs.GetTracer().SetTag(span, "test_id", *id)

	start := time.Now()
	err = ddClient.ResumeSyntheticTest(*id)
	apiDuration := time.Since(start).Milliseconds()
	obs.FinishSpan(span)

	if err != nil {
		obs.RecordAPICall(fmt.Sprintf("/api/v1/synthetics/tests/%s/status", *id), "PUT", 500, float64(apiDuration), err)
		obs.LogError(fmt.Sprintf("API call failed: %s", err.Error()))
		return fmt.Errorf("failed to resume synthetic test: %w", err)
	}

	obs.RecordAPICall(fmt.Sprintf("/api/v1/synthetics/tests/%s/status", *id), "PUT", 200, float64(apiDuration), nil)
	obs.GetMetrics().Count("synthetic.resumed", 1)
	obs.LogInfo(fmt.Sprintf("Synthetic test resumed: %s", *id))

	output := SyntheticTestOperationOutput{
		PublicID:  *id,
		Status:    "live",
		Operation: "resume",
	}

	// Output
	if *jsonOut {
		jsonData, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	} else {
		fmt.Println("Synthetic test resumed successfully")
		fmt.Println()
		fmt.Printf("ID: %s\n", output.PublicID)
	}

	return nil
}

// parseListResults parses the synthetic tests list API response
func (c *SyntheticsCommand) parseListResults(data []byte, tagsFilter string) (*SyntheticsListOutput, error) {
	var response struct {
		Tests []SyntheticTestAPIResponse `json:"tests"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	output := &SyntheticsListOutput{
		Summary: &SyntheticsSummary{
			API:     0,
			Browser: 0,
			Live:    0,
			Paused:  0,
		},
		Tests: []SyntheticTestData{},
	}

	// Parse tags filter if provided
	var filterTags []string
	if tagsFilter != "" {
		filterTags = strings.Split(tagsFilter, ",")
		for i := range filterTags {
			filterTags[i] = strings.TrimSpace(filterTags[i])
		}
	}

	// Process each test
	for _, t := range response.Tests {
		// Apply tags filter
		if len(filterTags) > 0 {
			hasAllTags := true
			for _, filterTag := range filterTags {
				found := false
				for _, tag := range t.Tags {
					if tag == filterTag {
						found = true
						break
					}
				}
				if !found {
					hasAllTags = false
					break
				}
			}
			if !hasAllTags {
				continue
			}
		}

		test := SyntheticTestData{
			PublicID:  t.PublicID,
			Name:      t.Name,
			Type:      t.Type,
			Status:    t.Status,
			Tags:      t.Tags,
			Locations: t.Locations,
			Message:   t.Message,
			URL:       t.Config.Request.URL,
		}
		output.Tests = append(output.Tests, test)

		// Update summary
		switch strings.ToLower(t.Type) {
		case "api":
			output.Summary.API++
		case "browser":
			output.Summary.Browser++
		}

		switch strings.ToLower(t.Status) {
		case "live":
			output.Summary.Live++
		case "paused":
			output.Summary.Paused++
		}
	}

	output.Total = len(output.Tests)

	// Sort tests by name
	sort.Slice(output.Tests, func(i, j int) bool {
		return output.Tests[i].Name < output.Tests[j].Name
	})

	return output, nil
}

// parseResults parses the synthetic test results API response
func (c *SyntheticsCommand) parseResults(data []byte, testID, testName string) (*SyntheticResultsOutput, error) {
	var response struct {
		Results []struct {
			ResultID  string  `json:"result_id"`
			CheckTime int64   `json:"check_time"`
			Result    struct {
				Passed      bool    `json:"passed"`
				Unhealthy   bool    `json:"unhealthy"`
				Duration    float64 `json:"duration"`
				ErrorMessage string  `json:"error_message,omitempty"`
			} `json:"result"`
			Location string `json:"location"`
		} `json:"results"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	output := &SyntheticResultsOutput{
		TestID:       testID,
		TestName:     testName,
		TotalResults: len(response.Results),
		Results:      []SyntheticResultData{},
	}

	if output.TotalResults == 0 {
		return output, nil
	}

	passed := 0
	failed := 0
	totalDuration := 0.0

	// Process each result
	for _, r := range response.Results {
		status := 0
		if r.Result.Passed {
			status = 0
			passed++
		} else {
			status = 1
			failed++
		}

		result := SyntheticResultData{
			ResultID:  r.ResultID,
			CheckTime: time.Unix(r.CheckTime/1000, 0).Format(time.RFC3339),
			Status:    status,
			Duration:  r.Result.Duration,
			Location:  r.Location,
			Error:     r.Result.ErrorMessage,
		}
		output.Results = append(output.Results, result)
		totalDuration += r.Result.Duration
	}

	// Calculate statistics
	if output.TotalResults > 0 {
		output.PassRate = (float64(passed) / float64(output.TotalResults)) * 100
		output.FailRate = (float64(failed) / float64(output.TotalResults)) * 100
		output.AvgDuration = totalDuration / float64(output.TotalResults)
	}

	// Sort results by check time (most recent first)
	sort.Slice(output.Results, func(i, j int) bool {
		return output.Results[i].CheckTime > output.Results[j].CheckTime
	})

	return output, nil
}

// printListFormatted prints the synthetic tests list in conversational format
func (c *SyntheticsCommand) printListFormatted(output *SyntheticsListOutput) {
	fmt.Println("Synthetic Tests Summary")
	fmt.Println()
	fmt.Printf("Total tests: %d\n", output.Total)
	fmt.Printf("  API tests: %d\n", output.Summary.API)
	fmt.Printf("  Browser tests: %d\n", output.Summary.Browser)
	fmt.Printf("  Live: %d\n", output.Summary.Live)
	fmt.Printf("  Paused: %d\n", output.Summary.Paused)

	if len(output.Tests) > 0 {
		fmt.Println()
		fmt.Println("Tests:")
		for i, t := range output.Tests {
			if i >= 10 {
				fmt.Printf("\n... and %d more tests (use --json to see all)\n", len(output.Tests)-10)
				break
			}
			statusSymbol := "✓"
			if t.Status == "paused" {
				statusSymbol = "⏸"
			}

			fmt.Printf("  %s [%s] %s\n", statusSymbol, t.PublicID, t.Name)
			fmt.Printf("      Type: %s | Status: %s", t.Type, t.Status)
			if t.URL != "" {
				fmt.Printf(" | URL: %s", t.URL)
			}
			fmt.Println()
		}
	}
}

// printTestDetails prints synthetic test details in conversational format
func (c *SyntheticsCommand) printTestDetails(test SyntheticTestAPIResponse) {
	fmt.Println("Synthetic Test Details")
	fmt.Println()
	fmt.Printf("ID: %s\n", test.PublicID)
	fmt.Printf("Name: %s\n", test.Name)
	fmt.Printf("Type: %s\n", test.Type)
	fmt.Printf("Status: %s\n", test.Status)

	if test.Config.Request.URL != "" {
		fmt.Printf("URL: %s\n", test.Config.Request.URL)
	}

	if len(test.Locations) > 0 {
		fmt.Printf("Locations: %s\n", strings.Join(test.Locations, ", "))
	}

	if test.Options.TickEvery > 0 {
		fmt.Printf("Frequency: every %d seconds\n", test.Options.TickEvery)
	}

	if len(test.Tags) > 0 {
		fmt.Printf("Tags: %s\n", strings.Join(test.Tags, ", "))
	}

	if test.Message != "" {
		fmt.Println()
		fmt.Printf("Message: %s\n", test.Message)
	}

	if len(test.Config.Assertions) > 0 {
		fmt.Println()
		fmt.Printf("Assertions: %d configured\n", len(test.Config.Assertions))
	}

	fmt.Println()
	fmt.Printf("Created: %s\n", test.CreatedAt)
	fmt.Printf("Modified: %s\n", test.ModifiedAt)
	if test.MonitorID > 0 {
		fmt.Printf("Monitor ID: %d\n", test.MonitorID)
	}
}

// printResultsFormatted prints synthetic test results in conversational format
func (c *SyntheticsCommand) printResultsFormatted(output *SyntheticResultsOutput, hours int) {
	fmt.Println("Synthetic Test Results")
	fmt.Println()
	fmt.Printf("Test: %s\n", output.TestName)
	fmt.Printf("Test ID: %s\n", output.TestID)
	fmt.Printf("Period: Last %d hours\n", hours)
	fmt.Println()
	fmt.Printf("Total results: %d\n", output.TotalResults)
	fmt.Printf("Pass rate: %.2f%%\n", output.PassRate)
	fmt.Printf("Fail rate: %.2f%%\n", output.FailRate)
	fmt.Printf("Average duration: %.2f ms\n", output.AvgDuration)

	if len(output.Results) > 0 {
		fmt.Println()
		fmt.Println("Recent Results:")
		fmt.Printf("%-25s %-12s %-10s %-15s\n", "Time", "Status", "Duration", "Location")
		fmt.Println(strings.Repeat("-", 70))
		for i, r := range output.Results {
			if i >= 10 {
				fmt.Printf("\n... and %d more results (use --json to see all)\n", len(output.Results)-10)
				break
			}
			statusStr := "PASS"
			statusSymbol := "✓"
			if r.Status != 0 {
				statusStr = "FAIL"
				statusSymbol = "✗"
			}

			checkTime, _ := time.Parse(time.RFC3339, r.CheckTime)
			timeStr := checkTime.Format("2006-01-02 15:04:05")

			fmt.Printf("%-25s %s %-10s %-10.2f ms %-15s\n",
				timeStr,
				statusSymbol,
				statusStr,
				r.Duration,
				r.Location,
			)

			if r.Error != "" {
				fmt.Printf("    Error: %s\n", r.Error)
			}
		}
	}
}

// Help prints the help message
func (c *SyntheticsCommand) Help() {
	fmt.Println("Usage: dd synthetics <subcommand> [options]")
	fmt.Println()
	fmt.Println("Manage Datadog synthetic tests - list, get, results, create, update, delete, pause, and resume tests.")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  list       List synthetic tests")
	fmt.Println("  get        Get a specific synthetic test")
	fmt.Println("  results    Get test results and performance metrics")
	fmt.Println("  create     Create a new synthetic test")
	fmt.Println("  update     Update a synthetic test")
	fmt.Println("  delete     Delete a synthetic test")
	fmt.Println("  pause      Pause a synthetic test")
	fmt.Println("  resume     Resume a synthetic test")
	fmt.Println()
	fmt.Println("List Options:")
	fmt.Println("  --type string")
	fmt.Println("        Filter by test type (api/browser)")
	fmt.Println("  --tags string")
	fmt.Println("        Filter by tags (comma-separated)")
	fmt.Println("  --json")
	fmt.Println("        Output as JSON")
	fmt.Println()
	fmt.Println("Get Options:")
	fmt.Println("  --id string")
	fmt.Println("        Test public ID (required)")
	fmt.Println("  --json")
	fmt.Println("        Output as JSON")
	fmt.Println()
	fmt.Println("Results Options:")
	fmt.Println("  --id string")
	fmt.Println("        Test public ID (required)")
	fmt.Println("  --hours int")
	fmt.Println("        Hours to look back (default: 24)")
	fmt.Println("  --json")
	fmt.Println("        Output as JSON")
	fmt.Println()
	fmt.Println("Create Options:")
	fmt.Println("  --name string")
	fmt.Println("        Test name (required)")
	fmt.Println("  --url string")
	fmt.Println("        URL to test (required)")
	fmt.Println("  --type string")
	fmt.Println("        Test type: api or browser (default: api)")
	fmt.Println("  --message string")
	fmt.Println("        Alert message")
	fmt.Println("  --locations string")
	fmt.Println("        Test locations (comma-separated, default: aws:us-east-1)")
	fmt.Println("  --json")
	fmt.Println("        Output as JSON")
	fmt.Println()
	fmt.Println("Update Options:")
	fmt.Println("  --id string")
	fmt.Println("        Test public ID (required)")
	fmt.Println("  --name string")
	fmt.Println("        New test name")
	fmt.Println("  --url string")
	fmt.Println("        New URL to test")
	fmt.Println("  --message string")
	fmt.Println("        New alert message")
	fmt.Println("  --json")
	fmt.Println("        Output as JSON")
	fmt.Println()
	fmt.Println("Delete Options:")
	fmt.Println("  --id string")
	fmt.Println("        Test public ID (required)")
	fmt.Println("  --json")
	fmt.Println("        Output as JSON")
	fmt.Println()
	fmt.Println("Pause Options:")
	fmt.Println("  --id string")
	fmt.Println("        Test public ID (required)")
	fmt.Println("  --json")
	fmt.Println("        Output as JSON")
	fmt.Println()
	fmt.Println("Resume Options:")
	fmt.Println("  --id string")
	fmt.Println("        Test public ID (required)")
	fmt.Println("  --json")
	fmt.Println("        Output as JSON")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # List all synthetic tests")
	fmt.Println("  dd synthetics list")
	fmt.Println()
	fmt.Println("  # List only API tests")
	fmt.Println("  dd synthetics list --type api")
	fmt.Println()
	fmt.Println("  # Get test details")
	fmt.Println("  dd synthetics get --id abc-123-def")
	fmt.Println()
	fmt.Println("  # Get test results from last 48 hours")
	fmt.Println("  dd synthetics results --id abc-123-def --hours 48")
	fmt.Println()
	fmt.Println("  # Create API test")
	fmt.Println("  dd synthetics create --name \"API Health Check\" \\")
	fmt.Println("    --url \"https://api.example.com/health\" \\")
	fmt.Println("    --type api \\")
	fmt.Println("    --message \"API health check failed @slack-ops\"")
	fmt.Println()
	fmt.Println("  # Update test URL")
	fmt.Println("  dd synthetics update --id abc-123-def \\")
	fmt.Println("    --url \"https://api.example.com/v2/health\"")
	fmt.Println()
	fmt.Println("  # Pause test during maintenance")
	fmt.Println("  dd synthetics pause --id abc-123-def")
	fmt.Println()
	fmt.Println("  # Resume test after maintenance")
	fmt.Println("  dd synthetics resume --id abc-123-def")
	fmt.Println()
	fmt.Println("  # Delete test")
	fmt.Println("  dd synthetics delete --id abc-123-def")
}
