package client

import (
	"encoding/json"
	"fmt"
	"time"
)

// Example demonstrates how to use the Datadog client in the CLI
func Example() {
	// Create client from environment variables
	client, err := NewClient()
	if err != nil {
		fmt.Printf("Failed to create client: %v\n", err)
		return
	}

	// Define time range (last hour)
	to := time.Now()
	from := to.Add(-1 * time.Hour)

	// Example 1: Query APM traces for a service
	fmt.Println("=== APM Traces ===")
	apmData, err := client.QueryAPM("my-service", from, to, "status:error")
	if err != nil {
		fmt.Printf("APM query failed: %v\n", err)
	} else {
		var apmResult map[string]interface{}
		if err := json.Unmarshal(apmData, &apmResult); err == nil {
			fmt.Printf("APM Result: %+v\n", apmResult)
		}
	}

	// Example 2: Search logs
	fmt.Println("\n=== Logs ===")
	logsData, err := client.SearchLogs("service:my-service ERROR", from, to, 50)
	if err != nil {
		fmt.Printf("Logs search failed: %v\n", err)
	} else {
		var logsResult map[string]interface{}
		if err := json.Unmarshal(logsData, &logsResult); err == nil {
			fmt.Printf("Logs Result: %+v\n", logsResult)
		}
	}

	// Example 3: Query metrics
	fmt.Println("\n=== Metrics ===")
	metricsData, err := client.QueryMetrics("avg:system.cpu.user{service:my-service}", from, to)
	if err != nil {
		fmt.Printf("Metrics query failed: %v\n", err)
	} else {
		var metricsResult map[string]interface{}
		if err := json.Unmarshal(metricsData, &metricsResult); err == nil {
			fmt.Printf("Metrics Result: %+v\n", metricsResult)
		}
	}

	// Example 4: Get service catalog
	fmt.Println("\n=== Service Catalog ===")
	catalogData, err := client.GetServiceCatalog()
	if err != nil {
		fmt.Printf("Service catalog query failed: %v\n", err)
	} else {
		var catalogResult map[string]interface{}
		if err := json.Unmarshal(catalogData, &catalogResult); err == nil {
			fmt.Printf("Catalog Result: %+v\n", catalogResult)
		}
	}

	// Example 5: Get SLOs
	fmt.Println("\n=== SLOs ===")
	slosData, err := client.GetSLOs([]string{"team:backend"})
	if err != nil {
		fmt.Printf("SLOs query failed: %v\n", err)
	} else {
		var slosResult map[string]interface{}
		if err := json.Unmarshal(slosData, &slosResult); err == nil {
			fmt.Printf("SLOs Result: %+v\n", slosResult)
		}
	}

	// Example 6: Get monitors
	fmt.Println("\n=== Monitors ===")
	monitorsData, err := client.GetMonitors([]string{"env:production"}, nil)
	if err != nil {
		fmt.Printf("Monitors query failed: %v\n", err)
	} else {
		var monitorsResult []map[string]interface{}
		if err := json.Unmarshal(monitorsData, &monitorsResult); err == nil {
			fmt.Printf("Found %d monitors\n", len(monitorsResult))
		}
	}

	// Example 7: Get security signals
	fmt.Println("\n=== Security Signals ===")
	from24h := to.Add(-24 * time.Hour)
	signalsData, err := client.GetSecuritySignals(from24h, to, "")
	if err != nil {
		fmt.Printf("Security signals query failed: %v\n", err)
	} else {
		var signalsResult map[string]interface{}
		if err := json.Unmarshal(signalsData, &signalsResult); err == nil {
			fmt.Printf("Security Signals Result: %+v\n", signalsResult)
		}
	}
}

// ExampleErrorHandling shows proper error handling patterns
func ExampleErrorHandling() {
	client, err := NewClient()
	if err != nil {
		// Handle configuration errors
		fmt.Printf("Configuration error: %v\n", err)
		return
	}

	to := time.Now()
	from := to.Add(-1 * time.Hour)

	data, err := client.QueryAPM("my-service", from, to, "")
	if err != nil {
		// Check if it's a Datadog API error
		if ddErr, ok := err.(*DatadogError); ok {
			switch ddErr.StatusCode {
			case 401:
				fmt.Println("Authentication failed - check your API keys")
			case 403:
				fmt.Println("Permission denied - check your API key permissions")
			case 404:
				fmt.Println("Resource not found")
			case 429:
				fmt.Println("Rate limit exceeded - try again later")
			default:
				fmt.Printf("API error (status %d): %s\n", ddErr.StatusCode, ddErr.Message)
				if len(ddErr.Errors) > 0 {
					fmt.Printf("Details: %v\n", ddErr.Errors)
				}
			}
		} else {
			// Network or other errors
			fmt.Printf("Request error: %v\n", err)
		}
		return
	}

	// Parse successful response
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		fmt.Printf("Failed to parse response: %v\n", err)
		return
	}

	fmt.Printf("Success: %+v\n", result)
}

// ExampleCreateIncident shows how to create an incident
func ExampleCreateIncident() {
	client, err := NewClient()
	if err != nil {
		fmt.Printf("Failed to create client: %v\n", err)
		return
	}

	// Create incident with custom fields
	fields := map[string]interface{}{
		"severity": "SEV-1",
		"services": []string{"web-app", "api"},
		"teams":    []string{"backend", "platform"},
	}

	data, err := client.CreateIncident(
		"High error rate in production",
		true, // customer_impacted
		fields,
	)

	if err != nil {
		fmt.Printf("Failed to create incident: %v\n", err)
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		fmt.Printf("Failed to parse incident response: %v\n", err)
		return
	}

	fmt.Printf("Incident created: %+v\n", result)
}
