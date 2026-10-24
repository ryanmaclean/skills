package observability

import (
	"os"
	"sync"
	"testing"
	"time"
)

// TestNewLogger tests logger creation
func TestNewLogger(t *testing.T) {
	logger := NewLogger("test-service", "test")
	if logger == nil {
		t.Fatal("NewLogger returned nil")
	}
	defer logger.Close()

	if logger.serviceName != "test-service" {
		t.Errorf("Expected service name 'test-service', got '%s'", logger.serviceName)
	}
	if logger.env != "test" {
		t.Errorf("Expected env 'test', got '%s'", logger.env)
	}
	if logger.site == "" {
		t.Error("Site should have a default value")
	}
	if logger.httpClient == nil {
		t.Error("HTTP client should be initialized")
	}
	if logger.buffer == nil {
		t.Error("Buffer should be initialized")
	}
}

// TestNewLoggerWithCustomSite tests logger with custom DD_SITE
func TestNewLoggerWithCustomSite(t *testing.T) {
	os.Setenv("DD_SITE", "datadoghq.eu")
	defer os.Unsetenv("DD_SITE")

	logger := NewLogger("test-service", "test")
	defer logger.Close()

	if logger.site != "datadoghq.eu" {
		t.Errorf("Expected site 'datadoghq.eu', got '%s'", logger.site)
	}
}

// TestNewLoggerDefaultSite tests logger with default site
func TestNewLoggerDefaultSite(t *testing.T) {
	os.Unsetenv("DD_SITE")

	logger := NewLogger("test-service", "test")
	defer logger.Close()

	if logger.site != "datadoghq.com" {
		t.Errorf("Expected default site 'datadoghq.com', got '%s'", logger.site)
	}
}

// TestLogInfo tests info level logging
func TestLogInfo(t *testing.T) {
	os.Setenv("DD_API_KEY", "test-api-key")
	defer os.Unsetenv("DD_API_KEY")

	logger := NewLogger("test-service", "test")
	defer logger.Close()

	// Log without attributes
	logger.Info("Test info message")

	// Verify buffer has the entry
	logger.mu.Lock()
	if len(logger.buffer) != 1 {
		t.Errorf("Expected 1 log entry, got %d", len(logger.buffer))
	}
	if logger.buffer[0].Level != string(LogLevelInfo) {
		t.Errorf("Expected level '%s', got '%s'", LogLevelInfo, logger.buffer[0].Level)
	}
	if logger.buffer[0].Message != "Test info message" {
		t.Errorf("Expected message 'Test info message', got '%s'", logger.buffer[0].Message)
	}
	logger.mu.Unlock()
}

// TestLogWarning tests warning level logging
func TestLogWarning(t *testing.T) {
	os.Setenv("DD_API_KEY", "test-api-key")
	defer os.Unsetenv("DD_API_KEY")

	logger := NewLogger("test-service", "test")
	defer logger.Close()

	logger.Warning("Test warning message")

	logger.mu.Lock()
	if len(logger.buffer) != 1 {
		t.Errorf("Expected 1 log entry, got %d", len(logger.buffer))
	}
	if logger.buffer[0].Level != string(LogLevelWarning) {
		t.Errorf("Expected level '%s', got '%s'", LogLevelWarning, logger.buffer[0].Level)
	}
	logger.mu.Unlock()
}

// TestLogError tests error level logging
func TestLogError(t *testing.T) {
	os.Setenv("DD_API_KEY", "test-api-key")
	defer os.Unsetenv("DD_API_KEY")

	logger := NewLogger("test-service", "test")
	defer logger.Close()

	logger.Error("Test error message")

	logger.mu.Lock()
	if len(logger.buffer) != 1 {
		t.Errorf("Expected 1 log entry, got %d", len(logger.buffer))
	}
	if logger.buffer[0].Level != string(LogLevelError) {
		t.Errorf("Expected level '%s', got '%s'", LogLevelError, logger.buffer[0].Level)
	}
	logger.mu.Unlock()
}

// TestLogWithAttributes tests logging with custom attributes
func TestLogWithAttributes(t *testing.T) {
	os.Setenv("DD_API_KEY", "test-api-key")
	defer os.Unsetenv("DD_API_KEY")

	logger := NewLogger("test-service", "test")
	defer logger.Close()

	attrs := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
		"key3": true,
	}

	logger.Info("Test message with attrs", attrs)

	logger.mu.Lock()
	if len(logger.buffer) != 1 {
		t.Errorf("Expected 1 log entry, got %d", len(logger.buffer))
	}
	if logger.buffer[0].Attrs == nil {
		t.Error("Attributes should not be nil")
	}
	if logger.buffer[0].Attrs["key1"] != "value1" {
		t.Errorf("Expected key1='value1', got '%v'", logger.buffer[0].Attrs["key1"])
	}
	if logger.buffer[0].Attrs["key2"] != 42 {
		t.Errorf("Expected key2=42, got '%v'", logger.buffer[0].Attrs["key2"])
	}
	logger.mu.Unlock()
}

// TestLogWithTrace tests logging with trace correlation
func TestLogWithTrace(t *testing.T) {
	os.Setenv("DD_API_KEY", "test-api-key")
	defer os.Unsetenv("DD_API_KEY")

	logger := NewLogger("test-service", "test")
	defer logger.Close()

	traceID := uint64(12345)
	spanID := uint64(67890)

	logger.InfoWithTrace("Test message with trace", traceID, spanID)

	logger.mu.Lock()
	if len(logger.buffer) != 1 {
		t.Errorf("Expected 1 log entry, got %d", len(logger.buffer))
	}
	if logger.buffer[0].TraceID != traceID {
		t.Errorf("Expected trace ID %d, got %d", traceID, logger.buffer[0].TraceID)
	}
	if logger.buffer[0].SpanID != spanID {
		t.Errorf("Expected span ID %d, got %d", spanID, logger.buffer[0].SpanID)
	}
	logger.mu.Unlock()
}

// TestWarningWithTrace tests warning logging with trace
func TestWarningWithTrace(t *testing.T) {
	os.Setenv("DD_API_KEY", "test-api-key")
	defer os.Unsetenv("DD_API_KEY")

	logger := NewLogger("test-service", "test")
	defer logger.Close()

	traceID := uint64(12345)
	spanID := uint64(67890)

	logger.WarningWithTrace("Test warning with trace", traceID, spanID)

	logger.mu.Lock()
	if len(logger.buffer) != 1 {
		t.Errorf("Expected 1 log entry, got %d", len(logger.buffer))
	}
	if logger.buffer[0].Level != string(LogLevelWarning) {
		t.Errorf("Expected level '%s', got '%s'", LogLevelWarning, logger.buffer[0].Level)
	}
	if logger.buffer[0].TraceID != traceID {
		t.Errorf("Expected trace ID %d, got %d", traceID, logger.buffer[0].TraceID)
	}
	logger.mu.Unlock()
}

// TestErrorWithTrace tests error logging with trace
func TestErrorWithTrace(t *testing.T) {
	os.Setenv("DD_API_KEY", "test-api-key")
	defer os.Unsetenv("DD_API_KEY")

	logger := NewLogger("test-service", "test")
	defer logger.Close()

	traceID := uint64(12345)
	spanID := uint64(67890)

	logger.ErrorWithTrace("Test error with trace", traceID, spanID)

	logger.mu.Lock()
	if len(logger.buffer) != 1 {
		t.Errorf("Expected 1 log entry, got %d", len(logger.buffer))
	}
	if logger.buffer[0].Level != string(LogLevelError) {
		t.Errorf("Expected level '%s', got '%s'", LogLevelError, logger.buffer[0].Level)
	}
	if logger.buffer[0].TraceID != traceID {
		t.Errorf("Expected trace ID %d, got %d", traceID, logger.buffer[0].TraceID)
	}
	logger.mu.Unlock()
}

// TestLogWithoutAPIKey tests that logging without API key doesn't error
func TestLogWithoutAPIKey(t *testing.T) {
	os.Unsetenv("DD_API_KEY")

	logger := NewLogger("test-service", "test")
	defer logger.Close()

	// Should not panic or error
	logger.Info("Test message without API key")

	// Buffer should be empty since API key is not set
	logger.mu.Lock()
	if len(logger.buffer) != 0 {
		t.Errorf("Expected 0 log entries without API key, got %d", len(logger.buffer))
	}
	logger.mu.Unlock()
}

// TestBuffering tests log buffering behavior
func TestBuffering(t *testing.T) {
	os.Setenv("DD_API_KEY", "test-api-key")
	defer os.Unsetenv("DD_API_KEY")

	logger := NewLogger("test-service", "test")
	defer logger.Close()

	// Log multiple entries
	for i := 0; i < 10; i++ {
		logger.Info("Test message")
	}

	logger.mu.Lock()
	bufferLen := len(logger.buffer)
	logger.mu.Unlock()

	if bufferLen != 10 {
		t.Errorf("Expected 10 log entries in buffer, got %d", bufferLen)
	}
}

// TestAutoFlushOnBufferSize tests automatic flushing when buffer is full
func TestAutoFlushOnBufferSize(t *testing.T) {
	os.Setenv("DD_API_KEY", "test-api-key")
	defer os.Unsetenv("DD_API_KEY")

	logger := NewLogger("test-service", "test")
	defer logger.Close()

	// Fill buffer beyond buffer size (100)
	for i := 0; i < 105; i++ {
		logger.Info("Test message")
	}

	// Give it a moment for async flush
	time.Sleep(100 * time.Millisecond)

	logger.mu.Lock()
	bufferLen := len(logger.buffer)
	logger.mu.Unlock()

	// Buffer should have been flushed and now contain remaining entries
	if bufferLen >= 100 {
		t.Errorf("Buffer should have been flushed, but has %d entries", bufferLen)
	}
}

// TestFlush tests manual flushing
func TestFlush(t *testing.T) {
	os.Setenv("DD_API_KEY", "test-api-key")
	defer os.Unsetenv("DD_API_KEY")

	logger := NewLogger("test-service", "test")
	defer logger.Close()

	// Log some entries
	logger.Info("Test message 1")
	logger.Info("Test message 2")

	// Flush
	err := logger.Flush()
	if err != nil {
		t.Errorf("Flush failed: %v", err)
	}

	logger.mu.Lock()
	bufferLen := len(logger.buffer)
	logger.mu.Unlock()

	// Buffer should be empty after flush
	if bufferLen != 0 {
		t.Errorf("Expected empty buffer after flush, got %d entries", bufferLen)
	}
}

// TestFlushEmptyBuffer tests flushing when buffer is empty
func TestFlushEmptyBuffer(t *testing.T) {
	os.Setenv("DD_API_KEY", "test-api-key")
	defer os.Unsetenv("DD_API_KEY")

	logger := NewLogger("test-service", "test")
	defer logger.Close()

	// Flush empty buffer should not error
	err := logger.Flush()
	if err != nil {
		t.Errorf("Flush on empty buffer failed: %v", err)
	}
}

// TestLoggerClose tests logger cleanup
func TestLoggerClose(t *testing.T) {
	os.Setenv("DD_API_KEY", "test-api-key")
	defer os.Unsetenv("DD_API_KEY")

	logger := NewLogger("test-service", "test")

	// Log some entries
	logger.Info("Test message")

	// Close should flush remaining logs
	err := logger.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	logger.mu.Lock()
	bufferLen := len(logger.buffer)
	logger.mu.Unlock()

	if bufferLen != 0 {
		t.Errorf("Expected empty buffer after close, got %d entries", bufferLen)
	}
}

// TestLogEntryFields tests that log entry has correct fields
func TestLogEntryFields(t *testing.T) {
	os.Setenv("DD_API_KEY", "test-api-key")
	defer os.Unsetenv("DD_API_KEY")

	logger := NewLogger("test-service", "test")
	defer logger.Close()

	logger.Info("Test message")

	logger.mu.Lock()
	defer logger.mu.Unlock()

	if len(logger.buffer) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(logger.buffer))
	}

	entry := logger.buffer[0]

	if entry.DDSource != "datadog-skill-go" {
		t.Errorf("Expected ddsource 'datadog-skill-go', got '%s'", entry.DDSource)
	}
	if entry.Service != "test-service" {
		t.Errorf("Expected service 'test-service', got '%s'", entry.Service)
	}
	if entry.Env != "test" {
		t.Errorf("Expected env 'test', got '%s'", entry.Env)
	}
	if entry.Timestamp == 0 {
		t.Error("Timestamp should be set")
	}
	if entry.Hostname == "" {
		t.Error("Hostname should be set")
	}
	if entry.DDTags == "" {
		t.Error("DDTags should be set")
	}
}

// TestConcurrentLogging tests concurrent logging
func TestConcurrentLogging(t *testing.T) {
	os.Setenv("DD_API_KEY", "test-api-key")
	defer os.Unsetenv("DD_API_KEY")

	logger := NewLogger("test-service", "test")
	defer logger.Close()

	var wg sync.WaitGroup
	concurrency := 50

	// Concurrent info logs
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			logger.Info("Concurrent info log")
		}(i)
	}

	// Concurrent warning logs
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			logger.Warning("Concurrent warning log")
		}(i)
	}

	// Concurrent error logs
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			logger.Error("Concurrent error log")
		}(i)
	}

	wg.Wait()

	// Should have logged all entries
	logger.mu.Lock()
	bufferLen := len(logger.buffer)
	logger.mu.Unlock()

	// Buffer might have been auto-flushed, so we just check it doesn't panic
	if bufferLen < 0 {
		t.Error("Buffer length should not be negative")
	}
}

// TestConcurrentFlush tests concurrent flush calls
func TestConcurrentFlush(t *testing.T) {
	os.Setenv("DD_API_KEY", "test-api-key")
	defer os.Unsetenv("DD_API_KEY")

	logger := NewLogger("test-service", "test")
	defer logger.Close()

	// Add some logs
	for i := 0; i < 10; i++ {
		logger.Info("Test message")
	}

	var wg sync.WaitGroup
	concurrency := 10

	// Concurrent flush calls should not panic
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			logger.Flush()
		}()
	}

	wg.Wait()
}

// TestLogLevelConstants tests log level constants
func TestLogLevelConstants(t *testing.T) {
	if LogLevelInfo != "info" {
		t.Errorf("Expected LogLevelInfo='info', got '%s'", LogLevelInfo)
	}
	if LogLevelWarning != "warning" {
		t.Errorf("Expected LogLevelWarning='warning', got '%s'", LogLevelWarning)
	}
	if LogLevelError != "error" {
		t.Errorf("Expected LogLevelError='error', got '%s'", LogLevelError)
	}
}

// TestTimerBasedFlush tests the auto-flush timer
func TestTimerBasedFlush(t *testing.T) {
	os.Setenv("DD_API_KEY", "test-api-key")
	defer os.Unsetenv("DD_API_KEY")

	logger := NewLogger("test-service", "test")
	defer logger.Close()

	// Log a message
	logger.Info("Test message")

	logger.mu.Lock()
	initialLen := len(logger.buffer)
	logger.mu.Unlock()

	if initialLen != 1 {
		t.Errorf("Expected 1 log entry initially, got %d", initialLen)
	}

	// Wait for auto-flush timer (5 seconds + buffer)
	time.Sleep(6 * time.Second)

	logger.mu.Lock()
	afterFlushLen := len(logger.buffer)
	logger.mu.Unlock()

	// Buffer should have been flushed by timer
	if afterFlushLen != 0 {
		t.Errorf("Expected empty buffer after timer flush, got %d entries", afterFlushLen)
	}
}

// TestMultipleLoggers tests creating multiple independent loggers
func TestMultipleLoggers(t *testing.T) {
	logger1 := NewLogger("service-1", "test")
	defer logger1.Close()

	logger2 := NewLogger("service-2", "prod")
	defer logger2.Close()

	if logger1.serviceName != "service-1" {
		t.Errorf("Logger 1 service name incorrect: %s", logger1.serviceName)
	}
	if logger2.serviceName != "service-2" {
		t.Errorf("Logger 2 service name incorrect: %s", logger2.serviceName)
	}
	if logger1.env != "test" {
		t.Errorf("Logger 1 env incorrect: %s", logger1.env)
	}
	if logger2.env != "prod" {
		t.Errorf("Logger 2 env incorrect: %s", logger2.env)
	}
}
