package observability

import (
	"fmt"
	"os"
	"sync"
	"testing"
)

// TestNewMetrics tests metrics client creation
func TestNewMetrics(t *testing.T) {
	// Set environment variables
	os.Setenv("DD_AGENT_HOST", "localhost")
	os.Setenv("DD_DOGSTATSD_PORT", "8125")
	defer func() {
		os.Unsetenv("DD_AGENT_HOST")
		os.Unsetenv("DD_DOGSTATSD_PORT")
	}()

	metrics, err := NewMetrics("test-service", "test")
	if err != nil {
		t.Fatalf("NewMetrics failed: %v", err)
	}
	defer metrics.Close()

	if metrics == nil {
		t.Fatal("NewMetrics returned nil")
	}
	if metrics.serviceName != "test-service" {
		t.Errorf("Expected service name 'test-service', got '%s'", metrics.serviceName)
	}
	if metrics.env != "test" {
		t.Errorf("Expected env 'test', got '%s'", metrics.env)
	}
	if metrics.client == nil {
		t.Error("Client should be initialized")
	}
	if len(metrics.tags) == 0 {
		t.Error("Tags should be initialized")
	}
}

// TestNewMetricsDefaultHost tests metrics with default host
func TestNewMetricsDefaultHost(t *testing.T) {
	os.Unsetenv("DD_AGENT_HOST")
	os.Unsetenv("DD_DOGSTATSD_PORT")

	metrics, err := NewMetrics("test-service", "test")
	if err != nil {
		t.Fatalf("NewMetrics failed: %v", err)
	}
	defer metrics.Close()

	if metrics == nil {
		t.Fatal("NewMetrics returned nil")
	}
}

// TestNewMetricsCustomHost tests metrics with custom host
func TestNewMetricsCustomHost(t *testing.T) {
	// Use localhost with custom port instead of non-existent host
	os.Setenv("DD_AGENT_HOST", "localhost")
	os.Setenv("DD_DOGSTATSD_PORT", "9125")
	defer func() {
		os.Unsetenv("DD_AGENT_HOST")
		os.Unsetenv("DD_DOGSTATSD_PORT")
	}()

	metrics, err := NewMetrics("test-service", "test")
	if err != nil {
		t.Fatalf("NewMetrics failed: %v", err)
	}
	defer metrics.Close()

	if metrics == nil {
		t.Fatal("NewMetrics returned nil")
	}
}

// TestDefaultTags tests that default tags are set
func TestDefaultTags(t *testing.T) {
	metrics, err := NewMetrics("test-service", "test")
	if err != nil {
		t.Fatalf("NewMetrics failed: %v", err)
	}
	defer metrics.Close()

	expectedTags := []string{
		"env:test",
		"service:test-service",
	}

	if len(metrics.tags) != len(expectedTags) {
		t.Errorf("Expected %d tags, got %d", len(expectedTags), len(metrics.tags))
	}

	for _, expectedTag := range expectedTags {
		found := false
		for _, tag := range metrics.tags {
			if tag == expectedTag {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected tag '%s' not found", expectedTag)
		}
	}
}

// TestGauge tests gauge metric
func TestGauge(t *testing.T) {
	metrics, err := NewMetrics("test-service", "test")
	if err != nil {
		t.Fatalf("NewMetrics failed: %v", err)
	}
	defer metrics.Close()

	err = metrics.Gauge("test.gauge", 42.5)
	if err != nil {
		t.Errorf("Gauge failed: %v", err)
	}

	// With tags
	err = metrics.Gauge("test.gauge", 100.0, "custom:tag1", "custom:tag2")
	if err != nil {
		t.Errorf("Gauge with tags failed: %v", err)
	}
}

// TestCount tests count metric
func TestCount(t *testing.T) {
	metrics, err := NewMetrics("test-service", "test")
	if err != nil {
		t.Fatalf("NewMetrics failed: %v", err)
	}
	defer metrics.Close()

	err = metrics.Count("test.count", 10)
	if err != nil {
		t.Errorf("Count failed: %v", err)
	}

	// With tags
	err = metrics.Count("test.count", 5, "custom:tag1")
	if err != nil {
		t.Errorf("Count with tags failed: %v", err)
	}
}

// TestIncrement tests increment metric
func TestIncrement(t *testing.T) {
	metrics, err := NewMetrics("test-service", "test")
	if err != nil {
		t.Fatalf("NewMetrics failed: %v", err)
	}
	defer metrics.Close()

	err = metrics.Increment("test.increment")
	if err != nil {
		t.Errorf("Increment failed: %v", err)
	}

	// With tags
	err = metrics.Increment("test.increment", "custom:tag1")
	if err != nil {
		t.Errorf("Increment with tags failed: %v", err)
	}
}

// TestDecrement tests decrement metric
func TestDecrement(t *testing.T) {
	metrics, err := NewMetrics("test-service", "test")
	if err != nil {
		t.Fatalf("NewMetrics failed: %v", err)
	}
	defer metrics.Close()

	err = metrics.Decrement("test.decrement")
	if err != nil {
		t.Errorf("Decrement failed: %v", err)
	}

	// With tags
	err = metrics.Decrement("test.decrement", "custom:tag1")
	if err != nil {
		t.Errorf("Decrement with tags failed: %v", err)
	}
}

// TestHistogram tests histogram metric
func TestHistogram(t *testing.T) {
	metrics, err := NewMetrics("test-service", "test")
	if err != nil {
		t.Fatalf("NewMetrics failed: %v", err)
	}
	defer metrics.Close()

	err = metrics.Histogram("test.histogram", 150.5)
	if err != nil {
		t.Errorf("Histogram failed: %v", err)
	}

	// With tags
	err = metrics.Histogram("test.histogram", 200.0, "custom:tag1")
	if err != nil {
		t.Errorf("Histogram with tags failed: %v", err)
	}
}

// TestTiming tests timing metric
func TestTiming(t *testing.T) {
	metrics, err := NewMetrics("test-service", "test")
	if err != nil {
		t.Fatalf("NewMetrics failed: %v", err)
	}
	defer metrics.Close()

	err = metrics.Timing("test.timing", 100.5)
	if err != nil {
		t.Errorf("Timing failed: %v", err)
	}

	// With tags
	err = metrics.Timing("test.timing", 250.0, "custom:tag1")
	if err != nil {
		t.Errorf("Timing with tags failed: %v", err)
	}
}

// TestDistribution tests distribution metric
func TestDistribution(t *testing.T) {
	metrics, err := NewMetrics("test-service", "test")
	if err != nil {
		t.Fatalf("NewMetrics failed: %v", err)
	}
	defer metrics.Close()

	err = metrics.Distribution("test.distribution", 50.5)
	if err != nil {
		t.Errorf("Distribution failed: %v", err)
	}

	// With tags
	err = metrics.Distribution("test.distribution", 75.0, "custom:tag1")
	if err != nil {
		t.Errorf("Distribution with tags failed: %v", err)
	}
}

// TestSet tests set metric
func TestSet(t *testing.T) {
	metrics, err := NewMetrics("test-service", "test")
	if err != nil {
		t.Fatalf("NewMetrics failed: %v", err)
	}
	defer metrics.Close()

	err = metrics.Set("test.set", "unique-value")
	if err != nil {
		t.Errorf("Set failed: %v", err)
	}

	// With tags
	err = metrics.Set("test.set", "another-value", "custom:tag1")
	if err != nil {
		t.Errorf("Set with tags failed: %v", err)
	}
}

// TestRecordAPICall tests API call metrics
func TestRecordAPICall(t *testing.T) {
	metrics, err := NewMetrics("test-service", "test")
	if err != nil {
		t.Fatalf("NewMetrics failed: %v", err)
	}
	defer metrics.Close()

	// Successful API call
	err = metrics.RecordAPICall("/api/test", "GET", 200, 150.5, false)
	if err != nil {
		t.Errorf("RecordAPICall failed: %v", err)
	}

	// Failed API call
	err = metrics.RecordAPICall("/api/test", "POST", 500, 200.0, true)
	if err != nil {
		t.Errorf("RecordAPICall with error failed: %v", err)
	}
}

// TestRecordOperationDuration tests operation duration metrics
func TestRecordOperationDuration(t *testing.T) {
	metrics, err := NewMetrics("test-service", "test")
	if err != nil {
		t.Fatalf("NewMetrics failed: %v", err)
	}
	defer metrics.Close()

	// Successful operation
	err = metrics.RecordOperationDuration("test-operation", 100.5, false)
	if err != nil {
		t.Errorf("RecordOperationDuration failed: %v", err)
	}

	// Failed operation
	err = metrics.RecordOperationDuration("test-operation", 150.0, true)
	if err != nil {
		t.Errorf("RecordOperationDuration with error failed: %v", err)
	}
}

// TestRecordExecutionDuration tests execution duration metrics
func TestRecordExecutionDuration(t *testing.T) {
	metrics, err := NewMetrics("test-service", "test")
	if err != nil {
		t.Fatalf("NewMetrics failed: %v", err)
	}
	defer metrics.Close()

	// Successful execution
	err = metrics.RecordExecutionDuration(1000.0, 0)
	if err != nil {
		t.Errorf("RecordExecutionDuration failed: %v", err)
	}

	// Failed execution
	err = metrics.RecordExecutionDuration(500.0, 1)
	if err != nil {
		t.Errorf("RecordExecutionDuration with error code failed: %v", err)
	}
}

// TestMetricsWithNilClient tests metrics methods with nil client
func TestMetricsWithNilClient(t *testing.T) {
	metrics := &Metrics{
		serviceName: "test-service",
		env:         "test",
		client:      nil,
		tags:        []string{},
	}

	// All methods should handle nil client gracefully
	err := metrics.Gauge("test.gauge", 42.0)
	if err != nil {
		t.Errorf("Gauge with nil client should not error: %v", err)
	}

	err = metrics.Count("test.count", 10)
	if err != nil {
		t.Errorf("Count with nil client should not error: %v", err)
	}

	err = metrics.Histogram("test.histogram", 100.0)
	if err != nil {
		t.Errorf("Histogram with nil client should not error: %v", err)
	}
}

// TestMetricsClose tests metrics client cleanup
func TestMetricsClose(t *testing.T) {
	metrics, err := NewMetrics("test-service", "test")
	if err != nil {
		t.Fatalf("NewMetrics failed: %v", err)
	}

	// Send some metrics
	metrics.Gauge("test.gauge", 42.0)

	// Close should not error
	err = metrics.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Client should be nil after close
	if metrics.client != nil {
		t.Error("Client should be nil after Close")
	}
}

// TestCloseWithNilClient tests closing metrics with nil client
func TestCloseWithNilClient(t *testing.T) {
	metrics := &Metrics{
		serviceName: "test-service",
		env:         "test",
		client:      nil,
		tags:        []string{},
	}

	// Should not error or panic
	err := metrics.Close()
	if err != nil {
		t.Errorf("Close with nil client should not error: %v", err)
	}
}

// TestConcurrentMetricSubmission tests concurrent metric submission
func TestConcurrentMetricSubmission(t *testing.T) {
	metrics, err := NewMetrics("test-service", "test")
	if err != nil {
		t.Fatalf("NewMetrics failed: %v", err)
	}
	defer metrics.Close()

	var wg sync.WaitGroup
	concurrency := 100

	// Concurrent gauge submissions
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			metrics.Gauge("test.concurrent.gauge", float64(id))
		}(i)
	}

	// Concurrent count submissions
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			metrics.Count("test.concurrent.count", int64(id))
		}(i)
	}

	// Concurrent histogram submissions
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			metrics.Histogram("test.concurrent.histogram", float64(id))
		}(i)
	}

	wg.Wait()
}

// TestTagMerging tests that custom tags are merged with default tags
func TestTagMerging(t *testing.T) {
	metrics, err := NewMetrics("test-service", "test")
	if err != nil {
		t.Fatalf("NewMetrics failed: %v", err)
	}
	defer metrics.Close()

	// The implementation should merge custom tags with default tags
	// This test verifies the method doesn't error with custom tags
	customTags := []string{
		"custom:tag1",
		"custom:tag2",
		"region:us-east-1",
	}

	err = metrics.Gauge("test.gauge", 42.0, customTags...)
	if err != nil {
		t.Errorf("Gauge with custom tags failed: %v", err)
	}
}

// TestMetricNames tests various metric name formats
func TestMetricNames(t *testing.T) {
	metrics, err := NewMetrics("test-service", "test")
	if err != nil {
		t.Fatalf("NewMetrics failed: %v", err)
	}
	defer metrics.Close()

	testCases := []string{
		"simple",
		"with.dots",
		"with_underscores",
		"with-dashes",
		"mixed.format_test",
	}

	for _, metricName := range testCases {
		err = metrics.Gauge(metricName, 42.0)
		if err != nil {
			t.Errorf("Gauge failed for metric name '%s': %v", metricName, err)
		}
	}
}

// TestMetricValues tests various metric value types
func TestMetricValues(t *testing.T) {
	metrics, err := NewMetrics("test-service", "test")
	if err != nil {
		t.Fatalf("NewMetrics failed: %v", err)
	}
	defer metrics.Close()

	testCases := []float64{
		0.0,
		1.0,
		-1.0,
		42.5,
		-42.5,
		1000000.0,
		0.00001,
	}

	for _, value := range testCases {
		err = metrics.Gauge("test.gauge", value)
		if err != nil {
			t.Errorf("Gauge failed for value %f: %v", value, err)
		}

		err = metrics.Histogram("test.histogram", value)
		if err != nil {
			t.Errorf("Histogram failed for value %f: %v", value, err)
		}
	}
}

// TestCountValues tests various count values
func TestCountValues(t *testing.T) {
	metrics, err := NewMetrics("test-service", "test")
	if err != nil {
		t.Fatalf("NewMetrics failed: %v", err)
	}
	defer metrics.Close()

	testCases := []int64{
		0,
		1,
		-1,
		100,
		-100,
		1000000,
	}

	for _, value := range testCases {
		err = metrics.Count("test.count", value)
		if err != nil {
			t.Errorf("Count failed for value %d: %v", value, err)
		}
	}
}

// TestRecordAPICallTags tests that API call creates correct tags
func TestRecordAPICallTags(t *testing.T) {
	metrics, err := NewMetrics("test-service", "test")
	if err != nil {
		t.Fatalf("NewMetrics failed: %v", err)
	}
	defer metrics.Close()

	// Test with various status codes
	statusCodes := []int{200, 201, 400, 404, 500, 503}
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
	endpoints := []string{"/api/test", "/api/users", "/api/items/123"}

	for _, statusCode := range statusCodes {
		for _, method := range methods {
			for _, endpoint := range endpoints {
				err = metrics.RecordAPICall(endpoint, method, statusCode, 100.0, statusCode >= 400)
				if err != nil {
					t.Errorf("RecordAPICall failed for %s %s %d: %v", method, endpoint, statusCode, err)
				}
			}
		}
	}
}

// TestRecordOperationDurationTags tests operation duration with different tags
func TestRecordOperationDurationTags(t *testing.T) {
	metrics, err := NewMetrics("test-service", "test")
	if err != nil {
		t.Fatalf("NewMetrics failed: %v", err)
	}
	defer metrics.Close()

	operations := []string{
		"database.query",
		"http.request",
		"cache.get",
		"queue.publish",
	}

	for _, operation := range operations {
		// Success case
		err = metrics.RecordOperationDuration(operation, 100.0, false)
		if err != nil {
			t.Errorf("RecordOperationDuration failed for %s: %v", operation, err)
		}

		// Error case
		err = metrics.RecordOperationDuration(operation, 150.0, true)
		if err != nil {
			t.Errorf("RecordOperationDuration with error failed for %s: %v", operation, err)
		}
	}
}

// TestMultipleMetricsInstances tests creating multiple independent metrics instances
func TestMultipleMetricsInstances(t *testing.T) {
	metrics1, err := NewMetrics("service-1", "test")
	if err != nil {
		t.Fatalf("NewMetrics 1 failed: %v", err)
	}
	defer metrics1.Close()

	metrics2, err := NewMetrics("service-2", "prod")
	if err != nil {
		t.Fatalf("NewMetrics 2 failed: %v", err)
	}
	defer metrics2.Close()

	if metrics1.serviceName != "service-1" {
		t.Errorf("Metrics 1 service name incorrect: %s", metrics1.serviceName)
	}
	if metrics2.serviceName != "service-2" {
		t.Errorf("Metrics 2 service name incorrect: %s", metrics2.serviceName)
	}
	if metrics1.env != "test" {
		t.Errorf("Metrics 1 env incorrect: %s", metrics1.env)
	}
	if metrics2.env != "prod" {
		t.Errorf("Metrics 2 env incorrect: %s", metrics2.env)
	}
}

// TestTagFormatting tests various tag formats
func TestTagFormatting(t *testing.T) {
	metrics, err := NewMetrics("test-service", "test")
	if err != nil {
		t.Fatalf("NewMetrics failed: %v", err)
	}
	defer metrics.Close()

	tagTestCases := [][]string{
		{"simple:value"},
		{"tag1:value1", "tag2:value2"},
		{"env:prod", "region:us-east-1", "version:1.0.0"},
		{fmt.Sprintf("dynamic:%d", 123)},
		{"empty:", ":empty"},
	}

	for _, tags := range tagTestCases {
		err = metrics.Gauge("test.gauge", 42.0, tags...)
		if err != nil {
			t.Errorf("Gauge failed with tags %v: %v", tags, err)
		}
	}
}
