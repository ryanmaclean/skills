package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/datadog/skill/internal/client"
)

// KubernetesCommand queries Kubernetes monitoring data
type KubernetesCommand struct {
	flags      *flag.FlagSet
	namespace  string
	deployment string
	service    string
	pod        string
	node       string
	state      string
	limit      int
	jsonOut    bool
}

// NewKubernetesCommand creates a new kubernetes command
func NewKubernetesCommand() Command {
	cmd := &KubernetesCommand{
		flags: flag.NewFlagSet("kubernetes", flag.ExitOnError),
	}

	cmd.flags.StringVar(&cmd.namespace, "namespace", "", "Kubernetes namespace")
	cmd.flags.StringVar(&cmd.deployment, "deployment", "", "Filter by deployment name")
	cmd.flags.StringVar(&cmd.service, "service", "", "Filter by service name")
	cmd.flags.StringVar(&cmd.pod, "pod", "", "Filter by pod name (partial match)")
	cmd.flags.StringVar(&cmd.node, "node", "", "Filter by node name")
	cmd.flags.StringVar(&cmd.state, "state", "running", "Filter by pod state (running, pending, failed, succeeded)")
	cmd.flags.IntVar(&cmd.limit, "limit", 100, "Maximum number of results (default: 100)")
	cmd.flags.BoolVar(&cmd.jsonOut, "json", false, "Output in JSON format")

	return cmd
}

// Name returns the command name
func (c *KubernetesCommand) Name() string {
	return "kubernetes"
}

// Description returns a short description
func (c *KubernetesCommand) Description() string {
	return "Query Kubernetes pod and cluster monitoring"
}

// Run executes the kubernetes command
func (c *KubernetesCommand) Run(args []string) error {
	if err := c.flags.Parse(args); err != nil {
		return err
	}

	ddClient, err := client.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Datadog client: %w", err)
	}

	// Build Kubernetes-specific tag filters
	var tagFilters []string

	if c.namespace != "" {
		tagFilters = append(tagFilters, fmt.Sprintf("kube_namespace:%s", c.namespace))
	}

	if c.deployment != "" {
		tagFilters = append(tagFilters, fmt.Sprintf("kube_deployment:%s", c.deployment))
	}

	if c.service != "" {
		tagFilters = append(tagFilters, fmt.Sprintf("kube_service:%s", c.service))
	}

	if c.node != "" {
		tagFilters = append(tagFilters, fmt.Sprintf("host:%s", c.node))
	}

	// Build query parameters
	params := make(map[string]interface{})

	if len(tagFilters) > 0 {
		params["filter[tags]"] = strings.Join(tagFilters, ",")
	}

	params["page[size]"] = c.limit

	resp, err := ddClient.ListContainers(params)
	if err != nil {
		return fmt.Errorf("failed to query Kubernetes pods: %w", err)
	}

	if c.jsonOut {
		fmt.Println(string(resp))
		return nil
	}

	return c.printResults(resp)
}

// printResults formats and displays Kubernetes pod results
func (c *KubernetesCommand) printResults(data []byte) error {
	var result struct {
		Data []struct {
			Type       string `json:"type"`
			ID         string `json:"id"`
			Attributes struct {
				ContainerID string    `json:"container_id"`
				CreatedAt   time.Time `json:"created_at"`
				Host        string    `json:"host"`
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
				TotalCount int `json:"total_count"`
			} `json:"pagination"`
		} `json:"meta"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("failed to parse results: %w", err)
	}

	// Extract Kubernetes metadata from tags
	type PodInfo struct {
		Container      string
		PodName        string
		Namespace      string
		Deployment     string
		Service        string
		ContainerName  string
		Node           string
		State          string
		Image          string
		StartedAt      time.Time
		Uptime         time.Duration
	}

	pods := make([]PodInfo, 0)

	for _, container := range result.Data {
		info := PodInfo{
			Container: container.Attributes.Name,
			State:     container.Attributes.State,
			Image:     container.Attributes.ImageName,
			StartedAt: container.Attributes.StartedAt,
			Uptime:    time.Since(container.Attributes.StartedAt),
			Node:      container.Attributes.Host,
		}

		// Extract Kubernetes tags
		for _, tag := range container.Attributes.Tags {
			parts := strings.SplitN(tag, ":", 2)
			if len(parts) != 2 {
				continue
			}
			key, value := parts[0], parts[1]

			switch key {
			case "pod_name":
				info.PodName = value
			case "kube_namespace":
				info.Namespace = value
			case "kube_deployment":
				info.Deployment = value
			case "kube_service":
				info.Service = value
			case "kube_container_name":
				info.ContainerName = value
			}
		}

		// Apply filters
		if c.pod != "" && !strings.Contains(strings.ToLower(info.PodName), strings.ToLower(c.pod)) {
			continue
		}

		if c.state != "" && !strings.EqualFold(info.State, c.state) {
			continue
		}

		pods = append(pods, info)
	}

	totalCount := result.Meta.Pagination.TotalCount
	displayedCount := len(pods)

	fmt.Printf("Kubernetes Pods (Showing: %d, Total: %d):\n", displayedCount, totalCount)
	fmt.Println(strings.Repeat("=", 80))

	if displayedCount == 0 {
		fmt.Println("No Kubernetes pods found matching the criteria.")
		return nil
	}

	// Group by namespace
	namespaceGroups := make(map[string]int)
	for _, pod := range pods {
		if pod.Namespace != "" {
			namespaceGroups[pod.Namespace]++
		}
	}

	if len(namespaceGroups) > 0 {
		fmt.Println("\nNamespace Summary:")
		for ns, count := range namespaceGroups {
			fmt.Printf("  %s: %d pods\n", ns, count)
		}
	}

	// Group by state
	stateGroups := make(map[string]int)
	for _, pod := range pods {
		stateGroups[pod.State]++
	}

	fmt.Println("\nState Summary:")
	for state, count := range stateGroups {
		fmt.Printf("  %s: %d\n", state, count)
	}

	fmt.Println("\nPod Details:")
	fmt.Println(strings.Repeat("-", 80))

	for i, pod := range pods {
		if i >= 50 {
			fmt.Printf("\n... and %d more pods (use --limit to see more)\n", displayedCount-50)
			break
		}

		// State indicator
		stateIcon := "●"
		if pod.State == "running" {
			stateIcon = "✓"
		} else if pod.State == "exited" || pod.State == "failed" {
			stateIcon = "✗"
		} else if pod.State == "pending" {
			stateIcon = "⏳"
		}

		fmt.Printf("\n%s Pod: %s\n", stateIcon, pod.PodName)

		if pod.Namespace != "" {
			fmt.Printf("  Namespace: %s\n", pod.Namespace)
		}

		if pod.ContainerName != "" {
			fmt.Printf("  Container: %s\n", pod.ContainerName)
		}

		fmt.Printf("  Image: %s\n", pod.Image)
		fmt.Printf("  State: %s\n", pod.State)

		if pod.Node != "" {
			fmt.Printf("  Node: %s\n", pod.Node)
		}

		if pod.Deployment != "" {
			fmt.Printf("  Deployment: %s\n", pod.Deployment)
		}

		if pod.Service != "" {
			fmt.Printf("  Service: %s\n", pod.Service)
		}

		if pod.Uptime > 0 {
			fmt.Printf("  Uptime: %s\n", formatDuration(pod.Uptime))
		}
	}

	return nil
}

// Help displays help information
func (c *KubernetesCommand) Help() {
	help := `dd kubernetes - Query Kubernetes Monitoring

DESCRIPTION:
  Query Kubernetes pod and cluster monitoring. Track pod health, deployment
  status, and cluster resources across namespaces.

USAGE:
  dd kubernetes [options]

OPTIONS:
  --namespace string    Kubernetes namespace
  --deployment string   Filter by deployment name
  --service string      Filter by service name
  --pod string          Filter by pod name (partial match)
  --node string         Filter by node name
  --state string        Filter by pod state (running, pending, failed, succeeded)
  --limit int           Maximum results to display (default: 100)
  --json                Output in JSON format

EXAMPLES:
  # List all running pods
  dd kubernetes

  # List pods in specific namespace
  dd kubernetes --namespace production

  # List pods for a deployment
  dd kubernetes --deployment web-app

  # Find failing pods
  dd kubernetes --state failed

  # List pods on specific node
  dd kubernetes --node ip-10-0-1-100

  # Search for pods by name
  dd kubernetes --pod api

  # Combine filters
  dd kubernetes --namespace prod --deployment api --state running

  # Get JSON output
  dd kubernetes --namespace prod --json

POD STATES:
  running    - Pod is running normally
  pending    - Pod is waiting to be scheduled
  failed     - Pod has failed to start
  succeeded  - Pod completed successfully
  unknown    - Pod state is unknown

USE CASES:
  1. Monitor pod health across namespaces
  2. Identify failing or pending pods
  3. Track deployment rollout status
  4. Debug pod scheduling issues
  5. Audit cluster resource usage

FILTERING BY NAMESPACE:
  List all production pods:
    dd kubernetes --namespace production

  List development pods:
    dd kubernetes --namespace dev

  Compare staging and production:
    dd kubernetes --namespace staging
    dd kubernetes --namespace prod

DEBUGGING WORKFLOWS:
  Find failing pods:
    dd kubernetes --state failed

  Check pending pods (scheduling issues):
    dd kubernetes --state pending

  Monitor deployment rollout:
    dd kubernetes --deployment my-app

  Investigate node issues:
    dd kubernetes --node problem-node

KUBERNETES TAGS:
  Datadog automatically tags Kubernetes containers:
    kube_namespace        Namespace name
    kube_deployment       Deployment name
    kube_service          Service name
    pod_name              Pod name
    kube_container_name   Container within pod
    host                  Node name

AUTHENTICATION:
  Requires DD_API_KEY and DD_APP_KEY environment variables.
`
	fmt.Println(strings.TrimSpace(help))
}
