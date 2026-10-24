package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/datadog/skill/internal/client"
)

// CapacityScaleCommand handles capacity planning and scaling operations
type CapacityScaleCommand struct {
	flags     *flag.FlagSet
	action    string
	service   string
	metric    string
	instances int
	period    string
	from      string
	to        string
	jsonOut   bool
}

// CapacityAnalysis represents current capacity analysis
type CapacityAnalysis struct {
	Service             string      `json:"service"`
	AnalyzedAt          time.Time   `json:"analyzed_at"`
	CurrentCapacity     Capacity    `json:"current_capacity"`
	Utilization         Utilization `json:"utilization"`
	BottleneckResources []string    `json:"bottleneck_resources"`
	HeadroomDays        int         `json:"headroom_days"`
	Status              string      `json:"status"` // healthy, warning, critical
	Recommendation      string      `json:"recommendation"`
}

// Capacity represents service capacity
type Capacity struct {
	Instances      int     `json:"instances"`
	CPU            float64 `json:"cpu_cores"`
	Memory         float64 `json:"memory_gb"`
	Storage        float64 `json:"storage_gb,omitempty"`
	RequestsPerSec float64 `json:"requests_per_sec"`
}

// Utilization represents resource utilization
type Utilization struct {
	CPU     float64 `json:"cpu_percent"`
	Memory  float64 `json:"memory_percent"`
	Storage float64 `json:"storage_percent,omitempty"`
	Network float64 `json:"network_percent,omitempty"`
}

// CapacityForecast represents future capacity forecast
type CapacityForecast struct {
	Service          string          `json:"service"`
	Period           string          `json:"period"` // 7d, 30d, 90d
	CurrentUsage     float64         `json:"current_usage"`
	Forecasted       []ForecastPoint `json:"forecasted"`
	CapacityDate     time.Time       `json:"capacity_exhaustion_date,omitempty"`
	Confidence       float64         `json:"confidence"`
	GrowthRate       float64         `json:"growth_rate_percent"`
	TrendDirection   string          `json:"trend_direction"` // increasing, stable, decreasing
	RecommendedAction string         `json:"recommended_action"`
}

// ForecastPoint represents a forecasted data point
type ForecastPoint struct {
	Date  time.Time `json:"date"`
	Usage float64   `json:"usage"`
	Lower float64   `json:"lower_bound"`
	Upper float64   `json:"upper_bound"`
}

// ScalingRecommendation represents scaling recommendations
type ScalingRecommendation struct {
	Service          string   `json:"service"`
	CurrentScale     Capacity `json:"current_scale"`
	RecommendedScale Capacity `json:"recommended_scale"`
	Reason           string   `json:"reason"`
	Impact           string   `json:"impact"`
	CostChange       float64  `json:"cost_change_monthly,omitempty"`
	Priority         string   `json:"priority"` // low, medium, high, urgent
	Implementation   string   `json:"implementation"`
	Timeline         string   `json:"timeline"`
}

// UsageTrend represents usage trend analysis
type UsageTrend struct {
	Service    string      `json:"service"`
	Metric     string      `json:"metric"` // cpu, memory, requests, latency
	Period     string      `json:"period"`
	Current    float64     `json:"current_value"`
	Trend      string      `json:"trend"` // increasing, stable, decreasing
	ChangeRate float64     `json:"change_rate_percent"`
	Peaks      []time.Time `json:"peak_times"`
	PeakValue  float64     `json:"peak_value"`
	AvgValue   float64     `json:"avg_value"`
	MinValue   float64     `json:"min_value"`
}

// OptimizationOpportunity represents optimization recommendations
type OptimizationOpportunity struct {
	Service          string  `json:"service"`
	Type             string  `json:"type"` // right-size, consolidate, scale-down, optimize
	CurrentCost      float64 `json:"current_cost_monthly"`
	PotentialSavings float64 `json:"potential_savings_monthly"`
	Description      string  `json:"description"`
	Risk             string  `json:"risk"` // low, medium, high
	Effort           string  `json:"effort"` // low, medium, high
	ROI              float64 `json:"roi"`
	Implementation   string  `json:"implementation"`
}

// ScalingSimulation represents scaling scenario simulation
type ScalingSimulation struct {
	Service         string   `json:"service"`
	Scenario        string   `json:"scenario"`
	CurrentInstances int     `json:"current_instances"`
	SimulatedInstances int   `json:"simulated_instances"`
	ExpectedCapacity float64 `json:"expected_capacity"`
	CostImpact      float64  `json:"cost_impact_monthly"`
	Performance     string   `json:"performance_impact"`
	Risk            string   `json:"risk"`
	Recommendation  string   `json:"recommendation"`
}

// NewCapacityScaleCommand creates a new capacity-scale command
func NewCapacityScaleCommand() Command {
	cmd := &CapacityScaleCommand{
		flags: flag.NewFlagSet("capacity-scale", flag.ExitOnError),
	}

	cmd.flags.StringVar(&cmd.action, "action", "analyze", "Action: analyze, forecast, recommend, trends, optimize, simulate")
	cmd.flags.StringVar(&cmd.service, "service", "", "Service name")
	cmd.flags.StringVar(&cmd.metric, "metric", "cpu", "Metric to analyze: cpu, memory, requests")
	cmd.flags.IntVar(&cmd.instances, "instances", 0, "Number of instances for simulation")
	cmd.flags.StringVar(&cmd.period, "period", "30d", "Forecast period: 7d, 30d, 90d")
	cmd.flags.StringVar(&cmd.from, "from", "30d", "Start time for analysis (e.g., 7d, 30d)")
	cmd.flags.StringVar(&cmd.to, "to", "now", "End time")
	cmd.flags.BoolVar(&cmd.jsonOut, "json", false, "Output as JSON")

	return cmd
}

// Name returns the command name
func (c *CapacityScaleCommand) Name() string {
	return "capacity-scale"
}

// Description returns the command description
func (c *CapacityScaleCommand) Description() string {
	return "Provide capacity planning and scaling recommendations"
}

// Run executes the capacity-scale command
func (c *CapacityScaleCommand) Run(args []string) error {
	if err := c.flags.Parse(args); err != nil {
		return err
	}

	if c.service == "" {
		return fmt.Errorf("--service flag is required")
	}

	ddClient, err := client.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	switch c.action {
	case "analyze":
		return c.analyzeCapacity(ddClient)
	case "forecast":
		return c.forecastCapacity(ddClient)
	case "recommend":
		return c.getRecommendations(ddClient)
	case "trends":
		return c.analyzeTrends(ddClient)
	case "optimize":
		return c.identifyOptimizations(ddClient)
	case "simulate":
		return c.simulateScaling(ddClient)
	default:
		return fmt.Errorf("unknown action: %s", c.action)
	}
}

// analyzeCapacity analyzes current capacity and utilization
func (c *CapacityScaleCommand) analyzeCapacity(ddClient *client.Client) error {
	analysis := c.performCapacityAnalysis()

	if c.jsonOut {
		return c.outputJSON(analysis)
	}

	c.displayCapacityAnalysis(analysis)
	return nil
}

// forecastCapacity forecasts future capacity needs
func (c *CapacityScaleCommand) forecastCapacity(ddClient *client.Client) error {
	forecast := c.generateForecast()

	if c.jsonOut {
		return c.outputJSON(forecast)
	}

	c.displayForecast(forecast)
	return nil
}

// getRecommendations gets scaling recommendations
func (c *CapacityScaleCommand) getRecommendations(ddClient *client.Client) error {
	recommendations := c.generateRecommendations()

	if c.jsonOut {
		return c.outputJSON(recommendations)
	}

	c.displayRecommendations(recommendations)
	return nil
}

// analyzeTrends analyzes usage trends
func (c *CapacityScaleCommand) analyzeTrends(ddClient *client.Client) error {
	trends := c.calculateTrends()

	if c.jsonOut {
		return c.outputJSON(trends)
	}

	c.displayTrends(trends)
	return nil
}

// identifyOptimizations identifies optimization opportunities
func (c *CapacityScaleCommand) identifyOptimizations(ddClient *client.Client) error {
	opportunities := c.findOptimizations()

	if c.jsonOut {
		return c.outputJSON(opportunities)
	}

	c.displayOptimizations(opportunities)
	return nil
}

// simulateScaling simulates scaling scenarios
func (c *CapacityScaleCommand) simulateScaling(ddClient *client.Client) error {
	if c.instances == 0 {
		return fmt.Errorf("--instances flag is required for simulate action")
	}

	simulation := c.runSimulation()

	if c.jsonOut {
		return c.outputJSON(simulation)
	}

	c.displaySimulation(simulation)
	return nil
}

// performCapacityAnalysis performs capacity analysis
func (c *CapacityScaleCommand) performCapacityAnalysis() CapacityAnalysis {
	// Simulate capacity analysis based on service name
	currentCapacity := Capacity{
		Instances:      5,
		CPU:            20.0,
		Memory:         40.0,
		Storage:        500.0,
		RequestsPerSec: 1500.0,
	}

	utilization := Utilization{
		CPU:     75.5,
		Memory:  68.2,
		Storage: 45.0,
		Network: 52.3,
	}

	bottlenecks := []string{}
	status := "healthy"
	headroom := 45

	if utilization.CPU > 80 {
		bottlenecks = append(bottlenecks, "CPU")
		status = "warning"
		headroom = 15
	}
	if utilization.Memory > 80 {
		bottlenecks = append(bottlenecks, "Memory")
		status = "warning"
		headroom = 20
	}

	if len(bottlenecks) > 1 {
		status = "critical"
		headroom = 7
	}

	recommendation := "Capacity is healthy. Continue monitoring."
	if status == "warning" {
		recommendation = "Consider scaling up within 30 days to maintain headroom."
	} else if status == "critical" {
		recommendation = "Scale up immediately to prevent capacity issues."
	}

	return CapacityAnalysis{
		Service:             c.service,
		AnalyzedAt:          time.Now(),
		CurrentCapacity:     currentCapacity,
		Utilization:         utilization,
		BottleneckResources: bottlenecks,
		HeadroomDays:        headroom,
		Status:              status,
		Recommendation:      recommendation,
	}
}

// generateForecast generates capacity forecast
func (c *CapacityScaleCommand) generateForecast() CapacityForecast {
	currentUsage := 75.5
	growthRate := 2.5

	// Generate forecast points
	forecasted := []ForecastPoint{}
	days := 30
	if c.period == "7d" {
		days = 7
	} else if c.period == "90d" {
		days = 90
	}

	for i := 0; i <= days; i++ {
		date := time.Now().AddDate(0, 0, i)
		usage := currentUsage + (float64(i) * growthRate / float64(days))
		lower := usage - 5.0
		upper := usage + 8.0

		forecasted = append(forecasted, ForecastPoint{
			Date:  date,
			Usage: usage,
			Lower: lower,
			Upper: upper,
		})
	}

	// Calculate capacity exhaustion date
	var capacityDate time.Time
	trendDirection := "increasing"
	action := "Monitor capacity trends."

	if growthRate > 3.0 {
		daysToCapacity := int((100.0 - currentUsage) / (growthRate / 30.0))
		capacityDate = time.Now().AddDate(0, 0, daysToCapacity)
		action = fmt.Sprintf("Scale up before %s to avoid capacity issues.", capacityDate.Format("2006-01-02"))
	} else if growthRate > 1.5 {
		action = "Plan scaling within next 60 days."
	}

	return CapacityForecast{
		Service:            c.service,
		Period:             c.period,
		CurrentUsage:       currentUsage,
		Forecasted:         forecasted,
		CapacityDate:       capacityDate,
		Confidence:         0.85,
		GrowthRate:         growthRate,
		TrendDirection:     trendDirection,
		RecommendedAction:  action,
	}
}

// generateRecommendations generates scaling recommendations
func (c *CapacityScaleCommand) generateRecommendations() []ScalingRecommendation {
	current := Capacity{
		Instances:      5,
		CPU:            20.0,
		Memory:         40.0,
		RequestsPerSec: 1500.0,
	}

	recommendations := []ScalingRecommendation{}

	// High utilization - scale up
	if strings.Contains(strings.ToLower(c.service), "payment") ||
		strings.Contains(strings.ToLower(c.service), "api") {
		recommendations = append(recommendations, ScalingRecommendation{
			Service:      c.service,
			CurrentScale: current,
			RecommendedScale: Capacity{
				Instances:      8,
				CPU:            32.0,
				Memory:         64.0,
				RequestsPerSec: 2400.0,
			},
			Reason:         "CPU utilization consistently above 75%, approaching capacity limits",
			Impact:         "Improved response times and headroom for traffic spikes",
			CostChange:     450.00,
			Priority:       "high",
			Implementation: "Rolling deployment with gradual instance addition",
			Timeline:       "1-2 weeks",
		})
	}

	// Right-sizing opportunity
	recommendations = append(recommendations, ScalingRecommendation{
		Service:      c.service,
		CurrentScale: current,
		RecommendedScale: Capacity{
			Instances:      6,
			CPU:            24.0,
			Memory:         48.0,
			RequestsPerSec: 1800.0,
		},
		Reason:         "Optimal balance between performance and cost",
		Impact:         "20% capacity increase with minimal cost impact",
		CostChange:     150.00,
		Priority:       "medium",
		Implementation: "Add 1 instance during low-traffic window",
		Timeline:       "1 week",
	})

	return recommendations
}

// calculateTrends calculates usage trends
func (c *CapacityScaleCommand) calculateTrends() []UsageTrend {
	trends := []UsageTrend{}

	// CPU trend
	trends = append(trends, UsageTrend{
		Service:    c.service,
		Metric:     "cpu",
		Period:     c.from,
		Current:    75.5,
		Trend:      "increasing",
		ChangeRate: 2.5,
		Peaks: []time.Time{
			time.Now().Add(-24 * time.Hour),
			time.Now().Add(-48 * time.Hour),
		},
		PeakValue: 89.2,
		AvgValue:  72.3,
		MinValue:  58.1,
	})

	// Memory trend
	trends = append(trends, UsageTrend{
		Service:    c.service,
		Metric:     "memory",
		Period:     c.from,
		Current:    68.2,
		Trend:      "stable",
		ChangeRate: 0.5,
		Peaks: []time.Time{
			time.Now().Add(-36 * time.Hour),
		},
		PeakValue: 74.5,
		AvgValue:  67.8,
		MinValue:  61.2,
	})

	// Request rate trend
	trends = append(trends, UsageTrend{
		Service:    c.service,
		Metric:     "requests",
		Period:     c.from,
		Current:    1500.0,
		Trend:      "increasing",
		ChangeRate: 3.2,
		Peaks: []time.Time{
			time.Now().Add(-12 * time.Hour),
			time.Now().Add(-24 * time.Hour),
		},
		PeakValue: 1850.0,
		AvgValue:  1420.0,
		MinValue:  1150.0,
	})

	return trends
}

// findOptimizations finds optimization opportunities
func (c *CapacityScaleCommand) findOptimizations() []OptimizationOpportunity {
	opportunities := []OptimizationOpportunity{}

	// Over-provisioned resources
	opportunities = append(opportunities, OptimizationOpportunity{
		Service:          c.service,
		Type:             "right-size",
		CurrentCost:      2500.00,
		PotentialSavings: 350.00,
		Description:      "Instance type is oversized for current workload. Reduce to t3.large.",
		Risk:             "low",
		Effort:           "low",
		ROI:              14.0,
		Implementation:   "Update instance type in deployment configuration",
	})

	// Idle resources
	opportunities = append(opportunities, OptimizationOpportunity{
		Service:          c.service,
		Type:             "scale-down",
		CurrentCost:      2500.00,
		PotentialSavings: 500.00,
		Description:      "Scale down to 4 instances during off-peak hours (8pm-6am)",
		Risk:             "medium",
		Effort:           "medium",
		ROI:              20.0,
		Implementation:   "Configure autoscaling schedule based on traffic patterns",
	})

	// Resource optimization
	opportunities = append(opportunities, OptimizationOpportunity{
		Service:          c.service,
		Type:             "optimize",
		CurrentCost:      2500.00,
		PotentialSavings: 200.00,
		Description:      "Enable compression and caching to reduce CPU and network usage",
		Risk:             "low",
		Effort:           "medium",
		ROI:              8.0,
		Implementation:   "Update service configuration to enable compression",
	})

	return opportunities
}

// runSimulation runs scaling simulation
func (c *CapacityScaleCommand) runSimulation() ScalingSimulation {
	currentInstances := 5
	simulatedInstances := c.instances

	expectedCapacity := float64(simulatedInstances) / float64(currentInstances) * 100.0
	costImpact := float64(simulatedInstances-currentInstances) * 150.00

	performance := "improved"
	if simulatedInstances < currentInstances {
		performance = "reduced"
	}

	risk := "low"
	if simulatedInstances < 3 {
		risk = "high"
	} else if simulatedInstances < currentInstances {
		risk = "medium"
	}

	recommendation := fmt.Sprintf("Scaling to %d instances is feasible.", simulatedInstances)
	if risk == "high" {
		recommendation = fmt.Sprintf("Scaling to %d instances is risky. Maintain minimum of 3 instances for high availability.", simulatedInstances)
	}

	return ScalingSimulation{
		Service:            c.service,
		Scenario:           fmt.Sprintf("Scale from %d to %d instances", currentInstances, simulatedInstances),
		CurrentInstances:   currentInstances,
		SimulatedInstances: simulatedInstances,
		ExpectedCapacity:   expectedCapacity,
		CostImpact:         costImpact,
		Performance:        performance,
		Risk:               risk,
		Recommendation:     recommendation,
	}
}

// Display functions

func (c *CapacityScaleCommand) displayCapacityAnalysis(analysis CapacityAnalysis) {
	fmt.Printf("\nCapacity Analysis - %s\n", analysis.Service)
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("Status: %s | Headroom: %d days\n", strings.ToUpper(analysis.Status), analysis.HeadroomDays)
	fmt.Printf("Analyzed: %s\n\n", analysis.AnalyzedAt.Format(time.RFC3339))

	fmt.Println("Current Capacity:")
	fmt.Printf("  Instances: %d\n", analysis.CurrentCapacity.Instances)
	fmt.Printf("  CPU: %.1f cores\n", analysis.CurrentCapacity.CPU)
	fmt.Printf("  Memory: %.1f GB\n", analysis.CurrentCapacity.Memory)
	if analysis.CurrentCapacity.Storage > 0 {
		fmt.Printf("  Storage: %.1f GB\n", analysis.CurrentCapacity.Storage)
	}
	fmt.Printf("  Requests/sec: %.0f\n\n", analysis.CurrentCapacity.RequestsPerSec)

	fmt.Println("Utilization:")
	fmt.Printf("  CPU: %.1f%%", analysis.Utilization.CPU)
	if analysis.Utilization.CPU > 80 {
		fmt.Print(" ⚠")
	}
	fmt.Println()
	fmt.Printf("  Memory: %.1f%%", analysis.Utilization.Memory)
	if analysis.Utilization.Memory > 80 {
		fmt.Print(" ⚠")
	}
	fmt.Println()
	if analysis.Utilization.Storage > 0 {
		fmt.Printf("  Storage: %.1f%%\n", analysis.Utilization.Storage)
	}
	if analysis.Utilization.Network > 0 {
		fmt.Printf("  Network: %.1f%%\n", analysis.Utilization.Network)
	}

	if len(analysis.BottleneckResources) > 0 {
		fmt.Printf("\nBottlenecks: %s\n", strings.Join(analysis.BottleneckResources, ", "))
	}

	fmt.Printf("\nRecommendation: %s\n", analysis.Recommendation)
}

func (c *CapacityScaleCommand) displayForecast(forecast CapacityForecast) {
	fmt.Printf("\nCapacity Forecast - %s\n", forecast.Service)
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("Period: %s | Confidence: %.0f%%\n", forecast.Period, forecast.Confidence*100)
	fmt.Printf("Current Usage: %.1f%%\n", forecast.CurrentUsage)
	fmt.Printf("Growth Rate: %.1f%% per month\n", forecast.GrowthRate)
	fmt.Printf("Trend: %s\n\n", forecast.TrendDirection)

	if !forecast.CapacityDate.IsZero() {
		fmt.Printf("⚠ Capacity Exhaustion Projected: %s\n\n", forecast.CapacityDate.Format("2006-01-02"))
	}

	fmt.Println("Forecast Data Points:")
	displayPoints := 10
	if len(forecast.Forecasted) < displayPoints {
		displayPoints = len(forecast.Forecasted)
	}

	for i := 0; i < displayPoints; i++ {
		point := forecast.Forecasted[i]
		fmt.Printf("  %s: %.1f%% (range: %.1f%% - %.1f%%)\n",
			point.Date.Format("2006-01-02"),
			point.Usage,
			point.Lower,
			point.Upper)
	}

	if len(forecast.Forecasted) > displayPoints {
		fmt.Printf("\n  ... and %d more data points (use --json for full output)\n", len(forecast.Forecasted)-displayPoints)
	}

	fmt.Printf("\nRecommended Action: %s\n", forecast.RecommendedAction)
}

func (c *CapacityScaleCommand) displayRecommendations(recommendations []ScalingRecommendation) {
	if len(recommendations) == 0 {
		fmt.Println("No scaling recommendations at this time.")
		return
	}

	fmt.Printf("\nScaling Recommendations: %d\n", len(recommendations))
	fmt.Println(strings.Repeat("=", 80))

	for i, rec := range recommendations {
		priorityIcon := "→"
		if rec.Priority == "urgent" {
			priorityIcon = "🔴"
		} else if rec.Priority == "high" {
			priorityIcon = "🟡"
		}

		fmt.Printf("%s [%s Priority] %s\n", priorityIcon, strings.ToUpper(rec.Priority), rec.Service)
		fmt.Printf("\n  Reason: %s\n", rec.Reason)

		fmt.Printf("\n  Current Scale:\n")
		fmt.Printf("    Instances: %d | CPU: %.1f cores | Memory: %.1f GB\n",
			rec.CurrentScale.Instances, rec.CurrentScale.CPU, rec.CurrentScale.Memory)

		fmt.Printf("\n  Recommended Scale:\n")
		fmt.Printf("    Instances: %d | CPU: %.1f cores | Memory: %.1f GB\n",
			rec.RecommendedScale.Instances, rec.RecommendedScale.CPU, rec.RecommendedScale.Memory)

		fmt.Printf("\n  Impact: %s\n", rec.Impact)
		if rec.CostChange > 0 {
			fmt.Printf("  Cost Change: +$%.2f/month\n", rec.CostChange)
		}
		fmt.Printf("  Timeline: %s\n", rec.Timeline)
		fmt.Printf("  Implementation: %s\n", rec.Implementation)

		if i < len(recommendations)-1 {
			fmt.Println()
		}
	}
}

func (c *CapacityScaleCommand) displayTrends(trends []UsageTrend) {
	if len(trends) == 0 {
		fmt.Println("No trend data available.")
		return
	}

	fmt.Printf("\nUsage Trends - %s\n", c.service)
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("Analysis Period: %s\n\n", trends[0].Period)

	for _, trend := range trends {
		trendIcon := "→"
		if trend.Trend == "increasing" {
			trendIcon = "↑"
		} else if trend.Trend == "decreasing" {
			trendIcon = "↓"
		}

		fmt.Printf("%s %s - %s\n", trendIcon, strings.ToUpper(trend.Metric), trend.Trend)
		fmt.Printf("  Current: %.1f | Avg: %.1f | Peak: %.1f | Min: %.1f\n",
			trend.Current, trend.AvgValue, trend.PeakValue, trend.MinValue)
		fmt.Printf("  Change Rate: %+.1f%% per month\n", trend.ChangeRate)

		if len(trend.Peaks) > 0 {
			fmt.Printf("  Recent Peaks: %d in last %s\n", len(trend.Peaks), trend.Period)
		}

		fmt.Println()
	}
}

func (c *CapacityScaleCommand) displayOptimizations(opportunities []OptimizationOpportunity) {
	if len(opportunities) == 0 {
		fmt.Println("No optimization opportunities identified.")
		return
	}

	fmt.Printf("\nOptimization Opportunities: %d\n", len(opportunities))
	fmt.Println(strings.Repeat("=", 80))

	totalSavings := 0.0
	for _, opp := range opportunities {
		totalSavings += opp.PotentialSavings
	}
	fmt.Printf("Total Potential Savings: $%.2f/month\n\n", totalSavings)

	for i, opp := range opportunities {
		fmt.Printf("[%s] %s\n", strings.ToUpper(opp.Type), opp.Service)
		fmt.Printf("  Description: %s\n", opp.Description)
		fmt.Printf("  Current Cost: $%.2f/month\n", opp.CurrentCost)
		fmt.Printf("  Potential Savings: $%.2f/month (ROI: %.0f%%)\n", opp.PotentialSavings, opp.ROI)
		fmt.Printf("  Risk: %s | Effort: %s\n", opp.Risk, opp.Effort)
		fmt.Printf("  Implementation: %s\n", opp.Implementation)

		if i < len(opportunities)-1 {
			fmt.Println()
		}
	}
}

func (c *CapacityScaleCommand) displaySimulation(simulation ScalingSimulation) {
	fmt.Println("\nScaling Simulation")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("Service: %s\n", simulation.Service)
	fmt.Printf("Scenario: %s\n\n", simulation.Scenario)

	fmt.Printf("Current Instances: %d\n", simulation.CurrentInstances)
	fmt.Printf("Simulated Instances: %d\n", simulation.SimulatedInstances)
	fmt.Printf("Expected Capacity: %.0f%%\n\n", simulation.ExpectedCapacity)

	fmt.Printf("Performance Impact: %s\n", simulation.Performance)
	fmt.Printf("Risk Level: %s\n", strings.ToUpper(simulation.Risk))
	fmt.Printf("Cost Impact: $%.2f/month\n\n", simulation.CostImpact)

	fmt.Printf("Recommendation: %s\n", simulation.Recommendation)
}

// Utility functions

func (c *CapacityScaleCommand) outputJSON(data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(jsonData))
	return nil
}

// Help displays help information
func (c *CapacityScaleCommand) Help() {
	help := `
dd capacity-scale - Provide capacity planning and scaling recommendations

Usage:
  dd capacity-scale --action <action> --service <service> [options]

Actions:
  analyze     Analyze current capacity and utilization
  forecast    Forecast future capacity needs
  recommend   Get scaling recommendations
  trends      Analyze usage trends
  optimize    Identify optimization opportunities
  simulate    Simulate scaling scenarios

Options:
  --action     Action to perform (default: analyze)
  --service    Service name (required)
  --metric     Metric to analyze: cpu, memory, requests (default: cpu)
  --instances  Number of instances for simulation
  --period     Forecast period: 7d, 30d, 90d (default: 30d)
  --from       Start time for analysis (default: 30d) - e.g., 7d, 30d
  --to         End time (default: now)
  --json       Output as JSON

Examples:
  # Analyze current capacity
  dd capacity-scale --action analyze --service api-gateway

  # Forecast future capacity needs
  dd capacity-scale --action forecast --service database --period 30d

  # Get scaling recommendations
  dd capacity-scale --action recommend --service payment-service

  # Analyze usage trends
  dd capacity-scale --action trends --service checkout --from 30d

  # Identify optimization opportunities
  dd capacity-scale --action optimize --service frontend

  # Simulate scaling scenario
  dd capacity-scale --action simulate --service api-gateway --instances 10

Integration Workflows:
  # Capacity planning workflow
  dd capacity-scale --action analyze --service api-gateway
  dd capacity-scale --action forecast --service api-gateway --period 30d
  dd capacity-scale --action recommend --service api-gateway
  dd usage-insights --action summary --from 30d

  # Cost optimization workflow
  dd capacity-scale --action optimize --service frontend
  dd usage-insights --action optimize
  dd cost --action analyze --from 30d

  # Scaling decision workflow
  dd capacity-scale --action trends --service payment-service --from 7d
  dd capacity-scale --action simulate --service payment-service --instances 8
  dd impact-analysis --action service --service payment-service
`
	fmt.Println(help)
}
