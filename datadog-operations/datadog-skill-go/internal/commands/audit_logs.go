package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"time"

	"github.com/datadog/skill/internal/client"
	"github.com/datadog/skill/internal/observability"
)

// AuditLogsCommand queries Datadog audit logs for compliance and security tracking
type AuditLogsCommand struct {
	flags      *flag.FlagSet
	action     string
	query      string
	from       string
	to         string
	user       string
	resource   string
	actionType string
	limit      int
	jsonOut    bool
}

// AuditEvent represents a parsed audit event from Datadog API
type AuditEvent struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Timestamp  string                 `json:"timestamp"`
	User       string                 `json:"user,omitempty"`
	UserEmail  string                 `json:"user_email,omitempty"`
	Action     string                 `json:"action"`
	Resource   string                 `json:"resource,omitempty"`
	ResourceID string                 `json:"resource_id,omitempty"`
	Status     string                 `json:"status,omitempty"`
	Message    string                 `json:"message,omitempty"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// AuditLogsResponse represents the formatted audit logs response
type AuditLogsResponse struct {
	Status      string       `json:"status"`
	TotalEvents int          `json:"total_events"`
	Events      []AuditEvent `json:"events,omitempty"`
}

// NewAuditLogsCommand creates a new audit logs command
func NewAuditLogsCommand() *AuditLogsCommand {
	cmd := &AuditLogsCommand{
		flags: flag.NewFlagSet("audit-logs", flag.ExitOnError),
	}

	cmd.flags.StringVar(&cmd.action, "action", "search", "Action: search, aggregate")
	cmd.flags.StringVar(&cmd.query, "query", "", "Search query (e.g., '@action:user.login @status:error')")
	cmd.flags.StringVar(&cmd.from, "from", "", "Start time (RFC3339 or relative like '1h')")
	cmd.flags.StringVar(&cmd.to, "to", "", "End time (RFC3339, defaults to now)")
	cmd.flags.StringVar(&cmd.user, "user", "", "Filter by user email or ID")
	cmd.flags.StringVar(&cmd.resource, "resource", "", "Filter by resource type (e.g., 'monitor', 'dashboard')")
	cmd.flags.StringVar(&cmd.actionType, "action-type", "", "Filter by action type (e.g., 'created', 'deleted', 'modified')")
	cmd.flags.IntVar(&cmd.limit, "limit", 100, "Maximum number of events to return (max 1000)")
	cmd.flags.BoolVar(&cmd.jsonOut, "json", false, "Output as JSON")

	return cmd
}

// Name returns the command name
func (c *AuditLogsCommand) Name() string {
	return "audit-logs"
}

// Description returns the command description
func (c *AuditLogsCommand) Description() string {
	return "Query Datadog audit logs for compliance and security tracking"
}

// Run executes the audit logs command
func (c *AuditLogsCommand) Run(args []string) error {
	// Initialize observability
	obs, err := observability.Init("query-audit-logs", "production")
	if err != nil {
		return fmt.Errorf("failed to init observability: %w", err)
	}
	defer obs.Shutdown(0)

	// Parse flags
	if err := c.flags.Parse(args); err != nil {
		return err
	}

	// Start tracing
	span := obs.StartSpan("audit_logs.query")
	defer obs.FinishSpan(span)

	obs.LogInfo(fmt.Sprintf("Querying audit logs with action: %s", c.action))

	// Initialize Datadog client
	ddClient, err := client.NewClient()
	if err != nil {
		return fmt.Errorf("failed to initialize Datadog client: %w", err)
	}

	// Execute action
	switch c.action {
	case "search":
		return c.searchAuditLogs(ddClient, obs)
	case "aggregate":
		return c.aggregateAuditLogs(ddClient, obs)
	default:
		return fmt.Errorf("unknown action: %s (use: search, aggregate)", c.action)
	}
}

// searchAuditLogs searches audit log events
func (c *AuditLogsCommand) searchAuditLogs(ddClient *client.Client, obs *observability.Observability) error {
	obs.LogInfo("Searching audit logs")

	// Parse time range
	fromTime, toTime, err := c.parseTimeRange()
	if err != nil {
		return err
	}

	// Build query
	query := c.buildQuery()

	// Search audit logs
	data, err := ddClient.SearchAuditLogs(query, fromTime, toTime, c.limit)
	if err != nil {
		return fmt.Errorf("failed to search audit logs: %w", err)
	}

	// Parse and display results
	return c.parseAndDisplayEvents(data, obs)
}

// aggregateAuditLogs aggregates audit log events
func (c *AuditLogsCommand) aggregateAuditLogs(ddClient *client.Client, obs *observability.Observability) error {
	obs.LogInfo("Aggregating audit logs")

	// Parse time range
	fromTime, toTime, err := c.parseTimeRange()
	if err != nil {
		return err
	}

	// Build query
	query := c.buildQuery()

	// Aggregate audit logs
	data, err := ddClient.AggregateAuditLogs(query, fromTime, toTime)
	if err != nil {
		return fmt.Errorf("failed to aggregate audit logs: %w", err)
	}

	// Parse and display aggregation results
	return c.parseAndDisplayAggregation(data, obs)
}

// parseTimeRange parses the from/to time range
func (c *AuditLogsCommand) parseTimeRange() (time.Time, time.Time, error) {
	// Default to last hour
	toTime := time.Now()
	fromTime := toTime.Add(-1 * time.Hour)

	// Parse 'to' time if provided
	if c.to != "" {
		t, err := time.Parse(time.RFC3339, c.to)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid 'to' time format (use RFC3339): %w", err)
		}
		toTime = t
	}

	// Parse 'from' time if provided
	if c.from != "" {
		// Try RFC3339 first
		t, err := time.Parse(time.RFC3339, c.from)
		if err != nil {
			// Try relative time (e.g., "1h", "30m", "2d")
			duration, err := parseRelativeTime(c.from)
			if err != nil {
				return time.Time{}, time.Time{}, fmt.Errorf("invalid 'from' time format (use RFC3339 or relative like '1h'): %w", err)
			}
			fromTime = toTime.Add(-duration)
		} else {
			fromTime = t
		}
	}

	return fromTime, toTime, nil
}

// parseRelativeTime parses relative time strings like "1h", "30m", "2d"
func parseRelativeTime(s string) (time.Duration, error) {
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid relative time format")
	}

	unit := s[len(s)-1]
	valueStr := s[:len(s)-1]

	var value int
	_, err := fmt.Sscanf(valueStr, "%d", &value)
	if err != nil {
		return 0, err
	}

	switch unit {
	case 'm':
		return time.Duration(value) * time.Minute, nil
	case 'h':
		return time.Duration(value) * time.Hour, nil
	case 'd':
		return time.Duration(value) * 24 * time.Hour, nil
	case 'w':
		return time.Duration(value) * 7 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unknown time unit '%c' (use m, h, d, w)", unit)
	}
}

// buildQuery builds the audit log query string
func (c *AuditLogsCommand) buildQuery() string {
	query := c.query

	// Add user filter
	if c.user != "" {
		if query != "" {
			query += " "
		}
		query += fmt.Sprintf("@usr.email:%s OR @usr.id:%s", c.user, c.user)
	}

	// Add resource filter
	if c.resource != "" {
		if query != "" {
			query += " "
		}
		query += fmt.Sprintf("@evt.name:%s", c.resource)
	}

	// Add action type filter
	if c.actionType != "" {
		if query != "" {
			query += " "
		}
		query += fmt.Sprintf("@evt.outcome:%s", c.actionType)
	}

	// If no query specified, use wildcard
	if query == "" {
		query = "*"
	}

	return query
}

// parseAndDisplayEvents parses and displays audit events
func (c *AuditLogsCommand) parseAndDisplayEvents(data []byte, obs *observability.Observability) error {
	// Parse JSON response
	var apiResponse struct {
		Data []struct {
			ID         string `json:"id"`
			Type       string `json:"type"`
			Attributes struct {
				Timestamp  string                 `json:"timestamp"`
				Message    string                 `json:"message"`
				Attributes map[string]interface{} `json:"attributes"`
			} `json:"attributes"`
		} `json:"data"`
		Meta struct {
			Page struct {
				After string `json:"after"`
			} `json:"page"`
		} `json:"meta"`
	}

	if err := json.Unmarshal(data, &apiResponse); err != nil {
		return fmt.Errorf("failed to parse API response: %w", err)
	}

	// Build response
	response := AuditLogsResponse{
		Status:      "success",
		TotalEvents: len(apiResponse.Data),
		Events:      make([]AuditEvent, 0),
	}

	// Parse events
	for _, item := range apiResponse.Data {
		event := AuditEvent{
			ID:         item.ID,
			Type:       item.Type,
			Timestamp:  item.Attributes.Timestamp,
			Message:    item.Attributes.Message,
			Attributes: item.Attributes.Attributes,
		}

		// Extract common fields from attributes
		if attrs, ok := item.Attributes.Attributes["attributes"].(map[string]interface{}); ok {
			if usr, ok := attrs["usr"].(map[string]interface{}); ok {
				if email, ok := usr["email"].(string); ok {
					event.UserEmail = email
				}
				if id, ok := usr["id"].(string); ok {
					event.User = id
				}
			}
			if evt, ok := attrs["evt"].(map[string]interface{}); ok {
				if name, ok := evt["name"].(string); ok {
					event.Action = name
				}
				if outcome, ok := evt["outcome"].(string); ok {
					event.Status = outcome
				}
			}
			if resource, ok := attrs["resource"].(map[string]interface{}); ok {
				if resType, ok := resource["type"].(string); ok {
					event.Resource = resType
				}
				if resID, ok := resource["id"].(string); ok {
					event.ResourceID = resID
				}
			}
		}

		response.Events = append(response.Events, event)
	}

	// Output results
	if c.jsonOut {
		output, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(output))
	} else {
		c.displayFormattedEvents(response)
	}

	return nil
}

// parseAndDisplayAggregation parses and displays aggregation results
func (c *AuditLogsCommand) parseAndDisplayAggregation(data []byte, obs *observability.Observability) error {
	// Parse JSON response
	var apiResponse struct {
		Data struct {
			Buckets []struct {
				By        map[string]interface{} `json:"by"`
				Computes  map[string]interface{} `json:"computes"`
				Total     int                    `json:"total"`
			} `json:"buckets"`
		} `json:"data"`
		Meta struct {
			Status string `json:"status"`
		} `json:"meta"`
	}

	if err := json.Unmarshal(data, &apiResponse); err != nil {
		return fmt.Errorf("failed to parse API response: %w", err)
	}

	// Display aggregation results
	if c.jsonOut {
		output, err := json.MarshalIndent(apiResponse, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(output))
	} else {
		c.displayFormattedAggregation(apiResponse)
	}

	return nil
}

// displayFormattedEvents displays formatted events output
func (c *AuditLogsCommand) displayFormattedEvents(response AuditLogsResponse) {
	fmt.Println("Audit Logs")
	fmt.Println("==========")
	fmt.Printf("Total events: %d\n", response.TotalEvents)

	if len(response.Events) > 0 {
		fmt.Println("\nEvents:")
		for i, event := range response.Events {
			if i >= 50 {
				fmt.Printf("\n... and %d more events (use --json for full list)\n", len(response.Events)-50)
				break
			}

			// Format timestamp
			timestamp := event.Timestamp
			if t, err := time.Parse(time.RFC3339, event.Timestamp); err == nil {
				timestamp = t.Format("2006-01-02 15:04:05 MST")
			}

			fmt.Printf("\n[%s] %s\n", timestamp, event.Action)
			if event.UserEmail != "" {
				fmt.Printf("  User: %s\n", event.UserEmail)
			} else if event.User != "" {
				fmt.Printf("  User ID: %s\n", event.User)
			}
			if event.Resource != "" {
				fmt.Printf("  Resource: %s", event.Resource)
				if event.ResourceID != "" {
					fmt.Printf(" (%s)", event.ResourceID)
				}
				fmt.Println()
			}
			if event.Status != "" {
				fmt.Printf("  Status: %s\n", event.Status)
			}
			if event.Message != "" && event.Message != event.Action {
				// Truncate long messages
				msg := event.Message
				if len(msg) > 100 {
					msg = msg[:97] + "..."
				}
				fmt.Printf("  Message: %s\n", msg)
			}
		}
	} else {
		fmt.Println("\nNo audit events found for the specified criteria.")
	}

	fmt.Println("\n💡 Tip: Use --limit to increase results or narrow your time range with --from and --to")
}

// displayFormattedAggregation displays formatted aggregation output
func (c *AuditLogsCommand) displayFormattedAggregation(apiResponse interface{}) {
	fmt.Println("Audit Logs Aggregation")
	fmt.Println("======================")

	// Display raw JSON for now (aggregation structure varies by query)
	output, _ := json.MarshalIndent(apiResponse, "", "  ")
	fmt.Println(string(output))

	fmt.Println("\n💡 Tip: Use --json flag for programmatic access to aggregation results")
}

// Help displays help information
func (c *AuditLogsCommand) Help() {
	help := `dd audit-logs - Query Datadog Audit Logs

DESCRIPTION:
  Query Datadog audit logs for compliance and security tracking. Audit logs
  capture all changes made to your Datadog organization, including user actions,
  configuration changes, and access patterns. Essential for SOC2, HIPAA, PCI-DSS,
  and other compliance requirements.

USAGE:
  dd audit-logs --action <action> [options]

ACTIONS:
  search           Search audit log events
  aggregate        Aggregate audit log events for analysis

EXAMPLES:
  # Search last hour of audit logs
  dd audit-logs --action search

  # Search last 24 hours
  dd audit-logs --action search --from 24h

  # Search specific time range
  dd audit-logs --action search \
    --from 2026-01-20T00:00:00Z \
    --to 2026-01-21T00:00:00Z

  # Find user login failures
  dd audit-logs --action search \
    --query "@action:user.login @status:error" \
    --from 7d

  # Find all monitor changes
  dd audit-logs --action search \
    --resource monitor \
    --from 24h

  # Find changes by specific user
  dd audit-logs --action search \
    --user "user@example.com" \
    --from 7d

  # Find all deletions
  dd audit-logs --action search \
    --action-type deleted \
    --from 24h

  # Aggregate events by action type
  dd audit-logs --action aggregate \
    --query "@evt.name:monitor.*" \
    --from 7d

  # Get more results
  dd audit-logs --action search --limit 500 --from 24h

  # Get JSON output
  dd audit-logs --action search --json

OPTIONS:
  --action          Action to perform (search, aggregate)
  --query           Search query (Datadog query syntax)
  --from            Start time (RFC3339 or relative like '1h', '7d')
  --to              End time (RFC3339, defaults to now)
  --user            Filter by user email or ID
  --resource        Filter by resource type (monitor, dashboard, etc.)
  --action-type     Filter by action (created, deleted, modified)
  --limit           Maximum events to return (default: 100, max: 1000)
  --json            Output as JSON

COMMON QUERIES:
  # Authentication events
  @action:user.login
  @action:user.logout
  @action:user.login @status:error

  # Configuration changes
  @evt.name:monitor.*
  @evt.name:dashboard.*
  @evt.name:integration.*

  # User management
  @evt.name:user.created
  @evt.name:user.deleted
  @evt.name:role.modified

  # API key management
  @evt.name:api_key.created
  @evt.name:api_key.deleted
  @evt.name:application_key.created

  # Security events
  @evt.outcome:error
  @evt.category:authentication
  @evt.category:authorization

TIME FORMATS:
  Absolute:  2026-01-20T15:04:05Z (RFC3339)
  Relative:  1h (1 hour ago)
             30m (30 minutes ago)
             7d (7 days ago)
             2w (2 weeks ago)

COMPLIANCE USE CASES:
  - SOC2: Track all configuration changes and access patterns
  - HIPAA: Audit access to sensitive data and systems
  - PCI-DSS: Monitor administrative actions and privilege changes
  - GDPR: Track data access and deletion requests
  - General: Security incident investigation and forensics

FIELDS AVAILABLE:
  @usr.email         User email address
  @usr.id            User ID
  @usr.name          User name
  @evt.name          Event name (action performed)
  @evt.outcome       Event outcome (success, error)
  @evt.category      Event category
  @resource.type     Resource type (monitor, dashboard, etc.)
  @resource.id       Resource ID
  @http.method       HTTP method for API calls
  @http.status_code  HTTP status code

NOTES:
  - Audit logs retention depends on your Datadog plan (typically 15-90 days)
  - Use time ranges to limit results and improve query performance
  - Aggregate action useful for compliance reporting and trend analysis
  - All times in UTC
  - Rate limits apply to audit log queries
  - Export results to long-term storage for extended retention

For more information: https://docs.datadoghq.com/api/latest/audit/
`
	fmt.Println(help)
}
