package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/datadog/skill/internal/client"
)

// ContainersCommand queries container monitoring data
type ContainersCommand struct {
	flags    *flag.FlagSet
	tags     string
	groupBy  string
	sort     string
	limit    int
	image    string
	state    string
	jsonOut  bool
}

// NewContainersCommand creates a new containers command
func NewContainersCommand() Command {
	cmd := &ContainersCommand{
		flags: flag.NewFlagSet("containers", flag.ExitOnError),
	}

	cmd.flags.StringVar(&cmd.tags, "tags", "", "Comma-separated tags to filter containers")
	cmd.flags.StringVar(&cmd.groupBy, "group-by", "", "Comma-separated tags to group containers")
	cmd.flags.StringVar(&cmd.sort, "sort", "", "Attribute to sort containers by (e.g., name, started_at)")
	cmd.flags.IntVar(&cmd.limit, "limit", 100, "Maximum number of results (default: 100)")
	cmd.flags.StringVar(&cmd.image, "image", "", "Filter by image name")
	cmd.flags.StringVar(&cmd.state, "state", "", "Filter by state (running, exited, paused)")
	cmd.flags.BoolVar(&cmd.jsonOut, "json", false, "Output in JSON format")

	return cmd
}

// Name returns the command name
func (c *ContainersCommand) Name() string {
	return "containers"
}

// Description returns a short description
func (c *ContainersCommand) Description() string {
	return "Query container monitoring for Docker, Kubernetes, and orchestrated environments"
}

// Run executes the containers command
func (c *ContainersCommand) Run(args []string) error {
	if err := c.flags.Parse(args); err != nil {
		return err
	}

	ddClient, err := client.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Datadog client: %w", err)
	}

	// Build query parameters
	params := make(map[string]interface{})

	if c.tags != "" {
		params["filter[tags]"] = c.tags
	}

	if c.groupBy != "" {
		params["group_by"] = c.groupBy
	}

	if c.sort != "" {
		params["sort"] = c.sort
	}

	if c.limit > 0 {
		params["page[size]"] = c.limit
	}

	resp, err := ddClient.ListContainers(params)
	if err != nil {
		return fmt.Errorf("failed to query containers: %w", err)
	}

	if c.jsonOut {
		fmt.Println(string(resp))
		return nil
	}

	return c.printResults(resp)
}

// printResults formats and displays container results
func (c *ContainersCommand) printResults(data []byte) error {
	var result struct {
		Data []struct {
			Type       string `json:"type"`
			ID         string `json:"id"`
			Attributes struct {
				ContainerID string    `json:"container_id"`
				CreatedAt   time.Time `json:"created_at"`
				Host        string    `json:"host"`
				ImageDigest string    `json:"image_digest"`
				ImageName   string    `json:"image_name"`
				ImageTags   []string  `json:"image_tags"`
				Name        string    `json:"name"`
				StartedAt   time.Time `json:"started_at"`
				State       string    `json:"state"`
				Tags        []string  `json:"tags"`
			} `json:"attributes"`
		} `json:"data"`
		Meta struct {
			Pagination struct {
				TotalCount int    `json:"total_count"`
				NextCursor string `json:"next_cursor"`
			} `json:"pagination"`
		} `json:"meta"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("failed to parse results: %w", err)
	}

	// Filter by image if specified
	filteredContainers := result.Data
	if c.image != "" {
		var filtered []struct {
			Type       string `json:"type"`
			ID         string `json:"id"`
			Attributes struct {
				ContainerID string    `json:"container_id"`
				CreatedAt   time.Time `json:"created_at"`
				Host        string    `json:"host"`
				ImageDigest string    `json:"image_digest"`
				ImageName   string    `json:"image_name"`
				ImageTags   []string  `json:"image_tags"`
				Name        string    `json:"name"`
				StartedAt   time.Time `json:"started_at"`
				State       string    `json:"state"`
				Tags        []string  `json:"tags"`
			} `json:"attributes"`
		}
		for _, container := range result.Data {
			if strings.Contains(strings.ToLower(container.Attributes.ImageName), strings.ToLower(c.image)) {
				filtered = append(filtered, container)
			}
		}
		filteredContainers = filtered
	}

	// Filter by state if specified
	if c.state != "" {
		var filtered []struct {
			Type       string `json:"type"`
			ID         string `json:"id"`
			Attributes struct {
				ContainerID string    `json:"container_id"`
				CreatedAt   time.Time `json:"created_at"`
				Host        string    `json:"host"`
				ImageDigest string    `json:"image_digest"`
				ImageName   string    `json:"image_name"`
				ImageTags   []string  `json:"image_tags"`
				Name        string    `json:"name"`
				StartedAt   time.Time `json:"started_at"`
				State       string    `json:"state"`
				Tags        []string  `json:"tags"`
			} `json:"attributes"`
		}
		for _, container := range filteredContainers {
			if strings.EqualFold(container.Attributes.State, c.state) {
				filtered = append(filtered, container)
			}
		}
		filteredContainers = filtered
	}

	totalCount := result.Meta.Pagination.TotalCount
	displayedCount := len(filteredContainers)

	fmt.Printf("Containers (Showing: %d, Total: %d):\n", displayedCount, totalCount)
	fmt.Println(strings.Repeat("=", 80))

	if displayedCount == 0 {
		fmt.Println("No containers found matching the criteria.")
		return nil
	}

	// Group containers by state
	stateGroups := make(map[string]int)
	for _, container := range filteredContainers {
		stateGroups[container.Attributes.State]++
	}

	fmt.Println("\nState Summary:")
	for state, count := range stateGroups {
		fmt.Printf("  %s: %d\n", state, count)
	}

	fmt.Println("\nContainer Details:")
	fmt.Println(strings.Repeat("-", 80))

	for i, container := range filteredContainers {
		if i >= 50 {
			fmt.Printf("\n... and %d more containers (use --limit to see more)\n", displayedCount-50)
			break
		}

		// State indicator
		stateIcon := "●"
		if container.Attributes.State == "running" {
			stateIcon = "✓"
		} else if container.Attributes.State == "exited" {
			stateIcon = "✗"
		}

		fmt.Printf("\n%s %s\n", stateIcon, container.Attributes.Name)
		fmt.Printf("  Container ID: %s\n", container.Attributes.ContainerID)
		fmt.Printf("  Image: %s\n", container.Attributes.ImageName)

		if len(container.Attributes.ImageTags) > 0 {
			fmt.Printf("  Image Tags: %s\n", strings.Join(container.Attributes.ImageTags, ", "))
		}

		fmt.Printf("  State: %s\n", container.Attributes.State)
		fmt.Printf("  Host: %s\n", container.Attributes.Host)

		uptime := time.Since(container.Attributes.StartedAt)
		if uptime > 0 {
			fmt.Printf("  Uptime: %s\n", formatDuration(uptime))
		}

		// Show relevant tags (skip common ones)
		relevantTags := filterRelevantTags(container.Attributes.Tags)
		if len(relevantTags) > 0 && len(relevantTags) <= 5 {
			fmt.Printf("  Tags: %s\n", strings.Join(relevantTags, ", "))
		} else if len(relevantTags) > 5 {
			fmt.Printf("  Tags: %s ... (%d more)\n", strings.Join(relevantTags[:5], ", "), len(relevantTags)-5)
		}
	}

	if result.Meta.Pagination.NextCursor != "" {
		fmt.Println("\nMore results available. Use pagination to fetch next page.")
	}

	return nil
}

// filterRelevantTags removes common/boring tags to show only interesting ones
func filterRelevantTags(tags []string) []string {
	var relevant []string
	skipPrefixes := []string{
		"container_id:",
		"container_name:",
		"image_id:",
		"image_name:",
		"short_image:",
	}

	for _, tag := range tags {
		skip := false
		for _, prefix := range skipPrefixes {
			if strings.HasPrefix(tag, prefix) {
				skip = true
				break
			}
		}
		if !skip {
			relevant = append(relevant, tag)
		}
	}

	return relevant
}

// formatDuration formats a duration in human-readable format
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	return fmt.Sprintf("%dd %dh", days, hours)
}

// Help displays help information
func (c *ContainersCommand) Help() {
	help := `dd containers - Query Container Monitoring

DESCRIPTION:
  Query container monitoring data for Docker, Kubernetes pods, and other
  containerized workloads. Track container state, resource usage, and health
  across your infrastructure.

USAGE:
  dd containers [options]

OPTIONS:
  --tags string       Comma-separated tags to filter containers
  --group-by string   Comma-separated tags to group containers
  --sort string       Attribute to sort by (name, started_at)
  --limit int         Maximum results to display (default: 100)
  --image string      Filter by image name (partial match)
  --state string      Filter by state (running, exited, paused)
  --json              Output in JSON format

EXAMPLES:
  # List all containers
  dd containers

  # List running containers only
  dd containers --state running

  # Filter by image name
  dd containers --image nginx

  # Filter by tags (environment and service)
  dd containers --tags "env:production,service:web"

  # Group by service
  dd containers --group-by service

  # Sort by start time
  dd containers --sort started_at --limit 20

  # Kubernetes pods (containers with kube tags)
  dd containers --tags "kube_namespace:default"

  # Get JSON output for scripting
  dd containers --state running --json

CONTAINER STATES:
  running  - Container is actively running
  exited   - Container has stopped
  paused   - Container execution is paused
  created  - Container created but not started
  dead     - Container is in dead state

USE CASES:
  1. Monitor container health across infrastructure
  2. Track Kubernetes pod states
  3. Identify containers using specific images
  4. Find containers in error states
  5. Audit container inventory

FILTERING:
  Use --tags for precise filtering:
    env:production        Production environment
    service:web           Web service containers
    kube_namespace:prod   Kubernetes namespace
    team:platform         Team ownership

  Combine multiple tags for AND logic:
    --tags "env:prod,service:api"

KUBERNETES INTEGRATION:
  Kubernetes pods appear as containers with tags:
    kube_namespace        Kubernetes namespace
    kube_deployment       Deployment name
    kube_service          Service name
    pod_name              Pod name
    kube_container_name   Container name within pod

  Example: List all pods in production namespace
    dd containers --tags "kube_namespace:production"

AUTHENTICATION:
  Requires DD_API_KEY and DD_APP_KEY environment variables.
`
	fmt.Println(strings.TrimSpace(help))
}
