package log

import (
	"fmt"

	"github.com/glennprays/log/internal/zapimpl"
	"go.uber.org/zap"
)

// Logger provides structured logging with required fields.
type Logger struct {
	zapLogger *zap.Logger
}

// New creates a new Logger instance with the provided configuration.
func New(cfg Config) (*Logger, error) {
	// Validate config
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Convert level to zap level
	zapLevel, err := cfg.Level.toZapLevel()
	if err != nil {
		return nil, err
	}

	// Build zap logger
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

// Debug logs a message at debug level with optional fields.
func (l *Logger) Debug(msg string, fields ...Field) {
	caller := getCaller(1)
	zapFields := toZapFields(fields)
	zapFields = append(zapFields,
		zap.String("caller", fmt.Sprintf("%s:%d", caller.file, caller.line)),
		zap.String("function", caller.function),
	)
	l.zapLogger.Debug(msg, zapFields...)
}

// Info logs a message at info level with optional fields.
func (l *Logger) Info(msg string, fields ...Field) {
	caller := getCaller(1)
	zapFields := toZapFields(fields)
	zapFields = append(zapFields,
		zap.String("caller", fmt.Sprintf("%s:%d", caller.file, caller.line)),
		zap.String("function", caller.function),
	)
	l.zapLogger.Info(msg, zapFields...)
}

// Warn logs a message at warn level with optional fields.
func (l *Logger) Warn(msg string, fields ...Field) {
	caller := getCaller(1)
	zapFields := toZapFields(fields)
	zapFields = append(zapFields,
		zap.String("caller", fmt.Sprintf("%s:%d", caller.file, caller.line)),
		zap.String("function", caller.function),
	)
	l.zapLogger.Warn(msg, zapFields...)
}

// Error logs a message at error level with optional fields.
func (l *Logger) Error(msg string, fields ...Field) {
	caller := getCaller(1)
	zapFields := toZapFields(fields)
	zapFields = append(zapFields,
		zap.String("caller", fmt.Sprintf("%s:%d", caller.file, caller.line)),
		zap.String("function", caller.function),
	)
	l.zapLogger.Error(msg, zapFields...)
}

// Fatal logs a message at fatal level with optional fields, then calls os.Exit(1).
func (l *Logger) Fatal(msg string, fields ...Field) {
	caller := getCaller(1)
	zapFields := toZapFields(fields)
	zapFields = append(zapFields,
		zap.String("caller", fmt.Sprintf("%s:%d", caller.file, caller.line)),
		zap.String("function", caller.function),
	)
	l.zapLogger.Fatal(msg, zapFields...)
}

// Sync flushes any buffered log entries. Applications should call Sync before exiting.
func (l *Logger) Sync() error {
	return l.zapLogger.Sync()
}
