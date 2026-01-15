package log

import (
	"fmt"
	"strings"

	"go.uber.org/zap/zapcore"
)

// Level represents the severity level of a log entry.
type Level string

const (
	// DebugLevel is for verbose debugging information.
	// Typically disabled in production environments.
	DebugLevel Level = "debug"

	// InfoLevel is for general informational messages.
	// This is the default and recommended level for production.
	InfoLevel Level = "info"

	// WarnLevel is for warning messages about potentially harmful situations.
	// More important than Info but doesn't require immediate action.
	WarnLevel Level = "warn"

	// ErrorLevel is for error messages about failures.
	// Applications running smoothly should not generate error-level logs.
	ErrorLevel Level = "error"

	// FatalLevel is for critical errors that cause the application to exit.
	// After logging, the application will call os.Exit(1).
	FatalLevel Level = "fatal"
)

// toZapLevel converts a Level to zapcore.Level.
func (l Level) toZapLevel() (zapcore.Level, error) {
	switch strings.ToLower(string(l)) {
	case "debug":
		return zapcore.DebugLevel, nil
	case "info":
		return zapcore.InfoLevel, nil
	case "warn", "warning":
		return zapcore.WarnLevel, nil
	case "error":
		return zapcore.ErrorLevel, nil
	case "fatal":
		return zapcore.FatalLevel, nil
	default:
		return zapcore.InfoLevel, fmt.Errorf("invalid log level: %s (valid: debug, info, warn, error, fatal)", l)
	}
}

// String returns the string representation of the Level.
func (l Level) String() string {
	return string(l)
}
