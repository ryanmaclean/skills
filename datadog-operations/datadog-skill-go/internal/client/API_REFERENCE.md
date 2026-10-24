# Datadog Client API Reference

Complete API reference for the Datadog Go client.

## Types

### Client

The main Datadog API client.

```go
type Client struct {
    apiKey     string
    appKey     string
    site       string
    baseURL    string
    httpClient *http.Client
}
```

### DatadogError

Custom error type for Datadog API errors.

```go
type DatadogError struct {
    StatusCode int      // HTTP status code
    Message    string   // Error message
    Errors     []string // Detailed error messages
}
```

**Methods:**
- `Error() string` - Returns formatted error message

## Constructor

### NewClient

Creates a new Datadog API client from environment variables.

```go
func NewClient() (*Client, error)
```

**Environment Variables:**
- `DD_API_KEY` - Datadog API key (required)
- `DD_APP_KEY` - Datadog application key (required)
- `DD_SITE` - Datadog site (optional, defaults to "datadoghq.com")

**Returns:**
- `*Client` - Configured client instance
- `error` - Error if credentials are missing

**Example:**
```go
client, err := client.NewClient()
if err != nil {
    log.Fatal(err)
}
```

## Core Methods

### QueryAPM

Query APM trace analytics for a service.

```go
func (c *Client) QueryAPM(service string, from, to time.Time, filter string) ([]byte, error)
```

**Parameters:**
- `service` - Service name to query
- `from` - Start time
- `to` - End time
- `filter` - Additional filter (e.g., "status:error")

**Returns:**
- `[]byte` - Raw JSON response
- `error` - Error if request fails

**Example:**
```go
from := time.Now().Add(-1 * time.Hour)
to := time.Now()
data, err := client.QueryAPM("my-service", from, to, "status:error")
```

**Response includes:**
- Request count
- P50, P95, P99 latency percentiles
- Grouped by resource name

---

### SearchLogs

Search logs with a query.

```go
func (c *Client) SearchLogs(query string, from, to time.Time, limit int) ([]byte, error)
```

**Parameters:**
- `query` - Log search query (e.g., "service:my-service ERROR")
- `from` - Start time
- `to` - End time
- `limit` - Maximum number of logs to return (defaults to 100 if <= 0)

**Returns:**
- `[]byte` - Raw JSON response with log events
- `error` - Error if request fails

**Example:**
```go
from := time.Now().Add(-1 * time.Hour)
to := time.Now()
data, err := client.SearchLogs("service:web-app status:error", from, to, 50)
```

---

### QueryMetrics

Query metrics with a metric query.

```go
func (c *Client) QueryMetrics(query string, from, to time.Time) ([]byte, error)
```

**Parameters:**
- `query` - Metric query (e.g., "avg:system.cpu.user{*}")
- `from` - Start time
- `to` - End time

**Returns:**
- `[]byte` - Raw JSON response with metric data
- `error` - Error if request fails

**Example:**
```go
from := time.Now().Add(-1 * time.Hour)
to := time.Now()
data, err := client.QueryMetrics("avg:system.cpu.user{service:my-service}", from, to)
```

---

### GetServiceCatalog

Retrieve the service catalog.

```go
func (c *Client) GetServiceCatalog() ([]byte, error)
```

**Returns:**
- `[]byte` - Raw JSON response with service definitions
- `error` - Error if request fails

**Example:**
```go
data, err := client.GetServiceCatalog()
```

## Security Methods

### GetSecuritySignals

Retrieve security monitoring signals.

```go
func (c *Client) GetSecuritySignals(from, to time.Time, service string) ([]byte, error)
```

**Parameters:**
- `from` - Start time
- `to` - End time
- `service` - Service name to filter (empty for all services)

**Returns:**
- `[]byte` - Raw JSON response with security signals
- `error` - Error if request fails

**Example:**
```go
from := time.Now().Add(-24 * time.Hour)
to := time.Now()
data, err := client.GetSecuritySignals(from, to, "my-service")
```

## SLO Methods

### GetSLOs

Retrieve SLOs, optionally filtered by tags.

```go
func (c *Client) GetSLOs(tags []string) ([]byte, error)
```

**Parameters:**
- `tags` - Tag filters (optional, nil for all SLOs)

**Returns:**
- `[]byte` - Raw JSON response with SLO data
- `error` - Error if request fails

**Example:**
```go
// Get all SLOs
data, err := client.GetSLOs(nil)

// Get SLOs with tags
data, err := client.GetSLOs([]string{"team:backend", "env:production"})
```

---

### GetSLOHistory

Retrieve SLO history for error budget calculation.

```go
func (c *Client) GetSLOHistory(sloID string, from, to time.Time) ([]byte, error)
```

**Parameters:**
- `sloID` - SLO identifier
- `from` - Start time
- `to` - End time

**Returns:**
- `[]byte` - Raw JSON response with SLO history
- `error` - Error if request fails

**Example:**
```go
from := time.Now().Add(-7 * 24 * time.Hour)
to := time.Now()
data, err := client.GetSLOHistory("abc123", from, to)
```

## Monitor Methods

### GetMonitors

Retrieve monitors, optionally filtered by tags.

```go
func (c *Client) GetMonitors(tags, monitorTags []string) ([]byte, error)
```

**Parameters:**
- `tags` - Resource tag filters (optional)
- `monitorTags` - Monitor tag filters (optional)

**Returns:**
- `[]byte` - Raw JSON response with monitor data
- `error` - Error if request fails

**Example:**
```go
// Get all monitors
data, err := client.GetMonitors(nil, nil)

// Get monitors with filters
data, err := client.GetMonitors(
    []string{"env:production"},
    []string{"service:my-service"},
)
```

## Incident Methods

### CreateIncident

Create an incident in Datadog.

```go
func (c *Client) CreateIncident(title string, customerImpacted bool, fields map[string]interface{}) ([]byte, error)
```

**Parameters:**
- `title` - Incident title
- `customerImpacted` - Whether customers are impacted
- `fields` - Additional incident fields (optional)

**Returns:**
- `[]byte` - Raw JSON response with created incident
- `error` - Error if request fails

**Example:**
```go
fields := map[string]interface{}{
    "severity": "SEV-1",
    "services": []string{"web-app", "api"},
    "teams":    []string{"backend"},
}

data, err := client.CreateIncident(
    "High error rate in production",
    true,
    fields,
)
```

## Helper Methods

### buildHeaders

Creates HTTP headers with authentication (internal use).

```go
func (c *Client) buildHeaders() http.Header
```

**Returns:**
- `http.Header` - Headers with API key, App key, and Content-Type

---

### doRequest

Makes an HTTP request with retry logic and error handling (internal use).

```go
func (c *Client) doRequest(method, endpoint string, body interface{}) ([]byte, error)
```

**Parameters:**
- `method` - HTTP method (GET, POST, etc.)
- `endpoint` - API endpoint path
- `body` - Request body (marshaled to JSON)

**Returns:**
- `[]byte` - Raw response body
- `error` - Error if request fails

**Features:**
- Automatic retry on failures (max 3 attempts)
- Exponential backoff (1s, 2s, 3s)
- Rate limit handling (429 status)
- Server error retry (5xx status)
- 30-second timeout per request

---

### parseError

Parses error responses from the Datadog API (internal use).

```go
func (c *Client) parseError(statusCode int, body []byte) error
```

**Parameters:**
- `statusCode` - HTTP status code
- `body` - Response body

**Returns:**
- `error` - Parsed DatadogError

## Error Handling

All methods return errors that can be type-asserted to `*DatadogError` for detailed information:

```go
data, err := client.QueryAPM("my-service", from, to, "")
if err != nil {
    if ddErr, ok := err.(*client.DatadogError); ok {
        fmt.Printf("Status: %d\n", ddErr.StatusCode)
        fmt.Printf("Message: %s\n", ddErr.Message)
        fmt.Printf("Errors: %v\n", ddErr.Errors)
    } else {
        fmt.Printf("Other error: %v\n", err)
    }
}
```

### Common HTTP Status Codes

- `200-299` - Success
- `400` - Bad Request (invalid parameters)
- `401` - Unauthorized (invalid API key)
- `403` - Forbidden (insufficient permissions)
- `404` - Not Found (resource doesn't exist)
- `429` - Too Many Requests (rate limited)
- `500-599` - Server Error (Datadog API issue)

## Response Parsing

All methods return raw JSON bytes. Parse them based on your needs:

```go
// Parse into map
var result map[string]interface{}
err := json.Unmarshal(data, &result)

// Parse into specific struct
type APMResponse struct {
    Data []struct {
        Attributes struct {
            Count int `json:"count"`
        } `json:"attributes"`
    } `json:"data"`
}
var apmResult APMResponse
err := json.Unmarshal(data, &apmResult)
```

## Thread Safety

The client is safe for concurrent use. Multiple goroutines can share the same client instance:

```go
client, _ := client.NewClient()

go func() {
    client.QueryAPM("service-1", from, to, "")
}()

go func() {
    client.SearchLogs("service:service-2", from, to, 100)
}()
```

## Timeouts

- Default HTTP timeout: 30 seconds per request
- Retry delays: 1s, 2s, 3s (exponential backoff)
- Maximum retries: 3 attempts
