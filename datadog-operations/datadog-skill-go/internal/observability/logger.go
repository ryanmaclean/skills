package observability

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

// LogLevel represents the severity level of a log entry
type LogLevel string

const (
	LogLevelInfo    LogLevel = "info"
	LogLevelWarning LogLevel = "warning"
	LogLevelError   LogLevel = "error"
)

// LogEntry represents a single log entry to be sent to Datadog
type LogEntry struct {
	DDSource  string                 `json:"ddsource"`
	DDTags    string                 `json:"ddtags"`
	Hostname  string                 `json:"hostname"`
	Message   string                 `json:"message"`
	Level     string                 `json:"level"`
	Timestamp int64                  `json:"timestamp"`
	Service   string                 `json:"service"`
	Env       string                 `json:"env"`
	TraceID   uint64                 `json:"dd.trace_id,omitempty"`
	SpanID    uint64                 `json:"dd.span_id,omitempty"`
	Attrs     map[string]interface{} `json:"attributes,omitempty"`
}

// Logger provides structured logging to Datadog
type Logger struct {
	serviceName        string
	env                string
	apiKey             string
	site               string
	hostname           string
	httpClient         *http.Client
	buffer             []*LogEntry
	bufferSize         int
	mu                 sync.Mutex
	flushTimer         *time.Timer
	failedSends        uint64 // Counter for failed HTTP sends
	failedMarshals     uint64 // Counter for failed JSON marshals
	lastFailureMessage string // Last failure message for debugging
}

// NewLogger creates a new Logger instance
func NewLogger(serviceName, env string) *Logger {
	apiKey := os.Getenv("DD_API_KEY")
	site := os.Getenv("DD_SITE")
	if site == "" {
		site = "datadoghq.com"
	}

	hostname, _ := os.Hostname()

	logger := &Logger{
		serviceName: serviceName,
		env:         env,
		apiKey:      apiKey,
		site:        site,
		hostname:    hostname,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		buffer:     make([]*LogEntry, 0, 100),
		bufferSize: 100,
	}

	// Start auto-flush timer
	logger.startFlushTimer()

	return logger
}

// startFlushTimer starts a timer to periodically flush logs
func (l *Logger) startFlushTimer() {
	l.flushTimer = time.AfterFunc(5*time.Second, func() {
		l.Flush()
		l.startFlushTimer()
	})
}

// Info logs an informational message
func (l *Logger) Info(message string, attrs ...map[string]interface{}) {
	l.log(LogLevelInfo, message, 0, 0, attrs...)
}

// Warning logs a warning message
func (l *Logger) Warning(message string, attrs ...map[string]interface{}) {
	l.log(LogLevelWarning, message, 0, 0, attrs...)
}

// Error logs an error message
func (l *Logger) Error(message string, attrs ...map[string]interface{}) {
	l.log(LogLevelError, message, 0, 0, attrs...)
}

// InfoWithTrace logs an informational message with trace correlation
func (l *Logger) InfoWithTrace(message string, traceID, spanID uint64, attrs ...map[string]interface{}) {
	l.log(LogLevelInfo, message, traceID, spanID, attrs...)
}

// WarningWithTrace logs a warning message with trace correlation
func (l *Logger) WarningWithTrace(message string, traceID, spanID uint64, attrs ...map[string]interface{}) {
	l.log(LogLevelWarning, message, traceID, spanID, attrs...)
}

// ErrorWithTrace logs an error message with trace correlation
func (l *Logger) ErrorWithTrace(message string, traceID, spanID uint64, attrs ...map[string]interface{}) {
	l.log(LogLevelError, message, traceID, spanID, attrs...)
}

// log creates and buffers a log entry
func (l *Logger) log(level LogLevel, message string, traceID, spanID uint64, attrs ...map[string]interface{}) {
	if l.apiKey == "" {
		return
	}

	entry := &LogEntry{
		DDSource:  "datadog-skill-go",
		DDTags:    fmt.Sprintf("env:%s,service:%s", l.env, l.serviceName),
		Hostname:  l.hostname,
		Message:   message,
		Level:     string(level),
		Timestamp: time.Now().UnixMilli(),
		Service:   l.serviceName,
		Env:       l.env,
	}

	if traceID != 0 {
		entry.TraceID = traceID
	}
	if spanID != 0 {
		entry.SpanID = spanID
	}

	if len(attrs) > 0 {
		entry.Attrs = attrs[0]
	}

	l.mu.Lock()
	l.buffer = append(l.buffer, entry)
	shouldFlush := len(l.buffer) >= l.bufferSize
	l.mu.Unlock()

	if shouldFlush {
		l.Flush()
	}
}

// Flush sends all buffered logs to Datadog
func (l *Logger) Flush() error {
	l.mu.Lock()
	if len(l.buffer) == 0 {
		l.mu.Unlock()
		return nil
	}

	toSend := l.buffer
	l.buffer = make([]*LogEntry, 0, l.bufferSize)
	l.mu.Unlock()

	return l.sendLogs(toSend)
}

// sendLogs sends a batch of logs to Datadog HTTP intake API
func (l *Logger) sendLogs(entries []*LogEntry) error {
	if l.apiKey == "" {
		return nil
	}

	payload, err := json.Marshal(entries)
	if err != nil {
		l.mu.Lock()
		l.failedMarshals++
		l.lastFailureMessage = fmt.Sprintf("marshal error: %v", err)
		l.mu.Unlock()
		// Log to stderr for debugging
		fmt.Fprintf(os.Stderr, "datadog-cli: logger failed to marshal logs: %v\n", err)
		return nil // Don't fail - observability should never break the application
	}

	url := fmt.Sprintf("https://http-intake.logs.%s/api/v2/logs", l.site)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		l.mu.Lock()
		l.failedSends++
		l.lastFailureMessage = fmt.Sprintf("request creation error: %v", err)
		l.mu.Unlock()
		fmt.Fprintf(os.Stderr, "datadog-cli: logger failed to create request: %v\n", err)
		return nil
	}

	req.Header.Set("DD-API-KEY", l.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := l.httpClient.Do(req)
	if err != nil {
		l.mu.Lock()
		l.failedSends++
		l.lastFailureMessage = fmt.Sprintf("HTTP error: %v", err)
		l.mu.Unlock()
		fmt.Fprintf(os.Stderr, "datadog-cli: logger failed to send logs: %v\n", err)
		return nil // Don't fail - observability should never break the application
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		l.mu.Lock()
		l.failedSends++
		l.lastFailureMessage = fmt.Sprintf("HTTP status %d", resp.StatusCode)
		l.mu.Unlock()
		fmt.Fprintf(os.Stderr, "datadog-cli: logger received non-200 status: %d\n", resp.StatusCode)
		return nil // Don't fail - observability should never break the application
	}

	return nil
}

// Close stops the flush timer and flushes remaining logs
func (l *Logger) Close() error {
	if l.flushTimer != nil {
		l.flushTimer.Stop()
	}
	return l.Flush()
}

// GetFailureStats returns meta-observability statistics about logger failures
func (l *Logger) GetFailureStats() (failedSends, failedMarshals uint64, lastFailure string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.failedSends, l.failedMarshals, l.lastFailureMessage
}
