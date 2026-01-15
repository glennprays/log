package log_test

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
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

	logger.Debug("req-123", "debug message", nil)
	logger.Info("req-123", "info message", nil)
	logger.Warn("req-123", "warn message", nil)
	logger.Error("req-123", "error message", nil)
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

	logger.Info(
		"abc-123",
		"testing fields",
		map[string]any{
			"ip":     "192.168.1.1",
			"method": "POST",
		},
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
		Level:   log.WarnLevel,
		Output:  log.OutputStdout,
	}

	logger, err := log.New(cfg)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	logger.Debug("req-123", "should not appear", nil)
	logger.Info("req-123", "should not appear", nil)
	logger.Warn("req-123", "should appear", nil)
	logger.Error("req-123", "should appear", nil)
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

	logger.Info("test-123", "test file output", nil)
	logger.Sync()

	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Errorf("log file was not created")
	}

	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	if len(content) == 0 {
		t.Error("log file is empty")
	}

	var logEntry map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(content), &logEntry); err != nil {
		t.Errorf("log output is not valid JSON: %v", err)
	}

	requiredFields := []string{"timestamp", "level", "message", "service", "env", "request_id", "metadata", "caller", "function"}
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

	logger.Info(
		"req-123",
		"test message",
		map[string]any{"key": "value"},
		log.String("user_id", "user-456"),
	)
}

func TestLogger_EmptyRequestId(t *testing.T) {
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

	testCases := []struct {
		name string
		fn   func()
	}{
		{"Debug", func() { logger.Debug("", "message", nil) }},
		{"Info", func() { logger.Info("", "message", nil) }},
		{"Warn", func() { logger.Warn("", "message", nil) }},
		{"Error", func() { logger.Error("", "message", nil) }},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if r == nil {
					t.Error("expected panic for empty requestId, got none")
				}
				msg, ok := r.(string)
				if !ok || !strings.Contains(msg, "requestId cannot be empty") {
					t.Errorf("expected panic message to contain 'requestId cannot be empty', got: %v", r)
				}
			}()
			tc.fn()
		})
	}
}

func TestLogger_NilMetadata(t *testing.T) {
	tmpFile := "test_nil_metadata.log"
	defer os.Remove(tmpFile)

	cfg := log.Config{
		Service:  "test-service",
		Env:      "dev",
		Level:    log.InfoLevel,
		Output:   log.OutputFile,
		FilePath: tmpFile,
	}

	logger, err := log.New(cfg)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	logger.Info("req-123", "test nil metadata", nil, log.String("user_id", "user-456"))
	logger.Sync()

	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	var logEntry map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(content), &logEntry); err != nil {
		t.Errorf("log output is not valid JSON: %v", err)
	}

	if _, exists := logEntry["metadata"]; !exists {
		t.Error("metadata field should exist even when nil")
	}

	if logEntry["metadata"] != nil {
		t.Errorf("expected metadata to be null, got: %v", logEntry["metadata"])
	}
}

func TestLogger_MetadataTypes(t *testing.T) {
	tmpFile := "test_metadata_types.log"
	defer os.Remove(tmpFile)

	cfg := log.Config{
		Service:  "test-service",
		Env:      "dev",
		Level:    log.InfoLevel,
		Output:   log.OutputFile,
		FilePath: tmpFile,
	}

	logger, err := log.New(cfg)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	type customStruct struct {
		Name  string
		Value int
	}

	testCases := []struct {
		name     string
		metadata any
	}{
		{"map", map[string]any{"ip": "127.0.0.1", "method": "GET"}},
		{"struct", customStruct{Name: "test", Value: 42}},
		{"string", "simple string metadata"},
		{"int", 12345},
		{"slice", []string{"item1", "item2", "item3"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logger.Info("req-"+tc.name, "testing metadata type: "+tc.name, tc.metadata)
		})
	}

	logger.Sync()

	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	lines := bytes.Split(bytes.TrimSpace(content), []byte("\n"))
	if len(lines) != len(testCases) {
		t.Errorf("expected %d log entries, got %d", len(testCases), len(lines))
	}

	for _, line := range lines {
		var logEntry map[string]any
		if err := json.Unmarshal(line, &logEntry); err != nil {
			t.Errorf("log output is not valid JSON: %v", err)
		}
		if _, exists := logEntry["metadata"]; !exists {
			t.Error("metadata field should exist")
		}
	}
}
