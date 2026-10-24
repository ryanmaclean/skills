# Integration Guide

How to integrate the Datadog client into CLI commands.

## Step 1: Import the Client

In your command handlers (e.g., `cmd/apm.go`, `cmd/logs.go`):

```go
package main

import (
    "encoding/json"
    "fmt"
    "time"

    "github.com/datadog/skill/internal/client"
)
```

## Step 2: Create Client Instance

Create the client in your command handler:

```go
func handleAPMCommand(args []string) error {
    // Create Datadog client
    ddClient, err := client.NewClient()
    if err != nil {
        return fmt.Errorf("failed to initialize Datadog client: %w", err)
    }

    // Use the client...
    return nil
}
```

## Step 3: Execute Queries

Use the client to query Datadog APIs:

```go
func handleAPMCommand(service string) error {
    ddClient, err := client.NewClient()
    if err != nil {
        return err
    }

    // Query last hour of traces
    to := time.Now()
    from := to.Add(-1 * time.Hour)

    data, err := ddClient.QueryAPM(service, from, to, "")
    if err != nil {
        return fmt.Errorf("APM query failed: %w", err)
    }

    // Parse and display results
    var result map[string]interface{}
    if err := json.Unmarshal(data, &result); err != nil {
        return fmt.Errorf("failed to parse response: %w", err)
    }

    fmt.Printf("APM data for %s:\n%+v\n", service, result)
    return nil
}
```

## Example Command Integration

### APM Command (`cmd/apm.go`)

```go
package main

import (
    "encoding/json"
    "flag"
    "fmt"
    "os"
    "time"

    "github.com/datadog/skill/internal/client"
)

func handleAPMCommand() {
    apmFlags := flag.NewFlagSet("apm", flag.ExitOnError)
    service := apmFlags.String("service", "", "Service name (required)")
    hours := apmFlags.Int("hours", 1, "Hours to look back")
    status := apmFlags.String("status", "", "Filter by status (e.g., error)")

    apmFlags.Parse(os.Args[2:])

    if *service == "" {
        fmt.Println("Error: --service is required")
        apmFlags.PrintDefaults()
        os.Exit(1)
    }

    // Create client
    ddClient, err := client.NewClient()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }

    // Build time range
    to := time.Now()
    from := to.Add(-time.Duration(*hours) * time.Hour)

    // Build filter
    filter := ""
    if *status != "" {
        filter = fmt.Sprintf("status:%s", *status)
    }

    // Query APM
    data, err := ddClient.QueryAPM(*service, from, to, filter)
    if err != nil {
        fmt.Fprintf(os.Stderr, "APM query failed: %v\n", err)
        os.Exit(1)
    }

    // Parse and display
    var result map[string]interface{}
    if err := json.Unmarshal(data, &result); err != nil {
        fmt.Fprintf(os.Stderr, "Failed to parse response: %v\n", err)
        os.Exit(1)
    }

    // Pretty print JSON
    prettyJSON, _ := json.MarshalIndent(result, "", "  ")
    fmt.Println(string(prettyJSON))
}
```

### Logs Command (`cmd/logs.go`)

```go
package main

import (
    "encoding/json"
    "flag"
    "fmt"
    "os"
    "time"

    "github.com/datadog/skill/internal/client"
)

func handleLogsCommand() {
    logsFlags := flag.NewFlagSet("logs", flag.ExitOnError)
    query := logsFlags.String("query", "", "Log query (required)")
    hours := logsFlags.Int("hours", 1, "Hours to look back")
    limit := logsFlags.Int("limit", 100, "Maximum logs to return")

    logsFlags.Parse(os.Args[2:])

    if *query == "" {
        fmt.Println("Error: --query is required")
        logsFlags.PrintDefaults()
        os.Exit(1)
    }

    // Create client
    ddClient, err := client.NewClient()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }

    // Build time range
    to := time.Now()
    from := to.Add(-time.Duration(*hours) * time.Hour)

    // Search logs
    data, err := ddClient.SearchLogs(*query, from, to, *limit)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Log search failed: %v\n", err)
        os.Exit(1)
    }

    // Parse and display
    var result map[string]interface{}
    if err := json.Unmarshal(data, &result); err != nil {
        fmt.Fprintf(os.Stderr, "Failed to parse response: %v\n", err)
        os.Exit(1)
    }

    // Extract and display logs
    if logsData, ok := result["data"].([]interface{}); ok {
        fmt.Printf("Found %d logs:\n\n", len(logsData))
        for i, log := range logsData {
            fmt.Printf("--- Log %d ---\n", i+1)
            logJSON, _ := json.MarshalIndent(log, "", "  ")
            fmt.Println(string(logJSON))
        }
    }
}
```

### Metrics Command (`cmd/metrics.go`)

```go
package main

import (
    "encoding/json"
    "flag"
    "fmt"
    "os"
    "time"

    "github.com/datadog/skill/internal/client"
)

func handleMetricsCommand() {
    metricsFlags := flag.NewFlagSet("metrics", flag.ExitOnError)
    query := metricsFlags.String("query", "", "Metric query (required)")
    hours := metricsFlags.Int("hours", 1, "Hours to look back")

    metricsFlags.Parse(os.Args[2:])

    if *query == "" {
        fmt.Println("Error: --query is required")
        fmt.Println("Example: --query 'avg:system.cpu.user{service:my-service}'")
        metricsFlags.PrintDefaults()
        os.Exit(1)
    }

    // Create client
    ddClient, err := client.NewClient()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }

    // Build time range
    to := time.Now()
    from := to.Add(-time.Duration(*hours) * time.Hour)

    // Query metrics
    data, err := ddClient.QueryMetrics(*query, from, to)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Metrics query failed: %v\n", err)
        os.Exit(1)
    }

    // Parse and display
    var result map[string]interface{}
    if err := json.Unmarshal(data, &result); err != nil {
        fmt.Fprintf(os.Stderr, "Failed to parse response: %v\n", err)
        os.Exit(1)
    }

    // Pretty print JSON
    prettyJSON, _ := json.MarshalIndent(result, "", "  ")
    fmt.Println(string(prettyJSON))
}
```

## Update Main Router

In `cmd/main.go`, add the new command handlers:

```go
package main

import (
    "fmt"
    "os"
)

const version = "0.1.0"

func main() {
    if len(os.Args) < 2 {
        printHelp()
        os.Exit(1)
    }

    command := os.Args[1]

    switch command {
    case "version", "--version", "-v":
        fmt.Printf("dd version %s\n", version)
    case "help", "--help", "-h":
        printHelp()
    case "apm":
        handleAPMCommand()
    case "logs":
        handleLogsCommand()
    case "metrics":
        handleMetricsCommand()
    case "context":
        fmt.Println("Context detection - coming soon")
    default:
        fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
        printHelp()
        os.Exit(1)
    }
}
```

## Error Handling Pattern

Always handle errors gracefully with helpful messages:

```go
func executeQuery(ddClient *client.Client) error {
    data, err := ddClient.QueryAPM("my-service", from, to, "")
    if err != nil {
        // Check for specific error types
        if ddErr, ok := err.(*client.DatadogError); ok {
            switch ddErr.StatusCode {
            case 401:
                return fmt.Errorf("authentication failed - check DD_API_KEY and DD_APP_KEY")
            case 403:
                return fmt.Errorf("permission denied - verify API key permissions")
            case 404:
                return fmt.Errorf("service not found - check service name")
            case 429:
                return fmt.Errorf("rate limited - please wait and retry")
            default:
                return fmt.Errorf("API error (%d): %s", ddErr.StatusCode, ddErr.Message)
            }
        }
        return fmt.Errorf("request failed: %w", err)
    }

    // Process data...
    return nil
}
```

## Testing Integration

Create integration tests that verify the client works with your commands:

```go
package main

import (
    "os"
    "testing"

    "github.com/datadog/skill/internal/client"
)

func TestClientIntegration(t *testing.T) {
    // Skip if credentials not set
    if os.Getenv("DD_API_KEY") == "" {
        t.Skip("DD_API_KEY not set, skipping integration test")
    }

    ddClient, err := client.NewClient()
    if err != nil {
        t.Fatalf("Failed to create client: %v", err)
    }

    // Test service catalog (doesn't require parameters)
    _, err = ddClient.GetServiceCatalog()
    if err != nil {
        t.Errorf("GetServiceCatalog failed: %v", err)
    }
}
```

## Environment Setup

Document required environment variables in your README:

```bash
# Required for Datadog API access
export DD_API_KEY="your_api_key_here"
export DD_APP_KEY="your_app_key_here"

# Optional - defaults to datadoghq.com
export DD_SITE="datadoghq.com"

# For EU region
export DD_SITE="datadoghq.eu"
```

## Usage Examples

After integration, users can run:

```bash
# Query APM traces
dd apm --service my-service --hours 2 --status error

# Search logs
dd logs --query "service:my-service ERROR" --hours 1 --limit 50

# Query metrics
dd metrics --query "avg:system.cpu.user{service:my-service}" --hours 2

# Get service catalog
dd catalog
```
