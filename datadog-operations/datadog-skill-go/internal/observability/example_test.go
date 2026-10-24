package observability_test

import (
	"context"
	"fmt"
	"time"

	"github.com/datadog/skill/internal/observability"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

// ExampleInit demonstrates how to initialize observability
func ExampleInit() {
	obs, err := observability.Init("my-service", "production")
	if err != nil {
		panic(err)
	}
	defer obs.Shutdown(0)

	// Use observability
	obs.LogInfo("Service started")
	obs.RecordMetric("requests.total", 1.0, "endpoint:/api/health")
}

// ExampleObservability_StartSpan demonstrates how to use tracing
func ExampleObservability_StartSpan() {
	obs, _ := observability.Init("my-service", "production")
	defer obs.Shutdown(0)

	// Create a span
	span := obs.StartSpan("process.request")
	defer obs.FinishSpan(span)

	// Add tags to span
	obs.GetTracer().SetTag(span, "user.id", "12345")
	obs.GetTracer().SetTag(span, "request.method", "GET")

	// Do work here
	time.Sleep(10 * time.Millisecond)
}

// ExampleObservability_TrackOperation demonstrates tracking operations with automatic span management
func ExampleObservability_TrackOperation() {
	obs, _ := observability.Init("my-service", "production")
	defer obs.Shutdown(0)

	ctx := context.Background()

	// Track operation with automatic span creation and duration recording
	err := obs.TrackOperation(ctx, "database.query", func(ctx context.Context, span tracer.Span) error {
		// Do database work here
		time.Sleep(50 * time.Millisecond)
		return nil
	})

	if err != nil {
		fmt.Println("Operation failed:", err)
	}
}

// ExampleObservability_LogInfoWithTrace demonstrates correlated logging with traces
func ExampleObservability_LogInfoWithTrace() {
	obs, _ := observability.Init("my-service", "production")
	defer obs.Shutdown(0)

	// Create a span
	span := obs.StartSpan("api.request")
	defer obs.FinishSpan(span)

	// Log with trace correlation
	obs.LogInfoWithTrace("Processing user request", span, map[string]interface{}{
		"user_id": "12345",
		"action":  "get_profile",
	})
}

// ExampleObservability_RecordAPICall demonstrates recording API call metrics
func ExampleObservability_RecordAPICall() {
	obs, _ := observability.Init("my-service", "production")
	defer obs.Shutdown(0)

	// Simulate API call
	start := time.Now()
	statusCode := 200
	var err error

	// ... make actual API call ...

	duration := time.Since(start).Milliseconds()

	// Record the API call
	obs.RecordAPICall("/api/users", "GET", statusCode, float64(duration), err)
}

// ExampleMetrics demonstrates direct metrics usage
func ExampleMetrics() {
	obs, _ := observability.Init("my-service", "production")
	defer obs.Shutdown(0)

	metrics := obs.GetMetrics()

	// Send different metric types
	metrics.Gauge("queue.size", 42.0)
	metrics.Count("requests.total", 1)
	metrics.Increment("cache.hits", "cache:redis")
	metrics.Histogram("response.time", 123.45, "endpoint:/api/users")
}
