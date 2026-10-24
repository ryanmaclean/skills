# Datadog Client Testing Guide

## Quick Start

Run all tests with coverage:
```bash
go test -v -cover ./internal/client/...
```

## Test Organization

### Test File Structure
```
internal/client/
├── datadog.go           # Implementation (992 lines, 55+ methods)
├── datadog_test.go      # Unit tests (1,897 lines, 89 tests)
├── TEST_SUMMARY.md      # This comprehensive test documentation
└── TESTING.md           # Quick reference guide
```

## Running Tests

### Basic Commands

```bash
# Run all tests
go test ./internal/client/...

# Run with verbose output
go test -v ./internal/client/...

# Run with coverage
go test -cover ./internal/client/...

# Run with detailed coverage report
go test -coverprofile=coverage.out ./internal/client/...
go tool cover -html=coverage.out

# Run specific test
go test -run TestQueryAPM ./internal/client/...

# Run test group (all incident tests)
go test -run TestIncidentOperations ./internal/client/...

# Run without cache
go test -count=1 ./internal/client/...
```

### Test Selection

```bash
# Run only client initialization tests
go test -run TestNewClient ./internal/client/...

# Run only HTTP infrastructure tests
go test -run "Test(DoRequest|RateLimit|Retry)" ./internal/client/...

# Run only API method tests
go test -run "Test(Query|Get|Create|Update|Delete)" ./internal/client/...

# Run only error handling tests
go test -run "Test(DatadogError|ParseError)" ./internal/client/...
```

### Coverage Analysis

```bash
# Generate coverage profile
go test -coverprofile=coverage.out ./internal/client/...

# View coverage in terminal
go tool cover -func=coverage.out

# View coverage in browser (HTML)
go tool cover -html=coverage.out

# Get coverage percentage
go test -cover ./internal/client/... | grep coverage
```

## Test Categories

### 1. Infrastructure Tests
```bash
go test -run "Test(NewClient|BuildHeaders|DoRequest)" ./internal/client/...
```
- Client initialization
- Header building
- HTTP request handling

### 2. Retry & Error Tests
```bash
go test -run "Test(RateLimit|Retry|MaxRetries|ParseError)" ./internal/client/...
```
- Rate limiting (429)
- 5xx retries
- Max retry limits
- Error parsing

### 3. Observability Tests
```bash
go test -run "Test(QueryAPM|SearchLogs|QueryMetrics)" ./internal/client/...
```
- APM traces
- Log search
- Metrics queries

### 4. Incident Management Tests
```bash
go test -run "TestIncidentOperations" ./internal/client/...
```
- Create incidents
- List incidents
- Update incidents
- Add timeline entries

### 5. Monitoring Tests
```bash
go test -run "TestMonitorOperations" ./internal/client/...
```
- Create monitors
- Mute/unmute monitors
- Delete monitors

### 6. Synthetic Tests
```bash
go test -run "TestSyntheticOperations" ./internal/client/...
```
- Create/update/delete synthetic tests
- Get test results
- Pause/resume tests

### 7. RUM Tests
```bash
go test -run "TestRUMOperations" ./internal/client/...
```
- RUM applications
- Views, sessions, errors
- Performance metrics

### 8. CI/CD Tests
```bash
go test -run "TestCICDOperations" ./internal/client/...
```
- Pipeline analytics
- Test analytics
- Failed tests

## Test Output

### Successful Test Run
```
=== RUN   TestNewClient
=== RUN   TestNewClient/valid_credentials_with_default_site
=== RUN   TestNewClient/valid_credentials_with_US_site
--- PASS: TestNewClient (0.00s)
    --- PASS: TestNewClient/valid_credentials_with_default_site (0.00s)
    --- PASS: TestNewClient/valid_credentials_with_US_site (0.00s)
PASS
ok      github.com/datadog/skill/internal/client    10.229s
```

### Coverage Output
```
ok      github.com/datadog/skill/internal/client    10.431s coverage: 66.3% of statements
```

## Debugging Tests

### Run with Race Detector
```bash
go test -race ./internal/client/...
```

### Run with CPU Profiling
```bash
go test -cpuprofile=cpu.prof ./internal/client/...
go tool pprof cpu.prof
```

### Run with Memory Profiling
```bash
go test -memprofile=mem.prof ./internal/client/...
go tool pprof mem.prof
```

### Run with Timeout
```bash
go test -timeout 30s ./internal/client/...
```

## Test Patterns

### Table-Driven Tests
```go
tests := []struct {
    name    string
    apiKey  string
    appKey  string
    wantErr bool
}{
    {
        name:    "valid credentials",
        apiKey:  "test_api_key",
        appKey:  "test_app_key",
        wantErr: false,
    },
    // ... more test cases
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // Test implementation
    })
}
```

### Mock HTTP Server
```go
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    // Verify request
    if r.Method != "POST" {
        t.Errorf("Expected POST, got %s", r.Method)
    }

    // Return mock response
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"data": []}`))
}))
defer server.Close()

client := &Client{
    apiKey:     "test",
    appKey:     "test",
    baseURL:    server.URL,
    httpClient: &http.Client{},
}
```

### Subtests
```go
t.Run("CreateMonitor", func(t *testing.T) {
    // Test implementation
})

t.Run("MuteMonitor", func(t *testing.T) {
    // Test implementation
})
```

## Continuous Integration

### GitHub Actions Example
```yaml
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - run: go test -v -cover ./internal/client/...
```

### Pre-commit Hook
```bash
#!/bin/bash
# .git/hooks/pre-commit
go test ./internal/client/... || exit 1
```

## Best Practices

1. **Always run tests before committing**
   ```bash
   go test ./internal/client/...
   ```

2. **Check coverage regularly**
   ```bash
   go test -cover ./internal/client/...
   ```

3. **Run with race detector for concurrency issues**
   ```bash
   go test -race ./internal/client/...
   ```

4. **Use subtests for related test cases**
   ```go
   t.Run("subtest_name", func(t *testing.T) { ... })
   ```

5. **Clean up resources with defer**
   ```go
   defer server.Close()
   defer os.Unsetenv("DD_API_KEY")
   ```

## Test Maintenance

### Adding New Tests

1. Add test function to `datadog_test.go`
2. Use table-driven pattern when appropriate
3. Use mock HTTP server for API calls
4. Add test to appropriate category in TEST_SUMMARY.md
5. Run tests to verify: `go test -v ./internal/client/...`

### Updating Existing Tests

1. Modify test in `datadog_test.go`
2. Ensure backward compatibility
3. Update TEST_SUMMARY.md if needed
4. Run full test suite: `go test -v ./internal/client/...`

## Troubleshooting

### Test Failures

**Symptom:** Tests fail with timeout errors
**Solution:** Increase timeout or check network connectivity
```bash
go test -timeout 60s ./internal/client/...
```

**Symptom:** Tests fail with "address already in use"
**Solution:** Mock servers not cleaning up properly
```bash
# Ensure defer server.Close() is present
defer server.Close()
```

**Symptom:** Flaky tests (sometimes pass, sometimes fail)
**Solution:** Check for race conditions
```bash
go test -race ./internal/client/...
```

## Performance

- **Test Suite Duration:** ~10 seconds
- **Average Test Duration:** ~0.1 seconds
- **Slowest Tests:** Retry logic tests (3 seconds each due to delays)

## Resources

- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [httptest Package](https://golang.org/pkg/net/http/httptest/)
- [Testing in Go (Best Practices)](https://go.dev/doc/tutorial/add-a-test)
- Datadog API Documentation: https://docs.datadoghq.com/api/

## Test Statistics Summary

- **Total Tests:** 89
- **Test Coverage:** 66.3%
- **Test File Size:** 1,897 lines
- **Implementation Size:** 992 lines
- **Test-to-Code Ratio:** 1.91:1
- **API Methods Tested:** 55+
- **Mock Servers Used:** 72
- **Error Scenarios:** 15+
