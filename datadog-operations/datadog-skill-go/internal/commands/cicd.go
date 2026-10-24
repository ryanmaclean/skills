package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/datadog/skill/internal/client"
	"github.com/datadog/skill/internal/context"
	"github.com/datadog/skill/internal/observability"
)

// CICDCommand queries CI/CD (CI Visibility) data
type CICDCommand struct {
	flags    *flag.FlagSet
	service  string
	duration string
	metric   string
	branch   string
	status   string
	json     bool
}

// CIPipeline represents a CI pipeline execution
type CIPipeline struct {
	PipelineID   string  `json:"pipeline_id"`
	Branch       string  `json:"branch"`
	Status       string  `json:"status"`
	Duration     float64 `json:"duration_ms"`
	Timestamp    string  `json:"timestamp"`
	Repository   string  `json:"repository"`
	CommitSHA    string  `json:"commit_sha"`
	TriggeredBy  string  `json:"triggered_by"`
}

// CITest represents a CI test execution
type CITest struct {
	TestName   string  `json:"test_name"`
	Suite      string  `json:"suite"`
	Status     string  `json:"status"`
	Duration   float64 `json:"duration_ms"`
	Flaky      bool    `json:"flaky"`
	Repository string  `json:"repository"`
	Branch     string  `json:"branch"`
}

// DORAMetrics represents DORA (DevOps Research and Assessment) metrics
type DORAMetrics struct {
	DeploymentFrequency float64 `json:"deployment_frequency_per_day"`
	LeadTime            float64 `json:"lead_time_hours"`
	ChangeFailureRate   float64 `json:"change_failure_rate_percent"`
	MTTR                float64 `json:"mttr_hours"`
	DeploymentCount     int     `json:"deployment_count"`
	FailedDeployments   int     `json:"failed_deployments"`
}

// PipelineStats represents aggregated pipeline statistics
type PipelineStats struct {
	TotalRuns       int     `json:"total_runs"`
	SuccessRate     float64 `json:"success_rate_percent"`
	FailureRate     float64 `json:"failure_rate_percent"`
	AvgDuration     float64 `json:"avg_duration_ms"`
	P95Duration     float64 `json:"p95_duration_ms"`
	TrendDirection  string  `json:"trend_direction"`
	TrendPercentage float64 `json:"trend_percentage"`
}

// TestStats represents aggregated test statistics
type TestStats struct {
	TotalTests      int      `json:"total_tests"`
	PassRate        float64  `json:"pass_rate_percent"`
	FailRate        float64  `json:"fail_rate_percent"`
	FlakyTests      int      `json:"flaky_tests"`
	AvgDuration     float64  `json:"avg_duration_ms"`
	FailedTestNames []string `json:"failed_test_names"`
}

// CICDResponse represents the formatted CI/CD response
type CICDResponse struct {
	Status          string            `json:"status"`
	Service         string            `json:"service"`
	Duration        string            `json:"duration"`
	Metric          string            `json:"metric"`
	Branch          string            `json:"branch,omitempty"`
	PipelineStats   *PipelineStats    `json:"pipeline_stats,omitempty"`
	TestStats       *TestStats        `json:"test_stats,omitempty"`
	DORAMetrics     *DORAMetrics      `json:"dora_metrics,omitempty"`
	RecentPipelines []CIPipeline      `json:"recent_pipelines,omitempty"`
	RecentTests     []CITest          `json:"recent_tests,omitempty"`
	RawData         map[string]interface{} `json:"raw_data,omitempty"`
}

// NewCICDCommand creates a new CI/CD command
func NewCICDCommand() *CICDCommand {
	cmd := &CICDCommand{
		flags: flag.NewFlagSet("cicd", flag.ExitOnError),
	}

	cmd.flags.StringVar(&cmd.service, "service", "", "Filter by service (auto-detect if not provided)")
	cmd.flags.StringVar(&cmd.duration, "duration", "7d", "Time range: 1h, 24h, 7d, 30d (default: 7d)")
	cmd.flags.StringVar(&cmd.metric, "metric", "all", "Metric type: pipelines, tests, deployments, all (default: all)")
	cmd.flags.StringVar(&cmd.branch, "branch", "", "Filter by branch (default: all branches)")
	cmd.flags.StringVar(&cmd.status, "status", "all", "Filter by status: passed, failed, all (default: all)")
	cmd.flags.BoolVar(&cmd.json, "json", false, "Output as JSON")

	return cmd
}

// Name returns the command name
func (c *CICDCommand) Name() string {
	return "cicd"
}

// Description returns the command description
func (c *CICDCommand) Description() string {
	return "Query CI/CD (CI Visibility) data for pipeline and test analysis"
}

// Run executes the CI/CD command
func (c *CICDCommand) Run(args []string) error {
	// Initialize observability
	obs, err := observability.Init("dd-cli-cicd", "production")
	if err != nil {
		return fmt.Errorf("failed to init observability: %w", err)
	}
	defer obs.Shutdown(0)

	// Parse flags
	if err := c.flags.Parse(args); err != nil {
		return err
	}

	// Start tracing
	span := obs.StartSpan("cicd.query")
	defer obs.FinishSpan(span)

	obs.LogInfo("Starting CI/CD query")

	// Auto-detect service if needed
	service := c.service
	if service == "" {
		detectSpan := obs.StartSpan("cicd.detect_context")
		obs.LogInfo("Auto-detecting service context")

		ctx, err := context.DetectContext(".")
		if err != nil {
			obs.LogWarning("Failed to detect context, querying all services")
		} else {
			service = ctx.ServiceName
			obs.LogInfo(fmt.Sprintf("Auto-detected service: %s", service))
			obs.GetTracer().SetTag(detectSpan, "service.name", service)
			obs.GetTracer().SetTag(detectSpan, "detection.method", ctx.DetectionMethod)
		}
		obs.FinishSpan(detectSpan)
	}

	// Parse duration
	fromTime, toTime, err := c.parseDuration()
	if err != nil {
		obs.LogError("Failed to parse duration: " + err.Error())
		return fmt.Errorf("failed to parse duration: %w", err)
	}

	// Create Datadog client
	clientSpan := obs.StartSpan("cicd.create_client")
	ddClient, err := client.NewClient()
	if err != nil {
		obs.LogError("Failed to create Datadog client: " + err.Error())
		return fmt.Errorf("failed to create Datadog client: %w", err)
	}
	obs.FinishSpan(clientSpan)

	// Build response
	response := &CICDResponse{
		Status:   "ok",
		Service:  service,
		Duration: c.duration,
		Metric:   c.metric,
		Branch:   c.branch,
	}

	// Query based on metric type
	obs.GetTracer().SetTag(span, "metric", c.metric)
	obs.GetTracer().SetTag(span, "service", service)
	obs.GetTracer().SetTag(span, "duration", c.duration)

	switch c.metric {
	case "pipelines":
		if err := c.queryPipelines(obs, ddClient, service, fromTime, toTime, response); err != nil {
			return err
		}
	case "tests":
		if err := c.queryTests(obs, ddClient, service, fromTime, toTime, response); err != nil {
			return err
		}
	case "deployments":
		if err := c.queryDeployments(obs, ddClient, service, fromTime, toTime, response); err != nil {
			return err
		}
	case "all":
		if err := c.queryPipelines(obs, ddClient, service, fromTime, toTime, response); err != nil {
			obs.LogWarning("Failed to query pipelines: " + err.Error())
		}
		if err := c.queryTests(obs, ddClient, service, fromTime, toTime, response); err != nil {
			obs.LogWarning("Failed to query tests: " + err.Error())
		}
		if err := c.queryDeployments(obs, ddClient, service, fromTime, toTime, response); err != nil {
			obs.LogWarning("Failed to query deployments: " + err.Error())
		}
	default:
		return fmt.Errorf("invalid metric type: %s (use pipelines, tests, deployments, or all)", c.metric)
	}

	// Record metrics
	if response.PipelineStats != nil {
		obs.GetMetrics().Gauge("cicd.pipeline.total_runs", float64(response.PipelineStats.TotalRuns), "service:"+service)
		obs.GetMetrics().Gauge("cicd.pipeline.success_rate", response.PipelineStats.SuccessRate, "service:"+service)
		obs.GetMetrics().Gauge("cicd.pipeline.avg_duration", response.PipelineStats.AvgDuration, "service:"+service)
	}
	if response.TestStats != nil {
		obs.GetMetrics().Gauge("cicd.test.total", float64(response.TestStats.TotalTests), "service:"+service)
		obs.GetMetrics().Gauge("cicd.test.pass_rate", response.TestStats.PassRate, "service:"+service)
		obs.GetMetrics().Gauge("cicd.test.flaky_count", float64(response.TestStats.FlakyTests), "service:"+service)
	}

	// Output
	if c.json {
		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	} else {
		c.printFormatted(response)
	}

	obs.LogInfo("CI/CD query complete")
	return nil
}

// queryPipelines queries CI pipeline data
func (c *CICDCommand) queryPipelines(obs *observability.Observability, ddClient *client.Client, service string, from, to time.Time, response *CICDResponse) error {
	span := obs.StartSpan("cicd.query_pipelines")
	defer obs.FinishSpan(span)

	obs.LogInfo("Querying CI pipelines")

	// Build filter
	filter := c.buildPipelineFilter(service)

	start := time.Now()
	rawData, err := ddClient.QueryCIPipelines(service, from, to, filter)
	apiDuration := time.Since(start).Milliseconds()

	if err != nil {
		obs.LogError("Pipeline query failed: " + err.Error())
		obs.RecordAPICall("/api/v2/ci/pipelines/analytics/aggregate", "POST", 500, float64(apiDuration), err)
		return fmt.Errorf("failed to query pipelines: %w", err)
	}

	obs.RecordAPICall("/api/v2/ci/pipelines/analytics/aggregate", "POST", 200, float64(apiDuration), nil)

	// Parse pipeline data
	stats, pipelines, err := c.parsePipelineData(rawData)
	if err != nil {
		obs.LogError("Failed to parse pipeline data: " + err.Error())
		return fmt.Errorf("failed to parse pipeline data: %w", err)
	}

	response.PipelineStats = stats
	response.RecentPipelines = pipelines

	obs.LogInfo(fmt.Sprintf("Found %d pipeline executions", stats.TotalRuns))
	return nil
}

// queryTests queries CI test data
func (c *CICDCommand) queryTests(obs *observability.Observability, ddClient *client.Client, service string, from, to time.Time, response *CICDResponse) error {
	span := obs.StartSpan("cicd.query_tests")
	defer obs.FinishSpan(span)

	obs.LogInfo("Querying CI tests")

	// Build filter
	filter := c.buildTestFilter(service)

	start := time.Now()
	rawData, err := ddClient.QueryCITests(service, from, to, filter)
	apiDuration := time.Since(start).Milliseconds()

	if err != nil {
		obs.LogError("Test query failed: " + err.Error())
		obs.RecordAPICall("/api/v2/ci/tests/analytics/aggregate", "POST", 500, float64(apiDuration), err)
		return fmt.Errorf("failed to query tests: %w", err)
	}

	obs.RecordAPICall("/api/v2/ci/tests/analytics/aggregate", "POST", 200, float64(apiDuration), nil)

	// Parse test data
	stats, tests, err := c.parseTestData(rawData)
	if err != nil {
		obs.LogError("Failed to parse test data: " + err.Error())
		return fmt.Errorf("failed to parse test data: %w", err)
	}

	response.TestStats = stats
	response.RecentTests = tests

	obs.LogInfo(fmt.Sprintf("Found %d test executions", stats.TotalTests))
	return nil
}

// queryDeployments queries deployment data and calculates DORA metrics
func (c *CICDCommand) queryDeployments(obs *observability.Observability, ddClient *client.Client, service string, from, to time.Time, response *CICDResponse) error {
	span := obs.StartSpan("cicd.query_deployments")
	defer obs.FinishSpan(span)

	obs.LogInfo("Querying deployments and calculating DORA metrics")

	// Query deployment events (tagged as ci.pipeline with deployment stage)
	filter := c.buildDeploymentFilter(service)

	start := time.Now()
	rawData, err := ddClient.GetCIPipelineExecutions(service, from, to, filter)
	apiDuration := time.Since(start).Milliseconds()

	if err != nil {
		obs.LogError("Deployment query failed: " + err.Error())
		obs.RecordAPICall("/api/v2/ci/pipelines/events/search", "POST", 500, float64(apiDuration), err)
		return fmt.Errorf("failed to query deployments: %w", err)
	}

	obs.RecordAPICall("/api/v2/ci/pipelines/events/search", "POST", 200, float64(apiDuration), nil)

	// Calculate DORA metrics
	dora, err := c.calculateDORAMetrics(rawData, from, to)
	if err != nil {
		obs.LogError("Failed to calculate DORA metrics: " + err.Error())
		return fmt.Errorf("failed to calculate DORA metrics: %w", err)
	}

	response.DORAMetrics = dora

	obs.LogInfo(fmt.Sprintf("Calculated DORA metrics: %d deployments", dora.DeploymentCount))
	return nil
}

// buildPipelineFilter constructs the pipeline query filter
func (c *CICDCommand) buildPipelineFilter(service string) string {
	var filters []string

	if service != "" {
		filters = append(filters, fmt.Sprintf("service:%s", service))
	}

	if c.branch != "" {
		filters = append(filters, fmt.Sprintf("git.branch:%s", c.branch))
	}

	if c.status != "all" {
		filters = append(filters, fmt.Sprintf("status:%s", c.status))
	}

	return strings.Join(filters, " AND ")
}

// buildTestFilter constructs the test query filter
func (c *CICDCommand) buildTestFilter(service string) string {
	var filters []string

	if service != "" {
		filters = append(filters, fmt.Sprintf("service:%s", service))
	}

	if c.branch != "" {
		filters = append(filters, fmt.Sprintf("git.branch:%s", c.branch))
	}

	if c.status != "all" {
		filters = append(filters, fmt.Sprintf("test.status:%s", c.status))
	}

	return strings.Join(filters, " AND ")
}

// buildDeploymentFilter constructs the deployment query filter
func (c *CICDCommand) buildDeploymentFilter(service string) string {
	var filters []string

	filters = append(filters, "ci.pipeline.stage:deploy")

	if service != "" {
		filters = append(filters, fmt.Sprintf("service:%s", service))
	}

	if c.branch != "" {
		filters = append(filters, fmt.Sprintf("git.branch:%s", c.branch))
	}

	return strings.Join(filters, " AND ")
}

// parseDuration parses the duration string into time range
func (c *CICDCommand) parseDuration() (time.Time, time.Time, error) {
	toTime := time.Now()
	var duration time.Duration

	if strings.HasSuffix(c.duration, "h") {
		hours := strings.TrimSuffix(c.duration, "h")
		var h int
		if _, err := fmt.Sscanf(hours, "%d", &h); err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid duration: %s", c.duration)
		}
		duration = time.Duration(h) * time.Hour
	} else if strings.HasSuffix(c.duration, "d") {
		days := strings.TrimSuffix(c.duration, "d")
		var d int
		if _, err := fmt.Sscanf(days, "%d", &d); err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid duration: %s", c.duration)
		}
		duration = time.Duration(d) * 24 * time.Hour
	} else {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid duration format: %s. Use format like '1h', '24h', '7d', '30d'", c.duration)
	}

	fromTime := toTime.Add(-duration)
	return fromTime, toTime, nil
}

// parsePipelineData parses pipeline data from API response
func (c *CICDCommand) parsePipelineData(rawData []byte) (*PipelineStats, []CIPipeline, error) {
	var apiResponse map[string]interface{}
	if err := json.Unmarshal(rawData, &apiResponse); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	stats := &PipelineStats{}
	var pipelines []CIPipeline

	// Extract pipeline data from API response
	data, ok := apiResponse["data"].(map[string]interface{})
	if !ok {
		return stats, pipelines, nil
	}

	buckets, ok := data["buckets"].([]interface{})
	if !ok {
		return stats, pipelines, nil
	}

	var durations []float64
	successCount := 0
	failureCount := 0

	for _, bucket := range buckets {
		bucketMap, ok := bucket.(map[string]interface{})
		if !ok {
			continue
		}

		by, ok := bucketMap["by"].(map[string]interface{})
		if !ok {
			continue
		}

		computes, ok := bucketMap["computes"].(map[string]interface{})
		if !ok {
			continue
		}

		// Extract pipeline information
		pipeline := CIPipeline{
			PipelineID: getStringValue(by, "ci.pipeline.id"),
			Branch:     getStringValue(by, "git.branch"),
			Status:     getStringValue(by, "status"),
			Repository: getStringValue(by, "git.repository"),
			CommitSHA:  getStringValue(by, "git.commit.sha"),
		}

		// Extract metrics
		if duration, ok := computes["c0"].(float64); ok {
			pipeline.Duration = duration / 1_000_000 // Convert ns to ms
			durations = append(durations, pipeline.Duration)
		}

		if count, ok := computes["c1"].(float64); ok {
			stats.TotalRuns += int(count)

			if pipeline.Status == "passed" || pipeline.Status == "success" {
				successCount += int(count)
			} else if pipeline.Status == "failed" || pipeline.Status == "error" {
				failureCount += int(count)
			}
		}

		pipelines = append(pipelines, pipeline)
	}

	// Calculate statistics
	if stats.TotalRuns > 0 {
		stats.SuccessRate = float64(successCount) / float64(stats.TotalRuns) * 100
		stats.FailureRate = float64(failureCount) / float64(stats.TotalRuns) * 100
	}

	if len(durations) > 0 {
		// Calculate average
		var totalDuration float64
		for _, d := range durations {
			totalDuration += d
		}
		stats.AvgDuration = totalDuration / float64(len(durations))

		// Calculate P95
		sort.Float64s(durations)
		p95Index := int(float64(len(durations)) * 0.95)
		if p95Index >= len(durations) {
			p95Index = len(durations) - 1
		}
		stats.P95Duration = durations[p95Index]
	}

	// Determine trend (simplified - in production would compare to previous period)
	if stats.SuccessRate >= 95 {
		stats.TrendDirection = "stable"
	} else if stats.SuccessRate >= 80 {
		stats.TrendDirection = "declining"
	} else {
		stats.TrendDirection = "degrading"
	}

	return stats, pipelines, nil
}

// parseTestData parses test data from API response
func (c *CICDCommand) parseTestData(rawData []byte) (*TestStats, []CITest, error) {
	var apiResponse map[string]interface{}
	if err := json.Unmarshal(rawData, &apiResponse); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	stats := &TestStats{
		FailedTestNames: []string{},
	}
	var tests []CITest

	// Extract test data from API response
	data, ok := apiResponse["data"].(map[string]interface{})
	if !ok {
		return stats, tests, nil
	}

	buckets, ok := data["buckets"].([]interface{})
	if !ok {
		return stats, tests, nil
	}

	var durations []float64
	passCount := 0
	failCount := 0
	flakyCount := 0
	failedTests := make(map[string]bool)

	for _, bucket := range buckets {
		bucketMap, ok := bucket.(map[string]interface{})
		if !ok {
			continue
		}

		by, ok := bucketMap["by"].(map[string]interface{})
		if !ok {
			continue
		}

		computes, ok := bucketMap["computes"].(map[string]interface{})
		if !ok {
			continue
		}

		// Extract test information
		test := CITest{
			TestName:   getStringValue(by, "test.name"),
			Suite:      getStringValue(by, "test.suite"),
			Status:     getStringValue(by, "test.status"),
			Repository: getStringValue(by, "git.repository"),
			Branch:     getStringValue(by, "git.branch"),
		}

		// Check for flaky tests (tests that sometimes pass, sometimes fail)
		if isFlaky, ok := by["test.is_flaky"].(bool); ok {
			test.Flaky = isFlaky
			if isFlaky {
				flakyCount++
			}
		}

		// Extract metrics
		if duration, ok := computes["c0"].(float64); ok {
			test.Duration = duration / 1_000_000 // Convert ns to ms
			durations = append(durations, test.Duration)
		}

		if count, ok := computes["c1"].(float64); ok {
			stats.TotalTests += int(count)

			if test.Status == "passed" || test.Status == "pass" {
				passCount += int(count)
			} else if test.Status == "failed" || test.Status == "fail" {
				failCount += int(count)
				if !failedTests[test.TestName] {
					failedTests[test.TestName] = true
					stats.FailedTestNames = append(stats.FailedTestNames, test.TestName)
				}
			}
		}

		tests = append(tests, test)
	}

	// Calculate statistics
	if stats.TotalTests > 0 {
		stats.PassRate = float64(passCount) / float64(stats.TotalTests) * 100
		stats.FailRate = float64(failCount) / float64(stats.TotalTests) * 100
	}

	stats.FlakyTests = flakyCount

	if len(durations) > 0 {
		var totalDuration float64
		for _, d := range durations {
			totalDuration += d
		}
		stats.AvgDuration = totalDuration / float64(len(durations))
	}

	// Limit failed test names to top 10
	if len(stats.FailedTestNames) > 10 {
		stats.FailedTestNames = stats.FailedTestNames[:10]
	}

	return stats, tests, nil
}

// calculateDORAMetrics calculates DORA metrics from deployment data
func (c *CICDCommand) calculateDORAMetrics(rawData []byte, from, to time.Time) (*DORAMetrics, error) {
	var apiResponse map[string]interface{}
	if err := json.Unmarshal(rawData, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	dora := &DORAMetrics{}

	// Extract deployment events
	data, ok := apiResponse["data"].([]interface{})
	if !ok {
		return dora, nil
	}

	var deploymentTimes []time.Time
	var leadTimes []float64
	failedCount := 0

	for _, event := range data {
		eventMap, ok := event.(map[string]interface{})
		if !ok {
			continue
		}

		attrs, ok := eventMap["attributes"].(map[string]interface{})
		if !ok {
			continue
		}

		status := getStringValue(attrs, "status")

		if timestamp := getStringValue(attrs, "timestamp"); timestamp != "" {
			if t, err := time.Parse(time.RFC3339, timestamp); err == nil {
				deploymentTimes = append(deploymentTimes, t)
			}
		}

		dora.DeploymentCount++

		if status == "failed" || status == "error" {
			failedCount++
		}

		// Extract lead time (time from commit to deployment)
		if leadTimeNs, ok := attrs["git.commit.lead_time"].(float64); ok {
			leadTimes = append(leadTimes, leadTimeNs/1_000_000_000/3600) // Convert ns to hours
		}
	}

	dora.FailedDeployments = failedCount

	// Calculate deployment frequency (deployments per day)
	if dora.DeploymentCount > 0 {
		daysElapsed := to.Sub(from).Hours() / 24
		if daysElapsed > 0 {
			dora.DeploymentFrequency = float64(dora.DeploymentCount) / daysElapsed
		}

		// Calculate change failure rate
		dora.ChangeFailureRate = float64(failedCount) / float64(dora.DeploymentCount) * 100
	}

	// Calculate average lead time
	if len(leadTimes) > 0 {
		var totalLeadTime float64
		for _, lt := range leadTimes {
			totalLeadTime += lt
		}
		dora.LeadTime = totalLeadTime / float64(len(leadTimes))
	}

	// Calculate MTTR (Mean Time To Recovery)
	// This would typically require incident data - simplified here
	if failedCount > 0 {
		// Estimate based on deployment frequency
		dora.MTTR = 24.0 / dora.DeploymentFrequency // Simplified calculation
	}

	return dora, nil
}

// printFormatted prints the CI/CD response in a conversational format
func (c *CICDCommand) printFormatted(response *CICDResponse) {
	fmt.Println("CI/CD (CI Visibility) Analysis")
	if response.Service != "" {
		fmt.Printf("Service: %s\n", response.Service)
	}
	fmt.Printf("Duration: %s\n", response.Duration)
	if response.Branch != "" {
		fmt.Printf("Branch: %s\n", response.Branch)
	}
	fmt.Println()

	// Pipeline stats
	if response.PipelineStats != nil {
		stats := response.PipelineStats
		fmt.Println("Pipeline Performance:")
		fmt.Printf("  Total Runs: %d\n", stats.TotalRuns)
		fmt.Printf("  Success Rate: %.1f%%\n", stats.SuccessRate)
		fmt.Printf("  Failure Rate: %.1f%%\n", stats.FailureRate)
		fmt.Printf("  Average Duration: %.0fms\n", stats.AvgDuration)
		fmt.Printf("  P95 Duration: %.0fms\n", stats.P95Duration)
		fmt.Printf("  Trend: %s\n", stats.TrendDirection)

		if len(response.RecentPipelines) > 0 {
			fmt.Println("\n  Recent Pipeline Runs:")
			for i, pipeline := range response.RecentPipelines {
				if i >= 5 {
					break
				}
				fmt.Printf("    - %s (%s): %.0fms [%s]\n",
					truncateString(pipeline.PipelineID, 40),
					pipeline.Status,
					pipeline.Duration,
					pipeline.Branch)
			}
		}
		fmt.Println()
	}

	// Test stats
	if response.TestStats != nil {
		stats := response.TestStats
		fmt.Println("Test Performance:")
		fmt.Printf("  Total Tests: %d\n", stats.TotalTests)
		fmt.Printf("  Pass Rate: %.1f%%\n", stats.PassRate)
		fmt.Printf("  Fail Rate: %.1f%%\n", stats.FailRate)
		fmt.Printf("  Flaky Tests: %d\n", stats.FlakyTests)
		fmt.Printf("  Average Duration: %.0fms\n", stats.AvgDuration)

		if len(stats.FailedTestNames) > 0 {
			fmt.Println("\n  Failed Tests:")
			for i, testName := range stats.FailedTestNames {
				if i >= 5 {
					break
				}
				fmt.Printf("    - %s\n", truncateString(testName, 80))
			}
		}

		if stats.FlakyTests > 0 {
			fmt.Printf("\n  Warning: %d flaky tests detected (tests that intermittently fail)\n", stats.FlakyTests)
		}
		fmt.Println()
	}

	// DORA metrics
	if response.DORAMetrics != nil {
		dora := response.DORAMetrics
		fmt.Println("DORA Metrics (DevOps Performance):")
		fmt.Printf("  Deployment Frequency: %.2f deployments/day\n", dora.DeploymentFrequency)
		fmt.Printf("  Lead Time for Changes: %.1f hours\n", dora.LeadTime)
		fmt.Printf("  Change Failure Rate: %.1f%%\n", dora.ChangeFailureRate)
		fmt.Printf("  Mean Time to Recovery: %.1f hours\n", dora.MTTR)
		fmt.Printf("  Total Deployments: %d (%d failed)\n", dora.DeploymentCount, dora.FailedDeployments)

		fmt.Println("\n  Performance Classification:")
		c.printDORAClassification(dora)
		fmt.Println()
	}

	// No data case
	if response.PipelineStats == nil && response.TestStats == nil && response.DORAMetrics == nil {
		fmt.Println("No CI/CD data found for the specified criteria")
	}
}

// printDORAClassification prints the DORA performance classification
func (c *CICDCommand) printDORAClassification(dora *DORAMetrics) {
	// Classification based on DORA research
	// Elite: Deploy on-demand (multiple times per day), lead time < 1 hour, change failure < 15%, MTTR < 1 hour
	// High: Deploy between weekly and monthly, lead time < 1 day, change failure < 15%, MTTR < 1 day
	// Medium: Deploy monthly to every 6 months, lead time < 1 week, change failure 16-30%, MTTR < 1 week
	// Low: Deploy less than every 6 months, lead time > 6 months, change failure > 30%, MTTR > 1 week

	eliteScore := 0
	highScore := 0
	mediumScore := 0

	// Deployment frequency
	if dora.DeploymentFrequency >= 3 {
		fmt.Println("    Deployment Frequency: Elite (multiple times per day)")
		eliteScore++
	} else if dora.DeploymentFrequency >= 0.14 { // ~1 per week
		fmt.Println("    Deployment Frequency: High (weekly)")
		highScore++
	} else if dora.DeploymentFrequency >= 0.03 { // ~1 per month
		fmt.Println("    Deployment Frequency: Medium (monthly)")
		mediumScore++
	} else {
		fmt.Println("    Deployment Frequency: Low (less than monthly)")
	}

	// Lead time
	if dora.LeadTime < 1 {
		fmt.Println("    Lead Time: Elite (< 1 hour)")
		eliteScore++
	} else if dora.LeadTime < 24 {
		fmt.Println("    Lead Time: High (< 1 day)")
		highScore++
	} else if dora.LeadTime < 168 { // 1 week
		fmt.Println("    Lead Time: Medium (< 1 week)")
		mediumScore++
	} else {
		fmt.Println("    Lead Time: Low (> 1 week)")
	}

	// Change failure rate
	if dora.ChangeFailureRate < 15 {
		fmt.Println("    Change Failure Rate: Elite/High (< 15%)")
		eliteScore++
	} else if dora.ChangeFailureRate < 30 {
		fmt.Println("    Change Failure Rate: Medium (15-30%)")
		mediumScore++
	} else {
		fmt.Println("    Change Failure Rate: Low (> 30%)")
	}

	// MTTR
	if dora.MTTR < 1 {
		fmt.Println("    MTTR: Elite (< 1 hour)")
		eliteScore++
	} else if dora.MTTR < 24 {
		fmt.Println("    MTTR: High (< 1 day)")
		highScore++
	} else if dora.MTTR < 168 {
		fmt.Println("    MTTR: Medium (< 1 week)")
		mediumScore++
	} else {
		fmt.Println("    MTTR: Low (> 1 week)")
	}

	// Overall classification
	fmt.Println()
	if eliteScore >= 3 {
		fmt.Println("  Overall Performance: Elite")
	} else if eliteScore + highScore >= 3 {
		fmt.Println("  Overall Performance: High")
	} else if mediumScore >= 2 {
		fmt.Println("  Overall Performance: Medium")
	} else {
		fmt.Println("  Overall Performance: Low - Consider improving CI/CD practices")
	}
}

// Help prints the help message
func (c *CICDCommand) Help() {
	fmt.Println("Usage: dd cicd [options]")
	fmt.Println()
	fmt.Println("Query CI/CD (CI Visibility) data for pipeline and test analysis")
	fmt.Println("Includes DORA metrics for DevOps performance measurement")
	fmt.Println()
	fmt.Println("Options:")
	c.flags.PrintDefaults()
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  dd cicd")
	fmt.Println("  dd cicd --service my-service")
	fmt.Println("  dd cicd --metric pipelines --duration 24h")
	fmt.Println("  dd cicd --metric tests --status failed")
	fmt.Println("  dd cicd --metric deployments --branch main")
	fmt.Println("  dd cicd --metric all --duration 30d --json")
	fmt.Println()
	fmt.Println("DORA Metrics:")
	fmt.Println("  Deployment Frequency - How often you deploy to production")
	fmt.Println("  Lead Time - Time from commit to production")
	fmt.Println("  Change Failure Rate - Percentage of deployments causing failures")
	fmt.Println("  Mean Time to Recovery - Time to recover from failures")
}

// Helper functions

func getStringValue(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
