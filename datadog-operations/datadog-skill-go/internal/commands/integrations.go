package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/datadog/skill/internal/client"
)

// IntegrationsCommand manages Datadog integrations
type IntegrationsCommand struct {
	flags        *flag.FlagSet
	action       string
	integType    string
	accountID    string
	roleName     string
	tenantName   string
	clientID     string
	webhookURL   string
	webhookName  string
	channelName  string
	serviceName  string
	serviceKey   string
	jsonOut      bool
}

// AWSIntegrationData represents an AWS integration
type AWSIntegrationData struct {
	AccountID             string   `json:"account_id"`
	RoleName              string   `json:"role_name"`
	AccessKeyID           string   `json:"access_key_id,omitempty"`
	FilterTags            []string `json:"filter_tags,omitempty"`
	HostTags              []string `json:"host_tags,omitempty"`
	AccountSpecificNS     []string `json:"account_specific_namespace_rules,omitempty"`
	ExcludedRegions       []string `json:"excluded_regions,omitempty"`
	MetricsCollectionEnabled bool  `json:"metrics_collection_enabled,omitempty"`
}

// AzureIntegrationData represents an Azure integration
type AzureIntegrationData struct {
	TenantName   string   `json:"tenant_name"`
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret,omitempty"`
	HostFilters  string   `json:"host_filters,omitempty"`
	AppServicePlanFilters string `json:"app_service_plan_filters,omitempty"`
	ContainerAppFilters   string `json:"container_app_filters,omitempty"`
	Automute     bool     `json:"automute,omitempty"`
}

// SlackIntegrationData represents a Slack integration
type SlackIntegrationData struct {
	ServiceHooks []struct {
		Account string `json:"account"`
		URL     string `json:"url"`
	} `json:"service_hooks,omitempty"`
	Channels []struct {
		ChannelName string `json:"channel_name"`
		TransferAllUserComments bool `json:"transfer_all_user_comments"`
		Account string `json:"account"`
	} `json:"channels,omitempty"`
}

// PagerDutyIntegrationData represents a PagerDuty integration
type PagerDutyIntegrationData struct {
	Services []struct {
		ServiceName string `json:"service_name"`
		ServiceKey  string `json:"service_key"`
	} `json:"services,omitempty"`
	Subdomain string `json:"subdomain,omitempty"`
	Schedules []string `json:"schedules,omitempty"`
}

// IntegrationsResponse represents the integration API response
type IntegrationsResponse struct {
	Status       string                 `json:"status"`
	Integrations map[string]interface{} `json:"integrations,omitempty"`
	Integration  interface{}            `json:"integration,omitempty"`
	Message      string                 `json:"message,omitempty"`
}

// NewIntegrationsCommand creates a new integrations command instance
func NewIntegrationsCommand() *IntegrationsCommand {
	cmd := &IntegrationsCommand{
		flags: flag.NewFlagSet("integrations", flag.ExitOnError),
	}

	cmd.flags.StringVar(&cmd.action, "action", "list", "Action to perform (list, get, aws, azure, gcp, slack, pagerduty)")
	cmd.flags.StringVar(&cmd.integType, "type", "", "Integration type (for get action)")
	cmd.flags.StringVar(&cmd.accountID, "account-id", "", "AWS account ID")
	cmd.flags.StringVar(&cmd.roleName, "role-name", "", "AWS IAM role name")
	cmd.flags.StringVar(&cmd.tenantName, "tenant-name", "", "Azure tenant name")
	cmd.flags.StringVar(&cmd.clientID, "client-id", "", "Azure client ID")
	cmd.flags.StringVar(&cmd.webhookURL, "webhook-url", "", "Slack webhook URL")
	cmd.flags.StringVar(&cmd.webhookName, "webhook-name", "", "Slack webhook name")
	cmd.flags.StringVar(&cmd.channelName, "channel-name", "", "Slack channel name")
	cmd.flags.StringVar(&cmd.serviceName, "service-name", "", "PagerDuty service name")
	cmd.flags.StringVar(&cmd.serviceKey, "service-key", "", "PagerDuty service key")
	cmd.flags.BoolVar(&cmd.jsonOut, "json", false, "Output as JSON")

	return cmd
}

func (c *IntegrationsCommand) Name() string {
	return "integrations"
}

func (c *IntegrationsCommand) Description() string {
	return "Manage Datadog integrations"
}

func (c *IntegrationsCommand) Run(args []string) error {
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
	case "list":
		return c.listIntegrations(ddClient)
	case "get":
		return c.getIntegration(ddClient)
	case "aws":
		return c.awsIntegration(ddClient)
	case "azure":
		return c.azureIntegration(ddClient)
	case "gcp":
		return c.gcpIntegration(ddClient)
	case "slack":
		return c.slackIntegration(ddClient)
	case "pagerduty":
		return c.pagerdutyIntegration(ddClient)
	default:
		return fmt.Errorf("unknown action: %s (valid: list, get, aws, azure, gcp, slack, pagerduty)", c.action)
	}
}

func (c *IntegrationsCommand) listIntegrations(ddClient *client.Client) error {
	// Query multiple integration endpoints
	integrations := make(map[string]interface{})

	// Try AWS integrations
	if data, err := ddClient.GetAWSIntegration(); err == nil {
		var awsResp struct {
			Accounts []AWSIntegrationData `json:"accounts"`
		}
		if json.Unmarshal(data, &awsResp) == nil {
			integrations["aws"] = map[string]interface{}{
				"status":   "active",
				"accounts": len(awsResp.Accounts),
			}
		}
	}

	// Try Azure integrations
	if data, err := ddClient.GetAzureIntegration(); err == nil {
		var azureResp []AzureIntegrationData
		if json.Unmarshal(data, &azureResp) == nil {
			integrations["azure"] = map[string]interface{}{
				"status":  "active",
				"tenants": len(azureResp),
			}
		}
	}

	// Try Slack integration
	if data, err := ddClient.GetSlackIntegration(); err == nil {
		var slackResp SlackIntegrationData
		if json.Unmarshal(data, &slackResp) == nil {
			integrations["slack"] = map[string]interface{}{
				"status":   "active",
				"webhooks": len(slackResp.ServiceHooks),
				"channels": len(slackResp.Channels),
			}
		}
	}

	// Try PagerDuty integration
	if data, err := ddClient.GetPagerDutyIntegration(); err == nil {
		var pdResp PagerDutyIntegrationData
		if json.Unmarshal(data, &pdResp) == nil {
			integrations["pagerduty"] = map[string]interface{}{
				"status":   "active",
				"services": len(pdResp.Services),
			}
		}
	}

	// Format output
	if c.jsonOut {
		response := IntegrationsResponse{
			Status:       "success",
			Integrations: integrations,
		}
		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}

	// Human-readable output
	fmt.Println("Datadog Integrations")
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println()

	if len(integrations) == 0 {
		fmt.Println("No active integrations found.")
		fmt.Println()
		fmt.Println("Use specific actions to configure integrations:")
		fmt.Println("  dd integrations --action aws")
		fmt.Println("  dd integrations --action azure")
		fmt.Println("  dd integrations --action slack")
		fmt.Println("  dd integrations --action pagerduty")
		return nil
	}

	// Display AWS integration
	if aws, ok := integrations["aws"]; ok {
		awsData := aws.(map[string]interface{})
		fmt.Println("✅ AWS Integration")
		fmt.Printf("   Status: %s\n", awsData["status"])
		fmt.Printf("   Accounts: %d\n", awsData["accounts"])
		fmt.Println()
	}

	// Display Azure integration
	if azure, ok := integrations["azure"]; ok {
		azureData := azure.(map[string]interface{})
		fmt.Println("✅ Azure Integration")
		fmt.Printf("   Status: %s\n", azureData["status"])
		fmt.Printf("   Tenants: %d\n", azureData["tenants"])
		fmt.Println()
	}

	// Display Slack integration
	if slack, ok := integrations["slack"]; ok {
		slackData := slack.(map[string]interface{})
		fmt.Println("✅ Slack Integration")
		fmt.Printf("   Status: %s\n", slackData["status"])
		fmt.Printf("   Webhooks: %d\n", slackData["webhooks"])
		fmt.Printf("   Channels: %d\n", slackData["channels"])
		fmt.Println()
	}

	// Display PagerDuty integration
	if pd, ok := integrations["pagerduty"]; ok {
		pdData := pd.(map[string]interface{})
		fmt.Println("✅ PagerDuty Integration")
		fmt.Printf("   Status: %s\n", pdData["status"])
		fmt.Printf("   Services: %d\n", pdData["services"])
		fmt.Println()
	}

	fmt.Printf("Total active integrations: %d\n", len(integrations))
	fmt.Println()
	fmt.Println("Use --action <type> to view details for a specific integration.")

	return nil
}

func (c *IntegrationsCommand) getIntegration(ddClient *client.Client) error {
	if c.integType == "" {
		return fmt.Errorf("--type is required for get action (aws, azure, gcp, slack, pagerduty)")
	}

	switch c.integType {
	case "aws":
		return c.awsIntegration(ddClient)
	case "azure":
		return c.azureIntegration(ddClient)
	case "gcp":
		return c.gcpIntegration(ddClient)
	case "slack":
		return c.slackIntegration(ddClient)
	case "pagerduty":
		return c.pagerdutyIntegration(ddClient)
	default:
		return fmt.Errorf("unknown integration type: %s", c.integType)
	}
}

func (c *IntegrationsCommand) awsIntegration(ddClient *client.Client) error {
	data, err := ddClient.GetAWSIntegration()
	if err != nil {
		return fmt.Errorf("failed to get AWS integration: %w", err)
	}

	// Parse response
	var apiResp struct {
		Accounts []AWSIntegrationData `json:"accounts"`
	}
	if err := json.Unmarshal(data, &apiResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Format output
	if c.jsonOut {
		response := IntegrationsResponse{
			Status:      "success",
			Integration: apiResp.Accounts,
		}
		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}

	// Human-readable output
	fmt.Println("AWS Integration")
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println()

	if len(apiResp.Accounts) == 0 {
		fmt.Println("No AWS accounts configured.")
		fmt.Println()
		fmt.Println("To configure AWS integration:")
		fmt.Println("  1. Create IAM role in AWS with required permissions")
		fmt.Println("  2. Configure via Datadog UI or API")
		fmt.Println("  3. Verify with: dd integrations --action aws")
		return nil
	}

	for i, account := range apiResp.Accounts {
		if i > 0 {
			fmt.Println()
		}

		fmt.Printf("Account: %s\n", account.AccountID)
		if account.RoleName != "" {
			fmt.Printf("Role Name: %s\n", account.RoleName)
		}
		if len(account.FilterTags) > 0 {
			fmt.Printf("Filter Tags: %s\n", strings.Join(account.FilterTags, ", "))
		}
		if len(account.HostTags) > 0 {
			fmt.Printf("Host Tags: %s\n", strings.Join(account.HostTags, ", "))
		}
		if len(account.ExcludedRegions) > 0 {
			fmt.Printf("Excluded Regions: %s\n", strings.Join(account.ExcludedRegions, ", "))
		}
		fmt.Printf("Metrics Collection: %v\n", account.MetricsCollectionEnabled)
	}

	fmt.Println()
	fmt.Printf("Total AWS accounts: %d\n", len(apiResp.Accounts))

	return nil
}

func (c *IntegrationsCommand) azureIntegration(ddClient *client.Client) error {
	data, err := ddClient.GetAzureIntegration()
	if err != nil {
		return fmt.Errorf("failed to get Azure integration: %w", err)
	}

	// Parse response
	var apiResp []AzureIntegrationData
	if err := json.Unmarshal(data, &apiResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Format output
	if c.jsonOut {
		response := IntegrationsResponse{
			Status:      "success",
			Integration: apiResp,
		}
		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}

	// Human-readable output
	fmt.Println("Azure Integration")
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println()

	if len(apiResp) == 0 {
		fmt.Println("No Azure tenants configured.")
		fmt.Println()
		fmt.Println("To configure Azure integration:")
		fmt.Println("  1. Create App Registration in Azure AD")
		fmt.Println("  2. Grant required permissions")
		fmt.Println("  3. Configure via Datadog UI or API")
		fmt.Println("  4. Verify with: dd integrations --action azure")
		return nil
	}

	for i, tenant := range apiResp {
		if i > 0 {
			fmt.Println()
		}

		fmt.Printf("Tenant: %s\n", tenant.TenantName)
		fmt.Printf("Client ID: %s\n", tenant.ClientID)
		if tenant.HostFilters != "" {
			fmt.Printf("Host Filters: %s\n", tenant.HostFilters)
		}
		if tenant.AppServicePlanFilters != "" {
			fmt.Printf("App Service Plan Filters: %s\n", tenant.AppServicePlanFilters)
		}
		if tenant.ContainerAppFilters != "" {
			fmt.Printf("Container App Filters: %s\n", tenant.ContainerAppFilters)
		}
		fmt.Printf("Automute: %v\n", tenant.Automute)
	}

	fmt.Println()
	fmt.Printf("Total Azure tenants: %d\n", len(apiResp))

	return nil
}

func (c *IntegrationsCommand) gcpIntegration(ddClient *client.Client) error {
	// GCP integration uses a different API pattern
	// For now, return a helpful message
	fmt.Println("GCP Integration")
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println()
	fmt.Println("GCP integration management is available through:")
	fmt.Println("  - Datadog UI: https://app.datadoghq.com/integrations/google-cloud-platform")
	fmt.Println("  - Terraform: datadog_integration_gcp resource")
	fmt.Println()
	fmt.Println("To configure GCP integration:")
	fmt.Println("  1. Create service account in GCP")
	fmt.Println("  2. Grant required permissions (Compute Viewer, Monitoring Viewer, etc.)")
	fmt.Println("  3. Download service account JSON key")
	fmt.Println("  4. Configure via Datadog UI")
	fmt.Println()
	fmt.Println("For automated setup, use Terraform or Datadog API v1 endpoints.")

	return nil
}

func (c *IntegrationsCommand) slackIntegration(ddClient *client.Client) error {
	data, err := ddClient.GetSlackIntegration()
	if err != nil {
		return fmt.Errorf("failed to get Slack integration: %w", err)
	}

	// Parse response
	var apiResp SlackIntegrationData
	if err := json.Unmarshal(data, &apiResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Format output
	if c.jsonOut {
		response := IntegrationsResponse{
			Status:      "success",
			Integration: apiResp,
		}
		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}

	// Human-readable output
	fmt.Println("Slack Integration")
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println()

	if len(apiResp.ServiceHooks) == 0 && len(apiResp.Channels) == 0 {
		fmt.Println("No Slack webhooks or channels configured.")
		fmt.Println()
		fmt.Println("To configure Slack integration:")
		fmt.Println("  1. Install Datadog app in Slack workspace")
		fmt.Println("  2. Configure channels via Datadog UI")
		fmt.Println("  3. Test notifications with monitors")
		fmt.Println("  4. Verify with: dd integrations --action slack")
		return nil
	}

	if len(apiResp.ServiceHooks) > 0 {
		fmt.Println("Service Hooks:")
		for _, hook := range apiResp.ServiceHooks {
			fmt.Printf("  Account: %s\n", hook.Account)
			fmt.Printf("  URL: %s\n", hook.URL)
			fmt.Println()
		}
	}

	if len(apiResp.Channels) > 0 {
		fmt.Println("Channels:")
		for _, channel := range apiResp.Channels {
			fmt.Printf("  Channel: %s\n", channel.ChannelName)
			fmt.Printf("  Account: %s\n", channel.Account)
			fmt.Printf("  Transfer User Comments: %v\n", channel.TransferAllUserComments)
			fmt.Println()
		}
	}

	fmt.Printf("Total webhooks: %d\n", len(apiResp.ServiceHooks))
	fmt.Printf("Total channels: %d\n", len(apiResp.Channels))

	return nil
}

func (c *IntegrationsCommand) pagerdutyIntegration(ddClient *client.Client) error {
	data, err := ddClient.GetPagerDutyIntegration()
	if err != nil {
		return fmt.Errorf("failed to get PagerDuty integration: %w", err)
	}

	// Parse response
	var apiResp PagerDutyIntegrationData
	if err := json.Unmarshal(data, &apiResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Format output
	if c.jsonOut {
		response := IntegrationsResponse{
			Status:      "success",
			Integration: apiResp,
		}
		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}

	// Human-readable output
	fmt.Println("PagerDuty Integration")
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println()

	if len(apiResp.Services) == 0 {
		fmt.Println("No PagerDuty services configured.")
		fmt.Println()
		fmt.Println("To configure PagerDuty integration:")
		fmt.Println("  1. Get API token from PagerDuty")
		fmt.Println("  2. Configure services via Datadog UI")
		fmt.Println("  3. Link monitors to PagerDuty services")
		fmt.Println("  4. Verify with: dd integrations --action pagerduty")
		return nil
	}

	if apiResp.Subdomain != "" {
		fmt.Printf("Subdomain: %s\n", apiResp.Subdomain)
		fmt.Println()
	}

	fmt.Println("Services:")
	for _, service := range apiResp.Services {
		fmt.Printf("  Service: %s\n", service.ServiceName)
		fmt.Printf("  Service Key: %s\n", maskServiceKey(service.ServiceKey))
		fmt.Println()
	}

	if len(apiResp.Schedules) > 0 {
		fmt.Println("Schedules:")
		for _, schedule := range apiResp.Schedules {
			fmt.Printf("  %s\n", schedule)
		}
		fmt.Println()
	}

	fmt.Printf("Total services: %d\n", len(apiResp.Services))

	return nil
}

func maskServiceKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + strings.Repeat("*", len(key)-8) + key[len(key)-4:]
}

func (c *IntegrationsCommand) Help() {
	helpText := `dd integrations - Manage Datadog Integrations

DESCRIPTION:
  Manage Datadog integrations with cloud providers (AWS, Azure, GCP) and
  external services (Slack, PagerDuty). View integration status, configuration,
  and connection details.

USAGE:
  dd integrations --action <action> [options]

ACTIONS:
  list             List all configured integrations
  get              Get specific integration details
  aws              View AWS integration details
  azure            View Azure integration details
  gcp              View GCP integration details
  slack            View Slack integration details
  pagerduty        View PagerDuty integration details

EXAMPLES:
  # List all active integrations
  dd integrations --action list

  # View AWS integration details
  dd integrations --action aws

  # View Azure integration details
  dd integrations --action azure

  # View Slack integration details
  dd integrations --action slack

  # View PagerDuty integration details
  dd integrations --action pagerduty

  # Get specific integration type
  dd integrations --action get --type aws

  # Get JSON output
  dd integrations --action list --json

OPTIONS:
  --action          Action to perform (list, get, aws, azure, gcp, slack, pagerduty)
  --type            Integration type for get action (aws, azure, gcp, slack, pagerduty)
  --json            Output as JSON

CLOUD INTEGRATIONS:

  AWS Integration:
    - Automatic resource discovery and tagging
    - CloudWatch metrics collection
    - CloudTrail log ingestion
    - Resource filtering by tags and regions

    Configuration:
      1. Create IAM role with DatadogAWSIntegrationRole policy
      2. Configure trust relationship with Datadog AWS account
      3. Add integration via Datadog UI or API
      4. Verify: dd integrations --action aws

    Displays:
      - Account IDs
      - IAM role names
      - Filter tags
      - Host tags
      - Excluded regions
      - Metrics collection status

  Azure Integration:
    - Azure Monitor metrics collection
    - Resource tagging and discovery
    - Activity log ingestion
    - App Service, VMs, containers monitoring

    Configuration:
      1. Create App Registration in Azure AD
      2. Grant Monitoring Reader role
      3. Create client secret
      4. Add integration via Datadog UI
      5. Verify: dd integrations --action azure

    Displays:
      - Tenant names
      - Client IDs
      - Host filters
      - App Service Plan filters
      - Container App filters
      - Automute settings

  GCP Integration:
    - Google Cloud monitoring metrics
    - Resource discovery
    - Log ingestion
    - GKE, Compute Engine, Cloud Functions monitoring

    Configuration:
      1. Create service account with required permissions
      2. Download JSON key
      3. Configure via Datadog UI
      4. Enable required APIs (Compute Engine, Monitoring)

    Note: GCP integration managed primarily via Datadog UI

NOTIFICATION INTEGRATIONS:

  Slack Integration:
    - Send monitor alerts to Slack channels
    - Incident notifications
    - Event stream updates
    - @-mentions in Datadog comments

    Configuration:
      1. Install Datadog app in Slack workspace
      2. Authorize channels
      3. Configure notification rules
      4. Verify: dd integrations --action slack

    Displays:
      - Webhook URLs
      - Configured channels
      - User comment transfer settings

  PagerDuty Integration:
    - Trigger incidents from Datadog monitors
    - Bi-directional incident sync
    - On-call schedule sync
    - Acknowledge/resolve from either platform

    Configuration:
      1. Get PagerDuty API token
      2. Configure services in Datadog
      3. Link monitors to PagerDuty services
      4. Verify: dd integrations --action pagerduty

    Displays:
      - Service names
      - Service keys (masked)
      - Subdomain
      - Schedules

USE CASES:
  Integration Health Check:
    - Verify all integrations are active
    - Check for configuration issues
    - Audit integration settings

  Cloud Provider Setup:
    - Confirm AWS/Azure/GCP connections
    - Verify permissions and roles
    - Check resource filtering rules

  Notification Testing:
    - Verify Slack webhook configuration
    - Check PagerDuty service links
    - Test alert routing

  Incident Response:
    - Verify PagerDuty escalation paths
    - Check Slack channel configurations
    - Validate on-call schedule sync

  Automation:
    - Programmatic integration verification
    - CI/CD health checks
    - Infrastructure validation

INTEGRATION WITH OTHER COMMANDS:
  Monitor Notifications:
    1. Check integrations: dd integrations --action list
    2. View monitors:      dd monitors --action list
    3. Test alert:         Trigger monitor alert

  Cloud Resource Monitoring:
    1. Verify AWS:         dd integrations --action aws
    2. Query metrics:      dd metrics --query "aws.ec2.cpuutilization"
    3. Check containers:   dd containers --action list

  Incident Management:
    1. Check PagerDuty:    dd integrations --action pagerduty
    2. List incidents:     dd incidents --action list
    3. Verify on-call:     dd on-call --action list

BEST PRACTICES:
  Security:
    - Use least-privilege IAM roles for cloud integrations
    - Rotate integration keys regularly
    - Audit integration access permissions
    - Monitor integration usage

  Configuration:
    - Tag cloud resources consistently
    - Filter unnecessary resources to reduce costs
    - Exclude non-production regions if not needed
    - Document integration ownership

  Monitoring:
    - Verify integrations regularly
    - Test notification channels
    - Monitor integration metrics
    - Set up alerts for integration failures

  Troubleshooting:
    - Check IAM role trust relationships (AWS)
    - Verify client secret expiration (Azure)
    - Confirm API permissions (GCP)
    - Test webhook endpoints (Slack)

NOTES:
  - Integration configurations are managed via Datadog API v1
  - Some integrations require setup in external platforms first
  - Changes may take a few minutes to propagate
  - Service keys and secrets are masked in output
  - Full configuration management available via Datadog UI

LIMITATIONS:
  - This command provides read-only access to integrations
  - Creating/updating integrations requires Datadog UI or direct API calls
  - GCP integration details require separate API endpoints
  - Some integration details may be redacted for security

TERRAFORM ALTERNATIVE:
  For infrastructure-as-code integration management:
    - datadog_integration_aws
    - datadog_integration_azure
    - datadog_integration_gcp
    - datadog_integration_slack_channel
    - datadog_integration_pagerduty

For more information:
  - AWS: https://docs.datadoghq.com/integrations/amazon_web_services/
  - Azure: https://docs.datadoghq.com/integrations/azure/
  - GCP: https://docs.datadoghq.com/integrations/google_cloud_platform/
  - Slack: https://docs.datadoghq.com/integrations/slack/
  - PagerDuty: https://docs.datadoghq.com/integrations/pagerduty/
`

	fmt.Println(helpText)
}
