package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"time"

	"github.com/datadog/skill/internal/client"
	"github.com/datadog/skill/internal/context"
	"github.com/datadog/skill/internal/observability"
)

// DeployCommand checks deploy readiness based on service health
type DeployCommand struct {
	flags      *flag.FlagSet
	service    string
	duration   int
	jsonOutput bool
	wd         string
}

// DeployDecision represents the deploy readiness decision
type DeployDecision struct {
	CanDeploy    bool          `json:"can_deploy"`
	Status       string        `json:"status"` // safe, blocked
	Blockers     []string      `json:"blockers"`
	Concerns     []string      `json:"concerns"`
	HealthReport *HealthReport `json:"health_report"`
	CheckedAt    time.Time     `json:"checked_at"`
}

// NewDeployCommand creates a new deploy command instance
func NewDeployCommand() *DeployCommand {
	cmd := &DeployCommand{
		flags: flag.NewFlagSet("deploy", flag.ExitOnError),
	}

	cmd.flags.StringVar(&cmd.service, "service", "", "Service name (auto-detect if not provided)")
	cmd.flags.IntVar(&cmd.duration, "duration", 1, "Hours to look back (default: 1)")
	cmd.flags.BoolVar(&cmd.jsonOutput, "json", false, "Output as JSON")
	cmd.flags.StringVar(&cmd.wd, "working-dir", ".", "Working directory")

	return cmd
}

func (d *DeployCommand) Name() string {
	return "deploy"
}

func (d *DeployCommand) Description() string {
	return "Check if it's safe to deploy"
}

func (d *DeployCommand) Run(args []string) error {
	// Initialize observability
	obs, err := observability.Init("dd-cli", "production")
	if err != nil {
		return fmt.Errorf("failed to init observability: %w", err)
	}
	defer obs.Shutdown(0)

	// Parse flags
	if err := d.flags.Parse(args); err != nil {
		return err
	}

	// Start tracing
	span := obs.StartSpan("deploy.check")
	defer obs.FinishSpan(span)

	obs.LogInfo("Starting deploy readiness check")

	// Detect service if not provided
	var ctx *context.ServiceContext
	if d.service == "" {
		obs.LogInfo("Auto-detecting service context")
		ctx, err = context.DetectContext(d.wd)
		if err != nil {
			obs.LogError("Failed to detect service context: " + err.Error())
			return fmt.Errorf("could not detect service name. Specify with --service or run in a git repository: %w", err)
		}
		d.service = ctx.ServiceName
	} else {
		ctx = &context.ServiceContext{
			ServiceName: d.service,
		}
	}

	obs.GetTracer().SetTag(span, "service.name", d.service)
	obs.GetTracer().SetTag(span, "check.duration_hours", d.duration)

	// Create Datadog client
	ddClient, err := client.NewClient()
	if err != nil {
		obs.LogError("Failed to create Datadog client: " + err.Error())
		return fmt.Errorf("failed to create Datadog client: %w", err)
	}

	// Perform comprehensive health check
	healthCmd := NewHealthCommand()
	healthCmd.service = d.service
	healthCmd.duration = d.duration
	healthCmd.wd = d.wd
	healthCmd.sinceDeploy = true // Focus on changes since last deploy

	report := healthCmd.analyzeHealth(ddClient, ctx, obs, span)

	// Make deploy decision
	decision := d.makeDeployDecision(report)

	obs.GetTracer().SetTag(span, "deploy.can_deploy", decision.CanDeploy)
	obs.GetTracer().SetTag(span, "deploy.status", decision.Status)
	obs.GetTracer().SetTag(span, "deploy.blockers_count", len(decision.Blockers))
	obs.GetTracer().SetTag(span, "deploy.concerns_count", len(decision.Concerns))

	// Record metrics
	deployValue := 0.0
	if decision.CanDeploy {
		deployValue = 1.0
	}

	obs.GetMetrics().Gauge("deploy.readiness",
		deployValue,
		"service:"+d.service,
		"status:"+decision.Status,
	)

	// Output decision
	if err := d.outputDecision(decision); err != nil {
		obs.LogError("Failed to output decision: " + err.Error())
		return err
	}

	obs.LogInfo("Deploy readiness check complete")

	// Exit with error code if not safe to deploy
	if !decision.CanDeploy {
		return fmt.Errorf("deployment NOT recommended - %d blocking issues", len(decision.Blockers))
	}

	return nil
}

func (d *DeployCommand) makeDeployDecision(report *HealthReport) *DeployDecision {
	decision := &DeployDecision{
		CanDeploy:    true,
		Status:       "safe",
		Blockers:     []string{},
		Concerns:     []string{},
		HealthReport: report,
		CheckedAt:    time.Now(),
	}

	// Analyze critical issues (blockers)
	for _, issue := range report.Issues {
		if issue.Severity == "critical" {
			decision.CanDeploy = false
			decision.Status = "blocked"

			blockerMsg := fmt.Sprintf("%s: %s", issue.Category, issue.Message)
			decision.Blockers = append(decision.Blockers, blockerMsg)
		}
	}

	// Analyze warnings (concerns)
	for _, issue := range report.Issues {
		if issue.Severity == "warning" {
			concernMsg := fmt.Sprintf("%s: %s", issue.Category, issue.Message)
			decision.Concerns = append(decision.Concerns, concernMsg)
		}
	}

	// Additional checks for deploy safety

	// Check error logs
	if errorLogs, ok := report.Metrics["error_logs"].(int); ok && errorLogs > 50 {
		decision.CanDeploy = false
		decision.Status = "blocked"
		decision.Blockers = append(decision.Blockers,
			fmt.Sprintf("High error rate: %d errors in recent logs", errorLogs))
	}

	// Check breaching SLOs
	if breachingSLOs, ok := report.Metrics["breaching_slos"].(int); ok && breachingSLOs > 0 {
		decision.CanDeploy = false
		decision.Status = "blocked"
		decision.Blockers = append(decision.Blockers,
			fmt.Sprintf("SLOs breaching: %d SLO(s) not meeting targets", breachingSLOs))
	}

	// Check security signals
	if securitySignals, ok := report.Metrics["security_signals"].(int); ok && securitySignals > 0 {
		// Count critical security signals from issues
		criticalSecurity := false
		for _, issue := range report.Issues {
			if issue.Category == "security" && issue.Severity == "critical" {
				criticalSecurity = true
				break
			}
		}

		if criticalSecurity {
			decision.CanDeploy = false
			decision.Status = "blocked"
			decision.Blockers = append(decision.Blockers,
				fmt.Sprintf("Critical security signals: %d security issues detected", securitySignals))
		} else if securitySignals > 0 {
			decision.Concerns = append(decision.Concerns,
				fmt.Sprintf("Security signals: %d security issues to monitor", securitySignals))
		}
	}

	// Note: If status is healthy or degraded (but no critical issues), safe to deploy
	// Degraded with warnings = can deploy but monitor closely
	if decision.CanDeploy && report.Status == "degraded" {
		decision.Status = "safe" // Still safe, just have concerns
	}

	return decision
}

func (d *DeployCommand) outputDecision(decision *DeployDecision) error {
	if d.jsonOutput {
		jsonData, err := json.MarshalIndent(decision, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	} else {
		d.printFormattedDecision(decision)
	}
	return nil
}

func (d *DeployCommand) printFormattedDecision(decision *DeployDecision) {
	// Header
	if decision.CanDeploy {
		fmt.Printf("✓ Deploy Readiness: **%s**\n\n", decision.HealthReport.Service)
		fmt.Println("✓ Safe to deploy")
	} else {
		fmt.Printf("✗ Deploy Readiness: **%s**\n\n", decision.HealthReport.Service)
		fmt.Println("✗ Deployment NOT recommended")
	}

	fmt.Println()

	// Blockers
	if len(decision.Blockers) > 0 {
		fmt.Println("Blocking Issues:")
		for _, blocker := range decision.Blockers {
			fmt.Printf("  • %s\n", blocker)
		}
		fmt.Println()
	}

	// Concerns
	if len(decision.Concerns) > 0 {
		fmt.Println("Concerns:")
		for _, concern := range decision.Concerns {
			fmt.Printf("  • %s\n", concern)
		}
		fmt.Println()
	}

	// Metrics summary
	metrics := decision.HealthReport.Metrics

	if errorLogs, ok := metrics["error_logs"].(int); ok && errorLogs > 0 {
		fmt.Printf("Error rate: %d errors in monitoring period\n", errorLogs)
	}

	if slowEndpoints, ok := metrics["slow_endpoints"].(int); ok && slowEndpoints > 0 {
		fmt.Printf("Slow endpoints: %d endpoints > 500ms P95\n", slowEndpoints)
	}

	if breachingSLOs, ok := metrics["breaching_slos"].(int); ok && breachingSLOs > 0 {
		fmt.Printf("SLOs: %d breaching targets\n", breachingSLOs)
	}

	if securitySignals, ok := metrics["security_signals"].(int); ok && securitySignals > 0 {
		fmt.Printf("Security signals: %d detected\n", securitySignals)
	}

	fmt.Println()

	// Final recommendation
	if decision.CanDeploy {
		if decision.HealthReport.Status == "healthy" {
			fmt.Println("Go ahead and deploy!")
		} else {
			fmt.Println("You can deploy, but monitor closely")
		}
	} else {
		fmt.Println("Fix critical issues before deploying")
	}
}

func (d *DeployCommand) Help() {
	fmt.Println("Usage: dd deploy [options]")
	fmt.Println()
	fmt.Println("Check if it's safe to deploy")
	fmt.Println()
	fmt.Println("Analyzes recent service health (APM, logs, errors, SLOs, security)")
	fmt.Println("and provides a clear GO/NO-GO decision for deployment.")
	fmt.Println()
	fmt.Println("Options:")
	d.flags.PrintDefaults()
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  dd deploy")
	fmt.Println("  dd deploy --service my-service")
	fmt.Println("  dd deploy --duration 2")
	fmt.Println("  dd deploy --json")
	fmt.Println()
	fmt.Println("Exit Codes:")
	fmt.Println("  0 = Safe to deploy (healthy or degraded with no blockers)")
	fmt.Println("  1 = Not safe to deploy (critical issues present)")
}
