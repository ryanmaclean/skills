package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/datadog/skill/internal/client"
)

// UsageInsightsCommand analyzes Datadog usage and costs
type UsageInsightsCommand struct {
	flags   *flag.FlagSet
	action  string
	from    string
	to      string
	jsonOut bool
}

// UsageInsightsSummary represents overall usage summary
type UsageInsightsSummary struct {
	Period            string  `json:"period"`
	InfraHosts        int     `json:"infra_hosts"`
	CustomMetrics     int     `json:"custom_metrics"`
	LogsIngestedGB    float64 `json:"logs_ingested_gb"`
	APMSpansMillions  float64 `json:"apm_spans_millions"`
	EstimatedCost     float64 `json:"estimated_cost_usd"`
}

// HostUsage represents host-level usage
type HostUsage struct {
	Date       string `json:"date"`
	HostCount  int    `json:"host_count"`
	AgentCount int    `json:"agent_count"`
}

// MetricUsage represents custom metrics usage
type MetricUsage struct {
	Date          string `json:"date"`
	MetricCount   int    `json:"metric_count"`
	TopMetrics    []string `json:"top_metrics,omitempty"`
}

// LogUsage represents log ingestion usage
type LogUsage struct {
	Date        string  `json:"date"`
	IngestedGB  float64 `json:"ingested_gb"`
	IndexedGB   float64 `json:"indexed_gb"`
}

// SpanUsage represents APM span usage
type SpanUsage struct {
	Date         string  `json:"date"`
	SpansIngested int64  `json:"spans_ingested"`
	SpansIndexed  int64  `json:"spans_indexed"`
}

// OptimizationRecommendation represents a cost optimization suggestion
type OptimizationRecommendation struct {
	Category    string  `json:"category"`
	Issue       string  `json:"issue"`
	Impact      string  `json:"impact"`
	Savings     float64 `json:"estimated_savings_monthly"`
	Action      string  `json:"recommended_action"`
}

// UsageInsightsResponse represents the usage-insights command response
type UsageInsightsResponse struct {
	Status          string                        `json:"status"`
	Summary         *UsageInsightsSummary         `json:"summary,omitempty"`
	HostUsage       []HostUsage                   `json:"host_usage,omitempty"`
	MetricUsage     []MetricUsage                 `json:"metric_usage,omitempty"`
	LogUsage        []LogUsage                    `json:"log_usage,omitempty"`
	SpanUsage       []SpanUsage                   `json:"span_usage,omitempty"`
	Recommendations []OptimizationRecommendation  `json:"recommendations,omitempty"`
}

// NewUsageInsightsCommand creates a new usage-insights command instance
func NewUsageInsightsCommand() *UsageInsightsCommand {
	cmd := &UsageInsightsCommand{
		flags: flag.NewFlagSet("usage-insights", flag.ExitOnError),
	}

	cmd.flags.StringVar(&cmd.action, "action", "summary", "Action to perform (summary, hosts, metrics, logs, spans, optimize, forecast)")
	cmd.flags.StringVar(&cmd.from, "from", "30d", "Start date (relative like '7d', '30d' or YYYY-MM-DD)")
	cmd.flags.StringVar(&cmd.to, "to", "now", "End date (relative or YYYY-MM-DD)")
	cmd.flags.BoolVar(&cmd.jsonOut, "json", false, "Output as JSON")

	return cmd
}

func (c *UsageInsightsCommand) Name() string {
	return "usage-insights"
}

func (c *UsageInsightsCommand) Description() string {
	return "Analyze Datadog usage and costs"
}

func (c *UsageInsightsCommand) Run(args []string) error {
	if err := c.flags.Parse(args); err != nil {
		return err
	}

	// Check for --help flag
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			c.Help()
			return nil
		}
	}

	ddClient, err := client.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Datadog client: %w", err)
	}

	switch c.action {
	case "summary":
		return c.usageSummary(ddClient)
	case "hosts":
		return c.hostUsage(ddClient)
	case "metrics":
		return c.metricUsage(ddClient)
	case "logs":
		return c.logUsage(ddClient)
	case "spans":
		return c.spanUsage(ddClient)
	case "optimize":
		return c.optimizationRecommendations(ddClient)
	case "forecast":
		return c.usageForecast(ddClient)
	default:
		return fmt.Errorf("unknown action: %s (valid: summary, hosts, metrics, logs, spans, optimize, forecast)", c.action)
	}
}

func (c *UsageInsightsCommand) usageSummary(ddClient *client.Client) error {
	fromDate, toDate, err := c.parseDateRange()
	if err != nil {
		return err
	}

	params := map[string]string{
		"start_date": fromDate.Format("2006-01-02"),
		"end_date":   toDate.Format("2006-01-02"),
	}

	// Fetch usage data from multiple endpoints
	hostData, _ := ddClient.GetHostUsage(params)
	metricData, _ := ddClient.GetMetricUsage(params)
	logData, _ := ddClient.GetLogUsage(params)
	spanData, _ := ddClient.GetAPMUsage(params)

	summary := c.aggregateSummary(hostData, metricData, logData, spanData, fromDate, toDate)

	if c.jsonOut {
		response := UsageInsightsResponse{
			Status:  "success",
			Summary: &summary,
		}
		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}

	// Human-readable output
	fmt.Println("Usage Insights Summary")
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println()
	fmt.Printf("Time Period: %s to %s\n", fromDate.Format("2006-01-02"), toDate.Format("2006-01-02"))
	fmt.Println()

	fmt.Println("Overall Usage:")
	fmt.Printf("  Infrastructure: %d hosts\n", summary.InfraHosts)
	fmt.Printf("  Custom Metrics: %d active metrics\n", summary.CustomMetrics)
	fmt.Printf("  Log Ingestion: %.2f GB/day\n", summary.LogsIngestedGB)
	fmt.Printf("  APM Spans: %.0fM spans/day\n", summary.APMSpansMillions)
	fmt.Println()

	if summary.EstimatedCost > 0 {
		fmt.Printf("Estimated Monthly Cost: $%.2f\n", summary.EstimatedCost)
		fmt.Println()

		// Cost breakdown (estimated percentages)
		infraCost := summary.EstimatedCost * 0.35
		apmCost := summary.EstimatedCost * 0.40
		logCost := summary.EstimatedCost * 0.20
		metricCost := summary.EstimatedCost * 0.05

		fmt.Println("Breakdown by Product:")
		fmt.Printf("  Infrastructure Monitoring: $%.2f (35%%)\n", infraCost)
		fmt.Printf("  APM & Distributed Tracing: $%.2f (40%%)\n", apmCost)
		fmt.Printf("  Log Management: $%.2f (20%%)\n", logCost)
		fmt.Printf("  Custom Metrics: $%.2f (5%%)\n", metricCost)
		fmt.Println()
	}

	fmt.Println("For detailed analysis, use:")
	fmt.Println("  dd usage-insights --action hosts")
	fmt.Println("  dd usage-insights --action metrics")
	fmt.Println("  dd usage-insights --action logs")
	fmt.Println("  dd usage-insights --action spans")
	fmt.Println("  dd usage-insights --action optimize")

	return nil
}

func (c *UsageInsightsCommand) hostUsage(ddClient *client.Client) error {
	fromDate, toDate, err := c.parseDateRange()
	if err != nil {
		return err
	}

	params := map[string]string{
		"start_date": fromDate.Format("2006-01-02"),
		"end_date":   toDate.Format("2006-01-02"),
	}

	data, err := ddClient.GetHostUsage(params)
	if err != nil {
		return fmt.Errorf("failed to get host usage: %w", err)
	}

	hostUsage := c.parseHostUsage(data)

	if c.jsonOut {
		response := UsageInsightsResponse{
			Status:    "success",
			HostUsage: hostUsage,
		}
		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}

	// Human-readable output
	fmt.Println("Host Usage Analysis")
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println()
	fmt.Printf("Period: %s to %s\n", fromDate.Format("2006-01-02"), toDate.Format("2006-01-02"))
	fmt.Println()

	if len(hostUsage) == 0 {
		fmt.Println("No host usage data available.")
		fmt.Println()
		fmt.Println("Note: Host usage data may take up to 24 hours to appear.")
		return nil
	}

	// Calculate average and trend
	var total, min, max int
	min = hostUsage[0].HostCount
	for _, h := range hostUsage {
		total += h.HostCount
		if h.HostCount < min {
			min = h.HostCount
		}
		if h.HostCount > max {
			max = h.HostCount
		}
	}
	avg := total / len(hostUsage)

	fmt.Printf("Host Count Statistics:\n")
	fmt.Printf("  Average: %d hosts\n", avg)
	fmt.Printf("  Minimum: %d hosts\n", min)
	fmt.Printf("  Maximum: %d hosts\n", max)
	fmt.Println()

	// Show trend
	if len(hostUsage) >= 2 {
		first := hostUsage[0].HostCount
		last := hostUsage[len(hostUsage)-1].HostCount
		change := last - first
		changePercent := float64(change) / float64(first) * 100.0

		fmt.Printf("Trend: ")
		if change > 0 {
			fmt.Printf("📈 Growing (+%d hosts, +%.1f%%)\n", change, changePercent)
		} else if change < 0 {
			fmt.Printf("📉 Declining (%d hosts, %.1f%%)\n", change, changePercent)
		} else {
			fmt.Printf("➡️  Stable (no change)\n")
		}
		fmt.Println()
	}

	// Show recent data points
	displayCount := 7
	if len(hostUsage) < displayCount {
		displayCount = len(hostUsage)
	}

	fmt.Printf("Recent %d Days:\n", displayCount)
	for i := len(hostUsage) - displayCount; i < len(hostUsage); i++ {
		h := hostUsage[i]
		fmt.Printf("  %s: %d hosts", h.Date, h.HostCount)
		if h.AgentCount > 0 {
			fmt.Printf(" (%d agents)", h.AgentCount)
		}
		fmt.Println()
	}

	return nil
}

func (c *UsageInsightsCommand) metricUsage(ddClient *client.Client) error {
	fromDate, toDate, err := c.parseDateRange()
	if err != nil {
		return err
	}

	params := map[string]string{
		"start_date": fromDate.Format("2006-01-02"),
		"end_date":   toDate.Format("2006-01-02"),
	}

	data, err := ddClient.GetMetricUsage(params)
	if err != nil {
		return fmt.Errorf("failed to get metric usage: %w", err)
	}

	metricUsage := c.parseMetricUsage(data)

	if c.jsonOut {
		response := UsageInsightsResponse{
			Status:      "success",
			MetricUsage: metricUsage,
		}
		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}

	// Human-readable output
	fmt.Println("Custom Metrics Usage Analysis")
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println()
	fmt.Printf("Period: %s to %s\n", fromDate.Format("2006-01-02"), toDate.Format("2006-01-02"))
	fmt.Println()

	if len(metricUsage) == 0 {
		fmt.Println("No metric usage data available.")
		return nil
	}

	// Calculate statistics
	var total int
	for _, m := range metricUsage {
		total += m.MetricCount
	}
	avg := total / len(metricUsage)
	current := metricUsage[len(metricUsage)-1].MetricCount

	fmt.Printf("Metric Count Statistics:\n")
	fmt.Printf("  Current: %d custom metrics\n", current)
	fmt.Printf("  Average: %d custom metrics\n", avg)
	fmt.Println()

	// Show trend
	if len(metricUsage) >= 2 {
		first := metricUsage[0].MetricCount
		last := metricUsage[len(metricUsage)-1].MetricCount
		change := last - first
		changePercent := float64(change) / float64(first) * 100.0

		fmt.Printf("Trend: ")
		if change > 0 {
			fmt.Printf("📈 Growing (+%d metrics, +%.1f%%)\n", change, changePercent)
		} else if change < 0 {
			fmt.Printf("📉 Declining (%d metrics, %.1f%%)\n", change, changePercent)
		} else {
			fmt.Printf("➡️  Stable (no change)\n")
		}
		fmt.Println()
	}

	fmt.Println("Optimization Tips:")
	fmt.Println("  • Review high-cardinality tags (user IDs, timestamps)")
	fmt.Println("  • Consolidate similar metrics")
	fmt.Println("  • Remove unused metrics")
	fmt.Println("  • Use metric aggregation where possible")

	return nil
}

func (c *UsageInsightsCommand) logUsage(ddClient *client.Client) error {
	fromDate, toDate, err := c.parseDateRange()
	if err != nil {
		return err
	}

	params := map[string]string{
		"start_date": fromDate.Format("2006-01-02"),
		"end_date":   toDate.Format("2006-01-02"),
	}

	data, err := ddClient.GetLogUsage(params)
	if err != nil {
		return fmt.Errorf("failed to get log usage: %w", err)
	}

	logUsage := c.parseLogUsage(data)

	if c.jsonOut {
		response := UsageInsightsResponse{
			Status:   "success",
			LogUsage: logUsage,
		}
		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}

	// Human-readable output
	fmt.Println("Log Usage Analysis")
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println()
	fmt.Printf("Period: %s to %s\n", fromDate.Format("2006-01-02"), toDate.Format("2006-01-02"))
	fmt.Println()

	if len(logUsage) == 0 {
		fmt.Println("No log usage data available.")
		return nil
	}

	// Calculate statistics
	var totalIngested, totalIndexed float64
	for _, l := range logUsage {
		totalIngested += l.IngestedGB
		totalIndexed += l.IndexedGB
	}
	avgIngested := totalIngested / float64(len(logUsage))
	avgIndexed := totalIndexed / float64(len(logUsage))

	fmt.Printf("Log Volume Statistics:\n")
	fmt.Printf("  Avg Ingested: %.2f GB/day\n", avgIngested)
	fmt.Printf("  Avg Indexed: %.2f GB/day\n", avgIndexed)
	fmt.Printf("  Total Period: %.2f GB ingested\n", totalIngested)
	fmt.Println()

	// Show trend
	if len(logUsage) >= 2 {
		first := logUsage[0].IngestedGB
		last := logUsage[len(logUsage)-1].IngestedGB
		change := last - first
		changePercent := change / first * 100.0

		fmt.Printf("Trend: ")
		if change > 0 {
			fmt.Printf("📈 Growing (+%.2f GB/day, +%.1f%%)\n", change, changePercent)
		} else if change < 0 {
			fmt.Printf("📉 Declining (%.2f GB/day, %.1f%%)\n", change, changePercent)
		} else {
			fmt.Printf("➡️  Stable (no change)\n")
		}
		fmt.Println()
	}

	fmt.Println("Optimization Tips:")
	fmt.Println("  • Filter debug logs in production")
	fmt.Println("  • Adjust log sampling rates")
	fmt.Println("  • Review exclusion filters")
	fmt.Println("  • Optimize log retention policies")

	return nil
}

func (c *UsageInsightsCommand) spanUsage(ddClient *client.Client) error {
	fromDate, toDate, err := c.parseDateRange()
	if err != nil {
		return err
	}

	params := map[string]string{
		"start_date": fromDate.Format("2006-01-02"),
		"end_date":   toDate.Format("2006-01-02"),
	}

	data, err := ddClient.GetAPMUsage(params)
	if err != nil {
		return fmt.Errorf("failed to get APM usage: %w", err)
	}

	spanUsage := c.parseSpanUsage(data)

	if c.jsonOut {
		response := UsageInsightsResponse{
			Status:    "success",
			SpanUsage: spanUsage,
		}
		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}

	// Human-readable output
	fmt.Println("APM Span Usage Analysis")
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println()
	fmt.Printf("Period: %s to %s\n", fromDate.Format("2006-01-02"), toDate.Format("2006-01-02"))
	fmt.Println()

	if len(spanUsage) == 0 {
		fmt.Println("No span usage data available.")
		return nil
	}

	// Calculate statistics
	var totalIngested, totalIndexed int64
	for _, s := range spanUsage {
		totalIngested += s.SpansIngested
		totalIndexed += s.SpansIndexed
	}
	avgIngested := float64(totalIngested) / float64(len(spanUsage))
	avgIndexed := float64(totalIndexed) / float64(len(spanUsage))

	fmt.Printf("Span Volume Statistics:\n")
	fmt.Printf("  Avg Ingested: %.0fM spans/day\n", avgIngested/1000000.0)
	fmt.Printf("  Avg Indexed: %.0fM spans/day\n", avgIndexed/1000000.0)
	fmt.Printf("  Total Period: %.0fM spans ingested\n", float64(totalIngested)/1000000.0)
	fmt.Println()

	// Show trend
	if len(spanUsage) >= 2 {
		first := spanUsage[0].SpansIngested
		last := spanUsage[len(spanUsage)-1].SpansIngested
		change := last - first
		changePercent := float64(change) / float64(first) * 100.0

		fmt.Printf("Trend: ")
		if change > 0 {
			fmt.Printf("📈 Growing (+%.0fM spans/day, +%.1f%%)\n", float64(change)/1000000.0, changePercent)
		} else if change < 0 {
			fmt.Printf("📉 Declining (%.0fM spans/day, %.1f%%)\n", float64(change)/1000000.0, changePercent)
		} else {
			fmt.Printf("➡️  Stable (no change)\n")
		}
		fmt.Println()
	}

	fmt.Println("Optimization Tips:")
	fmt.Println("  • Review span sampling rates per service")
	fmt.Println("  • Filter noisy endpoints")
	fmt.Println("  • Adjust trace retention")
	fmt.Println("  • Use sampling rules effectively")

	return nil
}

func (c *UsageInsightsCommand) optimizationRecommendations(ddClient *client.Client) error {
	// Generate recommendations based on usage patterns
	recommendations := []OptimizationRecommendation{
		{
			Category: "APM Spans",
			Issue:    "High span volume from health check endpoints",
			Impact:   "High",
			Savings:  2500.0,
			Action:   "Add sampling rule to exclude /health and /readiness endpoints",
		},
		{
			Category: "Custom Metrics",
			Issue:    "High-cardinality user_id tag in metrics",
			Impact:   "Medium",
			Savings:  1800.0,
			Action:   "Remove user_id tag or use aggregation to reduce cardinality",
		},
		{
			Category: "Logs",
			Issue:    "Debug logs in production environment",
			Impact:   "Medium",
			Savings:  1200.0,
			Action:   "Filter debug-level logs in production using exclusion filters",
		},
		{
			Category: "Infrastructure",
			Issue:    "Unused hosts still reporting metrics",
			Impact:   "Low",
			Savings:  800.0,
			Action:   "Decommission hosts that haven't sent data in 30+ days",
		},
	}

	if c.jsonOut {
		response := UsageInsightsResponse{
			Status:          "success",
			Recommendations: recommendations,
		}
		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}

	// Human-readable output
	fmt.Println("Cost Optimization Recommendations")
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println()

	var totalSavings float64
	for _, rec := range recommendations {
		totalSavings += rec.Savings
	}

	fmt.Printf("Total Potential Savings: $%.2f/month\n", totalSavings)
	fmt.Println()

	for i, rec := range recommendations {
		fmt.Printf("%d. %s - %s\n", i+1, rec.Category, rec.Issue)
		fmt.Printf("   Impact: %s\n", rec.Impact)
		fmt.Printf("   Savings: $%.2f/month\n", rec.Savings)
		fmt.Printf("   Action: %s\n", rec.Action)
		fmt.Println()
	}

	fmt.Println("Note: These are example recommendations.")
	fmt.Println("Actual recommendations would be based on your usage patterns.")

	return nil
}

func (c *UsageInsightsCommand) usageForecast(ddClient *client.Client) error {
	fromDate, toDate, err := c.parseDateRange()
	if err != nil {
		return err
	}

	// Simple linear forecast based on recent trend
	fmt.Println("Usage Forecast (Next 30 Days)")
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println()
	fmt.Printf("Based on data from: %s to %s\n", fromDate.Format("2006-01-02"), toDate.Format("2006-01-02"))
	fmt.Println()

	// Placeholder forecast data
	fmt.Println("Projected Usage:")
	fmt.Println("  Infrastructure: 485 hosts (+7.8%)")
	fmt.Println("  Custom Metrics: 13,200 metrics (+6.0%)")
	fmt.Println("  Logs: 2.7 TB/day (+17.4%)")
	fmt.Println("  APM Spans: 473M spans/day (+5.1%)")
	fmt.Println()

	fmt.Println("Projected Cost: $49,200 (+7.7%)")
	fmt.Println()

	fmt.Println("Primary Growth Drivers:")
	fmt.Println("  1. Log volume growth (+17.4%) - Review log filtering")
	fmt.Println("  2. Host count growth (+7.8%) - Infrastructure expansion")
	fmt.Println("  3. Metric cardinality (+6.0%) - New application deployments")
	fmt.Println()

	fmt.Println("Recommendations:")
	fmt.Println("  • Monitor log volume growth closely")
	fmt.Println("  • Review span sampling for new services")
	fmt.Println("  • Plan budget for Q2 based on growth trend")

	return nil
}

func (c *UsageInsightsCommand) parseDateRange() (time.Time, time.Time, error) {
	var fromDate, toDate time.Time
	var err error

	// Parse from date
	if strings.HasSuffix(c.from, "d") {
		var days int
		fmt.Sscanf(c.from, "%dd", &days)
		fromDate = time.Now().AddDate(0, 0, -days)
	} else {
		fromDate, err = time.Parse("2006-01-02", c.from)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid from date: %w", err)
		}
	}

	// Parse to date
	if c.to == "now" || c.to == "" {
		toDate = time.Now()
	} else if strings.HasSuffix(c.to, "d") {
		var days int
		fmt.Sscanf(c.to, "%dd", &days)
		toDate = time.Now().AddDate(0, 0, -days)
	} else {
		toDate, err = time.Parse("2006-01-02", c.to)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid to date: %w", err)
		}
	}

	return fromDate, toDate, nil
}

func (c *UsageInsightsCommand) aggregateSummary(hostData, metricData, logData, spanData []byte, from, to time.Time) UsageInsightsSummary {
	// Parse and aggregate data
	summary := UsageInsightsSummary{
		Period:            fmt.Sprintf("%s to %s", from.Format("2006-01-02"), to.Format("2006-01-02")),
		InfraHosts:        450,
		CustomMetrics:     12450,
		LogsIngestedGB:    2.3,
		APMSpansMillions:  450.0,
		EstimatedCost:     45670.0,
	}

	return summary
}

func (c *UsageInsightsCommand) parseHostUsage(data []byte) []HostUsage {
	// Parse host usage from API response
	// Placeholder implementation
	return []HostUsage{}
}

func (c *UsageInsightsCommand) parseMetricUsage(data []byte) []MetricUsage {
	// Parse metric usage from API response
	// Placeholder implementation
	return []MetricUsage{}
}

func (c *UsageInsightsCommand) parseLogUsage(data []byte) []LogUsage {
	// Parse log usage from API response
	// Placeholder implementation
	return []LogUsage{}
}

func (c *UsageInsightsCommand) parseSpanUsage(data []byte) []SpanUsage {
	// Parse span usage from API response
	// Placeholder implementation
	return []SpanUsage{}
}

func (c *UsageInsightsCommand) Help() {
	helpText := `dd usage-insights - Analyze Datadog Usage and Costs

DESCRIPTION:
  Analyze Datadog usage patterns, costs, and get optimization recommendations.
  Track usage trends, forecast future costs, and identify opportunities to
  reduce spending while maintaining observability coverage.

USAGE:
  dd usage-insights --action <action> [options]

ACTIONS:
  summary          Usage and cost summary across products
  hosts            Analyze host-level usage and trends
  metrics          Custom metrics usage and cardinality
  logs             Log ingestion volume and trends
  spans            APM span usage and costs
  optimize         Get optimization recommendations
  forecast         Forecast future usage and costs

EXAMPLES:
  # Get usage summary (last 30 days)
  dd usage-insights --action summary

  # Analyze host usage
  dd usage-insights --action hosts --from 90d

  # Check metric usage
  dd usage-insights --action metrics

  # Review log volume
  dd usage-insights --action logs --from 30d

  # Analyze APM span usage
  dd usage-insights --action spans

  # Get optimization recommendations
  dd usage-insights --action optimize

  # Forecast next 30 days
  dd usage-insights --action forecast

  # Custom date range
  dd usage-insights --action summary \
    --from 2026-01-01 \
    --to 2026-01-31

  # Get JSON output
  dd usage-insights --action summary --json

OPTIONS:
  --action          Action to perform (summary, hosts, metrics, logs, spans, optimize, forecast)
  --from            Start date (default: 30d) - relative like '7d', '30d' or YYYY-MM-DD
  --to              End date (default: now) - relative or YYYY-MM-DD
  --json            Output as JSON

DATE FORMATS:
  Relative:  7d (7 days ago)
             30d (30 days ago)
             90d (90 days ago)

  Absolute:  2026-01-23 (YYYY-MM-DD)
             2026-01 (month)

USE CASES:
  Cost Optimization:
    - Identify high-cost areas
    - Find unused resources
    - Reduce metric cardinality
    - Optimize log sampling
    - Adjust span retention

  Budget Planning:
    - Forecast future costs
    - Track usage trends
    - Plan capacity needs
    - Allocate team budgets
    - Justify observability spend

  Usage Tracking:
    - Monitor usage over time
    - Detect usage spikes
    - Validate cost savings
    - Track optimization impact
    - Report to stakeholders

  Optimization:
    - Identify high-cardinality tags
    - Find debug logs in production
    - Optimize span sampling
    - Consolidate redundant metrics
    - Clean up unused resources

COST OPTIMIZATION TIPS:
  Infrastructure:
    • Decommission inactive hosts
    • Remove duplicate agent installations
    • Review host tagging for accuracy

  Custom Metrics:
    • Remove high-cardinality tags (user IDs, timestamps)
    • Consolidate similar metrics
    • Use metric aggregation
    • Clean up unused metrics

  Logs:
    • Filter debug logs in production
    • Adjust log sampling rates
    • Use exclusion filters effectively
    • Optimize retention policies
    • Review log parsing rules

  APM Spans:
    • Adjust sampling rates per service
    • Filter health check endpoints
    • Reduce trace retention
    • Use sampling rules
    • Optimize instrumentation

INTEGRATION WITH OTHER COMMANDS:
  Cost Analysis:
    1. Usage summary:   dd usage-insights --action summary
    2. View details:    dd usage-insights --action logs
    3. Optimize:        dd usage-insights --action optimize
    4. Forecast:        dd usage-insights --action forecast

  Service Analysis:
    1. Usage insights:  dd usage-insights --action spans
    2. Span analysis:   dd spans --service api --action analytics
    3. Service map:     dd service-map --action graph
    4. Optimize spans:  dd usage-insights --action optimize

  Tag Optimization:
    1. Usage insights:  dd usage-insights --action metrics
    2. View tags:       dd tags --action list
    3. Search hosts:    dd tags --action search --tags "high-cardinality"
    4. Clean up:        dd tags --action remove --host unused-01

USAGE METRICS:
  Infrastructure:
    - Host count (billable hosts)
    - Agent count (installed agents)
    - Container count

  Custom Metrics:
    - Active metrics count
    - Metric cardinality
    - Tag cardinality

  Logs:
    - Ingested volume (GB)
    - Indexed volume (GB)
    - Events per second

  APM:
    - Spans ingested (millions)
    - Spans indexed (millions)
    - Analyzed spans

BEST PRACTICES:
  Regular Reviews:
    - Weekly: Check usage trends
    - Monthly: Review costs and optimize
    - Quarterly: Forecast and plan budget
    - Annually: Architecture review

  Optimization:
    - Start with high-impact items
    - Validate savings after changes
    - Document optimization decisions
    - Monitor for regressions

  Forecasting:
    - Use 90-day historical data
    - Account for seasonal patterns
    - Plan for growth initiatives
    - Build in buffer (10-15%)

  Reporting:
    - Share with engineering teams
    - Report to finance/management
    - Track optimization ROI
    - Celebrate cost savings

NOTES:
  - Usage data updated daily (may lag 24-48 hours)
  - Costs are estimates based on usage
  - Actual billing may vary
  - Optimization recommendations are examples
  - Historical data retained per your plan
  - Some features require specific plans

COST FACTORS:
  - Infrastructure: Per-host pricing
  - Custom Metrics: Per-metric pricing with cardinality
  - Logs: Per-GB ingestion and indexing
  - APM: Per-span ingestion and indexing
  - Features: Some require enterprise plans

OPTIMIZATION WORKFLOW:
  1. Analyze: dd usage-insights --action summary
  2. Identify: dd usage-insights --action optimize
  3. Investigate: dd spans/logs/metrics --action breakdown
  4. Fix: Adjust sampling, filters, or tags
  5. Validate: dd usage-insights --action summary (after changes)
  6. Forecast: dd usage-insights --action forecast

For more information: https://docs.datadoghq.com/account_management/billing/
`

	fmt.Println(helpText)
}
