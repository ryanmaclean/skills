package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/datadog/skill/internal/client"
	"github.com/datadog/skill/internal/observability"
)

// MetricsCommand queries Datadog metrics for time series analysis
type MetricsCommand struct {
	flags *flag.FlagSet
	query string
	from  string
	to    string
	json  bool
}

// MetricsAPIResponse represents the response from Datadog Metrics API
type MetricsAPIResponse struct {
	Status string `json:"status"`
	Series []struct {
		Expression  string      `json:"expression"`
		QueryIndex  int         `json:"query_index"`
		Pointlist   [][2]interface{} `json:"pointlist"`
		Scope       string      `json:"scope"`
		Unit        []struct {
			Family      string `json:"family"`
			ScaleFactor int    `json:"scale_factor"`
			Name        string `json:"name"`
			ShortName   string `json:"short_name"`
			Plural      string `json:"plural"`
			ID          int    `json:"id"`
		} `json:"unit,omitempty"`
		Aggr        string      `json:"aggr,omitempty"`
		Metric      string      `json:"metric,omitempty"`
		TagSet      []string    `json:"tag_set,omitempty"`
		Interval    int         `json:"interval,omitempty"`
	} `json:"series"`
	RespVersion int    `json:"resp_version"`
	Query       string `json:"query"`
	FromDate    int64  `json:"from_date"`
	ToDate      int64  `json:"to_date"`
	Message     string `json:"message,omitempty"`
}

// DataPoint represents a single time series data point
type DataPoint struct {
	Timestamp int64   `json:"timestamp"`
	Value     float64 `json:"value"`
}

// Statistics represents calculated statistics for a time series
type Statistics struct {
	Count  int     `json:"count"`
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	Avg    float64 `json:"avg"`
	P50    float64 `json:"p50"`
	P95    float64 `json:"p95"`
	P99    float64 `json:"p99"`
	StdDev float64 `json:"stddev"`
}

// TrendAnalysis represents trend analysis results
type TrendAnalysis struct {
	Direction      string  `json:"direction"`
	ChangePercent  float64 `json:"change_percent"`
	Status         string  `json:"status"`
	FirstHalfAvg   float64 `json:"first_half_avg"`
	SecondHalfAvg  float64 `json:"second_half_avg"`
}

// AnomalyDetection represents anomaly detection results
type AnomalyDetection struct {
	Status           string    `json:"status"`
	Count            int       `json:"count"`
	Percentage       float64   `json:"percentage"`
	ThresholdLower   float64   `json:"threshold_lower"`
	ThresholdUpper   float64   `json:"threshold_upper"`
	DetectedValues   []float64 `json:"detected_values,omitempty"`
}

// MetricsOutput represents the structured output
type MetricsOutput struct {
	Status    string `json:"status"`
	Metadata  struct {
		Query       string `json:"query"`
		From        string `json:"from"`
		To          string `json:"to"`
		Unit        string `json:"unit,omitempty"`
		Scope       string `json:"scope,omitempty"`
	} `json:"metadata"`
	Statistics       *Statistics       `json:"statistics"`
	Trend            *TrendAnalysis    `json:"trend"`
	Anomalies        *AnomalyDetection `json:"anomalies"`
	TimeSeries       []DataPoint       `json:"time_series,omitempty"`
	Message          string            `json:"message,omitempty"`
}

// NewMetricsCommand creates a new metrics command
func NewMetricsCommand() *MetricsCommand {
	cmd := &MetricsCommand{
		flags: flag.NewFlagSet("metrics", flag.ExitOnError),
	}

	cmd.flags.StringVar(&cmd.query, "query", "", "Metric query (e.g., 'system.cpu.user', 'avg:requests.count{service:api}')")
	cmd.flags.StringVar(&cmd.from, "from", "1h", "Start time (duration like '1h' ago, or Unix timestamp)")
	cmd.flags.StringVar(&cmd.to, "to", "now", "End time (default: now, or Unix timestamp)")
	cmd.flags.BoolVar(&cmd.json, "json", false, "Output as JSON")

	return cmd
}

// Name returns the command name
func (c *MetricsCommand) Name() string {
	return "metrics"
}

// Description returns the command description
func (c *MetricsCommand) Description() string {
	return "Query Datadog metrics for time series analysis"
}

// Run executes the metrics command
func (c *MetricsCommand) Run(args []string) error {
	// Initialize observability
	obs, err := observability.Init("query-metrics", "production")
	if err != nil {
		return fmt.Errorf("failed to init observability: %w", err)
	}
	defer obs.Shutdown(0)

	// Parse flags
	if err := c.flags.Parse(args); err != nil {
		return err
	}

	// Validate query
	if c.query == "" {
		obs.LogError("Metric query is required")
		return fmt.Errorf("metric query is required: use --query flag")
	}

	obs.LogInfo(fmt.Sprintf("Querying metrics: %s", c.query))

	// Parse time range
	span := obs.StartSpan("parse_time_range")
	fromTime, toTime, err := c.parseTimeRange()
	obs.FinishSpan(span)

	if err != nil {
		obs.LogError(fmt.Sprintf("Invalid time range: %s", err.Error()))
		return fmt.Errorf("invalid time range: %w", err)
	}

	obs.LogInfo(fmt.Sprintf("Time range: %s to %s", fromTime.Format(time.RFC3339), toTime.Format(time.RFC3339)))

	// Create Datadog client
	span = obs.StartSpan("create_client")
	ddClient, err := client.NewClient()
	obs.FinishSpan(span)

	if err != nil {
		obs.LogError(fmt.Sprintf("Failed to create Datadog client: %s", err.Error()))
		return fmt.Errorf("failed to create Datadog client: %w", err)
	}

	// Query metrics
	span = obs.StartSpan("query_metrics")
	obs.GetTracer().SetTag(span, "query", c.query)
	obs.GetTracer().SetTag(span, "from", c.from)
	obs.GetTracer().SetTag(span, "to", c.to)

	start := time.Now()
	responseData, err := ddClient.QueryMetrics(url.QueryEscape(c.query), fromTime, toTime)
	apiDuration := time.Since(start).Milliseconds()
	obs.FinishSpan(span)

	if err != nil {
		obs.RecordAPICall("/api/v1/query", "GET", 500, float64(apiDuration), err)
		obs.LogError(fmt.Sprintf("API call failed: %s", err.Error()))
		return fmt.Errorf("failed to query metrics: %w", err)
	}

	obs.RecordAPICall("/api/v1/query", "GET", 200, float64(apiDuration), nil)

	// Parse results
	span = obs.StartSpan("parse_results")
	output, err := c.parseResults(responseData)
	obs.FinishSpan(span)

	if err != nil {
		obs.LogError(fmt.Sprintf("Failed to parse results: %s", err.Error()))
		return fmt.Errorf("failed to parse results: %w", err)
	}

	// Set metadata
	output.Metadata.Query = c.query
	output.Metadata.From = c.from
	output.Metadata.To = c.to

	// Handle no data case
	if output.Status == "no_data" {
		obs.LogWarning("No metric data found")
		obs.GetMetrics().Gauge("metrics.datapoints", 0)

		if c.json {
			jsonData, _ := json.MarshalIndent(output, "", "  ")
			fmt.Println(string(jsonData))
		} else {
			fmt.Printf("No data found for metric query: %s\n", c.query)
			if output.Message != "" {
				fmt.Printf("Message: %s\n", output.Message)
			}
		}

		return nil
	}

	// Record metrics
	obs.GetMetrics().Gauge("metrics.datapoints", float64(output.Statistics.Count))
	obs.GetMetrics().Gauge("metrics.avg_value", output.Statistics.Avg)
	obs.GetMetrics().Gauge("metrics.anomalies", float64(output.Anomalies.Count))

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

	obs.LogInfo(fmt.Sprintf("Query completed: %d data points, %d anomalies", output.Statistics.Count, output.Anomalies.Count))
	return nil
}

// parseTimeRange parses the time range from flags
func (c *MetricsCommand) parseTimeRange() (time.Time, time.Time, error) {
	var toTime time.Time

	// Parse 'to' time
	if c.to == "now" || c.to == "" {
		toTime = time.Now()
	} else {
		// Try parsing as Unix timestamp
		if ts, err := strconv.ParseInt(c.to, 10, 64); err == nil {
			toTime = time.Unix(ts, 0)
		} else {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid 'to' time: %s (use 'now' or Unix timestamp)", c.to)
		}
	}

	var fromTime time.Time

	// Parse 'from' time
	if strings.HasSuffix(c.from, "h") {
		hours := strings.TrimSuffix(c.from, "h")
		h, err := strconv.Atoi(hours)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid duration format: %s", c.from)
		}
		fromTime = toTime.Add(-time.Duration(h) * time.Hour)
	} else if strings.HasSuffix(c.from, "d") {
		days := strings.TrimSuffix(c.from, "d")
		d, err := strconv.Atoi(days)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid duration format: %s", c.from)
		}
		fromTime = toTime.Add(-time.Duration(d) * 24 * time.Hour)
	} else if strings.HasSuffix(c.from, "m") {
		minutes := strings.TrimSuffix(c.from, "m")
		m, err := strconv.Atoi(minutes)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid duration format: %s", c.from)
		}
		fromTime = toTime.Add(-time.Duration(m) * time.Minute)
	} else {
		// Try parsing as Unix timestamp
		ts, err := strconv.ParseInt(c.from, 10, 64)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid 'from' time: %s (use format like '1h', '7d', or Unix timestamp)", c.from)
		}
		fromTime = time.Unix(ts, 0)
	}

	return fromTime, toTime, nil
}

// parseResults parses the raw API response into structured format
func (c *MetricsCommand) parseResults(rawData []byte) (*MetricsOutput, error) {
	var apiResponse MetricsAPIResponse
	if err := json.Unmarshal(rawData, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	output := &MetricsOutput{
		Status: "ok",
	}

	// Check for error message
	if apiResponse.Message != "" {
		output.Status = "no_data"
		output.Message = apiResponse.Message
		return output, nil
	}

	// Check if we have series data
	if len(apiResponse.Series) == 0 {
		output.Status = "no_data"
		output.Message = "No data points found for the specified metric and time range"
		return output, nil
	}

	// Extract all data points from all series
	var allValues []float64
	var timeSeries []DataPoint

	for _, series := range apiResponse.Series {
		// Set metadata from first series
		if output.Metadata.Scope == "" {
			output.Metadata.Scope = series.Scope
		}
		if output.Metadata.Unit == "" && len(series.Unit) > 0 {
			output.Metadata.Unit = series.Unit[0].Name
		}

		for _, point := range series.Pointlist {
			if len(point) != 2 {
				continue
			}

			// Extract timestamp and value
			var timestamp int64
			var value float64

			switch v := point[0].(type) {
			case float64:
				timestamp = int64(v / 1000) // Convert milliseconds to seconds
			case int64:
				timestamp = v / 1000
			}

			switch v := point[1].(type) {
			case float64:
				value = v
			case int64:
				value = float64(v)
			case nil:
				continue // Skip null values
			}

			allValues = append(allValues, value)
			timeSeries = append(timeSeries, DataPoint{
				Timestamp: timestamp,
				Value:     roundFloat(value, 2),
			})
		}
	}

	// Handle no data points case
	if len(allValues) == 0 {
		output.Status = "no_data"
		output.Message = "Time series contains no data points"
		return output, nil
	}

	// Calculate statistics
	stats := c.calculateStatistics(allValues)
	output.Statistics = stats

	// Analyze trend
	trend := c.analyzeTrend(allValues)
	output.Trend = trend

	// Detect anomalies
	anomalies := c.detectAnomalies(allValues, stats.Avg, stats.StdDev)
	output.Anomalies = anomalies

	// Determine overall status
	if anomalies.Count > (stats.Count / 10) {
		output.Status = "critical"
	} else if anomalies.Count > 0 || trend.Direction == "increasing" {
		output.Status = "warning"
	} else {
		output.Status = "ok"
	}

	// Keep last 100 data points for output
	if len(timeSeries) > 100 {
		output.TimeSeries = timeSeries[len(timeSeries)-100:]
	} else {
		output.TimeSeries = timeSeries
	}

	return output, nil
}

// calculateStatistics calculates statistical metrics for the time series
func (c *MetricsCommand) calculateStatistics(values []float64) *Statistics {
	if len(values) == 0 {
		return &Statistics{}
	}

	// Sort for percentile calculations
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	stats := &Statistics{
		Count: len(values),
		Min:   sorted[0],
		Max:   sorted[len(sorted)-1],
	}

	// Calculate average
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	stats.Avg = roundFloat(sum/float64(len(values)), 2)

	// Calculate percentiles
	stats.P50 = roundFloat(calculatePercentile(sorted, 0.50), 2)
	stats.P95 = roundFloat(calculatePercentile(sorted, 0.95), 2)
	stats.P99 = roundFloat(calculatePercentile(sorted, 0.99), 2)

	// Calculate standard deviation
	if len(values) > 1 {
		variance := 0.0
		for _, v := range values {
			diff := v - stats.Avg
			variance += diff * diff
		}
		variance /= float64(len(values) - 1)
		stats.StdDev = roundFloat(math.Sqrt(variance), 2)
	}

	stats.Min = roundFloat(stats.Min, 2)
	stats.Max = roundFloat(stats.Max, 2)

	return stats
}

// calculatePercentile calculates the percentile from a sorted slice
func calculatePercentile(sorted []float64, percentile float64) float64 {
	if len(sorted) == 0 {
		return 0.0
	}
	index := int(float64(len(sorted)) * percentile)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}

// analyzeTrend analyzes the trend by comparing first half vs second half
func (c *MetricsCommand) analyzeTrend(values []float64) *TrendAnalysis {
	trend := &TrendAnalysis{
		Direction:     "stable",
		ChangePercent: 0.0,
		Status:        "normal",
	}

	if len(values) < 2 {
		return trend
	}

	midpoint := len(values) / 2
	firstHalf := values[:midpoint]
	secondHalf := values[midpoint:]

	firstAvg := 0.0
	for _, v := range firstHalf {
		firstAvg += v
	}
	if len(firstHalf) > 0 {
		firstAvg /= float64(len(firstHalf))
	}

	secondAvg := 0.0
	for _, v := range secondHalf {
		secondAvg += v
	}
	if len(secondHalf) > 0 {
		secondAvg /= float64(len(secondHalf))
	}

	trend.FirstHalfAvg = roundFloat(firstAvg, 2)
	trend.SecondHalfAvg = roundFloat(secondAvg, 2)

	if firstAvg == 0 {
		trend.ChangePercent = 0.0
	} else {
		changePct := ((secondAvg - firstAvg) / firstAvg) * 100
		trend.ChangePercent = roundFloat(changePct, 2)

		if changePct > 10 {
			trend.Direction = "increasing"
			trend.Status = "warning"
		} else if changePct < -10 {
			trend.Direction = "decreasing"
			trend.Status = "improving"
		}
	}

	return trend
}

// detectAnomalies detects anomalies (values > 2 standard deviations from mean)
func (c *MetricsCommand) detectAnomalies(values []float64, mean, stddev float64) *AnomalyDetection {
	upperThreshold := mean + (2 * stddev)
	lowerThreshold := mean - (2 * stddev)

	var anomalousValues []float64
	for _, v := range values {
		if v > upperThreshold || v < lowerThreshold {
			anomalousValues = append(anomalousValues, v)
		}
	}

	anomalyCount := len(anomalousValues)
	anomalyPct := 0.0
	if len(values) > 0 {
		anomalyPct = (float64(anomalyCount) / float64(len(values))) * 100
	}

	status := "none"
	if anomalyCount > 0 {
		status = "detected"
	}

	detection := &AnomalyDetection{
		Status:         status,
		Count:          anomalyCount,
		Percentage:     roundFloat(anomalyPct, 2),
		ThresholdLower: roundFloat(lowerThreshold, 2),
		ThresholdUpper: roundFloat(upperThreshold, 2),
	}

	// Keep first 10 anomalous values for output
	if len(anomalousValues) > 10 {
		anomalousValues = anomalousValues[:10]
	}
	for i := range anomalousValues {
		anomalousValues[i] = roundFloat(anomalousValues[i], 2)
	}
	detection.DetectedValues = anomalousValues

	return detection
}

// printFormatted prints the metrics output in a conversational format
func (c *MetricsCommand) printFormatted(output *MetricsOutput) {
	fmt.Printf("Metric Analysis: %s\n", output.Metadata.Query)
	fmt.Printf("Time Range: %s to %s\n", output.Metadata.From, output.Metadata.To)
	if output.Metadata.Scope != "" {
		fmt.Printf("Scope: %s\n", output.Metadata.Scope)
	}
	fmt.Println()

	stats := output.Statistics
	fmt.Printf("Data points: %s\n", formatMetricNumber(stats.Count))
	fmt.Println()

	fmt.Println("Statistics:")
	fmt.Printf("  Min: %.2f\n", stats.Min)
	fmt.Printf("  Max: %.2f\n", stats.Max)
	fmt.Printf("  Avg: %.2f\n", stats.Avg)
	fmt.Printf("  P50: %.2f\n", stats.P50)
	fmt.Printf("  P95: %.2f\n", stats.P95)
	fmt.Printf("  P99: %.2f\n", stats.P99)
	fmt.Printf("  StdDev: %.2f\n", stats.StdDev)
	fmt.Println()

	trend := output.Trend
	fmt.Println("Trend Analysis:")
	fmt.Printf("  Direction: %s\n", trend.Direction)
	fmt.Printf("  Change: %.2f%%\n", trend.ChangePercent)
	fmt.Printf("  Status: %s\n", trend.Status)
	fmt.Println()

	anomalies := output.Anomalies
	fmt.Println("Anomaly Detection:")
	fmt.Printf("  Status: %s\n", anomalies.Status)
	fmt.Printf("  Count: %d (%.1f%%)\n", anomalies.Count, anomalies.Percentage)
	fmt.Printf("  Threshold: %.2f to %.2f\n", anomalies.ThresholdLower, anomalies.ThresholdUpper)

	if anomalies.Count > 0 {
		fmt.Println()
		fmt.Printf("WARNING: %d anomalous data points detected\n", anomalies.Count)
	}
}

// Help prints the help message
func (c *MetricsCommand) Help() {
	fmt.Println("Usage: dd metrics [options]")
	fmt.Println()
	fmt.Println("Query Datadog metrics for time series analysis.")
	fmt.Println("Provides statistical analysis, trend detection, and anomaly identification.")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --query string")
	fmt.Println("        Metric query (e.g., 'system.cpu.user', 'avg:requests.count{service:api}')")
	fmt.Println("  --from string")
	fmt.Println("        Start time (duration like '1h' ago, or Unix timestamp) (default: 1h)")
	fmt.Println("  --to string")
	fmt.Println("        End time (default: now, or Unix timestamp)")
	fmt.Println("  --json")
	fmt.Println("        Output as JSON")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  dd metrics --query 'system.cpu.user'")
	fmt.Println("  dd metrics --query 'avg:requests.count{service:api}'")
	fmt.Println("  dd metrics --query 'system.load.1' --from 24h")
	fmt.Println("  dd metrics --query 'avg:trace.express.request.duration{service:web}' --from 7d --json")
}
