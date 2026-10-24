package main

import (
	"fmt"
	"os"

	"github.com/datadog/skill/internal/commands"
)

// Version information set via ldflags during build
var (
	version   = "0.1.0"
	commit    = "unknown"
	buildDate = "unknown"
)

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}

	commandName := os.Args[1]
	commandArgs := os.Args[2:]

	// Handle built-in commands
	switch commandName {
	case "version", "--version", "-v":
		printVersion()
		return
	case "help", "--help", "-h":
		printHelp()
		return
	}

	// Get command
	cmd := getCommand(commandName)
	if cmd == nil {
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", commandName)
		printHelp()
		os.Exit(1)
	}

	// Check for --help flag
	for _, arg := range commandArgs {
		if arg == "--help" || arg == "-h" {
			cmd.Help()
			return
		}
	}

	// Run command
	if err := cmd.Run(commandArgs); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func getCommand(name string) commands.Command {
	switch name {
	case "context":
		return commands.NewContextCommand()
	case "apm":
		return commands.NewAPMCommand()
	case "logs":
		return commands.NewLogsCommand()
	case "metrics":
		return commands.NewMetricsCommand()
	case "llm":
		return commands.NewLLMCommand()
	case "database":
		return commands.NewDatabaseCommand()
	case "security":
		return commands.NewSecurityCommand()
	case "slos":
		return commands.NewSLOsCommand()
	case "catalog":
		return commands.NewCatalogCommand()
	case "health":
		return commands.NewHealthCommand()
	case "deploy":
		return commands.NewDeployCommand()
	case "incidents":
		return commands.NewIncidentsCommand()
	case "monitors":
		return commands.NewMonitorsCommand()
	case "watchdog":
		return commands.NewWatchdogCommand()
	case "cost":
		return commands.NewCostCommand()
	case "dashboards":
		return commands.NewDashboardsCommand()
	case "workflows":
		return commands.NewWorkflowsCommand()
	case "synthetics":
		return commands.NewSyntheticsCommand()
	case "rum":
		return commands.NewRUMCommand()
	case "network":
		return commands.NewNetworkCommand()
	case "cicd":
		return commands.NewCICDCommand()
	case "dora":
		return commands.NewDORACommand()
	case "cases":
		return commands.NewCasesCommand()
	case "containers":
		return commands.NewContainersCommand()
	case "kubernetes":
		return commands.NewKubernetesCommand()
	case "serverless":
		return commands.NewServerlessCommand()
	case "status-pages":
		return commands.NewStatusPagesCommand()
	case "on-call":
		return commands.NewOnCallCommand()
	case "downtimes":
		return commands.NewDowntimesCommand()
	case "notebooks":
		return commands.NewNotebooksCommand()
	case "teams":
		return commands.NewTeamsCommand()
	case "users":
		return commands.NewUsersCommand()
	case "roles":
		return commands.NewRolesCommand()
	case "service-accounts":
		return commands.NewServiceAccountsCommand()
	case "application-keys":
		return commands.NewApplicationKeysCommand()
	case "api-keys":
		return commands.NewAPIKeysCommand()
	case "audit-logs":
		return commands.NewAuditLogsCommand()
	case "slo-corrections":
		return commands.NewSLOCorrectionsCommand()
	case "error-budgets":
		return commands.NewErrorBudgetsCommand()
	case "slo-history":
		return commands.NewSLOHistoryCommand()
	case "events":
		return commands.NewEventsCommand()
	case "tags":
		return commands.NewTagsCommand()
	case "integrations":
		return commands.NewIntegrationsCommand()
	case "spans":
		return commands.NewSpansCommand()
	case "service-map":
		return commands.NewServiceMapCommand()
	case "usage-insights":
		return commands.NewUsageInsightsCommand()
	case "anomalies":
		return commands.NewAnomaliesCommand()
	case "correlation":
		return commands.NewCorrelationCommand()
	case "impact-analysis":
		return commands.NewImpactAnalysisCommand()
	case "auto-remediate":
		return commands.NewAutoRemediateCommand()
	case "change-management":
		return commands.NewChangeManagementCommand()
	case "capacity-scale":
		return commands.NewCapacityScaleCommand()
	case "ml-insights":
		return commands.NewMLInsightsCommand()
	case "predictions":
		return commands.NewPredictionsCommand()
	case "recommendations":
		return commands.NewRecommendationsCommand()
	default:
		return nil
	}
}

func printVersion() {
	fmt.Printf("dd version %s\n", version)
	if commit != "unknown" {
		fmt.Printf("  commit: %s\n", commit)
	}
	if buildDate != "unknown" {
		fmt.Printf("  built:  %s\n", buildDate)
	}
}

func printHelp() {
	fmt.Println("dd - Datadog CLI Skill")
	fmt.Println()
	fmt.Println("Usage: dd <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  context     Detect service context from project")
	fmt.Println("  version     Show version")
	fmt.Println("  help        Show this help")
	fmt.Println()
	fmt.Println("Data Management:")
	fmt.Println("  events        Query and post events for deployments, alerts, and changes")
	fmt.Println("  tags          Manage host tags for infrastructure organization and filtering")
	fmt.Println("  integrations  Manage cloud provider and service integrations (AWS, Azure, Slack, PagerDuty)")
	fmt.Println()
	fmt.Println("Query Operations:")
	fmt.Println("  apm           Query APM traces and performance")
	fmt.Println("  spans         Query APM spans for detailed trace analysis and latency investigation")
	fmt.Println("  service-map   Analyze service dependencies, topology, and call patterns")
	fmt.Println("  logs          Search and analyze logs")
	fmt.Println("  metrics       Query metrics for time series analysis")
	fmt.Println("  llm         Query LLM observability for GenAI applications")
	fmt.Println("  database    Query Database Monitoring for performance analysis")
	fmt.Println("  security    Query Security Monitoring Signals")
	fmt.Println("  slos        Query SLOs and check error budgets")
	fmt.Println("  watchdog    Query Watchdog for automated anomaly detection")
	fmt.Println("  catalog     Query Service Catalog for service metadata")
	fmt.Println("  rum         Query Real User Monitoring for frontend performance")
	fmt.Println("  network     Query Network Performance Monitoring for network analysis")
	fmt.Println("  cicd        Query CI/CD (CI Visibility) for pipeline and test analysis")
	fmt.Println()
	fmt.Println("Software Delivery:")
	fmt.Println("  dora        Query DORA Metrics for DevOps performance measurement")
	fmt.Println()
	fmt.Println("SRE & Reliability:")
	fmt.Println("  slo-corrections  Manage SLO corrections for accurate reporting")
	fmt.Println("  error-budgets    Query SLO error budgets for deployment decisions")
	fmt.Println("  slo-history      Query historical SLO data for trend analysis")
	fmt.Println()
	fmt.Println("Collaboration:")
	fmt.Println("  cases         Manage Case Management for issue tracking and resolution")
	fmt.Println("  status-pages  Manage Status Pages for customer communication")
	fmt.Println("  on-call       Manage On-Call scheduling and rotations")
	fmt.Println()
	fmt.Println("Infrastructure:")
	fmt.Println("  containers  Query container monitoring for Docker and Kubernetes")
	fmt.Println("  kubernetes  Query Kubernetes pod and cluster monitoring")
	fmt.Println("  serverless  Query serverless functions (Lambda, Azure Functions, Cloud Functions)")
	fmt.Println()
	fmt.Println("Management Operations:")
	fmt.Println("  incidents   Manage Datadog incidents")
	fmt.Println("  monitors    Manage Datadog monitors")
	fmt.Println("  downtimes   Manage monitor downtimes and scheduled maintenance windows")
	fmt.Println("  notebooks   Manage Datadog notebooks for documentation and investigation")
	fmt.Println("  dashboards  Manage Datadog dashboards")
	fmt.Println("  workflows   Manage Datadog workflow automation")
	fmt.Println("  synthetics  Manage Datadog synthetic tests")
	fmt.Println()
	fmt.Println("Platform & Administration:")
	fmt.Println("  teams            Manage Datadog teams for organization and access control")
	fmt.Println("  users            Manage Datadog users for access control and administration")
	fmt.Println("  roles            Manage Datadog roles and permissions for fine-grained access control")
	fmt.Println("  service-accounts Manage service accounts for automation and API access")
	fmt.Println("  api-keys         Manage API keys for primary authentication")
	fmt.Println("  application-keys Manage application keys for API authentication")
	fmt.Println()
	fmt.Println("Compliance & Security:")
	fmt.Println("  audit-logs       Query audit logs for compliance and security tracking")
	fmt.Println()
	fmt.Println("Smart Operations:")
	fmt.Println("  health      Check service health across multiple signals")
	fmt.Println("  deploy      Check if it's safe to deploy")
	fmt.Println()
	fmt.Println("Advanced Analytics:")
	fmt.Println("  anomalies        Detect anomalies across metrics, logs, traces, and infrastructure")
	fmt.Println("  correlation      Correlate events across multiple signals for root cause analysis")
	fmt.Println("  impact-analysis  Assess blast radius, change impact, and dependency effects")
	fmt.Println()
	fmt.Println("Automation & Remediation:")
	fmt.Println("  auto-remediate     Trigger automated remediation workflows based on detected conditions")
	fmt.Println("  change-management  Track, correlate, and manage changes with impact analysis")
	fmt.Println("  capacity-scale     Provide capacity planning and scaling recommendations")
	fmt.Println()
	fmt.Println("Machine Learning & Predictions:")
	fmt.Println("  ml-insights      ML-powered anomaly detection, pattern recognition, and forecasting")
	fmt.Println("  predictions      Predict incidents, capacity exhaustion, cost overruns, and SLO violations")
	fmt.Println("  recommendations  AI-driven optimization recommendations, auto-tuning, and best practices")
	fmt.Println()
	fmt.Println("FinOps:")
	fmt.Println("  cost            Analyze Datadog usage and costs for FinOps optimization")
	fmt.Println("  usage-insights  Deep usage analysis, cost optimization, and forecasting")
	fmt.Println()
	fmt.Println("Run 'dd <command> --help' for command-specific help")
}
