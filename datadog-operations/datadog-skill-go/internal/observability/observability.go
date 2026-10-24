package observability

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

// Observability provides a unified interface for Datadog observability
type Observability struct {
	serviceName string
	env         string
	tracer      *Tracer
	logger      *Logger
	metrics     *Metrics
	startTime   time.Time
	mu          sync.RWMutex
}

var (
	instance *Observability
	once     sync.Once
)

// Init initializes the observability package with service name and environment
func Init(serviceName, env string) (*Observability, error) {
	var initErr error

	once.Do(func() {
		// Create metrics client
		metrics, err := NewMetrics(serviceName, env)
		if err != nil {
			initErr = fmt.Errorf("failed to initialize metrics: %w", err)
			return
		}

		// Create tracer
		t := NewTracer(serviceName, env)
		if err := t.Start(); err != nil {
			initErr = fmt.Errorf("failed to start tracer: %w", err)
			return
		}

		// Create logger
		logger := NewLogger(serviceName, env)

		instance = &Observability{
			serviceName: serviceName,
			env:         env,
			tracer:      t,
			logger:      logger,
			metrics:     metrics,
			startTime:   time.Now(),
		}

		// Log initialization
		logger.Info(fmt.Sprintf("Observability initialized for service: %s", serviceName))
	})

	if initErr != nil {
		return nil, initErr
	}

	return instance, nil
}

// Get returns the global observability instance
func Get() *Observability {
	return instance
}

// GetTracer returns the tracer instance
func (o *Observability) GetTracer() *Tracer {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.tracer
}

// GetLogger returns the logger instance
func (o *Observability) GetLogger() *Logger {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.logger
}

// GetMetrics returns the metrics instance
func (o *Observability) GetMetrics() *Metrics {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.metrics
}

// StartSpan starts a new trace span
func (o *Observability) StartSpan(operationName string, opts ...tracer.StartSpanOption) tracer.Span {
	return o.tracer.StartSpan(operationName, opts...)
}

// StartSpanFromContext starts a span from a context
func (o *Observability) StartSpanFromContext(ctx context.Context, operationName string, opts ...tracer.StartSpanOption) (tracer.Span, context.Context) {
	return o.tracer.StartSpanFromContext(ctx, operationName, opts...)
}

// FinishSpan finishes a span
func (o *Observability) FinishSpan(span tracer.Span) {
	o.tracer.FinishSpan(span)
}

// LogInfo logs an info message
func (o *Observability) LogInfo(message string, attrs ...map[string]interface{}) {
	o.logger.Info(message, attrs...)
}

// LogWarning logs a warning message
func (o *Observability) LogWarning(message string, attrs ...map[string]interface{}) {
	o.logger.Warning(message, attrs...)
}

// LogError logs an error message
func (o *Observability) LogError(message string, attrs ...map[string]interface{}) {
	o.logger.Error(message, attrs...)
}

// LogInfoWithTrace logs an info message with trace correlation
func (o *Observability) LogInfoWithTrace(message string, span tracer.Span, attrs ...map[string]interface{}) {
	traceID := o.tracer.GetTraceID(span)
	spanID := o.tracer.GetSpanID(span)
	o.logger.InfoWithTrace(message, traceID, spanID, attrs...)
}

// LogWarningWithTrace logs a warning message with trace correlation
func (o *Observability) LogWarningWithTrace(message string, span tracer.Span, attrs ...map[string]interface{}) {
	traceID := o.tracer.GetTraceID(span)
	spanID := o.tracer.GetSpanID(span)
	o.logger.WarningWithTrace(message, traceID, spanID, attrs...)
}

// LogErrorWithTrace logs an error message with trace correlation
func (o *Observability) LogErrorWithTrace(message string, span tracer.Span, attrs ...map[string]interface{}) {
	traceID := o.tracer.GetTraceID(span)
	spanID := o.tracer.GetSpanID(span)
	o.logger.ErrorWithTrace(message, traceID, spanID, attrs...)
}

// RecordMetric sends a gauge metric
func (o *Observability) RecordMetric(name string, value float64, tags ...string) error {
	return o.metrics.Gauge(name, value, tags...)
}

// RecordCount sends a count metric
func (o *Observability) RecordCount(name string, value int64, tags ...string) error {
	return o.metrics.Count(name, value, tags...)
}

// RecordHistogram sends a histogram metric
func (o *Observability) RecordHistogram(name string, value float64, tags ...string) error {
	return o.metrics.Histogram(name, value, tags...)
}

// RecordAPICall records metrics and logs for an API call
func (o *Observability) RecordAPICall(endpoint, method string, statusCode int, durationMs float64, err error) {
	hasError := err != nil

	// Record metric
	o.metrics.RecordAPICall(endpoint, method, statusCode, durationMs, hasError)

	// Log if there was an error
	if hasError {
		o.logger.Error(fmt.Sprintf("API call failed: %s %s", method, endpoint), map[string]interface{}{
			"endpoint":    endpoint,
			"method":      method,
			"status_code": statusCode,
			"duration_ms": durationMs,
			"error":       err.Error(),
		})
	}
}

// TrackOperation tracks an operation with a span and records duration
func (o *Observability) TrackOperation(ctx context.Context, operationName string, fn func(context.Context, tracer.Span) error) error {
	span, ctx := o.StartSpanFromContext(ctx, operationName)
	defer o.FinishSpan(span)

	start := time.Now()
	err := fn(ctx, span)
	duration := time.Since(start).Milliseconds()

	if err != nil {
		o.tracer.SetError(span, err)
		o.metrics.RecordOperationDuration(operationName, float64(duration), true)
		o.LogErrorWithTrace(fmt.Sprintf("Operation failed: %s", operationName), span, map[string]interface{}{
			"error":       err.Error(),
			"duration_ms": duration,
		})
		return err
	}

	o.metrics.RecordOperationDuration(operationName, float64(duration), false)
	return nil
}

// Shutdown gracefully shuts down all observability components
func (o *Observability) Shutdown(exitCode int) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	duration := time.Since(o.startTime).Milliseconds()

	// Record execution duration
	if o.metrics != nil {
		o.metrics.RecordExecutionDuration(float64(duration), exitCode)
	}

	// Log shutdown
	if o.logger != nil {
		if exitCode == 0 {
			o.logger.Info(fmt.Sprintf("Service shutting down: %s", o.serviceName), map[string]interface{}{
				"duration_ms": duration,
				"exit_code":   exitCode,
			})
		} else {
			o.logger.Error(fmt.Sprintf("Service shutting down with error: %s", o.serviceName), map[string]interface{}{
				"duration_ms": duration,
				"exit_code":   exitCode,
			})
		}
	}

	// Flush logger
	if o.logger != nil {
		if err := o.logger.Close(); err != nil {
			return fmt.Errorf("failed to close logger: %w", err)
		}
	}

	// Close metrics
	if o.metrics != nil {
		if err := o.metrics.Close(); err != nil {
			return fmt.Errorf("failed to close metrics: %w", err)
		}
	}

	// Stop tracer
	if o.tracer != nil {
		o.tracer.Stop()
	}

	return nil
}

// Flush flushes all buffered data
func (o *Observability) Flush() error {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if o.logger != nil {
		return o.logger.Flush()
	}

	return nil
}
