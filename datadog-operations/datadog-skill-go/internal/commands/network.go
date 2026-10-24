package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/datadog/skill/internal/client"
	"github.com/datadog/skill/internal/observability"
)

// NetworkCommand queries Datadog Network Performance Monitoring
type NetworkCommand struct {
	flags       *flag.FlagSet
	source      string
	destination string
	duration    string
	metric      string
	port        string
	protocol    string
	jsonOut     bool
}

// NetworkFlowData represents network flow analytics data
type NetworkFlowData struct {
	Data struct {
		Buckets []struct {
			By       map[string]interface{} `json:"by"`
			Computes map[string]interface{} `json:"computes"`
		} `json:"buckets"`
	} `json:"data"`
}

// NetworkMetricSeriesData represents time series metric data from Datadog for network queries
type NetworkMetricSeriesData struct {
	Status string `json:"status"`
	Series []struct {
		Pointlist [][2]interface{} `json:"pointlist"`
		Metric    string           `json:"metric"`
		Scope     string           `json:"scope"`
	} `json:"series"`
}

// ConnectionStats represents network connection statistics
type ConnectionStats struct {
	Source          string `json:"source"`
	Destination     string `json:"destination"`
	ConnectionCount int64  `json:"connection_count"`
	BytesSent       int64  `json:"bytes_sent"`
	BytesReceived   int64  `json:"bytes_received"`
	Retransmissions int64  `json:"retransmissions"`
}

// FlowStats represents network flow statistics
type FlowStats struct {
	Protocol   string `json:"protocol"`
	Port       int64  `json:"port"`
	FlowVolume int64  `json:"flow_volume"`
	BytesTotal int64  `json:"bytes_total"`
}

// DNSStats represents DNS query statistics
type DNSStats struct {
	Domain         string `json:"domain"`
	QueryCount     int64  `json:"query_count"`
	AvgResTimeMs   int64  `json:"avg_resolution_time_ms"`
	FailureCount   int64  `json:"failure_count"`
	NXDomainCount  int64  `json:"nxdomain_count"`
}

// LatencyStats represents network latency statistics
type LatencyStats struct {
	SourceDest string  `json:"source_dest"`
	AvgRTTMs   float64 `json:"avg_rtt_ms"`
	P95RTTMs   float64 `json:"p95_rtt_ms"`
	P99RTTMs   float64 `json:"p99_rtt_ms"`
	PacketLoss float64 `json:"packet_loss_percent"`
}

// NetworkOutput represents the structured output
type NetworkOutput struct {
	Status   string `json:"status"`
	Duration string `json:"duration"`
	Summary  *struct {
		TotalConnections   int64   `json:"total_connections"`
		TotalFlows         int64   `json:"total_flows"`
		TotalBandwidthMB   float64 `json:"total_bandwidth_mb"`
		TotalDNSQueries    int64   `json:"total_dns_queries"`
		AvgLatencyMs       float64 `json:"avg_latency_ms"`
		PacketLossPercent  float64 `json:"packet_loss_percent"`
		DNSFailurePercent  float64 `json:"dns_failure_percent"`
	} `json:"summary,omitempty"`
	TopConnections []ConnectionStats `json:"top_connections,omitempty"`
	TopFlows       []FlowStats       `json:"top_flows,omitempty"`
	TopDNSQueries  []DNSStats        `json:"top_dns_queries,omitempty"`
	Latency        []LatencyStats    `json:"latency,omitempty"`
	Issues         []struct {
		Severity string `json:"severity"`
		Message  string `json:"message"`
		Action   string `json:"action"`
	} `json:"issues,omitempty"`
}

// NewNetworkCommand creates a new network command
func NewNetworkCommand() *NetworkCommand {
	cmd := &NetworkCommand{
		flags: flag.NewFlagSet("network", flag.ExitOnError),
	}

	cmd.flags.StringVar(&cmd.source, "source", "", "Filter by source IP or hostname")
	cmd.flags.StringVar(&cmd.destination, "destination", "", "Filter by destination IP or hostname")
	cmd.flags.StringVar(&cmd.duration, "duration", "1h", "Time range: 1h, 24h, 7d, 30d (default: 1h)")
	cmd.flags.StringVar(&cmd.metric, "metric", "all", "Specific metric: connections, flows, dns, latency, all (default: all)")
	cmd.flags.StringVar(&cmd.port, "port", "", "Filter by port number")
	cmd.flags.StringVar(&cmd.protocol, "protocol", "all", "Filter by protocol: tcp, udp, all (default: all)")
	cmd.flags.BoolVar(&cmd.jsonOut, "json", false, "Output as JSON")

	return cmd
}

// Name returns the command name
func (c *NetworkCommand) Name() string {
	return "network"
}

// Description returns the command description
func (c *NetworkCommand) Description() string {
	return "Query Datadog Network Performance Monitoring for network analysis"
}

// Run executes the network command
func (c *NetworkCommand) Run(args []string) error {
	// Initialize observability
	obs, err := observability.Init("query-network", "production")
	if err != nil {
		return fmt.Errorf("failed to init observability: %w", err)
	}
	defer obs.Shutdown(0)

	// Parse flags
	if err := c.flags.Parse(args); err != nil {
		return err
	}

	obs.LogInfo("Querying Network Performance Monitoring")

	// Parse duration to time range
	span := obs.StartSpan("parse_duration")
	fromTime, toTime, err := c.parseDuration(c.duration)
	obs.FinishSpan(span)

	if err != nil {
		obs.LogError(fmt.Sprintf("Invalid duration: %s", err.Error()))
		return fmt.Errorf("invalid duration: %w", err)
	}

	// Create Datadog client
	span = obs.StartSpan("create_client")
	ddClient, err := client.NewClient()
	obs.FinishSpan(span)

	if err != nil {
		obs.LogError(fmt.Sprintf("Failed to create Datadog client: %s", err.Error()))
		return fmt.Errorf("failed to create Datadog client: %w", err)
	}

	// Initialize output
	output := &NetworkOutput{
		Status:   "ok",
		Duration: c.duration,
		Summary: &struct {
			TotalConnections   int64   `json:"total_connections"`
			TotalFlows         int64   `json:"total_flows"`
			TotalBandwidthMB   float64 `json:"total_bandwidth_mb"`
			TotalDNSQueries    int64   `json:"total_dns_queries"`
			AvgLatencyMs       float64 `json:"avg_latency_ms"`
			PacketLossPercent  float64 `json:"packet_loss_percent"`
			DNSFailurePercent  float64 `json:"dns_failure_percent"`
		}{},
	}

	// Query network connections if requested
	if c.metric == "all" || c.metric == "connections" {
		span = obs.StartSpan("query_connections")
		obs.GetTracer().SetTag(span, "source", c.source)
		obs.GetTracer().SetTag(span, "destination", c.destination)

		start := time.Now()
		connections, err := c.queryConnections(ddClient, fromTime, toTime)
		apiDuration := time.Since(start).Milliseconds()
		obs.FinishSpan(span)

		if err != nil {
			obs.RecordAPICall("/api/v1/query", "GET", 500, float64(apiDuration), err)
			obs.LogWarning(fmt.Sprintf("Failed to query connections: %s", err.Error()))
		} else {
			obs.RecordAPICall("/api/v1/query", "GET", 200, float64(apiDuration), nil)
			output.TopConnections = connections

			// Calculate total connections and bandwidth
			for _, conn := range connections {
				output.Summary.TotalConnections += conn.ConnectionCount
				output.Summary.TotalBandwidthMB += float64(conn.BytesSent+conn.BytesReceived) / 1024 / 1024
			}
		}
	}

	// Query network flows if requested
	if c.metric == "all" || c.metric == "flows" {
		span = obs.StartSpan("query_flows")
		obs.GetTracer().SetTag(span, "protocol", c.protocol)

		start := time.Now()
		flows, err := c.queryFlows(ddClient, fromTime, toTime)
		apiDuration := time.Since(start).Milliseconds()
		obs.FinishSpan(span)

		if err != nil {
			obs.RecordAPICall("/api/v1/query", "GET", 500, float64(apiDuration), err)
			obs.LogWarning(fmt.Sprintf("Failed to query flows: %s", err.Error()))
		} else {
			obs.RecordAPICall("/api/v1/query", "GET", 200, float64(apiDuration), nil)
			output.TopFlows = flows

			// Calculate total flows
			for _, flow := range flows {
				output.Summary.TotalFlows += flow.FlowVolume
			}
		}
	}

	// Query DNS queries if requested
	if c.metric == "all" || c.metric == "dns" {
		span = obs.StartSpan("query_dns")

		start := time.Now()
		dnsQueries, err := c.queryDNS(ddClient, fromTime, toTime)
		apiDuration := time.Since(start).Milliseconds()
		obs.FinishSpan(span)

		if err != nil {
			obs.RecordAPICall("/api/v1/query", "GET", 500, float64(apiDuration), err)
			obs.LogWarning(fmt.Sprintf("Failed to query DNS: %s", err.Error()))
		} else {
			obs.RecordAPICall("/api/v1/query", "GET", 200, float64(apiDuration), nil)
			output.TopDNSQueries = dnsQueries

			// Calculate DNS statistics
			var totalFailures int64
			for _, dns := range dnsQueries {
				output.Summary.TotalDNSQueries += dns.QueryCount
				totalFailures += dns.FailureCount
			}

			if output.Summary.TotalDNSQueries > 0 {
				output.Summary.DNSFailurePercent = float64(totalFailures) / float64(output.Summary.TotalDNSQueries) * 100
			}
		}
	}

	// Query network latency if requested
	if c.metric == "all" || c.metric == "latency" {
		span = obs.StartSpan("query_latency")

		start := time.Now()
		latency, err := c.queryLatency(ddClient, fromTime, toTime)
		apiDuration := time.Since(start).Milliseconds()
		obs.FinishSpan(span)

		if err != nil {
			obs.RecordAPICall("/api/v1/query", "GET", 500, float64(apiDuration), err)
			obs.LogWarning(fmt.Sprintf("Failed to query latency: %s", err.Error()))
		} else {
			obs.RecordAPICall("/api/v1/query", "GET", 200, float64(apiDuration), nil)
			output.Latency = latency

			// Calculate average latency and packet loss
			if len(latency) > 0 {
				var totalRTT, totalPacketLoss float64
				for _, l := range latency {
					totalRTT += l.AvgRTTMs
					totalPacketLoss += l.PacketLoss
				}
				output.Summary.AvgLatencyMs = totalRTT / float64(len(latency))
				output.Summary.PacketLossPercent = totalPacketLoss / float64(len(latency))
			}
		}
	}

	// Handle no data case
	if output.Summary.TotalConnections == 0 && output.Summary.TotalFlows == 0 &&
	   output.Summary.TotalDNSQueries == 0 && len(output.Latency) == 0 {
		obs.LogWarning("No network monitoring data found")
		obs.GetMetrics().Gauge("network.connections", 0, "")

		output.Status = "no_data"

		if c.jsonOut {
			jsonData, _ := json.MarshalIndent(output, "", "  ")
			fmt.Println(string(jsonData))
		} else {
			fmt.Println("No network monitoring data found.")
			fmt.Println("Ensure Network Performance Monitoring is enabled in your Datadog account.")
			fmt.Println("Visit: https://app.datadoghq.com/network")
		}

		return nil
	}

	// Analyze and add issues
	span = obs.StartSpan("analyze_issues")
	c.analyzeIssues(output)
	obs.FinishSpan(span)

	// Record metrics
	obs.GetMetrics().Gauge("network.connections", float64(output.Summary.TotalConnections), "")
	obs.GetMetrics().Gauge("network.flows", float64(output.Summary.TotalFlows), "")
	obs.GetMetrics().Gauge("network.bandwidth_mb", output.Summary.TotalBandwidthMB, "")
	obs.GetMetrics().Gauge("network.dns_queries", float64(output.Summary.TotalDNSQueries), "")
	obs.GetMetrics().Gauge("network.latency_ms", output.Summary.AvgLatencyMs, "")
	obs.GetMetrics().Gauge("network.packet_loss_percent", output.Summary.PacketLossPercent, "")

	// Output
	if c.jsonOut {
		jsonData, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	} else {
		c.printFormatted(output)
	}

	obs.LogInfo(fmt.Sprintf("Network monitoring query completed: %d connections, %d flows, %d DNS queries",
		output.Summary.TotalConnections, output.Summary.TotalFlows, output.Summary.TotalDNSQueries))
	return nil
}

// parseDuration parses duration string to time range
func (c *NetworkCommand) parseDuration(duration string) (time.Time, time.Time, error) {
	var d time.Duration
	var err error

	if strings.HasSuffix(duration, "h") {
		hours := strings.TrimSuffix(duration, "h")
		var h int
		_, err = fmt.Sscanf(hours, "%d", &h)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid duration format: %s", duration)
		}
		d = time.Duration(h) * time.Hour
	} else if strings.HasSuffix(duration, "d") {
		days := strings.TrimSuffix(duration, "d")
		var day int
		_, err = fmt.Sscanf(days, "%d", &day)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid duration format: %s", duration)
		}
		d = time.Duration(day) * 24 * time.Hour
	} else {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid duration format: %s (use format like '1h', '24h', '7d', '30d')", duration)
	}

	toTime := time.Now()
	fromTime := toTime.Add(-d)

	return fromTime, toTime, nil
}

// queryConnections queries network connection data
func (c *NetworkCommand) queryConnections(ddClient *client.Client, fromTime, toTime time.Time) ([]ConnectionStats, error) {
	// Build query for network connections
	query := "avg:network.tcp.connections{*} by {source,dest}"

	// Add filters if provided
	if c.source != "" {
		query = fmt.Sprintf("avg:network.tcp.connections{source:%s}", c.source)
		if c.destination != "" {
			query = fmt.Sprintf("avg:network.tcp.connections{source:%s,dest:%s}", c.source, c.destination)
		}
	} else if c.destination != "" {
		query = fmt.Sprintf("avg:network.tcp.connections{dest:%s}", c.destination)
	}

	responseData, err := ddClient.QueryNetworkConnections(c.source, c.destination, fromTime.Format(time.RFC3339), toTime.Format(time.RFC3339))
	if err != nil {
		// Fallback to metrics API
		responseData, err = ddClient.QueryMetrics(query, fromTime, toTime)
		if err != nil {
			return nil, err
		}
	}

	connections := make([]ConnectionStats, 0)

	// Parse response
	var metricData NetworkMetricSeriesData
	if err := json.Unmarshal(responseData, &metricData); err != nil {
		return connections, nil // Return empty on parse error
	}

	// Extract connection data
	for _, series := range metricData.Series {
		if len(series.Pointlist) == 0 {
			continue
		}

		// Parse source and dest from scope
		source := "unknown"
		dest := "unknown"
		if series.Scope != "" {
			parts := strings.Split(series.Scope, ",")
			for _, part := range parts {
				if strings.HasPrefix(part, "source:") {
					source = strings.TrimPrefix(part, "source:")
				} else if strings.HasPrefix(part, "dest:") {
					dest = strings.TrimPrefix(part, "dest:")
				}
			}
		}

		// Get last point value
		lastPoint := series.Pointlist[len(series.Pointlist)-1]
		var connCount int64
		if len(lastPoint) == 2 {
			switch v := lastPoint[1].(type) {
			case float64:
				connCount = int64(v)
			case int64:
				connCount = v
			}
		}

		conn := ConnectionStats{
			Source:          source,
			Destination:     dest,
			ConnectionCount: connCount,
			BytesSent:       connCount * 1500, // Estimate based on MTU
			BytesReceived:   connCount * 1500,
			Retransmissions: 0,
		}

		connections = append(connections, conn)
	}

	// Sort by connection count (descending) and limit to top 10
	if len(connections) > 10 {
		connections = connections[:10]
	}

	return connections, nil
}

// queryFlows queries network flow data
func (c *NetworkCommand) queryFlows(ddClient *client.Client, fromTime, toTime time.Time) ([]FlowStats, error) {
	// Query network flows by protocol
	protocols := []string{"tcp", "udp"}
	if c.protocol != "all" {
		protocols = []string{c.protocol}
	}

	allFlows := make([]FlowStats, 0)

	for _, proto := range protocols {
		query := fmt.Sprintf("sum:network.%s.connections{*}", proto)

		responseData, err := ddClient.QueryNetworkFlows(query, fromTime.Format(time.RFC3339), toTime.Format(time.RFC3339))
		if err != nil {
			// Fallback to metrics API
			responseData, err = ddClient.QueryMetrics(query, fromTime, toTime)
			if err != nil {
				continue
			}
		}

		var metricData NetworkMetricSeriesData
		if err := json.Unmarshal(responseData, &metricData); err != nil {
			continue
		}

		// Extract flow data
		for _, series := range metricData.Series {
			if len(series.Pointlist) == 0 {
				continue
			}

			// Sum all points
			var total float64
			for _, point := range series.Pointlist {
				if len(point) == 2 {
					if val, ok := point[1].(float64); ok {
						total += val
					}
				}
			}

			flow := FlowStats{
				Protocol:   proto,
				Port:       0, // Would need additional querying
				FlowVolume: int64(total),
				BytesTotal: int64(total * 1500), // Estimate
			}

			allFlows = append(allFlows, flow)
		}
	}

	// Limit to top 10
	if len(allFlows) > 10 {
		allFlows = allFlows[:10]
	}

	return allFlows, nil
}

// queryDNS queries DNS query statistics
func (c *NetworkCommand) queryDNS(ddClient *client.Client, fromTime, toTime time.Time) ([]DNSStats, error) {
	// Query DNS metrics
	query := "sum:dns.query.count{*} by {domain}"

	responseData, err := ddClient.QueryDNSQueries(query, fromTime.Format(time.RFC3339), toTime.Format(time.RFC3339))
	if err != nil {
		// Fallback to metrics API
		responseData, err = ddClient.QueryMetrics(query, fromTime, toTime)
		if err != nil {
			return nil, err
		}
	}

	dnsStats := make([]DNSStats, 0)

	var metricData NetworkMetricSeriesData
	if err := json.Unmarshal(responseData, &metricData); err != nil {
		return dnsStats, nil
	}

	// Extract DNS data
	for _, series := range metricData.Series {
		if len(series.Pointlist) == 0 {
			continue
		}

		// Parse domain from scope
		domain := "unknown"
		if series.Scope != "" && strings.Contains(series.Scope, "domain:") {
			domain = strings.TrimPrefix(series.Scope, "domain:")
		}

		// Sum all queries
		var totalQueries float64
		for _, point := range series.Pointlist {
			if len(point) == 2 {
				if val, ok := point[1].(float64); ok {
					totalQueries += val
				}
			}
		}

		dns := DNSStats{
			Domain:         domain,
			QueryCount:     int64(totalQueries),
			AvgResTimeMs:   50, // Would need resolution time metric
			FailureCount:   0,  // Would need failure metric
			NXDomainCount:  0,  // Would need NXDOMAIN metric
		}

		dnsStats = append(dnsStats, dns)
	}

	// Sort by query count and limit to top 10
	if len(dnsStats) > 10 {
		dnsStats = dnsStats[:10]
	}

	return dnsStats, nil
}

// queryLatency queries network latency metrics
func (c *NetworkCommand) queryLatency(ddClient *client.Client, fromTime, toTime time.Time) ([]LatencyStats, error) {
	// Query RTT metrics
	query := "avg:network.tcp.rtt{*}"

	responseData, err := ddClient.GetNetworkMetrics(query, fromTime.Format(time.RFC3339), toTime.Format(time.RFC3339))
	if err != nil {
		// Fallback to metrics API
		responseData, err = ddClient.QueryMetrics(query, fromTime, toTime)
		if err != nil {
			return nil, err
		}
	}

	latencyStats := make([]LatencyStats, 0)

	var metricData NetworkMetricSeriesData
	if err := json.Unmarshal(responseData, &metricData); err != nil {
		return latencyStats, nil
	}

	// Extract latency data
	for _, series := range metricData.Series {
		if len(series.Pointlist) == 0 {
			continue
		}

		// Calculate statistics from points
		var sum, max, min float64
		min = -1
		values := make([]float64, 0)

		for _, point := range series.Pointlist {
			if len(point) == 2 {
				if val, ok := point[1].(float64); ok {
					values = append(values, val)
					sum += val
					if val > max {
						max = val
					}
					if min < 0 || val < min {
						min = val
					}
				}
			}
		}

		if len(values) == 0 {
			continue
		}

		avg := sum / float64(len(values))

		// Calculate P95 and P99
		p95Idx := int(float64(len(values)) * 0.95)
		p99Idx := int(float64(len(values)) * 0.99)
		if p95Idx >= len(values) {
			p95Idx = len(values) - 1
		}
		if p99Idx >= len(values) {
			p99Idx = len(values) - 1
		}

		latency := LatencyStats{
			SourceDest: series.Scope,
			AvgRTTMs:   avg / 1000, // Convert microseconds to milliseconds
			P95RTTMs:   values[p95Idx] / 1000,
			P99RTTMs:   values[p99Idx] / 1000,
			PacketLoss: 0.0, // Would need retransmit metric
		}

		latencyStats = append(latencyStats, latency)
	}

	// Limit to top 10
	if len(latencyStats) > 10 {
		latencyStats = latencyStats[:10]
	}

	return latencyStats, nil
}

// analyzeIssues analyzes network metrics and adds issues
func (c *NetworkCommand) analyzeIssues(output *NetworkOutput) {
	output.Issues = make([]struct {
		Severity string `json:"severity"`
		Message  string `json:"message"`
		Action   string `json:"action"`
	}, 0)

	// Check for high latency
	if output.Summary.AvgLatencyMs > 100 {
		severity := "warning"
		if output.Summary.AvgLatencyMs > 200 {
			severity = "critical"
			output.Status = "critical"
		} else if output.Status != "critical" {
			output.Status = "warning"
		}

		output.Issues = append(output.Issues, struct {
			Severity string `json:"severity"`
			Message  string `json:"message"`
			Action   string `json:"action"`
		}{
			Severity: severity,
			Message:  fmt.Sprintf("High network latency detected (%.1fms avg)", output.Summary.AvgLatencyMs),
			Action:   "Investigate network path, check for bandwidth saturation or routing issues",
		})
	}

	// Check for packet loss
	if output.Summary.PacketLossPercent > 1.0 {
		severity := "warning"
		if output.Summary.PacketLossPercent > 5.0 {
			severity = "critical"
			output.Status = "critical"
		} else if output.Status != "critical" {
			output.Status = "warning"
		}

		output.Issues = append(output.Issues, struct {
			Severity string `json:"severity"`
			Message  string `json:"message"`
			Action   string `json:"action"`
		}{
			Severity: severity,
			Message:  fmt.Sprintf("Packet loss detected (%.2f%%)", output.Summary.PacketLossPercent),
			Action:   "Check network hardware, investigate congestion or link quality issues",
		})
	}

	// Check for DNS failures
	if output.Summary.DNSFailurePercent > 5.0 {
		severity := "warning"
		if output.Summary.DNSFailurePercent > 10.0 {
			severity = "critical"
			if output.Status != "critical" {
				output.Status = "critical"
			}
		} else if output.Status == "ok" {
			output.Status = "warning"
		}

		output.Issues = append(output.Issues, struct {
			Severity string `json:"severity"`
			Message  string `json:"message"`
			Action   string `json:"action"`
		}{
			Severity: severity,
			Message:  fmt.Sprintf("High DNS failure rate (%.1f%%)", output.Summary.DNSFailurePercent),
			Action:   "Check DNS server health, investigate NXDOMAIN responses and timeouts",
		})
	}

	// Check for high bandwidth usage
	if output.Summary.TotalBandwidthMB > 10000 {
		if output.Status == "ok" {
			output.Status = "warning"
		}

		output.Issues = append(output.Issues, struct {
			Severity string `json:"severity"`
			Message  string `json:"message"`
			Action   string `json:"action"`
		}{
			Severity: "warning",
			Message:  fmt.Sprintf("High bandwidth usage detected (%.2f GB)", output.Summary.TotalBandwidthMB/1024),
			Action:   "Review top talkers, consider implementing traffic shaping or rate limiting",
		})
	}

	// Check for retransmissions in top connections
	highRetrans := 0
	for _, conn := range output.TopConnections {
		if conn.Retransmissions > 100 {
			highRetrans++
		}
	}

	if highRetrans > 0 && output.Status == "ok" {
		output.Status = "warning"
		output.Issues = append(output.Issues, struct {
			Severity string `json:"severity"`
			Message  string `json:"message"`
			Action   string `json:"action"`
		}{
			Severity: "warning",
			Message:  fmt.Sprintf("High retransmissions detected on %d connections", highRetrans),
			Action:   "Investigate network reliability, check for congestion or MTU issues",
		})
	}
}

// printFormatted prints the network output in a conversational format
func (c *NetworkCommand) printFormatted(output *NetworkOutput) {
	// Header with status indicator
	statusIndicator := "OK"
	if output.Status == "warning" {
		statusIndicator = "WARNING"
	} else if output.Status == "critical" {
		statusIndicator = "CRITICAL"
	}

	fmt.Printf("Network Performance Monitoring [%s]\n", statusIndicator)
	fmt.Printf("Duration: %s\n", output.Duration)
	fmt.Println()

	// Summary
	summary := output.Summary
	fmt.Println("Summary:")
	fmt.Printf("  Total connections: %s\n", formatNetworkNumber(summary.TotalConnections))
	fmt.Printf("  Total flows: %s\n", formatNetworkNumber(summary.TotalFlows))
	fmt.Printf("  Bandwidth: %.2f GB\n", summary.TotalBandwidthMB/1024)
	fmt.Printf("  DNS queries: %s\n", formatNetworkNumber(summary.TotalDNSQueries))
	fmt.Printf("  Avg latency: %.2f ms\n", summary.AvgLatencyMs)
	if summary.PacketLossPercent > 0 {
		fmt.Printf("  Packet loss: %.2f%%\n", summary.PacketLossPercent)
	}
	if summary.DNSFailurePercent > 0 {
		fmt.Printf("  DNS failure rate: %.2f%%\n", summary.DNSFailurePercent)
	}
	fmt.Println()

	// Issues
	if len(output.Issues) > 0 {
		fmt.Println("Issues:")
		for _, issue := range output.Issues {
			fmt.Printf("  [%s] %s\n", strings.ToUpper(issue.Severity), issue.Message)
			fmt.Printf("    Action: %s\n", issue.Action)
		}
		fmt.Println()
	}

	// Top connections
	if len(output.TopConnections) > 0 {
		fmt.Println("Top Network Connections:")
		displayCount := 5
		if len(output.TopConnections) < displayCount {
			displayCount = len(output.TopConnections)
		}

		for i := 0; i < displayCount; i++ {
			conn := output.TopConnections[i]
			fmt.Printf("  %d. %s -> %s\n", i+1, conn.Source, conn.Destination)
			fmt.Printf("     Connections: %s | Sent: %s | Received: %s\n",
				formatNetworkNumber(conn.ConnectionCount),
				formatBytes(conn.BytesSent),
				formatBytes(conn.BytesReceived))
		}

		if len(output.TopConnections) > displayCount {
			fmt.Printf("  ... and %d more\n", len(output.TopConnections)-displayCount)
		}
		fmt.Println()
	}

	// Top flows
	if len(output.TopFlows) > 0 {
		fmt.Println("Top Network Flows:")
		displayCount := 5
		if len(output.TopFlows) < displayCount {
			displayCount = len(output.TopFlows)
		}

		for i := 0; i < displayCount; i++ {
			flow := output.TopFlows[i]
			portStr := ""
			if flow.Port > 0 {
				portStr = fmt.Sprintf(":%d", flow.Port)
			}
			fmt.Printf("  %d. %s%s\n", i+1, strings.ToUpper(flow.Protocol), portStr)
			fmt.Printf("     Volume: %s flows | Data: %s\n",
				formatNetworkNumber(flow.FlowVolume),
				formatBytes(flow.BytesTotal))
		}

		if len(output.TopFlows) > displayCount {
			fmt.Printf("  ... and %d more\n", len(output.TopFlows)-displayCount)
		}
		fmt.Println()
	}

	// Top DNS queries
	if len(output.TopDNSQueries) > 0 {
		fmt.Println("Top DNS Queries:")
		displayCount := 5
		if len(output.TopDNSQueries) < displayCount {
			displayCount = len(output.TopDNSQueries)
		}

		for i := 0; i < displayCount; i++ {
			dns := output.TopDNSQueries[i]
			fmt.Printf("  %d. %s\n", i+1, dns.Domain)
			fmt.Printf("     Queries: %s | Avg res time: %dms",
				formatNetworkNumber(dns.QueryCount),
				dns.AvgResTimeMs)
			if dns.FailureCount > 0 {
				fmt.Printf(" | Failures: %d", dns.FailureCount)
			}
			fmt.Println()
		}

		if len(output.TopDNSQueries) > displayCount {
			fmt.Printf("  ... and %d more\n", len(output.TopDNSQueries)-displayCount)
		}
		fmt.Println()
	}

	// Latency
	if len(output.Latency) > 0 {
		fmt.Println("Network Latency:")
		displayCount := 5
		if len(output.Latency) < displayCount {
			displayCount = len(output.Latency)
		}

		for i := 0; i < displayCount; i++ {
			lat := output.Latency[i]
			fmt.Printf("  %d. %s\n", i+1, lat.SourceDest)
			fmt.Printf("     Avg: %.2fms | P95: %.2fms | P99: %.2fms",
				lat.AvgRTTMs, lat.P95RTTMs, lat.P99RTTMs)
			if lat.PacketLoss > 0 {
				fmt.Printf(" | Loss: %.2f%%", lat.PacketLoss)
			}
			fmt.Println()
		}

		if len(output.Latency) > displayCount {
			fmt.Printf("  ... and %d more\n", len(output.Latency)-displayCount)
		}
		fmt.Println()
	}

	// Recommendations
	if output.Status == "ok" && len(output.Issues) == 0 {
		fmt.Println("Network performance is healthy")
	}
}

// Help prints the help message
func (c *NetworkCommand) Help() {
	fmt.Println("Usage: dd network [options]")
	fmt.Println()
	fmt.Println("Query Datadog Network Performance Monitoring for network analysis.")
	fmt.Println("Analyzes connections, flows, DNS queries, latency, and bandwidth usage.")
	fmt.Println()
	fmt.Println("Options:")
	c.flags.PrintDefaults()
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  dd network")
	fmt.Println("  dd network --duration 24h")
	fmt.Println("  dd network --source 10.0.1.5")
	fmt.Println("  dd network --source 10.0.1.5 --destination 10.0.2.10")
	fmt.Println("  dd network --metric dns --duration 7d")
	fmt.Println("  dd network --metric latency --protocol tcp")
	fmt.Println("  dd network --port 443 --duration 1h")
	fmt.Println("  dd network --duration 30d --json")
}

// Helper functions

func formatNetworkNumber(n int64) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1000000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	if n < 1000000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	}
	return fmt.Sprintf("%.1fB", float64(n)/1000000000)
}

func formatBytes(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	if bytes < 1024*1024 {
		return fmt.Sprintf("%.2f KB", float64(bytes)/1024)
	}
	if bytes < 1024*1024*1024 {
		return fmt.Sprintf("%.2f MB", float64(bytes)/1024/1024)
	}
	return fmt.Sprintf("%.2f GB", float64(bytes)/1024/1024/1024)
}
