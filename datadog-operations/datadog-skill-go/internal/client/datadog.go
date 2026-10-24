package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

// Client represents a Datadog API client
type Client struct {
	apiKey     string
	appKey     string
	site       string
	baseURL    string
	httpClient *http.Client
}

// DatadogError represents an error response from the Datadog API
type DatadogError struct {
	StatusCode int
	Message    string
	Errors     []string
}

func (e *DatadogError) Error() string {
	if len(e.Errors) > 0 {
		return fmt.Sprintf("datadog api error (status %d): %s - %v", e.StatusCode, e.Message, e.Errors)
	}
	return fmt.Sprintf("datadog api error (status %d): %s", e.StatusCode, e.Message)
}

// NewClient creates a new Datadog API client from environment variables
func NewClient() (*Client, error) {
	apiKey := os.Getenv("DD_API_KEY")
	appKey := os.Getenv("DD_APP_KEY")
	site := os.Getenv("DD_SITE")

	if apiKey == "" {
		return nil, fmt.Errorf("DD_API_KEY environment variable is required")
	}

	if appKey == "" {
		return nil, fmt.Errorf("DD_APP_KEY environment variable is required")
	}

	if site == "" {
		site = "datadoghq.com"
	}

	baseURL := fmt.Sprintf("https://api.%s", site)

	return &Client{
		apiKey:  apiKey,
		appKey:  appKey,
		site:    site,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// buildHeaders creates HTTP headers with authentication
func (c *Client) buildHeaders() http.Header {
	headers := http.Header{}
	headers.Set("DD-API-KEY", c.apiKey)
	headers.Set("DD-APPLICATION-KEY", c.appKey)
	headers.Set("Content-Type", "application/json")
	return headers
}

// validateResourceID validates that a resource ID is safe to use in URLs
func validateResourceID(id string) error {
	if id == "" {
		return fmt.Errorf("resource ID cannot be empty")
	}
	// Check for path traversal attempts
	if len(id) > 0 && (id[0] == '/' || id[0] == '\\') {
		return fmt.Errorf("resource ID cannot start with path separator")
	}
	// Check for path traversal patterns
	for i := 0; i < len(id)-1; i++ {
		if (id[i] == '.' && id[i+1] == '.') {
			return fmt.Errorf("resource ID cannot contain '..' (path traversal)")
		}
		if (id[i] == '/' && id[i+1] == '/') {
			return fmt.Errorf("resource ID cannot contain '//' (double slash)")
		}
	}
	return nil
}

// DoRequest makes an HTTP request with retry logic and error handling
func (c *Client) DoRequest(method, endpoint string, body interface{}) ([]byte, error) {
	url := c.baseURL + endpoint
	maxRetries := 3
	retryDelay := time.Second

	var requestBody []byte
	var err error

	if body != nil {
		requestBody, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	for attempt := 0; attempt < maxRetries; attempt++ {
		var req *http.Request
		if len(requestBody) > 0 {
			req, err = http.NewRequest(method, url, bytes.NewReader(requestBody))
		} else {
			req, err = http.NewRequest(method, url, nil)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header = c.buildHeaders()

		resp, err := c.httpClient.Do(req)
		if err != nil {
			if attempt < maxRetries-1 {
				time.Sleep(retryDelay * time.Duration(attempt+1))
				continue
			}
			return nil, fmt.Errorf("request failed after %d attempts: %w", maxRetries, err)
		}

		defer resp.Body.Close()

		// Handle rate limiting
		if resp.StatusCode == http.StatusTooManyRequests {
			if attempt < maxRetries-1 {
				time.Sleep(retryDelay * time.Duration(attempt+1))
				continue
			}
			return nil, &DatadogError{
				StatusCode: resp.StatusCode,
				Message:    "rate limit exceeded",
			}
		}

		// Read response body
		responseBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		// Handle non-2xx status codes
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			// Retry on 5xx errors
			if resp.StatusCode >= 500 && attempt < maxRetries-1 {
				time.Sleep(retryDelay)
				continue
			}
			return nil, c.parseError(resp.StatusCode, responseBody)
		}

		return responseBody, nil
	}

	return nil, fmt.Errorf("request failed after %d attempts", maxRetries)
}

// parseError parses error responses from the Datadog API
func (c *Client) parseError(statusCode int, body []byte) error {
	var errorResp struct {
		Errors []string `json:"errors"`
		Error  string   `json:"error"`
	}

	if err := json.Unmarshal(body, &errorResp); err != nil {
		return &DatadogError{
			StatusCode: statusCode,
			Message:    string(body),
		}
	}

	message := errorResp.Error
	if message == "" && len(errorResp.Errors) > 0 {
		message = errorResp.Errors[0]
	}

	return &DatadogError{
		StatusCode: statusCode,
		Message:    message,
		Errors:     errorResp.Errors,
	}
}

// QueryAPM queries APM trace analytics for a service
func (c *Client) QueryAPM(service string, from, to time.Time, filter string) ([]byte, error) {
	query := fmt.Sprintf("service:%s", service)
	if filter != "" {
		query += " " + filter
	}

	// API v2 requires data.attributes wrapper per official documentation
	// https://docs.datadoghq.com/api/latest/spans/
	// IMPORTANT: type must come before attributes in JSON
	type DataAttributes struct {
		Filter  map[string]interface{}   `json:"filter"`
		Compute []map[string]interface{} `json:"compute"`
		GroupBy []map[string]interface{} `json:"group_by"`
	}

	type AggregateData struct {
		Type       string         `json:"type"`
		Attributes DataAttributes `json:"attributes"`
	}

	type AggregateRequest struct {
		Data AggregateData `json:"data"`
	}

	payload := AggregateRequest{
		Data: AggregateData{
			Type: "aggregate_request",
			Attributes: DataAttributes{
				Filter: map[string]interface{}{
					"from":  from.Format(time.RFC3339),
					"to":    to.Format(time.RFC3339),
					"query": query,
				},
				Compute: []map[string]interface{}{
					{
						"aggregation": "count",
						"type":        "total",
					},
					{
						"aggregation": "median", // Replaces pc50 (which doesn't exist)
						"metric":      "@duration",
						"type":        "total",
					},
					{
						"aggregation": "pc95",
						"metric":      "@duration",
						"type":        "total",
					},
					{
						"aggregation": "pc99",
						"metric":      "@duration",
						"type":        "total",
					},
				},
				GroupBy: []map[string]interface{}{
					{
						"facet": "resource_name",
						"limit": 20,
						// Note: sort removed - "aggregation" field in sort causes API validation error
					},
				},
			},
		},
	}

	return c.DoRequest("POST", "/api/v2/spans/analytics/aggregate", &payload)
}

// SearchLogs searches logs with the given query
func (c *Client) SearchLogs(query string, from, to time.Time, limit int) ([]byte, error) {
	if limit <= 0 {
		limit = 100
	}

	payload := map[string]interface{}{
		"filter": map[string]interface{}{
			"query": query,
			"from":  from.Format(time.RFC3339),
			"to":    to.Format(time.RFC3339),
		},
		"sort": "timestamp",
		"page": map[string]interface{}{
			"limit": limit,
		},
	}

	return c.DoRequest("POST", "/api/v2/logs/events/search", payload)
}

// QueryMetrics queries metrics with the given query
func (c *Client) QueryMetrics(query string, from, to time.Time) ([]byte, error) {
	fromTs := from.Unix()
	toTs := to.Unix()

	endpoint := fmt.Sprintf("/api/v1/query?query=%s&from=%d&to=%d", url.QueryEscape(query), fromTs, toTs)

	return c.DoRequest("GET", endpoint, nil)
}

// GetServiceCatalog retrieves the service catalog
func (c *Client) GetServiceCatalog() ([]byte, error) {
	return c.DoRequest("GET", "/api/v2/services/definitions", nil)
}

// GetSecuritySignals retrieves security monitoring signals
func (c *Client) GetSecuritySignals(from, to time.Time, service string) ([]byte, error) {
	query := ""
	if service != "" {
		query = fmt.Sprintf("service:%s", service)
	}

	payload := map[string]interface{}{
		"filter": map[string]interface{}{
			"query": query,
			"from":  from.Format(time.RFC3339),
			"to":    to.Format(time.RFC3339),
		},
		"sort": "-timestamp",
		"page": map[string]interface{}{
			"limit": 100,
		},
	}

	return c.DoRequest("POST", "/api/v2/security_monitoring/signals/search", payload)
}

// GetSLOs retrieves SLOs, optionally filtered by tags
func (c *Client) GetSLOs(tags []string) ([]byte, error) {
	endpoint := "/api/v1/slo"

	if len(tags) > 0 {
		tagsQuery := ""
		for i, tag := range tags {
			if i > 0 {
				tagsQuery += ","
			}
			tagsQuery += tag
		}
		endpoint += "?tags_query=" + url.QueryEscape(tagsQuery)
	}

	return c.DoRequest("GET", endpoint, nil)
}

// GetSLOHistory retrieves SLO history for error budget calculation
func (c *Client) GetSLOHistory(sloID string, from, to time.Time) ([]byte, error) {
	fromTs := from.Unix()
	toTs := to.Unix()

	endpoint := fmt.Sprintf("/api/v1/slo/%s/history?from_ts=%d&to_ts=%d", url.PathEscape(sloID), fromTs, toTs)

	return c.DoRequest("GET", endpoint, nil)
}

// GetMonitors retrieves monitors, optionally filtered by tags
func (c *Client) GetMonitors(tags, monitorTags []string) ([]byte, error) {
	endpoint := "/api/v1/monitor"
	params := ""

	if len(tags) > 0 {
		tagsStr := ""
		for i, tag := range tags {
			if i > 0 {
				tagsStr += ","
			}
			tagsStr += tag
		}
		params += "tags=" + tagsStr
	}

	if len(monitorTags) > 0 {
		monitorTagsStr := ""
		for i, tag := range monitorTags {
			if i > 0 {
				monitorTagsStr += ","
			}
			monitorTagsStr += tag
		}
		if params != "" {
			params += "&"
		}
		params += "monitor_tags=" + monitorTagsStr
	}

	if params != "" {
		endpoint += "?" + params
	}

	return c.DoRequest("GET", endpoint, nil)
}

// CreateIncident creates an incident in Datadog
func (c *Client) CreateIncident(title string, customerImpacted bool, fields map[string]interface{}) ([]byte, error) {
	if fields == nil {
		fields = make(map[string]interface{})
	}

	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "incidents",
			"attributes": map[string]interface{}{
				"title":             title,
				"customer_impacted": customerImpacted,
				"fields":            fields,
			},
		},
	}

	return c.DoRequest("POST", "/api/v2/incidents", payload)
}

// CreateIncidentFull creates an incident with full attributes
func (c *Client) CreateIncidentFull(title, severity string, customerImpacted bool, fields map[string]interface{}) ([]byte, error) {
	if fields == nil {
		fields = make(map[string]interface{})
	}

	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "incidents",
			"attributes": map[string]interface{}{
				"title":             title,
				"severity":          severity,
				"customer_impacted": customerImpacted,
				"fields":            fields,
			},
		},
	}

	return c.DoRequest("POST", "/api/v2/incidents", payload)
}

// ListIncidents retrieves incidents, optionally filtered by status and service
func (c *Client) ListIncidents(status, service string, limit int) ([]byte, error) {
	endpoint := "/api/v2/incidents"
	params := ""

	if status != "" {
		params += fmt.Sprintf("filter[state]=%s", status)
	}

	if service != "" {
		if params != "" {
			params += "&"
		}
		params += fmt.Sprintf("filter[service]=%s", service)
	}

	if limit > 0 {
		if params != "" {
			params += "&"
		}
		params += fmt.Sprintf("page[limit]=%d", limit)
	}

	if params != "" {
		endpoint += "?" + params
	}

	return c.DoRequest("GET", endpoint, nil)
}

// UpdateIncident updates an incident's status
func (c *Client) UpdateIncident(incidentID, newStatus string) ([]byte, error) {
	if err := validateResourceID(incidentID); err != nil {
		return nil, fmt.Errorf("invalid incident ID: %w", err)
	}

	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "incidents",
			"id":   incidentID,
			"attributes": map[string]interface{}{
				"state": newStatus,
			},
		},
	}

	endpoint := fmt.Sprintf("/api/v2/incidents/%s", url.PathEscape(incidentID))
	return c.DoRequest("PATCH", endpoint, payload)
}

// AddIncidentTimeline adds a timeline entry to an incident
func (c *Client) AddIncidentTimeline(incidentID, message string) error {
	if err := validateResourceID(incidentID); err != nil {
		return fmt.Errorf("invalid incident ID: %w", err)
	}

	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "incident_timeline",
			"attributes": map[string]interface{}{
				"content": map[string]interface{}{
					"content_type": "text",
					"message":      message,
				},
			},
		},
	}

	endpoint := fmt.Sprintf("/api/v2/incidents/%s/timeline", url.PathEscape(incidentID))
	_, err := c.DoRequest("POST", endpoint, payload)
	return err
}

// CreateMonitor creates a new monitor
func (c *Client) CreateMonitor(payload map[string]interface{}) ([]byte, error) {
	return c.DoRequest("POST", "/api/v1/monitor", payload)
}

// MuteMonitor mutes a monitor
func (c *Client) MuteMonitor(monitorID string, payload map[string]interface{}) ([]byte, error) {
	if err := validateResourceID(monitorID); err != nil {
		return nil, fmt.Errorf("invalid monitor ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v1/monitor/%s/mute", url.PathEscape(monitorID))
	return c.DoRequest("POST", endpoint, payload)
}

// UnmuteMonitor unmutes a monitor
func (c *Client) UnmuteMonitor(monitorID string, payload map[string]interface{}) ([]byte, error) {
	if err := validateResourceID(monitorID); err != nil {
		return nil, fmt.Errorf("invalid monitor ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v1/monitor/%s/unmute", url.PathEscape(monitorID))
	return c.DoRequest("POST", endpoint, payload)
}

// DeleteMonitor deletes a monitor
func (c *Client) DeleteMonitor(monitorID string) ([]byte, error) {
	if err := validateResourceID(monitorID); err != nil {
		return nil, fmt.Errorf("invalid monitor ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v1/monitor/%s", url.PathEscape(monitorID))
	return c.DoRequest("DELETE", endpoint, nil)
}

// WatchdogAlerts retrieves Watchdog anomaly alerts via Events API
func (c *Client) WatchdogAlerts(query string, from, to time.Time, limit int) ([]byte, error) {
	if limit <= 0 {
		limit = 100
	}

	payload := map[string]interface{}{
		"filter": map[string]interface{}{
			"query": query,
			"from":  from.Format(time.RFC3339),
			"to":    to.Format(time.RFC3339),
		},
		"page": map[string]interface{}{
			"limit": limit,
		},
		"sort": "timestamp",
	}

	return c.DoRequest("POST", "/api/v2/events/search", payload)
}

// QueryLLMSpans queries LLM observability data for GenAI applications
func (c *Client) QueryLLMSpans(payload interface{}) ([]byte, error) {
	return c.DoRequest("POST", "/api/v2/spans/analytics/aggregate", payload)
}

// ListDashboards retrieves all dashboards
func (c *Client) ListDashboards() ([]byte, error) {
	return c.DoRequest("GET", "/api/v1/dashboard", nil)
}

// GetDashboard retrieves a specific dashboard by ID
func (c *Client) GetDashboard(id string) ([]byte, error) {
	if err := validateResourceID(id); err != nil {
		return nil, fmt.Errorf("invalid dashboard ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v1/dashboard/%s", url.PathEscape(id))
	return c.DoRequest("GET", endpoint, nil)
}

// CreateDashboard creates a new dashboard
func (c *Client) CreateDashboard(payload map[string]interface{}) ([]byte, error) {
	return c.DoRequest("POST", "/api/v1/dashboard", payload)
}

// UpdateDashboard updates an existing dashboard
func (c *Client) UpdateDashboard(id string, payload map[string]interface{}) ([]byte, error) {
	if err := validateResourceID(id); err != nil {
		return nil, fmt.Errorf("invalid dashboard ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v1/dashboard/%s", url.PathEscape(id))
	return c.DoRequest("PUT", endpoint, payload)
}

// DeleteDashboard deletes a dashboard
func (c *Client) DeleteDashboard(id string) error {
	if err := validateResourceID(id); err != nil {
		return fmt.Errorf("invalid dashboard ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v1/dashboard/%s", url.PathEscape(id))
	_, err := c.DoRequest("DELETE", endpoint, nil)
	return err
}

// ListWorkflows retrieves all workflows
func (c *Client) ListWorkflows() ([]byte, error) {
	return c.DoRequest("GET", "/api/v2/workflows", nil)
}

// GetWorkflow retrieves a specific workflow by ID
func (c *Client) GetWorkflow(id string) ([]byte, error) {
	if err := validateResourceID(id); err != nil {
		return nil, fmt.Errorf("invalid workflow ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/workflows/%s", url.PathEscape(id))
	return c.DoRequest("GET", endpoint, nil)
}

// ExecuteWorkflow executes a workflow with optional input parameters
func (c *Client) ExecuteWorkflow(id string, params map[string]interface{}) ([]byte, error) {
	if err := validateResourceID(id); err != nil {
		return nil, fmt.Errorf("invalid workflow ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/workflows/%s/execute", url.PathEscape(id))

	payload := map[string]interface{}{}
	if params != nil && len(params) > 0 {
		payload["input_params"] = params
	}

	return c.DoRequest("POST", endpoint, payload)
}

// GetWorkflowExecution retrieves the status of a workflow execution
func (c *Client) GetWorkflowExecution(executionID string) ([]byte, error) {
	if err := validateResourceID(executionID); err != nil {
		return nil, fmt.Errorf("invalid execution ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/workflows/executions/%s", url.PathEscape(executionID))
	return c.DoRequest("GET", endpoint, nil)
}

// CreateWorkflow creates a new workflow
func (c *Client) CreateWorkflow(payload map[string]interface{}) ([]byte, error) {
	workflowPayload := map[string]interface{}{
		"data": map[string]interface{}{
			"type":       "workflows",
			"attributes": payload,
		},
	}

	return c.DoRequest("POST", "/api/v2/workflows", workflowPayload)
}

// UpdateWorkflow updates an existing workflow
func (c *Client) UpdateWorkflow(id string, payload map[string]interface{}) ([]byte, error) {
	if err := validateResourceID(id); err != nil {
		return nil, fmt.Errorf("invalid workflow ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/workflows/%s", url.PathEscape(id))

	workflowPayload := map[string]interface{}{
		"data": map[string]interface{}{
			"type":       "workflows",
			"id":         id,
			"attributes": payload,
		},
	}

	return c.DoRequest("PATCH", endpoint, workflowPayload)
}

// DeleteWorkflow deletes a workflow
func (c *Client) DeleteWorkflow(id string) error {
	if err := validateResourceID(id); err != nil {
		return fmt.Errorf("invalid workflow ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/workflows/%s", url.PathEscape(id))
	_, err := c.DoRequest("DELETE", endpoint, nil)
	return err
}

// ListSyntheticTests retrieves synthetic tests, optionally filtered by type
func (c *Client) ListSyntheticTests(testType string) ([]byte, error) {
	endpoint := "/api/v1/synthetics/tests"

	if testType != "" {
		endpoint += fmt.Sprintf("?type=%s", testType)
	}

	return c.DoRequest("GET", endpoint, nil)
}

// GetSyntheticTest retrieves a specific synthetic test by ID
func (c *Client) GetSyntheticTest(id string) ([]byte, error) {
	if err := validateResourceID(id); err != nil {
		return nil, fmt.Errorf("invalid synthetic test ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v1/synthetics/tests/%s", url.PathEscape(id))
	return c.DoRequest("GET", endpoint, nil)
}

// GetSyntheticResults retrieves test results for a synthetic test
func (c *Client) GetSyntheticResults(id string, from, to time.Time) ([]byte, error) {
	if err := validateResourceID(id); err != nil {
		return nil, fmt.Errorf("invalid synthetic test ID: %w", err)
	}
	fromTs := from.UnixMilli()
	toTs := to.UnixMilli()

	endpoint := fmt.Sprintf("/api/v1/synthetics/tests/%s/results?from_ts=%d&to_ts=%d", url.PathEscape(id), fromTs, toTs)
	return c.DoRequest("GET", endpoint, nil)
}

// CreateSyntheticTest creates a new synthetic test
func (c *Client) CreateSyntheticTest(payload map[string]interface{}) ([]byte, error) {
	return c.DoRequest("POST", "/api/v1/synthetics/tests", payload)
}

// UpdateSyntheticTest updates a synthetic test
func (c *Client) UpdateSyntheticTest(id string, payload map[string]interface{}) ([]byte, error) {
	if err := validateResourceID(id); err != nil {
		return nil, fmt.Errorf("invalid synthetic test ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v1/synthetics/tests/%s", url.PathEscape(id))
	return c.DoRequest("PUT", endpoint, payload)
}

// DeleteSyntheticTest deletes a synthetic test
func (c *Client) DeleteSyntheticTest(id string) error {
	if err := validateResourceID(id); err != nil {
		return fmt.Errorf("invalid synthetic test ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v1/synthetics/tests/%s", url.PathEscape(id))
	_, err := c.DoRequest("DELETE", endpoint, nil)
	return err
}

// PauseSyntheticTest pauses a synthetic test
func (c *Client) PauseSyntheticTest(id string) error {
	if err := validateResourceID(id); err != nil {
		return fmt.Errorf("invalid synthetic test ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v1/synthetics/tests/%s/status", url.PathEscape(id))
	payload := map[string]interface{}{
		"new_status": "paused",
	}
	_, err := c.DoRequest("PUT", endpoint, payload)
	return err
}

// ResumeSyntheticTest resumes a synthetic test
func (c *Client) ResumeSyntheticTest(id string) error {
	if err := validateResourceID(id); err != nil {
		return fmt.Errorf("invalid synthetic test ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v1/synthetics/tests/%s/status", url.PathEscape(id))
	payload := map[string]interface{}{
		"new_status": "live",
	}
	_, err := c.DoRequest("PUT", endpoint, payload)
	return err
}

// GetRUMApplications retrieves all RUM applications
func (c *Client) GetRUMApplications() ([]byte, error) {
	return c.DoRequest("GET", "/api/v2/rum/applications", nil)
}

// QueryRUMViews queries RUM view data for an application
func (c *Client) QueryRUMViews(applicationID string, from, to time.Time, filter string) ([]byte, error) {
	query := fmt.Sprintf("@type:view @application.id:%s", applicationID)
	if filter != "" {
		query += " " + filter
	}

	payload := map[string]interface{}{
		"filter": map[string]interface{}{
			"query": query,
			"from":  from.Format(time.RFC3339),
			"to":    to.Format(time.RFC3339),
		},
		"page": map[string]interface{}{
			"limit": 1000,
		},
		"sort": "timestamp",
	}

	return c.DoRequest("POST", "/api/v2/rum/events/search", payload)
}

// QueryRUMSessions queries RUM session data for an application
func (c *Client) QueryRUMSessions(applicationID string, from, to time.Time, filter string) ([]byte, error) {
	query := fmt.Sprintf("@type:session @application.id:%s", applicationID)
	if filter != "" {
		query += " " + filter
	}

	payload := map[string]interface{}{
		"filter": map[string]interface{}{
			"query": query,
			"from":  from.Format(time.RFC3339),
			"to":    to.Format(time.RFC3339),
		},
		"page": map[string]interface{}{
			"limit": 1000,
		},
		"sort": "timestamp",
	}

	return c.DoRequest("POST", "/api/v2/rum/events/search", payload)
}

// QueryRUMErrors queries RUM error data for an application
func (c *Client) QueryRUMErrors(applicationID string, from, to time.Time, filter string) ([]byte, error) {
	query := fmt.Sprintf("@type:error @application.id:%s", applicationID)
	if filter != "" {
		query += " " + filter
	}

	payload := map[string]interface{}{
		"filter": map[string]interface{}{
			"query": query,
			"from":  from.Format(time.RFC3339),
			"to":    to.Format(time.RFC3339),
		},
		"page": map[string]interface{}{
			"limit": 1000,
		},
		"sort": "timestamp",
	}

	return c.DoRequest("POST", "/api/v2/rum/events/search", payload)
}

// QueryRUMPerformance queries RUM performance metrics including Core Web Vitals
func (c *Client) QueryRUMPerformance(applicationID string, from, to time.Time, filter string) ([]byte, error) {
	query := fmt.Sprintf("@type:view @application.id:%s", applicationID)
	if filter != "" {
		query += " " + filter
	}

	payload := map[string]interface{}{
		"filter": map[string]interface{}{
			"query": query,
			"from":  from.Format(time.RFC3339),
			"to":    to.Format(time.RFC3339),
		},
		"page": map[string]interface{}{
			"limit": 1000,
		},
		"sort": "timestamp",
	}

	return c.DoRequest("POST", "/api/v2/rum/events/search", payload)
}

// QueryNetworkFlows queries network flow data
func (c *Client) QueryNetworkFlows(query, from, to string) ([]byte, error) {
	// Network flows can be queried through metrics API
	// Convert RFC3339 to Unix timestamp
	fromTime, err := time.Parse(time.RFC3339, from)
	if err != nil {
		return nil, fmt.Errorf("invalid from time: %w", err)
	}
	toTime, err := time.Parse(time.RFC3339, to)
	if err != nil {
		return nil, fmt.Errorf("invalid to time: %w", err)
	}

	fromTs := fromTime.Unix()
	toTs := toTime.Unix()

	endpoint := fmt.Sprintf("/api/v1/query?query=%s&from=%d&to=%d", url.QueryEscape(query), fromTs, toTs)
	return c.DoRequest("GET", endpoint, nil)
}

// QueryNetworkConnections queries network connection data between source and destination
func (c *Client) QueryNetworkConnections(source, destination, from, to string) ([]byte, error) {
	// Build query based on source and destination
	query := "avg:network.tcp.connections{*}"
	if source != "" && destination != "" {
		query = fmt.Sprintf("avg:network.tcp.connections{source:%s,dest:%s}", source, destination)
	} else if source != "" {
		query = fmt.Sprintf("avg:network.tcp.connections{source:%s}", source)
	} else if destination != "" {
		query = fmt.Sprintf("avg:network.tcp.connections{dest:%s}", destination)
	}

	// Convert RFC3339 to Unix timestamp
	fromTime, err := time.Parse(time.RFC3339, from)
	if err != nil {
		return nil, fmt.Errorf("invalid from time: %w", err)
	}
	toTime, err := time.Parse(time.RFC3339, to)
	if err != nil {
		return nil, fmt.Errorf("invalid to time: %w", err)
	}

	fromTs := fromTime.Unix()
	toTs := toTime.Unix()

	endpoint := fmt.Sprintf("/api/v1/query?query=%s&from=%d&to=%d", url.QueryEscape(query), fromTs, toTs)
	return c.DoRequest("GET", endpoint, nil)
}

// QueryDNSQueries queries DNS query statistics
func (c *Client) QueryDNSQueries(query, from, to string) ([]byte, error) {
	// Convert RFC3339 to Unix timestamp
	fromTime, err := time.Parse(time.RFC3339, from)
	if err != nil {
		return nil, fmt.Errorf("invalid from time: %w", err)
	}
	toTime, err := time.Parse(time.RFC3339, to)
	if err != nil {
		return nil, fmt.Errorf("invalid to time: %w", err)
	}

	fromTs := fromTime.Unix()
	toTs := toTime.Unix()

	endpoint := fmt.Sprintf("/api/v1/query?query=%s&from=%d&to=%d", url.QueryEscape(query), fromTs, toTs)
	return c.DoRequest("GET", endpoint, nil)
}

// GetNetworkMetrics queries general network metrics
func (c *Client) GetNetworkMetrics(query, from, to string) ([]byte, error) {
	// Convert RFC3339 to Unix timestamp
	fromTime, err := time.Parse(time.RFC3339, from)
	if err != nil {
		return nil, fmt.Errorf("invalid from time: %w", err)
	}
	toTime, err := time.Parse(time.RFC3339, to)
	if err != nil {
		return nil, fmt.Errorf("invalid to time: %w", err)
	}

	fromTs := fromTime.Unix()
	toTs := toTime.Unix()

	endpoint := fmt.Sprintf("/api/v1/query?query=%s&from=%d&to=%d", url.QueryEscape(query), fromTs, toTs)
	return c.DoRequest("GET", endpoint, nil)
}

// GetTopTalkers retrieves top network talkers by bandwidth
func (c *Client) GetTopTalkers(from, to time.Time, limit int) ([]byte, error) {
	if limit <= 0 {
		limit = 10
	}

	fromTs := from.Unix()
	toTs := to.Unix()

	// Query for top bandwidth consumers
	query := fmt.Sprintf("top(sum:network.bytes_sent{*} by {host}, %d, 'mean', 'desc')", limit)
	endpoint := fmt.Sprintf("/api/v1/query?query=%s&from=%d&to=%d", url.QueryEscape(query), fromTs, toTs)

	return c.DoRequest("GET", endpoint, nil)
}

// QueryCIPipelines queries CI pipeline analytics
func (c *Client) QueryCIPipelines(service string, from, to time.Time, filter string) ([]byte, error) {
	fromNs := from.UnixNano()
	toNs := to.UnixNano()

	query := ""
	if service != "" {
		query = fmt.Sprintf("service:%s", service)
	}
	if filter != "" {
		if query != "" {
			query += " AND "
		}
		query += filter
	}

	payload := map[string]interface{}{
		"filter": map[string]interface{}{
			"from":  fmt.Sprintf("%d", fromNs),
			"to":    fmt.Sprintf("%d", toNs),
			"query": query,
		},
		"compute": []map[string]interface{}{
			{"aggregation": "avg", "metric": "duration"},
			{"aggregation": "count", "metric": "*"},
		},
		"group_by": []map[string]interface{}{
			{
				"facet": "ci.pipeline.id",
				"limit": 50,
				"sort": map[string]interface{}{
					"order":       "desc",
					"aggregation": "count",
					"metric":      "*",
				},
			},
			{"facet": "status"},
			{"facet": "git.branch"},
			{"facet": "git.repository"},
			{"facet": "git.commit.sha"},
		},
	}

	return c.DoRequest("POST", "/api/v2/ci/pipelines/analytics/aggregate", payload)
}

// QueryCITests queries CI test analytics
func (c *Client) QueryCITests(service string, from, to time.Time, filter string) ([]byte, error) {
	fromNs := from.UnixNano()
	toNs := to.UnixNano()

	query := ""
	if service != "" {
		query = fmt.Sprintf("service:%s", service)
	}
	if filter != "" {
		if query != "" {
			query += " AND "
		}
		query += filter
	}

	payload := map[string]interface{}{
		"filter": map[string]interface{}{
			"from":  fmt.Sprintf("%d", fromNs),
			"to":    fmt.Sprintf("%d", toNs),
			"query": query,
		},
		"compute": []map[string]interface{}{
			{"aggregation": "avg", "metric": "test.duration"},
			{"aggregation": "count", "metric": "*"},
		},
		"group_by": []map[string]interface{}{
			{
				"facet": "test.name",
				"limit": 100,
				"sort": map[string]interface{}{
					"order":       "desc",
					"aggregation": "count",
					"metric":      "*",
				},
			},
			{"facet": "test.status"},
			{"facet": "test.suite"},
			{"facet": "test.is_flaky"},
			{"facet": "git.branch"},
			{"facet": "git.repository"},
		},
	}

	return c.DoRequest("POST", "/api/v2/ci/tests/analytics/aggregate", payload)
}

// GetCIPipelineExecutions retrieves CI pipeline execution events
func (c *Client) GetCIPipelineExecutions(service string, from, to time.Time, status string) ([]byte, error) {
	query := "@ci.level:pipeline"
	if service != "" {
		query += fmt.Sprintf(" AND service:%s", service)
	}
	if status != "" && status != "all" {
		query += fmt.Sprintf(" AND status:%s", status)
	}

	payload := map[string]interface{}{
		"filter": map[string]interface{}{
			"query": query,
			"from":  from.Format(time.RFC3339),
			"to":    to.Format(time.RFC3339),
		},
		"sort": "-timestamp",
		"page": map[string]interface{}{
			"limit": 1000,
		},
	}

	return c.DoRequest("POST", "/api/v2/ci/pipelines/events/search", payload)
}

// GetCITestExecutions retrieves CI test execution events
func (c *Client) GetCITestExecutions(service string, from, to time.Time, status string) ([]byte, error) {
	query := "@test.type:test"
	if service != "" {
		query += fmt.Sprintf(" AND service:%s", service)
	}
	if status != "" && status != "all" {
		query += fmt.Sprintf(" AND test.status:%s", status)
	}

	payload := map[string]interface{}{
		"filter": map[string]interface{}{
			"query": query,
			"from":  from.Format(time.RFC3339),
			"to":    to.Format(time.RFC3339),
		},
		"sort": "-timestamp",
		"page": map[string]interface{}{
			"limit": 1000,
		},
	}

	return c.DoRequest("POST", "/api/v2/ci/tests/events/search", payload)
}

// GetCIFailedTests retrieves failed CI tests
func (c *Client) GetCIFailedTests(service string, from, to time.Time) ([]byte, error) {
	return c.GetCITestExecutions(service, from, to, "failed")
}

// QueryDORADeployments retrieves DORA deployment events
func (c *Client) QueryDORADeployments(payload map[string]interface{}) ([]byte, error) {
	return c.DoRequest("POST", "/api/v2/dora/deployments", payload)
}

// QueryDORAFailures retrieves DORA failure/incident events
func (c *Client) QueryDORAFailures(payload map[string]interface{}) ([]byte, error) {
	return c.DoRequest("POST", "/api/v2/dora/failures", payload)
}

// GetDORADeployment retrieves a specific deployment event by ID
func (c *Client) GetDORADeployment(deploymentID string) ([]byte, error) {
	if err := validateResourceID(deploymentID); err != nil {
		return nil, fmt.Errorf("invalid deployment ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/dora/deployments/%s", url.PathEscape(deploymentID))
	return c.DoRequest("GET", endpoint, nil)
}

// GetDORAFailure retrieves a specific failure event by ID
func (c *Client) GetDORAFailure(failureID string) ([]byte, error) {
	if err := validateResourceID(failureID); err != nil {
		return nil, fmt.Errorf("invalid failure ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/dora/failures/%s", url.PathEscape(failureID))
	return c.DoRequest("GET", endpoint, nil)
}

// SendDORADeployment sends a deployment event to DORA Metrics
func (c *Client) SendDORADeployment(payload map[string]interface{}) ([]byte, error) {
	return c.DoRequest("POST", "/api/v2/dora/deployment", payload)
}

// SendDORAFailure sends a failure/incident event to DORA Metrics
func (c *Client) SendDORAFailure(payload map[string]interface{}) ([]byte, error) {
	return c.DoRequest("POST", "/api/v2/dora/failure", payload)
}

// DeleteDORADeployment deletes a deployment event
func (c *Client) DeleteDORADeployment(deploymentID string) error {
	if err := validateResourceID(deploymentID); err != nil {
		return fmt.Errorf("invalid deployment ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/dora/deployment/%s", url.PathEscape(deploymentID))
	_, err := c.DoRequest("DELETE", endpoint, nil)
	return err
}

// DeleteDORAFailure deletes a failure event
func (c *Client) DeleteDORAFailure(failureID string) error {
	if err := validateResourceID(failureID); err != nil {
		return fmt.Errorf("invalid failure ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/dora/failure/%s", url.PathEscape(failureID))
	_, err := c.DoRequest("DELETE", endpoint, nil)
	return err
}

// ListCaseProjects retrieves all Case Management projects
func (c *Client) ListCaseProjects() ([]byte, error) {
	return c.DoRequest("GET", "/api/v2/cases/projects", nil)
}

// CreateCaseProject creates a new Case Management project
func (c *Client) CreateCaseProject(payload map[string]interface{}) ([]byte, error) {
	return c.DoRequest("POST", "/api/v2/cases/projects", payload)
}

// GetCaseProject retrieves a specific project by ID
func (c *Client) GetCaseProject(projectID string) ([]byte, error) {
	if err := validateResourceID(projectID); err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/cases/projects/%s", url.PathEscape(projectID))
	return c.DoRequest("GET", endpoint, nil)
}

// DeleteCaseProject deletes a Case Management project
func (c *Client) DeleteCaseProject(projectID string) error {
	if err := validateResourceID(projectID); err != nil {
		return fmt.Errorf("invalid project ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/cases/projects/%s", url.PathEscape(projectID))
	_, err := c.DoRequest("DELETE", endpoint, nil)
	return err
}

// SearchCases searches and lists cases with optional filters
func (c *Client) SearchCases(payload map[string]interface{}) ([]byte, error) {
	return c.DoRequest("GET", "/api/v2/cases", payload)
}

// CreateCase creates a new case
func (c *Client) CreateCase(payload map[string]interface{}) ([]byte, error) {
	return c.DoRequest("POST", "/api/v2/cases", payload)
}

// GetCase retrieves a specific case by ID
func (c *Client) GetCase(caseID string) ([]byte, error) {
	if err := validateResourceID(caseID); err != nil {
		return nil, fmt.Errorf("invalid case ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/cases/%s", url.PathEscape(caseID))
	return c.DoRequest("GET", endpoint, nil)
}

// AssignCase assigns a case to a user
func (c *Client) AssignCase(caseID string, payload map[string]interface{}) ([]byte, error) {
	if err := validateResourceID(caseID); err != nil {
		return nil, fmt.Errorf("invalid case ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/cases/%s/assign", url.PathEscape(caseID))
	return c.DoRequest("POST", endpoint, payload)
}

// UnassignCase removes assignee from a case
func (c *Client) UnassignCase(caseID string) ([]byte, error) {
	if err := validateResourceID(caseID); err != nil {
		return nil, fmt.Errorf("invalid case ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/cases/%s/unassign", url.PathEscape(caseID))
	return c.DoRequest("POST", endpoint, nil)
}

// ArchiveCase archives a case
func (c *Client) ArchiveCase(caseID string) ([]byte, error) {
	if err := validateResourceID(caseID); err != nil {
		return nil, fmt.Errorf("invalid case ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/cases/%s/archive", url.PathEscape(caseID))
	return c.DoRequest("POST", endpoint, nil)
}

// UnarchiveCase unarchives a case
func (c *Client) UnarchiveCase(caseID string) ([]byte, error) {
	if err := validateResourceID(caseID); err != nil {
		return nil, fmt.Errorf("invalid case ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/cases/%s/unarchive", url.PathEscape(caseID))
	return c.DoRequest("POST", endpoint, nil)
}

// UpdateCaseStatus updates a case's status
func (c *Client) UpdateCaseStatus(caseID string, payload map[string]interface{}) ([]byte, error) {
	if err := validateResourceID(caseID); err != nil {
		return nil, fmt.Errorf("invalid case ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/cases/%s/status", url.PathEscape(caseID))
	return c.DoRequest("POST", endpoint, payload)
}

// UpdateCasePriority updates a case's priority
func (c *Client) UpdateCasePriority(caseID string, payload map[string]interface{}) ([]byte, error) {
	if err := validateResourceID(caseID); err != nil {
		return nil, fmt.Errorf("invalid case ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/cases/%s/priority", url.PathEscape(caseID))
	return c.DoRequest("POST", endpoint, payload)
}

// AddCaseComment adds a comment to a case
func (c *Client) AddCaseComment(caseID string, payload map[string]interface{}) ([]byte, error) {
	if err := validateResourceID(caseID); err != nil {
		return nil, fmt.Errorf("invalid case ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/cases/%s/comment", url.PathEscape(caseID))
	return c.DoRequest("POST", endpoint, payload)
}

// ListContainers retrieves all containers with optional filtering
func (c *Client) ListContainers(params map[string]interface{}) ([]byte, error) {
	endpoint := "/api/v2/containers"

	// Build query string from params
	if len(params) > 0 {
		queryParams := url.Values{}
		for key, value := range params {
			queryParams.Add(key, fmt.Sprintf("%v", value))
		}
		endpoint += "?" + queryParams.Encode()
	}

	return c.DoRequest("GET", endpoint, nil)
}

// ============================================================================
// Status Pages API Methods
// ============================================================================

// ListStatusPages retrieves all status pages
func (c *Client) ListStatusPages() ([]byte, error) {
	return c.DoRequest("GET", "/api/v2/statuspages", nil)
}

// CreateStatusPage creates a new status page
func (c *Client) CreateStatusPage(payload map[string]interface{}) ([]byte, error) {
	return c.DoRequest("POST", "/api/v2/statuspages", payload)
}

// GetStatusPage retrieves a specific status page
func (c *Client) GetStatusPage(pageID string) ([]byte, error) {
	if err := validateResourceID(pageID); err != nil {
		return nil, fmt.Errorf("invalid page ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/statuspages/%s", url.PathEscape(pageID))
	return c.DoRequest("GET", endpoint, nil)
}

// UpdateStatusPage updates a status page
func (c *Client) UpdateStatusPage(pageID string, payload map[string]interface{}) ([]byte, error) {
	if err := validateResourceID(pageID); err != nil {
		return nil, fmt.Errorf("invalid page ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/statuspages/%s", url.PathEscape(pageID))
	return c.DoRequest("PATCH", endpoint, payload)
}

// DeleteStatusPage deletes a status page
func (c *Client) DeleteStatusPage(pageID string) error {
	if err := validateResourceID(pageID); err != nil {
		return fmt.Errorf("invalid page ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/statuspages/%s", url.PathEscape(pageID))
	_, err := c.DoRequest("DELETE", endpoint, nil)
	return err
}

// ListStatusPageComponents retrieves all components for a status page
func (c *Client) ListStatusPageComponents(pageID string) ([]byte, error) {
	if err := validateResourceID(pageID); err != nil {
		return nil, fmt.Errorf("invalid page ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/statuspages/%s/components", url.PathEscape(pageID))
	return c.DoRequest("GET", endpoint, nil)
}

// CreateStatusPageComponent creates a new component on a status page
func (c *Client) CreateStatusPageComponent(pageID string, payload map[string]interface{}) ([]byte, error) {
	if err := validateResourceID(pageID); err != nil {
		return nil, fmt.Errorf("invalid page ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/statuspages/%s/components", url.PathEscape(pageID))
	return c.DoRequest("POST", endpoint, payload)
}

// GetStatusPageComponent retrieves a specific component
func (c *Client) GetStatusPageComponent(pageID, componentID string) ([]byte, error) {
	if err := validateResourceID(pageID); err != nil {
		return nil, fmt.Errorf("invalid page ID: %w", err)
	}
	if err := validateResourceID(componentID); err != nil {
		return nil, fmt.Errorf("invalid component ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/statuspages/%s/components/%s",
		url.PathEscape(pageID), url.PathEscape(componentID))
	return c.DoRequest("GET", endpoint, nil)
}

// UpdateStatusPageComponent updates a component
func (c *Client) UpdateStatusPageComponent(pageID, componentID string, payload map[string]interface{}) ([]byte, error) {
	if err := validateResourceID(pageID); err != nil {
		return nil, fmt.Errorf("invalid page ID: %w", err)
	}
	if err := validateResourceID(componentID); err != nil {
		return nil, fmt.Errorf("invalid component ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/statuspages/%s/components/%s",
		url.PathEscape(pageID), url.PathEscape(componentID))
	return c.DoRequest("PATCH", endpoint, payload)
}

// DeleteStatusPageComponent deletes a component
func (c *Client) DeleteStatusPageComponent(pageID, componentID string) error {
	if err := validateResourceID(pageID); err != nil {
		return fmt.Errorf("invalid page ID: %w", err)
	}
	if err := validateResourceID(componentID); err != nil {
		return fmt.Errorf("invalid component ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/statuspages/%s/components/%s",
		url.PathEscape(pageID), url.PathEscape(componentID))
	_, err := c.DoRequest("DELETE", endpoint, nil)
	return err
}

// ListStatusPageDegradations retrieves all degradations/incidents
func (c *Client) ListStatusPageDegradations() ([]byte, error) {
	return c.DoRequest("GET", "/api/v2/statuspages/degradations", nil)
}

// CreateStatusPageDegradation creates a new degradation/incident
func (c *Client) CreateStatusPageDegradation(pageID string, payload map[string]interface{}) ([]byte, error) {
	if err := validateResourceID(pageID); err != nil {
		return nil, fmt.Errorf("invalid page ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/statuspages/%s/degradations", url.PathEscape(pageID))
	return c.DoRequest("POST", endpoint, payload)
}

// GetStatusPageDegradation retrieves a specific degradation
func (c *Client) GetStatusPageDegradation(pageID, degradationID string) ([]byte, error) {
	if err := validateResourceID(pageID); err != nil {
		return nil, fmt.Errorf("invalid page ID: %w", err)
	}
	if err := validateResourceID(degradationID); err != nil {
		return nil, fmt.Errorf("invalid degradation ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/statuspages/%s/degradations/%s",
		url.PathEscape(pageID), url.PathEscape(degradationID))
	return c.DoRequest("GET", endpoint, nil)
}

// UpdateStatusPageDegradation updates a degradation/incident
func (c *Client) UpdateStatusPageDegradation(pageID, degradationID string, payload map[string]interface{}) ([]byte, error) {
	if err := validateResourceID(pageID); err != nil {
		return nil, fmt.Errorf("invalid page ID: %w", err)
	}
	if err := validateResourceID(degradationID); err != nil {
		return nil, fmt.Errorf("invalid degradation ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/statuspages/%s/degradations/%s",
		url.PathEscape(pageID), url.PathEscape(degradationID))
	return c.DoRequest("PATCH", endpoint, payload)
}

// DeleteStatusPageDegradation deletes/resolves a degradation
func (c *Client) DeleteStatusPageDegradation(pageID, degradationID string) error {
	if err := validateResourceID(pageID); err != nil {
		return fmt.Errorf("invalid page ID: %w", err)
	}
	if err := validateResourceID(degradationID); err != nil {
		return fmt.Errorf("invalid degradation ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/statuspages/%s/degradations/%s",
		url.PathEscape(pageID), url.PathEscape(degradationID))
	_, err := c.DoRequest("DELETE", endpoint, nil)
	return err
}

// ============================================================================
// On-Call API Methods
// ============================================================================

// ListOnCallSchedules retrieves all on-call schedules
func (c *Client) ListOnCallSchedules() ([]byte, error) {
	return c.DoRequest("GET", "/api/v2/on-call/schedules", nil)
}

// CreateOnCallSchedule creates a new on-call schedule
func (c *Client) CreateOnCallSchedule(payload map[string]interface{}) ([]byte, error) {
	return c.DoRequest("POST", "/api/v2/on-call/schedules", payload)
}

// GetOnCallSchedule retrieves a specific on-call schedule
func (c *Client) GetOnCallSchedule(scheduleID string) ([]byte, error) {
	if err := validateResourceID(scheduleID); err != nil {
		return nil, fmt.Errorf("invalid schedule ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/on-call/schedules/%s", url.PathEscape(scheduleID))
	return c.DoRequest("GET", endpoint, nil)
}

// UpdateOnCallSchedule updates an on-call schedule
func (c *Client) UpdateOnCallSchedule(scheduleID string, payload map[string]interface{}) ([]byte, error) {
	if err := validateResourceID(scheduleID); err != nil {
		return nil, fmt.Errorf("invalid schedule ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/on-call/schedules/%s", url.PathEscape(scheduleID))
	return c.DoRequest("PATCH", endpoint, payload)
}

// DeleteOnCallSchedule deletes an on-call schedule
func (c *Client) DeleteOnCallSchedule(scheduleID string) error {
	if err := validateResourceID(scheduleID); err != nil {
		return fmt.Errorf("invalid schedule ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/on-call/schedules/%s", url.PathEscape(scheduleID))
	_, err := c.DoRequest("DELETE", endpoint, nil)
	return err
}

// =============================================================================
// Downtimes API Methods (Phase 2 - Iteration 34)
// =============================================================================

// ListDowntimes retrieves all downtimes
func (c *Client) ListDowntimes(currentOnly bool) ([]byte, error) {
	endpoint := "/api/v2/downtime"
	if currentOnly {
		endpoint += "?current_only=true"
	}
	return c.DoRequest("GET", endpoint, nil)
}

// CreateDowntime creates a new downtime
func (c *Client) CreateDowntime(payload map[string]interface{}) ([]byte, error) {
	return c.DoRequest("POST", "/api/v2/downtime", payload)
}

// GetDowntime retrieves a specific downtime
func (c *Client) GetDowntime(downtimeID string) ([]byte, error) {
	if err := validateResourceID(downtimeID); err != nil {
		return nil, fmt.Errorf("invalid downtime ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/downtime/%s", url.PathEscape(downtimeID))
	return c.DoRequest("GET", endpoint, nil)
}

// UpdateDowntime updates a downtime
func (c *Client) UpdateDowntime(downtimeID string, payload map[string]interface{}) ([]byte, error) {
	if err := validateResourceID(downtimeID); err != nil {
		return nil, fmt.Errorf("invalid downtime ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/downtime/%s", url.PathEscape(downtimeID))
	return c.DoRequest("PATCH", endpoint, payload)
}

// CancelDowntime cancels/deletes a downtime
func (c *Client) CancelDowntime(downtimeID string) error {
	if err := validateResourceID(downtimeID); err != nil {
		return fmt.Errorf("invalid downtime ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/downtime/%s", url.PathEscape(downtimeID))
	_, err := c.DoRequest("DELETE", endpoint, nil)
	return err
}

// =============================================================================
// Notebooks API Methods (Phase 2 - Iteration 35)
// =============================================================================

// ListNotebooks retrieves all notebooks
func (c *Client) ListNotebooks(query, author string) ([]byte, error) {
	endpoint := "/api/v1/notebooks"
	params := make([]string, 0)

	if query != "" {
		params = append(params, "query="+url.QueryEscape(query))
	}
	if author != "" {
		params = append(params, "author_handle="+url.QueryEscape(author))
	}

	if len(params) > 0 {
		endpoint += "?" + url.QueryEscape(params[0])
		for i := 1; i < len(params); i++ {
			endpoint += "&" + url.QueryEscape(params[i])
		}
	}

	return c.DoRequest("GET", endpoint, nil)
}

// GetNotebook retrieves a specific notebook
func (c *Client) GetNotebook(notebookID string) ([]byte, error) {
	if err := validateResourceID(notebookID); err != nil {
		return nil, fmt.Errorf("invalid notebook ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v1/notebooks/%s", url.PathEscape(notebookID))
	return c.DoRequest("GET", endpoint, nil)
}

// DeleteNotebook deletes a notebook
func (c *Client) DeleteNotebook(notebookID string) error {
	if err := validateResourceID(notebookID); err != nil {
		return fmt.Errorf("invalid notebook ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v1/notebooks/%s", url.PathEscape(notebookID))
	_, err := c.DoRequest("DELETE", endpoint, nil)
	return err
}

// =============================================================================
// Teams API Methods (Phase 2 - Iteration 36)
// =============================================================================

// ListTeams retrieves all teams
func (c *Client) ListTeams(keyword string, filterMe bool, sortBy string) ([]byte, error) {
	endpoint := "/api/v2/team"
	params := make([]string, 0)

	if keyword != "" {
		params = append(params, "filter[keyword]="+url.QueryEscape(keyword))
	}
	if filterMe {
		params = append(params, "filter[me]=true")
	}
	if sortBy != "" {
		params = append(params, "sort="+url.QueryEscape(sortBy))
	}

	if len(params) > 0 {
		endpoint += "?" + params[0]
		for i := 1; i < len(params); i++ {
			endpoint += "&" + params[i]
		}
	}

	return c.DoRequest("GET", endpoint, nil)
}

// CreateTeam creates a new team
func (c *Client) CreateTeam(payload map[string]interface{}) ([]byte, error) {
	return c.DoRequest("POST", "/api/v2/team", payload)
}

// GetTeam retrieves a specific team
func (c *Client) GetTeam(teamID string) ([]byte, error) {
	if err := validateResourceID(teamID); err != nil {
		return nil, fmt.Errorf("invalid team ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/team/%s", url.PathEscape(teamID))
	return c.DoRequest("GET", endpoint, nil)
}

// UpdateTeam updates a team
func (c *Client) UpdateTeam(teamID string, payload map[string]interface{}) ([]byte, error) {
	if err := validateResourceID(teamID); err != nil {
		return nil, fmt.Errorf("invalid team ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/team/%s", url.PathEscape(teamID))
	return c.DoRequest("PATCH", endpoint, payload)
}

// DeleteTeam deletes a team
func (c *Client) DeleteTeam(teamID string) error {
	if err := validateResourceID(teamID); err != nil {
		return fmt.Errorf("invalid team ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/team/%s", url.PathEscape(teamID))
	_, err := c.DoRequest("DELETE", endpoint, nil)
	return err
}

// =============================================================================
// Users API Methods (Phase 2 - Iteration 37)
// =============================================================================

// ListUsers retrieves all users
func (c *Client) ListUsers(filter string) ([]byte, error) {
	endpoint := "/api/v2/users"

	if filter != "" {
		endpoint += "?filter=" + url.QueryEscape(filter)
	}

	return c.DoRequest("GET", endpoint, nil)
}

// GetUser retrieves a specific user
func (c *Client) GetUser(userID string) ([]byte, error) {
	if err := validateResourceID(userID); err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/users/%s", url.PathEscape(userID))
	return c.DoRequest("GET", endpoint, nil)
}

// UpdateUser updates a user
func (c *Client) UpdateUser(userID string, payload map[string]interface{}) ([]byte, error) {
	if err := validateResourceID(userID); err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/users/%s", url.PathEscape(userID))
	return c.DoRequest("PATCH", endpoint, payload)
}

// DisableUser disables a user
func (c *Client) DisableUser(userID string) error {
	if err := validateResourceID(userID); err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/users/%s", url.PathEscape(userID))
	_, err := c.DoRequest("DELETE", endpoint, nil)
	return err
}

// =============================================================================
// Roles API Methods (Phase 2 - Iteration 38)
// =============================================================================

// ListRoles retrieves all roles
func (c *Client) ListRoles() ([]byte, error) {
	return c.DoRequest("GET", "/api/v2/roles", nil)
}

// GetRole retrieves a specific role
func (c *Client) GetRole(roleID string) ([]byte, error) {
	if err := validateResourceID(roleID); err != nil {
		return nil, fmt.Errorf("invalid role ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/roles/%s", url.PathEscape(roleID))
	return c.DoRequest("GET", endpoint, nil)
}

// CreateRole creates a new role
func (c *Client) CreateRole(payload map[string]interface{}) ([]byte, error) {
	return c.DoRequest("POST", "/api/v2/roles", payload)
}

// DeleteRole deletes a role
func (c *Client) DeleteRole(roleID string) error {
	if err := validateResourceID(roleID); err != nil {
		return fmt.Errorf("invalid role ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/roles/%s", url.PathEscape(roleID))
	_, err := c.DoRequest("DELETE", endpoint, nil)
	return err
}

// ListAllPermissions retrieves all available permissions
func (c *Client) ListAllPermissions() ([]byte, error) {
	return c.DoRequest("GET", "/api/v2/permissions", nil)
}

// ListRolePermissions retrieves permissions for a specific role
func (c *Client) ListRolePermissions(roleID string) ([]byte, error) {
	if err := validateResourceID(roleID); err != nil {
		return nil, fmt.Errorf("invalid role ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/roles/%s/permissions", url.PathEscape(roleID))
	return c.DoRequest("GET", endpoint, nil)
}

// AddRolePermission adds a permission to a role
func (c *Client) AddRolePermission(roleID, permissionID string) error {
	if err := validateResourceID(roleID); err != nil {
		return fmt.Errorf("invalid role ID: %w", err)
	}
	if err := validateResourceID(permissionID); err != nil {
		return fmt.Errorf("invalid permission ID: %w", err)
	}

	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "permissions",
			"id":   permissionID,
		},
	}

	endpoint := fmt.Sprintf("/api/v2/roles/%s/permissions", url.PathEscape(roleID))
	_, err := c.DoRequest("POST", endpoint, payload)
	return err
}

// RemoveRolePermission removes a permission from a role
func (c *Client) RemoveRolePermission(roleID, permissionID string) error {
	if err := validateResourceID(roleID); err != nil {
		return fmt.Errorf("invalid role ID: %w", err)
	}
	if err := validateResourceID(permissionID); err != nil {
		return fmt.Errorf("invalid permission ID: %w", err)
	}

	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "permissions",
			"id":   permissionID,
		},
	}

	endpoint := fmt.Sprintf("/api/v2/roles/%s/permissions", url.PathEscape(roleID))
	_, err := c.DoRequest("DELETE", endpoint, payload)
	return err
}

// ===========================
// Service Accounts Operations
// ===========================

// ListServiceAccounts retrieves all service accounts with optional filters
func (c *Client) ListServiceAccounts(filter, filterAppID string) ([]byte, error) {
	// Service accounts use the users API with service_account filter
	endpoint := "/api/v2/users?filter[service_account]=true"

	if filter != "" {
		endpoint += "&filter[keyword]=" + url.QueryEscape(filter)
	}
	if filterAppID != "" {
		endpoint += "&filter[application_key]=" + url.QueryEscape(filterAppID)
	}

	return c.DoRequest("GET", endpoint, nil)
}

// GetServiceAccount retrieves a specific service account
func (c *Client) GetServiceAccount(accountID string) ([]byte, error) {
	if err := validateResourceID(accountID); err != nil {
		return nil, fmt.Errorf("invalid service account ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/users/%s", url.PathEscape(accountID))
	return c.DoRequest("GET", endpoint, nil)
}

// CreateServiceAccount creates a new service account
func (c *Client) CreateServiceAccount(payload map[string]interface{}) ([]byte, error) {
	return c.DoRequest("POST", "/api/v2/service_accounts", payload)
}

// UpdateServiceAccount updates an existing service account
func (c *Client) UpdateServiceAccount(accountID string, payload map[string]interface{}) ([]byte, error) {
	if err := validateResourceID(accountID); err != nil {
		return nil, fmt.Errorf("invalid service account ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/service_accounts/%s", url.PathEscape(accountID))
	return c.DoRequest("PATCH", endpoint, payload)
}

// DeleteServiceAccount deletes a service account
func (c *Client) DeleteServiceAccount(accountID string) error {
	if err := validateResourceID(accountID); err != nil {
		return fmt.Errorf("invalid service account ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/users/%s", url.PathEscape(accountID))
	_, err := c.DoRequest("DELETE", endpoint, nil)
	return err
}

// ===============================
// Application Keys Operations
// ===============================

// ListApplicationKeys retrieves application keys (current user or all org keys)
func (c *Client) ListApplicationKeys(allKeys bool) ([]byte, error) {
	endpoint := "/api/v2/current_user/application_keys"
	if allKeys {
		// Admin endpoint for all org keys
		endpoint = "/api/v2/application_keys"
	}
	return c.DoRequest("GET", endpoint, nil)
}

// GetApplicationKey retrieves a specific application key
func (c *Client) GetApplicationKey(keyID string) ([]byte, error) {
	if err := validateResourceID(keyID); err != nil {
		return nil, fmt.Errorf("invalid application key ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/current_user/application_keys/%s", url.PathEscape(keyID))
	return c.DoRequest("GET", endpoint, nil)
}

// CreateApplicationKey creates a new application key
func (c *Client) CreateApplicationKey(payload map[string]interface{}) ([]byte, error) {
	return c.DoRequest("POST", "/api/v2/current_user/application_keys", payload)
}

// UpdateApplicationKey updates an existing application key
func (c *Client) UpdateApplicationKey(keyID string, payload map[string]interface{}) ([]byte, error) {
	if err := validateResourceID(keyID); err != nil {
		return nil, fmt.Errorf("invalid application key ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/current_user/application_keys/%s", url.PathEscape(keyID))
	return c.DoRequest("PATCH", endpoint, payload)
}

// DeleteApplicationKey deletes an application key
func (c *Client) DeleteApplicationKey(keyID string) error {
	if err := validateResourceID(keyID); err != nil {
		return fmt.Errorf("invalid application key ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/current_user/application_keys/%s", url.PathEscape(keyID))
	_, err := c.DoRequest("DELETE", endpoint, nil)
	return err
}

// ====================
// API Keys Operations
// ====================

// ListAPIKeys retrieves all API keys for the organization
func (c *Client) ListAPIKeys() ([]byte, error) {
	return c.DoRequest("GET", "/api/v2/api_keys", nil)
}

// GetAPIKey retrieves a specific API key
func (c *Client) GetAPIKey(keyID string) ([]byte, error) {
	if err := validateResourceID(keyID); err != nil {
		return nil, fmt.Errorf("invalid API key ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/api_keys/%s", url.PathEscape(keyID))
	return c.DoRequest("GET", endpoint, nil)
}

// CreateAPIKey creates a new API key
func (c *Client) CreateAPIKey(payload map[string]interface{}) ([]byte, error) {
	return c.DoRequest("POST", "/api/v2/api_keys", payload)
}

// UpdateAPIKey updates an existing API key
func (c *Client) UpdateAPIKey(keyID string, payload map[string]interface{}) ([]byte, error) {
	if err := validateResourceID(keyID); err != nil {
		return nil, fmt.Errorf("invalid API key ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/api_keys/%s", url.PathEscape(keyID))
	return c.DoRequest("PATCH", endpoint, payload)
}

// DeleteAPIKey deletes an API key
func (c *Client) DeleteAPIKey(keyID string) error {
	if err := validateResourceID(keyID); err != nil {
		return fmt.Errorf("invalid API key ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v2/api_keys/%s", url.PathEscape(keyID))
	_, err := c.DoRequest("DELETE", endpoint, nil)
	return err
}

// =======================
// Audit Logs Operations
// =======================

// SearchAuditLogs searches audit log events
func (c *Client) SearchAuditLogs(query string, from, to time.Time, limit int) ([]byte, error) {
	// Validate limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	payload := map[string]interface{}{
		"filter": map[string]interface{}{
			"query": query,
			"from":  from.Format(time.RFC3339),
			"to":    to.Format(time.RFC3339),
		},
		"page": map[string]interface{}{
			"limit": limit,
		},
		"sort": "timestamp",
	}

	return c.DoRequest("POST", "/api/v2/audit/events/search", payload)
}

// AggregateAuditLogs aggregates audit log events
func (c *Client) AggregateAuditLogs(query string, from, to time.Time) ([]byte, error) {
	payload := map[string]interface{}{
		"filter": map[string]interface{}{
			"query": query,
			"from":  from.Format(time.RFC3339),
			"to":    to.Format(time.RFC3339),
		},
		"compute": []map[string]interface{}{
			{
				"aggregation": "count",
				"type":        "total",
			},
		},
		"group_by": []map[string]interface{}{
			{
				"facet": "@evt.name",
				"limit": 10,
				"sort": map[string]interface{}{
					"aggregation": "count",
					"order":       "desc",
					"type":        "total",
				},
			},
		},
	}

	return c.DoRequest("POST", "/api/v2/audit/events/aggregate", payload)
}

// =============================
// SLO Corrections Operations
// =============================

// ListSLOCorrections retrieves all SLO corrections
func (c *Client) ListSLOCorrections() ([]byte, error) {
	return c.DoRequest("GET", "/api/v1/slo/correction", nil)
}

// GetSLOCorrection retrieves a specific SLO correction
func (c *Client) GetSLOCorrection(correctionID string) ([]byte, error) {
	if err := validateResourceID(correctionID); err != nil {
		return nil, fmt.Errorf("invalid correction ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v1/slo/correction/%s", url.PathEscape(correctionID))
	return c.DoRequest("GET", endpoint, nil)
}

// CreateSLOCorrection creates a new SLO correction
func (c *Client) CreateSLOCorrection(payload map[string]interface{}) ([]byte, error) {
	return c.DoRequest("POST", "/api/v1/slo/correction", payload)
}

// UpdateSLOCorrection updates an existing SLO correction
func (c *Client) UpdateSLOCorrection(correctionID string, payload map[string]interface{}) ([]byte, error) {
	if err := validateResourceID(correctionID); err != nil {
		return nil, fmt.Errorf("invalid correction ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v1/slo/correction/%s", url.PathEscape(correctionID))
	return c.DoRequest("PATCH", endpoint, payload)
}

// DeleteSLOCorrection deletes an SLO correction
func (c *Client) DeleteSLOCorrection(correctionID string) error {
	if err := validateResourceID(correctionID); err != nil {
		return fmt.Errorf("invalid correction ID: %w", err)
	}
	endpoint := fmt.Sprintf("/api/v1/slo/correction/%s", url.PathEscape(correctionID))
	_, err := c.DoRequest("DELETE", endpoint, nil)
	return err
}

// QuerySLOHistory queries SLO history including error budget data
func (c *Client) QuerySLOHistory(sloID string, from, to time.Time) ([]byte, error) {
	if err := validateResourceID(sloID); err != nil {
		return nil, fmt.Errorf("invalid SLO ID: %w", err)
	}

	// Build query parameters
	endpoint := fmt.Sprintf("/api/v1/slo/%s/history?from_ts=%d&to_ts=%d",
		url.PathEscape(sloID),
		from.Unix(),
		to.Unix())

	return c.DoRequest("GET", endpoint, nil)
}

// =============================
// Events Operations
// =============================

// ListEvents retrieves events from the event stream
func (c *Client) ListEvents(params map[string]string) ([]byte, error) {
	// Build query string
	query := url.Values{}
	for k, v := range params {
		query.Add(k, v)
	}

	endpoint := "/api/v1/events"
	if len(query) > 0 {
		endpoint = fmt.Sprintf("%s?%s", endpoint, query.Encode())
	}

	return c.DoRequest("GET", endpoint, nil)
}

// GetEvent retrieves a specific event by ID
func (c *Client) GetEvent(eventID string) ([]byte, error) {
	if eventID == "" {
		return nil, fmt.Errorf("event ID is required")
	}
	endpoint := fmt.Sprintf("/api/v1/events/%s", url.PathEscape(eventID))
	return c.DoRequest("GET", endpoint, nil)
}

// PostEvent posts a new event to the event stream
func (c *Client) PostEvent(payload map[string]interface{}) ([]byte, error) {
	return c.DoRequest("POST", "/api/v1/events", payload)
}

// QueryEventsV2 queries events using the v2 API with advanced search
func (c *Client) QueryEventsV2(params map[string]string) ([]byte, error) {
	// Build query string
	query := url.Values{}
	for k, v := range params {
		query.Add(k, v)
	}

	endpoint := "/api/v2/events"
	if len(query) > 0 {
		endpoint = fmt.Sprintf("%s?%s", endpoint, query.Encode())
	}

	return c.DoRequest("GET", endpoint, nil)
}

// =============================
// Tags Operations
// =============================

// ListAllTags retrieves all tags across infrastructure
func (c *Client) ListAllTags(params map[string]string) ([]byte, error) {
	// Build query string
	query := url.Values{}
	for k, v := range params {
		query.Add(k, v)
	}

	endpoint := "/api/v1/tags/hosts"
	if len(query) > 0 {
		endpoint = fmt.Sprintf("%s?%s", endpoint, query.Encode())
	}

	return c.DoRequest("GET", endpoint, nil)
}

// GetHostTags retrieves tags for a specific host
func (c *Client) GetHostTags(hostName string, params map[string]string) ([]byte, error) {
	if hostName == "" {
		return nil, fmt.Errorf("host name is required")
	}

	// Build query string
	query := url.Values{}
	for k, v := range params {
		query.Add(k, v)
	}

	endpoint := fmt.Sprintf("/api/v1/tags/hosts/%s", url.PathEscape(hostName))
	if len(query) > 0 {
		endpoint = fmt.Sprintf("%s?%s", endpoint, query.Encode())
	}

	return c.DoRequest("GET", endpoint, nil)
}

// AddHostTags adds tags to a host
func (c *Client) AddHostTags(hostName string, tags []string, source string) ([]byte, error) {
	if hostName == "" {
		return nil, fmt.Errorf("host name is required")
	}

	payload := map[string]interface{}{
		"tags": tags,
	}

	if source != "" {
		payload["source"] = source
	}

	endpoint := fmt.Sprintf("/api/v1/tags/hosts/%s", url.PathEscape(hostName))
	return c.DoRequest("POST", endpoint, payload)
}

// UpdateHostTags updates (replaces) tags for a host
func (c *Client) UpdateHostTags(hostName string, tags []string, source string) ([]byte, error) {
	if hostName == "" {
		return nil, fmt.Errorf("host name is required")
	}

	payload := map[string]interface{}{
		"tags": tags,
	}

	if source != "" {
		payload["source"] = source
	}

	endpoint := fmt.Sprintf("/api/v1/tags/hosts/%s", url.PathEscape(hostName))
	return c.DoRequest("PUT", endpoint, payload)
}

// RemoveHostTags removes all tags from a host
func (c *Client) RemoveHostTags(hostName string, params map[string]string) error {
	if hostName == "" {
		return fmt.Errorf("host name is required")
	}

	// Build query string
	query := url.Values{}
	for k, v := range params {
		query.Add(k, v)
	}

	endpoint := fmt.Sprintf("/api/v1/tags/hosts/%s", url.PathEscape(hostName))
	if len(query) > 0 {
		endpoint = fmt.Sprintf("%s?%s", endpoint, query.Encode())
	}

	_, err := c.DoRequest("DELETE", endpoint, nil)
	return err
}

// =============================
// Integrations Operations
// =============================

// GetAWSIntegration retrieves AWS integration configuration
func (c *Client) GetAWSIntegration() ([]byte, error) {
	return c.DoRequest("GET", "/api/v1/integration/aws", nil)
}

// GetAzureIntegration retrieves Azure integration configuration
func (c *Client) GetAzureIntegration() ([]byte, error) {
	return c.DoRequest("GET", "/api/v1/integration/azure", nil)
}

// GetGCPIntegration retrieves GCP integration configuration
func (c *Client) GetGCPIntegration() ([]byte, error) {
	return c.DoRequest("GET", "/api/v1/integration/gcp", nil)
}

// GetSlackIntegration retrieves Slack integration configuration
func (c *Client) GetSlackIntegration() ([]byte, error) {
	return c.DoRequest("GET", "/api/v1/integration/slack", nil)
}

// GetPagerDutyIntegration retrieves PagerDuty integration configuration
func (c *Client) GetPagerDutyIntegration() ([]byte, error) {
	return c.DoRequest("GET", "/api/v1/integration/pagerduty", nil)
}

// =============================
// Spans Operations
// =============================

// SearchSpans searches for spans with filtering
func (c *Client) SearchSpans(query map[string]interface{}) ([]byte, error) {
	return c.DoRequest("POST", "/api/v2/spans/events/search", query)
}

// AggregateSpans aggregates span analytics
func (c *Client) AggregateSpans(query map[string]interface{}) ([]byte, error) {
	return c.DoRequest("POST", "/api/v2/spans/analytics/aggregate", query)
}

// =============================
// Service Dependencies Operations
// =============================

// GetServiceDependencies retrieves service dependency map
func (c *Client) GetServiceDependencies() ([]byte, error) {
	return c.DoRequest("GET", "/api/v1/service_dependencies", nil)
}

// GetServiceDependenciesForService retrieves dependencies for specific service
func (c *Client) GetServiceDependenciesForService(service string) ([]byte, error) {
	if service == "" {
		return nil, fmt.Errorf("service name is required")
	}
	endpoint := fmt.Sprintf("/api/v2/services/%s/dependencies", url.PathEscape(service))
	return c.DoRequest("GET", endpoint, nil)
}

// =============================
// Usage Operations
// =============================

// GetHostUsage retrieves infrastructure host usage
func (c *Client) GetHostUsage(params map[string]string) ([]byte, error) {
	query := url.Values{}
	for k, v := range params {
		query.Add(k, v)
	}

	endpoint := "/api/v1/usage/hosts"
	if len(query) > 0 {
		endpoint = fmt.Sprintf("%s?%s", endpoint, query.Encode())
	}

	return c.DoRequest("GET", endpoint, nil)
}

// GetMetricUsage retrieves custom metrics usage
func (c *Client) GetMetricUsage(params map[string]string) ([]byte, error) {
	query := url.Values{}
	for k, v := range params {
		query.Add(k, v)
	}

	endpoint := "/api/v1/usage/timeseries"
	if len(query) > 0 {
		endpoint = fmt.Sprintf("%s?%s", endpoint, query.Encode())
	}

	return c.DoRequest("GET", endpoint, nil)
}

// GetLogUsage retrieves log management usage
func (c *Client) GetLogUsage(params map[string]string) ([]byte, error) {
	query := url.Values{}
	for k, v := range params {
		query.Add(k, v)
	}

	endpoint := "/api/v1/usage/logs"
	if len(query) > 0 {
		endpoint = fmt.Sprintf("%s?%s", endpoint, query.Encode())
	}

	return c.DoRequest("GET", endpoint, nil)
}

// GetAPMUsage retrieves APM/tracing usage
func (c *Client) GetAPMUsage(params map[string]string) ([]byte, error) {
	query := url.Values{}
	for k, v := range params {
		query.Add(k, v)
	}

	endpoint := "/api/v1/usage/traces"
	if len(query) > 0 {
		endpoint = fmt.Sprintf("%s?%s", endpoint, query.Encode())
	}

	return c.DoRequest("GET", endpoint, nil)
}

// GetWatchdogAnomalies retrieves anomalies detected by Watchdog
func (c *Client) GetWatchdogAnomalies(params map[string]string) ([]byte, error) {
	query := url.Values{}
	for k, v := range params {
		query.Add(k, v)
	}

	endpoint := "/api/v2/watchdog/anomalies"
	if len(query) > 0 {
		endpoint = fmt.Sprintf("%s?%s", endpoint, query.Encode())
	}

	return c.DoRequest("GET", endpoint, nil)
}

// DetectMetricAnomalies detects anomalies in metric data
func (c *Client) DetectMetricAnomalies(query map[string]interface{}) ([]byte, error) {
	return c.DoRequest("POST", "/api/v2/metrics/anomalies", query)
}

// DetectLogAnomalies detects anomalies in log patterns
func (c *Client) DetectLogAnomalies(query map[string]interface{}) ([]byte, error) {
	return c.DoRequest("POST", "/api/v2/logs/analytics/anomalies", query)
}
