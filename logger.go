package log

import (
	"fmt"
	"io"
	"strconv"
	"sync"

	"github.com/glennprays/log/internal/zapimpl"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger provides structured logging with required traceId and metadata fields.
// All log methods require a traceId for request traceability and accept optional
// metadata for contextual information.
type Logger struct {
	zapLogger    *zap.Logger
	enableCaller bool // Cached from config for fast runtime access

	// fileSink is the shared lifecycle state for file output. A root logger
	// and every With child hold the same *fileSink, so closing either
	// terminates writes for all of them. It is nil for stdout output and Nop.
	fileSink *fileSink
}

// fileSink guards the closeable file sink shared by a logger and its children.
type fileSink struct {
	mu     sync.RWMutex
	closer io.Closer
	closed bool
}

// New creates a new Logger instance with the provided configuration.
// Returns an error if the configuration is invalid.
//
// Example:
//
//	logger, err := log.New(log.Config{
//	    Service: "my-service",
//	    Env:     "production",
//	    Level:   log.InfoLevel,
//	    Output:  log.OutputStdout,
//	})
func New(cfg Config) (*Logger, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	zapLevel, err := cfg.Level.toZapLevel()
	if err != nil {
		return nil, err
	}

	zapLogger, sink, err := zapimpl.BuildLogger(
		cfg.Service,
		cfg.Env,
		zapLevel,
		string(cfg.Output),
		cfg.FilePath,
		cfg.MaxSizeMB,
		cfg.MaxBackups,
		cfg.MaxAgeDays,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build logger: %w", err)
	}

	l := &Logger{
		zapLogger:    zapLogger,
		enableCaller: cfg.EnableCaller,
	}
	if sink != nil {
		l.fileSink = &fileSink{closer: sink}
	}
	return l, nil
}

// With creates a child logger with pre-bound fields.
// The pre-bound fields will be included in all subsequent log calls from the child logger.
// The parent logger remains unchanged (immutable pattern).
//
// Example:
//
//	userLogger := logger.With(log.String("user_id", "user-456"))
//	userLogger.Info("req-123", "user action", nil)  // includes user_id field
//	logger.Info("req-456", "other action", nil)     // does not include user_id field
//
// Multiple levels of nesting are supported:
//
//	serviceLogger := logger.With(log.String("layer", "api"))
//	userLogger := serviceLogger.With(log.String("user_id", "user-456"))
//	userLogger.Info("req-123", "action", nil)  // includes both layer and user_id
func (l *Logger) With(fields ...Field) *Logger {
	if len(fields) == 0 {
		return l
	}
	validateUserFields(fields)
	zapFields := toZapFields(fields)
	return &Logger{
		zapLogger:    l.zapLogger.With(zapFields...),
		enableCaller: l.enableCaller, // Preserve parent's setting
		fileSink:     l.fileSink,     // Children share the parent's file lifecycle
	}
}

// Debug logs a message at debug level.
//
// Parameters:
//   - traceId: Trace identifier for request traceability (required, panics if empty)
//   - msg: Human-readable log message (required)
//   - metadata: Contextual information (can be nil, always included in output)
//   - fields: Additional structured fields (optional)
//
// Panics if traceId is empty.
func (l *Logger) Debug(traceId string, msg string, metadata any, fields ...Field) {
	l.log(zapcore.DebugLevel, traceId, msg, metadata, fields)
}

// Info logs a message at info level.
//
// Parameters:
//   - traceId: Trace identifier for request traceability (required, panics if empty)
//   - msg: Human-readable log message (required)
//   - metadata: Contextual information (can be nil, always included in output)
//   - fields: Additional structured fields (optional)
//
// Panics if traceId is empty.
func (l *Logger) Info(traceId string, msg string, metadata any, fields ...Field) {
	l.log(zapcore.InfoLevel, traceId, msg, metadata, fields)
}

// Warn logs a message at warn level.
//
// Parameters:
//   - traceId: Trace identifier for request traceability (required, panics if empty)
//   - msg: Human-readable log message (required)
//   - metadata: Contextual information (can be nil, always included in output)
//   - fields: Additional structured fields (optional)
//
// Panics if traceId is empty.
func (l *Logger) Warn(traceId string, msg string, metadata any, fields ...Field) {
	l.log(zapcore.WarnLevel, traceId, msg, metadata, fields)
}

// Error logs a message at error level.
//
// Parameters:
//   - traceId: Trace identifier for request traceability (required, panics if empty)
//   - msg: Human-readable log message (required)
//   - metadata: Contextual information (can be nil, always included in output)
//   - fields: Additional structured fields (optional)
//
// Panics if traceId is empty.
func (l *Logger) Error(traceId string, msg string, metadata any, fields ...Field) {
	l.log(zapcore.ErrorLevel, traceId, msg, metadata, fields)
}

// Fatal logs a message at fatal level, then calls os.Exit(1).
//
// Parameters:
//   - traceId: Trace identifier for request traceability (required, panics if empty)
//   - msg: Human-readable log message (required)
//   - metadata: Contextual information (can be nil, always included in output)
//   - fields: Additional structured fields (optional)
//
// Panics if traceId is empty. After logging, this method calls os.Exit(1).
func (l *Logger) Fatal(traceId string, msg string, metadata any, fields ...Field) {
	l.log(zapcore.FatalLevel, traceId, msg, metadata, fields)
}

// log is the shared event path for all level methods. It validates inputs,
// then consults Zap's level filter (Check) before allocating any field slice
// or capturing caller information, so disabled levels do no unnecessary work.
// Writing through the CheckedEntry preserves Zap's fatal exit action.
func (l *Logger) log(level zapcore.Level, traceId, msg string, metadata any, fields []Field) {
	if traceId == "" {
		panic("log: traceId cannot be empty")
	}
	validateUserFields(fields)

	if l.fileSink == nil {
		l.write(level, traceId, msg, metadata, fields)
		return
	}

	l.fileSink.mu.RLock()
	defer l.fileSink.mu.RUnlock()
	if l.fileSink.closed {
		// Terminal discard: after Close, nonfatal entries are dropped.
		if level == zapcore.FatalLevel {
			// Preserve the fatal-exit contract without touching the sink.
			zap.NewNop().Fatal(msg)
		}
		return
	}
	l.write(level, traceId, msg, metadata, fields)
}

// write performs the actual entry emission through Zap. It must be called
// with the shared file-sink read lock already held (or with no sink).
func (l *Logger) write(level zapcore.Level, traceId, msg string, metadata any, fields []Field) {
	ce := l.zapLogger.Check(level, msg)
	if ce == nil {
		return
	}

	zapFields := make([]zap.Field, 0, len(fields)+3)
	zapFields = append(zapFields, toZapFields(fields)...)
	zapFields = append(zapFields,
		zap.String("trace_id", traceId),
		zap.Any("metadata", metadata),
	)

	// Add caller and function only if enabled. Skip counts from getCaller:
	// +1 runtime.Caller, +2 getCaller, +3 write — landing on the Info/Warn/...
	// entry method's caller (the user's code).
	if l.enableCaller {
		caller := getCaller(3)
		zapFields = append(zapFields,
			zap.String("caller", caller.file+":"+strconv.Itoa(caller.line)),
			zap.String("function", caller.function),
		)
	}

	ce.Write(zapFields...)
}

// isReservedFieldKey reports whether key collides with a JSON field owned by
// the library: timestamp, level, message, service, env, trace_id, metadata,
// caller, or function.
func isReservedFieldKey(key string) bool {
	switch key {
	case "timestamp", "level", "message", "service", "env", "trace_id", "metadata", "caller", "function":
		return true
	}
	return false
}

// validateUserFields panics when a caller-supplied field uses a library-owned
// key. Duplicate JSON keys would let user fields forge or override required
// fields, so a collision is a programming error, never a silent rename.
func validateUserFields(fields []Field) {
	for _, f := range fields {
		if isReservedFieldKey(f.zapField.Key) {
			panic(fmt.Sprintf("log: field key %q is reserved", f.zapField.Key))
		}
	}
}

// Sync flushes any buffered log entries.
// Applications should call Sync before exiting to ensure all logs are written.
// This is the v0.2-compatible explicit flush API and remains fully supported.
// stdout writes are unbuffered, so Sync returns nil for stdout output; only
// file output has a sink to flush. For file output, Sync waits for any
// in-flight write to finish, so it is safe to call concurrently with Close.
// After Close, Sync returns nil.
//
// Example:
//
//	func main() {
//	    logger, _ := log.New(log.Config{...})
//	    defer logger.Close()
//	    // ... application code
//	}
func (l *Logger) Sync() error {
	if l.fileSink == nil {
		// stdout: entries are written with unbuffered write(2) as each call
		// completes, so there is nothing to flush. Syncing os.Stdout would
		// fsync a pipe/tty and fail with EBADF/EINVAL on most terminals.
		return nil
	}
	l.fileSink.mu.RLock()
	defer l.fileSink.mu.RUnlock()
	if l.fileSink.closed {
		return nil
	}
	return l.zapLogger.Sync()
}

// Close releases the file sink backing this logger. It is a no-op returning
// nil for stdout loggers and Nop.
//
// A logger created with file output and every logger derived from it via With
// share the same file sink: calling Close on any one of them terminates
// writes for all of them. Close blocks until an in-flight write or Sync on
// any parent/child logger completes, then closes the sink. The caller must
// stop all parent/child writes before calling Close.
//
// Close is idempotent: repeated calls return nil. If closing the sink fails,
// the error is returned but the sink is still considered closed and
// subsequent nonfatal entries are discarded. After Close, Sync returns nil,
// nonfatal entry methods (Debug, Info, Warn, Error) return without writing,
// and Fatal still exits with status 1 without writing.
func (l *Logger) Close() error {
	if l.fileSink == nil {
		return nil
	}
	l.fileSink.mu.Lock()
	defer l.fileSink.mu.Unlock()
	if l.fileSink.closed {
		return nil
	}
	l.fileSink.closed = true
	return l.fileSink.closer.Close()
}

// nopLogger is the shared no-op logger returned by Nop. A zero fileSink is
// intentional: Nop has no output to close, so Close must be a no-op.
var nopLogger = &Logger{zapLogger: zap.NewNop()}

// Nop returns a logger that validates inputs under the same policy but emits
// nothing. Empty trace IDs and reserved field keys still panic, and Fatal
// still terminates the process with status 1; only record emission and
// internal Zap errors are suppressed. Sync and Close return nil. With on a
// Nop logger stays valid and remains silent. The returned logger is shared:
// With on it produces an independent child, and entry calls on the shared
// instance carry no per-logger state.
func Nop() *Logger {
	return nopLogger
}
