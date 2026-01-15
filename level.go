package log

import (
	"fmt"
	"strings"

	"go.uber.org/zap/zapcore"
)

// Level represents a log level.
type Level string

const (
	// DebugLevel logs are typically voluminous, and are usually disabled in production.
	DebugLevel Level = "debug"
	// InfoLevel is the default logging priority.
	InfoLevel Level = "info"
	// WarnLevel logs are more important than Info, but don't need individual human review.
	WarnLevel Level = "warn"
	// ErrorLevel logs are high-priority. Applications running smoothly shouldn't generate any error-level logs.
	ErrorLevel Level = "error"
	// FatalLevel logs a message, then calls os.Exit(1).
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
