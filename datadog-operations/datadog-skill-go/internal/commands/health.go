package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"sync"
	"time"

	"github.com/datadog/skill/internal/client"
	"github.com/datadog/skill/internal/context"
	"github.com/datadog/skill/internal/observability"
)

// HealthCommand performs comprehensive health checks across APM, logs, errors, and SLOs
type HealthCommand struct {
	flags       *flag.FlagSet
	service     string
	duration    int
	sinceDeploy bool
	jsonOutput  bool
	summary     bool
	wd          string
}

// HealthIssue represents a health issue detected during analysis
type HealthIssue struct {
	Severity string                 `json:"severity"` // critical, warning, info
	Category string                 `json:"category"` // performance, errors, security, slo, monitoring
	Message  string                 `json:"message"`
	Details  map[string]interface{} `json:"details,omitempty"`
	Action   string                 `json:"action,omitempty"`
}

// HealthReport represents the comprehensive health analysis results
type HealthReport struct {
	Status          string                 `json:"status"` // healthy, degraded, critical
	Service         string                 `json:"service"`
	CheckedAt       time.Time              `json:"checked_at"`
	Issues          []HealthIssue          `json:"issues"`
	Metrics         map[string]interface{} `json:"metrics"`
	Recommendations []string               `json:"recommendations"`
}

// NewHealthCommand creates a new health command instance
func NewHealthCommand() *HealthCommand {
	cmd := &HealthCommand{
		flags: flag.NewFlagSet("health", flag.ExitOnError),
	}

	cmd.flags.StringVar(&cmd.service, "service", "", "Service name (auto-detect if not provided)")
	cmd.flags.IntVar(&cmd.duration, "duration", 1, "Hours to look back (default: 1)")
	cmd.flags.BoolVar(&cmd.sinceDeploy, "since-deploy", false, "Check since last deploy")
	cmd.flags.BoolVar(&cmd.jsonOutput, "json", false, "Output as JSON")
	cmd.flags.BoolVar(&cmd.summary, "summary", false, "Ultra-short summary (one line)")
	cmd.flags.StringVar(&cmd.wd, "working-dir", ".", "Working directory")

	return cmd
}

func (h *HealthCommand) Name() string {
	return "health"
}

func (h *HealthCommand) Description() string {
	return "Smart health check for your service"
}

func (h *HealthCommand) Run(args []string) error {
	// Initialize observability
	obs, err := observability.Init("dd-cli", "production")
	if err != nil {
		return fmt.Errorf("failed to init observability: %w", err)
	}
	defer obs.Shutdown(0)

	// Parse flags
	if err := h.flags.Parse(args); err != nil {
		return err
	}

	// Start tracing
	span := obs.StartSpan("health.check")
	defer obs.FinishSpan(span)

	obs.LogInfo("Starting health check")

	// Detect service if not provided
	var ctx *context.ServiceContext
	if h.service == "" {
		obs.LogInfo("Auto-detecting service context")
		ctx, err = context.DetectContext(h.wd)
		if err != nil {
			obs.LogError("Failed to detect service context: " + err.Error())
			return fmt.Errorf("could not detect service name. Specify with --service or run in a git repository: %w", err)
		}
		h.service = ctx.ServiceName
	} else {
		ctx = &context.ServiceContext{
			ServiceName: h.service,
		}
	}

	obs.GetTracer().SetTag(span, "service.name", h.service)
	obs.GetTracer().SetTag(span, "check.duration_hours", h.duration)
	obs.GetTracer().SetTag(span, "check.since_deploy", h.sinceDeploy)

	// Create Datadog client
	ddClient, err := client.NewClient()
	if err != nil {
		obs.LogError("Failed to create Datadog client: " + err.Error())
		return fmt.Errorf("failed to create Datadog client: %w", err)
	}

	// Perform health analysis
	report := h.analyzeHealth(ddClient, ctx, obs, span)

	obs.GetTracer().SetTag(span, "health.status", report.Status)
	obs.GetTracer().SetTag(span, "health.issues_count", len(report.Issues))

	// Record metrics
	obs.GetMetrics().Gauge("health.check.status",
		h.statusToValue(report.Status),
		"service:"+h.service,
		"status:"+report.Status,
	)

	// Output results
	if err := h.outputReport(report); err != nil {
		obs.LogError("Failed to output report: " + err.Error())
		return err
	}

	obs.LogInfo("Health check complete")

	// Exit with error code if critical
	if report.Status == "critical" {
		return fmt.Errorf("service health is critical")
	}

	return nil
}

func (h *HealthCommand) analyzeHealth(
	ddClient *client.Client,
	ctx *context.ServiceContext,
	obs *observability.Observability,
	parentSpan interface{},
) *HealthReport {
	report := &HealthReport{
		Status:          "healthy",
		Service:         h.service,
		CheckedAt:       time.Now(),
		Issues:          []HealthIssue{},
		Metrics:         make(map[string]interface{}),
		Recommendations: []string{},
	}

	// Determine time range
	toTime := time.Now()
	fromTime := toTime.Add(-time.Duration(h.duration) * time.Hour)

	// If checking since deploy, use last commit time (capped at 24 hours)
	if h.sinceDeploy && !ctx.LastCommitTime.IsZero() {
		maxLookback := toTime.Add(-24 * time.Hour)
		if ctx.LastCommitTime.After(maxLookback) {
			fromTime = ctx.LastCommitTime
		} else {
			fromTime = maxLookback
		}
	}

	// Run checks in parallel
	var wg sync.WaitGroup
	var mu sync.Mutex

	checks := []struct {
		name string
		fn   func()
	}{
		{"apm_performance", func() { h.checkAPMPerformance(ddClient, fromTime, toTime, report, &mu) }},
		{"error_logs", func() { h.checkErrorLogs(ddClient, fromTime, toTime, report, &mu) }},
		{"security_signals", func() { h.checkSecuritySignals(ddClient, fromTime, toTime, report, &mu) }},
		{"slos", func() { h.checkSLOs(ddClient, report, &mu) }},
	}

	wg.Add(len(checks))
	for _, check := range checks {
		go func(name string, fn func()) {
			defer wg.Done()
			checkSpan := obs.StartSpan("health.check." + name)
			defer obs.FinishSpan(checkSpan)
			fn()
		}(check.name, check.fn)
	}

	wg.Wait()

	// Calculate overall status
	report.Status = h.calculateOverallStatus(report)

	// Generate recommendations
	report.Recommendations = h.generateRecommendations(report)

	return report
}

func (h *HealthCommand) checkAPMPerformance(
	ddClient *client.Client,
	fromTime, toTime time.Time,
	report *HealthReport,
	mu *sync.Mutex,
) {
	data, err := ddClient.QueryAPM(h.service, fromTime, toTime, "")
	if err != nil {
		mu.Lock()
		report.Issues = append(report.Issues, HealthIssue{
			Severity: "info",
			Category: "monitoring",
			Message:  fmt.Sprintf("Could not check APM performance: %v", err),
		})
		mu.Unlock()
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return
	}

	dataMap, ok := result["data"].(map[string]interface{})
	if !ok {
		return
	}

	buckets, ok := dataMap["buckets"].([]interface{})
	if !ok || len(buckets) == 0 {
		mu.Lock()
		report.Metrics["apm_endpoints"] = 0
		mu.Unlock()
		return
	}

	mu.Lock()
	defer mu.Unlock()

	totalRequests := 0
	slowEndpoints := []map[string]interface{}{}

	for _, bucketInterface := range buckets {
		bucket, ok := bucketInterface.(map[string]interface{})
		if !ok {
			continue
		}

		by, _ := bucket["by"].(map[string]interface{})
		resourceName, _ := by["resource_name"].(string)
		if resourceName == "" {
			resourceName = "unknown"
		}

		computes, _ := bucket["computes"].(map[string]interface{})
		count, _ := computes["c0"].(float64)
		p95Ns, _ := computes["c2"].(float64)
		p95Ms := p95Ns / 1_000_000

		totalRequests += int(count)

		// Flag slow endpoints (P95 > 500ms)
		if p95Ms > 500 {
			slowEndpoints = append(slowEndpoints, map[string]interface{}{
				"endpoint":  resourceName,
				"p95_ms":    int(p95Ms),
				"requests":  int(count),
			})
		}
	}

	report.Metrics["total_requests"] = totalRequests
	report.Metrics["endpoints_checked"] = len(buckets)
	report.Metrics["slow_endpoints"] = len(slowEndpoints)

	// Report slow endpoints (top 5)
	for i, endpoint := range slowEndpoints {
		if i >= 5 {
			break
		}
		report.Issues = append(report.Issues, HealthIssue{
			Severity: "warning",
			Category: "performance",
			Message:  fmt.Sprintf("Slow endpoint: %v", endpoint["endpoint"]),
			Details: map[string]interface{}{
				"p95_latency_ms": endpoint["p95_ms"],
				"request_count":  endpoint["requests"],
			},
			Action: "Investigate performance bottleneck or scale resources",
		})
	}
}

func (h *HealthCommand) checkErrorLogs(
	ddClient *client.Client,
	fromTime, toTime time.Time,
	report *HealthReport,
	mu *sync.Mutex,
) {
	query := fmt.Sprintf("service:%s status:error", h.service)
	data, err := ddClient.SearchLogs(query, fromTime, toTime, 100)
	if err != nil {
		mu.Lock()
		report.Issues = append(report.Issues, HealthIssue{
			Severity: "info",
			Category: "monitoring",
			Message:  fmt.Sprintf("Could not check error logs: %v", err),
		})
		mu.Unlock()
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return
	}

	logs, ok := result["data"].([]interface{})
	if !ok {
		mu.Lock()
		report.Metrics["error_logs"] = 0
		mu.Unlock()
		return
	}

	mu.Lock()
	defer mu.Unlock()

	errorCount := len(logs)
	report.Metrics["error_logs"] = errorCount

	if errorCount > 0 {
		// Analyze error patterns
		errorMessages := make(map[string]int)
		for _, logInterface := range logs {
			log, ok := logInterface.(map[string]interface{})
			if !ok {
				continue
			}

			attrs, _ := log["attributes"].(map[string]interface{})
			message, _ := attrs["message"].(string)
			if len(message) > 100 {
				message = message[:100]
			}
			errorMessages[message]++
		}

		// Find top errors
		type errorEntry struct {
			message string
			count   int
		}
		var topErrors []errorEntry
		for msg, count := range errorMessages {
			topErrors = append(topErrors, errorEntry{msg, count})
		}

		// Sort by count (simple bubble sort for top 3)
		for i := 0; i < len(topErrors) && i < 3; i++ {
			for j := i + 1; j < len(topErrors); j++ {
				if topErrors[j].count > topErrors[i].count {
					topErrors[i], topErrors[j] = topErrors[j], topErrors[i]
				}
			}
		}

		// Take top 3
		if len(topErrors) > 3 {
			topErrors = topErrors[:3]
		}

		topErrorsList := []map[string]interface{}{}
		for _, e := range topErrors {
			topErrorsList = append(topErrorsList, map[string]interface{}{
				"message": e.message,
				"count":   e.count,
			})
		}

		severity := "warning"
		if errorCount > 50 {
			severity = "critical"
		}

		report.Issues = append(report.Issues, HealthIssue{
			Severity: severity,
			Category: "errors",
			Message:  fmt.Sprintf("Found %d error logs", errorCount),
			Details: map[string]interface{}{
				"error_count": errorCount,
				"top_errors":  topErrorsList,
			},
			Action: "Review error logs and fix recurring issues",
		})
	}
}

func (h *HealthCommand) checkSecuritySignals(
	ddClient *client.Client,
	fromTime, toTime time.Time,
	report *HealthReport,
	mu *sync.Mutex,
) {
	data, err := ddClient.GetSecuritySignals(fromTime, toTime, h.service)
	if err != nil {
		mu.Lock()
		report.Issues = append(report.Issues, HealthIssue{
			Severity: "info",
			Category: "monitoring",
			Message:  fmt.Sprintf("Could not check security signals: %v", err),
		})
		mu.Unlock()
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return
	}

	signals, ok := result["data"].([]interface{})
	if !ok {
		mu.Lock()
		report.Metrics["security_signals"] = 0
		mu.Unlock()
		return
	}

	mu.Lock()
	defer mu.Unlock()

	signalCount := len(signals)
	report.Metrics["security_signals"] = signalCount

	if signalCount > 0 {
		// Count by severity
		severities := make(map[string]int)
		for _, signalInterface := range signals {
			signal, ok := signalInterface.(map[string]interface{})
			if !ok {
				continue
			}

			attrs, _ := signal["attributes"].(map[string]interface{})
			severity, _ := attrs["severity"].(string)
			if severity == "" {
				severity = "unknown"
			}
			severities[severity]++
		}

		criticalCount := severities["critical"]
		highCount := severities["high"]

		if criticalCount > 0 || highCount > 0 {
			severity := "warning"
			if criticalCount > 0 {
				severity = "critical"
			}

			report.Issues = append(report.Issues, HealthIssue{
				Severity: severity,
				Category: "security",
				Message:  "Security signals detected",
				Details: map[string]interface{}{
					"critical": criticalCount,
					"high":     highCount,
					"total":    signalCount,
				},
				Action: "Review security signals immediately",
			})
		}
	}
}

func (h *HealthCommand) checkSLOs(
	ddClient *client.Client,
	report *HealthReport,
	mu *sync.Mutex,
) {
	data, err := ddClient.GetSLOs([]string{fmt.Sprintf("service:%s", h.service)})
	if err != nil {
		mu.Lock()
		report.Issues = append(report.Issues, HealthIssue{
			Severity: "info",
			Category: "monitoring",
			Message:  fmt.Sprintf("Could not check SLOs: %v", err),
		})
		mu.Unlock()
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return
	}

	slos, ok := result["data"].([]interface{})
	if !ok {
		mu.Lock()
		report.Metrics["slos_checked"] = 0
		mu.Unlock()
		return
	}

	mu.Lock()
	defer mu.Unlock()

	report.Metrics["slos_checked"] = len(slos)

	if len(slos) == 0 {
		return
	}

	breachingSLOs := []map[string]interface{}{}

	for _, sloInterface := range slos {
		slo, ok := sloInterface.(map[string]interface{})
		if !ok {
			continue
		}

		attrs, _ := slo["attributes"].(map[string]interface{})
		name, _ := attrs["name"].(string)
		sloValue, _ := attrs["slo_value"].(float64)
		target, _ := attrs["target_threshold"].(float64)

		if sloValue > 0 && target > 0 && sloValue < target {
			breachingSLOs = append(breachingSLOs, map[string]interface{}{
				"name":    name,
				"current": sloValue,
				"target":  target,
			})
		}
	}

	report.Metrics["breaching_slos"] = len(breachingSLOs)

	if len(breachingSLOs) > 0 {
		report.Issues = append(report.Issues, HealthIssue{
			Severity: "critical",
			Category: "slo",
			Message:  fmt.Sprintf("%d SLO(s) breaching target", len(breachingSLOs)),
			Details: map[string]interface{}{
				"slos": breachingSLOs,
			},
			Action: "Review SLO breaches and take corrective action",
		})
	}
}

func (h *HealthCommand) calculateOverallStatus(report *HealthReport) string {
	criticalCount := 0
	warningCount := 0

	for _, issue := range report.Issues {
		if issue.Severity == "critical" {
			criticalCount++
		} else if issue.Severity == "warning" {
			warningCount++
		}
	}

	if criticalCount > 0 {
		return "critical"
	} else if warningCount > 0 {
		return "degraded"
	}
	return "healthy"
}

func (h *HealthCommand) generateRecommendations(report *HealthReport) []string {
	var recommendations []string

	// Check error rate
	if errorCount, ok := report.Metrics["error_logs"].(int); ok && errorCount > 50 {
		recommendations = append(recommendations,
			fmt.Sprintf("High error rate (%d errors) - investigate root cause before deploying", errorCount))
	}

	// Check slow endpoints
	if slowCount, ok := report.Metrics["slow_endpoints"].(int); ok && slowCount > 5 {
		recommendations = append(recommendations,
			fmt.Sprintf("%d slow endpoints detected - consider optimization or scaling", slowCount))
	}

	// Check SLOs
	if breaching, ok := report.Metrics["breaching_slos"].(int); ok && breaching > 0 {
		recommendations = append(recommendations,
			"SLOs breaching - deployment not recommended until issues resolved")
	}

	// Check security
	if securitySignals, ok := report.Metrics["security_signals"].(int); ok && securitySignals > 0 {
		recommendations = append(recommendations,
			"Security signals detected - review before deploying changes")
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Service health looks good")
	}

	return recommendations
}

func (h *HealthCommand) outputReport(report *HealthReport) error {
	if h.summary {
		h.printQuickSummary(report)
	} else if h.jsonOutput {
		jsonData, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	} else {
		h.printFormattedReport(report)
	}
	return nil
}

func (h *HealthCommand) printQuickSummary(report *HealthReport) {
	if report.Status == "healthy" {
		fmt.Printf("Service %s is healthy\n", report.Service)
		return
	}

	issuesText := []string{}
	metrics := report.Metrics

	if errorLogs, ok := metrics["error_logs"].(int); ok && errorLogs > 10 {
		issuesText = append(issuesText, fmt.Sprintf("%d errors", errorLogs))
	}

	if slowEndpoints, ok := metrics["slow_endpoints"].(int); ok && slowEndpoints > 0 {
		issuesText = append(issuesText, fmt.Sprintf("%d slow endpoints", slowEndpoints))
	}

	if breachingSLOs, ok := metrics["breaching_slos"].(int); ok && breachingSLOs > 0 {
		issuesText = append(issuesText, fmt.Sprintf("%d SLOs breaching", breachingSLOs))
	}

	if len(issuesText) > 0 {
		fmt.Printf("%s: %s\n", report.Service, join(issuesText, ", "))
	} else {
		fmt.Printf("%s has minor issues\n", report.Service)
	}
}

func (h *HealthCommand) printFormattedReport(report *HealthReport) {
	statusEmoji := map[string]string{
		"healthy":  "✓",
		"degraded": "⚠",
		"critical": "✗",
	}

	emoji := statusEmoji[report.Status]
	if emoji == "" {
		emoji = "?"
	}

	fmt.Printf("%s Service: **%s**\n", emoji, report.Service)
	fmt.Printf("Status: **%s**\n\n", toUpper(report.Status))

	// Metrics
	if len(report.Metrics) > 0 {
		fmt.Println("Metrics:")
		if totalRequests, ok := report.Metrics["total_requests"].(int); ok {
			fmt.Printf("  • %s requests\n", formatNumber(int64(totalRequests)))
		}
		if endpointsChecked, ok := report.Metrics["endpoints_checked"].(int); ok {
			fmt.Printf("  • %d endpoints checked\n", endpointsChecked)
		}
		if errorLogs, ok := report.Metrics["error_logs"].(int); ok {
			fmt.Printf("  • %d error logs\n", errorLogs)
		}
		if securitySignals, ok := report.Metrics["security_signals"].(int); ok {
			fmt.Printf("  • %d security signals\n", securitySignals)
		}
		if slosChecked, ok := report.Metrics["slos_checked"].(int); ok {
			fmt.Printf("  • %d SLOs monitored\n", slosChecked)
		}
		fmt.Println()
	}

	// Issues
	if len(report.Issues) > 0 {
		critical := []HealthIssue{}
		warnings := []HealthIssue{}

		for _, issue := range report.Issues {
			if issue.Severity == "critical" {
				critical = append(critical, issue)
			} else if issue.Severity == "warning" {
				warnings = append(warnings, issue)
			}
		}

		if len(critical) > 0 {
			fmt.Println("Critical Issues:")
			for _, issue := range critical {
				fmt.Printf("  • %s\n", issue.Message)
				if issue.Action != "" {
					fmt.Printf("    Action: %s\n", issue.Action)
				}
			}
			fmt.Println()
		}

		if len(warnings) > 0 {
			fmt.Println("Warnings:")
			for _, issue := range warnings {
				fmt.Printf("  • %s\n", issue.Message)
				if issue.Action != "" {
					fmt.Printf("    Action: %s\n", issue.Action)
				}
			}
			fmt.Println()
		}
	}

	// Recommendations
	if len(report.Recommendations) > 0 {
		fmt.Println("Recommendations:")
		for _, rec := range report.Recommendations {
			fmt.Printf("  • %s\n", rec)
		}
		fmt.Println()
	}
}

func (h *HealthCommand) statusToValue(status string) float64 {
	switch status {
	case "healthy":
		return 1.0
	case "degraded":
		return 0.5
	case "critical":
		return 0.0
	default:
		return 0.0
	}
}

func (h *HealthCommand) Help() {
	fmt.Println("Usage: dd health [options]")
	fmt.Println()
	fmt.Println("Smart health check for your service")
	fmt.Println()
	fmt.Println("Analyzes APM traces, error logs, security signals, and SLOs to provide")
	fmt.Println("comprehensive health status with actionable recommendations.")
	fmt.Println()
	fmt.Println("Options:")
	h.flags.PrintDefaults()
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  dd health")
	fmt.Println("  dd health --service my-service")
	fmt.Println("  dd health --duration 24")
	fmt.Println("  dd health --since-deploy")
	fmt.Println("  dd health --json")
	fmt.Println("  dd health --summary")
}

// Utility functions

func join(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

func toUpper(s string) string {
	if s == "" {
		return s
	}
	bytes := []byte(s)
	for i := 0; i < len(bytes); i++ {
		if bytes[i] >= 'a' && bytes[i] <= 'z' {
			bytes[i] = bytes[i] - 'a' + 'A'
		}
	}
	return string(bytes)
}
