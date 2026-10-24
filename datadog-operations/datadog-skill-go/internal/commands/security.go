package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/datadog/skill/internal/client"
	"github.com/datadog/skill/internal/context"
	"github.com/datadog/skill/internal/observability"
)

// SecurityCommand queries Datadog Security Monitoring Signals
type SecurityCommand struct {
	flags    *flag.FlagSet
	service  string
	duration string
	severity string
	limit    int
	json     bool
}

// SecuritySignal represents a parsed security signal from Datadog API
type SecuritySignal struct {
	ID         string                 `json:"id"`
	Timestamp  string                 `json:"timestamp"`
	Severity   string                 `json:"severity"`
	RuleName   string                 `json:"rule_name,omitempty"`
	Service    string                 `json:"service,omitempty"`
	Tags       []string               `json:"tags,omitempty"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// SecurityResponse represents the formatted security signals response
type SecurityResponse struct {
	Status           string                 `json:"status"`
	TotalSignals     int                    `json:"total_signals"`
	Duration         string                 `json:"duration"`
	Service          string                 `json:"service"`
	Summary          map[string]int         `json:"summary"`
	AttackTypes      map[string]int         `json:"attack_types"`
	AffectedServices map[string]int         `json:"affected_services"`
	RecentSignals    []SecuritySignal       `json:"recent_signals,omitempty"`
	RawData          map[string]interface{} `json:"raw_data,omitempty"`
}

// NewSecurityCommand creates a new security command
func NewSecurityCommand() *SecurityCommand {
	cmd := &SecurityCommand{
		flags: flag.NewFlagSet("security", flag.ExitOnError),
	}

	cmd.flags.StringVar(&cmd.service, "service", "", "Filter by service (auto-detect if not provided)")
	cmd.flags.StringVar(&cmd.duration, "duration", "24h", "Time range: 1h, 24h, 7d, 30d")
	cmd.flags.StringVar(&cmd.severity, "severity", "", "Filter by severity: critical, high, medium, low, info")
	cmd.flags.IntVar(&cmd.limit, "limit", 20, "Number of results (default: 20)")
	cmd.flags.BoolVar(&cmd.json, "json", false, "Output as JSON")

	return cmd
}

// Name returns the command name
func (c *SecurityCommand) Name() string {
	return "security"
}

// Description returns the command description
func (c *SecurityCommand) Description() string {
	return "Query Datadog Security Monitoring Signals for threats and attacks"
}

// Run executes the security command
func (c *SecurityCommand) Run(args []string) error {
	// Initialize observability
	obs, err := observability.Init("query-security-signals", "production")
	if err != nil {
		return fmt.Errorf("failed to init observability: %w", err)
	}
	defer obs.Shutdown(0)

	// Parse flags
	if err := c.flags.Parse(args); err != nil {
		return err
	}

	// Start tracing
	span := obs.StartSpan("security.query")
	defer obs.FinishSpan(span)

	obs.LogInfo("Starting security signals query")

	// Auto-detect service if needed
	service := c.service
	if service == "" {
		detectSpan := obs.StartSpan("security.detect_context")
		obs.LogInfo("Auto-detecting service context")

		ctx, err := context.DetectContext(".")
		if err != nil {
			obs.LogWarning("Failed to detect context, querying all services")
		} else {
			service = ctx.ServiceName
			obs.LogInfo(fmt.Sprintf("Auto-detected service: %s", service))
			obs.GetTracer().SetTag(detectSpan, "service.name", service)
			obs.GetTracer().SetTag(detectSpan, "detection.method", ctx.DetectionMethod)
		}
		obs.FinishSpan(detectSpan)
	}

	// Parse duration
	parseSpan := obs.StartSpan("security.parse_duration")
	fromTime, toTime, err := c.parseDuration()
	if err != nil {
		obs.LogError("Failed to parse duration: " + err.Error())
		obs.FinishSpan(parseSpan)
		return fmt.Errorf("failed to parse duration: %w", err)
	}
	obs.GetTracer().SetTag(parseSpan, "from", fromTime.Format(time.RFC3339))
	obs.GetTracer().SetTag(parseSpan, "to", toTime.Format(time.RFC3339))
	obs.FinishSpan(parseSpan)

	// Create Datadog client
	clientSpan := obs.StartSpan("security.create_client")
	ddClient, err := client.NewClient()
	if err != nil {
		obs.LogError("Failed to create Datadog client: " + err.Error())
		obs.FinishSpan(clientSpan)
		return fmt.Errorf("failed to create Datadog client: %w", err)
	}
	obs.FinishSpan(clientSpan)

	// Query security signals
	querySpan := obs.StartSpan("security.api_query")
	obs.GetTracer().SetTag(querySpan, "service", service)
	obs.GetTracer().SetTag(querySpan, "duration", c.duration)
	obs.GetTracer().SetTag(querySpan, "severity", c.severity)

	start := time.Now()
	rawData, err := ddClient.GetSecuritySignals(fromTime, toTime, service)
	apiDuration := time.Since(start).Milliseconds()

	if err != nil {
		obs.LogError("API call failed: " + err.Error())
		obs.RecordAPICall("/api/v2/security_monitoring/signals/search", "POST", 500, float64(apiDuration), err)
		obs.GetTracer().SetError(querySpan, err)
		obs.FinishSpan(querySpan)
		return fmt.Errorf("failed to query security signals: %w", err)
	}

	obs.RecordAPICall("/api/v2/security_monitoring/signals/search", "POST", 200, float64(apiDuration), nil)
	obs.FinishSpan(querySpan)

	// Parse and analyze results
	analyzeSpan := obs.StartSpan("security.parse_results")
	response, err := c.parseResponse(rawData, service)
	if err != nil {
		obs.LogError("Failed to parse response: " + err.Error())
		obs.FinishSpan(analyzeSpan)
		return fmt.Errorf("failed to parse response: %w", err)
	}
	obs.FinishSpan(analyzeSpan)

	// Record metrics
	obs.LogInfo(fmt.Sprintf("Found %d security signals", response.TotalSignals))
	obs.GetMetrics().Gauge("security.signals.total", float64(response.TotalSignals))
	obs.GetMetrics().Gauge("security.signals.critical", float64(response.Summary["critical"]))
	obs.GetMetrics().Gauge("security.signals.high", float64(response.Summary["high"]))
	obs.GetMetrics().Gauge("security.signals.medium", float64(response.Summary["medium"]))
	obs.GetMetrics().Gauge("security.signals.low", float64(response.Summary["low"]))
	obs.GetMetrics().Count("security.query.count", 1,
		"service:"+service,
		"status:"+response.Status,
	)

	// Output
	if c.json {
		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	} else {
		c.printFormatted(response)
	}

	obs.LogInfo("Security signals query complete")
	return nil
}

// parseDuration parses the duration string into time range
func (c *SecurityCommand) parseDuration() (time.Time, time.Time, error) {
	toTime := time.Now()
	var duration time.Duration

	if strings.HasSuffix(c.duration, "h") {
		hours := strings.TrimSuffix(c.duration, "h")
		var h int
		if _, err := fmt.Sscanf(hours, "%d", &h); err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid duration: %s", c.duration)
		}
		duration = time.Duration(h) * time.Hour
	} else if strings.HasSuffix(c.duration, "d") {
		days := strings.TrimSuffix(c.duration, "d")
		var d int
		if _, err := fmt.Sscanf(days, "%d", &d); err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid duration: %s", c.duration)
		}
		duration = time.Duration(d) * 24 * time.Hour
	} else {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid duration format: %s. Use format like '1h', '24h', '7d', '30d'", c.duration)
	}

	fromTime := toTime.Add(-duration)
	return fromTime, toTime, nil
}

// parseResponse parses the raw API response into structured format
func (c *SecurityCommand) parseResponse(rawData []byte, service string) (*SecurityResponse, error) {
	var apiResponse map[string]interface{}
	if err := json.Unmarshal(rawData, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	response := &SecurityResponse{
		Duration:         c.duration,
		Service:          service,
		Summary:          make(map[string]int),
		AttackTypes:      make(map[string]int),
		AffectedServices: make(map[string]int),
		RecentSignals:    []SecuritySignal{},
		RawData:          apiResponse,
	}

	// Extract signals from API response
	signals, ok := apiResponse["data"].([]interface{})
	if !ok {
		response.TotalSignals = 0
		response.Status = "ok"
		// Initialize summary with zeros
		response.Summary["critical"] = 0
		response.Summary["high"] = 0
		response.Summary["medium"] = 0
		response.Summary["low"] = 0
		response.Summary["info"] = 0
		return response, nil
	}

	response.TotalSignals = len(signals)

	if response.TotalSignals == 0 {
		response.Status = "ok"
		// Initialize summary with zeros
		response.Summary["critical"] = 0
		response.Summary["high"] = 0
		response.Summary["medium"] = 0
		response.Summary["low"] = 0
		response.Summary["info"] = 0
		return response, nil
	}

	// Parse signals and collect statistics
	for i, signalItem := range signals {
		signalMap, ok := signalItem.(map[string]interface{})
		if !ok {
			continue
		}

		attrs, ok := signalMap["attributes"].(map[string]interface{})
		if !ok {
			continue
		}

		severity := getSecurityString(attrs, "severity", "unknown")
		timestamp := getSecurityString(attrs, "timestamp", "")
		signalID := getSecurityString(signalMap, "id", "")

		// Count by severity
		response.Summary[severity]++

		// Extract tags
		tags := []string{}
		if tagsArr, ok := attrs["tags"].([]interface{}); ok {
			for _, tag := range tagsArr {
				if tagStr, ok := tag.(string); ok {
					tags = append(tags, tagStr)

					// Count attack types and services
					if strings.HasPrefix(tagStr, "attack_type:") || strings.HasPrefix(tagStr, "rule_name:") {
						response.AttackTypes[tagStr]++
					}
					if strings.HasPrefix(tagStr, "service:") {
						response.AffectedServices[tagStr]++
					}
				}
			}
		}

		// Extract rule name and service from tags
		ruleName := extractTagValue(tags, "rule_name:")
		serviceTag := extractTagValue(tags, "service:")

		// Filter by severity if specified
		if c.severity != "" && severity != c.severity {
			continue
		}

		// Keep recent signals for detailed output (limit to configured limit)
		if i < c.limit {
			signal := SecuritySignal{
				ID:         signalID,
				Timestamp:  timestamp,
				Severity:   severity,
				RuleName:   ruleName,
				Service:    serviceTag,
				Tags:       tags,
				Attributes: attrs,
			}
			response.RecentSignals = append(response.RecentSignals, signal)
		}
	}

	// Determine overall status
	criticalCount := response.Summary["critical"]
	highCount := response.Summary["high"]

	if criticalCount > 0 {
		response.Status = "critical"
	} else if highCount > 0 {
		response.Status = "warning"
	} else {
		response.Status = "ok"
	}

	return response, nil
}

// extractTagValue extracts value from tag with given prefix
func extractTagValue(tags []string, prefix string) string {
	for _, tag := range tags {
		if strings.HasPrefix(tag, prefix) {
			return strings.TrimPrefix(tag, prefix)
		}
	}
	return ""
}

// printFormatted prints the security response in a conversational format
func (c *SecurityCommand) printFormatted(response *SecurityResponse) {
	fmt.Println("Security Signals Analysis")
	fmt.Printf("Duration: %s\n", response.Duration)
	if response.Service != "" {
		fmt.Printf("Service: %s\n", response.Service)
	}
	fmt.Println()
	fmt.Printf("Found %d security signals\n", response.TotalSignals)

	if response.TotalSignals == 0 {
		fmt.Println()
		fmt.Println("No security signals found for the specified criteria")
		return
	}

	fmt.Println()
	fmt.Println("Severity breakdown:")
	fmt.Printf("  Critical: %d\n", response.Summary["critical"])
	fmt.Printf("  High: %d\n", response.Summary["high"])
	fmt.Printf("  Medium: %d\n", response.Summary["medium"])
	fmt.Printf("  Low: %d\n", response.Summary["low"])
	fmt.Printf("  Info: %d\n", response.Summary["info"])

	criticalCount := response.Summary["critical"]
	highCount := response.Summary["high"]

	if criticalCount > 0 || highCount > 0 {
		fmt.Println()
		fmt.Printf("ALERT: %d critical and %d high-severity security signals detected\n",
			criticalCount, highCount)
	}

	if len(response.AttackTypes) > 0 {
		fmt.Println()
		fmt.Println("Top attack types:")

		// Sort attack types by count
		type attackType struct {
			name  string
			count int
		}
		var attacks []attackType
		for name, count := range response.AttackTypes {
			attacks = append(attacks, attackType{name, count})
		}
		sort.Slice(attacks, func(i, j int) bool {
			return attacks[i].count > attacks[j].count
		})

		for i, attack := range attacks {
			if i >= 5 {
				break
			}
			fmt.Printf("  %s: %d\n", attack.name, attack.count)
		}
	}

	if len(response.AffectedServices) > 0 {
		fmt.Println()
		fmt.Println("Affected services:")

		// Sort services by count
		type serviceCount struct {
			name  string
			count int
		}
		var services []serviceCount
		for name, count := range response.AffectedServices {
			services = append(services, serviceCount{name, count})
		}
		sort.Slice(services, func(i, j int) bool {
			return services[i].count > services[j].count
		})

		for i, svc := range services {
			if i >= 5 {
				break
			}
			fmt.Printf("  %s: %d\n", svc.name, svc.count)
		}
	}

	// Show recent signals if available
	if len(response.RecentSignals) > 0 && len(response.RecentSignals) <= 5 {
		fmt.Println()
		fmt.Printf("Recent signals (%d):\n", len(response.RecentSignals))
		for i, signal := range response.RecentSignals {
			if i >= 5 {
				break
			}
			fmt.Printf("  %d. [%s] %s\n", i+1, strings.ToUpper(signal.Severity), signal.RuleName)
			if signal.Service != "" {
				fmt.Printf("     Service: %s\n", signal.Service)
			}
			if signal.Timestamp != "" {
				fmt.Printf("     Time: %s\n", signal.Timestamp)
			}
		}
	}
}

// Help prints the help message
func (c *SecurityCommand) Help() {
	fmt.Println("Usage: dd security [options]")
	fmt.Println()
	fmt.Println("Query Datadog Security Monitoring Signals for threats and attacks")
	fmt.Println()
	fmt.Println("Options:")
	c.flags.PrintDefaults()
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  dd security")
	fmt.Println("  dd security --service my-api --duration 1h")
	fmt.Println("  dd security --severity critical --duration 7d")
	fmt.Println("  dd security --json")
	fmt.Println("  dd security --duration 30d --limit 50")
}

// Helper functions

func getSecurityString(m map[string]interface{}, key, defaultVal string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultVal
}
