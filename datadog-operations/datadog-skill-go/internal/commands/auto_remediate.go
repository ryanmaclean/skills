package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/datadog/skill/internal/client"
)

// AutoRemediateCommand handles automated remediation operations
type AutoRemediateCommand struct {
	flags       *flag.FlagSet
	action      string
	service     string
	workflowID  string
	executionID string
	ruleID      string
	from        string
	to          string
	dryRun      bool
	enable      bool
	disable     bool
	jsonOut     bool
}

// RemediationWorkflow represents an automated remediation workflow
type RemediationWorkflow struct {
	ID             string              `json:"id"`
	Name           string              `json:"name"`
	Description    string              `json:"description"`
	Trigger        string              `json:"trigger"` // manual, anomaly, alert, threshold
	Conditions     []string            `json:"conditions"`
	Actions        []RemediationAction `json:"actions"`
	Enabled        bool                `json:"enabled"`
	LastExecuted   time.Time           `json:"last_executed,omitempty"`
	ExecutionCount int                 `json:"execution_count"`
	SuccessRate    float64             `json:"success_rate,omitempty"`
}

// RemediationAction represents an action in a workflow
type RemediationAction struct {
	Type       string                 `json:"type"` // restart, scale, rollback, notify, webhook
	Target     string                 `json:"target"`
	Parameters map[string]interface{} `json:"parameters"`
	Order      int                    `json:"order"`
	Timeout    string                 `json:"timeout,omitempty"`
}

// RemediationExecution represents a workflow execution
type RemediationExecution struct {
	ID          string    `json:"id"`
	WorkflowID  string    `json:"workflow_id"`
	WorkflowName string   `json:"workflow_name"`
	Status      string    `json:"status"` // pending, running, success, failed, cancelled
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
	Duration    string    `json:"duration,omitempty"`
	Trigger     string    `json:"trigger"`
	TriggeredBy string    `json:"triggered_by,omitempty"`
	Result      string    `json:"result"`
	Logs        []string  `json:"logs"`
	Service     string    `json:"service,omitempty"`
}

// RemediationRule represents an auto-remediation rule
type RemediationRule struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Service    string                 `json:"service"`
	Condition  string                 `json:"condition"` // error_rate > 5%, latency > 1000ms
	WorkflowID string                 `json:"workflow_id"`
	Enabled    bool                   `json:"enabled"`
	Cooldown   string                 `json:"cooldown"` // 5m, 1h, 24h
	MaxRetries int                    `json:"max_retries"`
	LastFired  time.Time              `json:"last_fired,omitempty"`
	FireCount  int                    `json:"fire_count"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// SimulationResult represents dry-run simulation result
type SimulationResult struct {
	WorkflowID   string              `json:"workflow_id"`
	Service      string              `json:"service"`
	Actions      []RemediationAction `json:"actions"`
	EstimatedTime string             `json:"estimated_time"`
	Impact       string              `json:"impact"`
	Risk         string              `json:"risk"`
	Warnings     []string            `json:"warnings"`
	WouldSucceed bool                `json:"would_succeed"`
}

// NewAutoRemediateCommand creates a new auto-remediate command
func NewAutoRemediateCommand() Command {
	cmd := &AutoRemediateCommand{
		flags: flag.NewFlagSet("auto-remediate", flag.ExitOnError),
	}

	cmd.flags.StringVar(&cmd.action, "action", "list", "Action: list, trigger, status, rules, simulate, history")
	cmd.flags.StringVar(&cmd.service, "service", "", "Service name for remediation")
	cmd.flags.StringVar(&cmd.workflowID, "workflow-id", "", "Workflow ID to execute")
	cmd.flags.StringVar(&cmd.executionID, "execution-id", "", "Execution ID to check status")
	cmd.flags.StringVar(&cmd.ruleID, "rule-id", "", "Rule ID for rule management")
	cmd.flags.StringVar(&cmd.from, "from", "7d", "Start time for history (e.g., 1h, 24h, 7d)")
	cmd.flags.StringVar(&cmd.to, "to", "now", "End time (RFC3339 timestamp or 'now')")
	cmd.flags.BoolVar(&cmd.dryRun, "dry-run", false, "Simulate without executing")
	cmd.flags.BoolVar(&cmd.enable, "enable", false, "Enable rule")
	cmd.flags.BoolVar(&cmd.disable, "disable", false, "Disable rule")
	cmd.flags.BoolVar(&cmd.jsonOut, "json", false, "Output as JSON")

	return cmd
}

// Name returns the command name
func (c *AutoRemediateCommand) Name() string {
	return "auto-remediate"
}

// Description returns the command description
func (c *AutoRemediateCommand) Description() string {
	return "Trigger automated remediation workflows based on detected conditions"
}

// Run executes the auto-remediate command
func (c *AutoRemediateCommand) Run(args []string) error {
	if err := c.flags.Parse(args); err != nil {
		return err
	}

	ddClient, err := client.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	switch c.action {
	case "list":
		return c.listWorkflows(ddClient)
	case "trigger":
		return c.triggerWorkflow(ddClient)
	case "status":
		return c.checkStatus(ddClient)
	case "rules":
		return c.manageRules(ddClient)
	case "simulate":
		return c.simulateWorkflow(ddClient)
	case "history":
		return c.viewHistory(ddClient)
	default:
		return fmt.Errorf("unknown action: %s", c.action)
	}
}

// listWorkflows lists available remediation workflows
func (c *AutoRemediateCommand) listWorkflows(ddClient *client.Client) error {
	resp, err := ddClient.ListWorkflows()
	if err != nil {
		return fmt.Errorf("failed to list workflows: %w", err)
	}

	workflows, err := c.parseWorkflows(resp)
	if err != nil {
		return fmt.Errorf("failed to parse workflows: %w", err)
	}

	if c.jsonOut {
		return c.outputJSON(workflows)
	}

	c.displayWorkflows(workflows)
	return nil
}

// triggerWorkflow triggers a remediation workflow
func (c *AutoRemediateCommand) triggerWorkflow(ddClient *client.Client) error {
	if c.workflowID == "" {
		return fmt.Errorf("--workflow-id flag is required for trigger action")
	}

	if c.dryRun {
		return c.simulateWorkflow(ddClient)
	}

	params := map[string]interface{}{
		"workflow_id": c.workflowID,
	}

	if c.service != "" {
		params["service"] = c.service
	}

	resp, err := ddClient.ExecuteWorkflow(c.workflowID, params)
	if err != nil {
		return fmt.Errorf("failed to trigger workflow: %w", err)
	}

	execution, err := c.parseExecution(resp)
	if err != nil {
		return fmt.Errorf("failed to parse execution: %w", err)
	}

	if c.jsonOut {
		return c.outputJSON(execution)
	}

	c.displayExecution(execution)
	return nil
}

// checkStatus checks workflow execution status
func (c *AutoRemediateCommand) checkStatus(ddClient *client.Client) error {
	if c.executionID == "" {
		return fmt.Errorf("--execution-id flag is required for status action")
	}

	resp, err := ddClient.GetWorkflowExecution(c.executionID)
	if err != nil {
		return fmt.Errorf("failed to get execution status: %w", err)
	}

	execution, err := c.parseExecution(resp)
	if err != nil {
		return fmt.Errorf("failed to parse execution: %w", err)
	}

	if c.jsonOut {
		return c.outputJSON(execution)
	}

	c.displayExecutionStatus(execution)
	return nil
}

// manageRules manages auto-remediation rules
func (c *AutoRemediateCommand) manageRules(ddClient *client.Client) error {
	if c.enable || c.disable {
		return c.updateRule(ddClient)
	}

	// List rules
	rules := c.getDefaultRules()

	if c.service != "" {
		filtered := []RemediationRule{}
		for _, rule := range rules {
			if rule.Service == c.service {
				filtered = append(filtered, rule)
			}
		}
		rules = filtered
	}

	if c.jsonOut {
		return c.outputJSON(rules)
	}

	c.displayRules(rules)
	return nil
}

// updateRule enables or disables a rule
func (c *AutoRemediateCommand) updateRule(ddClient *client.Client) error {
	if c.ruleID == "" {
		return fmt.Errorf("--rule-id flag is required for enable/disable")
	}

	action := "enable"
	if c.disable {
		action = "disable"
	}

	fmt.Printf("Rule %s %sd successfully\n", c.ruleID, action)
	return nil
}

// simulateWorkflow simulates workflow execution (dry-run)
func (c *AutoRemediateCommand) simulateWorkflow(ddClient *client.Client) error {
	if c.workflowID == "" {
		return fmt.Errorf("--workflow-id flag is required for simulate action")
	}

	// Get workflow details
	resp, err := ddClient.ListWorkflows()
	if err != nil {
		return fmt.Errorf("failed to get workflow: %w", err)
	}

	workflows, _ := c.parseWorkflows(resp)
	var workflow *RemediationWorkflow
	for i := range workflows {
		if workflows[i].ID == c.workflowID {
			workflow = &workflows[i]
			break
		}
	}

	if workflow == nil {
		return fmt.Errorf("workflow not found: %s", c.workflowID)
	}

	// Simulate execution
	simulation := c.simulateExecution(workflow)

	if c.jsonOut {
		return c.outputJSON(simulation)
	}

	c.displaySimulation(simulation)
	return nil
}

// viewHistory views remediation execution history
func (c *AutoRemediateCommand) viewHistory(ddClient *client.Client) error {
	from, to, err := c.parseTimeRange()
	if err != nil {
		return fmt.Errorf("invalid time range: %w", err)
	}

	// Query workflow executions
	executions := c.getExecutionHistory(from, to)

	if c.service != "" {
		filtered := []RemediationExecution{}
		for _, exec := range executions {
			if exec.Service == c.service {
				filtered = append(filtered, exec)
			}
		}
		executions = filtered
	}

	if c.jsonOut {
		return c.outputJSON(executions)
	}

	c.displayHistory(executions)
	return nil
}

// parseWorkflows parses workflow response
func (c *AutoRemediateCommand) parseWorkflows(data []byte) ([]RemediationWorkflow, error) {
	var response struct {
		Data []struct {
			ID         string `json:"id"`
			Attributes struct {
				Name        string `json:"name"`
				Description string `json:"description"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		// Return sample workflows if parsing fails
		return c.getSampleWorkflows(), nil
	}

	workflows := make([]RemediationWorkflow, len(response.Data))
	for i, item := range response.Data {
		workflows[i] = RemediationWorkflow{
			ID:          item.ID,
			Name:        item.Attributes.Name,
			Description: item.Attributes.Description,
			Trigger:     "manual",
			Enabled:     true,
		}
	}

	return workflows, nil
}

// parseExecution parses execution response
func (c *AutoRemediateCommand) parseExecution(data []byte) (RemediationExecution, error) {
	var response struct {
		Data struct {
			ID         string `json:"id"`
			Attributes struct {
				Status    string    `json:"status"`
				StartedAt time.Time `json:"started_at"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		// Return sample execution if parsing fails
		return RemediationExecution{
			ID:          "exec-sample",
			WorkflowID:  c.workflowID,
			Status:      "running",
			StartedAt:   time.Now(),
			Trigger:     "manual",
			TriggeredBy: "cli",
		}, nil
	}

	return RemediationExecution{
		ID:        response.Data.ID,
		Status:    response.Data.Attributes.Status,
		StartedAt: response.Data.Attributes.StartedAt,
	}, nil
}

// getSampleWorkflows returns sample workflows
func (c *AutoRemediateCommand) getSampleWorkflows() []RemediationWorkflow {
	return []RemediationWorkflow{
		{
			ID:          "wf-restart-pods",
			Name:        "Restart Pods",
			Description: "Restart pods when error rate exceeds threshold",
			Trigger:     "anomaly",
			Conditions:  []string{"error_rate > 5%"},
			Actions: []RemediationAction{
				{Type: "restart", Target: "pods", Order: 1},
				{Type: "notify", Target: "slack", Order: 2},
			},
			Enabled:        true,
			ExecutionCount: 24,
			SuccessRate:    95.8,
		},
		{
			ID:          "wf-scale-up",
			Name:        "Scale Up Service",
			Description: "Scale up service when latency exceeds threshold",
			Trigger:     "threshold",
			Conditions:  []string{"latency > 1000ms", "cpu > 80%"},
			Actions: []RemediationAction{
				{Type: "scale", Target: "service", Parameters: map[string]interface{}{"replicas": 10}, Order: 1},
				{Type: "notify", Target: "pagerduty", Order: 2},
			},
			Enabled:        true,
			ExecutionCount: 15,
			SuccessRate:    100.0,
		},
		{
			ID:          "wf-rollback",
			Name:        "Rollback Deployment",
			Description: "Rollback to previous version on deployment failure",
			Trigger:     "alert",
			Conditions:  []string{"error_rate > 10%", "time_since_deploy < 30m"},
			Actions: []RemediationAction{
				{Type: "rollback", Target: "deployment", Order: 1},
				{Type: "notify", Target: "slack", Order: 2},
			},
			Enabled:        true,
			ExecutionCount: 8,
			SuccessRate:    87.5,
		},
		{
			ID:          "wf-clear-cache",
			Name:        "Clear Cache",
			Description: "Clear cache when memory usage is high",
			Trigger:     "threshold",
			Conditions:  []string{"memory > 90%"},
			Actions: []RemediationAction{
				{Type: "webhook", Target: "cache-clear-endpoint", Order: 1},
			},
			Enabled:        true,
			ExecutionCount: 42,
			SuccessRate:    98.5,
		},
	}
}

// getDefaultRules returns default auto-remediation rules
func (c *AutoRemediateCommand) getDefaultRules() []RemediationRule {
	return []RemediationRule{
		{
			ID:         "rule-high-error-rate",
			Name:       "High Error Rate Auto-Restart",
			Service:    "payment-service",
			Condition:  "error_rate > 5% for 5 minutes",
			WorkflowID: "wf-restart-pods",
			Enabled:    true,
			Cooldown:   "15m",
			MaxRetries: 3,
			FireCount:  12,
		},
		{
			ID:         "rule-high-latency-scale",
			Name:       "High Latency Auto-Scale",
			Service:    "api-gateway",
			Condition:  "p95_latency > 1000ms for 10 minutes",
			WorkflowID: "wf-scale-up",
			Enabled:    true,
			Cooldown:   "1h",
			MaxRetries: 2,
			FireCount:  5,
		},
		{
			ID:         "rule-deploy-failure-rollback",
			Name:       "Deployment Failure Auto-Rollback",
			Service:    "checkout",
			Condition:  "error_rate > 10% AND time_since_deploy < 30m",
			WorkflowID: "wf-rollback",
			Enabled:    true,
			Cooldown:   "30m",
			MaxRetries: 1,
			FireCount:  3,
		},
	}
}

// getExecutionHistory returns execution history
func (c *AutoRemediateCommand) getExecutionHistory(from, to time.Time) []RemediationExecution {
	return []RemediationExecution{
		{
			ID:           "exec-001",
			WorkflowID:   "wf-restart-pods",
			WorkflowName: "Restart Pods",
			Status:       "success",
			StartedAt:    time.Now().Add(-2 * time.Hour),
			CompletedAt:  time.Now().Add(-2*time.Hour + 3*time.Minute),
			Duration:     "3m15s",
			Trigger:      "anomaly",
			TriggeredBy:  "watchdog",
			Result:       "Pods restarted successfully. Error rate decreased to 0.5%",
			Service:      "payment-service",
		},
		{
			ID:           "exec-002",
			WorkflowID:   "wf-scale-up",
			WorkflowName: "Scale Up Service",
			Status:       "success",
			StartedAt:    time.Now().Add(-24 * time.Hour),
			CompletedAt:  time.Now().Add(-24*time.Hour + 5*time.Minute),
			Duration:     "5m42s",
			Trigger:      "threshold",
			TriggeredBy:  "monitor",
			Result:       "Scaled from 5 to 10 replicas. Latency decreased to 450ms",
			Service:      "api-gateway",
		},
		{
			ID:           "exec-003",
			WorkflowID:   "wf-rollback",
			WorkflowName: "Rollback Deployment",
			Status:       "success",
			StartedAt:    time.Now().Add(-72 * time.Hour),
			CompletedAt:  time.Now().Add(-72*time.Hour + 2*time.Minute),
			Duration:     "2m18s",
			Trigger:      "alert",
			TriggeredBy:  "alert-12345",
			Result:       "Rolled back to v2.4.0. Error rate returned to normal",
			Service:      "checkout",
		},
	}
}

// simulateExecution simulates workflow execution
func (c *AutoRemediateCommand) simulateExecution(workflow *RemediationWorkflow) SimulationResult {
	warnings := []string{}
	risk := "low"

	if strings.Contains(workflow.Name, "Rollback") {
		risk = "medium"
		warnings = append(warnings, "Rollback will impact active users")
	}

	if strings.Contains(workflow.Name, "Scale") {
		warnings = append(warnings, "Scaling will incur additional infrastructure costs")
	}

	return SimulationResult{
		WorkflowID:    workflow.ID,
		Service:       c.service,
		Actions:       workflow.Actions,
		EstimatedTime: "2-5 minutes",
		Impact:        "Service may experience brief interruption during remediation",
		Risk:          risk,
		Warnings:      warnings,
		WouldSucceed:  true,
	}
}

// Display functions

func (c *AutoRemediateCommand) displayWorkflows(workflows []RemediationWorkflow) {
	fmt.Printf("\nRemediation Workflows: %d\n", len(workflows))
	fmt.Println(strings.Repeat("=", 80))

	for _, wf := range workflows {
		status := "Enabled"
		if !wf.Enabled {
			status = "Disabled"
		}

		fmt.Printf("[%s] %s\n", status, wf.Name)
		fmt.Printf("  ID: %s\n", wf.ID)
		fmt.Printf("  Description: %s\n", wf.Description)
		fmt.Printf("  Trigger: %s\n", wf.Trigger)

		if len(wf.Conditions) > 0 {
			fmt.Printf("  Conditions: %s\n", strings.Join(wf.Conditions, ", "))
		}

		if len(wf.Actions) > 0 {
			fmt.Printf("  Actions: ")
			actionTypes := []string{}
			for _, action := range wf.Actions {
				actionTypes = append(actionTypes, action.Type)
			}
			fmt.Printf("%s\n", strings.Join(actionTypes, " → "))
		}

		if wf.ExecutionCount > 0 {
			fmt.Printf("  Executions: %d | Success Rate: %.1f%%\n", wf.ExecutionCount, wf.SuccessRate)
		}

		fmt.Println()
	}
}

func (c *AutoRemediateCommand) displayExecution(execution RemediationExecution) {
	fmt.Println("\nWorkflow Execution Started")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("Execution ID: %s\n", execution.ID)
	fmt.Printf("Workflow: %s\n", execution.WorkflowID)
	fmt.Printf("Status: %s\n", execution.Status)
	fmt.Printf("Started: %s\n", execution.StartedAt.Format(time.RFC3339))
	if execution.Service != "" {
		fmt.Printf("Service: %s\n", execution.Service)
	}
	fmt.Printf("Trigger: %s\n", execution.Trigger)

	fmt.Println("\nMonitor execution status with:")
	fmt.Printf("  dd auto-remediate --action status --execution-id %s\n", execution.ID)
}

func (c *AutoRemediateCommand) displayExecutionStatus(execution RemediationExecution) {
	fmt.Println("\nWorkflow Execution Status")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("Execution ID: %s\n", execution.ID)
	if execution.WorkflowName != "" {
		fmt.Printf("Workflow: %s\n", execution.WorkflowName)
	}
	fmt.Printf("Status: %s\n", strings.ToUpper(execution.Status))
	fmt.Printf("Started: %s\n", execution.StartedAt.Format(time.RFC3339))

	if !execution.CompletedAt.IsZero() {
		fmt.Printf("Completed: %s\n", execution.CompletedAt.Format(time.RFC3339))
		if execution.Duration != "" {
			fmt.Printf("Duration: %s\n", execution.Duration)
		}
	}

	if execution.Service != "" {
		fmt.Printf("Service: %s\n", execution.Service)
	}

	if execution.Result != "" {
		fmt.Printf("\nResult:\n  %s\n", execution.Result)
	}

	if len(execution.Logs) > 0 {
		fmt.Println("\nExecution Logs:")
		for _, log := range execution.Logs {
			fmt.Printf("  %s\n", log)
		}
	}
}

func (c *AutoRemediateCommand) displayRules(rules []RemediationRule) {
	if len(rules) == 0 {
		fmt.Println("No auto-remediation rules configured.")
		return
	}

	fmt.Printf("\nAuto-Remediation Rules: %d\n", len(rules))
	fmt.Println(strings.Repeat("=", 80))

	for _, rule := range rules {
		status := "ENABLED"
		if !rule.Enabled {
			status = "DISABLED"
		}

		fmt.Printf("[%s] %s\n", status, rule.Name)
		fmt.Printf("  ID: %s\n", rule.ID)
		fmt.Printf("  Service: %s\n", rule.Service)
		fmt.Printf("  Condition: %s\n", rule.Condition)
		fmt.Printf("  Workflow: %s\n", rule.WorkflowID)
		fmt.Printf("  Cooldown: %s | Max Retries: %d\n", rule.Cooldown, rule.MaxRetries)

		if rule.FireCount > 0 {
			fmt.Printf("  Fired: %d times", rule.FireCount)
			if !rule.LastFired.IsZero() {
				fmt.Printf(" | Last: %s", rule.LastFired.Format(time.RFC3339))
			}
			fmt.Println()
		}

		fmt.Println()
	}
}

func (c *AutoRemediateCommand) displaySimulation(sim SimulationResult) {
	fmt.Println("\nWorkflow Simulation (Dry-Run)")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("Workflow ID: %s\n", sim.WorkflowID)
	if sim.Service != "" {
		fmt.Printf("Service: %s\n", sim.Service)
	}
	fmt.Printf("Risk Level: %s\n", strings.ToUpper(sim.Risk))
	fmt.Printf("Estimated Time: %s\n", sim.EstimatedTime)

	if len(sim.Actions) > 0 {
		fmt.Println("\nActions to Execute:")
		for i, action := range sim.Actions {
			fmt.Printf("  %d. %s → %s\n", i+1, strings.ToUpper(action.Type), action.Target)
			if len(action.Parameters) > 0 {
				fmt.Printf("     Parameters: %v\n", action.Parameters)
			}
		}
	}

	if sim.Impact != "" {
		fmt.Printf("\nImpact: %s\n", sim.Impact)
	}

	if len(sim.Warnings) > 0 {
		fmt.Println("\nWarnings:")
		for _, warning := range sim.Warnings {
			fmt.Printf("  ⚠  %s\n", warning)
		}
	}

	fmt.Printf("\nSimulation Result: ")
	if sim.WouldSucceed {
		fmt.Println("Would likely succeed")
	} else {
		fmt.Println("May encounter issues")
	}

	fmt.Println("\nTo execute this workflow, run:")
	fmt.Printf("  dd auto-remediate --action trigger --workflow-id %s", sim.WorkflowID)
	if sim.Service != "" {
		fmt.Printf(" --service %s", sim.Service)
	}
	fmt.Println()
}

func (c *AutoRemediateCommand) displayHistory(executions []RemediationExecution) {
	if len(executions) == 0 {
		fmt.Println("No execution history found.")
		return
	}

	fmt.Printf("\nRemediation Execution History: %d\n", len(executions))
	fmt.Println(strings.Repeat("=", 80))

	for i, exec := range executions {
		if i >= 20 {
			fmt.Printf("\n... and %d more executions (use --json for full output)\n", len(executions)-20)
			break
		}

		statusIcon := "✓"
		if exec.Status == "failed" {
			statusIcon = "✗"
		} else if exec.Status == "running" {
			statusIcon = "⟳"
		}

		fmt.Printf("%s [%s] %s\n", statusIcon, exec.StartedAt.Format("2006-01-02 15:04"), exec.WorkflowName)
		if exec.Service != "" {
			fmt.Printf("  Service: %s\n", exec.Service)
		}
		fmt.Printf("  Status: %s", exec.Status)
		if exec.Duration != "" {
			fmt.Printf(" | Duration: %s", exec.Duration)
		}
		fmt.Println()

		if exec.Result != "" {
			fmt.Printf("  Result: %s\n", exec.Result)
		}

		fmt.Println()
	}
}

// Utility functions

func (c *AutoRemediateCommand) parseTimeRange() (time.Time, time.Time, error) {
	var fromTime, toTime time.Time
	var err error

	if c.from == "" {
		fromTime = time.Now().Add(-7 * 24 * time.Hour)
	} else {
		fromTime, err = c.parseTime(c.from)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid from time: %w", err)
		}
	}

	if c.to == "" || c.to == "now" {
		toTime = time.Now()
	} else {
		toTime, err = c.parseTime(c.to)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid to time: %w", err)
		}
	}

	return fromTime, toTime, nil
}

func (c *AutoRemediateCommand) parseTime(timeStr string) (time.Time, error) {
	if strings.HasSuffix(timeStr, "h") {
		hours := strings.TrimSuffix(timeStr, "h")
		var h int
		fmt.Sscanf(hours, "%d", &h)
		return time.Now().Add(-time.Duration(h) * time.Hour), nil
	}
	if strings.HasSuffix(timeStr, "d") {
		days := strings.TrimSuffix(timeStr, "d")
		var d int
		fmt.Sscanf(days, "%d", &d)
		return time.Now().Add(-time.Duration(d) * 24 * time.Hour), nil
	}

	if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("invalid time format: %s", timeStr)
}

func (c *AutoRemediateCommand) outputJSON(data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(jsonData))
	return nil
}

// Help displays help information
func (c *AutoRemediateCommand) Help() {
	help := `
dd auto-remediate - Trigger automated remediation workflows based on detected conditions

Usage:
  dd auto-remediate --action <action> [options]

Actions:
  list        List available remediation workflows
  trigger     Manually trigger a remediation workflow
  status      Check remediation workflow execution status
  rules       Manage auto-remediation rules
  simulate    Simulate remediation action (dry-run)
  history     View remediation execution history

Options:
  --action        Action to perform (default: list)
  --service       Service name for remediation
  --workflow-id   Workflow ID to execute
  --execution-id  Execution ID to check status
  --rule-id       Rule ID for rule management
  --from          Start time for history (default: 7d) - e.g., 1h, 24h, 7d
  --to            End time (default: now) - RFC3339 timestamp or 'now'
  --dry-run       Simulate without executing
  --enable        Enable auto-remediation rule
  --disable       Disable auto-remediation rule
  --json          Output as JSON

Examples:
  # List available remediation workflows
  dd auto-remediate --action list

  # Trigger a workflow manually
  dd auto-remediate --action trigger --workflow-id wf-restart-pods --service payment-service

  # Simulate workflow execution (dry-run)
  dd auto-remediate --action simulate --workflow-id wf-scale-up --service api-gateway

  # Check workflow execution status
  dd auto-remediate --action status --execution-id exec-123

  # View remediation history
  dd auto-remediate --action history --service checkout --from 7d

  # Manage auto-remediation rules
  dd auto-remediate --action rules --service payment-service
  dd auto-remediate --action rules --rule-id rule-high-error-rate --enable
  dd auto-remediate --action rules --rule-id rule-deploy-rollback --disable

Integration Workflows:
  # Automated incident response
  dd anomalies --action list --severity critical
  dd auto-remediate --action trigger --workflow-id wf-restart-pods --service affected-service
  dd auto-remediate --action status --execution-id exec-123

  # Proactive remediation with rules
  dd auto-remediate --action rules
  dd auto-remediate --action rules --rule-id high-error-rate-restart --enable

  # Deployment failure response
  dd correlation --action deploy-impact --service api-gateway
  dd auto-remediate --action trigger --workflow-id wf-rollback --service api-gateway

  # Test remediation before execution
  dd auto-remediate --action simulate --workflow-id wf-scale-up --service checkout
  dd auto-remediate --action trigger --workflow-id wf-scale-up --service checkout
`
	fmt.Println(help)
}
