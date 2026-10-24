package observability

import (
	"context"
	"sync"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

// Tracer provides a clean API for Datadog tracing operations
type Tracer struct {
	serviceName string
	env         string
	mu          sync.RWMutex
	started     bool
}

// NewTracer creates a new Tracer instance
func NewTracer(serviceName, env string) *Tracer {
	return &Tracer{
		serviceName: serviceName,
		env:         env,
		started:     false,
	}
}

// Start initializes the Datadog tracer
func (t *Tracer) Start() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.started {
		return nil
	}

	tracer.Start(
		tracer.WithService(t.serviceName),
		tracer.WithEnv(t.env),
	)

	t.started = true
	return nil
}

// Stop stops the Datadog tracer
func (t *Tracer) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.started {
		return
	}

	tracer.Stop()
	t.started = false
}

// StartSpan creates and starts a new span with the given operation name
func (t *Tracer) StartSpan(operationName string, opts ...tracer.StartSpanOption) tracer.Span {
	return tracer.StartSpan(operationName, opts...)
}

// StartSpanFromContext creates a span that is a child of the span in the context
func (t *Tracer) StartSpanFromContext(ctx context.Context, operationName string, opts ...tracer.StartSpanOption) (tracer.Span, context.Context) {
	return tracer.StartSpanFromContext(ctx, operationName, opts...)
}

// FinishSpan finishes the given span
func (t *Tracer) FinishSpan(span tracer.Span) {
	if span != nil {
		span.Finish()
	}
}

// SetTag sets a tag on the given span
func (t *Tracer) SetTag(span tracer.Span, key string, value interface{}) {
	if span != nil {
		span.SetTag(key, value)
	}
}

// SetError marks the span as having an error
func (t *Tracer) SetError(span tracer.Span, err error) {
	if span != nil && err != nil {
		span.SetTag("error", err)
	}
}

// SetResource sets the resource name for the span
func (t *Tracer) SetResource(span tracer.Span, resource string) {
	if span != nil {
		span.SetTag("resource.name", resource)
	}
}

// GetTraceID returns the trace ID from a span
func (t *Tracer) GetTraceID(span tracer.Span) uint64 {
	if span != nil {
		return span.Context().TraceID()
	}
	return 0
}

// GetSpanID returns the span ID from a span
func (t *Tracer) GetSpanID(span tracer.Span) uint64 {
	if span != nil {
		return span.Context().SpanID()
	}
	return 0
}

// IsStarted returns whether the tracer has been started
func (t *Tracer) IsStarted() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.started
}
