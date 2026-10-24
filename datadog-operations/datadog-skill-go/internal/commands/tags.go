package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"sort"
	"strings"

	"github.com/datadog/skill/internal/client"
)

// TagsCommand manages Datadog host tags
type TagsCommand struct {
	flags   *flag.FlagSet
	action  string
	host    string
	tags    string
	source  string
	jsonOut bool
}

// HostTagsData represents tags for a host
type HostTagsData struct {
	Host string   `json:"host"`
	Tags []string `json:"tags"`
}

// TagsListResponse represents the list tags API response
type TagsListResponse struct {
	Status string                  `json:"status"`
	Tags   map[string]HostTagsData `json:"tags,omitempty"`
}

// TagsHostResponse represents a single host tags API response
type TagsHostResponse struct {
	Status string       `json:"status"`
	Host   string       `json:"host"`
	Tags   []string     `json:"tags"`
	Source map[string][]string `json:"source,omitempty"`
}

// NewTagsCommand creates a new tags command instance
func NewTagsCommand() *TagsCommand {
	cmd := &TagsCommand{
		flags: flag.NewFlagSet("tags", flag.ExitOnError),
	}

	cmd.flags.StringVar(&cmd.action, "action", "list", "Action to perform (list, host, add, remove, update, search)")
	cmd.flags.StringVar(&cmd.host, "host", "", "Host name (required for host, add, remove, update actions)")
	cmd.flags.StringVar(&cmd.tags, "tags", "", "Comma-separated tags (required for add, update actions)")
	cmd.flags.StringVar(&cmd.source, "source", "", "Tag source to filter by")
	cmd.flags.BoolVar(&cmd.jsonOut, "json", false, "Output as JSON")

	return cmd
}

func (c *TagsCommand) Name() string {
	return "tags"
}

func (c *TagsCommand) Description() string {
	return "Manage Datadog host tags"
}

func (c *TagsCommand) Run(args []string) error {
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
		return c.listTags(ddClient)
	case "host":
		return c.getHostTags(ddClient)
	case "add":
		return c.addHostTags(ddClient)
	case "remove":
		return c.removeHostTags(ddClient)
	case "update":
		return c.updateHostTags(ddClient)
	case "search":
		return c.searchTags(ddClient)
	default:
		return fmt.Errorf("unknown action: %s (valid: list, host, add, remove, update, search)", c.action)
	}
}

func (c *TagsCommand) listTags(ddClient *client.Client) error {
	params := make(map[string]string)
	if c.source != "" {
		params["source"] = c.source
	}

	data, err := ddClient.ListAllTags(params)
	if err != nil {
		return fmt.Errorf("failed to list tags: %w", err)
	}

	// Parse response
	var apiResp struct {
		Tags map[string][]string `json:"tags"`
	}
	if err := json.Unmarshal(data, &apiResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Format output
	if c.jsonOut {
		// Convert to structured format
		tagsMap := make(map[string]HostTagsData)
		for host, tags := range apiResp.Tags {
			tagsMap[host] = HostTagsData{
				Host: host,
				Tags: tags,
			}
		}
		response := TagsListResponse{
			Status: "success",
			Tags:   tagsMap,
		}
		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}

	// Human-readable output
	fmt.Printf("Tags Across Infrastructure (%d hosts)\n", len(apiResp.Tags))
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println()

	if len(apiResp.Tags) == 0 {
		fmt.Println("No hosts with tags found.")
		return nil
	}

	// Sort hosts for consistent output
	hosts := make([]string, 0, len(apiResp.Tags))
	for host := range apiResp.Tags {
		hosts = append(hosts, host)
	}
	sort.Strings(hosts)

	// Display first 10 hosts (or all if less than 10)
	displayCount := 10
	if len(hosts) < displayCount {
		displayCount = len(hosts)
	}

	for i := 0; i < displayCount; i++ {
		host := hosts[i]
		tags := apiResp.Tags[host]

		if i > 0 {
			fmt.Println()
		}

		fmt.Printf("Host: %s\n", host)
		if len(tags) > 0 {
			fmt.Printf("Tags (%d): %s\n", len(tags), strings.Join(tags, ", "))
		} else {
			fmt.Println("Tags: (none)")
		}
	}

	if len(hosts) > displayCount {
		fmt.Printf("\n... and %d more hosts (use --host to view specific host)\n", len(hosts)-displayCount)
	}

	// Collect unique tag keys for summary
	tagKeys := make(map[string]bool)
	for _, tags := range apiResp.Tags {
		for _, tag := range tags {
			parts := strings.SplitN(tag, ":", 2)
			tagKeys[parts[0]] = true
		}
	}

	fmt.Println()
	fmt.Println("Summary:")
	fmt.Printf("  Total hosts: %d\n", len(hosts))
	fmt.Printf("  Unique tag keys: %d\n", len(tagKeys))

	return nil
}

func (c *TagsCommand) getHostTags(ddClient *client.Client) error {
	if c.host == "" {
		return fmt.Errorf("--host is required for host action")
	}

	params := make(map[string]string)
	if c.source != "" {
		params["source"] = c.source
	}

	data, err := ddClient.GetHostTags(c.host, params)
	if err != nil {
		return fmt.Errorf("failed to get host tags: %w", err)
	}

	// Parse response
	var apiResp struct {
		Tags map[string][]string `json:"tags"`
	}
	if err := json.Unmarshal(data, &apiResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Flatten all tags
	var allTags []string
	tagSources := make(map[string][]string)
	for source, tags := range apiResp.Tags {
		allTags = append(allTags, tags...)
		tagSources[source] = tags
	}

	// Remove duplicates and sort
	uniqueTags := make(map[string]bool)
	for _, tag := range allTags {
		uniqueTags[tag] = true
	}
	allTags = make([]string, 0, len(uniqueTags))
	for tag := range uniqueTags {
		allTags = append(allTags, tag)
	}
	sort.Strings(allTags)

	// Format output
	if c.jsonOut {
		response := TagsHostResponse{
			Status: "success",
			Host:   c.host,
			Tags:   allTags,
			Source: tagSources,
		}
		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}

	// Human-readable output
	fmt.Printf("Host Tags: %s\n", c.host)
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println()

	if len(allTags) == 0 {
		fmt.Println("No tags found for this host.")
		return nil
	}

	fmt.Println("All Tags:")
	for _, tag := range allTags {
		fmt.Printf("  %s\n", tag)
	}

	fmt.Println()
	fmt.Printf("Total tags: %d\n", len(allTags))

	// Show tags by source if multiple sources
	if len(tagSources) > 1 {
		fmt.Println()
		fmt.Println("Tags by Source:")
		sources := make([]string, 0, len(tagSources))
		for source := range tagSources {
			sources = append(sources, source)
		}
		sort.Strings(sources)
		for _, source := range sources {
			tags := tagSources[source]
			fmt.Printf("  %s (%d): %s\n", source, len(tags), strings.Join(tags, ", "))
		}
	}

	return nil
}

func (c *TagsCommand) addHostTags(ddClient *client.Client) error {
	if c.host == "" {
		return fmt.Errorf("--host is required for add action")
	}
	if c.tags == "" {
		return fmt.Errorf("--tags is required for add action")
	}

	// Parse tags
	tagsToAdd := strings.Split(c.tags, ",")
	for i, tag := range tagsToAdd {
		tagsToAdd[i] = strings.TrimSpace(tag)
	}

	data, err := ddClient.AddHostTags(c.host, tagsToAdd, c.source)
	if err != nil {
		return fmt.Errorf("failed to add host tags: %w", err)
	}

	// Parse response
	var apiResp struct {
		Host string   `json:"host"`
		Tags []string `json:"tags"`
	}
	if err := json.Unmarshal(data, &apiResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Format output
	if c.jsonOut {
		response := TagsHostResponse{
			Status: "success",
			Host:   apiResp.Host,
			Tags:   apiResp.Tags,
		}
		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}

	// Human-readable output
	fmt.Println("Tags Added Successfully")
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println()

	fmt.Printf("✅ Host: %s\n", c.host)
	fmt.Printf("Added tags: %s\n", strings.Join(tagsToAdd, ", "))
	fmt.Println()
	fmt.Printf("Current tags (%d):\n", len(apiResp.Tags))
	for _, tag := range apiResp.Tags {
		fmt.Printf("  %s\n", tag)
	}

	return nil
}

func (c *TagsCommand) removeHostTags(ddClient *client.Client) error {
	if c.host == "" {
		return fmt.Errorf("--host is required for remove action")
	}

	params := make(map[string]string)
	if c.source != "" {
		params["source"] = c.source
	}

	err := ddClient.RemoveHostTags(c.host, params)
	if err != nil {
		return fmt.Errorf("failed to remove host tags: %w", err)
	}

	// Format output
	if c.jsonOut {
		response := map[string]string{
			"status": "success",
			"host":   c.host,
			"action": "removed",
		}
		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}

	// Human-readable output
	fmt.Println("Tags Removed Successfully")
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println()

	fmt.Printf("✅ Removed all tags from host: %s\n", c.host)
	if c.source != "" {
		fmt.Printf("Source: %s\n", c.source)
	}

	return nil
}

func (c *TagsCommand) updateHostTags(ddClient *client.Client) error {
	if c.host == "" {
		return fmt.Errorf("--host is required for update action")
	}
	if c.tags == "" {
		return fmt.Errorf("--tags is required for update action")
	}

	// Parse tags
	newTags := strings.Split(c.tags, ",")
	for i, tag := range newTags {
		newTags[i] = strings.TrimSpace(tag)
	}

	data, err := ddClient.UpdateHostTags(c.host, newTags, c.source)
	if err != nil {
		return fmt.Errorf("failed to update host tags: %w", err)
	}

	// Parse response
	var apiResp struct {
		Host string   `json:"host"`
		Tags []string `json:"tags"`
	}
	if err := json.Unmarshal(data, &apiResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Format output
	if c.jsonOut {
		response := TagsHostResponse{
			Status: "success",
			Host:   apiResp.Host,
			Tags:   apiResp.Tags,
		}
		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}

	// Human-readable output
	fmt.Println("Tags Updated Successfully")
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println()

	fmt.Printf("✅ Host: %s\n", c.host)
	fmt.Printf("New tags (%d):\n", len(apiResp.Tags))
	for _, tag := range apiResp.Tags {
		fmt.Printf("  %s\n", tag)
	}

	return nil
}

func (c *TagsCommand) searchTags(ddClient *client.Client) error {
	if c.tags == "" {
		return fmt.Errorf("--tags is required for search action")
	}

	// Get all tags
	params := make(map[string]string)
	if c.source != "" {
		params["source"] = c.source
	}

	data, err := ddClient.ListAllTags(params)
	if err != nil {
		return fmt.Errorf("failed to list tags: %w", err)
	}

	// Parse response
	var apiResp struct {
		Tags map[string][]string `json:"tags"`
	}
	if err := json.Unmarshal(data, &apiResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Search for hosts with matching tags
	searchTags := strings.Split(c.tags, ",")
	for i, tag := range searchTags {
		searchTags[i] = strings.TrimSpace(tag)
	}

	matchingHosts := make(map[string][]string)
	for host, tags := range apiResp.Tags {
		matchedTags := make([]string, 0)
		for _, searchTag := range searchTags {
			for _, hostTag := range tags {
				if strings.Contains(hostTag, searchTag) {
					matchedTags = append(matchedTags, hostTag)
					break
				}
			}
		}
		if len(matchedTags) > 0 {
			matchingHosts[host] = matchedTags
		}
	}

	// Format output
	if c.jsonOut {
		tagsMap := make(map[string]HostTagsData)
		for host, tags := range matchingHosts {
			tagsMap[host] = HostTagsData{
				Host: host,
				Tags: tags,
			}
		}
		response := TagsListResponse{
			Status: "success",
			Tags:   tagsMap,
		}
		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}

	// Human-readable output
	fmt.Printf("Search Results (%d hosts found)\n", len(matchingHosts))
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println()
	fmt.Printf("Searching for tags: %s\n", strings.Join(searchTags, ", "))
	fmt.Println()

	if len(matchingHosts) == 0 {
		fmt.Println("No hosts found matching the search criteria.")
		return nil
	}

	// Sort hosts
	hosts := make([]string, 0, len(matchingHosts))
	for host := range matchingHosts {
		hosts = append(hosts, host)
	}
	sort.Strings(hosts)

	// Display results
	for i, host := range hosts {
		if i > 0 {
			fmt.Println()
		}
		matchedTags := matchingHosts[host]
		fmt.Printf("Host: %s\n", host)
		fmt.Printf("Matched tags: %s\n", strings.Join(matchedTags, ", "))
	}

	return nil
}

func (c *TagsCommand) Help() {
	helpText := `dd tags - Manage Datadog Host Tags

DESCRIPTION:
  Manage tags for hosts, containers, and services to enable proper organization
  and filtering across Datadog. Tags are key-value pairs that help you group
  and filter your infrastructure, applications, and services.

USAGE:
  dd tags --action <action> [options]

ACTIONS:
  list             List all tags across infrastructure
  host             Get tags for specific host
  add              Add tags to host
  remove           Remove all tags from host
  update           Replace all tags for host
  search           Search for hosts by tags

EXAMPLES:
  # List all tags (shows first 10 hosts)
  dd tags --action list

  # List tags from specific source
  dd tags --action list --source "datadog"

  # Get tags for specific host
  dd tags --action host --host web-prod-01

  # Add tags to host
  dd tags --action add \
    --host web-prod-01 \
    --tags "env:prod,team:platform,criticality:high"

  # Update tags (replace all existing)
  dd tags --action update \
    --host web-prod-01 \
    --tags "env:prod,team:platform,service:web,version:v2.1.0"

  # Remove all tags from host
  dd tags --action remove --host web-prod-01

  # Remove tags from specific source
  dd tags --action remove --host web-prod-01 --source "user"

  # Search for hosts with specific tags
  dd tags --action search --tags "env:prod"

  # Search for multiple tags
  dd tags --action search --tags "env:prod,team:platform"

  # Get JSON output
  dd tags --action list --json

OPTIONS:
  --action          Action to perform (list, host, add, remove, update, search)
  --host            Host name (required for host, add, remove, update actions)
  --tags            Comma-separated tags (required for add, update, search actions)
  --source          Tag source to filter by (optional)
  --json            Output as JSON

TAG FORMAT:
  Tags can be simple labels or key:value pairs:
    Simple:    production, web, critical
    Key-Value: env:prod, team:platform, service:api, version:v1.2.3

  Comma-separated without spaces:
    ✅ "env:prod,team:platform,service:api"
    ❌ "env:prod, team:platform, service:api" (spaces not recommended)

TAG SOURCES:
  Tags can come from different sources:
  - datadog:   Datadog Agent tags
  - users:     User-defined tags via API/UI
  - integration: Cloud integration tags (AWS, Azure, GCP)

  Use --source to filter or modify tags from specific source.

USE CASES:
  Infrastructure Organization:
    - Tag hosts by environment (env:prod, env:staging)
    - Tag by team (team:platform, team:data)
    - Tag by criticality (criticality:high, criticality:low)

  Service Grouping:
    - Tag by service name (service:api, service:web)
    - Tag by version (version:v1.2.3)
    - Tag by region (region:us-east-1)

  Cost Allocation:
    - Tag by cost center (cost-center:eng)
    - Tag by project (project:migration)
    - Tag by customer (customer:acme-corp)

  Filtering & Queries:
    - Use tags to filter metrics: system.cpu.user{env:prod,team:platform}
    - Use tags to filter logs: service:api env:prod
    - Use tags to filter APM traces

  Automation:
    - Tag new hosts automatically during provisioning
    - Update tags during deployments
    - Search for hosts needing updates

INTEGRATION WITH OTHER COMMANDS:
  Metrics Queries:
    1. Tag hosts:    dd tags --action add --host web-01 --tags "env:prod"
    2. Query metrics: dd metrics --query "avg:system.cpu{env:prod}"

  Container Monitoring:
    1. Tag containers: dd tags --action add --host container-01 --tags "app:api"
    2. Monitor:       dd containers --action list

  Service Discovery:
    1. Search:   dd tags --action search --tags "service:api"
    2. Monitor:  dd health --service api
    3. Check:    dd apm --service api

  Deployment Workflow:
    1. Add version tag: dd tags --action add --host web-01 --tags "version:v1.2.3"
    2. Post event:      dd events --action post --title "Deploy v1.2.3"
    3. Verify:          dd tags --action host --host web-01

BEST PRACTICES:
  Consistent Naming:
    - Use lowercase for tags
    - Use hyphens for multi-word values: cost-center:eng-platform
    - Use consistent key names across infrastructure

  Reserved Tags:
    - env: environment (prod, staging, dev)
    - service: service name
    - version: application version
    - team: owning team
    - region: cloud region

  Tag Hierarchy:
    - Use hierarchical tags: service:api, service:api-v2
    - Avoid over-tagging (too many tags per host)
    - Keep tag cardinality reasonable

  Tag Management:
    - Use 'update' to replace all tags (atomic operation)
    - Use 'add' to append tags without removing existing
    - Use 'remove' carefully (removes ALL tags)
    - Document tag schema for your organization

NOTES:
  - Maximum 1000 tags per host
  - Tag keys and values are case-sensitive
  - Tags propagate to metrics, logs, traces, and events
  - Some tags are added automatically by Datadog Agent
  - Cloud integration tags (AWS, Azure, GCP) are read-only
  - Changes may take a few minutes to propagate

TAG SOURCES:
  Datadog Agent automatically adds:
    - host name
    - availability zone
    - instance type
    - cloud provider tags

  You can add custom tags via:
    - API (this command)
    - Datadog Agent configuration
    - Cloud provider tags (synced automatically)
    - Kubernetes labels (synced automatically)

SEARCH FUNCTIONALITY:
  The search action supports partial matching:
    --tags "env:prod"        Matches: env:prod, env:production
    --tags "team"            Matches: team:platform, team:data, team-name:ops
    --tags "env:prod,team"   Matches hosts with both patterns

For more information: https://docs.datadoghq.com/api/latest/tags/
`

	fmt.Println(helpText)
}
