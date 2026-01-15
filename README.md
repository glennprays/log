# log

An opinionated structured logging library for Go that enforces consistency and makes required fields unavoidable.

## Overview

`github.com/glennprays/log` is a policy layer on top of [Zap](https://github.com/uber-go/zap) that provides:

- **Standardized log structure** - Required fields on every log entry
- **Structured logging** - JSON output to stdout by default
- **Collector-friendly** - Works seamlessly with Fluent Bit, Promtail, Vector, and other log collectors
- **Type-safe fields** - Field helpers prevent logging errors
- **Runtime context** - Automatic caller and function name injection
- **Simple API** - Clean interface that hides Zap implementation details

## Status

✅ **Core functionality implemented (v0.1.0)**

⚠️ **This library is in early development (v0.x)**

The API may change before v1.0.0. See [PLAN.md](PLAN.md) for the implementation roadmap.

## Installation

```bash
go get github.com/glennprays/log
```

## Quick Start

```go
package main

import (
    "github.com/glennprays/log"
)

func main() {
    // Create a logger instance
    logger, err := log.New(log.Config{
        Service: "my-service",
        Env:     "dev",
        Level:   log.InfoLevel,
        Output:  log.OutputStdout,
    })
    if err != nil {
        panic(err)
    }

    // Log with required fields
    logger.Info("user logged in",
        log.String("request_id", "abc-123"),
        log.String("user_id", "user-456"),
        log.Any("metadata", map[string]any{
            "ip": "192.168.1.1",
            "method": "POST",
        }),
    )
}
```

**Output** (JSON, one line):
```json
{
  "timestamp": "2025-01-15T10:30:00Z",
  "level": "info",
  "message": "user logged in",
  "service": "my-service",
  "env": "dev",
  "request_id": "abc-123",
  "caller": "main.go:15",
  "function": "main.main",
  "user_id": "user-456",
  "metadata": {
    "ip": "192.168.1.1",
    "method": "POST"
  }
}
```

## Configuration

```go
type Config struct {
    Service    string     // Service name (required)
    Env        string     // Environment: dev, staging, prod (required)
    Level      Level      // Log level: InfoLevel, WarnLevel, etc. (required)
    Output     OutputType // OutputStdout or OutputFile (required)
    FilePath   string     // File path (required if Output is OutputFile)
    MaxSizeMB  int        // Max size in MB before rotation (default: 100)
    MaxBackups int        // Max number of old log files (default: 3)
    MaxAgeDays int        // Max days to retain old logs (default: 28)
}
```

### Output Options

**stdout (default)**:
```go
log.New(log.Config{
    Service: "my-service",
    Env:     "production",
    Level:   log.InfoLevel,
    Output:  log.OutputStdout,
})
```

**File with rotation**:
```go
log.New(log.Config{
    Service:    "my-service",
    Env:        "production",
    Level:      log.InfoLevel,
    Output:     log.OutputFile,
    FilePath:   "/var/log/my-service.log",
    MaxSizeMB:  100,  // Optional: defaults to 100MB
    MaxBackups: 3,    // Optional: defaults to 3
    MaxAgeDays: 28,   // Optional: defaults to 28 days
})
```

## Required vs Optional Fields

### Required Fields (Always Present)

These fields are automatically included in every log entry:

| Field | Source | Description |
|-------|--------|-------------|
| `timestamp` | auto | RFC3339 or epoch timestamp |
| `level` | auto | debug, info, warn, error, fatal |
| `message` | caller | Human-readable log message |
| `service` | config | Service name from Config |
| `env` | config | Environment from Config |
| `request_id` | **explicit** | **Must be provided by caller** |
| `caller` | auto | file:line from runtime.Caller |
| `function` | auto | Function name from runtime |

**Note**: `request_id` must be explicitly passed as a field in every log call. This enforces request traceability.

### Optional Fields (User-Controlled)

Add any additional structured data using field helpers:

```go
logger.Info("processing request",
    log.String("request_id", "abc-123"),      // Required
    log.String("user_id", "user-456"),        // Optional
    log.Int("response_code", 200),            // Optional
    log.Any("metadata", map[string]any{...}), // Optional
    log.Error(err),                           // Optional
)
```

## Log Levels

Supported levels in order of severity:

- `Debug` - Verbose debugging information
- `Info` - General informational messages
- `Warn` - Warning messages for potentially harmful situations
- `Error` - Error messages for failures
- `Fatal` - Critical errors that cause the application to exit (calls `os.Exit`)

```go
logger.Debug("debugging info", log.String("request_id", "abc-123"))
logger.Info("normal operation", log.String("request_id", "abc-123"))
logger.Warn("something unusual", log.String("request_id", "abc-123"))
logger.Error("operation failed", log.String("request_id", "abc-123"), log.Error(err))
logger.Fatal("critical failure", log.String("request_id", "abc-123"), log.Error(err))
```

## Field Helpers

Type-safe field constructors:

```go
log.String(key, value)           // String field
log.Int(key, value)              // Integer field
log.Int64(key, value)            // Int64 field
log.Float64(key, value)          // Float64 field
log.Bool(key, value)             // Boolean field
log.Any(key, value)              // Any type (marshaled as JSON)
log.Error(err)                   // Error field (uses "error" as key)
```

## Best Practices

### Flush Logs on Shutdown

Always call `Sync()` before your application exits to ensure all buffered logs are written:

```go
func main() {
    logger, err := log.New(log.Config{...})
    if err != nil {
        panic(err)
    }
    defer logger.Sync()  // Flush logs on exit

    // Your application code...
}
```

### Log Levels in Production

- Use `InfoLevel` or `WarnLevel` in production
- Reserve `DebugLevel` for development environments
- Use `ErrorLevel` for actual errors that need attention
- Use `FatalLevel` only for unrecoverable errors

### Request ID Pattern

Always pass a request ID for traceability:

```go
// At the start of request handling
requestID := generateRequestID()

// Use it in all log calls
logger.Info("processing request", log.String("request_id", requestID))
logger.Error("request failed", log.String("request_id", requestID), log.Error(err))
```

## Collector Integration

This library outputs structured JSON logs to stdout, making it compatible with:

- **Fluent Bit** - Lightweight log processor and forwarder
- **Promtail** - Log aggregator for Loki
- **Vector** - High-performance observability data pipeline
- Any stdout-based log collector

**Example Docker setup**:
```dockerfile
CMD ["./my-app"]  # Logs go to stdout
# Use Fluent Bit sidecar or Docker logging driver to collect
```

## Security Considerations

**This library does not automatically redact sensitive data.**

You are responsible for:
- Not logging PII (personally identifiable information)
- Not logging secrets (API keys, tokens, passwords)
- Not logging full request/response bodies containing sensitive data
- Using debug-level for verbose fields in production

## Non-Goals

The following are explicitly out of scope for v1:

- OpenTelemetry SDK integration
- Tracing or metrics
- Direct collector integration
- Log storage backends (Elasticsearch, Loki, etc.)
- Business-specific log schemas

This library **only emits logs**. Use external collectors and backends for log aggregation and storage.

## Development

### Run Tests
```bash
go test ./...
go test -v ./...
go test -cover ./...
```

### Build
```bash
go build ./...
```

### Dependencies
```bash
go mod tidy
go mod verify
```

## Design Philosophy

This library prioritizes:

- **Correctness** - Enforce required fields, fail fast on misconfiguration
- **Simplicity** - Boring, predictable, trustworthy
- **Long-term maintainability** - Stable API, internal flexibility
- **Open-source readiness** - Clear versioning and documentation

## Roadmap

See [PLAN.md](PLAN.md) for the detailed 9-step implementation plan.

**Current status: v0.1.0 - Core implementation complete ✅**

Completed steps:
- ✅ Step 1: Initialize Module
- ✅ Step 2: Define Config
- ✅ Step 3: Define Field Abstraction
- ✅ Step 4: Caller & Function Extraction
- ✅ Step 5: Implement zap Wrapper (internal)
- ✅ Step 6: Logger Instance
- ✅ Step 7: File Output + Rotation
- ✅ Step 8: Testing
- ✅ Step 9: Documentation

Next steps for v1.0.0:
- Production testing and feedback
- API stabilization
- Performance benchmarks
- Additional examples and use cases

## License

[To be determined]
