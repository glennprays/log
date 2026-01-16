package log

import (
	"fmt"

	"github.com/glennprays/log/internal/zapimpl"
	"go.uber.org/zap"
)

// Logger provides structured logging with required requestId and metadata fields.
// All log methods require a requestId for request traceability and accept optional
// metadata for contextual information.
type Logger struct {
	zapLogger *zap.Logger
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

	zapLogger, err := zapimpl.BuildLogger(
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

	return &Logger{
		zapLogger: zapLogger,
	}, nil
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
	zapFields := toZapFields(fields)
	return &Logger{
		zapLogger: l.zapLogger.With(zapFields...),
	}
}

// Debug logs a message at debug level.
//
// Parameters:
//   - requestId: Request identifier for traceability (required, panics if empty)
//   - msg: Human-readable log message (required)
//   - metadata: Contextual information (can be nil, always included in output)
//   - fields: Additional structured fields (optional)
//
// Panics if requestId is empty.
func (l *Logger) Debug(requestId string, msg string, metadata any, fields ...Field) {
	if requestId == "" {
		panic("log: requestId cannot be empty")
	}
	caller := getCaller(1)
	zapFields := toZapFields(fields)
	zapFields = append(zapFields,
		zap.String("request_id", requestId),
		zap.Any("metadata", metadata),
		zap.String("caller", fmt.Sprintf("%s:%d", caller.file, caller.line)),
		zap.String("function", caller.function),
	)
	l.zapLogger.Debug(msg, zapFields...)
}

// Info logs a message at info level.
//
// Parameters:
//   - requestId: Request identifier for traceability (required, panics if empty)
//   - msg: Human-readable log message (required)
//   - metadata: Contextual information (can be nil, always included in output)
//   - fields: Additional structured fields (optional)
//
// Panics if requestId is empty.
func (l *Logger) Info(requestId string, msg string, metadata any, fields ...Field) {
	if requestId == "" {
		panic("log: requestId cannot be empty")
	}
	caller := getCaller(1)
	zapFields := toZapFields(fields)
	zapFields = append(zapFields,
		zap.String("request_id", requestId),
		zap.Any("metadata", metadata),
		zap.String("caller", fmt.Sprintf("%s:%d", caller.file, caller.line)),
		zap.String("function", caller.function),
	)
	l.zapLogger.Info(msg, zapFields...)
}

// Warn logs a message at warn level.
//
// Parameters:
//   - requestId: Request identifier for traceability (required, panics if empty)
//   - msg: Human-readable log message (required)
//   - metadata: Contextual information (can be nil, always included in output)
//   - fields: Additional structured fields (optional)
//
// Panics if requestId is empty.
func (l *Logger) Warn(requestId string, msg string, metadata any, fields ...Field) {
	if requestId == "" {
		panic("log: requestId cannot be empty")
	}
	caller := getCaller(1)
	zapFields := toZapFields(fields)
	zapFields = append(zapFields,
		zap.String("request_id", requestId),
		zap.Any("metadata", metadata),
		zap.String("caller", fmt.Sprintf("%s:%d", caller.file, caller.line)),
		zap.String("function", caller.function),
	)
	l.zapLogger.Warn(msg, zapFields...)
}

// Error logs a message at error level.
//
// Parameters:
//   - requestId: Request identifier for traceability (required, panics if empty)
//   - msg: Human-readable log message (required)
//   - metadata: Contextual information (can be nil, always included in output)
//   - fields: Additional structured fields (optional)
//
// Panics if requestId is empty.
func (l *Logger) Error(requestId string, msg string, metadata any, fields ...Field) {
	if requestId == "" {
		panic("log: requestId cannot be empty")
	}
	caller := getCaller(1)
	zapFields := toZapFields(fields)
	zapFields = append(zapFields,
		zap.String("request_id", requestId),
		zap.Any("metadata", metadata),
		zap.String("caller", fmt.Sprintf("%s:%d", caller.file, caller.line)),
		zap.String("function", caller.function),
	)
	l.zapLogger.Error(msg, zapFields...)
}

// Fatal logs a message at fatal level, then calls os.Exit(1).
//
// Parameters:
//   - requestId: Request identifier for traceability (required, panics if empty)
//   - msg: Human-readable log message (required)
//   - metadata: Contextual information (can be nil, always included in output)
//   - fields: Additional structured fields (optional)
//
// Panics if requestId is empty. After logging, this method calls os.Exit(1).
func (l *Logger) Fatal(requestId string, msg string, metadata any, fields ...Field) {
	if requestId == "" {
		panic("log: requestId cannot be empty")
	}
	caller := getCaller(1)
	zapFields := toZapFields(fields)
	zapFields = append(zapFields,
		zap.String("request_id", requestId),
		zap.Any("metadata", metadata),
		zap.String("caller", fmt.Sprintf("%s:%d", caller.file, caller.line)),
		zap.String("function", caller.function),
	)
	l.zapLogger.Fatal(msg, zapFields...)
}

// Sync flushes any buffered log entries.
// Applications should call Sync before exiting to ensure all logs are written.
//
// Example:
//
//	func main() {
//	    logger, _ := log.New(log.Config{...})
//	    defer logger.Sync()
//	    // ... application code
//	}
func (l *Logger) Sync() error {
	return l.zapLogger.Sync()
}
