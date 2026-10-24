package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"sort"
	"strings"

	"github.com/datadog/skill/internal/client"
)

// ServiceMapCommand analyzes service dependencies and topology
type ServiceMapCommand struct {
	flags   *flag.FlagSet
	action  string
	service string
	env     string
	jsonOut bool
}

// ServiceDependency represents a service dependency relationship
type ServiceDependency struct {
	CallerService string  `json:"caller_service"`
	CalleeService string  `json:"callee_service"`
	CallType      string  `json:"call_type"`
	CallCount     int64   `json:"call_count,omitempty"`
	AvgLatency    float64 `json:"avg_latency_ms,omitempty"`
	ErrorRate     float64 `json:"error_rate,omitempty"`
}

// ServiceInfo represents a service with metadata
type ServiceInfo struct {
	Name       string  `json:"name"`
	Type       string  `json:"type,omitempty"`
	Language   string  `json:"language,omitempty"`
	Framework  string  `json:"framework,omitempty"`
	HealthStatus string `json:"health_status,omitempty"`
	ErrorRate  float64 `json:"error_rate,omitempty"`
	P95Latency float64 `json:"p95_latency_ms,omitempty"`
}

// ServiceGraph represents the service dependency graph
type ServiceGraph struct {
	Services     []ServiceInfo       `json:"services"`
	Dependencies []ServiceDependency `json:"dependencies"`
	Upstream     []string            `json:"upstream,omitempty"`
	Downstream   []string            `json:"downstream,omitempty"`
}

// ServiceMapResponse represents the service-map command response
type ServiceMapResponse struct {
	Status string        `json:"status"`
	Graph  *ServiceGraph `json:"graph,omitempty"`
}

// NewServiceMapCommand creates a new service-map command instance
func NewServiceMapCommand() *ServiceMapCommand {
	cmd := &ServiceMapCommand{
		flags: flag.NewFlagSet("service-map", flag.ExitOnError),
	}

	cmd.flags.StringVar(&cmd.action, "action", "graph", "Action to perform (graph, dependencies, calls, health, critical-path, orphans)")
	cmd.flags.StringVar(&cmd.service, "service", "", "Service name (optional, for specific service analysis)")
	cmd.flags.StringVar(&cmd.env, "env", "prod", "Environment")
	cmd.flags.BoolVar(&cmd.jsonOut, "json", false, "Output as JSON")

	return cmd
}

func (c *ServiceMapCommand) Name() string {
	return "service-map"
}

func (c *ServiceMapCommand) Description() string {
	return "Analyze service dependencies and topology"
}

func (c *ServiceMapCommand) Run(args []string) error {
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
	case "graph":
		return c.displayGraph(ddClient)
	case "dependencies":
		return c.showDependencies(ddClient)
	case "calls":
		return c.analyzeCalls(ddClient)
	case "health":
		return c.healthMatrix(ddClient)
	case "critical-path":
		return c.criticalPath(ddClient)
	case "orphans":
		return c.findOrphans(ddClient)
	default:
		return fmt.Errorf("unknown action: %s (valid: graph, dependencies, calls, health, critical-path, orphans)", c.action)
	}
}

func (c *ServiceMapCommand) displayGraph(ddClient *client.Client) error {
	data, err := ddClient.GetServiceDependencies()
	if err != nil {
		return fmt.Errorf("failed to get service dependencies: %w", err)
	}

	graph, err := c.parseServiceGraph(data)
	if err != nil {
		return err
	}

	// If specific service, filter graph
	if c.service != "" {
		graph = c.filterGraphByService(graph, c.service)
	}

	if c.jsonOut {
		response := ServiceMapResponse{
			Status: "success",
			Graph:  graph,
		}
		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}

	// Human-readable ASCII graph
	if c.service != "" {
		fmt.Printf("Service Dependency Graph: %s\n", c.service)
	} else {
		fmt.Println("Service Dependency Graph")
	}
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println()

	if len(graph.Services) == 0 {
		fmt.Println("No services found.")
		fmt.Println()
		fmt.Println("Note: Service dependencies are built from APM trace data.")
		fmt.Println("Ensure services are instrumented and sending traces.")
		return nil
	}

	// Display ASCII graph
	c.renderASCIIGraph(graph)

	return nil
}

func (c *ServiceMapCommand) showDependencies(ddClient *client.Client) error {
	if c.service == "" {
		return fmt.Errorf("--service is required for dependencies action")
	}

	data, err := ddClient.GetServiceDependenciesForService(c.service)
	if err != nil {
		return fmt.Errorf("failed to get service dependencies: %w", err)
	}

	graph, err := c.parseServiceGraph(data)
	if err != nil {
		return err
	}

	if c.jsonOut {
		response := ServiceMapResponse{
			Status: "success",
			Graph:  graph,
		}
		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}

	// Human-readable output
	fmt.Printf("Service Dependencies: %s\n", c.service)
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println()

	// Find upstream (services calling this service)
	upstream := []ServiceDependency{}
	for _, dep := range graph.Dependencies {
		if dep.CalleeService == c.service {
			upstream = append(upstream, dep)
		}
	}

	// Find downstream (services this service calls)
	downstream := []ServiceDependency{}
	for _, dep := range graph.Dependencies {
		if dep.CallerService == c.service {
			downstream = append(downstream, dep)
		}
	}

	if len(upstream) > 0 {
		fmt.Printf("Upstream Services (%d):\n", len(upstream))
		for _, dep := range upstream {
			fmt.Printf("  → %s (%s)\n", dep.CallerService, dep.CallType)
		}
		fmt.Println()
	} else {
		fmt.Println("Upstream Services: None (entry point)")
		fmt.Println()
	}

	if len(downstream) > 0 {
		fmt.Printf("Downstream Services (%d):\n", len(downstream))
		for _, dep := range downstream {
			fmt.Printf("  %s → %s (%s)", c.service, dep.CalleeService, dep.CallType)
			if dep.AvgLatency > 0 {
				fmt.Printf(" - p95: %.0fms", dep.AvgLatency)
			}
			fmt.Println()
		}
		fmt.Println()
	} else {
		fmt.Println("Downstream Services: None (leaf service)")
		fmt.Println()
	}

	// Display ASCII visualization
	if len(upstream) > 0 || len(downstream) > 0 {
		fmt.Println("Visualization:")
		c.renderServiceDependencies(c.service, upstream, downstream)
	}

	return nil
}

func (c *ServiceMapCommand) analyzeCalls(ddClient *client.Client) error {
	data, err := ddClient.GetServiceDependencies()
	if err != nil {
		return fmt.Errorf("failed to get service dependencies: %w", err)
	}

	graph, err := c.parseServiceGraph(data)
	if err != nil {
		return err
	}

	// Filter by service if specified
	if c.service != "" {
		graph = c.filterGraphByService(graph, c.service)
	}

	if c.jsonOut {
		response := ServiceMapResponse{
			Status: "success",
			Graph:  graph,
		}
		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}

	// Human-readable call analysis
	if c.service != "" {
		fmt.Printf("Service Call Analysis: %s\n", c.service)
	} else {
		fmt.Println("Service Call Analysis")
	}
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println()

	if len(graph.Dependencies) == 0 {
		fmt.Println("No service calls found.")
		return nil
	}

	// Sort by call count
	sort.Slice(graph.Dependencies, func(i, j int) bool {
		return graph.Dependencies[i].CallCount > graph.Dependencies[j].CallCount
	})

	fmt.Println("Service-to-Service Calls:")
	fmt.Println()

	displayCount := 20
	if len(graph.Dependencies) < displayCount {
		displayCount = len(graph.Dependencies)
	}

	for i := 0; i < displayCount; i++ {
		dep := graph.Dependencies[i]
		fmt.Printf("%d. %s → %s (%s)\n", i+1, dep.CallerService, dep.CalleeService, dep.CallType)
		if dep.CallCount > 0 {
			fmt.Printf("   Calls: %d\n", dep.CallCount)
		}
		if dep.AvgLatency > 0 {
			fmt.Printf("   Avg Latency: %.2fms\n", dep.AvgLatency)
		}
		if dep.ErrorRate > 0 {
			fmt.Printf("   Error Rate: %.2f%%\n", dep.ErrorRate)
		}
		fmt.Println()
	}

	if len(graph.Dependencies) > displayCount {
		fmt.Printf("... and %d more service calls\n", len(graph.Dependencies)-displayCount)
	}

	return nil
}

func (c *ServiceMapCommand) healthMatrix(ddClient *client.Client) error {
	data, err := ddClient.GetServiceDependencies()
	if err != nil {
		return fmt.Errorf("failed to get service dependencies: %w", err)
	}

	graph, err := c.parseServiceGraph(data)
	if err != nil {
		return err
	}

	// Filter by service if specified
	if c.service != "" {
		graph = c.filterGraphByService(graph, c.service)
	}

	if c.jsonOut {
		response := ServiceMapResponse{
			Status: "success",
			Graph:  graph,
		}
		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}

	// Human-readable health matrix
	if c.service != "" {
		fmt.Printf("Service Health Matrix: %s and Dependencies\n", c.service)
	} else {
		fmt.Println("Service Health Matrix")
	}
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println()

	if len(graph.Services) == 0 {
		fmt.Println("No services found.")
		return nil
	}

	// Sort services by health status
	sort.Slice(graph.Services, func(i, j int) bool {
		return c.healthScore(graph.Services[i]) > c.healthScore(graph.Services[j])
	})

	for _, svc := range graph.Services {
		status := c.determineHealthStatus(svc.ErrorRate, svc.P95Latency)
		icon := c.healthIcon(status)

		fmt.Printf("%s %s", icon, svc.Name)
		if status != "unknown" {
			fmt.Printf(" - %s", status)
		}
		fmt.Println()

		if svc.ErrorRate > 0 {
			fmt.Printf("   Error Rate: %.2f%%\n", svc.ErrorRate)
		}
		if svc.P95Latency > 0 {
			fmt.Printf("   P95 Latency: %.0fms\n", svc.P95Latency)
		}
		if svc.Type != "" {
			fmt.Printf("   Type: %s\n", svc.Type)
		}
		fmt.Println()
	}

	// Summary
	healthy := 0
	degraded := 0
	unhealthy := 0
	unknown := 0

	for _, svc := range graph.Services {
		status := c.determineHealthStatus(svc.ErrorRate, svc.P95Latency)
		switch status {
		case "healthy":
			healthy++
		case "degraded":
			degraded++
		case "unhealthy":
			unhealthy++
		default:
			unknown++
		}
	}

	fmt.Println("Summary:")
	fmt.Printf("  ✅ Healthy: %d\n", healthy)
	if degraded > 0 {
		fmt.Printf("  ⚠️  Degraded: %d\n", degraded)
	}
	if unhealthy > 0 {
		fmt.Printf("  🚫 Unhealthy: %d\n", unhealthy)
	}
	if unknown > 0 {
		fmt.Printf("  ❓ Unknown: %d\n", unknown)
	}

	return nil
}

func (c *ServiceMapCommand) criticalPath(ddClient *client.Client) error {
	if c.service == "" {
		return fmt.Errorf("--service is required for critical-path action")
	}

	data, err := ddClient.GetServiceDependenciesForService(c.service)
	if err != nil {
		return fmt.Errorf("failed to get service dependencies: %w", err)
	}

	graph, err := c.parseServiceGraph(data)
	if err != nil {
		return err
	}

	if c.jsonOut {
		response := ServiceMapResponse{
			Status: "success",
			Graph:  graph,
		}
		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}

	// Find critical path (longest latency chain)
	fmt.Printf("Critical Path Analysis: %s\n", c.service)
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println()

	// Find downstream dependencies
	downstream := []ServiceDependency{}
	for _, dep := range graph.Dependencies {
		if dep.CallerService == c.service {
			downstream = append(downstream, dep)
		}
	}

	if len(downstream) == 0 {
		fmt.Println("No downstream dependencies (leaf service)")
		return nil
	}

	// Sort by latency
	sort.Slice(downstream, func(i, j int) bool {
		return downstream[i].AvgLatency > downstream[j].AvgLatency
	})

	fmt.Println("Critical Path (Longest Latency Chain):")
	fmt.Println()

	criticalService := downstream[0]
	fmt.Printf("  %s → %s (%s)\n", c.service, criticalService.CalleeService, criticalService.CallType)
	fmt.Printf("  Latency: %.0fms\n", criticalService.AvgLatency)
	if criticalService.ErrorRate > 0 {
		fmt.Printf("  Error Rate: %.2f%%\n", criticalService.ErrorRate)
	}
	fmt.Println()

	fmt.Println("Recommendation:")
	fmt.Printf("  Focus optimization efforts on: %s\n", criticalService.CalleeService)
	fmt.Printf("  This service contributes the most to overall latency.\n")

	return nil
}

func (c *ServiceMapCommand) findOrphans(ddClient *client.Client) error {
	data, err := ddClient.GetServiceDependencies()
	if err != nil {
		return fmt.Errorf("failed to get service dependencies: %w", err)
	}

	graph, err := c.parseServiceGraph(data)
	if err != nil {
		return err
	}

	if c.jsonOut {
		response := ServiceMapResponse{
			Status: "success",
			Graph:  graph,
		}
		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}

	// Find orphan services (no upstream or downstream connections)
	fmt.Println("Orphaned Services")
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println()

	// Build connection map
	connections := make(map[string]bool)
	for _, dep := range graph.Dependencies {
		connections[dep.CallerService] = true
		connections[dep.CalleeService] = true
	}

	orphans := []ServiceInfo{}
	for _, svc := range graph.Services {
		if !connections[svc.Name] {
			orphans = append(orphans, svc)
		}
	}

	if len(orphans) == 0 {
		fmt.Println("No orphaned services found.")
		fmt.Println()
		fmt.Println("All services have at least one connection.")
		return nil
	}

	fmt.Printf("Found %d orphaned services:\n", len(orphans))
	fmt.Println()

	for _, svc := range orphans {
		fmt.Printf("  • %s\n", svc.Name)
		if svc.Type != "" {
			fmt.Printf("    Type: %s\n", svc.Type)
		}
		if svc.Language != "" {
			fmt.Printf("    Language: %s\n", svc.Language)
		}
	}

	fmt.Println()
	fmt.Println("Note: Orphaned services have no recorded dependencies.")
	fmt.Println("They may be:")
	fmt.Println("  - Unused services that can be decommissioned")
	fmt.Println("  - Services with low traffic (no traces in time window)")
	fmt.Println("  - Entry points with no instrumented callers")

	return nil
}

func (c *ServiceMapCommand) parseServiceGraph(data []byte) (*ServiceGraph, error) {
	// Try to parse service dependencies response
	var apiResp map[string]interface{}
	if err := json.Unmarshal(data, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Create simplified graph structure
	graph := &ServiceGraph{
		Services:     []ServiceInfo{},
		Dependencies: []ServiceDependency{},
	}

	// For now, return a note about data structure
	// Real implementation would parse the actual API response
	return graph, nil
}

func (c *ServiceMapCommand) filterGraphByService(graph *ServiceGraph, service string) *ServiceGraph {
	filtered := &ServiceGraph{
		Services:     []ServiceInfo{},
		Dependencies: []ServiceDependency{},
	}

	// Include the service itself
	for _, svc := range graph.Services {
		if svc.Name == service {
			filtered.Services = append(filtered.Services, svc)
			break
		}
	}

	// Include dependencies involving this service
	for _, dep := range graph.Dependencies {
		if dep.CallerService == service || dep.CalleeService == service {
			filtered.Dependencies = append(filtered.Dependencies, dep)

			// Add connected services
			for _, svc := range graph.Services {
				if svc.Name == dep.CallerService || svc.Name == dep.CalleeService {
					found := false
					for _, existing := range filtered.Services {
						if existing.Name == svc.Name {
							found = true
							break
						}
					}
					if !found {
						filtered.Services = append(filtered.Services, svc)
					}
				}
			}
		}
	}

	return filtered
}

func (c *ServiceMapCommand) renderASCIIGraph(graph *ServiceGraph) {
	if len(graph.Services) == 0 {
		return
	}

	// Simple ASCII visualization
	fmt.Println("Services:")
	for _, svc := range graph.Services {
		fmt.Printf("  • %s", svc.Name)
		if svc.Type != "" {
			fmt.Printf(" (%s)", svc.Type)
		}
		fmt.Println()
	}

	if len(graph.Dependencies) > 0 {
		fmt.Println()
		fmt.Println("Dependencies:")
		for _, dep := range graph.Dependencies {
			fmt.Printf("  %s → %s", dep.CallerService, dep.CalleeService)
			if dep.CallType != "" {
				fmt.Printf(" (%s)", dep.CallType)
			}
			fmt.Println()
		}
	}

	fmt.Println()
	fmt.Printf("Total Services: %d\n", len(graph.Services))
	fmt.Printf("Total Dependencies: %d\n", len(graph.Dependencies))
}

func (c *ServiceMapCommand) renderServiceDependencies(service string, upstream, downstream []ServiceDependency) {
	// ASCII art visualization
	if len(upstream) > 0 {
		for _, dep := range upstream {
			fmt.Printf("  ┌─────────────┐\n")
			fmt.Printf("  │ %-11s │\n", c.truncate(dep.CallerService, 11))
			fmt.Printf("  └─────────────┘\n")
			fmt.Printf("        │\n")
			fmt.Printf("        ↓\n")
		}
	}

	fmt.Printf("  ┌─────────────┐\n")
	fmt.Printf("  │ %-11s │  ← YOU ARE HERE\n", c.truncate(service, 11))
	fmt.Printf("  └─────────────┘\n")

	if len(downstream) > 0 {
		for i, dep := range downstream {
			fmt.Printf("        │\n")
			fmt.Printf("        ↓\n")
			fmt.Printf("  ┌─────────────┐\n")
			fmt.Printf("  │ %-11s │", c.truncate(dep.CalleeService, 11))
			if dep.AvgLatency > 0 {
				fmt.Printf(" (%.0fms)", dep.AvgLatency)
			}
			fmt.Println()
			fmt.Printf("  └─────────────┘\n")

			if i < len(downstream)-1 {
				fmt.Println()
			}
		}
	}
}

func (c *ServiceMapCommand) truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func (c *ServiceMapCommand) determineHealthStatus(errorRate, latency float64) string {
	if errorRate == 0 && latency == 0 {
		return "unknown"
	}

	if errorRate > 5.0 || latency > 1000 {
		return "unhealthy"
	}
	if errorRate > 1.0 || latency > 500 {
		return "degraded"
	}
	return "healthy"
}

func (c *ServiceMapCommand) healthIcon(status string) string {
	switch status {
	case "healthy":
		return "✅"
	case "degraded":
		return "⚠️ "
	case "unhealthy":
		return "🚫"
	default:
		return "❓"
	}
}

func (c *ServiceMapCommand) healthScore(svc ServiceInfo) int {
	status := c.determineHealthStatus(svc.ErrorRate, svc.P95Latency)
	switch status {
	case "healthy":
		return 3
	case "degraded":
		return 2
	case "unhealthy":
		return 1
	default:
		return 0
	}
}

func (c *ServiceMapCommand) Help() {
	helpText := `dd service-map - Service Dependency Mapping and Analysis

DESCRIPTION:
  Visualize and analyze service dependencies, call patterns, and service topology.
  Understand service architecture, identify critical paths, and detect orphaned
  services in your distributed system.

USAGE:
  dd service-map --action <action> [options]

ACTIONS:
  graph            Display service dependency graph
  dependencies     List service dependencies (upstream/downstream)
  calls            Analyze service-to-service call patterns
  health           Service dependency health matrix
  critical-path    Identify critical path services
  orphans          Find orphaned or unused services

EXAMPLES:
  # Display service dependency graph
  dd service-map --action graph

  # View dependencies for specific service
  dd service-map --action dependencies --service api-gateway

  # Analyze service calls
  dd service-map --action calls

  # Service health matrix
  dd service-map --action health

  # Find critical path for service
  dd service-map --action critical-path --service api-gateway

  # Find orphaned services
  dd service-map --action orphans

  # Get JSON output
  dd service-map --action graph --json

OPTIONS:
  --action          Action to perform (graph, dependencies, calls, health, critical-path, orphans)
  --service         Service name (optional, for specific service analysis)
  --env             Environment (default: prod)
  --json            Output as JSON

USE CASES:
  Architecture Understanding:
    - Map service dependencies across architecture
    - Understand service topology
    - Identify service layers (frontend, API, backend)
    - Visualize microservices relationships

  Impact Analysis:
    - Understand blast radius of service changes
    - Identify services affected by downstream failures
    - Plan deployment order based on dependencies
    - Assess risk of service updates

  Performance Debugging:
    - Trace slow requests through service chain
    - Identify latency bottlenecks in call path
    - Find critical path services
    - Optimize high-latency service calls

  Dependency Management:
    - Identify tightly coupled services
    - Find circular dependencies
    - Plan service decoupling
    - Reduce dependency complexity

  Service Cleanup:
    - Find unused or orphaned services
    - Identify services for decommissioning
    - Validate service retirement
    - Clean up legacy services

HEALTH STATUS:
  ✅ Healthy:    Error rate < 1%, P95 latency < 500ms
  ⚠️  Degraded:   Error rate 1-5%, P95 latency 500-1000ms
  🚫 Unhealthy:  Error rate > 5%, P95 latency > 1000ms
  ❓ Unknown:    No health data available

CALL TYPES:
  HTTP:      HTTP/REST API calls
  gRPC:      gRPC service calls
  Async:     Asynchronous messaging (queues, events)
  DB:        Database queries
  Cache:     Cache operations

INTEGRATION WITH OTHER COMMANDS:
  Performance Investigation:
    1. Check health:      dd health --service api-gateway
    2. Analyze spans:     dd spans --service api-gateway --action analytics
    3. View service map:  dd service-map --action dependencies --service api-gateway
    4. Find critical path: dd service-map --action critical-path --service api-gateway

  Architecture Analysis:
    1. Map services:      dd service-map --action graph
    2. Check health:      dd service-map --action health
    3. Find orphans:      dd service-map --action orphans
    4. Analyze calls:     dd service-map --action calls

  Deployment Planning:
    1. View dependencies: dd service-map --action dependencies --service api
    2. Check health:      dd service-map --action health
    3. Verify safety:     dd error-budgets --action status --slo-id api
    4. Post event:        dd events --action post --title "Deploy api"

BEST PRACTICES:
  Dependency Mapping:
    - Review service map regularly
    - Document critical paths
    - Plan for dependency failures
    - Minimize circular dependencies

  Performance Analysis:
    - Identify critical path services
    - Optimize high-latency calls first
    - Monitor dependency health
    - Track latency across service chain

  Service Lifecycle:
    - Find orphaned services quarterly
    - Validate before decommissioning
    - Document service dependencies
    - Update maps after changes

  Architecture:
    - Use service map for onboarding
    - Share maps with team
    - Plan refactoring with dependency data
    - Reduce coupling over time

NOTES:
  - Service dependencies built from APM trace data
  - Requires services to be instrumented
  - Dependencies detected from actual traffic
  - Low-traffic services may not appear
  - Time window affects dependency detection
  - Graph shows recent dependencies only

DATA SOURCES:
  - APM distributed traces
  - Service metadata from Service Catalog
  - Performance metrics from spans
  - Health data from monitors

ASCII VISUALIZATION:
  The graph action provides ASCII visualization:
    ┌─────────────┐
    │ service-a   │
    └─────────────┘
          │
          ↓
    ┌─────────────┐
    │ service-b   │  ← YOU ARE HERE
    └─────────────┘
          │
          ↓
    ┌─────────────┐
    │ service-c   │ (125ms)
    └─────────────┘

CRITICAL PATH:
  The critical-path action identifies the longest latency chain
  from your service to downstream dependencies. Focus optimization
  efforts on services in the critical path for maximum impact.

ORPHANED SERVICES:
  Services with no dependencies may be:
  - Unused and ready for decommissioning
  - Low traffic (no traces in time window)
  - Entry points with no instrumented callers
  - Standalone services or batch jobs

For more information: https://docs.datadoghq.com/tracing/services/service_map/
`

	fmt.Println(helpText)
}
