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

// WorkflowsCommand manages Datadog workflow automation
type WorkflowsCommand struct {
	flags *flag.FlagSet
}

// WorkflowData represents a single workflow
type WorkflowData struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	ModifiedAt  time.Time              `json:"modified_at"`
	Tags        []string               `json:"tags,omitempty"`
	Status      string                 `json:"status,omitempty"`
	LastRunAt   *time.Time             `json:"last_run_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// WorkflowExecutionData represents a workflow execution
type WorkflowExecutionData struct {
	ID           string                 `json:"id"`
	WorkflowID   string                 `json:"workflow_id"`
	Status       string                 `json:"status"`
	StartedAt    time.Time              `json:"started_at"`
	FinishedAt   *time.Time             `json:"finished_at,omitempty"`
	Duration     *int64                 `json:"duration_ms,omitempty"`
	Error        string                 `json:"error,omitempty"`
	OutputParams map[string]interface{} `json:"output_params,omitempty"`
}

// WorkflowsOutput represents the structured output
type WorkflowsOutput struct {
	Status         string                 `json:"status"`
	TotalWorkflows int                    `json:"total_workflows,omitempty"`
	Workflows      []WorkflowData         `json:"workflows,omitempty"`
	Workflow       *WorkflowData          `json:"workflow,omitempty"`
	Execution      *WorkflowExecutionData `json:"execution,omitempty"`
	Message        string                 `json:"message,omitempty"`
	RawData        map[string]interface{} `json:"raw_data,omitempty"`
}

// WorkflowListAPIResponse represents the list API response structure
type WorkflowListAPIResponse struct {
	Data []struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		Attributes struct {
			Name        string                 `json:"name"`
			Description string                 `json:"description"`
			CreatedAt   string                 `json:"created_at"`
			ModifiedAt  string                 `json:"modified_at"`
			Tags        []string               `json:"tags"`
			LastRunAt   *string                `json:"last_run_at"`
			Metadata    map[string]interface{} `json:"metadata"`
		} `json:"attributes"`
	} `json:"data"`
}

// WorkflowAPIResponse represents a single workflow API response
type WorkflowAPIResponse struct {
	Data struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		Attributes struct {
			Name        string                 `json:"name"`
			Description string                 `json:"description"`
			CreatedAt   string                 `json:"created_at"`
			ModifiedAt  string                 `json:"modified_at"`
			Tags        []string               `json:"tags"`
			LastRunAt   *string                `json:"last_run_at"`
			Metadata    map[string]interface{} `json:"metadata"`
		} `json:"attributes"`
	} `json:"data"`
}

// WorkflowExecutionAPIResponse represents the execution API response
type WorkflowExecutionAPIResponse struct {
	Data struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		Attributes struct {
			WorkflowID   string                 `json:"workflow_id"`
			Status       string                 `json:"status"`
			StartedAt    string                 `json:"started_at"`
			FinishedAt   *string                `json:"finished_at"`
			Error        string                 `json:"error"`
			OutputParams map[string]interface{} `json:"output_params"`
		} `json:"attributes"`
	} `json:"data"`
}

// NewWorkflowsCommand creates a new workflows command
func NewWorkflowsCommand() *WorkflowsCommand {
	cmd := &WorkflowsCommand{
		flags: flag.NewFlagSet("workflows", flag.ContinueOnError),
	}

	return cmd
}

// Name returns the command name
func (c *WorkflowsCommand) Name() string {
	return "workflows"
}

// Description returns the command description
func (c *WorkflowsCommand) Description() string {
	return "Manage Datadog workflow automation - list, get, execute, create, update, delete workflows"
}

// Run executes the workflows command
func (c *WorkflowsCommand) Run(args []string) error {
	// Initialize observability
	obs, err := observability.Init("manage-workflows", "production")
	if err != nil {
		return fmt.Errorf("failed to init observability: %w", err)
	}
	defer obs.Shutdown(0)

	if len(args) == 0 {
		c.Help()
		return fmt.Errorf("subcommand required: list, get, execute, create, update, or delete")
	}

	subcommand := args[0]
	subArgs := args[1:]

	obs.LogInfo(fmt.Sprintf("Executing workflows subcommand: %s", subcommand))

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
	case "get":
		return c.runGet(subArgs, ddClient, obs)
	case "execute":
		return c.runExecute(subArgs, ddClient, obs)
	case "create":
		return c.runCreate(subArgs, ddClient, obs)
	case "update":
		return c.runUpdate(subArgs, ddClient, obs)
	case "delete":
		return c.runDelete(subArgs, ddClient, obs)
	default:
		c.Help()
		return fmt.Errorf("unknown subcommand: %s", subcommand)
	}
}

// runList executes the list subcommand
func (c *WorkflowsCommand) runList(args []string, ddClient *client.Client, obs *observability.Observability) error {
	listFlags := flag.NewFlagSet("list", flag.ExitOnError)
	tags := listFlags.String("tags", "", "Filter by tags (comma-separated)")
	limit := listFlags.Int("limit", 100, "Max workflows to return")
	jsonOut := listFlags.Bool("json", false, "Output as JSON")

	if err := listFlags.Parse(args); err != nil {
		return err
	}

	obs.LogInfo("Listing workflows")

	// Query workflows
	span := obs.StartSpan("list_workflows")
	if *tags != "" {
		obs.GetTracer().SetTag(span, "tags", *tags)
	}

	start := time.Now()
	responseData, err := ddClient.ListWorkflows()
	apiDuration := time.Since(start).Milliseconds()
	obs.FinishSpan(span)

	if err != nil {
		obs.RecordAPICall("/api/v2/workflows", "GET", 500, float64(apiDuration), err)
		obs.LogError(fmt.Sprintf("API call failed: %s", err.Error()))
		return fmt.Errorf("failed to list workflows: %w", err)
	}

	obs.RecordAPICall("/api/v2/workflows", "GET", 200, float64(apiDuration), nil)

	// Parse results
	span = obs.StartSpan("process_results")
	output, err := c.parseListResults(responseData, *tags, *limit)
	obs.FinishSpan(span)

	if err != nil {
		obs.LogError(fmt.Sprintf("Failed to parse results: %s", err.Error()))
		return fmt.Errorf("failed to parse results: %w", err)
	}

	// Record metrics
	obs.GetMetrics().Gauge("workflows.total", float64(output.TotalWorkflows))

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

	obs.LogInfo(fmt.Sprintf("Listed %d workflows", output.TotalWorkflows))
	return nil
}

// runGet executes the get subcommand
func (c *WorkflowsCommand) runGet(args []string, ddClient *client.Client, obs *observability.Observability) error {
	getFlags := flag.NewFlagSet("get", flag.ExitOnError)
	id := getFlags.String("id", "", "Workflow ID (required)")
	jsonOut := getFlags.Bool("json", false, "Output as JSON")

	if err := getFlags.Parse(args); err != nil {
		return err
	}

	if *id == "" {
		return fmt.Errorf("--id is required")
	}

	obs.LogInfo(fmt.Sprintf("Getting workflow: %s", *id))

	// Get workflow
	span := obs.StartSpan("get_workflow")
	obs.GetTracer().SetTag(span, "workflow_id", *id)

	start := time.Now()
	responseData, err := ddClient.GetWorkflow(*id)
	apiDuration := time.Since(start).Milliseconds()
	obs.FinishSpan(span)

	if err != nil {
		obs.RecordAPICall(fmt.Sprintf("/api/v2/workflows/%s", *id), "GET", 500, float64(apiDuration), err)
		obs.LogError(fmt.Sprintf("API call failed: %s", err.Error()))
		return fmt.Errorf("failed to get workflow: %w", err)
	}

	obs.RecordAPICall(fmt.Sprintf("/api/v2/workflows/%s", *id), "GET", 200, float64(apiDuration), nil)

	// Parse result
	span = obs.StartSpan("parse_result")
	output, err := c.parseGetResult(responseData)
	obs.FinishSpan(span)

	if err != nil {
		obs.LogError(fmt.Sprintf("Failed to parse result: %s", err.Error()))
		return fmt.Errorf("failed to parse result: %w", err)
	}

	obs.GetMetrics().Count("workflow.fetched", 1)

	// Output
	if *jsonOut {
		jsonData, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	} else {
		c.printGetFormatted(output)
	}

	obs.LogInfo(fmt.Sprintf("Workflow retrieved: %s", output.Workflow.ID))
	return nil
}

// runExecute executes the execute subcommand
func (c *WorkflowsCommand) runExecute(args []string, ddClient *client.Client, obs *observability.Observability) error {
	execFlags := flag.NewFlagSet("execute", flag.ExitOnError)
	id := execFlags.String("id", "", "Workflow ID (required)")
	params := execFlags.String("params", "", "Input parameters as JSON (optional)")
	wait := execFlags.Bool("wait", false, "Wait for execution to complete")
	timeout := execFlags.Int("timeout", 300, "Timeout in seconds when waiting")
	jsonOut := execFlags.Bool("json", false, "Output as JSON")

	if err := execFlags.Parse(args); err != nil {
		return err
	}

	if *id == "" {
		return fmt.Errorf("--id is required")
	}

	obs.LogInfo(fmt.Sprintf("Executing workflow: %s", *id))

	// Parse parameters
	var inputParams map[string]interface{}
	if *params != "" {
		if err := json.Unmarshal([]byte(*params), &inputParams); err != nil {
			return fmt.Errorf("failed to parse params JSON: %w", err)
		}
	}

	// Execute workflow
	span := obs.StartSpan("execute_workflow")
	obs.GetTracer().SetTag(span, "workflow_id", *id)
	obs.GetTracer().SetTag(span, "wait", fmt.Sprintf("%t", *wait))

	start := time.Now()
	responseData, err := ddClient.ExecuteWorkflow(*id, inputParams)
	apiDuration := time.Since(start).Milliseconds()
	obs.FinishSpan(span)

	if err != nil {
		obs.RecordAPICall(fmt.Sprintf("/api/v2/workflows/%s/execute", *id), "POST", 500, float64(apiDuration), err)
		obs.LogError(fmt.Sprintf("API call failed: %s", err.Error()))
		return fmt.Errorf("failed to execute workflow: %w", err)
	}

	obs.RecordAPICall(fmt.Sprintf("/api/v2/workflows/%s/execute", *id), "POST", 200, float64(apiDuration), nil)

	// Parse initial result
	span = obs.StartSpan("parse_result")
	output, err := c.parseExecuteResult(responseData)
	obs.FinishSpan(span)

	if err != nil {
		obs.LogError(fmt.Sprintf("Failed to parse result: %s", err.Error()))
		return fmt.Errorf("failed to parse result: %w", err)
	}

	obs.GetMetrics().Count("workflow.executed", 1)

	// If wait flag is set, poll for completion
	if *wait && output.Execution != nil {
		executionID := output.Execution.ID
		obs.LogInfo(fmt.Sprintf("Waiting for execution to complete: %s", executionID))

		completedExecution, err := c.waitForExecution(ddClient, obs, executionID, *timeout)
		if err != nil {
			obs.LogError(fmt.Sprintf("Failed waiting for execution: %s", err.Error()))
			return fmt.Errorf("failed waiting for execution: %w", err)
		}

		output.Execution = completedExecution
		output.Status = completedExecution.Status
	}

	// Output
	if *jsonOut {
		jsonData, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	} else {
		c.printExecuteFormatted(output)
	}

	obs.LogInfo(fmt.Sprintf("Workflow executed: %s", output.Execution.ID))
	return nil
}

// runCreate executes the create subcommand
func (c *WorkflowsCommand) runCreate(args []string, ddClient *client.Client, obs *observability.Observability) error {
	createFlags := flag.NewFlagSet("create", flag.ExitOnError)
	name := createFlags.String("name", "", "Workflow name (required)")
	description := createFlags.String("description", "", "Workflow description")
	definition := createFlags.String("definition", "", "Workflow definition as JSON (required)")
	tags := createFlags.String("tags", "", "Tags (comma-separated)")
	jsonOut := createFlags.Bool("json", false, "Output as JSON")

	if err := createFlags.Parse(args); err != nil {
		return err
	}

	if *name == "" {
		return fmt.Errorf("--name is required")
	}
	if *definition == "" {
		return fmt.Errorf("--definition is required")
	}

	obs.LogInfo(fmt.Sprintf("Creating workflow: %s", *name))

	// Parse definition
	var workflowDef map[string]interface{}
	if err := json.Unmarshal([]byte(*definition), &workflowDef); err != nil {
		return fmt.Errorf("failed to parse definition JSON: %w", err)
	}

	// Build payload
	payload := map[string]interface{}{
		"name":       *name,
		"definition": workflowDef,
	}

	if *description != "" {
		payload["description"] = *description
	}

	if *tags != "" {
		tagsList := []string{}
		for _, tag := range strings.Split(*tags, ",") {
			tagsList = append(tagsList, strings.TrimSpace(tag))
		}
		payload["tags"] = tagsList
	}

	// Create workflow
	span := obs.StartSpan("create_workflow")
	obs.GetTracer().SetTag(span, "name", *name)

	start := time.Now()
	responseData, err := ddClient.CreateWorkflow(payload)
	apiDuration := time.Since(start).Milliseconds()
	obs.FinishSpan(span)

	if err != nil {
		obs.RecordAPICall("/api/v2/workflows", "POST", 500, float64(apiDuration), err)
		obs.LogError(fmt.Sprintf("API call failed: %s", err.Error()))
		return fmt.Errorf("failed to create workflow: %w", err)
	}

	obs.RecordAPICall("/api/v2/workflows", "POST", 200, float64(apiDuration), nil)

	// Parse result
	span = obs.StartSpan("parse_result")
	output, err := c.parseGetResult(responseData)
	obs.FinishSpan(span)

	if err != nil {
		obs.LogError(fmt.Sprintf("Failed to parse result: %s", err.Error()))
		return fmt.Errorf("failed to parse result: %w", err)
	}

	output.Status = "created"
	output.Message = "Workflow created successfully"

	obs.GetMetrics().Count("workflow.created", 1)

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

	obs.LogInfo(fmt.Sprintf("Workflow created: %s", output.Workflow.ID))
	return nil
}

// runUpdate executes the update subcommand
func (c *WorkflowsCommand) runUpdate(args []string, ddClient *client.Client, obs *observability.Observability) error {
	updateFlags := flag.NewFlagSet("update", flag.ExitOnError)
	id := updateFlags.String("id", "", "Workflow ID (required)")
	name := updateFlags.String("name", "", "New workflow name")
	description := updateFlags.String("description", "", "New workflow description")
	definition := updateFlags.String("definition", "", "New workflow definition as JSON")
	jsonOut := updateFlags.Bool("json", false, "Output as JSON")

	if err := updateFlags.Parse(args); err != nil {
		return err
	}

	if *id == "" {
		return fmt.Errorf("--id is required")
	}

	obs.LogInfo(fmt.Sprintf("Updating workflow: %s", *id))

	// Build payload
	payload := make(map[string]interface{})

	if *name != "" {
		payload["name"] = *name
	}

	if *description != "" {
		payload["description"] = *description
	}

	if *definition != "" {
		var workflowDef map[string]interface{}
		if err := json.Unmarshal([]byte(*definition), &workflowDef); err != nil {
			return fmt.Errorf("failed to parse definition JSON: %w", err)
		}
		payload["definition"] = workflowDef
	}

	if len(payload) == 0 {
		return fmt.Errorf("at least one of --name, --description, or --definition is required")
	}

	// Update workflow
	span := obs.StartSpan("update_workflow")
	obs.GetTracer().SetTag(span, "workflow_id", *id)

	start := time.Now()
	responseData, err := ddClient.UpdateWorkflow(*id, payload)
	apiDuration := time.Since(start).Milliseconds()
	obs.FinishSpan(span)

	if err != nil {
		obs.RecordAPICall(fmt.Sprintf("/api/v2/workflows/%s", *id), "PATCH", 500, float64(apiDuration), err)
		obs.LogError(fmt.Sprintf("API call failed: %s", err.Error()))
		return fmt.Errorf("failed to update workflow: %w", err)
	}

	obs.RecordAPICall(fmt.Sprintf("/api/v2/workflows/%s", *id), "PATCH", 200, float64(apiDuration), nil)

	// Parse result
	span = obs.StartSpan("parse_result")
	output, err := c.parseGetResult(responseData)
	obs.FinishSpan(span)

	if err != nil {
		obs.LogError(fmt.Sprintf("Failed to parse result: %s", err.Error()))
		return fmt.Errorf("failed to parse result: %w", err)
	}

	output.Status = "updated"
	output.Message = "Workflow updated successfully"

	obs.GetMetrics().Count("workflow.updated", 1)

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

	obs.LogInfo(fmt.Sprintf("Workflow updated: %s", output.Workflow.ID))
	return nil
}

// runDelete executes the delete subcommand
func (c *WorkflowsCommand) runDelete(args []string, ddClient *client.Client, obs *observability.Observability) error {
	deleteFlags := flag.NewFlagSet("delete", flag.ExitOnError)
	id := deleteFlags.String("id", "", "Workflow ID (required)")
	jsonOut := deleteFlags.Bool("json", false, "Output as JSON")

	if err := deleteFlags.Parse(args); err != nil {
		return err
	}

	if *id == "" {
		return fmt.Errorf("--id is required")
	}

	obs.LogInfo(fmt.Sprintf("Deleting workflow: %s", *id))

	// Delete workflow
	span := obs.StartSpan("delete_workflow")
	obs.GetTracer().SetTag(span, "workflow_id", *id)

	start := time.Now()
	err := ddClient.DeleteWorkflow(*id)
	apiDuration := time.Since(start).Milliseconds()
	obs.FinishSpan(span)

	if err != nil {
		obs.RecordAPICall(fmt.Sprintf("/api/v2/workflows/%s", *id), "DELETE", 500, float64(apiDuration), err)
		obs.LogError(fmt.Sprintf("API call failed: %s", err.Error()))
		return fmt.Errorf("failed to delete workflow: %w", err)
	}

	obs.RecordAPICall(fmt.Sprintf("/api/v2/workflows/%s", *id), "DELETE", 204, float64(apiDuration), nil)

	obs.GetMetrics().Count("workflow.deleted", 1)

	output := &WorkflowsOutput{
		Status:  "deleted",
		Message: fmt.Sprintf("Workflow %s deleted successfully", *id),
		Workflow: &WorkflowData{
			ID: *id,
		},
	}

	// Output
	if *jsonOut {
		jsonData, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	} else {
		c.printDeleteFormatted(output)
	}

	obs.LogInfo(fmt.Sprintf("Workflow deleted: %s", *id))
	return nil
}

// waitForExecution polls for workflow execution completion
func (c *WorkflowsCommand) waitForExecution(ddClient *client.Client, obs *observability.Observability, executionID string, timeoutSec int) (*WorkflowExecutionData, error) {
	timeout := time.After(time.Duration(timeoutSec) * time.Second)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return nil, fmt.Errorf("timeout waiting for execution to complete after %d seconds", timeoutSec)
		case <-ticker.C:
			// Query execution status
			span := obs.StartSpan("check_execution_status")
			obs.GetTracer().SetTag(span, "execution_id", executionID)

			// Note: Assuming there's a GetWorkflowExecution method
			responseData, err := ddClient.GetWorkflowExecution(executionID)
			obs.FinishSpan(span)

			if err != nil {
				obs.LogError(fmt.Sprintf("Failed to check execution status: %s", err.Error()))
				continue
			}

			// Parse execution status
			var response WorkflowExecutionAPIResponse
			if err := json.Unmarshal(responseData, &response); err != nil {
				obs.LogError(fmt.Sprintf("Failed to parse execution response: %s", err.Error()))
				continue
			}

			execution := c.parseExecutionData(response)

			// Check if execution is complete
			status := strings.ToLower(execution.Status)
			if status == "completed" || status == "success" || status == "failed" || status == "error" {
				return execution, nil
			}

			obs.LogInfo(fmt.Sprintf("Execution %s status: %s", executionID, execution.Status))
		}
	}
}

// parseListResults parses the list workflows API response
func (c *WorkflowsCommand) parseListResults(data []byte, tagsFilter string, limit int) (*WorkflowsOutput, error) {
	var response WorkflowListAPIResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	output := &WorkflowsOutput{
		Status:    "ok",
		Workflows: []WorkflowData{},
	}

	// Parse tags filter
	var filterTags []string
	if tagsFilter != "" {
		for _, tag := range strings.Split(tagsFilter, ",") {
			filterTags = append(filterTags, strings.TrimSpace(tag))
		}
	}

	count := 0
	for _, item := range response.Data {
		// Apply tag filter
		if len(filterTags) > 0 {
			hasAllTags := true
			for _, filterTag := range filterTags {
				found := false
				for _, tag := range item.Attributes.Tags {
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

		// Apply limit
		if limit > 0 && count >= limit {
			break
		}

		createdAt, _ := time.Parse(time.RFC3339, item.Attributes.CreatedAt)
		modifiedAt, _ := time.Parse(time.RFC3339, item.Attributes.ModifiedAt)

		var lastRunAt *time.Time
		if item.Attributes.LastRunAt != nil && *item.Attributes.LastRunAt != "" {
			t, _ := time.Parse(time.RFC3339, *item.Attributes.LastRunAt)
			lastRunAt = &t
		}

		workflow := WorkflowData{
			ID:          item.ID,
			Name:        item.Attributes.Name,
			Description: item.Attributes.Description,
			CreatedAt:   createdAt,
			ModifiedAt:  modifiedAt,
			Tags:        item.Attributes.Tags,
			LastRunAt:   lastRunAt,
			Metadata:    item.Attributes.Metadata,
		}

		output.Workflows = append(output.Workflows, workflow)
		count++
	}

	output.TotalWorkflows = len(output.Workflows)

	return output, nil
}

// parseGetResult parses a single workflow API response
func (c *WorkflowsCommand) parseGetResult(data []byte) (*WorkflowsOutput, error) {
	var response WorkflowAPIResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	createdAt, _ := time.Parse(time.RFC3339, response.Data.Attributes.CreatedAt)
	modifiedAt, _ := time.Parse(time.RFC3339, response.Data.Attributes.ModifiedAt)

	var lastRunAt *time.Time
	if response.Data.Attributes.LastRunAt != nil && *response.Data.Attributes.LastRunAt != "" {
		t, _ := time.Parse(time.RFC3339, *response.Data.Attributes.LastRunAt)
		lastRunAt = &t
	}

	workflow := &WorkflowData{
		ID:          response.Data.ID,
		Name:        response.Data.Attributes.Name,
		Description: response.Data.Attributes.Description,
		CreatedAt:   createdAt,
		ModifiedAt:  modifiedAt,
		Tags:        response.Data.Attributes.Tags,
		LastRunAt:   lastRunAt,
		Metadata:    response.Data.Attributes.Metadata,
	}

	return &WorkflowsOutput{
		Status:   "ok",
		Workflow: workflow,
	}, nil
}

// parseExecuteResult parses the workflow execution API response
func (c *WorkflowsCommand) parseExecuteResult(data []byte) (*WorkflowsOutput, error) {
	var response WorkflowExecutionAPIResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	execution := c.parseExecutionData(response)

	return &WorkflowsOutput{
		Status:    execution.Status,
		Execution: execution,
		Message:   fmt.Sprintf("Workflow execution started: %s", execution.ID),
	}, nil
}

// parseExecutionData converts API response to execution data
func (c *WorkflowsCommand) parseExecutionData(response WorkflowExecutionAPIResponse) *WorkflowExecutionData {
	startedAt, _ := time.Parse(time.RFC3339, response.Data.Attributes.StartedAt)

	var finishedAt *time.Time
	if response.Data.Attributes.FinishedAt != nil && *response.Data.Attributes.FinishedAt != "" {
		t, _ := time.Parse(time.RFC3339, *response.Data.Attributes.FinishedAt)
		finishedAt = &t
	}

	var duration *int64
	if finishedAt != nil {
		d := finishedAt.Sub(startedAt).Milliseconds()
		duration = &d
	}

	return &WorkflowExecutionData{
		ID:           response.Data.ID,
		WorkflowID:   response.Data.Attributes.WorkflowID,
		Status:       response.Data.Attributes.Status,
		StartedAt:    startedAt,
		FinishedAt:   finishedAt,
		Duration:     duration,
		Error:        response.Data.Attributes.Error,
		OutputParams: response.Data.Attributes.OutputParams,
	}
}

// printListFormatted prints the list results in a conversational format
func (c *WorkflowsCommand) printListFormatted(output *WorkflowsOutput) {
	fmt.Println("Workflows Summary")
	fmt.Println()
	fmt.Printf("Total workflows: %d\n", output.TotalWorkflows)

	if len(output.Workflows) > 0 {
		fmt.Println()
		fmt.Println("Workflows:")

		count := 0
		for _, wf := range output.Workflows {
			if count >= 20 {
				fmt.Printf("\n... and %d more workflows (use --json to see all)\n", len(output.Workflows)-20)
				break
			}

			fmt.Printf("  [%s] %s\n", wf.ID, wf.Name)
			if wf.Description != "" {
				fmt.Printf("      %s\n", wf.Description)
			}
			if wf.LastRunAt != nil {
				fmt.Printf("      Last run: %s\n", wf.LastRunAt.Format(time.RFC3339))
			}
			if len(wf.Tags) > 0 {
				fmt.Printf("      Tags: %s\n", strings.Join(wf.Tags, ", "))
			}
			count++
		}
	}
}

// printGetFormatted prints the get result in a conversational format
func (c *WorkflowsCommand) printGetFormatted(output *WorkflowsOutput) {
	fmt.Println("Workflow Details")
	fmt.Println()
	fmt.Printf("ID: %s\n", output.Workflow.ID)
	fmt.Printf("Name: %s\n", output.Workflow.Name)
	if output.Workflow.Description != "" {
		fmt.Printf("Description: %s\n", output.Workflow.Description)
	}
	fmt.Printf("Created: %s\n", output.Workflow.CreatedAt.Format(time.RFC3339))
	fmt.Printf("Modified: %s\n", output.Workflow.ModifiedAt.Format(time.RFC3339))
	if output.Workflow.LastRunAt != nil {
		fmt.Printf("Last Run: %s\n", output.Workflow.LastRunAt.Format(time.RFC3339))
	}
	if len(output.Workflow.Tags) > 0 {
		fmt.Printf("Tags: %s\n", strings.Join(output.Workflow.Tags, ", "))
	}
}

// printExecuteFormatted prints the execution result in a conversational format
func (c *WorkflowsCommand) printExecuteFormatted(output *WorkflowsOutput) {
	fmt.Println("Workflow Execution")
	fmt.Println()
	fmt.Printf("Execution ID: %s\n", output.Execution.ID)
	fmt.Printf("Workflow ID: %s\n", output.Execution.WorkflowID)
	fmt.Printf("Status: %s\n", output.Execution.Status)
	fmt.Printf("Started: %s\n", output.Execution.StartedAt.Format(time.RFC3339))

	if output.Execution.FinishedAt != nil {
		fmt.Printf("Finished: %s\n", output.Execution.FinishedAt.Format(time.RFC3339))
	}

	if output.Execution.Duration != nil {
		fmt.Printf("Duration: %dms\n", *output.Execution.Duration)
	}

	if output.Execution.Error != "" {
		fmt.Printf("Error: %s\n", output.Execution.Error)
	}

	if len(output.Execution.OutputParams) > 0 {
		fmt.Println("Output Parameters:")
		for key, value := range output.Execution.OutputParams {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}
}

// printCreateFormatted prints the create result in a conversational format
func (c *WorkflowsCommand) printCreateFormatted(output *WorkflowsOutput) {
	fmt.Println("Workflow created successfully")
	fmt.Println()
	fmt.Printf("ID: %s\n", output.Workflow.ID)
	fmt.Printf("Name: %s\n", output.Workflow.Name)
	if output.Workflow.Description != "" {
		fmt.Printf("Description: %s\n", output.Workflow.Description)
	}
	fmt.Printf("Created: %s\n", output.Workflow.CreatedAt.Format(time.RFC3339))
}

// printUpdateFormatted prints the update result in a conversational format
func (c *WorkflowsCommand) printUpdateFormatted(output *WorkflowsOutput) {
	fmt.Println("Workflow updated successfully")
	fmt.Println()
	fmt.Printf("ID: %s\n", output.Workflow.ID)
	fmt.Printf("Name: %s\n", output.Workflow.Name)
	fmt.Printf("Modified: %s\n", output.Workflow.ModifiedAt.Format(time.RFC3339))
}

// printDeleteFormatted prints the delete result in a conversational format
func (c *WorkflowsCommand) printDeleteFormatted(output *WorkflowsOutput) {
	fmt.Println("Workflow deleted successfully")
	fmt.Println()
	fmt.Printf("ID: %s\n", output.Workflow.ID)
}

// Help prints the help message
func (c *WorkflowsCommand) Help() {
	fmt.Println("Usage: dd workflows <subcommand> [options]")
	fmt.Println()
	fmt.Println("Manage Datadog workflow automation - list, get, execute, create, update, delete workflows")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  list       List workflows")
	fmt.Println("  get        Get a specific workflow")
	fmt.Println("  execute    Execute a workflow")
	fmt.Println("  create     Create a new workflow")
	fmt.Println("  update     Update an existing workflow")
	fmt.Println("  delete     Delete a workflow")
	fmt.Println()
	fmt.Println("List options:")
	fmt.Println("  --tags string      Filter by tags (comma-separated)")
	fmt.Println("  --limit int        Max workflows to return (default 100)")
	fmt.Println("  --json             Output as JSON")
	fmt.Println()
	fmt.Println("Get options:")
	fmt.Println("  --id string        Workflow ID (required)")
	fmt.Println("  --json             Output as JSON")
	fmt.Println()
	fmt.Println("Execute options:")
	fmt.Println("  --id string        Workflow ID (required)")
	fmt.Println("  --params string    Input parameters as JSON (optional)")
	fmt.Println("  --wait             Wait for execution to complete")
	fmt.Println("  --timeout int      Timeout in seconds when waiting (default 300)")
	fmt.Println("  --json             Output as JSON")
	fmt.Println()
	fmt.Println("Create options:")
	fmt.Println("  --name string        Workflow name (required)")
	fmt.Println("  --description string Workflow description")
	fmt.Println("  --definition string  Workflow definition as JSON (required)")
	fmt.Println("  --tags string        Tags (comma-separated)")
	fmt.Println("  --json               Output as JSON")
	fmt.Println()
	fmt.Println("Update options:")
	fmt.Println("  --id string          Workflow ID (required)")
	fmt.Println("  --name string        New workflow name")
	fmt.Println("  --description string New workflow description")
	fmt.Println("  --definition string  New workflow definition as JSON")
	fmt.Println("  --json               Output as JSON")
	fmt.Println()
	fmt.Println("Delete options:")
	fmt.Println("  --id string        Workflow ID (required)")
	fmt.Println("  --json             Output as JSON")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # List all workflows")
	fmt.Println("  dd workflows list")
	fmt.Println()
	fmt.Println("  # Get workflow details")
	fmt.Println("  dd workflows get --id workflow-abc123")
	fmt.Println()
	fmt.Println("  # Execute workflow with parameters")
	fmt.Println("  dd workflows execute --id workflow-abc123 \\")
	fmt.Println("    --params '{\"service\":\"payment-api\",\"severity\":\"high\"}' \\")
	fmt.Println("    --wait")
	fmt.Println()
	fmt.Println("  # Create workflow")
	fmt.Println("  dd workflows create --name \"Auto-remediation\" \\")
	fmt.Println("    --description \"Automatically restart failing services\" \\")
	fmt.Println("    --definition '{\"steps\":[...]}' \\")
	fmt.Println("    --tags \"automation,remediation\"")
	fmt.Println()
	fmt.Println("  # Update workflow")
	fmt.Println("  dd workflows update --id workflow-abc123 \\")
	fmt.Println("    --name \"Updated workflow name\"")
	fmt.Println()
	fmt.Println("  # Delete workflow")
	fmt.Println("  dd workflows delete --id workflow-abc123")
}
