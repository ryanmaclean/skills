package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"time"

	"github.com/datadog/skill/internal/client"
	"github.com/datadog/skill/internal/observability"
)

// SLOHistoryCommand queries historical SLO compliance data
type SLOHistoryCommand struct {
	flags   *flag.FlagSet
	action  string
	sloID   string
	from    string
	to      string
	jsonOut bool
}

// SLOHistoryData represents parsed historical SLO data
type SLOHistoryData struct {
	SLOID           string                   `json:"slo_id"`
	SLOName         string                   `json:"slo_name,omitempty"`
	FromTS          int64                    `json:"from_ts"`
	ToTS            int64                    `json:"to_ts"`
	TargetThreshold float64                  `json:"target_threshold"`
	OverallSLI      float64                  `json:"overall_sli"`
	OverallUptime   float64                  `json:"overall_uptime,omitempty"`
	ErrorBudget     ErrorBudgetHistory       `json:"error_budget"`
	Series          []SLOHistorySeries       `json:"series,omitempty"`
	Compliance      string                   `json:"compliance"`
}

// ErrorBudgetHistory represents error budget over time
type ErrorBudgetHistory struct {
	Total     float64 `json:"total"`
	Used      float64 `json:"used"`
	Remaining float64 `json:"remaining"`
	Percent   float64 `json:"percent"`
}

// SLOHistorySeries represents time series data points
type SLOHistorySeries struct {
	Timestamp int64   `json:"timestamp"`
	SLI       float64 `json:"sli"`
}

// SLOHistoryQueryResponse represents the formatted response
type SLOHistoryQueryResponse struct {
	Status  string         `json:"status"`
	History SLOHistoryData `json:"history"`
}

// NewSLOHistoryCommand creates a new SLO history command
func NewSLOHistoryCommand() *SLOHistoryCommand {
	cmd := &SLOHistoryCommand{
		flags: flag.NewFlagSet("slo-history", flag.ExitOnError),
	}

	cmd.flags.StringVar(&cmd.action, "action", "query", "Action: query, trends, report")
	cmd.flags.StringVar(&cmd.sloID, "slo-id", "", "SLO ID to query history for (required)")
	cmd.flags.StringVar(&cmd.from, "from", "", "Start time (RFC3339 or relative like '7d', '30d')")
	cmd.flags.StringVar(&cmd.to, "to", "", "End time (RFC3339, defaults to now)")
	cmd.flags.BoolVar(&cmd.jsonOut, "json", false, "Output as JSON")

	return cmd
}

// Name returns the command name
func (c *SLOHistoryCommand) Name() string {
	return "slo-history"
}

// Description returns the command description
func (c *SLOHistoryCommand) Description() string {
	return "Query historical SLO compliance data for trend analysis"
}

// Run executes the SLO history command
func (c *SLOHistoryCommand) Run(args []string) error {
	// Initialize observability
	obs, err := observability.Init("query-slo-history", "production")
	if err != nil {
		return fmt.Errorf("failed to init observability: %w", err)
	}
	defer obs.Shutdown(0)

	// Parse flags
	if err := c.flags.Parse(args); err != nil {
		return err
	}

	// Validate required fields
	if c.sloID == "" {
		return fmt.Errorf("--slo-id is required")
	}

	// Start tracing
	span := obs.StartSpan("slo_history.query")
	defer obs.FinishSpan(span)

	obs.LogInfo(fmt.Sprintf("Querying SLO history with action: %s", c.action))

	// Initialize Datadog client
	ddClient, err := client.NewClient()
	if err != nil {
		return fmt.Errorf("failed to initialize Datadog client: %w", err)
	}

	// Execute action
	switch c.action {
	case "query":
		return c.queryHistory(ddClient, obs)
	case "trends":
		return c.analyzeTrends(ddClient, obs)
	case "report":
		return c.generateReport(ddClient, obs)
	default:
		return fmt.Errorf("unknown action: %s (use: query, trends, report)", c.action)
	}
}

// queryHistory queries SLO historical data
func (c *SLOHistoryCommand) queryHistory(ddClient *client.Client, obs *observability.Observability) error {
	obs.LogInfo(fmt.Sprintf("Querying history for SLO: %s", c.sloID))

	// Parse time range
	fromTime, toTime, err := c.parseTimeRange()
	if err != nil {
		return err
	}

	// Query SLO history
	data, err := ddClient.QuerySLOHistory(c.sloID, fromTime, toTime)
	if err != nil {
		return fmt.Errorf("failed to query SLO history: %w", err)
	}

	// Parse and display results
	return c.parseAndDisplay(data, obs, "query")
}

// analyzeTrends analyzes SLO trends
func (c *SLOHistoryCommand) analyzeTrends(ddClient *client.Client, obs *observability.Observability) error {
	obs.LogInfo(fmt.Sprintf("Analyzing trends for SLO: %s", c.sloID))

	// Parse time range
	fromTime, toTime, err := c.parseTimeRange()
	if err != nil {
		return err
	}

	// Query SLO history
	data, err := ddClient.QuerySLOHistory(c.sloID, fromTime, toTime)
	if err != nil {
		return fmt.Errorf("failed to query SLO history: %w", err)
	}

	// Parse and display with trend analysis
	return c.parseAndDisplay(data, obs, "trends")
}

// generateReport generates compliance report
func (c *SLOHistoryCommand) generateReport(ddClient *client.Client, obs *observability.Observability) error {
	obs.LogInfo(fmt.Sprintf("Generating report for SLO: %s", c.sloID))

	// Parse time range
	fromTime, toTime, err := c.parseTimeRange()
	if err != nil {
		return err
	}

	// Query SLO history
	data, err := ddClient.QuerySLOHistory(c.sloID, fromTime, toTime)
	if err != nil {
		return fmt.Errorf("failed to query SLO history: %w", err)
	}

	// Parse and display as report
	return c.parseAndDisplay(data, obs, "report")
}

// parseTimeRange parses the from/to time range
func (c *SLOHistoryCommand) parseTimeRange() (time.Time, time.Time, error) {
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
		// Try RFC3339 first
		t, err := time.Parse(time.RFC3339, c.from)
		if err != nil {
			// Try relative time (e.g., "7d", "30d", "90d")
			duration, err := parseRelativeDuration(c.from)
			if err != nil {
				return time.Time{}, time.Time{}, fmt.Errorf("invalid 'from' time format (use RFC3339 or relative like '7d'): %w", err)
			}
			fromTime = toTime.Add(-duration)
		} else {
			fromTime = t
		}
	}

	return fromTime, toTime, nil
}

// parseRelativeDuration parses relative duration strings like "7d", "30d", "90d"
func parseRelativeDuration(s string) (time.Duration, error) {
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid relative duration format")
	}

	unit := s[len(s)-1]
	valueStr := s[:len(s)-1]

	var value int
	_, err := fmt.Sscanf(valueStr, "%d", &value)
	if err != nil {
		return 0, err
	}

	switch unit {
	case 'd':
		return time.Duration(value) * 24 * time.Hour, nil
	case 'w':
		return time.Duration(value) * 7 * 24 * time.Hour, nil
	case 'm':
		return time.Duration(value) * 30 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unknown duration unit '%c' (use d, w, m)", unit)
	}
}

// parseAndDisplay parses and displays SLO history
func (c *SLOHistoryCommand) parseAndDisplay(data []byte, obs *observability.Observability, displayMode string) error {
	// Parse JSON response
	var apiResponse struct {
		Data struct {
			FromTS  int64 `json:"from_ts"`
			ToTS    int64 `json:"to_ts"`
			Overall struct {
				SLIValue float64 `json:"sli_value"`
				Uptime   float64 `json:"uptime"`
				Target   float64 `json:"target"`
				ErrorBudget struct {
					Total     float64 `json:"total"`
					Used      float64 `json:"used"`
					Remaining float64 `json:"remaining"`
				} `json:"error_budget"`
			} `json:"overall"`
			Series struct {
				Times      []int64   `json:"times"`
				Values     []float64 `json:"values"`
			} `json:"series"`
		} `json:"data"`
		Meta struct {
			SLO struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"slo"`
		} `json:"meta"`
	}

	if err := json.Unmarshal(data, &apiResponse); err != nil {
		return fmt.Errorf("failed to parse API response: %w", err)
	}

	// Calculate error budget percentage
	errorBudgetPercent := 0.0
	if apiResponse.Data.Overall.ErrorBudget.Total > 0 {
		errorBudgetPercent = (apiResponse.Data.Overall.ErrorBudget.Remaining / apiResponse.Data.Overall.ErrorBudget.Total) * 100
	}

	// Determine compliance
	compliance := "COMPLIANT"
	if apiResponse.Data.Overall.SLIValue < apiResponse.Data.Overall.Target {
		compliance = "NON-COMPLIANT"
	}

	// Build series data
	series := make([]SLOHistorySeries, 0)
	for i, ts := range apiResponse.Data.Series.Times {
		if i < len(apiResponse.Data.Series.Values) {
			series = append(series, SLOHistorySeries{
				Timestamp: ts,
				SLI:       apiResponse.Data.Series.Values[i],
			})
		}
	}

	// Build history data
	history := SLOHistoryData{
		SLOID:           apiResponse.Meta.SLO.ID,
		SLOName:         apiResponse.Meta.SLO.Name,
		FromTS:          apiResponse.Data.FromTS,
		ToTS:            apiResponse.Data.ToTS,
		TargetThreshold: apiResponse.Data.Overall.Target,
		OverallSLI:      apiResponse.Data.Overall.SLIValue,
		OverallUptime:   apiResponse.Data.Overall.Uptime,
		ErrorBudget: ErrorBudgetHistory{
			Total:     apiResponse.Data.Overall.ErrorBudget.Total,
			Used:      apiResponse.Data.Overall.ErrorBudget.Used,
			Remaining: apiResponse.Data.Overall.ErrorBudget.Remaining,
			Percent:   errorBudgetPercent,
		},
		Series:     series,
		Compliance: compliance,
	}

	response := SLOHistoryQueryResponse{
		Status:  "success",
		History: history,
	}

	// Output results
	if c.jsonOut {
		output, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(output))
	} else {
		switch displayMode {
		case "query":
			c.displayFormatted(history)
		case "trends":
			c.displayTrends(history)
		case "report":
			c.displayReport(history)
		default:
			c.displayFormatted(history)
		}
	}

	return nil
}

// displayFormatted displays formatted history output
func (c *SLOHistoryCommand) displayFormatted(history SLOHistoryData) {
	fmt.Println("SLO Historical Data")
	fmt.Println("===================")
	fmt.Printf("SLO: %s\n", history.SLOName)
	fmt.Printf("SLO ID: %s\n", history.SLOID)
	fmt.Println()

	// Time range
	fromTime := time.Unix(history.FromTS, 0)
	toTime := time.Unix(history.ToTS, 0)
	days := int(toTime.Sub(fromTime).Hours() / 24)
	fmt.Printf("Time Period: %s to %s (%d days)\n",
		fromTime.Format("2006-01-02"),
		toTime.Format("2006-01-02"),
		days)
	fmt.Println()

	// Overall compliance
	complianceIcon := "✓"
	if history.Compliance == "NON-COMPLIANT" {
		complianceIcon = "✗"
	}
	fmt.Printf("Overall Compliance: %s %s\n", complianceIcon, history.Compliance)
	fmt.Printf("Target: %.2f%%\n", history.TargetThreshold)
	fmt.Printf("Actual SLI: %.2f%%\n", history.OverallSLI)
	if history.OverallUptime > 0 {
		fmt.Printf("Uptime: %.2f%%\n", history.OverallUptime)
	}
	fmt.Println()

	// Error budget
	fmt.Println("Error Budget:")
	fmt.Printf("  Total:     %.6f\n", history.ErrorBudget.Total)
	fmt.Printf("  Used:      %.6f (%.2f%%)\n", history.ErrorBudget.Used, 100-history.ErrorBudget.Percent)
	fmt.Printf("  Remaining: %.6f (%.2f%%)\n", history.ErrorBudget.Remaining, history.ErrorBudget.Percent)
	fmt.Println()

	// Data points
	if len(history.Series) > 0 {
		fmt.Printf("Data Points: %d time series measurements\n", len(history.Series))
	}
}

// displayTrends displays trend analysis
func (c *SLOHistoryCommand) displayTrends(history SLOHistoryData) {
	c.displayFormatted(history)

	fmt.Println()
	fmt.Println("Trend Analysis")
	fmt.Println("==============")

	// Calculate trend if we have series data
	if len(history.Series) >= 2 {
		first := history.Series[0].SLI
		last := history.Series[len(history.Series)-1].SLI
		change := last - first

		trendIcon := "→"
		trendText := "STABLE"
		if change > 0.1 {
			trendIcon = "↗"
			trendText = "IMPROVING"
		} else if change < -0.1 {
			trendIcon = "↘"
			trendText = "DEGRADING"
		}

		fmt.Printf("Trend: %s %s\n", trendIcon, trendText)
		fmt.Printf("Change: %.2f%% (from %.2f%% to %.2f%%)\n", change, first, last)
		fmt.Println()

		// Calculate volatility
		if len(history.Series) > 2 {
			var sum, sumSq float64
			for _, point := range history.Series {
				sum += point.SLI
				sumSq += point.SLI * point.SLI
			}
			mean := sum / float64(len(history.Series))
			variance := (sumSq / float64(len(history.Series))) - (mean * mean)
			stdDev := 0.0
			if variance > 0 {
				stdDev = variance // Approximation
			}

			volatilityText := "LOW"
			if stdDev > 0.5 {
				volatilityText = "HIGH"
			} else if stdDev > 0.2 {
				volatilityText = "MODERATE"
			}

			fmt.Printf("Volatility: %s\n", volatilityText)
			fmt.Printf("Standard Deviation: ~%.2f%%\n", stdDev)
		}
	}

	// Compliance summary
	fmt.Println()
	if history.Compliance == "COMPLIANT" {
		fmt.Println("✓ SLO maintained over the time period")
	} else {
		fmt.Println("✗ SLO violated during the time period")
		fmt.Printf("  Miss: %.2f%% (target was %.2f%%, actual %.2f%%)\n",
			history.TargetThreshold-history.OverallSLI,
			history.TargetThreshold,
			history.OverallSLI)
	}
}

// displayReport displays compliance report format
func (c *SLOHistoryCommand) displayReport(history SLOHistoryData) {
	fromTime := time.Unix(history.FromTS, 0)
	toTime := time.Unix(history.ToTS, 0)
	days := int(toTime.Sub(fromTime).Hours() / 24)

	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("                   SLO COMPLIANCE REPORT")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Printf("SLO Name:        %s\n", history.SLOName)
	fmt.Printf("SLO ID:          %s\n", history.SLOID)
	fmt.Printf("Report Period:   %s to %s\n", fromTime.Format("2006-01-02"), toTime.Format("2006-01-02"))
	fmt.Printf("Duration:        %d days\n", days)
	fmt.Printf("Generated:       %s\n", time.Now().Format("2006-01-02 15:04:05 MST"))
	fmt.Println()
	fmt.Println("───────────────────────────────────────────────────────────────")
	fmt.Println("COMPLIANCE SUMMARY")
	fmt.Println("───────────────────────────────────────────────────────────────")
	fmt.Println()

	complianceIcon := "✓ PASS"
	if history.Compliance == "NON-COMPLIANT" {
		complianceIcon = "✗ FAIL"
	}
	fmt.Printf("Overall Status:  %s\n", complianceIcon)
	fmt.Printf("Target SLO:      %.2f%%\n", history.TargetThreshold)
	fmt.Printf("Achieved SLI:    %.2f%%\n", history.OverallSLI)

	if history.OverallSLI >= history.TargetThreshold {
		delta := history.OverallSLI - history.TargetThreshold
		fmt.Printf("Margin:          +%.2f%% above target\n", delta)
	} else {
		delta := history.TargetThreshold - history.OverallSLI
		fmt.Printf("Shortfall:       -%.2f%% below target\n", delta)
	}

	fmt.Println()
	fmt.Println("───────────────────────────────────────────────────────────────")
	fmt.Println("ERROR BUDGET")
	fmt.Println("───────────────────────────────────────────────────────────────")
	fmt.Println()
	fmt.Printf("Allocated:       %.6f\n", history.ErrorBudget.Total)
	fmt.Printf("Consumed:        %.6f (%.2f%%)\n", history.ErrorBudget.Used, 100-history.ErrorBudget.Percent)
	fmt.Printf("Remaining:       %.6f (%.2f%%)\n", history.ErrorBudget.Remaining, history.ErrorBudget.Percent)

	budgetStatus := "Healthy"
	if history.ErrorBudget.Percent < 10 {
		budgetStatus = "Critical - Near exhaustion"
	} else if history.ErrorBudget.Percent < 25 {
		budgetStatus = "Warning - Running low"
	}
	fmt.Printf("Status:          %s\n", budgetStatus)

	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════════")
}

// Help displays help information
func (c *SLOHistoryCommand) Help() {
	help := `dd slo-history - Query SLO Historical Data

DESCRIPTION:
  Query historical SLO compliance data for trend analysis, reporting, and
  capacity planning. Analyze SLO performance over time to make informed
  decisions about SLO targets and system improvements.

USAGE:
  dd slo-history --action <action> --slo-id <id> [options]

ACTIONS:
  query            Query historical SLO data
  trends           Analyze SLO trends over time
  report           Generate compliance report

EXAMPLES:
  # Query last 30 days (default)
  dd slo-history --action query --slo-id "abc123"

  # Query last 7 days
  dd slo-history --action query --slo-id "abc123" --from 7d

  # Query last 90 days
  dd slo-history --action query --slo-id "abc123" --from 90d

  # Query specific time range
  dd slo-history --action query --slo-id "abc123" \
    --from "2026-01-01T00:00:00Z" \
    --to "2026-01-31T23:59:59Z"

  # Analyze trends
  dd slo-history --action trends --slo-id "abc123" --from 30d

  # Generate compliance report
  dd slo-history --action report --slo-id "abc123" --from 90d

  # Get JSON output
  dd slo-history --action query --slo-id "abc123" --json

OPTIONS:
  --action          Action to perform (query, trends, report)
  --slo-id          SLO ID to query (required)
  --from            Start time (RFC3339 or relative like '7d', '30d', '90d')
  --to              End time (RFC3339, defaults to now)
  --json            Output as JSON

TIME FORMATS:
  Relative:  7d (7 days ago)
             30d (30 days ago)
             90d (90 days ago)
             1w (1 week ago)
             1m (1 month ago - ~30 days)

  Absolute:  2026-01-01T00:00:00Z (RFC3339)

TREND ANALYSIS:
  IMPROVING:  SLI increased over time period
  STABLE:     SLI relatively constant
  DEGRADING:  SLI decreased over time period

  Volatility:
    LOW:      Consistent performance
    MODERATE: Some variation
    HIGH:     Significant fluctuation

COMPLIANCE STATES:
  COMPLIANT:     Actual SLI >= Target SLO
  NON-COMPLIANT: Actual SLI < Target SLO

USE CASES:
  Monthly SLO Reports:
    - Generate compliance reports for stakeholders
    - Document SLO achievement for management
    - Support quarterly business reviews

  Trend Analysis:
    - Identify long-term reliability trends
    - Detect degradation early
    - Validate system improvements

  SLO Tuning:
    - Determine if SLO targets are too strict or too loose
    - Analyze error budget consumption patterns
    - Adjust SLO targets based on historical data

  Capacity Planning:
    - Predict future reliability based on trends
    - Identify seasonal patterns
    - Plan infrastructure investments

  Post-Incident Analysis:
    - Understand impact of incidents on SLO
    - Validate corrections were applied correctly
    - Support post-mortem documentation

REPORT TYPES:
  Query:   Standard historical data display
  Trends:  Includes trend analysis and volatility
  Report:  Formal compliance report format

INTEGRATION WITH SLO MANAGEMENT:
  1. Define SLO (dd slos)
  2. Monitor real-time (existing monitoring)
  3. Apply corrections (dd slo-corrections)
  4. Check error budget (dd error-budgets)
  5. Analyze trends (dd slo-history) ← THIS COMMAND
  6. Make data-driven decisions

RECOMMENDED TIME RANGES:
  Weekly Review:     7d
  Monthly Report:    30d
  Quarterly Report:  90d
  Annual Review:     365d
  Trend Analysis:    90d minimum

NOTES:
  - Historical data retention depends on Datadog plan
  - Corrections affect historical calculations
  - Use consistent time windows for accurate trend analysis
  - Combine with error budget data for complete picture
  - Generate reports quarterly for stakeholder reviews

For more information: https://docs.datadoghq.com/api/latest/service-level-objectives/
`
	fmt.Println(help)
}
