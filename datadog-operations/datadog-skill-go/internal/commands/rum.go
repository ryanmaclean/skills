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

// RUMCommand queries Real User Monitoring data
type RUMCommand struct {
	flags       *flag.FlagSet
	application string
	duration    string
	metric      string
	filter      string
	json        bool
}

// RUMResponse represents the formatted RUM response
type RUMResponse struct {
	Status      string                 `json:"status"`
	Application string                 `json:"application"`
	Duration    string                 `json:"duration"`
	Metric      string                 `json:"metric"`
	Summary     *RUMSummary            `json:"summary,omitempty"`
	Views       []RUMView              `json:"views,omitempty"`
	Sessions    []RUMSession           `json:"sessions,omitempty"`
	Errors      []RUMError             `json:"errors,omitempty"`
	Performance *RUMPerformanceMetrics `json:"performance,omitempty"`
	Geography   map[string]int         `json:"geography,omitempty"`
	Devices     map[string]int         `json:"devices,omitempty"`
	Browsers    map[string]int         `json:"browsers,omitempty"`
}

// RUMSummary provides high-level RUM statistics
type RUMSummary struct {
	TotalViews          int64   `json:"total_views"`
	UniqueViews         int64   `json:"unique_views"`
	TotalSessions       int64   `json:"total_sessions"`
	AvgSessionDuration  float64 `json:"avg_session_duration_seconds"`
	TotalErrors         int64   `json:"total_errors"`
	ErrorRate           float64 `json:"error_rate_percent"`
	BounceRate          float64 `json:"bounce_rate_percent"`
	AvgViewsPerSession  float64 `json:"avg_views_per_session"`
	EngagedSessions     int64   `json:"engaged_sessions"`
	EngagementRate      float64 `json:"engagement_rate_percent"`
	DesktopPercentage   float64 `json:"desktop_percentage"`
	MobilePercentage    float64 `json:"mobile_percentage"`
	TabletPercentage    float64 `json:"tablet_percentage"`
}

// RUMView represents a page view
type RUMView struct {
	Name            string  `json:"name"`
	ViewCount       int64   `json:"view_count"`
	AvgLoadTime     int64   `json:"avg_load_time_ms"`
	BounceRate      float64 `json:"bounce_rate_percent"`
	ErrorCount      int64   `json:"error_count"`
	UniqueUsers     int64   `json:"unique_users"`
}

// RUMSession represents user session data
type RUMSession struct {
	SessionID       string  `json:"session_id"`
	Duration        float64 `json:"duration_seconds"`
	ViewCount       int64   `json:"view_count"`
	ErrorCount      int64   `json:"error_count"`
	UserAgent       string  `json:"user_agent"`
	Geography       string  `json:"geography"`
	DeviceType      string  `json:"device_type"`
}

// RUMError represents a JavaScript or network error
type RUMError struct {
	Message     string `json:"message"`
	Type        string `json:"type"`
	Count       int64  `json:"count"`
	ViewName    string `json:"view_name"`
	Stack       string `json:"stack,omitempty"`
	AffectedUsers int64 `json:"affected_users"`
}

// RUMPerformanceMetrics represents Core Web Vitals and other performance metrics
type RUMPerformanceMetrics struct {
	LCP             *PerformanceMetric `json:"lcp,omitempty"`              // Largest Contentful Paint
	FID             *PerformanceMetric `json:"fid,omitempty"`              // First Input Delay
	CLS             *PerformanceMetric `json:"cls,omitempty"`              // Cumulative Layout Shift
	TTFB            *PerformanceMetric `json:"ttfb,omitempty"`             // Time to First Byte
	FCP             *PerformanceMetric `json:"fcp,omitempty"`              // First Contentful Paint
	DomInteractive  *PerformanceMetric `json:"dom_interactive,omitempty"`  // DOM Interactive
	LoadTime        *PerformanceMetric `json:"load_time,omitempty"`        // Total Load Time
}

// PerformanceMetric represents a single performance metric with statistics
type PerformanceMetric struct {
	P50   float64 `json:"p50"`
	P75   float64 `json:"p75"`
	P90   float64 `json:"p90"`
	P95   float64 `json:"p95"`
	P99   float64 `json:"p99"`
	Avg   float64 `json:"avg"`
	Unit  string  `json:"unit"`
	Grade string  `json:"grade,omitempty"` // good, needs_improvement, poor
}

// NewRUMCommand creates a new RUM command
func NewRUMCommand() *RUMCommand {
	cmd := &RUMCommand{
		flags: flag.NewFlagSet("rum", flag.ExitOnError),
	}

	cmd.flags.StringVar(&cmd.application, "application", "", "RUM application ID (required if multiple apps exist)")
	cmd.flags.StringVar(&cmd.duration, "duration", "24h", "Time range: 1h, 24h, 7d, 30d")
	cmd.flags.StringVar(&cmd.metric, "metric", "all", "Metric type: views, sessions, errors, performance, all")
	cmd.flags.StringVar(&cmd.filter, "filter", "", "Additional filter (e.g., '@view.name:checkout', '@geo.country:US')")
	cmd.flags.BoolVar(&cmd.json, "json", false, "Output as JSON")

	return cmd
}

// Name returns the command name
func (c *RUMCommand) Name() string {
	return "rum"
}

// Description returns the command description
func (c *RUMCommand) Description() string {
	return "Query Real User Monitoring data for frontend performance analysis"
}

// Run executes the RUM command
func (c *RUMCommand) Run(args []string) error {
	// Initialize observability
	obs, err := observability.Init("query-rum", "production")
	if err != nil {
		return fmt.Errorf("failed to init observability: %w", err)
	}
	defer obs.Shutdown(0)

	// Parse flags
	if err := c.flags.Parse(args); err != nil {
		return err
	}

	// Start tracing
	span := obs.StartSpan("rum.query")
	defer obs.FinishSpan(span)

	obs.LogInfo("Starting RUM query")
	obs.GetTracer().SetTag(span, "metric", c.metric)
	obs.GetTracer().SetTag(span, "duration", c.duration)

	// Validate metric type
	validMetrics := map[string]bool{
		"views":       true,
		"sessions":    true,
		"errors":      true,
		"performance": true,
		"all":         true,
	}
	if !validMetrics[c.metric] {
		obs.LogError(fmt.Sprintf("Invalid metric type: %s", c.metric))
		return fmt.Errorf("invalid metric type: %s. Valid options: views, sessions, errors, performance, all", c.metric)
	}

	// Parse duration
	fromTime, toTime, err := c.parseDuration()
	if err != nil {
		obs.LogError("Failed to parse duration: " + err.Error())
		return fmt.Errorf("failed to parse duration: %w", err)
	}

	// Create Datadog client
	clientSpan := obs.StartSpan("rum.create_client")
	ddClient, err := client.NewClient()
	if err != nil {
		obs.LogError("Failed to create Datadog client: " + err.Error())
		obs.FinishSpan(clientSpan)
		return fmt.Errorf("failed to create Datadog client: %w", err)
	}
	obs.FinishSpan(clientSpan)

	// If no application specified, try to get applications list
	applicationID := c.application
	if applicationID == "" {
		appSpan := obs.StartSpan("rum.get_applications")
		obs.LogInfo("Fetching RUM applications")

		start := time.Now()
		appsData, err := ddClient.GetRUMApplications()
		apiDuration := time.Since(start).Milliseconds()

		if err != nil {
			obs.RecordAPICall("/api/v2/rum/applications", "GET", 500, float64(apiDuration), err)
			obs.LogError("Failed to get RUM applications: " + err.Error())
			obs.FinishSpan(appSpan)
			return fmt.Errorf("failed to get RUM applications: %w. Specify --application if you have multiple apps", err)
		}

		obs.RecordAPICall("/api/v2/rum/applications", "GET", 200, float64(apiDuration), nil)
		obs.FinishSpan(appSpan)

		// Parse applications and use the first one
		applicationID, err = c.extractFirstApplication(appsData)
		if err != nil {
			obs.LogError("Failed to extract application: " + err.Error())
			return fmt.Errorf("failed to extract application: %w", err)
		}

		if applicationID == "" {
			obs.LogError("No RUM applications found")
			return fmt.Errorf("no RUM applications found. Create a RUM application first")
		}

		obs.LogInfo(fmt.Sprintf("Using RUM application: %s", applicationID))
	}

	// Query RUM data based on metric type
	response := &RUMResponse{
		Status:      "ok",
		Application: applicationID,
		Duration:    c.duration,
		Metric:      c.metric,
		Geography:   make(map[string]int),
		Devices:     make(map[string]int),
		Browsers:    make(map[string]int),
	}

	// Query different metrics based on request
	if c.metric == "views" || c.metric == "all" {
		if err := c.queryViews(ddClient, obs, applicationID, fromTime, toTime, response); err != nil {
			return err
		}
	}

	if c.metric == "sessions" || c.metric == "all" {
		if err := c.querySessions(ddClient, obs, applicationID, fromTime, toTime, response); err != nil {
			return err
		}
	}

	if c.metric == "errors" || c.metric == "all" {
		if err := c.queryErrors(ddClient, obs, applicationID, fromTime, toTime, response); err != nil {
			return err
		}
	}

	if c.metric == "performance" || c.metric == "all" {
		if err := c.queryPerformance(ddClient, obs, applicationID, fromTime, toTime, response); err != nil {
			return err
		}
	}

	// Calculate summary statistics
	response.Summary = c.calculateSummary(response)

	// Record metrics
	if response.Summary != nil {
		obs.GetMetrics().Gauge("rum.views", float64(response.Summary.TotalViews), "application:"+applicationID)
		obs.GetMetrics().Gauge("rum.sessions", float64(response.Summary.TotalSessions), "application:"+applicationID)
		obs.GetMetrics().Gauge("rum.errors", float64(response.Summary.TotalErrors), "application:"+applicationID)
		obs.GetMetrics().Gauge("rum.error_rate", response.Summary.ErrorRate, "application:"+applicationID)
	}

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

	obs.LogInfo("RUM query complete")
	return nil
}

// parseDuration parses the duration string into time range
func (c *RUMCommand) parseDuration() (time.Time, time.Time, error) {
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

// extractFirstApplication extracts the first application ID from the API response
func (c *RUMCommand) extractFirstApplication(data []byte) (string, error) {
	var response struct {
		Data []struct {
			ID         string `json:"id"`
			Attributes struct {
				Name string `json:"name"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal applications: %w", err)
	}

	if len(response.Data) == 0 {
		return "", nil
	}

	return response.Data[0].ID, nil
}

// queryViews queries RUM view data
func (c *RUMCommand) queryViews(ddClient *client.Client, obs *observability.Observability, applicationID string, from, to time.Time, response *RUMResponse) error {
	span := obs.StartSpan("rum.query_views")
	defer obs.FinishSpan(span)

	obs.LogInfo("Querying RUM views")

	start := time.Now()
	data, err := ddClient.QueryRUMViews(applicationID, from, to, c.filter)
	apiDuration := time.Since(start).Milliseconds()

	if err != nil {
		obs.RecordAPICall("/api/v2/rum/events/search", "POST", 500, float64(apiDuration), err)
		obs.LogError("Failed to query views: " + err.Error())
		return fmt.Errorf("failed to query views: %w", err)
	}

	obs.RecordAPICall("/api/v2/rum/events/search", "POST", 200, float64(apiDuration), nil)

	// Parse views
	views, err := c.parseViews(data)
	if err != nil {
		obs.LogError("Failed to parse views: " + err.Error())
		return fmt.Errorf("failed to parse views: %w", err)
	}

	response.Views = views
	obs.LogInfo(fmt.Sprintf("Found %d views", len(views)))
	return nil
}

// querySessions queries RUM session data
func (c *RUMCommand) querySessions(ddClient *client.Client, obs *observability.Observability, applicationID string, from, to time.Time, response *RUMResponse) error {
	span := obs.StartSpan("rum.query_sessions")
	defer obs.FinishSpan(span)

	obs.LogInfo("Querying RUM sessions")

	start := time.Now()
	data, err := ddClient.QueryRUMSessions(applicationID, from, to, c.filter)
	apiDuration := time.Since(start).Milliseconds()

	if err != nil {
		obs.RecordAPICall("/api/v2/rum/events/search", "POST", 500, float64(apiDuration), err)
		obs.LogError("Failed to query sessions: " + err.Error())
		return fmt.Errorf("failed to query sessions: %w", err)
	}

	obs.RecordAPICall("/api/v2/rum/events/search", "POST", 200, float64(apiDuration), nil)

	// Parse sessions
	sessions, err := c.parseSessions(data)
	if err != nil {
		obs.LogError("Failed to parse sessions: " + err.Error())
		return fmt.Errorf("failed to parse sessions: %w", err)
	}

	response.Sessions = sessions
	obs.LogInfo(fmt.Sprintf("Found %d sessions", len(sessions)))
	return nil
}

// queryErrors queries RUM error data
func (c *RUMCommand) queryErrors(ddClient *client.Client, obs *observability.Observability, applicationID string, from, to time.Time, response *RUMResponse) error {
	span := obs.StartSpan("rum.query_errors")
	defer obs.FinishSpan(span)

	obs.LogInfo("Querying RUM errors")

	start := time.Now()
	data, err := ddClient.QueryRUMErrors(applicationID, from, to, c.filter)
	apiDuration := time.Since(start).Milliseconds()

	if err != nil {
		obs.RecordAPICall("/api/v2/rum/events/search", "POST", 500, float64(apiDuration), err)
		obs.LogError("Failed to query errors: " + err.Error())
		return fmt.Errorf("failed to query errors: %w", err)
	}

	obs.RecordAPICall("/api/v2/rum/events/search", "POST", 200, float64(apiDuration), nil)

	// Parse errors
	errors, err := c.parseErrors(data)
	if err != nil {
		obs.LogError("Failed to parse errors: " + err.Error())
		return fmt.Errorf("failed to parse errors: %w", err)
	}

	response.Errors = errors
	obs.LogInfo(fmt.Sprintf("Found %d error patterns", len(errors)))
	return nil
}

// queryPerformance queries RUM performance metrics (Core Web Vitals)
func (c *RUMCommand) queryPerformance(ddClient *client.Client, obs *observability.Observability, applicationID string, from, to time.Time, response *RUMResponse) error {
	span := obs.StartSpan("rum.query_performance")
	defer obs.FinishSpan(span)

	obs.LogInfo("Querying RUM performance metrics")

	start := time.Now()
	data, err := ddClient.QueryRUMPerformance(applicationID, from, to, c.filter)
	apiDuration := time.Since(start).Milliseconds()

	if err != nil {
		obs.RecordAPICall("/api/v2/rum/events/search", "POST", 500, float64(apiDuration), err)
		obs.LogError("Failed to query performance: " + err.Error())
		return fmt.Errorf("failed to query performance: %w", err)
	}

	obs.RecordAPICall("/api/v2/rum/events/search", "POST", 200, float64(apiDuration), nil)

	// Parse performance metrics
	performance, err := c.parsePerformance(data)
	if err != nil {
		obs.LogError("Failed to parse performance: " + err.Error())
		return fmt.Errorf("failed to parse performance: %w", err)
	}

	response.Performance = performance
	obs.LogInfo("Performance metrics retrieved")
	return nil
}

// parseViews parses view data from API response
func (c *RUMCommand) parseViews(data []byte) ([]RUMView, error) {
	var response struct {
		Data []struct {
			Attributes map[string]interface{} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	// Group by view name
	viewMap := make(map[string]*RUMView)

	for _, item := range response.Data {
		attrs := item.Attributes
		viewName := getStringAttr(attrs, "view.name", "unknown")

		if _, exists := viewMap[viewName]; !exists {
			viewMap[viewName] = &RUMView{
				Name: viewName,
			}
		}

		view := viewMap[viewName]
		view.ViewCount++

		if loadTime := getFloatAttr(attrs, "view.loading_time", 0); loadTime > 0 {
			view.AvgLoadTime += int64(loadTime / 1000000) // Convert ns to ms
		}
	}

	// Convert to slice and calculate averages
	views := make([]RUMView, 0, len(viewMap))
	for _, view := range viewMap {
		if view.ViewCount > 0 {
			view.AvgLoadTime /= view.ViewCount
		}
		views = append(views, *view)
	}

	// Sort by view count
	sort.Slice(views, func(i, j int) bool {
		return views[i].ViewCount > views[j].ViewCount
	})

	// Limit to top 20
	if len(views) > 20 {
		views = views[:20]
	}

	return views, nil
}

// parseSessions parses session data from API response
func (c *RUMCommand) parseSessions(data []byte) ([]RUMSession, error) {
	var response struct {
		Data []struct {
			Attributes map[string]interface{} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	// Group by session ID
	sessionMap := make(map[string]*RUMSession)

	for _, item := range response.Data {
		attrs := item.Attributes
		sessionID := getStringAttr(attrs, "session.id", "")
		if sessionID == "" {
			continue
		}

		if _, exists := sessionMap[sessionID]; !exists {
			sessionMap[sessionID] = &RUMSession{
				SessionID:  sessionID,
				UserAgent:  getStringAttr(attrs, "user_agent", "unknown"),
				Geography:  getStringAttr(attrs, "geo.country", "unknown"),
				DeviceType: getStringAttr(attrs, "device.type", "unknown"),
			}
		}

		session := sessionMap[sessionID]
		duration := getFloatAttr(attrs, "session.time_spent", 0) / 1000000000 // Convert ns to seconds
		if duration > session.Duration {
			session.Duration = duration
		}
	}

	// Convert to slice
	sessions := make([]RUMSession, 0, len(sessionMap))
	for _, session := range sessionMap {
		sessions = append(sessions, *session)
	}

	// Sort by duration
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Duration > sessions[j].Duration
	})

	// Limit to top 100
	if len(sessions) > 100 {
		sessions = sessions[:100]
	}

	return sessions, nil
}

// parseErrors parses error data from API response
func (c *RUMCommand) parseErrors(data []byte) ([]RUMError, error) {
	var response struct {
		Data []struct {
			Attributes map[string]interface{} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	// Group by error message
	errorMap := make(map[string]*RUMError)

	for _, item := range response.Data {
		attrs := item.Attributes
		message := getStringAttr(attrs, "error.message", "unknown error")

		if _, exists := errorMap[message]; !exists {
			errorMap[message] = &RUMError{
				Message:  message,
				Type:     getStringAttr(attrs, "error.type", "unknown"),
				ViewName: getStringAttr(attrs, "view.name", "unknown"),
				Stack:    getStringAttr(attrs, "error.stack", ""),
			}
		}

		errorMap[message].Count++
	}

	// Convert to slice
	errors := make([]RUMError, 0, len(errorMap))
	for _, error := range errorMap {
		errors = append(errors, *error)
	}

	// Sort by count
	sort.Slice(errors, func(i, j int) bool {
		return errors[i].Count > errors[j].Count
	})

	// Limit to top 20
	if len(errors) > 20 {
		errors = errors[:20]
	}

	return errors, nil
}

// parsePerformance parses performance metrics from API response
func (c *RUMCommand) parsePerformance(data []byte) (*RUMPerformanceMetrics, error) {
	var response struct {
		Data []struct {
			Attributes map[string]interface{} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	// Collect all performance metrics
	var lcpValues, fidValues, clsValues, ttfbValues, fcpValues, loadTimeValues []float64

	for _, item := range response.Data {
		attrs := item.Attributes

		if lcp := getFloatAttr(attrs, "view.largest_contentful_paint", 0); lcp > 0 {
			lcpValues = append(lcpValues, lcp/1000000000) // Convert ns to seconds
		}
		if fid := getFloatAttr(attrs, "view.first_input_delay", 0); fid > 0 {
			fidValues = append(fidValues, fid/1000000) // Convert ns to ms
		}
		if cls := getFloatAttr(attrs, "view.cumulative_layout_shift", 0); cls >= 0 {
			clsValues = append(clsValues, cls)
		}
		if ttfb := getFloatAttr(attrs, "view.time_to_first_byte", 0); ttfb > 0 {
			ttfbValues = append(ttfbValues, ttfb/1000000) // Convert ns to ms
		}
		if fcp := getFloatAttr(attrs, "view.first_contentful_paint", 0); fcp > 0 {
			fcpValues = append(fcpValues, fcp/1000000000) // Convert ns to seconds
		}
		if loadTime := getFloatAttr(attrs, "view.loading_time", 0); loadTime > 0 {
			loadTimeValues = append(loadTimeValues, loadTime/1000000) // Convert ns to ms
		}
	}

	performance := &RUMPerformanceMetrics{}

	if len(lcpValues) > 0 {
		performance.LCP = calculatePercentiles(lcpValues, "s")
		performance.LCP.Grade = gradeWebVital("lcp", performance.LCP.P75)
	}
	if len(fidValues) > 0 {
		performance.FID = calculatePercentiles(fidValues, "ms")
		performance.FID.Grade = gradeWebVital("fid", performance.FID.P75)
	}
	if len(clsValues) > 0 {
		performance.CLS = calculatePercentiles(clsValues, "")
		performance.CLS.Grade = gradeWebVital("cls", performance.CLS.P75)
	}
	if len(ttfbValues) > 0 {
		performance.TTFB = calculatePercentiles(ttfbValues, "ms")
	}
	if len(fcpValues) > 0 {
		performance.FCP = calculatePercentiles(fcpValues, "s")
	}
	if len(loadTimeValues) > 0 {
		performance.LoadTime = calculatePercentiles(loadTimeValues, "ms")
	}

	return performance, nil
}

// calculatePercentiles calculates percentiles for a set of values
func calculatePercentiles(values []float64, unit string) *PerformanceMetric {
	if len(values) == 0 {
		return nil
	}

	sort.Float64s(values)

	metric := &PerformanceMetric{
		Unit: unit,
	}

	// Calculate average
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	metric.Avg = sum / float64(len(values))

	// Calculate percentiles
	metric.P50 = percentile(values, 0.50)
	metric.P75 = percentile(values, 0.75)
	metric.P90 = percentile(values, 0.90)
	metric.P95 = percentile(values, 0.95)
	metric.P99 = percentile(values, 0.99)

	return metric
}

// percentile calculates the specified percentile from sorted values
func percentile(sortedValues []float64, p float64) float64 {
	if len(sortedValues) == 0 {
		return 0
	}

	index := p * float64(len(sortedValues)-1)
	lower := int(index)
	upper := lower + 1

	if upper >= len(sortedValues) {
		return sortedValues[len(sortedValues)-1]
	}

	// Linear interpolation
	weight := index - float64(lower)
	return sortedValues[lower]*(1-weight) + sortedValues[upper]*weight
}

// gradeWebVital grades a Core Web Vital based on Google's thresholds
func gradeWebVital(metric string, value float64) string {
	switch metric {
	case "lcp": // Largest Contentful Paint (seconds)
		if value <= 2.5 {
			return "good"
		} else if value <= 4.0 {
			return "needs_improvement"
		}
		return "poor"
	case "fid": // First Input Delay (ms)
		if value <= 100 {
			return "good"
		} else if value <= 300 {
			return "needs_improvement"
		}
		return "poor"
	case "cls": // Cumulative Layout Shift (unitless)
		if value <= 0.1 {
			return "good"
		} else if value <= 0.25 {
			return "needs_improvement"
		}
		return "poor"
	default:
		return ""
	}
}

// calculateSummary calculates summary statistics from RUM data
func (c *RUMCommand) calculateSummary(response *RUMResponse) *RUMSummary {
	summary := &RUMSummary{}

	// Views statistics
	if len(response.Views) > 0 {
		for _, view := range response.Views {
			summary.TotalViews += view.ViewCount
			summary.UniqueViews++ // Each view name is unique
		}
	}

	// Session statistics
	if len(response.Sessions) > 0 {
		summary.TotalSessions = int64(len(response.Sessions))

		totalDuration := 0.0
		engagedCount := int64(0)
		bouncedCount := int64(0)

		deviceCounts := make(map[string]int)

		for _, session := range response.Sessions {
			totalDuration += session.Duration

			// Engaged session: > 30 seconds
			if session.Duration > 30 {
				engagedCount++
			}

			// Bounced session: < 10 seconds and 1 view
			if session.Duration < 10 && session.ViewCount <= 1 {
				bouncedCount++
			}

			// Count device types
			deviceCounts[session.DeviceType]++
		}

		if summary.TotalSessions > 0 {
			summary.AvgSessionDuration = totalDuration / float64(summary.TotalSessions)
			summary.EngagedSessions = engagedCount
			summary.EngagementRate = (float64(engagedCount) / float64(summary.TotalSessions)) * 100
			summary.BounceRate = (float64(bouncedCount) / float64(summary.TotalSessions)) * 100
		}

		if summary.TotalViews > 0 && summary.TotalSessions > 0 {
			summary.AvgViewsPerSession = float64(summary.TotalViews) / float64(summary.TotalSessions)
		}

		// Device percentages
		totalDevices := float64(summary.TotalSessions)
		if totalDevices > 0 {
			summary.DesktopPercentage = (float64(deviceCounts["desktop"]) / totalDevices) * 100
			summary.MobilePercentage = (float64(deviceCounts["mobile"]) / totalDevices) * 100
			summary.TabletPercentage = (float64(deviceCounts["tablet"]) / totalDevices) * 100
		}
	}

	// Error statistics
	if len(response.Errors) > 0 {
		for _, error := range response.Errors {
			summary.TotalErrors += error.Count
		}

		if summary.TotalViews > 0 {
			summary.ErrorRate = (float64(summary.TotalErrors) / float64(summary.TotalViews)) * 100
		}
	}

	return summary
}

// printFormatted prints the RUM response in a conversational format
func (c *RUMCommand) printFormatted(response *RUMResponse) {
	fmt.Printf("Real User Monitoring Analysis\n")
	fmt.Printf("Application: %s\n", response.Application)
	fmt.Printf("Duration: %s\n", response.Duration)
	fmt.Println()

	if response.Summary != nil {
		summary := response.Summary
		fmt.Println("Summary:")
		fmt.Printf("  Total Views: %d (Unique: %d)\n", summary.TotalViews, summary.UniqueViews)
		fmt.Printf("  Total Sessions: %d\n", summary.TotalSessions)
		fmt.Printf("  Avg Session Duration: %.1f seconds\n", summary.AvgSessionDuration)
		fmt.Printf("  Views per Session: %.2f\n", summary.AvgViewsPerSession)
		fmt.Printf("  Engagement Rate: %.1f%%\n", summary.EngagementRate)
		fmt.Printf("  Bounce Rate: %.1f%%\n", summary.BounceRate)

		if summary.TotalErrors > 0 {
			fmt.Printf("  Total Errors: %d (Error Rate: %.2f%%)\n", summary.TotalErrors, summary.ErrorRate)
		}

		fmt.Println()
		fmt.Println("Device Distribution:")
		fmt.Printf("  Desktop: %.1f%%\n", summary.DesktopPercentage)
		fmt.Printf("  Mobile: %.1f%%\n", summary.MobilePercentage)
		fmt.Printf("  Tablet: %.1f%%\n", summary.TabletPercentage)
	}

	// Top views
	if len(response.Views) > 0 {
		fmt.Println()
		fmt.Printf("Top Views (%d):\n", len(response.Views))
		for i, view := range response.Views {
			if i >= 10 {
				break
			}
			fmt.Printf("  %d. %s\n", i+1, view.Name)
			fmt.Printf("     Views: %d | Avg Load Time: %dms\n", view.ViewCount, view.AvgLoadTime)
		}
	}

	// Performance metrics (Core Web Vitals)
	if response.Performance != nil {
		fmt.Println()
		fmt.Println("Core Web Vitals:")

		if response.Performance.LCP != nil {
			lcp := response.Performance.LCP
			fmt.Printf("  Largest Contentful Paint (LCP): %.2fs (P75: %.2fs) [%s]\n",
				lcp.Avg, lcp.P75, lcp.Grade)
		}

		if response.Performance.FID != nil {
			fid := response.Performance.FID
			fmt.Printf("  First Input Delay (FID): %.1fms (P75: %.1fms) [%s]\n",
				fid.Avg, fid.P75, fid.Grade)
		}

		if response.Performance.CLS != nil {
			cls := response.Performance.CLS
			fmt.Printf("  Cumulative Layout Shift (CLS): %.3f (P75: %.3f) [%s]\n",
				cls.Avg, cls.P75, cls.Grade)
		}

		if response.Performance.TTFB != nil {
			ttfb := response.Performance.TTFB
			fmt.Printf("  Time to First Byte (TTFB): %.0fms (P95: %.0fms)\n",
				ttfb.Avg, ttfb.P95)
		}

		if response.Performance.LoadTime != nil {
			loadTime := response.Performance.LoadTime
			fmt.Printf("  Total Load Time: %.0fms (P95: %.0fms)\n",
				loadTime.Avg, loadTime.P95)
		}
	}

	// Top errors
	if len(response.Errors) > 0 {
		fmt.Println()
		fmt.Printf("Top Errors (%d):\n", len(response.Errors))
		for i, error := range response.Errors {
			if i >= 5 {
				break
			}
			fmt.Printf("  %d. [%s] %s\n", i+1, error.Type, truncateRUMString(error.Message, 80))
			fmt.Printf("     Count: %d | View: %s\n", error.Count, error.ViewName)
		}
	}

	// Session insights
	if len(response.Sessions) > 0 && response.Summary != nil {
		fmt.Println()
		fmt.Println("Session Insights:")
		fmt.Printf("  %d engaged sessions (>30s)\n", response.Summary.EngagedSessions)
		fmt.Printf("  Average %.2f views per session\n", response.Summary.AvgViewsPerSession)

		// Show longest sessions
		if len(response.Sessions) > 0 {
			fmt.Printf("  Longest session: %.1f seconds\n", response.Sessions[0].Duration)
		}
	}
}

// Helper functions

func getStringAttr(attrs map[string]interface{}, key, defaultVal string) string {
	if val, ok := attrs[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultVal
}

func getFloatAttr(attrs map[string]interface{}, key string, defaultVal float64) float64 {
	if val, ok := attrs[key]; ok {
		switch v := val.(type) {
		case float64:
			return v
		case int:
			return float64(v)
		case int64:
			return float64(v)
		}
	}
	return defaultVal
}

func truncateRUMString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// Help prints the help message
func (c *RUMCommand) Help() {
	fmt.Println("Usage: dd rum [options]")
	fmt.Println()
	fmt.Println("Query Real User Monitoring data for frontend performance analysis.")
	fmt.Println("Analyzes user experience through page views, sessions, errors, and Core Web Vitals.")
	fmt.Println()
	fmt.Println("Options:")
	c.flags.PrintDefaults()
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  dd rum")
	fmt.Println("  dd rum --application abc123")
	fmt.Println("  dd rum --metric performance --duration 7d")
	fmt.Println("  dd rum --metric errors --filter '@view.name:checkout'")
	fmt.Println("  dd rum --metric sessions --duration 24h --json")
	fmt.Println("  dd rum --filter '@geo.country:US'")
	fmt.Println()
	fmt.Println("Metrics:")
	fmt.Println("  views       - Page views and loading times")
	fmt.Println("  sessions    - User sessions and engagement")
	fmt.Println("  errors      - JavaScript and network errors")
	fmt.Println("  performance - Core Web Vitals (LCP, FID, CLS)")
	fmt.Println("  all         - All metrics (default)")
	fmt.Println()
	fmt.Println("Core Web Vitals Grading:")
	fmt.Println("  LCP (Largest Contentful Paint): <2.5s good, 2.5-4s needs improvement, >4s poor")
	fmt.Println("  FID (First Input Delay): <100ms good, 100-300ms needs improvement, >300ms poor")
	fmt.Println("  CLS (Cumulative Layout Shift): <0.1 good, 0.1-0.25 needs improvement, >0.25 poor")
}
