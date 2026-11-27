package log

import (
	"io"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// newBenchmarkLogger constructs a Logger around a Zap core writing to
// io.Discard so benchmarks measure the emission path without producing
// output. Its zero fileSink is intentional: benchmarks target the default
// stdout configuration.
func newBenchmarkLogger(level zapcore.Level, enableCaller bool) *Logger {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		MessageKey:     "message",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
	}
	encoder := zapcore.NewJSONEncoder(encoderConfig)
	core := zapcore.NewCore(encoder, zapcore.AddSync(io.Discard), level)
	zapLogger := zap.New(core).With(
		zap.String("service", "benchmark"),
		zap.String("env", "test"),
	)
	return &Logger{
		zapLogger:    zapLogger,
		enableCaller: enableCaller,
	}
}

// BenchmarkLoggerInfoCallerDisabled measures an info entry with caller
// capture disabled (the production default).
func BenchmarkLoggerInfoCallerDisabled(b *testing.B) {
	logger := newBenchmarkLogger(zapcore.InfoLevel, false)
	b.ReportAllocs()
	for b.Loop() {
		logger.Info("benchmark-trace", "benchmark", map[string]string{"request": "benchmark"}, String("component", "benchmark"))
	}
}

// BenchmarkLoggerInfoCallerEnabled measures an info entry with caller
// capture enabled to make the runtime.Caller cost measurable.
func BenchmarkLoggerInfoCallerEnabled(b *testing.B) {
	logger := newBenchmarkLogger(zapcore.InfoLevel, true)
	b.ReportAllocs()
	for b.Loop() {
		logger.Info("benchmark-trace", "benchmark", map[string]string{"request": "benchmark"}, String("component", "benchmark"))
	}
}

// BenchmarkLoggerDebugFiltered measures a debug entry below the configured
// level: the event path must return before field-slice allocation and
// caller lookup.
func BenchmarkLoggerDebugFiltered(b *testing.B) {
	logger := newBenchmarkLogger(zapcore.WarnLevel, true)
	b.ReportAllocs()
	for b.Loop() {
		logger.Debug("benchmark-trace", "benchmark", map[string]string{"request": "benchmark"}, String("component", "benchmark"))
	}
}
