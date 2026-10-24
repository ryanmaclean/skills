package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"sort"
	"time"

	"github.com/datadog/skill/internal/client"
	"github.com/datadog/skill/internal/context"
	"github.com/datadog/skill/internal/observability"
)

// DORACommand queries DORA (DevOps Research and Assessment) Metrics
type DORACommand struct {
	flags    *flag.FlagSet
	service  string
	env      string
	duration string
	metric   string
	team     string
	jsonOut  bool
}

// DORADeployment represents a deployment event
type DORADeployment struct {
	ID          string    `json:"id"`
	Service     string    `json:"service"`
	Environment string    `json:"env,omitempty"`
	Team        string    `json:"team,omitempty"`
	Version     string    `json:"version,omitempty"`
	StartedAt   time.Time `json:"started_at"`
	FinishedAt  time.Time `json:"finished_at"`
	Duration    int64     `json:"duration_ms"`
	Repository  string    `json:"repository_url,omitempty"`
	CommitSHA   string    `json:"commit_sha,omitempty"`
}

// DORAFailure represents a failure/incident event
type DORAFailure struct {
	ID          string    `json:"id"`
	Service     string    `json:"service"`
	Environment string    `json:"env,omitempty"`
	Team        string    `json:"team,omitempty"`
	StartedAt   time.Time `json:"started_at"`
	FinishedAt  time.Time `json:"finished_at,omitempty"`
	Duration    int64     `json:"duration_ms"`
	Impact      string    `json:"impact,omitempty"`
}

// Note: DORAMetrics type is defined in cicd.go and reused here
// Using MTTR (Mean Time To Restore) and FailedDeployments from cicd.go struct

// DORAOutput represents the formatted DORA response
type DORAOutput struct {
	Status      string           `json:"status"`
	Service     string           `json:"service"`
	Environment string           `json:"env,omitempty"`
	Duration    string           `json:"duration"`
	Metrics     *DORAMetrics     `json:"metrics,omitempty"`
	Deployments []DORADeployment `json:"recent_deployments,omitempty"`
	Failures    []DORAFailure    `json:"recent_failures,omitempty"`
}

// NewDORACommand creates a new DORA command
func NewDORACommand() *DORACommand {
	cmd := &DORACommand{
		flags: flag.NewFlagSet("dora", flag.ExitOnError),
	}

	cmd.flags.StringVar(&cmd.service, "service", "", "Service name (auto-detect if not provided)")
	cmd.flags.StringVar(&cmd.env, "env", "", "Environment (e.g., production, staging)")
	cmd.flags.StringVar(&cmd.duration, "duration", "7d", "Time range: 1d, 7d, 30d, 90d")
	cmd.flags.StringVar(&cmd.metric, "metric", "all", "Metric type: deployments, failures, metrics, all")
	cmd.flags.StringVar(&cmd.team, "team", "", "Filter by team")
	cmd.flags.BoolVar(&cmd.jsonOut, "json", false, "Output as JSON")

	return cmd
}

func (c *DORACommand) Name() string {
	return "dora"
}

func (c *DORACommand) Description() string {
	return "Query DORA Metrics for DevOps performance measurement"
}

func (c *DORACommand) Run(args []string) error {
	// Initialize observability
	obs, err := observability.Init("query-dora", "production")
	if err != nil {
		return fmt.Errorf("failed to init observability: %w", err)
	}
	defer obs.Shutdown(0)

	// Parse flags
	if err := c.flags.Parse(args); err != nil {
		return err
	}

	// Detect service if not provided
	serviceName := c.service
	if serviceName == "" {
		span := obs.StartSpan("detect_context")
		obs.LogInfo("Auto-detecting service name")

		ctx, err := context.DetectContext(".")
		obs.FinishSpan(span)

		if err != nil || ctx.ServiceName == "" {
			obs.LogError("Could not detect service name")
			return fmt.Errorf("could not detect service name: specify with --service or run in a git repository")
		}

		serviceName = ctx.ServiceName
		obs.LogInfo(fmt.Sprintf("Auto-detected service: %s", serviceName))
	}

	obs.LogInfo(fmt.Sprintf("Querying DORA metrics for service: %s", serviceName))

	// Parse duration
	fromTime, toTime, err := c.parseDuration(c.duration)
	if err != nil {
		obs.LogError(fmt.Sprintf("Invalid duration: %s", err.Error()))
		return fmt.Errorf("invalid duration: %w", err)
	}

	// Create Datadog client
	ddClient, err := client.NewClient()
	if err != nil {
		obs.LogError(fmt.Sprintf("Failed to create Datadog client: %s", err.Error()))
		return fmt.Errorf("failed to create Datadog client: %w", err)
	}

	// Initialize output
	output := &DORAOutput{
		Status:      "ok",
		Service:     serviceName,
		Environment: c.env,
		Duration:    c.duration,
	}

	// Query based on metric type
	switch c.metric {
	case "deployments":
		deployments, err := c.queryDeployments(ddClient, serviceName, fromTime, toTime, obs)
		if err != nil {
			return err
		}
		output.Deployments = deployments
	case "failures":
		failures, err := c.queryFailures(ddClient, serviceName, fromTime, toTime, obs)
		if err != nil {
			return err
		}
		output.Failures = failures
	case "metrics":
		metrics, err := c.calculateMetrics(ddClient, serviceName, fromTime, toTime, obs)
		if err != nil {
			return err
		}
		output.Metrics = metrics
	case "all":
		deployments, err := c.queryDeployments(ddClient, serviceName, fromTime, toTime, obs)
		if err != nil {
			return err
		}
		failures, err := c.queryFailures(ddClient, serviceName, fromTime, toTime, obs)
		if err != nil {
			return err
		}
		metrics := c.computeMetrics(deployments, failures, fromTime, toTime)

		output.Deployments = deployments
		output.Failures = failures
		output.Metrics = metrics
	default:
		return fmt.Errorf("invalid metric type: %s (use: deployments, failures, metrics, all)", c.metric)
	}

	// Output results
	if c.jsonOut {
		jsonData, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	} else {
		c.printFormatted(output)
	}

	obs.LogInfo("Query completed successfully")
	return nil
}

func (c *DORACommand) queryDeployments(ddClient *client.Client, service string, from, to time.Time, obs *observability.Observability) ([]DORADeployment, error) {
	span := obs.StartSpan("query_deployments")
	defer obs.FinishSpan(span)

	// Build query filter
	filter := map[string]interface{}{
		"service": service,
		"from":    from.Format(time.RFC3339),
		"to":      to.Format(time.RFC3339),
	}
	if c.env != "" {
		filter["env"] = c.env
	}
	if c.team != "" {
		filter["team"] = c.team
	}

	payload := map[string]interface{}{
		"filter": filter,
		"page": map[string]interface{}{
			"limit": 100,
		},
	}

	data, err := ddClient.QueryDORADeployments(payload)
	if err != nil {
		obs.LogError(fmt.Sprintf("Failed to query deployments: %s", err.Error()))
		return nil, fmt.Errorf("failed to query deployments: %w", err)
	}

	var response struct {
		Data []struct {
			ID         string `json:"id"`
			Attributes struct {
				Service      string                 `json:"service"`
				Env          string                 `json:"env"`
				Team         string                 `json:"team"`
				Version      string                 `json:"version"`
				StartedAt    int64                  `json:"started_at"`
				FinishedAt   int64                  `json:"finished_at"`
				RepositoryURL string                `json:"repository_url"`
				CommitSHA    string                 `json:"commit_sha"`
				Tags         []string               `json:"tags"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse deployments response: %w", err)
	}

	deployments := make([]DORADeployment, 0, len(response.Data))
	for _, item := range response.Data {
		startedAt := time.Unix(0, item.Attributes.StartedAt)
		finishedAt := time.Unix(0, item.Attributes.FinishedAt)
		duration := finishedAt.Sub(startedAt).Milliseconds()

		deployments = append(deployments, DORADeployment{
			ID:          item.ID,
			Service:     item.Attributes.Service,
			Environment: item.Attributes.Env,
			Team:        item.Attributes.Team,
			Version:     item.Attributes.Version,
			StartedAt:   startedAt,
			FinishedAt:  finishedAt,
			Duration:    duration,
			Repository:  item.Attributes.RepositoryURL,
			CommitSHA:   item.Attributes.CommitSHA,
		})
	}

	// Sort by start time (most recent first)
	sort.Slice(deployments, func(i, j int) bool {
		return deployments[i].StartedAt.After(deployments[j].StartedAt)
	})

	return deployments, nil
}

func (c *DORACommand) queryFailures(ddClient *client.Client, service string, from, to time.Time, obs *observability.Observability) ([]DORAFailure, error) {
	span := obs.StartSpan("query_failures")
	defer obs.FinishSpan(span)

	// Build query filter
	filter := map[string]interface{}{
		"service": service,
		"from":    from.Format(time.RFC3339),
		"to":      to.Format(time.RFC3339),
	}
	if c.env != "" {
		filter["env"] = c.env
	}
	if c.team != "" {
		filter["team"] = c.team
	}

	payload := map[string]interface{}{
		"filter": filter,
		"page": map[string]interface{}{
			"limit": 100,
		},
	}

	data, err := ddClient.QueryDORAFailures(payload)
	if err != nil {
		obs.LogError(fmt.Sprintf("Failed to query failures: %s", err.Error()))
		return nil, fmt.Errorf("failed to query failures: %w", err)
	}

	var response struct {
		Data []struct {
			ID         string `json:"id"`
			Attributes struct {
				Service    string   `json:"service"`
				Env        string   `json:"env"`
				Team       string   `json:"team"`
				StartedAt  int64    `json:"started_at"`
				FinishedAt int64    `json:"finished_at"`
				Impact     string   `json:"impact"`
				Tags       []string `json:"tags"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse failures response: %w", err)
	}

	failures := make([]DORAFailure, 0, len(response.Data))
	for _, item := range response.Data {
		startedAt := time.Unix(0, item.Attributes.StartedAt)
		var finishedAt time.Time
		var duration int64
		if item.Attributes.FinishedAt > 0 {
			finishedAt = time.Unix(0, item.Attributes.FinishedAt)
			duration = finishedAt.Sub(startedAt).Milliseconds()
		}

		failures = append(failures, DORAFailure{
			ID:          item.ID,
			Service:     item.Attributes.Service,
			Environment: item.Attributes.Env,
			Team:        item.Attributes.Team,
			StartedAt:   startedAt,
			FinishedAt:  finishedAt,
			Duration:    duration,
			Impact:      item.Attributes.Impact,
		})
	}

	// Sort by start time (most recent first)
	sort.Slice(failures, func(i, j int) bool {
		return failures[i].StartedAt.After(failures[j].StartedAt)
	})

	return failures, nil
}

func (c *DORACommand) calculateMetrics(ddClient *client.Client, service string, from, to time.Time, obs *observability.Observability) (*DORAMetrics, error) {
	deployments, err := c.queryDeployments(ddClient, service, from, to, obs)
	if err != nil {
		return nil, err
	}

	failures, err := c.queryFailures(ddClient, service, from, to, obs)
	if err != nil {
		return nil, err
	}

	return c.computeMetrics(deployments, failures, from, to), nil
}

func (c *DORACommand) computeMetrics(deployments []DORADeployment, failures []DORAFailure, from, to time.Time) *DORAMetrics {
	metrics := &DORAMetrics{
		DeploymentCount:   len(deployments),
		FailedDeployments: len(failures),
	}

	if len(deployments) == 0 {
		return metrics
	}

	// Calculate deployment frequency (deployments per day)
	totalDays := to.Sub(from).Hours() / 24
	if totalDays > 0 {
		metrics.DeploymentFrequency = float64(len(deployments)) / totalDays
	}

	// Calculate change failure rate
	if len(deployments) > 0 {
		metrics.ChangeFailureRate = (float64(len(failures)) / float64(len(deployments))) * 100
	}

	// Calculate average lead time (for deployments with commit data)
	var totalLeadTime float64
	leadTimeCount := 0
	for _, dep := range deployments {
		if !dep.StartedAt.IsZero() && !dep.FinishedAt.IsZero() {
			leadTime := dep.FinishedAt.Sub(dep.StartedAt).Hours()
			totalLeadTime += leadTime
			leadTimeCount++
		}
	}
	if leadTimeCount > 0 {
		metrics.LeadTime = totalLeadTime / float64(leadTimeCount)
	}

	// Calculate average time to restore (MTTR)
	var totalRestoreTime float64
	restoreCount := 0
	for _, failure := range failures {
		if !failure.StartedAt.IsZero() && !failure.FinishedAt.IsZero() {
			restoreTime := failure.FinishedAt.Sub(failure.StartedAt).Hours()
			totalRestoreTime += restoreTime
			restoreCount++
		}
	}
	if restoreCount > 0 {
		metrics.MTTR = totalRestoreTime / float64(restoreCount)
	}

	return metrics
}

func (c *DORACommand) parseDuration(duration string) (time.Time, time.Time, error) {
	toTime := time.Now()
	var fromTime time.Time

	switch duration {
	case "1d":
		fromTime = toTime.AddDate(0, 0, -1)
	case "7d":
		fromTime = toTime.AddDate(0, 0, -7)
	case "30d":
		fromTime = toTime.AddDate(0, 0, -30)
	case "90d":
		fromTime = toTime.AddDate(0, 0, -90)
	default:
		return time.Time{}, time.Time{}, fmt.Errorf("invalid duration: %s (use: 1d, 7d, 30d, 90d)", duration)
	}

	return fromTime, toTime, nil
}

func (c *DORACommand) printFormatted(output *DORAOutput) {
	fmt.Printf("DORA Metrics: %s\n", output.Service)
	if output.Environment != "" {
		fmt.Printf("Environment: %s\n", output.Environment)
	}
	fmt.Printf("Duration: %s\n", output.Duration)
	fmt.Println()

	if output.Metrics != nil {
		fmt.Println("=== Key Metrics ===")
		fmt.Printf("Deployment Frequency: %.2f per day\n", output.Metrics.DeploymentFrequency)
		fmt.Printf("Lead Time: %.2f hours\n", output.Metrics.LeadTime)
		fmt.Printf("Change Failure Rate: %.2f%%\n", output.Metrics.ChangeFailureRate)
		fmt.Printf("Time to Restore: %.2f hours\n", output.Metrics.MTTR)
		fmt.Println()
		fmt.Printf("Total Deployments: %d\n", output.Metrics.DeploymentCount)
		fmt.Printf("Total Failures: %d\n", output.Metrics.FailedDeployments)
		fmt.Println()
	}

	if len(output.Deployments) > 0 {
		fmt.Println("=== Recent Deployments ===")
		for i, dep := range output.Deployments {
			if i >= 5 {
				break
			}
			fmt.Printf("  [%s] %s\n", dep.StartedAt.Format("2006-01-02 15:04"), dep.ID)
			if dep.Version != "" {
				fmt.Printf("    Version: %s\n", dep.Version)
			}
			if dep.CommitSHA != "" {
				fmt.Printf("    Commit: %s\n", dep.CommitSHA[:min(8, len(dep.CommitSHA))])
			}
			fmt.Printf("    Duration: %dms\n", dep.Duration)
		}
		fmt.Println()
	}

	if len(output.Failures) > 0 {
		fmt.Println("=== Recent Failures ===")
		for i, fail := range output.Failures {
			if i >= 5 {
				break
			}
			fmt.Printf("  [%s] %s\n", fail.StartedAt.Format("2006-01-02 15:04"), fail.ID)
			if fail.Impact != "" {
				fmt.Printf("    Impact: %s\n", fail.Impact)
			}
			if fail.Duration > 0 {
				fmt.Printf("    Duration: %dms\n", fail.Duration)
			}
		}
		fmt.Println()
	}

	// DORA metrics rating
	if output.Metrics != nil {
		fmt.Println("=== Performance Rating ===")
		c.printPerformanceRating(output.Metrics)
	}
}

func (c *DORACommand) printPerformanceRating(metrics *DORAMetrics) {
	// Based on DORA research categories: Elite, High, Medium, Low

	// Deployment Frequency rating
	fmt.Print("Deployment Frequency: ")
	if metrics.DeploymentFrequency >= 1.0 {
		fmt.Println("🏆 Elite (Multiple times per day)")
	} else if metrics.DeploymentFrequency >= 0.14 { // ~1 per week
		fmt.Println("✅ High (Weekly)")
	} else if metrics.DeploymentFrequency >= 0.03 { // ~1 per month
		fmt.Println("⚠️  Medium (Monthly)")
	} else {
		fmt.Println("❌ Low (Less than monthly)")
	}

	// Lead Time rating
	fmt.Print("Lead Time: ")
	if metrics.LeadTime < 24 {
		fmt.Println("🏆 Elite (Less than 1 day)")
	} else if metrics.LeadTime < 168 { // 1 week
		fmt.Println("✅ High (Less than 1 week)")
	} else if metrics.LeadTime < 720 { // 1 month
		fmt.Println("⚠️  Medium (Less than 1 month)")
	} else {
		fmt.Println("❌ Low (More than 1 month)")
	}

	// Change Failure Rate rating
	fmt.Print("Change Failure Rate: ")
	if metrics.ChangeFailureRate < 15 {
		fmt.Println("🏆 Elite (0-15%)")
	} else if metrics.ChangeFailureRate < 30 {
		fmt.Println("✅ High (15-30%)")
	} else if metrics.ChangeFailureRate < 45 {
		fmt.Println("⚠️  Medium (30-45%)")
	} else {
		fmt.Println("❌ Low (>45%)")
	}

	// Time to Restore rating
	fmt.Print("Time to Restore: ")
	if metrics.MTTR < 1 {
		fmt.Println("🏆 Elite (Less than 1 hour)")
	} else if metrics.MTTR < 24 {
		fmt.Println("✅ High (Less than 1 day)")
	} else if metrics.MTTR < 168 { // 1 week
		fmt.Println("⚠️  Medium (Less than 1 week)")
	} else {
		fmt.Println("❌ Low (More than 1 week)")
	}
}

func (c *DORACommand) Help() {
	fmt.Println("Usage: dd dora [options]")
	fmt.Println()
	fmt.Println("Query DORA (DevOps Research and Assessment) Metrics.")
	fmt.Println("Measures deployment frequency, lead time, change failure rate, and time to restore.")
	fmt.Println()
	fmt.Println("Options:")
	c.flags.PrintDefaults()
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  dd dora")
	fmt.Println("  dd dora --service my-service")
	fmt.Println("  dd dora --env production --duration 30d")
	fmt.Println("  dd dora --metric deployments --team platform")
	fmt.Println("  dd dora --metric all --json")
	fmt.Println()
	fmt.Println("DORA Metrics:")
	fmt.Println("  Deployment Frequency - How often you deploy to production")
	fmt.Println("  Lead Time - Time from commit to production deployment")
	fmt.Println("  Change Failure Rate - Percentage of deployments causing failures")
	fmt.Println("  Time to Restore - Time to recover from production incidents")
	fmt.Println()
	fmt.Println("Performance Ratings (based on DORA research):")
	fmt.Println("  Elite:  Multiple deploys/day, <1hr lead time, <15% failure rate, <1hr restore")
	fmt.Println("  High:   Weekly deploys, <1d lead time, <30% failure rate, <1d restore")
	fmt.Println("  Medium: Monthly deploys, <1mo lead time, <45% failure rate, <1wk restore")
	fmt.Println("  Low:    Below medium thresholds")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
