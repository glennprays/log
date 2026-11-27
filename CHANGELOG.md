# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.3.0] - 2026-11-26

### Breaking Changes

**Reserved user-field keys now rejected**

Nine JSON keys are now owned by the library and rejected as user fields (via `With()` or any log method's `fields` parameter). Using one panics with `log: field key "<key>" is reserved`:

`timestamp`, `level`, `message`, `service`, `env`, `trace_id`, `metadata`, `caller`, `function`

**Migration**: replace each collision with a distinct business key (e.g. `http_status` instead of `level`). Duplicate JSON keys would otherwise let a user field forge or override a required field in JSON decoders. Note that `error` remains permitted through `log.Error(err)`.

All v0.2 method signatures (`Config`, `Level`, `OutputType`, field helpers, `New`, `With`, `Debug`, `Info`, `Warn`, `Error`, `Fatal`, `Sync`) retain their signatures and behavior; v0.2 consumer source compiles without edits.

### Added

- `Logger.Close()` — concurrency-safe file-sink shutdown. Idempotent; blocks until an in-flight write or `Sync` on the logger or any `With` child completes. No-op for stdout and `Nop`. After close, nonfatal entries are discarded, `Sync` returns `nil`, and `Fatal` still exits with status 1.
- `log.Nop()` — no-op logger preserving input validation (empty trace IDs and reserved keys still panic) and fatal exit status 1, while emitting no records.
- Runnable examples: [`examples/stdout`](examples/stdout/main.go) and [`examples/file`](examples/file/main.go) (artifact-free; `go run` per directory).
- GitHub Actions CI (`quality` job on every push/pull request; `published-install` job on version tags) covering formatting, `go mod verify`, vet, race tests, build, benchmark execution, examples, and v0.2 consumer source compatibility (local replace on PRs; published tag install after release).
- Performance benchmarks: `BenchmarkLoggerInfoCallerDisabled`, `BenchmarkLoggerInfoCallerEnabled`, `BenchmarkLoggerDebugFiltered` (informational; no CI thresholds).

### Fixed

- stdout zero-value default: `Config.Output` omitted now correctly defaults to `OutputStdout`.
- stdout `Sync` no longer attempts to fsync `os.Stdout`, which failed with `sync /dev/stdout: bad file descriptor` when stdout was a pipe or terminal. stdout writes are unbuffered, so `Sync` returns `nil` for stdout output; only file output has a sink to flush.
- Construction-time file preflight: an unusable `FilePath` fails in `New` instead of dropping the first record on first write.
- Concurrent stdout writes are serialized through a locked sink so entries never interleave mid-JSON.
- Disabled-level fast path: entries below the configured level return before field-slice allocation and caller lookup.
- Corrected `log.Error(err)` documentation (key `error`; writes the error's message, plus verbose/cause details where exposed).

---

## [v0.2.0] - 2026-01-21

### Breaking Changes

**1. Request ID renamed to Trace ID**

The `request_id` field and `requestId` parameter have been renamed to `trace_id` and `traceId` for better alignment with distributed tracing standards.

**Migration**: Update all logger method calls to use `traceId`:

```go
// Before (v0.1.0)
logger.Info("req-123", "message", nil)

// After (v0.2.0)
logger.Info("trace-123", "message", nil)  // Same syntax, just semantic change
```

**Impact**:
- JSON output field changed from `"request_id"` to `"trace_id"`
- Parameter name changed from `requestId` to `traceId` in all log methods
- Log collectors, dashboards, and queries depending on `request_id` field must be updated

**2. Caller information now optional**

`caller` (file:line) and `function` fields are now **optional** and **disabled by default** for better production performance. These fields are only included when explicitly enabled via configuration.

**Migration**: To maintain v0.1.0 behavior (include caller information), add `EnableCaller: true` to your Config:

```go
// Before (v0.1.0) - caller/function always present
logger, err := log.New(log.Config{
    Service: "my-service",
    Env:     "production",
    Level:   log.InfoLevel,
    Output:  log.OutputStdout,
})

// After (v0.2.0) - enable caller explicitly
logger, err := log.New(log.Config{
    Service:      "my-service",
    Env:          "production",
    Level:        log.InfoLevel,
    Output:       log.OutputStdout,
    EnableCaller: true,  // Add this line
})
```

**Impact**:
- By default, logs no longer include `caller` and `function` fields
- Log collectors, dashboards, and alerts depending on these fields will need updates OR `EnableCaller: true` must be set
- Performance improved: ~200-500ns faster per log call when disabled (default)

**Why this change?**
- `runtime.Caller()` has measurable overhead in high-throughput services
- Production logs rarely need file:line information
- Dev/staging environments can enable for debugging

### Added

- New `EnableCaller bool` configuration option (default: false)
- Trace ID support for distributed tracing alignment
- Performance improvement: ~200-500ns faster per log call when caller is disabled (default)
- Comprehensive documentation for caller information performance trade-offs

### Changed

- Renamed `request_id` field to `trace_id` in JSON output
- Renamed `requestId` parameter to `traceId` in all log method signatures
- `Logger` struct now includes `enableCaller bool` field (internal change)
- Child loggers created with `With()` preserve parent's `EnableCaller` setting
- Removed unused `zap.AddCaller()` and `zap.AddCallerSkip()` from internal zapimpl
- `caller` and `function` fields moved from required to optional in documentation

### Deprecated

None.

### Removed

None (fields are optional, not removed).

### Fixed

None.

### Security

No security changes.

---

## [v0.1.0] - 2025-01-XX

### Added

- Initial release
- Structured logging with Zap backend
- Required fields: service, env, timestamp, level, message, request_id, metadata, caller, function
- JSON output to stdout
- File output with rotation via lumberjack
- Field helpers: String, Int, Int64, Float64, Bool, Any, Error
- Child loggers with `With()` method
- Comprehensive test suite
