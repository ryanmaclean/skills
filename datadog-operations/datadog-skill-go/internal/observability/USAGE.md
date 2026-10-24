# Observability Library - Quick Usage Guide

## Basic Setup

```go
import "github.com/datadog/skill/internal/observability"

func main() {
    obs, err := observability.Init("dd-cli", "production")
    if err != nil {
        panic(err)
    }
    defer obs.Shutdown(0)
    
    // Your code here
}
```

## Tracing Examples

### Simple Span
```go
span := obs.StartSpan("fetch.data")
defer obs.FinishSpan(span)

obs.GetTracer().SetTag(span, "user.id", "12345")
```

### Context-Based Span
```go
span, ctx := obs.StartSpanFromContext(ctx, "process.request")
defer obs.FinishSpan(span)

// Pass ctx to child functions
result, err := processData(ctx)
```

### Track Operation (Recommended)
```go
err := obs.TrackOperation(ctx, "database.query", func(ctx context.Context, span tracer.Span) error {
    obs.GetTracer().SetTag(span, "query.type", "SELECT")
    return db.Query(ctx, "SELECT * FROM users")
})
```

## Logging Examples

### Basic Logging
```go
obs.LogInfo("Starting process")
obs.LogWarning("Rate limit approaching", map[string]interface{}{
    "current": 90,
    "limit": 100,
})
obs.LogError("Failed to connect", map[string]interface{}{
    "error": err.Error(),
})
```

### Correlated Logging (within span)
```go
span := obs.StartSpan("api.call")
defer obs.FinishSpan(span)

obs.LogInfoWithTrace("API call started", span)
// Automatically includes trace_id and span_id for correlation
```

## Metrics Examples

### Gauge (Current Value)
```go
metrics := obs.GetMetrics()
metrics.Gauge("queue.size", 42.0)
metrics.Gauge("memory.usage.mb", 1024.5, "host:server01")
```

### Counter (Cumulative)
```go
metrics.Count("requests.total", 1)
metrics.Increment("cache.hits")
metrics.Decrement("active.connections")
```

### Histogram (Distribution)
```go
metrics.Histogram("response.time.ms", 123.45)
metrics.Histogram("payload.bytes", 2048.0, "endpoint:/api/users")
```

### Timing (Duration)
```go
metrics.Timing("db.query.ms", 45.2)
```

## Complete Command Pattern

```go
package main

import (
    "context"
    "github.com/datadog/skill/internal/observability"
    "gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

func main() {
    obs, _ := observability.Init("dd-cli", "production")
    defer obs.Shutdown(0)
    
    ctx := context.Background()
    
    err := obs.TrackOperation(ctx, "command.execute", func(ctx context.Context, span tracer.Span) error {
        obs.GetTracer().SetTag(span, "command", "health-check")
        
        // Do work
        obs.LogInfoWithTrace("Checking service health", span)
        
        obs.GetMetrics().Increment("commands.executed", "command:health-check")
        
        return checkHealth(ctx)
    })
    
    if err != nil {
        obs.LogError("Command failed", map[string]interface{}{
            "error": err.Error(),
        })
        obs.Shutdown(1)
    }
}

func checkHealth(ctx context.Context) error {
    // Your implementation
    return nil
}
```

## Environment Variables

```bash
export DD_API_KEY="your-api-key"
export DD_SITE="datadoghq.com"
export DD_ENV="production"
export DD_AGENT_HOST="localhost"
export DD_DOGSTATSD_PORT="8125"
```

## Key Principles

1. Always defer `Shutdown()` to flush data
2. Use `TrackOperation()` for automatic span management
3. Add meaningful tags to spans for filtering
4. Use correlated logging within spans
5. Record both success and failure metrics
6. Observability never breaks your app (graceful degradation)
