# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is **github.com/glennprays/log**, an opinionated structured logging library for Go 1.25 that acts as a policy layer on top of Zap. The library enforces standardized log structure with required fields and outputs collector-friendly JSON logs to stdout.

**Core Philosophy**: Make required fields unavoidable, ensure predictable machine-readable logs, hide implementation details (Zap), and maintain a stable public API suitable for open source.

**Current State**: Early planning phase - only foundation files exist. Implementation follows the 9-step plan in PLAN.md.

## Development Commands

### Testing
```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -cover ./...

# Run specific test
go test -run TestName ./...
```

### Building
```bash
# Build the module
go build ./...

# Verify module integrity
go mod verify

# Update dependencies
go mod tidy
```

### Linting (when configured)
```bash
# Run golangci-lint (if configured)
golangci-lint run
```

## Architecture & Design Principles

### Public API Design

**Logger Instance Model** - No global functions:
```go
l, err := log.New(log.Config{...})
l.Info("message", log.String("request_id", "abc-123"))
```

**Field Abstraction** - Wrapper around Zap fields:
- `log.String(key, value)`
- `log.Int(key, value)`
- `log.Any(key, value)`
- `log.Error(err)`

**Zap Must Remain Internal** - The `internal/zapimpl/` package wraps Zap. Never expose `zap.Field`, `zap.Logger`, or any Zap types in the public API.

### Required vs Optional Fields

**Required Fields** (automatically injected on every log entry):
- `timestamp` - auto-generated (RFC3339/epoch)
- `level` - auto (debug/info/warn/error/fatal)
- `message` - caller-provided
- `service` - from config
- `env` - from config (dev/staging/prod)
- `request_id` - **must be explicitly provided by caller**
- `caller` - auto (file:line via runtime.Caller)
- `function` - auto (extracted from runtime)

**Optional Fields** (user-controlled):
- `metadata` - flexible key-value map
- `request`, `response`, `headers` - structured data
- `error` - error details
- Custom key-value pairs

This distinction is fundamental to the library design. Required fields ensure log consistency; optional fields provide flexibility.

### Output Strategy

**Default**: JSON to stdout (one entry per line) - collector-friendly, no ANSI formatting

**Optional**: File output with rotation via lumberjack (configured in Config)

**Target Collectors**: Fluent Bit, Promtail, Vector - any stdout-based collector

### Internal Structure

```
log/
├── Public API files (logger.go, config.go, field.go, level.go, output.go, caller.go, error.go)
├── internal/zapimpl/     # Zap wrapper (NEVER expose publicly)
└── tests/                # Test capture via buffers, assert required fields
```

## Critical Constraints

### Non-Goals (v1 Scope Boundaries)
The following are **explicitly out of scope** for v1:
- OpenTelemetry SDK
- Tracing or metrics
- Direct integration with collectors (Fluent Bit/Promtail)
- Log storage backends (Elastic, Loki, etc.)
- Business-specific log schemas

This library **only emits logs**. Do not add features in these categories.

### Security & Logging Policy
- **No automatic redaction** - caller is responsible for sensitive data (PII, secrets, request bodies)
- Encourage minimal request/response logging
- Recommend verbose fields be debug-level only

### Implementation Principles
- **Correctness over convenience** - enforce required fields, fail fast on misconfiguration
- **Simplicity** - boring, predictable, trustworthy
- **Long-term maintainability** - stable API, internal flexibility
- **Open-source readiness** - clear versioning (v0.x → v1.0.0)

## Implementation Plan

The project follows a detailed 9-step implementation plan documented in **PLAN.md**:

1. Initialize Module (✓ completed)
2. Define Config
3. Define Field Abstraction
4. Caller & Function Extraction
5. Implement Zap Wrapper (internal)
6. Logger Instance
7. File Output + Rotation
8. Testing
9. Documentation

Refer to PLAN.md for detailed requirements and design decisions for each step.

## Key Design Patterns

### Runtime Caller Extraction
Use `runtime.Caller` to automatically inject `caller` (file:line) and `function` name into every log entry. This must happen transparently without caller intervention.

### Config Validation
Strictly validate Config on logger creation. Fail fast with clear errors for invalid service names, environments, log levels, or file paths.

### Field Helper Pattern
All fields use helper functions (`log.String`, `log.Int`, etc.) that internally create Zap fields but are exposed as opaque `Field` types in the public API.

### Logger Methods Signature
```go
func (l *Logger) Info(msg string, fields ...Field)
```
Message is required and separate from fields. This enforces human-readable log messages.
