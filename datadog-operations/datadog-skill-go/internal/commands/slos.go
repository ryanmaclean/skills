package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/datadog/skill/internal/client"
	"github.com/datadog/skill/internal/context"
	"github.com/datadog/skill/internal/observability"
)

// SLOsCommand queries Datadog SLOs
type SLOsCommand struct {
	flags   *flag.FlagSet
	service string
	tags    string
	jsonOut bool
}

// SLOData represents a single SLO
type SLOData struct {
	ID                   string   `json:"id"`
	Name                 string   `json:"name"`
	Type                 string   `json:"type"`
	CurrentValue         float64  `json:"current_value"`
	Target               float64  `json:"target"`
	Warning              float64  `json:"warning"`
	ErrorBudgetRemaining float64  `json:"error_budget_remaining"`
	Timeframe            string   `json:"timeframe"`
	Tags                 []string `json:"tags"`
	Status               string   `json:"status"`
	BudgetStatus         string   `json:"budget_status"`
}

// SLOsOutput represents the structured output
type SLOsOutput struct {
	Status    string                 `json:"status"`
	TotalSLOs int                    `json:"total_slos"`
	Summary   *SLOsSummary           `json:"summary,omitempty"`
	SLOs      []SLOData              `json:"slos"`
	RawData   map[string]interface{} `json:"raw_data,omitempty"`
}

// SLOsSummary contains summary statistics
type SLOsSummary struct {
	Breaching       int `json:"breaching"`
	Warning         int `json:"warning"`
	OK              int `json:"ok"`
	BudgetExhausted int `json:"budget_exhausted"`
	BudgetLow       int `json:"budget_low"`
}

// SLOResponse represents the API response structure
type SLOResponse struct {
	Data []struct {
		ID         string   `json:"id"`
		Name       string   `json:"name"`
		Type       string   `json:"type"`
		Tags       []string `json:"tags"`
		Thresholds []struct {
			Timeframe      string  `json:"timeframe"`
			Target         float64 `json:"target"`
			TargetDisplay  string  `json:"target_display"`
			Warning        float64 `json:"warning"`
			WarningDisplay string  `json:"warning_display"`
		} `json:"thresholds"`
		SLIValue *float64 `json:"sli_value,omitempty"`
	} `json:"data"`
}

// SLOHistoryResponse represents the SLO history API response
type SLOHistoryResponse struct {
	Data struct {
		Overall struct {
			SLIValue             *float64               `json:"sli_value,omitempty"`
			Uptime               *float64               `json:"uptime,omitempty"`
			History              [][]interface{}        `json:"history,omitempty"`
			ErrorBudgetRemaining map[string]interface{} `json:"error_budget_remaining,omitempty"`
		} `json:"overall"`
	} `json:"data"`
}

// NewSLOsCommand creates a new SLOs command
func NewSLOsCommand() *SLOsCommand {
	cmd := &SLOsCommand{
		flags: flag.NewFlagSet("slos", flag.ExitOnError),
	}

	cmd.flags.StringVar(&cmd.service, "service", "", "Filter by service tag (auto-detect if not provided)")
	cmd.flags.StringVar(&cmd.tags, "tags", "", "Additional tags to filter by (comma-separated)")
	cmd.flags.BoolVar(&cmd.jsonOut, "json", false, "Output as JSON")

	return cmd
}

// Name returns the command name
func (c *SLOsCommand) Name() string {
	return "slos"
}

// Description returns the command description
func (c *SLOsCommand) Description() string {
	return "Query Datadog SLOs and check error budgets"
}

// Run executes the SLOs command
func (c *SLOsCommand) Run(args []string) error {
	// Initialize observability
	obs, err := observability.Init("query-slos", "production")
	if err != nil {
		return fmt.Errorf("failed to init observability: %w", err)
	}
	defer obs.Shutdown(0)

	// Parse flags
	if err := c.flags.Parse(args); err != nil {
		return err
	}

	obs.LogInfo("Starting SLO query")

	// Auto-detect service if needed
	serviceName := c.service
	if serviceName == "" {
		span := obs.StartSpan("detect_context")
		obs.LogInfo("Auto-detecting service name")

		ctx, err := context.DetectContext(".")
		obs.FinishSpan(span)

		if err == nil && ctx.ServiceName != "" {
			serviceName = ctx.ServiceName
			obs.LogInfo(fmt.Sprintf("Auto-detected service: %s", serviceName))
		}
	}

	// Parse tags
	var tagsList []string
	if c.tags != "" {
		tagsList = strings.Split(c.tags, ",")
		for i := range tagsList {
			tagsList[i] = strings.TrimSpace(tagsList[i])
		}
	}

	// Add service tag if provided
	if serviceName != "" {
		tagsList = append(tagsList, fmt.Sprintf("service:%s", serviceName))
	}

	// Create Datadog client
	span := obs.StartSpan("create_client")
	ddClient, err := client.NewClient()
	obs.FinishSpan(span)

	if err != nil {
		obs.LogError(fmt.Sprintf("Failed to create Datadog client: %s", err.Error()))
		return fmt.Errorf("failed to create Datadog client: %w", err)
	}

	// Query SLOs
	span = obs.StartSpan("query_slos")
	if serviceName != "" {
		obs.GetTracer().SetTag(span, "service", serviceName)
	}
	if len(tagsList) > 0 {
		obs.GetTracer().SetTag(span, "tags", strings.Join(tagsList, ","))
	}

	start := time.Now()
	responseData, err := ddClient.GetSLOs(tagsList)
	apiDuration := time.Since(start).Milliseconds()
	obs.FinishSpan(span)

	if err != nil {
		obs.RecordAPICall("/api/v1/slo", "GET", 500, float64(apiDuration), err)
		obs.LogError(fmt.Sprintf("API call failed: %s", err.Error()))
		return fmt.Errorf("failed to query SLOs: %w", err)
	}

	obs.RecordAPICall("/api/v1/slo", "GET", 200, float64(apiDuration), nil)

	// Parse results
	span = obs.StartSpan("parse_results")
	output, err := c.parseResults(responseData, ddClient, obs)
	obs.FinishSpan(span)

	if err != nil {
		obs.LogError(fmt.Sprintf("Failed to parse results: %s", err.Error()))
		return fmt.Errorf("failed to parse results: %w", err)
	}

	// Handle no data case
	if output.TotalSLOs == 0 {
		obs.LogWarning("No SLOs found")
		obs.GetMetrics().Gauge("slos.total", 0)

		if c.jsonOut {
			jsonData, _ := json.MarshalIndent(output, "", "  ")
			fmt.Println(string(jsonData))
		} else {
			fmt.Println("No SLOs found for the specified criteria")
			if serviceName != "" {
				fmt.Printf("Service: %s\n", serviceName)
			}
		}

		return nil
	}

	// Record metrics
	obs.GetMetrics().Gauge("slos.total", float64(output.TotalSLOs))
	obs.GetMetrics().Gauge("slos.breaching", float64(output.Summary.Breaching))
	obs.GetMetrics().Gauge("slos.budget_exhausted", float64(output.Summary.BudgetExhausted))
	obs.LogInfo(fmt.Sprintf("Found %d SLOs, %d breaching", output.TotalSLOs, output.Summary.Breaching))

	// Output
	if c.jsonOut {
		jsonData, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	} else {
		c.printFormatted(serviceName, output)
	}

	obs.LogInfo("SLO query completed")
	return nil
}

// parseResults parses the SLO API response
func (c *SLOsCommand) parseResults(data []byte, ddClient *client.Client, obs *observability.Observability) (*SLOsOutput, error) {
	var response SLOResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	output := &SLOsOutput{
		Status:    "ok",
		TotalSLOs: len(response.Data),
		SLOs:      []SLOData{},
		Summary: &SLOsSummary{
			Breaching:       0,
			Warning:         0,
			OK:              0,
			BudgetExhausted: 0,
			BudgetLow:       0,
		},
	}

	if output.TotalSLOs == 0 {
		return output, nil
	}

	// Process each SLO
	for _, sloItem := range response.Data {
		slo := c.processSLO(sloItem, ddClient, obs)
		output.SLOs = append(output.SLOs, slo)

		// Update summary
		switch slo.Status {
		case "breaching":
			output.Summary.Breaching++
		case "warning":
			output.Summary.Warning++
		case "ok":
			output.Summary.OK++
		}

		switch slo.BudgetStatus {
		case "exhausted":
			output.Summary.BudgetExhausted++
		case "low":
			output.Summary.BudgetLow++
		}
	}

	// Determine overall status
	if output.Summary.Breaching > 0 {
		output.Status = "breaching"
	} else if output.Summary.Warning > 0 {
		output.Status = "warning"
	} else {
		output.Status = "ok"
	}

	// Sort SLOs by status (breaching first)
	sort.Slice(output.SLOs, func(i, j int) bool {
		statusOrder := map[string]int{"breaching": 0, "warning": 1, "ok": 2}
		return statusOrder[output.SLOs[i].Status] < statusOrder[output.SLOs[j].Status]
	})

	return output, nil
}

// processSLO processes a single SLO and enriches it with history data
func (c *SLOsCommand) processSLO(sloItem struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	Tags       []string `json:"tags"`
	Thresholds []struct {
		Timeframe      string  `json:"timeframe"`
		Target         float64 `json:"target"`
		TargetDisplay  string  `json:"target_display"`
		Warning        float64 `json:"warning"`
		WarningDisplay string  `json:"warning_display"`
	} `json:"thresholds"`
	SLIValue *float64 `json:"sli_value,omitempty"`
}, ddClient *client.Client, obs *observability.Observability) SLOData {
	slo := SLOData{
		ID:   sloItem.ID,
		Name: sloItem.Name,
		Type: sloItem.Type,
		Tags: sloItem.Tags,
	}

	// Get primary threshold (usually the first one, or longest timeframe)
	if len(sloItem.Thresholds) > 0 {
		threshold := sloItem.Thresholds[0]
		slo.Target = threshold.Target * 100
		slo.Warning = threshold.Warning * 100
		slo.Timeframe = threshold.Timeframe
	}

	// Get current value
	if sloItem.SLIValue != nil {
		slo.CurrentValue = *sloItem.SLIValue * 100
	}

	// Get SLO history for error budget calculation
	historySpan := obs.StartSpan("get_slo_history")
	obs.GetTracer().SetTag(historySpan, "slo.id", sloItem.ID)

	// Query last 30 days
	toTime := time.Now()
	fromTime := toTime.Add(-30 * 24 * time.Hour)

	historyData, err := ddClient.GetSLOHistory(sloItem.ID, fromTime, toTime)
	obs.FinishSpan(historySpan)

	if err == nil {
		var historyResp SLOHistoryResponse
		if err := json.Unmarshal(historyData, &historyResp); err == nil {
			// Update current value from history if not already set
			if slo.CurrentValue == 0 && historyResp.Data.Overall.SLIValue != nil {
				slo.CurrentValue = *historyResp.Data.Overall.SLIValue * 100
			}
			if slo.CurrentValue == 0 && historyResp.Data.Overall.Uptime != nil {
				slo.CurrentValue = *historyResp.Data.Overall.Uptime * 100
			}

			// Calculate error budget remaining
			if historyResp.Data.Overall.ErrorBudgetRemaining != nil {
				if remaining, ok := historyResp.Data.Overall.ErrorBudgetRemaining["remaining"].(float64); ok {
					slo.ErrorBudgetRemaining = remaining
				}
			} else {
				// Calculate manually if not provided
				slo.ErrorBudgetRemaining = c.calculateErrorBudget(slo.CurrentValue, slo.Target)
			}
		}
	}

	// Calculate status
	slo.Status = c.calculateStatus(slo.CurrentValue, slo.Target, slo.Warning)
	slo.BudgetStatus = c.calculateBudgetStatus(slo.ErrorBudgetRemaining)

	return slo
}

// calculateErrorBudget calculates error budget remaining percentage
func (c *SLOsCommand) calculateErrorBudget(currentValue, target float64) float64 {
	if target == 0 {
		return 0
	}

	// Error budget = (1 - target) * 100
	// Consumed = (target - current)
	// Remaining = Error budget - Consumed
	errorBudget := (100 - target)
	consumed := target - currentValue
	remaining := errorBudget - consumed

	// Convert to percentage of original error budget
	if errorBudget > 0 {
		return math.Round((remaining / errorBudget) * 100)
	}

	return 0
}

// calculateStatus calculates SLO status based on thresholds
func (c *SLOsCommand) calculateStatus(current, target, warning float64) string {
	if current < target {
		return "breaching"
	} else if warning > 0 && current < warning {
		return "warning"
	}
	return "ok"
}

// calculateBudgetStatus calculates error budget status
func (c *SLOsCommand) calculateBudgetStatus(remaining float64) string {
	if remaining <= 0 {
		return "exhausted"
	} else if remaining < 20 {
		return "low"
	}
	return "healthy"
}

// printFormatted prints the SLOs response in a conversational format
func (c *SLOsCommand) printFormatted(serviceName string, output *SLOsOutput) {
	fmt.Println("SLO Status Report")
	if serviceName != "" {
		fmt.Printf("Service: %s\n", serviceName)
	}
	fmt.Println()
	fmt.Printf("Found %d SLOs\n", output.TotalSLOs)
	fmt.Println()
	fmt.Println("Status breakdown:")
	fmt.Printf("  Breaching: %d\n", output.Summary.Breaching)
	fmt.Printf("  Warning: %d\n", output.Summary.Warning)
	fmt.Printf("  OK: %d\n", output.Summary.OK)
	fmt.Println()
	fmt.Println("Error budget status:")
	fmt.Printf("  Exhausted: %d\n", output.Summary.BudgetExhausted)
	fmt.Printf("  Low (<20%%): %d\n", output.Summary.BudgetLow)

	// Show breaching SLOs
	if output.Summary.Breaching > 0 {
		fmt.Println()
		fmt.Printf("ALERT: %d SLO(s) breaching target\n", output.Summary.Breaching)
		fmt.Println()
		fmt.Println("Breaching SLOs:")
		for _, slo := range output.SLOs {
			if slo.Status == "breaching" {
				fmt.Printf("  %s\n", slo.Name)
				fmt.Printf("    Current: %.2f%% | Target: %.2f%%\n", slo.CurrentValue, slo.Target)
				fmt.Printf("    Error budget: %.0f%%\n", slo.ErrorBudgetRemaining)
				if slo.Timeframe != "" {
					fmt.Printf("    Timeframe: %s\n", slo.Timeframe)
				}
			}
		}
	}

	// Show budget exhausted
	if output.Summary.BudgetExhausted > 0 {
		fmt.Println()
		fmt.Printf("WARNING: %d SLO(s) have exhausted error budget\n", output.Summary.BudgetExhausted)
		for _, slo := range output.SLOs {
			if slo.BudgetStatus == "exhausted" {
				fmt.Printf("  %s (%.2f%% remaining)\n", slo.Name, slo.ErrorBudgetRemaining)
			}
		}
	}

	// Show all SLOs in table format
	if output.TotalSLOs > 0 && (output.Summary.Breaching == 0 || output.TotalSLOs > output.Summary.Breaching) {
		fmt.Println()
		fmt.Println("All SLOs:")
		fmt.Printf("%-50s %-10s %-10s %-12s %-15s\n", "Name", "Current", "Target", "Status", "Error Budget")
		fmt.Println(strings.Repeat("-", 100))
		for _, slo := range output.SLOs {
			statusSymbol := ""
			switch slo.Status {
			case "breaching":
				statusSymbol = "✗"
			case "warning":
				statusSymbol = "⚠"
			case "ok":
				statusSymbol = "✓"
			}

			name := slo.Name
			if len(name) > 47 {
				name = name[:47] + "..."
			}

			fmt.Printf("%-50s %-10s %-10s %-12s %-15s\n",
				name,
				fmt.Sprintf("%.2f%%", slo.CurrentValue),
				fmt.Sprintf("%.2f%%", slo.Target),
				fmt.Sprintf("%s %s", statusSymbol, slo.Status),
				fmt.Sprintf("%.0f%%", slo.ErrorBudgetRemaining),
			)
		}
	}
}

// Help prints the help message
func (c *SLOsCommand) Help() {
	fmt.Println("Usage: dd slos [options]")
	fmt.Println()
	fmt.Println("Query Datadog SLOs and check error budgets")
	fmt.Println()
	fmt.Println("Options:")
	c.flags.PrintDefaults()
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  dd slos")
	fmt.Println("  dd slos --service my-service")
	fmt.Println("  dd slos --tags team:backend,env:prod")
	fmt.Println("  dd slos --json")
}
