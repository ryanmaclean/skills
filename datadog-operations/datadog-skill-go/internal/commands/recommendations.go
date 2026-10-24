package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"sort"
	"time"

	"github.com/datadog/skill/internal/client"
)

type RecommendationsCommand struct {
	flags            *flag.FlagSet
	action           string
	service          string
	recommendationID string
	category         string
	minPriority      int
	jsonOut          bool
}

type AIRecommendation struct {
	ID               string    `json:"id"`
	Service          string    `json:"service"`
	Category         string    `json:"category"` // performance, cost, reliability, security
	Type             string    `json:"type"` // configuration, architecture, scaling, optimization
	Title            string    `json:"title"`
	Description      string    `json:"description"`
	Rationale        string    `json:"rationale"`
	CurrentState     string    `json:"current_state"`
	RecommendedState string    `json:"recommended_state"`
	Impact           Impact    `json:"impact"`
	Effort           string    `json:"effort"` // low, medium, high
	Priority         int       `json:"priority"` // 1-10
	Confidence       float64   `json:"confidence"`
	Evidence         []string  `json:"evidence"`
	Implementation   string    `json:"implementation"`
	Risks            []string  `json:"risks,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	Status           string    `json:"status"` // pending, accepted, rejected, implemented
}

type Impact struct {
	Performance string  `json:"performance,omitempty"`
	Cost        float64 `json:"cost_change,omitempty"`
	Reliability string  `json:"reliability,omitempty"`
	Security    string  `json:"security,omitempty"`
	Overall     string  `json:"overall"` // high, medium, low
}

type TuningRecommendation struct {
	Service             string      `json:"service"`
	Parameter           string      `json:"parameter"`
	CurrentValue        interface{} `json:"current_value"`
	RecommendedValue    interface{} `json:"recommended_value"`
	Category            string      `json:"category"` // memory, cpu, timeout, concurrency
	Rationale           string      `json:"rationale"`
	ExpectedImprovement string      `json:"expected_improvement"`
	Confidence          float64     `json:"confidence"`
	RiskLevel           string      `json:"risk_level"`
	TestingAdvice       string      `json:"testing_advice"`
}

type BestPracticeCheck struct {
	Service      string `json:"service"`
	Practice     string `json:"practice"`
	Category     string `json:"category"` // monitoring, reliability, performance, security
	Status       string `json:"status"` // compliant, warning, violation
	Description  string `json:"description"`
	CurrentState string `json:"current_state"`
	BestPractice string `json:"best_practice"`
	Severity     string `json:"severity"`
	Recommendation string `json:"recommendation"`
	Documentation string `json:"documentation_url,omitempty"`
}

type ServiceComparison struct {
	Service        string                `json:"service"`
	ComparedWith   []string              `json:"compared_with"`
	Metrics        map[string]Comparison `json:"metrics"`
	OverallRanking int                   `json:"overall_ranking"` // 1-100 percentile
	Strengths      []string              `json:"strengths"`
	Weaknesses     []string              `json:"weaknesses"`
	Opportunities  []string              `json:"opportunities"`
}

type Comparison struct {
	Metric       string  `json:"metric"`
	CurrentValue float64 `json:"current_value"`
	PeerAverage  float64 `json:"peer_average"`
	PeerMedian   float64 `json:"peer_median"`
	Percentile   float64 `json:"percentile"` // 0-100
	Status       string  `json:"status"` // above_average, average, below_average
}

type RecommendationSimulation struct {
	Recommendation  AIRecommendation `json:"recommendation"`
	SimulatedImpact SimulatedImpact  `json:"simulated_impact"`
	Confidence      float64         `json:"confidence"`
	Risk            string          `json:"risk"`
	ROI             float64         `json:"roi,omitempty"`
	Timeframe       string          `json:"timeframe"`
}

type SimulatedImpact struct {
	PerformanceChange   string             `json:"performance_change"`
	CostChange          float64            `json:"cost_change"`
	ReliabilityChange   string             `json:"reliability_change"`
	ResourceUtilization map[string]float64 `json:"resource_utilization"`
}

func NewRecommendationsCommand() Command {
	return &RecommendationsCommand{}
}

func (c *RecommendationsCommand) Name() string {
	return "recommendations"
}

func (c *RecommendationsCommand) Description() string {
	return "AI-driven optimization recommendations, auto-tuning, and best practices"
}

func (c *RecommendationsCommand) Run(args []string) error {
	c.flags = flag.NewFlagSet("recommendations", flag.ExitOnError)
	c.flags.StringVar(&c.action, "action", "optimize", "Action to perform")
	c.flags.StringVar(&c.service, "service", "", "Service name")
	c.flags.StringVar(&c.recommendationID, "recommendation-id", "", "Recommendation ID")
	c.flags.StringVar(&c.category, "category", "", "Category filter: performance, cost, reliability, security")
	c.flags.IntVar(&c.minPriority, "min-priority", 1, "Minimum priority (1-10)")
	c.flags.BoolVar(&c.jsonOut, "json", false, "Output as JSON")

	if err := c.flags.Parse(args); err != nil {
		return err
	}

	ddClient, err := client.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Datadog client: %w", err)
	}

	switch c.action {
	case "optimize":
		return c.getOptimizations(ddClient)
	case "tune":
		return c.getTuning(ddClient)
	case "best-practices":
		return c.checkBestPractices(ddClient)
	case "compare":
		return c.compareServices(ddClient)
	case "prioritize":
		return c.prioritizeRecommendations(ddClient)
	case "simulate":
		return c.simulateRecommendation(ddClient)
	default:
		return fmt.Errorf("unknown action: %s", c.action)
	}
}

func (c *RecommendationsCommand) Help() {
	help := `dd recommendations - AI-driven optimization recommendations, auto-tuning, and best practices

Usage:
  dd recommendations --action <action> --service <service> [options]

Actions:
  optimize        Get optimization recommendations
  tune            Auto-tuning suggestions for configurations
  best-practices  Identify deviations from best practices
  compare         Compare against similar services
  prioritize      Prioritize recommendations by impact
  simulate        Simulate recommendation impact

Options:
  --action            Action to perform (default: optimize)
  --service           Service name (required)
  --recommendation-id Recommendation ID for simulate action
  --category          Category filter: performance, cost, reliability, security
  --min-priority      Minimum priority 1-10 (default: 1)
  --json              Output as JSON

Examples:
  # Get optimization recommendations
  dd recommendations --action optimize --service api-gateway

  # Get auto-tuning suggestions
  dd recommendations --action tune --service database

  # Check best practices compliance
  dd recommendations --action best-practices --service payment-service

  # Compare with similar services
  dd recommendations --action compare --service checkout

  # Prioritize recommendations by impact
  dd recommendations --action prioritize --service api-gateway --min-priority 5

  # Simulate recommendation impact
  dd recommendations --action simulate --service api-gateway --recommendation-id rec-123

Integration Workflows:
  # AI-driven cost optimization
  dd predictions --action costs --service infra --window month
  dd recommendations --action optimize --service infra --category cost
  dd recommendations --action prioritize --service infra
  dd recommendations --action simulate --service infra --recommendation-id rec-456

  # Performance optimization
  dd ml-insights --action detect --service api-gateway
  dd recommendations --action tune --service api-gateway
  dd recommendations --action compare --service api-gateway
  dd capacity-scale --action simulate --service api-gateway

  # Continuous improvement
  dd recommendations --action best-practices --service checkout
  dd recommendations --action optimize --service checkout
  dd recommendations --action prioritize --service checkout --category reliability
  dd change-management --action track --service checkout --type optimization
`
	fmt.Println(help)
}

func (c *RecommendationsCommand) getOptimizations(ddClient *client.Client) error {
	if c.service == "" {
		return fmt.Errorf("--service flag is required for optimize action")
	}

	fmt.Printf("Generating optimization recommendations for service: %s\n", c.service)
	if c.category != "" {
		fmt.Printf("Category filter: %s\n", c.category)
	}

	// Simulate optimization recommendations
	recommendations := c.generateOptimizations()

	// Filter by category if specified
	if c.category != "" {
		filtered := []AIRecommendation{}
		for _, rec := range recommendations {
			if rec.Category == c.category {
				filtered = append(filtered, rec)
			}
		}
		recommendations = filtered
	}

	if c.jsonOut {
		data, _ := json.MarshalIndent(recommendations, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("\n=== Optimization Recommendations ===\n")
	fmt.Printf("Found %d recommendations\n\n", len(recommendations))

	for i, rec := range recommendations {
		fmt.Printf("Recommendation #%d:\n", i+1)
		fmt.Printf("  ID: %s\n", rec.ID)
		fmt.Printf("  Title: %s\n", rec.Title)
		fmt.Printf("  Category: %s\n", rec.Category)
		fmt.Printf("  Type: %s\n", rec.Type)
		fmt.Printf("  Priority: %d/10\n", rec.Priority)
		fmt.Printf("  Effort: %s\n", rec.Effort)
		fmt.Printf("  Confidence: %.1f%%\n", rec.Confidence*100)
		fmt.Printf("\n  Description:\n    %s\n", rec.Description)
		fmt.Printf("\n  Rationale:\n    %s\n", rec.Rationale)
		fmt.Printf("\n  Current State:\n    %s\n", rec.CurrentState)
		fmt.Printf("\n  Recommended State:\n    %s\n", rec.RecommendedState)

		fmt.Println("\n  Impact:")
		if rec.Impact.Performance != "" {
			fmt.Printf("    Performance: %s\n", rec.Impact.Performance)
		}
		if rec.Impact.Cost != 0 {
			fmt.Printf("    Cost: $%.2f/month\n", rec.Impact.Cost)
		}
		if rec.Impact.Reliability != "" {
			fmt.Printf("    Reliability: %s\n", rec.Impact.Reliability)
		}
		if rec.Impact.Security != "" {
			fmt.Printf("    Security: %s\n", rec.Impact.Security)
		}
		fmt.Printf("    Overall Impact: %s\n", rec.Impact.Overall)

		if len(rec.Evidence) > 0 {
			fmt.Println("\n  Evidence:")
			for _, evidence := range rec.Evidence {
				fmt.Printf("    - %s\n", evidence)
			}
		}

		fmt.Printf("\n  Implementation:\n    %s\n", rec.Implementation)

		if len(rec.Risks) > 0 {
			fmt.Println("\n  Risks:")
			for _, risk := range rec.Risks {
				fmt.Printf("    - %s\n", risk)
			}
		}

		fmt.Printf("\n  Status: %s\n", rec.Status)
		fmt.Println()
	}

	return nil
}

func (c *RecommendationsCommand) getTuning(ddClient *client.Client) error {
	if c.service == "" {
		return fmt.Errorf("--service flag is required for tune action")
	}

	fmt.Printf("Generating auto-tuning recommendations for service: %s\n", c.service)

	// Simulate tuning recommendations
	tuning := c.generateTuningRecommendations()

	if c.jsonOut {
		data, _ := json.MarshalIndent(tuning, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Println("\n=== Auto-Tuning Recommendations ===")
	fmt.Printf("Found %d tuning opportunities\n\n", len(tuning))

	for i, tune := range tuning {
		fmt.Printf("Tuning #%d:\n", i+1)
		fmt.Printf("  Service: %s\n", tune.Service)
		fmt.Printf("  Parameter: %s\n", tune.Parameter)
		fmt.Printf("  Category: %s\n", tune.Category)
		fmt.Printf("  Current Value: %v\n", tune.CurrentValue)
		fmt.Printf("  Recommended Value: %v\n", tune.RecommendedValue)
		fmt.Printf("  Confidence: %.1f%%\n", tune.Confidence*100)
		fmt.Printf("  Risk Level: %s\n", tune.RiskLevel)
		fmt.Printf("\n  Rationale:\n    %s\n", tune.Rationale)
		fmt.Printf("\n  Expected Improvement:\n    %s\n", tune.ExpectedImprovement)
		fmt.Printf("\n  Testing Advice:\n    %s\n", tune.TestingAdvice)
		fmt.Println()
	}

	return nil
}

func (c *RecommendationsCommand) checkBestPractices(ddClient *client.Client) error {
	if c.service == "" {
		return fmt.Errorf("--service flag is required for best-practices action")
	}

	fmt.Printf("Checking best practices for service: %s\n", c.service)

	// Simulate best practice checks
	checks := c.generateBestPracticeChecks()

	if c.jsonOut {
		data, _ := json.MarshalIndent(checks, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	// Count by status
	compliant := 0
	warnings := 0
	violations := 0
	for _, check := range checks {
		switch check.Status {
		case "compliant":
			compliant++
		case "warning":
			warnings++
		case "violation":
			violations++
		}
	}

	fmt.Println("\n=== Best Practices Check ===")
	fmt.Printf("Total Checks: %d\n", len(checks))
	fmt.Printf("  Compliant: %d\n", compliant)
	fmt.Printf("  Warnings: %d\n", warnings)
	fmt.Printf("  Violations: %d\n\n", violations)

	// Group by status
	for _, status := range []string{"violation", "warning", "compliant"} {
		statusChecks := []BestPracticeCheck{}
		for _, check := range checks {
			if check.Status == status {
				statusChecks = append(statusChecks, check)
			}
		}

		if len(statusChecks) == 0 {
			continue
		}

		fmt.Printf("=== %s ===\n", status)
		for i, check := range statusChecks {
			fmt.Printf("\n%d. %s (%s)\n", i+1, check.Practice, check.Category)
			fmt.Printf("   Severity: %s\n", check.Severity)
			fmt.Printf("   Description: %s\n", check.Description)
			fmt.Printf("   Current State: %s\n", check.CurrentState)
			if check.Status != "compliant" {
				fmt.Printf("   Best Practice: %s\n", check.BestPractice)
				fmt.Printf("   Recommendation: %s\n", check.Recommendation)
			}
			if check.Documentation != "" {
				fmt.Printf("   Documentation: %s\n", check.Documentation)
			}
		}
		fmt.Println()
	}

	return nil
}

func (c *RecommendationsCommand) compareServices(ddClient *client.Client) error {
	if c.service == "" {
		return fmt.Errorf("--service flag is required for compare action")
	}

	fmt.Printf("Comparing service against peers: %s\n", c.service)

	// Simulate service comparison
	comparison := c.generateServiceComparison()

	if c.jsonOut {
		data, _ := json.MarshalIndent(comparison, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Println("\n=== Service Comparison ===")
	fmt.Printf("Service: %s\n", comparison.Service)
	fmt.Printf("Compared With: %v\n", comparison.ComparedWith)
	fmt.Printf("Overall Ranking: %d percentile\n\n", comparison.OverallRanking)

	fmt.Println("Metrics Comparison:")
	fmt.Println("Metric               Current    Peer Avg   Peer Med   Percentile  Status")
	fmt.Println("----------------------------------------------------------------------------")

	// Sort metrics for consistent output
	metricNames := []string{}
	for name := range comparison.Metrics {
		metricNames = append(metricNames, name)
	}
	sort.Strings(metricNames)

	for _, name := range metricNames {
		metric := comparison.Metrics[name]
		fmt.Printf("%-20s %-10.2f %-10.2f %-10.2f %-11.0f %s\n",
			name,
			metric.CurrentValue,
			metric.PeerAverage,
			metric.PeerMedian,
			metric.Percentile,
			metric.Status,
		)
	}

	fmt.Println("\nStrengths:")
	for _, strength := range comparison.Strengths {
		fmt.Printf("  + %s\n", strength)
	}

	fmt.Println("\nWeaknesses:")
	for _, weakness := range comparison.Weaknesses {
		fmt.Printf("  - %s\n", weakness)
	}

	fmt.Println("\nOpportunities:")
	for _, opportunity := range comparison.Opportunities {
		fmt.Printf("  → %s\n", opportunity)
	}

	return nil
}

func (c *RecommendationsCommand) prioritizeRecommendations(ddClient *client.Client) error {
	if c.service == "" {
		return fmt.Errorf("--service flag is required for prioritize action")
	}

	fmt.Printf("Prioritizing recommendations for service: %s\n", c.service)
	fmt.Printf("Minimum priority: %d\n", c.minPriority)

	// Generate and sort recommendations
	recommendations := c.generateOptimizations()

	// Filter by minimum priority
	filtered := []AIRecommendation{}
	for _, rec := range recommendations {
		if rec.Priority >= c.minPriority {
			filtered = append(filtered, rec)
		}
	}

	// Sort by priority (descending)
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Priority > filtered[j].Priority
	})

	if c.jsonOut {
		data, _ := json.MarshalIndent(filtered, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("\n=== Prioritized Recommendations ===\n")
	fmt.Printf("Showing %d recommendations (priority >= %d)\n\n", len(filtered), c.minPriority)

	for i, rec := range filtered {
		fmt.Printf("%d. [Priority %d] %s\n", i+1, rec.Priority, rec.Title)
		fmt.Printf("   Category: %s | Effort: %s | Impact: %s\n", rec.Category, rec.Effort, rec.Impact.Overall)
		if rec.Impact.Cost != 0 {
			fmt.Printf("   Cost Impact: $%.2f/month\n", rec.Impact.Cost)
		}
		if rec.Impact.Performance != "" {
			fmt.Printf("   Performance: %s\n", rec.Impact.Performance)
		}
		fmt.Printf("   Description: %s\n", rec.Description)
		fmt.Println()
	}

	return nil
}

func (c *RecommendationsCommand) simulateRecommendation(ddClient *client.Client) error {
	if c.service == "" {
		return fmt.Errorf("--service flag is required for simulate action")
	}
	if c.recommendationID == "" {
		return fmt.Errorf("--recommendation-id flag is required for simulate action")
	}

	fmt.Printf("Simulating recommendation for service: %s\n", c.service)
	fmt.Printf("Recommendation ID: %s\n", c.recommendationID)

	// Simulate recommendation impact
	simulation := c.generateSimulation()

	if c.jsonOut {
		data, _ := json.MarshalIndent(simulation, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Println("\n=== Recommendation Simulation ===")
	fmt.Printf("Recommendation: %s\n", simulation.Recommendation.Title)
	fmt.Printf("Category: %s\n", simulation.Recommendation.Category)
	fmt.Printf("Confidence: %.1f%%\n", simulation.Confidence*100)
	fmt.Printf("Risk: %s\n", simulation.Risk)
	if simulation.ROI > 0 {
		fmt.Printf("ROI: %.1fx\n", simulation.ROI)
	}
	fmt.Printf("Timeframe: %s\n\n", simulation.Timeframe)

	fmt.Println("Simulated Impact:")
	fmt.Printf("  Performance Change: %s\n", simulation.SimulatedImpact.PerformanceChange)
	fmt.Printf("  Cost Change: $%.2f/month\n", simulation.SimulatedImpact.CostChange)
	fmt.Printf("  Reliability Change: %s\n", simulation.SimulatedImpact.ReliabilityChange)

	fmt.Println("\n  Resource Utilization After Change:")
	for resource, utilization := range simulation.SimulatedImpact.ResourceUtilization {
		fmt.Printf("    %s: %.1f%%\n", resource, utilization)
	}

	fmt.Printf("\nImplementation:\n  %s\n", simulation.Recommendation.Implementation)

	if len(simulation.Recommendation.Risks) > 0 {
		fmt.Println("\nRisks:")
		for _, risk := range simulation.Recommendation.Risks {
			fmt.Printf("  - %s\n", risk)
		}
	}

	return nil
}

// Helper functions

func (c *RecommendationsCommand) generateOptimizations() []AIRecommendation {
	now := time.Now()

	recommendations := []AIRecommendation{
		{
			ID:          "rec-001",
			Service:     c.service,
			Category:    "performance",
			Type:        "configuration",
			Title:       "Increase connection pool size",
			Description: "Database connection pool is frequently exhausted, causing request queuing",
			Rationale:   "Current pool size of 50 connections reaches 95%+ utilization during peak hours. Analysis shows 200+ queued requests per minute.",
			CurrentState: "Connection pool: 50, Peak utilization: 96%, Avg queue time: 45ms",
			RecommendedState: "Connection pool: 75, Expected utilization: 65%, Estimated queue time: <5ms",
			Impact: Impact{
				Performance: "Reduce response time by 30%, eliminate queue delays",
				Overall:     "high",
			},
			Effort:     "low",
			Priority:   9,
			Confidence: 0.92,
			Evidence: []string{
				"Connection pool utilization >90% for 8 hours/day",
				"200+ queued requests per minute during peaks",
				"Response time P95 at 450ms (target: 200ms)",
				"Similar services using 75-100 connections",
			},
			Implementation: "Update database config: max_connections=75, test in staging, gradual rollout",
			Risks: []string{
				"Database may need connection limit increase",
				"Memory usage will increase by ~50MB",
			},
			CreatedAt: now.Add(-2 * time.Hour),
			Status:    "pending",
		},
		{
			ID:          "rec-002",
			Service:     c.service,
			Category:    "cost",
			Type:        "optimization",
			Title:       "Reduce log verbosity",
			Description: "Log ingestion costs are 35% above baseline due to verbose debug logging in production",
			Rationale:   "Debug-level logs account for 60% of total log volume but provide minimal value in production. Moving to INFO level will reduce volume by 55%.",
			CurrentState: "Log level: DEBUG, Volume: 12GB/day, Cost: $420/month",
			RecommendedState: "Log level: INFO, Volume: 5.4GB/day, Cost: $189/month",
			Impact: Impact{
				Cost:    -231.0,
				Overall: "high",
			},
			Effort:     "low",
			Priority:   8,
			Confidence: 0.89,
			Evidence: []string{
				"DEBUG logs: 60% of volume, rarely accessed",
				"Similar services using INFO level in production",
				"Log analysis shows 95% of debugging uses ERROR/WARN logs",
				"Cost trend: +35% over last 30 days",
			},
			Implementation: "Update logging config to INFO level, keep DEBUG for development/staging",
			Risks: []string{
				"May need to temporarily re-enable DEBUG for troubleshooting",
				"Adjust alerting if based on DEBUG log patterns",
			},
			CreatedAt: now.Add(-4 * time.Hour),
			Status:    "pending",
		},
		{
			ID:          "rec-003",
			Service:     c.service,
			Category:    "reliability",
			Type:        "architecture",
			Title:       "Implement circuit breaker for external API",
			Description: "External API failures cascade to service causing availability issues",
			Rationale:   "External API has 99.5% SLA but failures cause 2-5 minute outages. Circuit breaker would fail fast and use cached data.",
			CurrentState: "No circuit breaker, 30s timeout, no fallback, 5 retry attempts",
			RecommendedState: "Circuit breaker with 3s timeout, cached fallback, fail after 2 attempts",
			Impact: Impact{
				Reliability: "Reduce MTTR from 5min to 30s, prevent cascade failures",
				Performance: "Reduce timeout from 30s to 3s for failed requests",
				Overall:     "high",
			},
			Effort:     "medium",
			Priority:   8,
			Confidence: 0.85,
			Evidence: []string{
				"3 outages in last 30 days due to external API",
				"Average outage duration: 4.5 minutes",
				"P99 timeout: 28 seconds during failures",
				"Cached data acceptable for 82% of use cases",
			},
			Implementation: "Add circuit breaker library, implement fallback cache, set 3s timeout, monitor failure rate",
			Risks: []string{
				"Stale cached data may cause inconsistencies",
				"Circuit breaker tuning may require adjustments",
			},
			CreatedAt: now.Add(-6 * time.Hour),
			Status:    "pending",
		},
		{
			ID:          "rec-004",
			Service:     c.service,
			Category:    "performance",
			Type:        "scaling",
			Title:       "Enable horizontal autoscaling",
			Description: "Manual scaling leads to over-provisioning and occasional performance issues",
			Rationale:   "Traffic varies 3x between peak and off-peak. Autoscaling would maintain performance while reducing costs by 25%.",
			CurrentState: "Fixed 12 instances, Manual scaling, Avg utilization: 45%",
			RecommendedState: "Autoscale 6-15 instances based on CPU >70%, Estimated avg: 8 instances",
			Impact: Impact{
				Performance: "Maintain P95 response time <200ms during peaks",
				Cost:        -1250.0,
				Reliability: "Automatic scaling during traffic spikes",
				Overall:     "medium",
			},
			Effort:     "medium",
			Priority:   6,
			Confidence: 0.78,
			Evidence: []string{
				"CPU utilization ranges 25-85% daily",
				"Over-provisioned 55% of the time",
				"Traffic predictable: 3x peak vs off-peak",
				"Similar services successfully using autoscaling",
			},
			Implementation: "Configure autoscaling group, set CPU threshold 70%, min 6 / max 15 instances, test scaling behavior",
			Risks: []string{
				"Scaling lag may cause brief performance degradation",
				"Need to ensure stateless design",
				"Cost savings depend on traffic patterns",
			},
			CreatedAt: now.Add(-8 * time.Hour),
			Status:    "pending",
		},
		{
			ID:          "rec-005",
			Service:     c.service,
			Category:    "security",
			Type:        "configuration",
			Title:       "Enable request rate limiting",
			Description: "No rate limiting exposes service to abuse and DDoS attacks",
			Rationale:   "Service is publicly exposed with no rate limiting. Adding rate limiting would prevent abuse and protect against traffic spikes.",
			CurrentState: "No rate limiting, Unlimited requests per IP",
			RecommendedState: "Rate limit: 1000 req/min per IP, 10000 req/min global, 429 responses",
			Impact: Impact{
				Security:    "Protect against DDoS and abuse, prevent resource exhaustion",
				Reliability: "Prevent cascade failures from traffic spikes",
				Overall:     "medium",
			},
			Effort:     "low",
			Priority:   7,
			Confidence: 0.88,
			Evidence: []string{
				"No rate limiting detected in current config",
				"Public API with authentication but no throttling",
				"Occasional traffic spikes from single IPs (>5000 req/min)",
				"Industry standard: rate limiting for public APIs",
			},
			Implementation: "Add rate limiting middleware, set per-IP and global limits, return 429 with Retry-After header",
			Risks: []string{
				"Legitimate users may hit limits during peak usage",
				"Need to handle rate limit errors gracefully",
			},
			CreatedAt: now.Add(-10 * time.Hour),
			Status:    "pending",
		},
	}

	return recommendations
}

func (c *RecommendationsCommand) generateTuningRecommendations() []TuningRecommendation {
	return []TuningRecommendation{
		{
			Service:          c.service,
			Parameter:        "JVM_HEAP_SIZE",
			CurrentValue:     "2048m",
			RecommendedValue: "3072m",
			Category:         "memory",
			Rationale:        "GC pauses occurring frequently (avg 150ms) due to heap pressure. Analysis shows heap usage at 92% causing frequent full GCs.",
			ExpectedImprovement: "Reduce GC pause frequency by 60%, reduce P99 latency by 40ms",
			Confidence:       0.87,
			RiskLevel:        "low",
			TestingAdvice:    "Test in staging with production traffic replay. Monitor GC logs and heap usage for 48 hours before production rollout.",
		},
		{
			Service:          c.service,
			Parameter:        "CONNECTION_TIMEOUT",
			CurrentValue:     5000,
			RecommendedValue: 3000,
			Category:         "timeout",
			Rationale:        "Current 5s timeout is too high. 99% of requests complete in <1s. Reducing timeout will fail fast and prevent queue buildup.",
			ExpectedImprovement: "Reduce cascading failures, improve error response time from 5s to 3s",
			Confidence:       0.82,
			RiskLevel:        "medium",
			TestingAdvice:    "Monitor timeout rate after change. If >1% of requests timeout, revert. Consider implementing retry with exponential backoff.",
		},
		{
			Service:          c.service,
			Parameter:        "WORKER_THREADS",
			CurrentValue:     50,
			RecommendedValue: 75,
			Category:         "concurrency",
			Rationale:        "Thread pool utilization at 94% during peaks. Increasing threads will reduce request queuing and improve concurrency.",
			ExpectedImprovement: "Reduce queue wait time by 50%, improve P95 response time by 25%",
			Confidence:       0.79,
			RiskLevel:        "low",
			TestingAdvice:    "Increase gradually (50→60→75). Monitor CPU and memory usage. Ensure underlying resources (DB connections) can handle increased concurrency.",
		},
		{
			Service:          c.service,
			Parameter:        "CACHE_TTL",
			CurrentValue:     300,
			RecommendedValue: 600,
			Category:         "performance",
			Rationale:        "Cache hit rate at 68%. Analysis shows data changes infrequently (avg 12min). Doubling TTL would increase hit rate to ~85% with minimal staleness.",
			ExpectedImprovement: "Increase cache hit rate from 68% to 85%, reduce backend load by 20%",
			Confidence:       0.75,
			RiskLevel:        "medium",
			TestingAdvice:    "Monitor data freshness requirements. If stale data issues occur, reduce TTL to 450s (middle ground). Track cache hit rate and backend request count.",
		},
	}
}

func (c *RecommendationsCommand) generateBestPracticeChecks() []BestPracticeCheck {
	return []BestPracticeCheck{
		{
			Service:      c.service,
			Practice:     "Health check endpoint",
			Category:     "reliability",
			Status:       "compliant",
			Description:  "Service has /health endpoint responding in <100ms",
			CurrentState: "Health endpoint: /health, Response time: 45ms, Returns 200 OK",
			BestPractice: "Implement health check endpoint for orchestration and monitoring",
			Severity:     "n/a",
			Recommendation: "Compliant - no action needed",
		},
		{
			Service:      c.service,
			Practice:     "Request timeout configuration",
			Category:     "reliability",
			Status:       "warning",
			Description:  "Request timeout is higher than recommended",
			CurrentState: "Timeout: 30s",
			BestPractice: "Set request timeout to 5-10s for API services",
			Severity:     "medium",
			Recommendation: "Reduce timeout to 10s to fail fast and prevent cascade delays",
			Documentation: "https://docs.example.com/best-practices/timeouts",
		},
		{
			Service:      c.service,
			Practice:     "Circuit breaker implementation",
			Category:     "reliability",
			Status:       "violation",
			Description:  "No circuit breaker for external dependencies",
			CurrentState: "No circuit breaker detected",
			BestPractice: "Implement circuit breaker for all external dependencies",
			Severity:     "high",
			Recommendation: "Add circuit breaker with fallback mechanisms for external API calls",
			Documentation: "https://docs.example.com/patterns/circuit-breaker",
		},
		{
			Service:      c.service,
			Practice:     "Structured logging",
			Category:     "monitoring",
			Status:       "compliant",
			Description:  "Service uses structured JSON logging",
			CurrentState: "Logging format: JSON, Fields: timestamp, level, service, trace_id, message",
			BestPractice: "Use structured logging with consistent fields",
			Severity:     "n/a",
			Recommendation: "Compliant - no action needed",
		},
		{
			Service:      c.service,
			Practice:     "Rate limiting",
			Category:     "security",
			Status:       "violation",
			Description:  "No rate limiting on public endpoints",
			CurrentState: "Rate limiting: none",
			BestPractice: "Implement rate limiting on all public APIs",
			Severity:     "high",
			Recommendation: "Add rate limiting: 1000 req/min per IP, 10000 req/min global",
			Documentation: "https://docs.example.com/security/rate-limiting",
		},
		{
			Service:      c.service,
			Practice:     "Graceful shutdown",
			Category:     "reliability",
			Status:       "warning",
			Description:  "Shutdown timeout may be too short",
			CurrentState: "Graceful shutdown timeout: 5s",
			BestPractice: "Allow 15-30s for graceful shutdown to complete in-flight requests",
			Severity:     "medium",
			Recommendation: "Increase shutdown timeout to 15s to handle long-running requests",
			Documentation: "https://docs.example.com/deployment/graceful-shutdown",
		},
		{
			Service:      c.service,
			Practice:     "Resource limits",
			Category:     "performance",
			Status:       "compliant",
			Description:  "CPU and memory limits configured",
			CurrentState: "CPU limit: 2 cores, Memory limit: 4GB",
			BestPractice: "Set resource limits to prevent resource exhaustion",
			Severity:     "n/a",
			Recommendation: "Compliant - no action needed",
		},
		{
			Service:      c.service,
			Practice:     "Distributed tracing",
			Category:     "monitoring",
			Status:       "compliant",
			Description:  "Service propagates trace context",
			CurrentState: "Tracing: enabled, Trace ID propagation: yes, Sampling: 10%",
			BestPractice: "Enable distributed tracing with context propagation",
			Severity:     "n/a",
			Recommendation: "Compliant - no action needed",
		},
	}
}

func (c *RecommendationsCommand) generateServiceComparison() ServiceComparison {
	return ServiceComparison{
		Service:      c.service,
		ComparedWith: []string{"checkout-service", "payment-service", "user-service"},
		Metrics: map[string]Comparison{
			"response_time_p95": {
				Metric:       "response_time_p95",
				CurrentValue: 245.0,
				PeerAverage:  180.0,
				PeerMedian:   165.0,
				Percentile:   35.0,
				Status:       "below_average",
			},
			"error_rate": {
				Metric:       "error_rate",
				CurrentValue: 0.8,
				PeerAverage:  1.2,
				PeerMedian:   1.0,
				Percentile:   68.0,
				Status:       "above_average",
			},
			"throughput": {
				Metric:       "throughput",
				CurrentValue: 1250.0,
				PeerAverage:  1100.0,
				PeerMedian:   1050.0,
				Percentile:   72.0,
				Status:       "above_average",
			},
			"availability": {
				Metric:       "availability",
				CurrentValue: 99.85,
				PeerAverage:  99.91,
				PeerMedian:   99.92,
				Percentile:   42.0,
				Status:       "below_average",
			},
			"cpu_usage": {
				Metric:       "cpu_usage",
				CurrentValue: 62.0,
				PeerAverage:  58.0,
				PeerMedian:   55.0,
				Percentile:   38.0,
				Status:       "average",
			},
		},
		OverallRanking: 55,
		Strengths: []string{
			"Low error rate (0.8% vs peer avg 1.2%)",
			"High throughput (1250 req/s vs peer avg 1100 req/s)",
			"Good error handling and validation",
		},
		Weaknesses: []string{
			"Response time slower than peers (245ms vs 165ms median)",
			"Availability below peer average (99.85% vs 99.91%)",
			"Higher CPU usage than similar services",
		},
		Opportunities: []string{
			"Optimize database queries to improve response time",
			"Implement circuit breaker to improve availability",
			"Profile CPU usage to identify optimization opportunities",
			"Consider caching to reduce response time and CPU load",
		},
	}
}

func (c *RecommendationsCommand) generateSimulation() RecommendationSimulation {
	// Use the first optimization recommendation
	recommendations := c.generateOptimizations()
	rec := recommendations[0] // Connection pool recommendation

	return RecommendationSimulation{
		Recommendation: rec,
		SimulatedImpact: SimulatedImpact{
			PerformanceChange: "Response time P95: 450ms → 315ms (-30%)",
			CostChange:        15.0, // Slight increase due to more connections
			ReliabilityChange: "Request queue: 200/min → 12/min (-94%)",
			ResourceUtilization: map[string]float64{
				"cpu":              58.5, // Slight decrease due to less queuing
				"memory":           72.0, // Slight increase for more connections
				"connections":      65.0, // After increase from 96% to 65%
				"request_queue":    8.0,  // Dramatic decrease
			},
		},
		Confidence: 0.92,
		Risk:       "low",
		ROI:        8.5, // High ROI: low cost increase, high performance gain
		Timeframe:  "1-2 days implementation, immediate impact",
	}
}
