package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"time"

	"github.com/datadog/skill/internal/client"
	"github.com/datadog/skill/internal/observability"
)

// ErrorBudgetsCommand queries and analyzes SLO error budgets
type ErrorBudgetsCommand struct {
	flags    *flag.FlagSet
	action   string
	sloID    string
	from     string
	to       string
	warning  float64
	critical float64
	jsonOut  bool
}

// ErrorBudget represents parsed error budget data
type ErrorBudget struct {
	SLOID              string  `json:"slo_id"`
	SLOName            string  `json:"slo_name,omitempty"`
	TargetThreshold    float64 `json:"target_threshold"`
	ErrorBudgetTotal   float64 `json:"error_budget_total"`
	ErrorBudgetUsed    float64 `json:"error_budget_used"`
	ErrorBudgetRemaining float64 `json:"error_budget_remaining"`
	ErrorBudgetPercent float64 `json:"error_budget_percent"`
	BurnRate           float64 `json:"burn_rate,omitempty"`
	Status             string  `json:"status"`
	TimeWindowDays     int     `json:"time_window_days"`
	CurrentSLI         float64 `json:"current_sli,omitempty"`
}

// ErrorBudgetsResponse represents the formatted error budgets response
type ErrorBudgetsResponse struct {
	Status       string        `json:"status"`
	TotalSLOs    int           `json:"total_slos"`
	ErrorBudgets []ErrorBudget `json:"error_budgets,omitempty"`
}

// NewErrorBudgetsCommand creates a new error budgets command
func NewErrorBudgetsCommand() *ErrorBudgetsCommand {
	cmd := &ErrorBudgetsCommand{
		flags: flag.NewFlagSet("error-budgets", flag.ExitOnError),
	}

	cmd.flags.StringVar(&cmd.action, "action", "query", "Action: query, status, summary")
	cmd.flags.StringVar(&cmd.sloID, "slo-id", "", "SLO ID to query error budget for")
	cmd.flags.StringVar(&cmd.from, "from", "", "Start time (RFC3339, defaults to SLO time window)")
	cmd.flags.StringVar(&cmd.to, "to", "", "End time (RFC3339, defaults to now)")
	cmd.flags.Float64Var(&cmd.warning, "warning", 25.0, "Warning threshold percentage (default: 25%)")
	cmd.flags.Float64Var(&cmd.critical, "critical", 10.0, "Critical threshold percentage (default: 10%)")
	cmd.flags.BoolVar(&cmd.jsonOut, "json", false, "Output as JSON")

	return cmd
}

// Name returns the command name
func (c *ErrorBudgetsCommand) Name() string {
	return "error-budgets"
}

// Description returns the command description
func (c *ErrorBudgetsCommand) Description() string {
	return "Query and analyze SLO error budgets for deployment decisions"
}

// Run executes the error budgets command
func (c *ErrorBudgetsCommand) Run(args []string) error {
	// Initialize observability
	obs, err := observability.Init("query-error-budgets", "production")
	if err != nil {
		return fmt.Errorf("failed to init observability: %w", err)
	}
	defer obs.Shutdown(0)

	// Parse flags
	if err := c.flags.Parse(args); err != nil {
		return err
	}

	// Start tracing
	span := obs.StartSpan("error_budgets.query")
	defer obs.FinishSpan(span)

	obs.LogInfo(fmt.Sprintf("Querying error budgets with action: %s", c.action))

	// Initialize Datadog client
	ddClient, err := client.NewClient()
	if err != nil {
		return fmt.Errorf("failed to initialize Datadog client: %w", err)
	}

	// Execute action
	switch c.action {
	case "query":
		return c.queryErrorBudget(ddClient, obs)
	case "status":
		return c.getErrorBudgetStatus(ddClient, obs)
	case "summary":
		return c.summarizeErrorBudgets(ddClient, obs)
	default:
		return fmt.Errorf("unknown action: %s (use: query, status, summary)", c.action)
	}
}

// queryErrorBudget queries error budget for a specific SLO
func (c *ErrorBudgetsCommand) queryErrorBudget(ddClient *client.Client, obs *observability.Observability) error {
	if c.sloID == "" {
		return fmt.Errorf("--slo-id is required for query action")
	}

	obs.LogInfo(fmt.Sprintf("Querying error budget for SLO: %s", c.sloID))

	// Parse time range
	fromTime, toTime, err := c.parseTimeRange()
	if err != nil {
		return err
	}

	// Query SLO history (includes error budget data)
	data, err := ddClient.QuerySLOHistory(c.sloID, fromTime, toTime)
	if err != nil {
		return fmt.Errorf("failed to query error budget: %w", err)
	}

	// Parse and display result
	return c.parseAndDisplaySingle(data, obs)
}

// getErrorBudgetStatus gets current error budget status
func (c *ErrorBudgetsCommand) getErrorBudgetStatus(ddClient *client.Client, obs *observability.Observability) error {
	if c.sloID == "" {
		return fmt.Errorf("--slo-id is required for status action")
	}

	obs.LogInfo(fmt.Sprintf("Getting error budget status for SLO: %s", c.sloID))

	// Use current time window
	toTime := time.Now()
	fromTime := toTime.Add(-30 * 24 * time.Hour) // Default 30 days

	// Query SLO history
	data, err := ddClient.QuerySLOHistory(c.sloID, fromTime, toTime)
	if err != nil {
		return fmt.Errorf("failed to get error budget status: %w", err)
	}

	// Parse and display with status focus
	return c.parseAndDisplayStatus(data, obs)
}

// summarizeErrorBudgets summarizes error budgets (placeholder for multi-SLO support)
func (c *ErrorBudgetsCommand) summarizeErrorBudgets(ddClient *client.Client, obs *observability.Observability) error {
	obs.LogInfo("Summarizing error budgets")

	// For now, require SLO ID (future: list all SLOs and summarize)
	if c.sloID == "" {
		return fmt.Errorf("--slo-id is required (multi-SLO summary not yet implemented)")
	}

	return c.queryErrorBudget(ddClient, obs)
}

// parseTimeRange parses the from/to time range
func (c *ErrorBudgetsCommand) parseTimeRange() (time.Time, time.Time, error) {
	// Default to last 30 days
	toTime := time.Now()
	fromTime := toTime.Add(-30 * 24 * time.Hour)

	// Parse 'to' time if provided
	if c.to != "" {
		t, err := time.Parse(time.RFC3339, c.to)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid 'to' time format (use RFC3339): %w", err)
		}
		toTime = t
	}

	// Parse 'from' time if provided
	if c.from != "" {
		t, err := time.Parse(time.RFC3339, c.from)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid 'from' time format (use RFC3339): %w", err)
		}
		fromTime = t
	}

	return fromTime, toTime, nil
}

// parseAndDisplaySingle parses and displays single SLO error budget
func (c *ErrorBudgetsCommand) parseAndDisplaySingle(data []byte, obs *observability.Observability) error {
	// Parse JSON response
	var apiResponse struct {
		Data struct {
			FromTS     int64   `json:"from_ts"`
			ToTS       int64   `json:"to_ts"`
			Overall    struct {
				SLIValue   float64 `json:"sli_value"`
				Uptime     float64 `json:"uptime"`
				Target     float64 `json:"target"`
				ErrorBudget struct {
					Total     float64 `json:"total"`
					Used      float64 `json:"used"`
					Remaining float64 `json:"remaining"`
				} `json:"error_budget"`
			} `json:"overall"`
		} `json:"data"`
		Meta struct {
			SLO struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				Type string `json:"type"`
			} `json:"slo"`
		} `json:"meta"`
	}

	if err := json.Unmarshal(data, &apiResponse); err != nil {
		return fmt.Errorf("failed to parse API response: %w", err)
	}

	// Calculate time window
	fromTime := time.Unix(apiResponse.Data.FromTS, 0)
	toTime := time.Unix(apiResponse.Data.ToTS, 0)
	timeWindowDays := int(toTime.Sub(fromTime).Hours() / 24)

	// Calculate error budget percentage
	errorBudgetPercent := 0.0
	if apiResponse.Data.Overall.ErrorBudget.Total > 0 {
		errorBudgetPercent = (apiResponse.Data.Overall.ErrorBudget.Remaining / apiResponse.Data.Overall.ErrorBudget.Total) * 100
	}

	// Determine status based on thresholds
	status := "HEALTHY"
	if errorBudgetPercent <= c.critical {
		status = "CRITICAL"
	} else if errorBudgetPercent <= c.warning {
		status = "WARNING"
	}

	// Build error budget object
	budget := ErrorBudget{
		SLOID:                apiResponse.Meta.SLO.ID,
		SLOName:              apiResponse.Meta.SLO.Name,
		TargetThreshold:      apiResponse.Data.Overall.Target,
		ErrorBudgetTotal:     apiResponse.Data.Overall.ErrorBudget.Total,
		ErrorBudgetUsed:      apiResponse.Data.Overall.ErrorBudget.Used,
		ErrorBudgetRemaining: apiResponse.Data.Overall.ErrorBudget.Remaining,
		ErrorBudgetPercent:   errorBudgetPercent,
		Status:               status,
		TimeWindowDays:       timeWindowDays,
		CurrentSLI:           apiResponse.Data.Overall.SLIValue,
	}

	// Calculate burn rate (used per day)
	if timeWindowDays > 0 {
		budget.BurnRate = apiResponse.Data.Overall.ErrorBudget.Used / float64(timeWindowDays)
	}

	// Output result
	if c.jsonOut {
		output, err := json.MarshalIndent(budget, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(output))
	} else {
		c.displayFormattedSingle(budget)
	}

	return nil
}

// parseAndDisplayStatus parses and displays status-focused view
func (c *ErrorBudgetsCommand) parseAndDisplayStatus(data []byte, obs *observability.Observability) error {
	// Reuse single display logic
	return c.parseAndDisplaySingle(data, obs)
}

// displayFormattedSingle displays formatted single error budget output
func (c *ErrorBudgetsCommand) displayFormattedSingle(budget ErrorBudget) {
	fmt.Println("Error Budget Status")
	fmt.Println("===================")
	fmt.Printf("SLO: %s\n", budget.SLOName)
	fmt.Printf("SLO ID: %s\n", budget.SLOID)
	fmt.Printf("Target: %.2f%%\n", budget.TargetThreshold)
	fmt.Printf("Current SLI: %.2f%%\n", budget.CurrentSLI)
	fmt.Println()

	// Error budget details
	fmt.Println("Error Budget:")
	fmt.Printf("  Total:     %.6f\n", budget.ErrorBudgetTotal)
	fmt.Printf("  Used:      %.6f (%.2f%%)\n", budget.ErrorBudgetUsed, (budget.ErrorBudgetUsed/budget.ErrorBudgetTotal)*100)
	fmt.Printf("  Remaining: %.6f (%.2f%%)\n", budget.ErrorBudgetRemaining, budget.ErrorBudgetPercent)
	fmt.Println()

	// Status indicator
	statusIcon := "✓"
	statusColor := "HEALTHY"
	if budget.Status == "CRITICAL" {
		statusIcon = "✗"
		statusColor = "CRITICAL"
	} else if budget.Status == "WARNING" {
		statusIcon = "⚠"
		statusColor = "WARNING"
	}

	fmt.Printf("Status: %s %s\n", statusIcon, statusColor)
	fmt.Println()

	// Burn rate
	if budget.BurnRate > 0 {
		fmt.Printf("Burn Rate: %.6f per day\n", budget.BurnRate)
		if budget.ErrorBudgetRemaining > 0 {
			daysRemaining := budget.ErrorBudgetRemaining / budget.BurnRate
			fmt.Printf("Days Until Exhaustion: %.1f days (at current rate)\n", daysRemaining)
		}
		fmt.Println()
	}

	// Time window
	fmt.Printf("Time Window: %d days\n", budget.TimeWindowDays)
	fmt.Println()

	// Deployment recommendation
	fmt.Println("Deployment Recommendation:")
	switch budget.Status {
	case "HEALTHY":
		fmt.Println("  ✓ GO - Error budget is healthy, safe to deploy")
		fmt.Println("  Remaining budget can absorb potential issues")
	case "WARNING":
		fmt.Printf("  ⚠ CAUTION - Error budget below %.0f%%\n", c.warning)
		fmt.Println("  Consider extra testing and staged rollout")
		fmt.Println("  Have rollback plan ready")
	case "CRITICAL":
		fmt.Printf("  ✗ NO-GO - Error budget below %.0f%%\n", c.critical)
		fmt.Println("  Focus on stability over new features")
		fmt.Println("  Consider change freeze until budget recovers")
		fmt.Println("  Investigate recent issues consuming budget")
	}
}

// Help displays help information
func (c *ErrorBudgetsCommand) Help() {
	help := `dd error-budgets - Query SLO Error Budgets

DESCRIPTION:
  Query and analyze SLO error budgets to make data-driven deployment decisions.
  Error budgets determine how much downtime or errors are acceptable before
  violating SLO commitments. Essential for SRE change management workflows.

USAGE:
  dd error-budgets --action <action> [options]

ACTIONS:
  query            Query error budget for specific SLO
  status           Get current error budget status
  summary          Summarize error budgets (requires --slo-id for now)

EXAMPLES:
  # Query error budget for SLO
  dd error-budgets --action query --slo-id "abc123"

  # Get current status
  dd error-budgets --action status --slo-id "abc123"

  # Query with custom time range
  dd error-budgets --action query \
    --slo-id "abc123" \
    --from "2026-01-01T00:00:00Z" \
    --to "2026-01-31T23:59:59Z"

  # Query with custom thresholds
  dd error-budgets --action query \
    --slo-id "abc123" \
    --warning 30 \
    --critical 15

  # Get JSON output
  dd error-budgets --action query --slo-id "abc123" --json

OPTIONS:
  --action          Action to perform (query, status, summary)
  --slo-id          SLO ID to query (required)
  --from            Start time (RFC3339, defaults to SLO time window)
  --to              End time (RFC3339, defaults to now)
  --warning         Warning threshold percentage (default: 25%)
  --critical        Critical threshold percentage (default: 10%)
  --json            Output as JSON

ERROR BUDGET CALCULATION:
  Error Budget = (1 - Target) × Time Window

  Example: 99.9% uptime target over 30 days
  Error Budget = (1 - 0.999) × 30 days = 0.03 days = 43.2 minutes

  Remaining % = (Remaining / Total) × 100

STATUS LEVELS:
  HEALTHY:   Error budget > warning threshold (default: >25%)
             Safe to deploy, budget can absorb issues

  WARNING:   Error budget between critical and warning (10-25%)
             Deploy with caution, use staged rollout
             Have rollback plan ready

  CRITICAL:  Error budget < critical threshold (default: <10%)
             Consider change freeze
             Focus on stability over features
             Investigate budget consumption

DEPLOYMENT DECISION FRAMEWORK:
  HEALTHY (>25%):
    ✓ Deploy normally
    ✓ Standard testing process
    ✓ Regular deployment velocity

  WARNING (10-25%):
    ⚠ Deploy with extra caution
    ⚠ Increase testing coverage
    ⚠ Use staged/canary rollout
    ⚠ Monitor closely post-deploy
    ⚠ Have rollback plan ready

  CRITICAL (<10%):
    ✗ Consider change freeze
    ✗ Focus on stability improvements
    ✗ Fix issues consuming budget
    ✗ Only emergency fixes
    ✗ Wait for budget recovery

BURN RATE ANALYSIS:
  Burn Rate = Error Budget Used / Days Elapsed
  Days Remaining = Error Budget Remaining / Burn Rate

  Use burn rate to project budget exhaustion
  High burn rate indicates recent issues
  Adjust deployment cadence based on burn rate

SRE WORKFLOW:
  1. Check error budget before deployment
  2. Make go/no-go decision based on status
  3. If deploying, monitor budget consumption
  4. If budget depleted, focus on reliability
  5. Review budget trends weekly/monthly

INTEGRATION WITH SLO MANAGEMENT:
  1. Define SLO (dd slos command)
  2. Apply corrections if needed (dd slo-corrections)
  3. Check error budget (dd error-budgets)
  4. Make deployment decision
  5. Analyze trends (dd slo-history)

USE CASES:
  - Pre-deployment risk assessment
  - Change freeze decisions
  - Incident prioritization
  - Release go/no-go decisions
  - Team velocity management
  - Stakeholder communication

NOTES:
  - Error budgets reset based on SLO time window (rolling window)
  - Corrections affect error budget calculations
  - Use conservative thresholds for critical services
  - Track burn rate to identify systemic issues
  - Coordinate with stakeholders on change freezes

For more information: https://docs.datadoghq.com/api/latest/service-level-objectives/
`
	fmt.Println(help)
}
