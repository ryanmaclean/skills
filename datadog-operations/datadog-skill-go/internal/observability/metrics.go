package observability

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/DataDog/datadog-go/v5/statsd"
)

// Metrics provides a clean API for sending metrics to Datadog via StatsD
type Metrics struct {
	serviceName        string
	env                string
	client             *statsd.Client
	mu                 sync.RWMutex
	tags               []string
	failedSends        uint64 // Counter for failed metric sends
	lastFailureMessage string // Last failure message for debugging
}

// NewMetrics creates a new Metrics instance
func NewMetrics(serviceName, env string) (*Metrics, error) {
	// Get DD agent address from env or use default
	agentAddr := os.Getenv("DD_AGENT_HOST")
	if agentAddr == "" {
		agentAddr = "localhost"
	}

	// Get DD agent port from env or use default
	agentPort := os.Getenv("DD_DOGSTATSD_PORT")
	if agentPort == "" {
		agentPort = "8125"
	}

	addr := fmt.Sprintf("%s:%s", agentAddr, agentPort)

	// Create StatsD client
	client, err := statsd.New(addr,
		statsd.WithNamespace("datadog.skill."),
		statsd.WithTags([]string{
			fmt.Sprintf("env:%s", env),
			fmt.Sprintf("service:%s", serviceName),
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create statsd client: %w", err)
	}

	return &Metrics{
		serviceName: serviceName,
		env:         env,
		client:      client,
		tags: []string{
			fmt.Sprintf("env:%s", env),
			fmt.Sprintf("service:%s", serviceName),
		},
	}, nil
}

// trackFailure records a metric send failure for meta-observability
func (m *Metrics) trackFailure(err error) {
	if err == nil {
		return
	}
	m.mu.Lock()
	m.failedSends++
	m.lastFailureMessage = err.Error()
	m.mu.Unlock()
	fmt.Fprintf(os.Stderr, "datadog-cli: metrics failed to send: %v\n", err)
}

// Gauge sends a gauge metric
func (m *Metrics) Gauge(name string, value float64, tags ...string) error {
	m.mu.RLock()
	if m.client == nil {
		m.mu.RUnlock()
		m.trackFailure(fmt.Errorf("statsd client is nil"))
		return nil
	}

	allTags := append(m.tags, tags...)
	client := m.client
	m.mu.RUnlock()

	err := client.Gauge(name, value, allTags, 1)
	m.trackFailure(err)
	return nil // Don't propagate errors - observability should never break the application
}

// Count sends a count metric
func (m *Metrics) Count(name string, value int64, tags ...string) error {
	m.mu.RLock()
	if m.client == nil {
		m.mu.RUnlock()
		m.trackFailure(fmt.Errorf("statsd client is nil"))
		return nil
	}

	allTags := append(m.tags, tags...)
	client := m.client
	m.mu.RUnlock()

	err := client.Count(name, value, allTags, 1)
	m.trackFailure(err)
	return nil
}

// Increment increments a count metric by 1
func (m *Metrics) Increment(name string, tags ...string) error {
	return m.Count(name, 1, tags...)
}

// Decrement decrements a count metric by 1
func (m *Metrics) Decrement(name string, tags ...string) error {
	return m.Count(name, -1, tags...)
}

// Histogram sends a histogram metric
func (m *Metrics) Histogram(name string, value float64, tags ...string) error {
	m.mu.RLock()
	if m.client == nil {
		m.mu.RUnlock()
		m.trackFailure(fmt.Errorf("statsd client is nil"))
		return nil
	}

	allTags := append(m.tags, tags...)
	client := m.client
	m.mu.RUnlock()

	err := client.Histogram(name, value, allTags, 1)
	m.trackFailure(err)
	return nil
}

// Timing sends a timing metric (in milliseconds)
func (m *Metrics) Timing(name string, valueMs float64, tags ...string) error {
	m.mu.RLock()
	if m.client == nil {
		m.mu.RUnlock()
		m.trackFailure(fmt.Errorf("statsd client is nil"))
		return nil
	}

	allTags := append(m.tags, tags...)
	// Convert milliseconds to time.Duration
	duration := time.Duration(valueMs * float64(time.Millisecond))
	client := m.client
	m.mu.RUnlock()

	err := client.Timing(name, duration, allTags, 1)
	m.trackFailure(err)
	return nil
}

// Distribution sends a distribution metric
func (m *Metrics) Distribution(name string, value float64, tags ...string) error {
	m.mu.RLock()
	if m.client == nil {
		m.mu.RUnlock()
		m.trackFailure(fmt.Errorf("statsd client is nil"))
		return nil
	}

	allTags := append(m.tags, tags...)
	client := m.client
	m.mu.RUnlock()

	err := client.Distribution(name, value, allTags, 1)
	m.trackFailure(err)
	return nil
}

// Set sends a set metric
func (m *Metrics) Set(name, value string, tags ...string) error {
	m.mu.RLock()
	if m.client == nil {
		m.mu.RUnlock()
		m.trackFailure(fmt.Errorf("statsd client is nil"))
		return nil
	}

	allTags := append(m.tags, tags...)
	client := m.client
	m.mu.RUnlock()

	err := client.Set(name, value, allTags, 1)
	m.trackFailure(err)
	return nil
}

// RecordAPICall records metrics for an API call
func (m *Metrics) RecordAPICall(endpoint, method string, statusCode int, durationMs float64, hasError bool) error {
	tags := []string{
		fmt.Sprintf("endpoint:%s", endpoint),
		fmt.Sprintf("method:%s", method),
		fmt.Sprintf("status:%d", statusCode),
	}
	if hasError {
		tags = append(tags, "error:true")
	}

	if err := m.Count("api.calls", 1, tags...); err != nil {
		return err
	}

	if err := m.Histogram("api.duration", durationMs, tags...); err != nil {
		return err
	}

	return nil
}

// RecordOperationDuration records the duration of an operation
func (m *Metrics) RecordOperationDuration(operation string, durationMs float64, hasError bool) error {
	tags := []string{
		fmt.Sprintf("operation:%s", operation),
	}
	if hasError {
		tags = append(tags, "status:error")
	} else {
		tags = append(tags, "status:ok")
	}

	return m.Histogram("operation.duration", durationMs, tags...)
}

// RecordExecutionDuration records script execution duration
func (m *Metrics) RecordExecutionDuration(durationMs float64, exitCode int) error {
	tags := []string{
		fmt.Sprintf("exit_code:%d", exitCode),
	}

	return m.Histogram("execution.duration", durationMs, tags...)
}

// Close closes the StatsD client
func (m *Metrics) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.client == nil {
		return nil
	}

	err := m.client.Close()
	m.client = nil
	return err
}

// GetFailureStats returns meta-observability statistics about metrics failures
func (m *Metrics) GetFailureStats() (failedSends uint64, lastFailure string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.failedSends, m.lastFailureMessage
}
