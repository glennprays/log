# log

An opinionated structured logging library for Go that enforces consistency and makes required fields unavoidable.

## Overview

`github.com/glennprays/log` is a policy layer on top of [Zap](https://github.com/uber-go/zap) that provides:

- **Standardized log structure** - Required fields on every log entry
- **Structured logging** - JSON output to stdout by default
- **Collector-friendly** - Works seamlessly with Fluent Bit, Promtail, Vector, and other log collectors
- **Type-safe fields** - Field helpers prevent logging errors
- **Optional runtime context** - Caller and function name injection when enabled
- **Simple API** - Clean interface that hides Zap implementation details

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
        Service:      "my-service",
        Env:          "dev",
        Level:        log.InfoLevel,
        Output:       log.OutputStdout,
        EnableCaller: true,  // Enable for dev debugging
    })
    if err != nil {
        panic(err)
    }

    // Log with traceId, metadata, and additional fields
    logger.Info(
        "abc-123",                              // traceId (required)
        "user logged in",                       // message (required)
        map[string]any{                         // metadata (optional, can be nil)
            "ip": "192.168.1.1",
            "method": "POST",
        },
        log.String("user_id", "user-456"),     // additional fields
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
  "trace_id": "abc-123",
  "caller": "main.go:15",
  "function": "main.main",
  "user_id": "user-456",
  "metadata": {
    "ip": "192.168.1.1",
    "method": "POST"
  }
}
```

**Note**: `caller` and `function` fields are only included when `EnableCaller: true` is set in Config.

## Configuration

```go
type Config struct {
    Service      string     // Service name (required)
    Env          string     // Environment: dev, staging, prod (required)
    Level        Level      // Log level: InfoLevel, WarnLevel, etc. (required)
    Output       OutputType // OutputStdout or OutputFile (required)
    FilePath     string     // File path (required if Output is OutputFile)
    MaxSizeMB    int        // Max size in MB before rotation (default: 100)
    MaxBackups   int        // Max number of old log files (default: 3)
    MaxAgeDays   int        // Max days to retain old logs (default: 28)
    EnableCaller bool       // Enable caller/function extraction (default: false)
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
| `message` | parameter | Human-readable log message (required parameter) |
| `service` | config | Service name from Config |
| `env` | config | Environment from Config |
| `trace_id` | **parameter** | **Required parameter - must be provided** |
| `metadata` | parameter | Contextual data (required parameter, can be nil) |

**Note**: `trace_id` and `metadata` are required parameters in all log methods. Empty `trace_id` will cause a panic.

### Optional Auto-Generated Fields

These fields are automatically included when enabled via configuration:

| Field | Source | Description | Config |
|-------|--------|-------------|--------|
| `caller` | auto | file:line from runtime.Caller | `EnableCaller: true` |
| `function` | auto | Function name from runtime | `EnableCaller: true` |

**Performance Note**: Caller extraction uses `runtime.Caller()` which has overhead (~200-500ns per call). Disable in production for better performance, enable in dev/staging for debugging.

### Optional Fields (User-Controlled)

Add any additional structured data using field helpers:

```go
logger.Info(
    "req-123",                                 // traceId (required)
    "processing request",                      // message (required)
    map[string]any{"trace": "xyz"},           // metadata (required, can be nil)
    log.String("user_id", "user-456"),        // additional field
    log.Int("response_code", 200),            // additional field
    log.Error(err),                           // additional field
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
logger.Debug("req-123", "debugging info", nil)
logger.Info("req-123", "normal operation", nil)
logger.Warn("req-123", "something unusual", nil)
logger.Error("req-123", "operation failed", nil, log.Error(err))
logger.Fatal("req-123", "critical failure", nil, log.Error(err))
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

## Child Loggers with Pre-bound Fields

Create child loggers with pre-bound fields using the `With()` method. This is useful for adding contextual fields that apply to multiple log calls:

### Basic Usage

```go
// Create a child logger with pre-bound user context
userLogger := logger.With(
    log.String("user_id", "user-456"),
    log.String("session_id", "sess-789"),
)

// All logs from child logger include pre-bound fields
userLogger.Info("req-123", "user logged in", nil)
userLogger.Info("req-456", "user updated profile", nil)

// Original logger is unchanged
logger.Info("req-789", "system event", nil)  // does not include user fields
```

### Nested Child Loggers

Multiple levels of nesting are supported. Fields accumulate from all parent loggers:

```go
serviceLogger := logger.With(log.String("layer", "api"))
userLogger := serviceLogger.With(log.String("user_id", "user-456"))
actionLogger := userLogger.With(log.String("action", "purchase"))

// Logs include all accumulated fields: layer, user_id, action
actionLogger.Info("req-123", "processing", nil)
```

### Benefits

- **Reduce repetition** - Set common fields once instead of on every log call
- **Contextual logging** - Create loggers for specific components or operations
- **Immutable** - Parent logger remains unchanged
- **Composable** - Build loggers with accumulating context

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

### Trace ID and Empty String Validation

The traceId parameter is required and cannot be empty. Empty strings will cause a panic:

```go
traceID := generateTraceID()

// Correct - traceId is provided
logger.Info(traceID, "processing request", nil)

// PANIC - empty traceId
logger.Info("", "this will panic", nil)  // panic: log: traceId cannot be empty
```

### Metadata vs Fields

**When to use metadata:**
- Contextual information that applies to the entire request
- Request-level data: IP address, user agent, HTTP method, path, headers
- Example: `map[string]any{"ip": "192.168.1.1", "method": "GET", "path": "/api/users"}`

**When to use fields:**
- Specific log entry details
- Business data: user_id, order_id, product_id, response_code
- Example: `log.String("user_id", "user-123"), log.Int("response_code", 200)`

**Using nil metadata:**
```go
// Simple logs without contextual information
logger.Info("req-123", "cache hit", nil)
logger.Debug("req-123", "processing step 1", nil)
```

## Caller Information

The library can automatically include caller information (`caller` and `function` fields) in every log entry by enabling it in the configuration.

### Configuration

```go
logger, err := log.New(log.Config{
    Service:      "my-service",
    Env:          "production",
    Level:        log.InfoLevel,
    Output:       log.OutputStdout,
    EnableCaller: true,  // Enable caller extraction
})
```

### Performance Considerations

**Caller extraction has overhead**:
- Uses `runtime.Caller()` which costs ~200-500ns per log call
- In high-throughput services, this can add up
- Recommended: Disable in production, enable in dev/staging

**When to enable**:
- Development environments for debugging
- Staging environments for troubleshooting
- Low-traffic production services where overhead is negligible

**When to disable**:
- High-throughput production services
- Performance-critical code paths
- When log volume is very high

### Example Outputs

**With EnableCaller: true**:
```json
{
  "timestamp": "2025-01-15T10:30:00Z",
  "level": "info",
  "message": "user logged in",
  "service": "my-service",
  "env": "dev",
  "trace_id": "abc-123",
  "caller": "handler.go:45",
  "function": "handlers.LoginHandler",
  "metadata": null
}
```

**With EnableCaller: false (default)**:
```json
{
  "timestamp": "2025-01-15T10:30:00Z",
  "level": "info",
  "message": "user logged in",
  "service": "my-service",
  "env": "production",
  "trace_id": "abc-123",
  "metadata": null
}
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

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details
