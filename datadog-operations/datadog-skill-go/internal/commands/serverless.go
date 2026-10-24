package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/datadog/skill/internal/client"
)

// ServerlessCommand queries serverless function monitoring
type ServerlessCommand struct {
	flags      *flag.FlagSet
	function   string
	provider   string
	region     string
	duration   string
	metric     string
	jsonOut    bool
}

// NewServerlessCommand creates a new serverless command
func NewServerlessCommand() Command {
	cmd := &ServerlessCommand{
		flags: flag.NewFlagSet("serverless", flag.ExitOnError),
	}

	cmd.flags.StringVar(&cmd.function, "function", "", "Function name (partial match)")
	cmd.flags.StringVar(&cmd.provider, "provider", "", "Cloud provider (aws, azure, gcp)")
	cmd.flags.StringVar(&cmd.region, "region", "", "Cloud region")
	cmd.flags.StringVar(&cmd.duration, "duration", "1h", "Time range (e.g., 1h, 24h, 7d)")
	cmd.flags.StringVar(&cmd.metric, "metric", "invocations", "Metric type (invocations, duration, errors, cold-starts)")
	cmd.flags.BoolVar(&cmd.jsonOut, "json", false, "Output in JSON format")

	return cmd
}

// Name returns the command name
func (c *ServerlessCommand) Name() string {
	return "serverless"
}

// Description returns a short description
func (c *ServerlessCommand) Description() string {
	return "Query serverless function monitoring for AWS Lambda, Azure Functions, and Google Cloud Functions"
}

// Run executes the serverless command
func (c *ServerlessCommand) Run(args []string) error {
	if err := c.flags.Parse(args); err != nil {
		return err
	}

	ddClient, err := client.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Datadog client: %w", err)
	}

	// Parse duration
	duration, err := c.parseDuration(c.duration)
	if err != nil {
		return fmt.Errorf("invalid duration: %w", err)
	}

	to := time.Now()
	from := to.Add(-duration)

	// Build metric query based on provider and metric type
	query := c.buildMetricQuery()

	resp, err := ddClient.QueryMetrics(query, from, to)
	if err != nil {
		return fmt.Errorf("failed to query serverless metrics: %w", err)
	}

	if c.jsonOut {
		fmt.Println(string(resp))
		return nil
	}

	return c.printResults(resp, from, to)
}

// buildMetricQuery constructs the metrics query based on parameters
func (c *ServerlessCommand) buildMetricQuery() string {
	var metricName string

	// Determine metric name based on provider and metric type
	switch c.provider {
	case "aws", "":
		// AWS Lambda metrics
		switch c.metric {
		case "invocations":
			metricName = "aws.lambda.enhanced.invocations"
		case "duration":
			metricName = "aws.lambda.enhanced.duration"
		case "errors":
			metricName = "aws.lambda.enhanced.errors"
		case "cold-starts":
			metricName = "aws.lambda.enhanced.init_duration"
		default:
			metricName = "aws.lambda.enhanced.invocations"
		}
	case "azure":
		// Azure Functions metrics
		switch c.metric {
		case "invocations":
			metricName = "azure.functions.function_execution_count"
		case "duration":
			metricName = "azure.functions.function_execution_time"
		case "errors":
			metricName = "azure.functions.http_server_errors"
		default:
			metricName = "azure.functions.function_execution_count"
		}
	case "gcp":
		// Google Cloud Functions metrics
		switch c.metric {
		case "invocations":
			metricName = "gcp.cloudfunctions.function.execution_count"
		case "duration":
			metricName = "gcp.cloudfunctions.function.execution_times"
		case "errors":
			metricName = "gcp.cloudfunctions.function.error_count"
		default:
			metricName = "gcp.cloudfunctions.function.execution_count"
		}
	default:
		metricName = "aws.lambda.enhanced.invocations"
	}

	// Build query with filters
	filters := []string{}

	if c.function != "" {
		switch c.provider {
		case "azure":
			filters = append(filters, fmt.Sprintf("functionname:%s", c.function))
		case "gcp":
			filters = append(filters, fmt.Sprintf("function_name:%s", c.function))
		default:
			filters = append(filters, fmt.Sprintf("functionname:%s", c.function))
		}
	}

	if c.region != "" {
		switch c.provider {
		case "azure":
			filters = append(filters, fmt.Sprintf("region:%s", c.region))
		case "gcp":
			filters = append(filters, fmt.Sprintf("region:%s", c.region))
		default:
			filters = append(filters, fmt.Sprintf("region:%s", c.region))
		}
	}

	// Construct query
	if len(filters) > 0 {
		return fmt.Sprintf("sum:%s{%s}", metricName, strings.Join(filters, ","))
	}

	return fmt.Sprintf("sum:%s{*}", metricName)
}

// printResults formats and displays serverless metrics
func (c *ServerlessCommand) printResults(data []byte, from, to time.Time) error {
	var result struct {
		Series []struct {
			QueryIndex  int               `json:"query_index"`
			DisplayName string            `json:"display_name"`
			Pointlist   [][]float64       `json:"pointlist"`
			TagSet      []string          `json:"tag_set"`
			Unit        []map[string]interface{} `json:"unit"`
		} `json:"series"`
		Status string `json:"status"`
		Error  string `json:"error"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("failed to parse results: %w", err)
	}

	if result.Error != "" {
		return fmt.Errorf("API error: %s", result.Error)
	}

	if len(result.Series) == 0 {
		fmt.Println("No serverless function data found.")
		fmt.Println("\nTip: Serverless monitoring requires:")
		fmt.Println("  1. Datadog Lambda Extension or Forwarder installed")
		fmt.Println("  2. Functions instrumented with Datadog libraries")
		fmt.Println("  3. Enhanced metrics enabled")
		return nil
	}

	// Display header
	providerName := c.getProviderName()
	metricType := c.getMetricTypeName()

	fmt.Printf("Serverless Functions - %s (%s)\n", providerName, metricType)
	fmt.Printf("Time Range: %s to %s\n", from.Format("2006-01-02 15:04:05"), to.Format("2006-01-02 15:04:05"))
	fmt.Println(strings.Repeat("=", 80))

	// Group metrics by function
	functionMetrics := make(map[string][]float64)
	functionTags := make(map[string][]string)

	for _, series := range result.Series {
		functionName := c.extractFunctionName(series.TagSet)
		if functionName == "" {
			functionName = "Unknown"
		}

		// Aggregate data points
		var values []float64
		for _, point := range series.Pointlist {
			if len(point) >= 2 {
				values = append(values, point[1])
			}
		}

		functionMetrics[functionName] = values
		functionTags[functionName] = series.TagSet
	}

	// Display summary
	fmt.Printf("\nFunctions Found: %d\n\n", len(functionMetrics))

	// Display metrics for each function
	for functionName, values := range functionMetrics {
		if len(values) == 0 {
			continue
		}

		// Calculate statistics
		total, avg, min, max := calculateStats(values)

		fmt.Printf("Function: %s\n", functionName)

		switch c.metric {
		case "invocations":
			fmt.Printf("  Total Invocations: %.0f\n", total)
			fmt.Printf("  Avg per interval: %.2f\n", avg)
			fmt.Printf("  Peak: %.0f\n", max)
		case "duration":
			fmt.Printf("  Avg Duration: %.2f ms\n", avg)
			fmt.Printf("  Min: %.2f ms\n", min)
			fmt.Printf("  Max: %.2f ms\n", max)
		case "errors":
			fmt.Printf("  Total Errors: %.0f\n", total)
			fmt.Printf("  Avg per interval: %.2f\n", avg)
			fmt.Printf("  Peak: %.0f\n", max)
		case "cold-starts":
			fmt.Printf("  Avg Init Duration: %.2f ms\n", avg)
			fmt.Printf("  Min: %.2f ms\n", min)
			fmt.Printf("  Max: %.2f ms\n", max)
		}

		// Show relevant tags
		tags := functionTags[functionName]
		relevantTags := c.filterRelevantTags(tags)
		if len(relevantTags) > 0 && len(relevantTags) <= 3 {
			fmt.Printf("  Tags: %s\n", strings.Join(relevantTags, ", "))
		}

		fmt.Println()
	}

	return nil
}

// extractFunctionName extracts function name from tags
func (c *ServerlessCommand) extractFunctionName(tags []string) string {
	for _, tag := range tags {
		if strings.HasPrefix(tag, "functionname:") {
			return strings.TrimPrefix(tag, "functionname:")
		}
		if strings.HasPrefix(tag, "function_name:") {
			return strings.TrimPrefix(tag, "function_name:")
		}
	}
	return ""
}

// filterRelevantTags removes common tags to show only interesting ones
func (c *ServerlessCommand) filterRelevantTags(tags []string) []string {
	var relevant []string
	skipPrefixes := []string{
		"functionname:",
		"function_name:",
	}

	for _, tag := range tags {
		skip := false
		for _, prefix := range skipPrefixes {
			if strings.HasPrefix(tag, prefix) {
				skip = true
				break
			}
		}
		if !skip && len(tag) > 0 {
			relevant = append(relevant, tag)
		}
	}

	return relevant
}

// calculateStats calculates basic statistics from values
func calculateStats(values []float64) (total, avg, min, max float64) {
	if len(values) == 0 {
		return 0, 0, 0, 0
	}

	total = 0
	min = values[0]
	max = values[0]

	for _, v := range values {
		total += v
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	avg = total / float64(len(values))
	return
}

// getProviderName returns human-readable provider name
func (c *ServerlessCommand) getProviderName() string {
	switch c.provider {
	case "aws":
		return "AWS Lambda"
	case "azure":
		return "Azure Functions"
	case "gcp":
		return "Google Cloud Functions"
	default:
		return "AWS Lambda"
	}
}

// getMetricTypeName returns human-readable metric type
func (c *ServerlessCommand) getMetricTypeName() string {
	switch c.metric {
	case "invocations":
		return "Invocations"
	case "duration":
		return "Execution Duration"
	case "errors":
		return "Errors"
	case "cold-starts":
		return "Cold Start Duration"
	default:
		return "Invocations"
	}
}

// parseDuration parses duration string (e.g., "1h", "24h", "7d")
func (c *ServerlessCommand) parseDuration(dur string) (time.Duration, error) {
	if len(dur) < 2 {
		return 0, fmt.Errorf("invalid duration format")
	}

	unit := dur[len(dur)-1]
	valueStr := dur[:len(dur)-1]

	var value int
	_, err := fmt.Sscanf(valueStr, "%d", &value)
	if err != nil {
		return 0, fmt.Errorf("invalid duration value: %w", err)
	}

	switch unit {
	case 'h':
		return time.Duration(value) * time.Hour, nil
	case 'd':
		return time.Duration(value) * 24 * time.Hour, nil
	case 'm':
		return time.Duration(value) * time.Minute, nil
	default:
		return 0, fmt.Errorf("unsupported duration unit: %c (use h, d, or m)", unit)
	}
}

// Help displays help information
func (c *ServerlessCommand) Help() {
	help := `dd serverless - Query Serverless Function Monitoring

DESCRIPTION:
  Query serverless function monitoring for AWS Lambda, Azure Functions, and
  Google Cloud Functions. Track invocations, duration, errors, and cold starts.

USAGE:
  dd serverless [options]

OPTIONS:
  --function string   Function name (partial match)
  --provider string   Cloud provider (aws, azure, gcp) - default: aws
  --region string     Cloud region
  --duration string   Time range (default: 1h)
                      Examples: 1h, 6h, 24h, 7d, 30d
  --metric string     Metric type (default: invocations)
                      Options: invocations, duration, errors, cold-starts
  --json              Output in JSON format

EXAMPLES:
  # Query all Lambda function invocations (last hour)
  dd serverless

  # Query specific function
  dd serverless --function my-api-handler

  # Query execution duration
  dd serverless --metric duration

  # Query error counts
  dd serverless --metric errors

  # Query cold start durations
  dd serverless --metric cold-starts

  # Query Azure Functions
  dd serverless --provider azure --function my-function

  # Query Google Cloud Functions
  dd serverless --provider gcp --function my-function

  # Query specific region
  dd serverless --region us-east-1

  # Query last 24 hours
  dd serverless --duration 24h

  # Get JSON output
  dd serverless --function my-api --json

PROVIDERS:
  aws    - AWS Lambda (default)
  azure  - Azure Functions
  gcp    - Google Cloud Functions

METRICS:
  invocations  - Function invocation counts
  duration     - Execution time in milliseconds
  errors       - Error counts
  cold-starts  - Cold start initialization time

USE CASES:
  1. Monitor function invocation rates
  2. Track execution performance
  3. Identify high-error functions
  4. Analyze cold start patterns
  5. Compare functions across regions

AWS LAMBDA METRICS:
  Enhanced metrics (recommended):
    aws.lambda.enhanced.invocations
    aws.lambda.enhanced.duration
    aws.lambda.enhanced.errors
    aws.lambda.enhanced.init_duration
    aws.lambda.enhanced.timeout
    aws.lambda.enhanced.out_of_memory

  Standard CloudWatch metrics:
    aws.lambda.invocations
    aws.lambda.duration
    aws.lambda.errors
    aws.lambda.throttles

AZURE FUNCTIONS METRICS:
  azure.functions.function_execution_count
  azure.functions.function_execution_time
  azure.functions.http_server_errors

GOOGLE CLOUD FUNCTIONS METRICS:
  gcp.cloudfunctions.function.execution_count
  gcp.cloudfunctions.function.execution_times
  gcp.cloudfunctions.function.error_count

REQUIREMENTS:
  For AWS Lambda enhanced metrics, you need:
    1. Datadog Lambda Extension installed
    2. Functions instrumented with Datadog library
    3. Enhanced metrics enabled

  For Azure Functions:
    1. Azure integration configured
    2. Datadog extension installed

  For Google Cloud Functions:
    1. GCP integration configured
    2. Datadog tracing enabled

TROUBLESHOOTING:
  If no data appears:
    1. Verify serverless integration is configured
    2. Check that functions have been invoked recently
    3. Ensure enhanced metrics are enabled (AWS)
    4. Verify function names match exactly

AUTHENTICATION:
  Requires DD_API_KEY and DD_APP_KEY environment variables.
`
	fmt.Println(strings.TrimSpace(help))
}
