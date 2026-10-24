package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/datadog/skill/internal/client"
	"github.com/datadog/skill/internal/observability"
)

// CatalogCommand queries Datadog Service Catalog
type CatalogCommand struct {
	flags  *flag.FlagSet
	search string
	team   string
	env    string
	json   bool
}

// ServiceDefinition represents a service from the catalog
type ServiceDefinition struct {
	Name        string   `json:"name"`
	Kind        string   `json:"kind,omitempty"`
	Description string   `json:"description,omitempty"`
	Tier        string   `json:"tier,omitempty"`
	Lifecycle   string   `json:"lifecycle,omitempty"`
	Application string   `json:"application,omitempty"`
	Team        string   `json:"team,omitempty"`
	Owner       string   `json:"owner,omitempty"`
	Languages   []string `json:"languages,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Repos       []string `json:"repos,omitempty"`
	Docs        []string `json:"docs,omitempty"`
	Dashboards  []string `json:"dashboards,omitempty"`
}

// CatalogOutput represents the structured output
type CatalogOutput struct {
	Status       string                 `json:"status"`
	TotalCount   int                    `json:"total_count"`
	FilteredCount int                   `json:"filtered_count"`
	Summary      *CatalogSummary        `json:"summary,omitempty"`
	Services     []ServiceDefinition    `json:"services"`
	RawData      map[string]interface{} `json:"raw_data,omitempty"`
}

// CatalogSummary contains summary statistics
type CatalogSummary struct {
	ByKind    map[string]int `json:"by_kind"`
	ByTier    map[string]int `json:"by_tier"`
	WithTeam  int            `json:"with_team"`
	WithDocs  int            `json:"with_docs"`
	WithRepos int            `json:"with_repos"`
}

// NewCatalogCommand creates a new catalog command
func NewCatalogCommand() *CatalogCommand {
	cmd := &CatalogCommand{
		flags: flag.NewFlagSet("catalog", flag.ExitOnError),
	}

	cmd.flags.StringVar(&cmd.search, "search", "", "Search term for service name")
	cmd.flags.StringVar(&cmd.team, "team", "", "Filter by team tag")
	cmd.flags.StringVar(&cmd.env, "env", "", "Filter by environment tag")
	cmd.flags.BoolVar(&cmd.json, "json", false, "Output as JSON")

	return cmd
}

// Name returns the command name
func (c *CatalogCommand) Name() string {
	return "catalog"
}

// Description returns the command description
func (c *CatalogCommand) Description() string {
	return "Query Datadog Service Catalog for service metadata and ownership"
}

// Run executes the catalog command
func (c *CatalogCommand) Run(args []string) error {
	// Initialize observability
	obs, err := observability.Init("query-service-catalog", "production")
	if err != nil {
		return fmt.Errorf("failed to init observability: %w", err)
	}
	defer obs.Shutdown(0)

	// Parse flags
	if err := c.flags.Parse(args); err != nil {
		return err
	}

	// Start tracing
	span := obs.StartSpan("catalog.query")
	defer obs.FinishSpan(span)

	obs.LogInfo("Starting service catalog query")

	// Create Datadog client
	clientSpan := obs.StartSpan("catalog.create_client")
	ddClient, err := client.NewClient()
	if err != nil {
		obs.LogError("Failed to create Datadog client: " + err.Error())
		obs.FinishSpan(clientSpan)
		return fmt.Errorf("failed to create Datadog client: %w", err)
	}
	obs.FinishSpan(clientSpan)

	// Query service catalog
	querySpan := obs.StartSpan("catalog.api_query")
	if c.search != "" {
		obs.GetTracer().SetTag(querySpan, "search", c.search)
	}
	if c.team != "" {
		obs.GetTracer().SetTag(querySpan, "team", c.team)
	}
	if c.env != "" {
		obs.GetTracer().SetTag(querySpan, "env", c.env)
	}

	start := time.Now()
	rawData, err := ddClient.GetServiceCatalog()
	apiDuration := time.Since(start).Milliseconds()

	if err != nil {
		obs.LogError("API call failed: " + err.Error())
		obs.RecordAPICall("/api/v2/services/definitions", "GET", 500, float64(apiDuration), err)
		obs.GetTracer().SetError(querySpan, err)
		obs.FinishSpan(querySpan)
		return fmt.Errorf("failed to query service catalog: %w", err)
	}

	obs.RecordAPICall("/api/v2/services/definitions", "GET", 200, float64(apiDuration), nil)
	obs.FinishSpan(querySpan)

	// Parse and filter results
	parseSpan := obs.StartSpan("catalog.parse_results")
	output, err := c.parseResponse(rawData)
	if err != nil {
		obs.LogError("Failed to parse response: " + err.Error())
		obs.FinishSpan(parseSpan)
		return fmt.Errorf("failed to parse response: %w", err)
	}
	obs.FinishSpan(parseSpan)

	// Apply filters
	filterSpan := obs.StartSpan("catalog.filter_results")
	filteredServices := c.filterServices(output.Services)
	output.FilteredCount = len(filteredServices)
	output.Services = filteredServices
	obs.FinishSpan(filterSpan)

	// Recalculate summary for filtered results
	if output.FilteredCount != output.TotalCount {
		output.Summary = c.calculateSummary(filteredServices)
	}

	// Record metrics
	obs.LogInfo(fmt.Sprintf("Found %d services (filtered to %d)", output.TotalCount, output.FilteredCount))
	obs.GetMetrics().Gauge("catalog.services.total", float64(output.TotalCount))
	obs.GetMetrics().Gauge("catalog.services.filtered", float64(output.FilteredCount))
	obs.GetMetrics().Gauge("catalog.services.with_team", float64(output.Summary.WithTeam))
	obs.GetMetrics().Gauge("catalog.services.with_docs", float64(output.Summary.WithDocs))
	obs.GetMetrics().Count("catalog.query.count", 1, "status:"+output.Status)

	// Output
	if c.json {
		jsonData, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	} else {
		c.printFormatted(output)
	}

	obs.LogInfo("Service catalog query complete")
	return nil
}

// parseResponse parses the raw API response into structured format
func (c *CatalogCommand) parseResponse(rawData []byte) (*CatalogOutput, error) {
	var apiResponse struct {
		Data []struct {
			ID         string `json:"id"`
			Type       string `json:"type"`
			Attributes struct {
				Kind        string   `json:"kind"`
				Description string   `json:"description"`
				Tier        string   `json:"tier"`
				Lifecycle   string   `json:"lifecycle"`
				Application string   `json:"application"`
				Languages   []string `json:"languages"`
				Tags        []string `json:"tags"`
				Contacts    []struct {
					Type    string `json:"type"`
					Contact string `json:"contact"`
				} `json:"contacts"`
				Links []struct {
					Type string `json:"type"`
					URL  string `json:"url"`
				} `json:"links"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(rawData, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	output := &CatalogOutput{
		Status:        "ok",
		TotalCount:    len(apiResponse.Data),
		FilteredCount: 0,
		Services:      []ServiceDefinition{},
	}

	// Parse services
	for _, item := range apiResponse.Data {
		service := ServiceDefinition{
			Name:        item.ID,
			Kind:        item.Attributes.Kind,
			Description: item.Attributes.Description,
			Tier:        item.Attributes.Tier,
			Lifecycle:   item.Attributes.Lifecycle,
			Application: item.Attributes.Application,
			Languages:   item.Attributes.Languages,
			Tags:        item.Attributes.Tags,
		}

		// Extract contacts (team and owner)
		for _, contact := range item.Attributes.Contacts {
			switch contact.Type {
			case "team":
				service.Team = contact.Contact
			case "email":
				service.Owner = contact.Contact
			}
		}

		// Extract links (repos, docs, dashboards)
		for _, link := range item.Attributes.Links {
			switch link.Type {
			case "repo":
				service.Repos = append(service.Repos, link.URL)
			case "doc":
				service.Docs = append(service.Docs, link.URL)
			case "dashboard":
				service.Dashboards = append(service.Dashboards, link.URL)
			}
		}

		output.Services = append(output.Services, service)
	}

	// Calculate summary
	output.Summary = c.calculateSummary(output.Services)

	return output, nil
}

// calculateSummary calculates summary statistics for services
func (c *CatalogCommand) calculateSummary(services []ServiceDefinition) *CatalogSummary {
	summary := &CatalogSummary{
		ByKind:    make(map[string]int),
		ByTier:    make(map[string]int),
		WithTeam:  0,
		WithDocs:  0,
		WithRepos: 0,
	}

	for _, service := range services {
		// Count by kind
		kind := service.Kind
		if kind == "" {
			kind = "unknown"
		}
		summary.ByKind[kind]++

		// Count by tier
		tier := service.Tier
		if tier == "" {
			tier = "unknown"
		}
		summary.ByTier[tier]++

		// Count services with metadata
		if service.Team != "" {
			summary.WithTeam++
		}
		if len(service.Docs) > 0 {
			summary.WithDocs++
		}
		if len(service.Repos) > 0 {
			summary.WithRepos++
		}
	}

	return summary
}

// filterServices applies search and tag filters
func (c *CatalogCommand) filterServices(services []ServiceDefinition) []ServiceDefinition {
	filtered := []ServiceDefinition{}

	for _, service := range services {
		// Apply search filter
		if c.search != "" {
			searchLower := strings.ToLower(c.search)
			nameLower := strings.ToLower(service.Name)
			descLower := strings.ToLower(service.Description)

			if !strings.Contains(nameLower, searchLower) && !strings.Contains(descLower, searchLower) {
				continue
			}
		}

		// Apply team filter
		if c.team != "" {
			teamLower := strings.ToLower(c.team)
			serviceTeamLower := strings.ToLower(service.Team)

			if !strings.Contains(serviceTeamLower, teamLower) {
				continue
			}
		}

		// Apply env filter (check tags)
		if c.env != "" {
			envTag := fmt.Sprintf("env:%s", c.env)
			found := false
			for _, tag := range service.Tags {
				if strings.EqualFold(tag, envTag) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		filtered = append(filtered, service)
	}

	return filtered
}

// printFormatted prints the catalog response in a conversational format
func (c *CatalogCommand) printFormatted(output *CatalogOutput) {
	fmt.Println("Service Catalog")
	fmt.Println()

	if output.FilteredCount == 0 {
		fmt.Println("No services found for the specified criteria")
		if c.search != "" {
			fmt.Printf("Search: %s\n", c.search)
		}
		if c.team != "" {
			fmt.Printf("Team: %s\n", c.team)
		}
		if c.env != "" {
			fmt.Printf("Environment: %s\n", c.env)
		}
		return
	}

	// Summary
	fmt.Printf("Found %d services", output.FilteredCount)
	if output.FilteredCount != output.TotalCount {
		fmt.Printf(" (filtered from %d total)", output.TotalCount)
	}
	fmt.Println()
	fmt.Println()

	// Statistics
	summary := output.Summary
	fmt.Println("Statistics:")
	fmt.Printf("  Services with team: %d (%.0f%%)\n",
		summary.WithTeam,
		float64(summary.WithTeam)/float64(output.FilteredCount)*100)
	fmt.Printf("  Services with docs: %d (%.0f%%)\n",
		summary.WithDocs,
		float64(summary.WithDocs)/float64(output.FilteredCount)*100)
	fmt.Printf("  Services with repos: %d (%.0f%%)\n",
		summary.WithRepos,
		float64(summary.WithRepos)/float64(output.FilteredCount)*100)
	fmt.Println()

	// By kind
	if len(summary.ByKind) > 0 {
		fmt.Println("By kind:")

		// Sort by count descending
		type kindCount struct {
			kind  string
			count int
		}
		var kinds []kindCount
		for kind, count := range summary.ByKind {
			kinds = append(kinds, kindCount{kind, count})
		}
		sort.Slice(kinds, func(i, j int) bool {
			return kinds[i].count > kinds[j].count
		})

		for _, k := range kinds {
			fmt.Printf("  %s: %d\n", k.kind, k.count)
		}
		fmt.Println()
	}

	// By tier
	if len(summary.ByTier) > 0 {
		fmt.Println("By tier:")

		// Sort by tier priority
		tierOrder := map[string]int{"critical": 1, "high": 2, "medium": 3, "low": 4, "unknown": 5}
		type tierCount struct {
			tier  string
			count int
		}
		var tiers []tierCount
		for tier, count := range summary.ByTier {
			tiers = append(tiers, tierCount{tier, count})
		}
		sort.Slice(tiers, func(i, j int) bool {
			orderI := tierOrder[tiers[i].tier]
			orderJ := tierOrder[tiers[j].tier]
			if orderI == 0 {
				orderI = 99
			}
			if orderJ == 0 {
				orderJ = 99
			}
			return orderI < orderJ
		})

		for _, t := range tiers {
			tierSymbol := ""
			switch t.tier {
			case "critical":
				tierSymbol = "⚠"
			case "high":
				tierSymbol = "●"
			case "medium":
				tierSymbol = "◐"
			case "low":
				tierSymbol = "○"
			default:
				tierSymbol = "?"
			}
			fmt.Printf("  %s %s: %d\n", tierSymbol, t.tier, t.count)
		}
		fmt.Println()
	}

	// Services table
	fmt.Println("Services:")
	fmt.Printf("%-40s %-15s %-10s %-30s\n", "Name", "Kind", "Tier", "Team")
	fmt.Println(strings.Repeat("-", 100))

	// Sort services by name
	sortedServices := make([]ServiceDefinition, len(output.Services))
	copy(sortedServices, output.Services)
	sort.Slice(sortedServices, func(i, j int) bool {
		return sortedServices[i].Name < sortedServices[j].Name
	})

	// Display services (limit to first 20)
	displayLimit := 20
	for i, service := range sortedServices {
		if i >= displayLimit {
			fmt.Printf("... and %d more\n", len(sortedServices)-displayLimit)
			break
		}

		name := service.Name
		if len(name) > 37 {
			name = name[:37] + "..."
		}

		kind := service.Kind
		if kind == "" {
			kind = "-"
		}
		if len(kind) > 15 {
			kind = kind[:12] + "..."
		}

		tier := service.Tier
		if tier == "" {
			tier = "-"
		}

		team := service.Team
		if team == "" {
			team = "-"
		}
		if len(team) > 27 {
			team = team[:27] + "..."
		}

		fmt.Printf("%-40s %-15s %-10s %-30s\n", name, kind, tier, team)
	}

	// Show contact info if only filtering
	if c.search != "" && output.FilteredCount <= 5 {
		fmt.Println()
		fmt.Println("Service details:")
		for _, service := range sortedServices {
			c.printServiceDetails(service)
		}
	}
}

// printServiceDetails prints detailed information about a service
func (c *CatalogCommand) printServiceDetails(service ServiceDefinition) {
	fmt.Println()
	fmt.Printf("Service: %s\n", service.Name)

	if service.Description != "" {
		fmt.Printf("  Description: %s\n", service.Description)
	}

	if service.Kind != "" {
		fmt.Printf("  Kind: %s\n", service.Kind)
	}

	if service.Tier != "" {
		fmt.Printf("  Tier: %s\n", service.Tier)
	}

	if service.Team != "" || service.Owner != "" {
		fmt.Println("  Contacts:")
		if service.Team != "" {
			fmt.Printf("    Team: %s\n", service.Team)
		}
		if service.Owner != "" {
			fmt.Printf("    Owner: %s\n", service.Owner)
		}
	}

	if len(service.Languages) > 0 {
		fmt.Printf("  Languages: %s\n", strings.Join(service.Languages, ", "))
	}

	if len(service.Repos) > 0 {
		fmt.Println("  Repositories:")
		for _, repo := range service.Repos {
			fmt.Printf("    - %s\n", repo)
		}
	}

	if len(service.Docs) > 0 {
		fmt.Println("  Documentation:")
		for _, doc := range service.Docs {
			fmt.Printf("    - %s\n", doc)
		}
	}

	if len(service.Dashboards) > 0 {
		fmt.Println("  Dashboards:")
		for _, dashboard := range service.Dashboards {
			fmt.Printf("    - %s\n", dashboard)
		}
	}

	if len(service.Tags) > 0 {
		fmt.Printf("  Tags: %s\n", strings.Join(service.Tags, ", "))
	}
}

// Help prints the help message
func (c *CatalogCommand) Help() {
	fmt.Println("Usage: dd catalog [options]")
	fmt.Println()
	fmt.Println("Query Datadog Service Catalog for service metadata and ownership")
	fmt.Println()
	fmt.Println("Options:")
	c.flags.PrintDefaults()
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  dd catalog")
	fmt.Println("  dd catalog --search api")
	fmt.Println("  dd catalog --team backend")
	fmt.Println("  dd catalog --env production")
	fmt.Println("  dd catalog --search payment --team platform")
	fmt.Println("  dd catalog --json")
}
