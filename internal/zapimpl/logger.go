package zapimpl

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// BuildLogger creates a zap logger based on the provided configuration.
func BuildLogger(service, env string, level zapcore.Level, outputType, filePath string, maxSizeMB, maxBackups, maxAgeDays int) (*zap.Logger, error) {
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

	// Create write syncer based on output type
	var writeSyncer zapcore.WriteSyncer
	if outputType == "file" {
		// File output with rotation via lumberjack
		lumberjackLogger := &lumberjack.Logger{
			Filename:   filePath,
			MaxSize:    maxSizeMB,
			MaxBackups: maxBackups,
			MaxAge:     maxAgeDays,
			Compress:   false, // No compression in v1
		}
		writeSyncer = zapcore.AddSync(lumberjackLogger)
	} else {
		// stdout output
		writeSyncer = zapcore.AddSync(os.Stdout)
	}

	// Create core
	core := zapcore.NewCore(encoder, writeSyncer, level)

	// Build logger with initial fields
	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(3))

	// Add service and env as default fields
	logger = logger.With(
		zap.String("service", service),
		zap.String("env", env),
	)

	return logger, nil
}
