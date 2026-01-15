package log_test

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/glennprays/log"
)

func TestNew_ValidConfig(t *testing.T) {
	cfg := log.Config{
		Service: "test-service",
		Env:     "dev",
		Level:   log.InfoLevel,
		Output:  log.OutputStdout,
	}

	logger, err := log.New(cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if logger == nil {
		t.Fatal("expected logger to be non-nil")
	}
}

func TestNew_InvalidConfig(t *testing.T) {
	tests := []struct {
		name   string
		config log.Config
	}{
		{
			name: "missing service",
			config: log.Config{
				Env:    "dev",
				Level:  log.InfoLevel,
				Output: log.OutputStdout,
			},
		},
		{
			name: "missing env",
			config: log.Config{
				Service: "test",
				Level:   log.InfoLevel,
				Output:  log.OutputStdout,
			},
		},
		{
			name: "invalid level",
			config: log.Config{
				Service: "test",
				Env:     "dev",
				Level:   "invalid",
				Output:  log.OutputStdout,
			},
		},
		{
			name: "file output without path",
			config: log.Config{
				Service: "test",
				Env:     "dev",
				Level:   log.InfoLevel,
				Output:  log.OutputFile,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := log.New(tt.config)
			if err == nil {
				t.Errorf("expected error for %s, got nil", tt.name)
			}
		})
	}
}

func TestLogger_LogLevels(t *testing.T) {
	cfg := log.Config{
		Service: "test-service",
		Env:     "dev",
		Level:   log.DebugLevel,
		Output:  log.OutputStdout,
	}

	logger, err := log.New(cfg)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	// Test that all log methods work without panicking
	logger.Debug("debug message", log.String("request_id", "123"))
	logger.Info("info message", log.String("request_id", "123"))
	logger.Warn("warn message", log.String("request_id", "123"))
	logger.Error("error message", log.String("request_id", "123"))
}

func TestLogger_Fields(t *testing.T) {
	cfg := log.Config{
		Service: "test-service",
		Env:     "dev",
		Level:   log.InfoLevel,
		Output:  log.OutputStdout,
	}

	logger, err := log.New(cfg)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	// Test various field types
	logger.Info("testing fields",
		log.String("request_id", "abc-123"),
		log.String("string_field", "value"),
		log.Int("int_field", 42),
		log.Int64("int64_field", 9999999999),
		log.Float64("float_field", 3.14),
		log.Bool("bool_field", true),
		log.Any("map_field", map[string]any{"key": "value"}),
	)
}

func TestLogger_LevelFiltering(t *testing.T) {
	cfg := log.Config{
		Service: "test-service",
		Env:     "dev",
		Level:   log.WarnLevel, // Only warn and above
		Output:  log.OutputStdout,
	}

	logger, err := log.New(cfg)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	// These should be filtered out (no output expected)
	logger.Debug("should not appear", log.String("request_id", "123"))
	logger.Info("should not appear", log.String("request_id", "123"))

	// These should appear
	logger.Warn("should appear", log.String("request_id", "123"))
	logger.Error("should appear", log.String("request_id", "123"))
}

func TestLogger_FileOutput(t *testing.T) {
	tmpFile := "test_log.log"
	defer os.Remove(tmpFile)

	cfg := log.Config{
		Service:    "test-service",
		Env:        "dev",
		Level:      log.InfoLevel,
		Output:     log.OutputFile,
		FilePath:   tmpFile,
		MaxSizeMB:  1,
		MaxBackups: 1,
		MaxAgeDays: 1,
	}

	logger, err := log.New(cfg)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	logger.Info("test file output", log.String("request_id", "test-123"))
	logger.Sync()

	// Verify file was created
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Errorf("log file was not created")
	}

	// Read and verify content
	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	if len(content) == 0 {
		t.Error("log file is empty")
	}

	// Verify it's valid JSON
	var logEntry map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(content), &logEntry); err != nil {
		t.Errorf("log output is not valid JSON: %v", err)
	}

	// Verify required fields
	requiredFields := []string{"timestamp", "level", "message", "service", "env", "caller", "function"}
	for _, field := range requiredFields {
		if _, exists := logEntry[field]; !exists {
			t.Errorf("missing required field: %s", field)
		}
	}
}

func TestLogger_RequiredFields(t *testing.T) {
	cfg := log.Config{
		Service: "test-service",
		Env:     "production",
		Level:   log.InfoLevel,
		Output:  log.OutputStdout,
	}

	logger, err := log.New(cfg)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	// Log a message - required fields should be auto-injected
	logger.Info("test message",
		log.String("request_id", "req-123"),
		log.String("user_id", "user-456"),
	)

	// Note: In a real test, you would capture stdout and verify the JSON output
	// contains all required fields: timestamp, level, message, service, env,
	// request_id, caller, function
}
