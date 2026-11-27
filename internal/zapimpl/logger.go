package zapimpl

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// BuildLogger creates a zap logger based on the provided configuration.
// For file output it also returns the underlying lumberjack sink as an
// io.Closer so the caller can shut it down; for stdout it returns a nil
// closer.
func BuildLogger(service, env string, level zapcore.Level, outputType, filePath string, maxSizeMB, maxBackups, maxAgeDays int) (*zap.Logger, io.Closer, error) {
	// Create encoder config for JSON output
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "", // We'll add caller manually
		FunctionKey:    "", // We'll add function manually
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Create JSON encoder
	encoder := zapcore.NewJSONEncoder(encoderConfig)

	// Create write syncer and closeable sink based on output type
	var writeSyncer zapcore.WriteSyncer
	var sink io.Closer
	if outputType == "file" {
		// Preflight the file target so an unusable path fails here, at
		// construction, instead of dropping the first record on the first write.
		if err := prepareLogFile(filePath); err != nil {
			return nil, nil, fmt.Errorf("prepare log file: %w", err)
		}
		// File output with rotation via lumberjack
		lumberjackLogger := &lumberjack.Logger{
			Filename:   filePath,
			MaxSize:    maxSizeMB,
			MaxBackups: maxBackups,
			MaxAge:     maxAgeDays,
			Compress:   false, // No compression in v1
		}
		writeSyncer = zapcore.AddSync(lumberjackLogger)
		sink = lumberjackLogger
	} else {
		// stdout output; Lock serializes concurrent writes as zap requires
		writeSyncer = zapcore.Lock(zapcore.AddSync(os.Stdout))
	}

	// Create core
	core := zapcore.NewCore(encoder, writeSyncer, level)

	// Build logger
	logger := zap.New(core)

	// Add service and env as default fields
	logger = logger.With(
		zap.String("service", service),
		zap.String("env", env),
	)

	return logger, sink, nil
}

// prepareLogFile ensures the log file target is usable before logging starts:
// it creates parent directories and opens the file with append semantics so
// invalid paths fail at construction rather than losing the first record.
func prepareLogFile(filePath string) error {
	path := filepath.Clean(filePath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create log directory: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close log file after preflight: %w", err)
	}
	return nil
}
