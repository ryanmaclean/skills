package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/datadog/skill/internal/client"
	"github.com/datadog/skill/internal/observability"
)

// CostCommand analyzes Datadog usage and costs for FinOps optimization
type CostCommand struct {
	flags     *flag.FlagSet
	duration  string
	breakdown string
	json      bool
}

// UsageAPIResponse represents the generic usage API response structure
type UsageAPIResponse struct {
	Usage []map[string]interface{} `json:"usage"`
}

// UsageData represents usage metrics for a specific product
type UsageData struct {
	IngestedSpansBytes int64   `json:"ingested_spans_bytes,omitempty"`
	IngestedSpansGB    float64 `json:"ingested_spans_gb,omitempty"`
	IndexedSpans       int64   `json:"indexed_spans,omitempty"`
	AvgHosts           int     `json:"avg_hosts,omitempty"`
	IngestedBytes      int64   `json:"ingested_bytes,omitempty"`
	IngestedGB         float64 `json:"ingested_gb,omitempty"`
	IndexedBytes       int64   `json:"indexed_bytes,omitempty"`
	IndexedGB          float64 `json:"indexed_gb,omitempty"`
	AvgContainers      int     `json:"avg_containers,omitempty"`
	AvgCustomMetrics   int     `json:"avg_custom_metrics,omitempty"`
	AvgSyntheticsTests int     `json:"avg_synthetics_tests,omitempty"`
	EstimatedCostUSD   struct {
		IndexedSpans float64 `json:"indexed_spans,omitempty"`
		Ingested     float64 `json:"ingested,omitempty"`
		Indexed      float64 `json:"indexed,omitempty"`
		Hosts        float64 `json:"hosts,omitempty"`
		Containers   float64 `json:"containers,omitempty"`
		Total        float64 `json:"total"`
	} `json:"estimated_cost_usd"`
}

// Recommendation represents a cost optimization recommendation
type Recommendation struct {
	Category           string  `json:"category"`
	Priority           string  `json:"priority"`
	Issue              string  `json:"issue"`
	Detail             string  `json:"detail"`
	Recommendation     string  `json:"recommendation"`
	PotentialSavingsUSD float64 `json:"potential_savings_usd"`
}

// CostOutput represents the structured cost analysis output
type CostOutput struct {
	Status         string `json:"status"`
	AnalysisPeriod struct {
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
		Duration  string `json:"duration"`
	} `json:"analysis_period"`
	UsageSummary UsageSummary `json:"usage_summary"`
	CostSummary struct {
		TotalEstimatedMonthlyUSD    float64 `json:"total_estimated_monthly_usd"`
		PotentialSavingsUSD         float64 `json:"potential_savings_usd"`
		OptimizationOpportunityPct  float64 `json:"optimization_opportunity_pct"`
	} `json:"cost_summary"`
	TopConsumers    []Consumer       `json:"top_consumers"`
	Recommendations []Recommendation `json:"recommendations"`
	NextSteps       []string         `json:"next_steps"`
}

// Consumer represents a top cost consumer
type Consumer struct {
	Type     string  `json:"type"`
	Name     string  `json:"name"`
	Value    float64 `json:"value"`
	CostUSD  float64 `json:"cost_usd"`
	Percent  float64 `json:"percent"`
}

// NewCostCommand creates a new cost analysis command
func NewCostCommand() *CostCommand {
	cmd := &CostCommand{
		flags: flag.NewFlagSet("cost", flag.ExitOnError),
	}

	cmd.flags.StringVar(&cmd.duration, "duration", "30d", "Time range: 24h, 7d, 30d")
	cmd.flags.StringVar(&cmd.breakdown, "breakdown", "service", "Breakdown by: service, host, env, all")
	cmd.flags.BoolVar(&cmd.json, "json", false, "Output as JSON")

	return cmd
}

// Name returns the command name
func (c *CostCommand) Name() string {
	return "cost"
}

// Description returns the command description
func (c *CostCommand) Description() string {
	return "Analyze Datadog usage and costs for FinOps optimization"
}

// Run executes the cost analysis command
func (c *CostCommand) Run(args []string) error {
	// Initialize observability
	obs, err := observability.Init("analyze-usage-cost", "production")
	if err != nil {
		return fmt.Errorf("failed to init observability: %w", err)
	}
	defer obs.Shutdown(0)

	// Parse flags
	if err := c.flags.Parse(args); err != nil {
		return err
	}

	obs.LogInfo(fmt.Sprintf("Analyzing Datadog usage and costs for %s", c.duration))

	// Calculate date range
	span := obs.StartSpan("calculate_dates")
	daysAgo, err := c.parseDuration(c.duration)
	if err != nil {
		obs.FinishSpan(span)
		obs.LogError(fmt.Sprintf("Invalid duration: %s", err.Error()))
		return err
	}
	startDate, endDate := c.calculateDateRange(daysAgo)
	obs.FinishSpan(span)

	obs.LogInfo(fmt.Sprintf("Date range: %s to %s", startDate, endDate))

	// Create Datadog client
	span = obs.StartSpan("create_client")
	ddClient, err := client.NewClient()
	obs.FinishSpan(span)

	if err != nil {
		obs.LogError(fmt.Sprintf("Failed to create Datadog client: %s", err.Error()))
		return fmt.Errorf("failed to create Datadog client: %w", err)
	}

	// Initialize output structure
	output := &CostOutput{
		Status: "ok",
	}
	output.AnalysisPeriod.StartDate = startDate
	output.AnalysisPeriod.EndDate = endDate
	output.AnalysisPeriod.Duration = c.duration

	// Query usage metrics for each product
	var totalCost float64

	// Query APM usage
	apmData, err := c.queryAPMUsage(ddClient, obs, startDate, endDate)
	if err == nil {
		output.UsageSummary.APM = apmData
		totalCost += apmData.EstimatedCostUSD.Total
		obs.LogInfo(fmt.Sprintf("APM: %d indexed spans, est. cost: $%.2f", apmData.IndexedSpans, apmData.EstimatedCostUSD.Total))
	}

	// Query Logs usage
	logsData, err := c.queryLogsUsage(ddClient, obs, startDate, endDate)
	if err == nil {
		output.UsageSummary.Logs = logsData
		totalCost += logsData.EstimatedCostUSD.Total
		obs.LogInfo(fmt.Sprintf("Logs: %.1fGB ingested, est. cost: $%.2f", logsData.IngestedGB, logsData.EstimatedCostUSD.Total))
	}

	// Query Infrastructure usage
	infraData, err := c.queryInfrastructureUsage(ddClient, obs, startDate, endDate)
	if err == nil {
		output.UsageSummary.Infrastructure = infraData
		totalCost += infraData.EstimatedCostUSD.Total
		obs.LogInfo(fmt.Sprintf("Infrastructure: %d hosts, %d containers, est. cost: $%.2f", infraData.AvgHosts, infraData.AvgContainers, infraData.EstimatedCostUSD.Total))
	}

	// Query Custom Metrics usage
	metricsData, err := c.queryMetricsUsage(ddClient, obs, startDate, endDate)
	if err == nil {
		output.UsageSummary.CustomMetrics = metricsData
		totalCost += metricsData.EstimatedCostUSD.Total
		obs.LogInfo(fmt.Sprintf("Custom metrics: %d, est. cost: $%.2f", metricsData.AvgCustomMetrics, metricsData.EstimatedCostUSD.Total))
	}

	// Calculate cost summary
	span = obs.StartSpan("calculate_totals")
	output.CostSummary.TotalEstimatedMonthlyUSD = roundFloat(totalCost, 2)
	obs.GetMetrics().Gauge("usage.total_cost", totalCost)
	obs.FinishSpan(span)

	// Generate recommendations
	span = obs.StartSpan("generate_recommendations")
	recommendations := c.generateRecommendations(
		output.UsageSummary.APM,
		output.UsageSummary.Logs,
		output.UsageSummary.Infrastructure,
		output.UsageSummary.CustomMetrics,
	)
	output.Recommendations = recommendations

	potentialSavings := 0.0
	for _, rec := range recommendations {
		potentialSavings += rec.PotentialSavingsUSD
	}
	output.CostSummary.PotentialSavingsUSD = roundFloat(potentialSavings, 2)

	if totalCost > 0 {
		output.CostSummary.OptimizationOpportunityPct = roundFloat((potentialSavings/totalCost)*100, 1)
	}
	obs.FinishSpan(span)

	// Identify top consumers
	span = obs.StartSpan("identify_top_consumers")
	topConsumers := c.identifyTopConsumers(output.UsageSummary, totalCost)
	output.TopConsumers = topConsumers
	obs.FinishSpan(span)

	// Add next steps
	output.NextSteps = []string{
		"Review high-priority recommendations for immediate cost reduction",
		"Implement APM sampling for high-volume services",
		"Configure log exclusion filters for noisy patterns",
		"Audit and remove unused custom metrics",
		"Set up cost anomaly alerts in Datadog",
	}

	// Record results as tags
	obs.GetMetrics().Gauge("cost.total_monthly", totalCost)
	obs.GetMetrics().Gauge("cost.potential_savings", potentialSavings)

	// Output results
	if c.json {
		jsonData, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	} else {
		c.printFormatted(output)
	}

	obs.LogInfo(fmt.Sprintf("Analysis completed: $%.2f cost, $%.2f potential savings", totalCost, potentialSavings))
	return nil
}

// parseDuration parses duration string to days
func (c *CostCommand) parseDuration(duration string) (int, error) {
	switch duration {
	case "24h", "1d":
		return 1, nil
	case "7d":
		return 7, nil
	case "30d":
		return 30, nil
	case "90d":
		return 90, nil
	default:
		return 0, fmt.Errorf("invalid duration: %s (use: 24h, 7d, 30d, 90d)", duration)
	}
}

// calculateDateRange calculates start and end dates for usage API
func (c *CostCommand) calculateDateRange(daysAgo int) (string, string) {
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -daysAgo)

	return startDate.Format("2006-01-02"), endDate.Format("2006-01-02")
}

// queryAPMUsage queries APM usage metrics
func (c *CostCommand) queryAPMUsage(ddClient *client.Client, obs *observability.Observability, startDate, endDate string) (UsageData, error) {
	span := obs.StartSpan("query_apm_usage")
	defer obs.FinishSpan(span)

	endpoint := fmt.Sprintf("/api/v1/usage/traces?start_hr=%sT00&end_hr=%sT00", startDate, endDate)

	start := time.Now()
	responseData, err := ddClient.DoRequest("GET", endpoint, nil)
	apiDuration := time.Since(start).Milliseconds()

	if err != nil {
		obs.RecordAPICall(endpoint, "GET", 500, float64(apiDuration), err)
		return UsageData{}, err
	}

	obs.RecordAPICall(endpoint, "GET", 200, float64(apiDuration), nil)

	var response UsageAPIResponse
	if err := json.Unmarshal(responseData, &response); err != nil {
		return UsageData{}, err
	}

	data := UsageData{}

	if len(response.Usage) > 0 {
		var ingestedBytes int64
		var indexedSpans int64

		for _, usage := range response.Usage {
			if val, ok := usage["ingested_events_bytes"].(float64); ok {
				ingestedBytes += int64(val)
			}
			if val, ok := usage["indexed_events_count"].(float64); ok {
				indexedSpans += int64(val)
			}
		}

		ingestedGB := float64(ingestedBytes) / (1024 * 1024 * 1024)

		// Estimate costs (2026 pricing: $1.70 per million indexed spans)
		spanCost := (float64(indexedSpans) / 1_000_000) * 1.70

		data.IngestedSpansBytes = ingestedBytes
		data.IngestedSpansGB = roundFloat(ingestedGB, 2)
		data.IndexedSpans = indexedSpans
		data.EstimatedCostUSD.IndexedSpans = roundFloat(spanCost, 2)
		data.EstimatedCostUSD.Total = roundFloat(spanCost, 2)
	}

	return data, nil
}

// queryLogsUsage queries logs usage metrics
func (c *CostCommand) queryLogsUsage(ddClient *client.Client, obs *observability.Observability, startDate, endDate string) (UsageData, error) {
	span := obs.StartSpan("query_logs_usage")
	defer obs.FinishSpan(span)

	endpoint := fmt.Sprintf("/api/v1/usage/logs?start_hr=%sT00&end_hr=%sT00", startDate, endDate)

	start := time.Now()
	responseData, err := ddClient.DoRequest("GET", endpoint, nil)
	apiDuration := time.Since(start).Milliseconds()

	if err != nil {
		obs.RecordAPICall(endpoint, "GET", 500, float64(apiDuration), err)
		return UsageData{}, err
	}

	obs.RecordAPICall(endpoint, "GET", 200, float64(apiDuration), nil)

	var response UsageAPIResponse
	if err := json.Unmarshal(responseData, &response); err != nil {
		return UsageData{}, err
	}

	data := UsageData{}

	if len(response.Usage) > 0 {
		var ingestedBytes int64
		var indexedBytes int64

		for _, usage := range response.Usage {
			if val, ok := usage["ingested_events_bytes"].(float64); ok {
				ingestedBytes += int64(val)
			}
			if val, ok := usage["indexed_events_count"].(float64); ok {
				indexedBytes += int64(val)
			}
		}

		ingestedGB := float64(ingestedBytes) / (1024 * 1024 * 1024)
		indexedGB := float64(indexedBytes) / (1024 * 1024 * 1024)

		// Estimate costs (Ingested: $0.10/GB, Indexed: $0.10/GB)
		ingestedCost := ingestedGB * 0.10
		indexedCost := indexedGB * 0.10
		totalCost := ingestedCost + indexedCost

		data.IngestedBytes = ingestedBytes
		data.IngestedGB = roundFloat(ingestedGB, 2)
		data.IndexedBytes = indexedBytes
		data.IndexedGB = roundFloat(indexedGB, 2)
		data.EstimatedCostUSD.Ingested = roundFloat(ingestedCost, 2)
		data.EstimatedCostUSD.Indexed = roundFloat(indexedCost, 2)
		data.EstimatedCostUSD.Total = roundFloat(totalCost, 2)
	}

	return data, nil
}

// queryInfrastructureUsage queries infrastructure usage metrics
func (c *CostCommand) queryInfrastructureUsage(ddClient *client.Client, obs *observability.Observability, startDate, endDate string) (UsageData, error) {
	span := obs.StartSpan("query_infra_usage")
	defer obs.FinishSpan(span)

	endpoint := fmt.Sprintf("/api/v1/usage/hosts?start_hr=%sT00&end_hr=%sT00", startDate, endDate)

	start := time.Now()
	responseData, err := ddClient.DoRequest("GET", endpoint, nil)
	apiDuration := time.Since(start).Milliseconds()

	if err != nil {
		obs.RecordAPICall(endpoint, "GET", 500, float64(apiDuration), err)
		return UsageData{}, err
	}

	obs.RecordAPICall(endpoint, "GET", 200, float64(apiDuration), nil)

	var response UsageAPIResponse
	if err := json.Unmarshal(responseData, &response); err != nil {
		return UsageData{}, err
	}

	data := UsageData{}

	if len(response.Usage) > 0 {
		var hostCounts []int
		var containerCounts []int

		for _, usage := range response.Usage {
			if val, ok := usage["host_count"].(float64); ok && val > 0 {
				hostCounts = append(hostCounts, int(val))
			}
			if val, ok := usage["container_count"].(float64); ok && val > 0 {
				containerCounts = append(containerCounts, int(val))
			}
		}

		avgHosts := 0
		avgContainers := 0

		if len(hostCounts) > 0 {
			sum := 0
			for _, count := range hostCounts {
				sum += count
			}
			avgHosts = sum / len(hostCounts)
		}

		if len(containerCounts) > 0 {
			sum := 0
			for _, count := range containerCounts {
				sum += count
			}
			avgContainers = sum / len(containerCounts)
		}

		// Estimate costs ($15/host/month, $1/container/month)
		hostCost := float64(avgHosts) * 15
		containerCost := float64(avgContainers) * 1
		totalCost := hostCost + containerCost

		data.AvgHosts = avgHosts
		data.AvgContainers = avgContainers
		data.EstimatedCostUSD.Hosts = roundFloat(hostCost, 2)
		data.EstimatedCostUSD.Containers = roundFloat(containerCost, 2)
		data.EstimatedCostUSD.Total = roundFloat(totalCost, 2)
	}

	return data, nil
}

// queryMetricsUsage queries custom metrics usage
func (c *CostCommand) queryMetricsUsage(ddClient *client.Client, obs *observability.Observability, startDate, endDate string) (UsageData, error) {
	span := obs.StartSpan("query_metrics_usage")
	defer obs.FinishSpan(span)

	endpoint := fmt.Sprintf("/api/v1/usage/timeseries?start_hr=%sT00&end_hr=%sT00", startDate, endDate)

	start := time.Now()
	responseData, err := ddClient.DoRequest("GET", endpoint, nil)
	apiDuration := time.Since(start).Milliseconds()

	if err != nil {
		obs.RecordAPICall(endpoint, "GET", 500, float64(apiDuration), err)
		return UsageData{}, err
	}

	obs.RecordAPICall(endpoint, "GET", 200, float64(apiDuration), nil)

	var response UsageAPIResponse
	if err := json.Unmarshal(responseData, &response); err != nil {
		return UsageData{}, err
	}

	data := UsageData{}

	if len(response.Usage) > 0 {
		var metricCounts []int

		for _, usage := range response.Usage {
			if val, ok := usage["num_custom_timeseries"].(float64); ok && val > 0 {
				metricCounts = append(metricCounts, int(val))
			}
		}

		avgMetrics := 0
		if len(metricCounts) > 0 {
			sum := 0
			for _, count := range metricCounts {
				sum += count
			}
			avgMetrics = sum / len(metricCounts)
		}

		// Estimate cost ($0.05 per metric per month)
		metricsCost := float64(avgMetrics) * 0.05

		data.AvgCustomMetrics = avgMetrics
		data.EstimatedCostUSD.Total = roundFloat(metricsCost, 2)
	}

	return data, nil
}

// generateRecommendations generates cost optimization recommendations
func (c *CostCommand) generateRecommendations(apmData, logsData, infraData, metricsData UsageData) []Recommendation {
	var recommendations []Recommendation

	// APM recommendations
	if apmData.IngestedSpansBytes > 0 && apmData.IndexedSpans > 0 {
		retentionRate := float64(apmData.IndexedSpans) / (float64(apmData.IngestedSpansBytes) / 1000)
		if retentionRate > 0.15 {
			recommendations = append(recommendations, Recommendation{
				Category:   "apm",
				Priority:   "high",
				Issue:      "High APM span retention rate",
				Detail:     fmt.Sprintf("Retaining %.1f%% of ingested spans. Industry standard is 10-15%%.", retentionRate*100),
				Recommendation: "Review tag-based retention filters to reduce indexed spans. Focus on high-value traces (errors, slow requests) and sample normal traffic.",
				PotentialSavingsUSD: roundFloat(apmData.EstimatedCostUSD.Total*0.30, 2),
			})
		}
	}

	if apmData.AvgHosts > 10 {
		recommendations = append(recommendations, Recommendation{
			Category:   "apm",
			Priority:   "medium",
			Issue:      fmt.Sprintf("APM enabled on %d hosts", apmData.AvgHosts),
			Detail:     "Verify all hosts require APM monitoring.",
			Recommendation: "Disable APM on non-production hosts, batch jobs, and internal services.",
			PotentialSavingsUSD: roundFloat(apmData.EstimatedCostUSD.Total*0.20, 2),
		})
	}

	// Logs recommendations
	if logsData.IngestedGB > 100 {
		recommendations = append(recommendations, Recommendation{
			Category:   "logs",
			Priority:   "high",
			Issue:      "High log ingestion volume",
			Detail:     fmt.Sprintf("Ingesting %.1fGB of logs.", logsData.IngestedGB),
			Recommendation: "Implement log filtering at source. Exclude debug logs, health checks, and high-frequency events.",
			PotentialSavingsUSD: roundFloat(logsData.EstimatedCostUSD.Total*0.40, 2),
		})
	}

	if logsData.IndexedBytes > 0 && logsData.IngestedBytes > 0 {
		indexedRatio := float64(logsData.IndexedBytes) / float64(logsData.IngestedBytes)
		if indexedRatio > 0.30 {
			recommendations = append(recommendations, Recommendation{
				Category:   "logs",
				Priority:   "medium",
				Issue:      "High log indexing rate",
				Detail:     fmt.Sprintf("Indexing %.1f%% of ingested logs.", indexedRatio*100),
				Recommendation: "Review log indexing rules. Index only logs needed for search/alerting.",
				PotentialSavingsUSD: roundFloat(logsData.EstimatedCostUSD.Total*0.25, 2),
			})
		}
	}

	// Infrastructure recommendations
	if infraData.AvgContainers > 50 {
		recommendations = append(recommendations, Recommendation{
			Category:   "infrastructure",
			Priority:   "medium",
			Issue:      "High container count",
			Detail:     fmt.Sprintf("Monitoring %d containers on average.", infraData.AvgContainers),
			Recommendation: "Exclude ephemeral and test containers. Use container exclusion rules.",
			PotentialSavingsUSD: roundFloat(infraData.EstimatedCostUSD.Total*0.15, 2),
		})
	}

	// Custom metrics recommendations
	if metricsData.AvgCustomMetrics > 100 {
		recommendations = append(recommendations, Recommendation{
			Category:   "metrics",
			Priority:   "low",
			Issue:      "High custom metrics count",
			Detail:     fmt.Sprintf("Using %d custom metrics.", metricsData.AvgCustomMetrics),
			Recommendation: "Audit custom metrics for unused or redundant metrics. Use metric tags instead of separate metrics.",
			PotentialSavingsUSD: roundFloat(metricsData.EstimatedCostUSD.Total*0.10, 2),
		})
	}

	// General recommendation
	recommendations = append(recommendations, Recommendation{
		Category:   "general",
		Priority:   "info",
		Issue:      "Cost visibility",
		Detail:     "Regular cost analysis enables proactive optimization.",
		Recommendation: "Schedule weekly cost reviews. Set up usage alerts for anomaly detection.",
		PotentialSavingsUSD: 0,
	})

	return recommendations
}

// UsageSummary represents the usage summary structure
type UsageSummary struct {
	APM            UsageData
	Logs           UsageData
	Infrastructure UsageData
	CustomMetrics  UsageData
	Synthetics     UsageData
}

// identifyTopConsumers identifies top cost consumers
func (c *CostCommand) identifyTopConsumers(usage UsageSummary, totalCost float64) []Consumer {

	var consumers []Consumer

	if usage.APM.EstimatedCostUSD.Total > 0 {
		consumers = append(consumers, Consumer{
			Type:    "APM",
			Name:    "Indexed Spans",
			Value:   float64(usage.APM.IndexedSpans),
			CostUSD: usage.APM.EstimatedCostUSD.Total,
			Percent: roundFloat((usage.APM.EstimatedCostUSD.Total/totalCost)*100, 1),
		})
	}

	if usage.Logs.EstimatedCostUSD.Total > 0 {
		consumers = append(consumers, Consumer{
			Type:    "Logs",
			Name:    "Ingested Logs",
			Value:   usage.Logs.IngestedGB,
			CostUSD: usage.Logs.EstimatedCostUSD.Total,
			Percent: roundFloat((usage.Logs.EstimatedCostUSD.Total/totalCost)*100, 1),
		})
	}

	if usage.Infrastructure.EstimatedCostUSD.Total > 0 {
		consumers = append(consumers, Consumer{
			Type:    "Infrastructure",
			Name:    fmt.Sprintf("%d hosts, %d containers", usage.Infrastructure.AvgHosts, usage.Infrastructure.AvgContainers),
			Value:   float64(usage.Infrastructure.AvgHosts + usage.Infrastructure.AvgContainers),
			CostUSD: usage.Infrastructure.EstimatedCostUSD.Total,
			Percent: roundFloat((usage.Infrastructure.EstimatedCostUSD.Total/totalCost)*100, 1),
		})
	}

	if usage.CustomMetrics.EstimatedCostUSD.Total > 0 {
		consumers = append(consumers, Consumer{
			Type:    "Custom Metrics",
			Name:    "Custom Timeseries",
			Value:   float64(usage.CustomMetrics.AvgCustomMetrics),
			CostUSD: usage.CustomMetrics.EstimatedCostUSD.Total,
			Percent: roundFloat((usage.CustomMetrics.EstimatedCostUSD.Total/totalCost)*100, 1),
		})
	}

	// Sort by cost descending
	sort.Slice(consumers, func(i, j int) bool {
		return consumers[i].CostUSD > consumers[j].CostUSD
	})

	// Return top 5
	if len(consumers) > 5 {
		return consumers[:5]
	}
	return consumers
}

// printFormatted prints the cost analysis output in a conversational format
func (c *CostCommand) printFormatted(output *CostOutput) {
	fmt.Println("Datadog Usage & Cost Analysis")
	fmt.Printf("Period: %s to %s (%s)\n", output.AnalysisPeriod.StartDate, output.AnalysisPeriod.EndDate, output.AnalysisPeriod.Duration)
	fmt.Println()

	fmt.Printf("Total estimated monthly cost: $%.2f\n", output.CostSummary.TotalEstimatedMonthlyUSD)
	fmt.Printf("Potential savings: $%.2f (%.1f%%)\n", output.CostSummary.PotentialSavingsUSD, output.CostSummary.OptimizationOpportunityPct)
	fmt.Println()

	// Top consumers
	if len(output.TopConsumers) > 0 {
		fmt.Println("Top Cost Consumers:")
		for i, consumer := range output.TopConsumers {
			fmt.Printf("  %d. %s - %s: $%.2f (%.1f%%)\n", i+1, consumer.Type, consumer.Name, consumer.CostUSD, consumer.Percent)
		}
		fmt.Println()
	}

	// Usage breakdown
	fmt.Println("Usage Summary:")
	if output.UsageSummary.APM.IndexedSpans > 0 {
		fmt.Printf("  APM: %s indexed spans, %.2fGB ingested - $%.2f\n",
			formatNumber(output.UsageSummary.APM.IndexedSpans),
			output.UsageSummary.APM.IngestedSpansGB,
			output.UsageSummary.APM.EstimatedCostUSD.Total)
	}
	if output.UsageSummary.Logs.IngestedGB > 0 {
		fmt.Printf("  Logs: %.2fGB ingested, %.2fGB indexed - $%.2f\n",
			output.UsageSummary.Logs.IngestedGB,
			output.UsageSummary.Logs.IndexedGB,
			output.UsageSummary.Logs.EstimatedCostUSD.Total)
	}
	if output.UsageSummary.Infrastructure.AvgHosts > 0 {
		fmt.Printf("  Infrastructure: %d hosts, %d containers - $%.2f\n",
			output.UsageSummary.Infrastructure.AvgHosts,
			output.UsageSummary.Infrastructure.AvgContainers,
			output.UsageSummary.Infrastructure.EstimatedCostUSD.Total)
	}
	if output.UsageSummary.CustomMetrics.AvgCustomMetrics > 0 {
		fmt.Printf("  Custom Metrics: %d timeseries - $%.2f\n",
			output.UsageSummary.CustomMetrics.AvgCustomMetrics,
			output.UsageSummary.CustomMetrics.EstimatedCostUSD.Total)
	}
	fmt.Println()

	// High priority recommendations
	fmt.Println("Top Recommendations:")
	count := 0
	for _, rec := range output.Recommendations {
		if rec.Priority == "high" || rec.Priority == "critical" {
			fmt.Printf("  [%s] %s\n", strings.ToUpper(rec.Priority), rec.Issue)
			fmt.Printf("    %s\n", rec.Detail)
			fmt.Printf("    Recommendation: %s\n", rec.Recommendation)
			if rec.PotentialSavingsUSD > 0 {
				fmt.Printf("    Potential savings: $%.2f\n", rec.PotentialSavingsUSD)
			}
			fmt.Println()
			count++
			if count >= 3 {
				break
			}
		}
	}

	// Next steps
	if len(output.NextSteps) > 0 {
		fmt.Println("Next Steps:")
		for _, step := range output.NextSteps {
			fmt.Printf("  - %s\n", step)
		}
	}
}

// Help prints the help message
func (c *CostCommand) Help() {
	fmt.Println("Usage: dd cost [options]")
	fmt.Println()
	fmt.Println("Analyze Datadog usage and costs for FinOps optimization.")
	fmt.Println("Provides usage metrics, cost estimates, and optimization recommendations.")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --duration string")
	fmt.Println("        Time range: 24h, 7d, 30d, 90d (default: 30d)")
	fmt.Println("  --breakdown string")
	fmt.Println("        Breakdown by: service, host, env, all (default: service)")
	fmt.Println("  --json")
	fmt.Println("        Output as JSON")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  dd cost")
	fmt.Println("  dd cost --duration 7d")
	fmt.Println("  dd cost --duration 30d --json")
	fmt.Println("  dd cost --breakdown all")
	fmt.Println()
	fmt.Println("Cost Estimates:")
	fmt.Println("  APM: $1.70 per million indexed spans")
	fmt.Println("  Logs: $0.10 per GB ingested + $0.10 per GB indexed")
	fmt.Println("  Infrastructure: $15 per host/month + $1 per container/month")
	fmt.Println("  Custom Metrics: $0.05 per metric/month")
}
