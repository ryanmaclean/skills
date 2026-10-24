package observability

import (
	"context"
	"errors"
	"sync"
	"testing"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

// TestNewTracer tests tracer creation
func TestNewTracer(t *testing.T) {
	tr := NewTracer("test-service", "test")
	if tr == nil {
		t.Fatal("NewTracer returned nil")
	}
	if tr.serviceName != "test-service" {
		t.Errorf("Expected service name 'test-service', got '%s'", tr.serviceName)
	}
	if tr.env != "test" {
		t.Errorf("Expected env 'test', got '%s'", tr.env)
	}
	if tr.started {
		t.Error("Tracer should not be started initially")
	}
}

// TestTracerStart tests tracer initialization
func TestTracerStart(t *testing.T) {
	tr := NewTracer("test-service", "test")

	err := tr.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer tr.Stop()

	if !tr.IsStarted() {
		t.Error("Tracer should be started after Start()")
	}
}

// TestTracerStartIdempotent tests that Start is idempotent
func TestTracerStartIdempotent(t *testing.T) {
	tr := NewTracer("test-service", "test")

	// Start multiple times
	err := tr.Start()
	if err != nil {
		t.Fatalf("First Start failed: %v", err)
	}

	err = tr.Start()
	if err != nil {
		t.Fatalf("Second Start failed: %v", err)
	}

	err = tr.Start()
	if err != nil {
		t.Fatalf("Third Start failed: %v", err)
	}

	defer tr.Stop()

	if !tr.IsStarted() {
		t.Error("Tracer should be started")
	}
}

// TestTracerStop tests tracer shutdown
func TestTracerStop(t *testing.T) {
	tr := NewTracer("test-service", "test")

	err := tr.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	tr.Stop()

	if tr.IsStarted() {
		t.Error("Tracer should not be started after Stop()")
	}
}

// TestTracerStopIdempotent tests that Stop is idempotent
func TestTracerStopIdempotent(t *testing.T) {
	tr := NewTracer("test-service", "test")

	err := tr.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Stop multiple times should not panic
	tr.Stop()
	tr.Stop()
	tr.Stop()

	if tr.IsStarted() {
		t.Error("Tracer should not be started after multiple Stop() calls")
	}
}

// TestTracerStopWithoutStart tests stopping a tracer that was never started
func TestTracerStopWithoutStart(t *testing.T) {
	tr := NewTracer("test-service", "test")

	// Should not panic
	tr.Stop()

	if tr.IsStarted() {
		t.Error("Tracer should not be started")
	}
}

// TestStartSpanBasic tests basic span creation
func TestStartSpanBasic(t *testing.T) {
	tr := NewTracer("test-service", "test")
	err := tr.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer tr.Stop()

	span := tr.StartSpan("test-operation")
	if span == nil {
		t.Fatal("StartSpan returned nil")
	}

	tr.FinishSpan(span)
}

// TestStartSpanWithOptions tests span creation with options
func TestStartSpanWithOptions(t *testing.T) {
	tr := NewTracer("test-service", "test")
	err := tr.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer tr.Stop()

	span := tr.StartSpan("test-operation",
		tracer.ResourceName("test-resource"),
		tracer.Tag("custom-tag", "custom-value"),
	)
	if span == nil {
		t.Fatal("StartSpan returned nil")
	}

	tr.FinishSpan(span)
}

// TestStartSpanFromContext tests span creation from context
func TestStartSpanFromContext(t *testing.T) {
	tr := NewTracer("test-service", "test")
	err := tr.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer tr.Stop()

	ctx := context.Background()
	span, newCtx := tr.StartSpanFromContext(ctx, "test-operation")
	if span == nil {
		t.Fatal("StartSpanFromContext returned nil span")
	}
	if newCtx == nil {
		t.Fatal("StartSpanFromContext returned nil context")
	}

	tr.FinishSpan(span)
}

// TestStartSpanFromContextWithParent tests child span creation
func TestStartSpanFromContextWithParent(t *testing.T) {
	tr := NewTracer("test-service", "test")
	err := tr.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer tr.Stop()

	// Create parent span
	parentSpan, ctx := tr.StartSpanFromContext(context.Background(), "parent-operation")
	defer tr.FinishSpan(parentSpan)

	// Create child span
	childSpan, childCtx := tr.StartSpanFromContext(ctx, "child-operation")
	if childSpan == nil {
		t.Fatal("Child span is nil")
	}
	if childCtx == nil {
		t.Fatal("Child context is nil")
	}
	defer tr.FinishSpan(childSpan)

	// Child should have same trace ID as parent
	parentTraceID := tr.GetTraceID(parentSpan)
	childTraceID := tr.GetTraceID(childSpan)
	if parentTraceID != childTraceID {
		t.Errorf("Child span trace ID (%d) should match parent trace ID (%d)", childTraceID, parentTraceID)
	}
}

// TestFinishSpan tests span finishing
func TestFinishSpan(t *testing.T) {
	tr := NewTracer("test-service", "test")
	err := tr.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer tr.Stop()

	span := tr.StartSpan("test-operation")

	// Should not panic
	tr.FinishSpan(span)
}

// TestFinishNilSpan tests finishing a nil span
func TestFinishNilSpan(t *testing.T) {
	tr := NewTracer("test-service", "test")
	err := tr.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer tr.Stop()

	// Should not panic
	tr.FinishSpan(nil)
}

// TestSetTag tests setting tags on spans
func TestSetTag(t *testing.T) {
	tr := NewTracer("test-service", "test")
	err := tr.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer tr.Stop()

	span := tr.StartSpan("test-operation")
	defer tr.FinishSpan(span)

	// Should not panic
	tr.SetTag(span, "test-key", "test-value")
	tr.SetTag(span, "numeric-key", 42)
	tr.SetTag(span, "bool-key", true)
}

// TestSetTagOnNilSpan tests setting tags on nil span
func TestSetTagOnNilSpan(t *testing.T) {
	tr := NewTracer("test-service", "test")
	err := tr.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer tr.Stop()

	// Should not panic
	tr.SetTag(nil, "test-key", "test-value")
}

// TestSetError tests marking spans with errors
func TestSetError(t *testing.T) {
	tr := NewTracer("test-service", "test")
	err := tr.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer tr.Stop()

	span := tr.StartSpan("test-operation")
	defer tr.FinishSpan(span)

	testErr := errors.New("test error")

	// Should not panic
	tr.SetError(span, testErr)
}

// TestSetErrorOnNilSpan tests setting error on nil span
func TestSetErrorOnNilSpan(t *testing.T) {
	tr := NewTracer("test-service", "test")
	err := tr.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer tr.Stop()

	testErr := errors.New("test error")

	// Should not panic
	tr.SetError(nil, testErr)
}

// TestSetErrorWithNilError tests setting nil error
func TestSetErrorWithNilError(t *testing.T) {
	tr := NewTracer("test-service", "test")
	err := tr.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer tr.Stop()

	span := tr.StartSpan("test-operation")
	defer tr.FinishSpan(span)

	// Should not panic
	tr.SetError(span, nil)
}

// TestSetResource tests setting resource name
func TestSetResource(t *testing.T) {
	tr := NewTracer("test-service", "test")
	err := tr.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer tr.Stop()

	span := tr.StartSpan("test-operation")
	defer tr.FinishSpan(span)

	// Should not panic
	tr.SetResource(span, "test-resource")
}

// TestSetResourceOnNilSpan tests setting resource on nil span
func TestSetResourceOnNilSpan(t *testing.T) {
	tr := NewTracer("test-service", "test")
	err := tr.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer tr.Stop()

	// Should not panic
	tr.SetResource(nil, "test-resource")
}

// TestGetTraceID tests getting trace ID from span
func TestGetTraceID(t *testing.T) {
	tr := NewTracer("test-service", "test")
	err := tr.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer tr.Stop()

	span := tr.StartSpan("test-operation")
	defer tr.FinishSpan(span)

	traceID := tr.GetTraceID(span)
	if traceID == 0 {
		t.Error("TraceID should not be 0 for valid span")
	}
}

// TestGetTraceIDFromNilSpan tests getting trace ID from nil span
func TestGetTraceIDFromNilSpan(t *testing.T) {
	tr := NewTracer("test-service", "test")
	err := tr.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer tr.Stop()

	traceID := tr.GetTraceID(nil)
	if traceID != 0 {
		t.Error("TraceID should be 0 for nil span")
	}
}

// TestGetSpanID tests getting span ID from span
func TestGetSpanID(t *testing.T) {
	tr := NewTracer("test-service", "test")
	err := tr.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer tr.Stop()

	span := tr.StartSpan("test-operation")
	defer tr.FinishSpan(span)

	spanID := tr.GetSpanID(span)
	if spanID == 0 {
		t.Error("SpanID should not be 0 for valid span")
	}
}

// TestGetSpanIDFromNilSpan tests getting span ID from nil span
func TestGetSpanIDFromNilSpan(t *testing.T) {
	tr := NewTracer("test-service", "test")
	err := tr.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer tr.Stop()

	spanID := tr.GetSpanID(nil)
	if spanID != 0 {
		t.Error("SpanID should be 0 for nil span")
	}
}

// TestIsStarted tests the IsStarted method
func TestIsStarted(t *testing.T) {
	tr := NewTracer("test-service", "test")

	if tr.IsStarted() {
		t.Error("Tracer should not be started initially")
	}

	err := tr.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if !tr.IsStarted() {
		t.Error("Tracer should be started after Start()")
	}

	tr.Stop()

	if tr.IsStarted() {
		t.Error("Tracer should not be started after Stop()")
	}
}

// TestConcurrentSpanCreation tests concurrent span creation
func TestConcurrentSpanCreation(t *testing.T) {
	tr := NewTracer("test-service", "test")
	err := tr.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer tr.Stop()

	var wg sync.WaitGroup
	concurrency := 100

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			span := tr.StartSpan("concurrent-operation")
			tr.SetTag(span, "operation-id", id)
			tr.FinishSpan(span)
		}(i)
	}

	wg.Wait()
}

// TestConcurrentStartStop tests concurrent start and stop calls
func TestConcurrentStartStop(t *testing.T) {
	tr := NewTracer("test-service", "test")

	var wg sync.WaitGroup
	concurrency := 10

	// Multiple goroutines trying to start
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tr.Start()
		}()
	}

	wg.Wait()

	if !tr.IsStarted() {
		t.Error("Tracer should be started after concurrent Start() calls")
	}

	// Multiple goroutines trying to stop
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tr.Stop()
		}()
	}

	wg.Wait()

	if tr.IsStarted() {
		t.Error("Tracer should not be started after concurrent Stop() calls")
	}
}

// TestMultipleTracers tests creating multiple independent tracers
func TestMultipleTracers(t *testing.T) {
	tr1 := NewTracer("service-1", "test")
	tr2 := NewTracer("service-2", "prod")

	if tr1.serviceName != "service-1" {
		t.Errorf("Tracer 1 service name incorrect: %s", tr1.serviceName)
	}
	if tr2.serviceName != "service-2" {
		t.Errorf("Tracer 2 service name incorrect: %s", tr2.serviceName)
	}
	if tr1.env != "test" {
		t.Errorf("Tracer 1 env incorrect: %s", tr1.env)
	}
	if tr2.env != "prod" {
		t.Errorf("Tracer 2 env incorrect: %s", tr2.env)
	}
}
