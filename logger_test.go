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

func TestLogger_With_PreBoundFields(t *testing.T) {
	tmpFile := "test_with_prebound.log"
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

	// Create child logger with pre-bound fields
	childLogger := logger.With(
		log.String("user_id", "user-456"),
		log.String("session_id", "sess-789"),
	)

	childLogger.Info("req-123", "child logger message", nil)
	logger.Sync()

	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	var logEntry map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(content), &logEntry); err != nil {
		t.Errorf("log output is not valid JSON: %v", err)
	}

	// Check pre-bound fields are present
	if logEntry["user_id"] != "user-456" {
		t.Errorf("expected user_id=user-456, got %v", logEntry["user_id"])
	}
	if logEntry["session_id"] != "sess-789" {
		t.Errorf("expected session_id=sess-789, got %v", logEntry["session_id"])
	}
}

func TestLogger_With_ParentUnchanged(t *testing.T) {
	tmpFile := "test_with_parent_unchanged.log"
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

	// Create child logger
	_ = logger.With(log.String("user_id", "user-456"))

	// Log from parent
	logger.Info("req-123", "parent logger message", nil)
	logger.Sync()

	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	var logEntry map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(content), &logEntry); err != nil {
		t.Errorf("log output is not valid JSON: %v", err)
	}

	// Check pre-bound field is NOT present in parent
	if _, exists := logEntry["user_id"]; exists {
		t.Error("parent logger should not have child's pre-bound fields")
	}
}

func TestLogger_With_NestedChildren(t *testing.T) {
	tmpFile := "test_with_nested.log"
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

	// Create nested child loggers
	serviceLogger := logger.With(log.String("layer", "api"))
	userLogger := serviceLogger.With(log.String("user_id", "user-456"))
	actionLogger := userLogger.With(log.String("action", "login"))

	actionLogger.Info("req-123", "nested child message", nil)
	logger.Sync()

	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	var logEntry map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(content), &logEntry); err != nil {
		t.Errorf("log output is not valid JSON: %v", err)
	}

	// Check all pre-bound fields from all levels are present
	if logEntry["layer"] != "api" {
		t.Errorf("expected layer=api, got %v", logEntry["layer"])
	}
	if logEntry["user_id"] != "user-456" {
		t.Errorf("expected user_id=user-456, got %v", logEntry["user_id"])
	}
	if logEntry["action"] != "login" {
		t.Errorf("expected action=login, got %v", logEntry["action"])
	}
}

func TestLogger_With_CallerCorrectness(t *testing.T) {
	tmpFile := "test_with_caller.log"
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

	childLogger := logger.With(log.String("user_id", "user-456"))
	childLogger.Info("req-123", "caller test", nil) // This line number matters

	logger.Sync()

	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	var logEntry map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(content), &logEntry); err != nil {
		t.Errorf("log output is not valid JSON: %v", err)
	}

	// Caller should point to the Info() call line, not With() call
	caller, ok := logEntry["caller"].(string)
	if !ok {
		t.Fatal("caller field is not a string")
	}
	if !strings.Contains(caller, "logger_test.go") {
		t.Errorf("caller should contain logger_test.go, got %s", caller)
	}
	// Function should be TestLogger_With_CallerCorrectness
	function, ok := logEntry["function"].(string)
	if !ok {
		t.Fatal("function field is not a string")
	}
	if !strings.Contains(function, "TestLogger_With_CallerCorrectness") {
		t.Errorf("function should contain TestLogger_With_CallerCorrectness, got %s", function)
	}
}

func TestLogger_With_AllLogLevels(t *testing.T) {
	tmpFile := "test_with_levels.log"
	defer os.Remove(tmpFile)

	cfg := log.Config{
		Service:  "test-service",
		Env:      "dev",
		Level:    log.DebugLevel,
		Output:   log.OutputFile,
		FilePath: tmpFile,
	}

	logger, err := log.New(cfg)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	childLogger := logger.With(log.String("context", "test"))

	childLogger.Debug("req-1", "debug message", nil)
	childLogger.Info("req-2", "info message", nil)
	childLogger.Warn("req-3", "warn message", nil)
	childLogger.Error("req-4", "error message", nil)
	logger.Sync()

	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	lines := bytes.Split(bytes.TrimSpace(content), []byte("\n"))
	if len(lines) != 4 {
		t.Fatalf("expected 4 log entries, got %d", len(lines))
	}

	// Check each entry has pre-bound field
	for i, line := range lines {
		var logEntry map[string]any
		if err := json.Unmarshal(line, &logEntry); err != nil {
			t.Errorf("line %d: log output is not valid JSON: %v", i, err)
		}
		if logEntry["context"] != "test" {
			t.Errorf("line %d: expected context=test, got %v", i, logEntry["context"])
		}
	}
}

func TestLogger_With_EmptyFieldsSlice(t *testing.T) {
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

	// Call With() with no fields
	sameLogger := logger.With()

	// Should return same logger instance
	if sameLogger != logger {
		t.Error("With() with no fields should return the same logger instance")
	}
}

func TestLogger_With_AllFieldTypes(t *testing.T) {
	tmpFile := "test_with_field_types.log"
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

	childLogger := logger.With(
		log.String("string_field", "value"),
		log.Int("int_field", 42),
		log.Int64("int64_field", 9999999999),
		log.Float64("float_field", 3.14),
		log.Bool("bool_field", true),
		log.Any("any_field", map[string]any{"key": "value"}),
	)

	childLogger.Info("req-123", "field types test", nil)
	logger.Sync()

	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	var logEntry map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(content), &logEntry); err != nil {
		t.Errorf("log output is not valid JSON: %v", err)
	}

	// Check all field types
	if logEntry["string_field"] != "value" {
		t.Errorf("expected string_field=value, got %v", logEntry["string_field"])
	}
	if logEntry["int_field"] != float64(42) { // JSON unmarshal converts to float64
		t.Errorf("expected int_field=42, got %v", logEntry["int_field"])
	}
	if logEntry["int64_field"] != float64(9999999999) {
		t.Errorf("expected int64_field=9999999999, got %v", logEntry["int64_field"])
	}
	if logEntry["float_field"] != 3.14 {
		t.Errorf("expected float_field=3.14, got %v", logEntry["float_field"])
	}
	if logEntry["bool_field"] != true {
		t.Errorf("expected bool_field=true, got %v", logEntry["bool_field"])
	}
	anyField, ok := logEntry["any_field"].(map[string]any)
	if !ok || anyField["key"] != "value" {
		t.Errorf("expected any_field={key: value}, got %v", logEntry["any_field"])
	}
}

func TestLogger_With_RequestIdValidation(t *testing.T) {
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

	childLogger := logger.With(log.String("user_id", "user-456"))

	// Should still panic on empty requestId
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

	childLogger.Info("", "this should panic", nil)
}

func TestLogger_With_Metadata(t *testing.T) {
	tmpFile := "test_with_metadata.log"
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

	childLogger := logger.With(log.String("user_id", "user-456"))

	// Log with both nil and non-nil metadata
	childLogger.Info("req-1", "nil metadata", nil)
	childLogger.Info("req-2", "map metadata", map[string]any{"ip": "192.168.1.1"})
	logger.Sync()

	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	lines := bytes.Split(bytes.TrimSpace(content), []byte("\n"))
	if len(lines) != 2 {
		t.Fatalf("expected 2 log entries, got %d", len(lines))
	}

	// Check first entry (nil metadata)
	var entry1 map[string]any
	if err := json.Unmarshal(lines[0], &entry1); err != nil {
		t.Errorf("entry 1: log output is not valid JSON: %v", err)
	}
	if entry1["user_id"] != "user-456" {
		t.Errorf("entry 1: expected user_id=user-456, got %v", entry1["user_id"])
	}
	if entry1["metadata"] != nil {
		t.Errorf("entry 1: expected metadata=null, got %v", entry1["metadata"])
	}

	// Check second entry (map metadata)
	var entry2 map[string]any
	if err := json.Unmarshal(lines[1], &entry2); err != nil {
		t.Errorf("entry 2: log output is not valid JSON: %v", err)
	}
	if entry2["user_id"] != "user-456" {
		t.Errorf("entry 2: expected user_id=user-456, got %v", entry2["user_id"])
	}
	metadata, ok := entry2["metadata"].(map[string]any)
	if !ok || metadata["ip"] != "192.168.1.1" {
		t.Errorf("entry 2: expected metadata={ip: 192.168.1.1}, got %v", entry2["metadata"])
	}
}
