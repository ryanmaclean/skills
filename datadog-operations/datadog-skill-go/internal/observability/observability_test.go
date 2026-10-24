package observability

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

// TestInit tests the initialization of the observability package
func TestInit(t *testing.T) {
	// Reset singleton for testing
	resetSingleton()

	// Set environment variables for testing
	os.Setenv("DD_AGENT_HOST", "localhost")
	os.Setenv("DD_DOGSTATSD_PORT", "8125")
	defer func() {
		os.Unsetenv("DD_AGENT_HOST")
		os.Unsetenv("DD_DOGSTATSD_PORT")
	}()

	obs, err := Init("test-service", "test")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if obs == nil {
		t.Fatal("Init returned nil observability")
	}
	defer obs.Shutdown(0)

	// Verify components are initialized
	if obs.serviceName != "test-service" {
		t.Errorf("Expected service name 'test-service', got '%s'", obs.serviceName)
	}
	if obs.env != "test" {
		t.Errorf("Expected env 'test', got '%s'", obs.env)
	}
	if obs.tracer == nil {
		t.Error("Tracer not initialized")
	}
	if obs.logger == nil {
		t.Error("Logger not initialized")
	}
	if obs.metrics == nil {
		t.Error("Metrics not initialized")
	}
}

// TestSingletonPattern tests that multiple Init calls return the same instance
func TestSingletonPattern(t *testing.T) {
	// Reset singleton for testing
	resetSingleton()

	obs1, err := Init("test-service", "test")
	if err != nil {
		t.Fatalf("First Init failed: %v", err)
	}
	defer obs1.Shutdown(0)

	obs2, err := Init("different-service", "prod")
	if err != nil {
		t.Fatalf("Second Init failed: %v", err)
	}

	// Should return the same instance
	if obs1 != obs2 {
		t.Error("Init did not return singleton instance")
	}

	// Original values should be retained
	if obs2.serviceName != "test-service" {
		t.Errorf("Singleton should retain original service name, got '%s'", obs2.serviceName)
	}
}

// TestGet tests the global instance retrieval
func TestGet(t *testing.T) {
	// Reset singleton for testing
	resetSingleton()

	// Get before Init should return nil
	if Get() != nil {
		t.Error("Get should return nil before Init")
	}

	obs, err := Init("test-service", "test")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer obs.Shutdown(0)

	// Get after Init should return instance
	got := Get()
	if got != obs {
		t.Error("Get did not return the initialized instance")
	}
}

// TestGetters tests the getter methods for components
func TestGetters(t *testing.T) {
	// Reset singleton for testing
	resetSingleton()

	obs, err := Init("test-service", "test")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer obs.Shutdown(0)

	if obs.GetTracer() == nil {
		t.Error("GetTracer returned nil")
	}
	if obs.GetLogger() == nil {
		t.Error("GetLogger returned nil")
	}
	if obs.GetMetrics() == nil {
		t.Error("GetMetrics returned nil")
	}
}

// TestStartSpan tests span creation
func TestStartSpan(t *testing.T) {
	// Reset singleton for testing
	resetSingleton()

	obs, err := Init("test-service", "test")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer obs.Shutdown(0)

	span := obs.StartSpan("test-operation")
	if span == nil {
		t.Fatal("StartSpan returned nil")
	}

	// Verify we can set tags
	span.SetTag("test-key", "test-value")

	obs.FinishSpan(span)
}

// TestObservabilityStartSpanFromContext tests span creation from context
func TestObservabilityStartSpanFromContext(t *testing.T) {
	// Reset singleton for testing
	resetSingleton()

	obs, err := Init("test-service", "test")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer obs.Shutdown(0)

	ctx := context.Background()
	span, newCtx := obs.StartSpanFromContext(ctx, "test-operation")
	if span == nil {
		t.Fatal("StartSpanFromContext returned nil span")
	}
	if newCtx == nil {
		t.Fatal("StartSpanFromContext returned nil context")
	}

	obs.FinishSpan(span)
}

// TestObservabilityFinishSpan tests span finishing
func TestObservabilityFinishSpan(t *testing.T) {
	// Reset singleton for testing
	resetSingleton()

	obs, err := Init("test-service", "test")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer obs.Shutdown(0)

	span := obs.StartSpan("test-operation")

	// Should not panic when finishing a valid span
	obs.FinishSpan(span)

	// Should not panic when finishing nil span
	obs.FinishSpan(nil)
}

// TestLogging tests the logging methods
func TestLogging(t *testing.T) {
	// Reset singleton for testing
	resetSingleton()

	obs, err := Init("test-service", "test")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer obs.Shutdown(0)

	// Test Info logging
	obs.LogInfo("Test info message")
	obs.LogInfo("Test info with attrs", map[string]interface{}{
		"key": "value",
	})

	// Test Warning logging
	obs.LogWarning("Test warning message")
	obs.LogWarning("Test warning with attrs", map[string]interface{}{
		"key": "value",
	})

	// Test Error logging
	obs.LogError("Test error message")
	obs.LogError("Test error with attrs", map[string]interface{}{
		"key": "value",
	})
}

// TestLoggingWithTrace tests logging with trace correlation
func TestLoggingWithTrace(t *testing.T) {
	// Reset singleton for testing
	resetSingleton()

	obs, err := Init("test-service", "test")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer obs.Shutdown(0)

	span := obs.StartSpan("test-operation")
	defer obs.FinishSpan(span)

	// Test Info logging with trace
	obs.LogInfoWithTrace("Test info with trace", span)
	obs.LogInfoWithTrace("Test info with trace and attrs", span, map[string]interface{}{
		"key": "value",
	})

	// Test Warning logging with trace
	obs.LogWarningWithTrace("Test warning with trace", span)

	// Test Error logging with trace
	obs.LogErrorWithTrace("Test error with trace", span)
}

// TestMetrics tests the metrics recording methods
func TestMetrics(t *testing.T) {
	// Reset singleton for testing
	resetSingleton()

	obs, err := Init("test-service", "test")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer obs.Shutdown(0)

	// Test gauge metric
	err = obs.RecordMetric("test.gauge", 42.5, "tag1:value1")
	if err != nil {
		t.Errorf("RecordMetric failed: %v", err)
	}

	// Test count metric
	err = obs.RecordCount("test.count", 10, "tag1:value1")
	if err != nil {
		t.Errorf("RecordCount failed: %v", err)
	}

	// Test histogram metric
	err = obs.RecordHistogram("test.histogram", 100.5, "tag1:value1")
	if err != nil {
		t.Errorf("RecordHistogram failed: %v", err)
	}
}

// TestObservabilityRecordAPICall tests API call recording
func TestObservabilityRecordAPICall(t *testing.T) {
	// Reset singleton for testing
	resetSingleton()

	obs, err := Init("test-service", "test")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer obs.Shutdown(0)

	// Test successful API call
	obs.RecordAPICall("/api/test", "GET", 200, 150.5, nil)

	// Test failed API call
	obs.RecordAPICall("/api/test", "POST", 500, 200.0, errors.New("test error"))
}

// TestTrackOperation tests operation tracking
func TestTrackOperation(t *testing.T) {
	// Reset singleton for testing
	resetSingleton()

	obs, err := Init("test-service", "test")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer obs.Shutdown(0)

	ctx := context.Background()

	// Test successful operation
	err = obs.TrackOperation(ctx, "test-operation", func(ctx context.Context, span tracer.Span) error {
		// Simulate work
		time.Sleep(10 * time.Millisecond)
		return nil
	})
	if err != nil {
		t.Errorf("TrackOperation failed: %v", err)
	}

	// Test failed operation
	testErr := errors.New("operation failed")
	err = obs.TrackOperation(ctx, "failed-operation", func(ctx context.Context, span tracer.Span) error {
		return testErr
	})
	if err != testErr {
		t.Errorf("Expected error %v, got %v", testErr, err)
	}
}

// TestObservabilityFlush tests the flush functionality
func TestObservabilityFlush(t *testing.T) {
	// Reset singleton for testing
	resetSingleton()

	obs, err := Init("test-service", "test")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer obs.Shutdown(0)

	// Log some messages
	obs.LogInfo("Test message 1")
	obs.LogInfo("Test message 2")

	// Flush should not error
	err = obs.Flush()
	if err != nil {
		t.Errorf("Flush failed: %v", err)
	}
}

// TestShutdown tests graceful shutdown
func TestShutdown(t *testing.T) {
	// Reset singleton for testing
	resetSingleton()

	obs, err := Init("test-service", "test")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Log and record some data
	obs.LogInfo("Test message before shutdown")
	obs.RecordMetric("test.metric", 42.0)

	// Shutdown with success exit code
	err = obs.Shutdown(0)
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}
}

// TestShutdownWithError tests shutdown with error exit code
func TestShutdownWithError(t *testing.T) {
	// Reset singleton for testing
	resetSingleton()

	obs, err := Init("test-service", "test")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Shutdown with error exit code
	err = obs.Shutdown(1)
	if err != nil {
		t.Errorf("Shutdown with error code failed: %v", err)
	}
}

// TestConcurrentAccess tests concurrent access to observability methods
func TestConcurrentAccess(t *testing.T) {
	// Reset singleton for testing
	resetSingleton()

	obs, err := Init("test-service", "test")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer obs.Shutdown(0)

	var wg sync.WaitGroup
	concurrency := 10

	// Test concurrent span creation
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			span := obs.StartSpan("concurrent-operation")
			obs.FinishSpan(span)
		}(i)
	}

	// Test concurrent logging
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			obs.LogInfo("Concurrent log message")
		}(i)
	}

	// Test concurrent metrics
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			obs.RecordMetric("test.concurrent", float64(id))
		}(i)
	}

	wg.Wait()
}

// TestInitWithoutAPIKey tests initialization without Datadog API key
func TestInitWithoutAPIKey(t *testing.T) {
	// Reset singleton for testing
	resetSingleton()

	// Ensure no API key is set
	os.Unsetenv("DD_API_KEY")

	obs, err := Init("test-service", "test")
	if err != nil {
		t.Fatalf("Init should not fail without API key: %v", err)
	}
	if obs == nil {
		t.Fatal("Init returned nil without API key")
	}
	defer obs.Shutdown(0)

	// Should be able to log (logs won't be sent but shouldn't error)
	obs.LogInfo("Test message without API key")
}

// resetSingleton resets the singleton instance for testing
// This is a test helper function
func resetSingleton() {
	instance = nil
	once = sync.Once{}
}
