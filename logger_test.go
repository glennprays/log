package log_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

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
	tmpFile := filepath.Join(t.TempDir(), "test_log.log")

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

	requiredFields := []string{"timestamp", "level", "message", "service", "env", "trace_id", "metadata"}
	// Note: caller and function are now optional fields controlled by EnableCaller config
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

func TestLogger_EmptyTraceId(t *testing.T) {
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
					t.Error("expected panic for empty traceId, got none")
				}
				msg, ok := r.(string)
				if !ok || !strings.Contains(msg, "traceId cannot be empty") {
					t.Errorf("expected panic message to contain 'traceId cannot be empty', got: %v", r)
				}
			}()
			tc.fn()
		})
	}
}

func TestLogger_NilMetadata(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test_nil_metadata.log")

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
	tmpFile := filepath.Join(t.TempDir(), "test_metadata_types.log")

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
	tmpFile := filepath.Join(t.TempDir(), "test_with_prebound.log")

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
	tmpFile := filepath.Join(t.TempDir(), "test_with_parent_unchanged.log")

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
	tmpFile := filepath.Join(t.TempDir(), "test_with_nested.log")

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
	tmpFile := filepath.Join(t.TempDir(), "test_with_caller.log")

	cfg := log.Config{
		Service:      "test-service",
		Env:          "dev",
		Level:        log.InfoLevel,
		Output:       log.OutputFile,
		FilePath:     tmpFile,
		EnableCaller: true, // Enable caller for this test
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
	tmpFile := filepath.Join(t.TempDir(), "test_with_levels.log")

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
	tmpFile := filepath.Join(t.TempDir(), "test_with_field_types.log")

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

func TestLogger_With_TraceIdValidation(t *testing.T) {
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

	// Should still panic on empty traceId
	defer func() {
		r := recover()
		if r == nil {
			t.Error("expected panic for empty traceId, got none")
		}
		msg, ok := r.(string)
		if !ok || !strings.Contains(msg, "traceId cannot be empty") {
			t.Errorf("expected panic message to contain 'traceId cannot be empty', got: %v", r)
		}
	}()

	childLogger.Info("", "this should panic", nil)
}

func TestLogger_With_Metadata(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test_with_metadata.log")

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

func TestLogger_CallerDisabledByDefault(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test_caller_disabled.log")

	cfg := log.Config{
		Service:  "test-service",
		Env:      "production",
		Level:    log.InfoLevel,
		Output:   log.OutputFile,
		FilePath: tmpFile,
		// EnableCaller not set - defaults to false
	}

	logger, err := log.New(cfg)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	logger.Info("req-123", "test message", nil, log.String("user_id", "user-456"))
	logger.Sync()

	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	var logEntry map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(content), &logEntry); err != nil {
		t.Errorf("log output is not valid JSON: %v", err)
	}

	// Caller and function should NOT be present
	if _, exists := logEntry["caller"]; exists {
		t.Error("caller field should not be present when EnableCaller is false")
	}
	if _, exists := logEntry["function"]; exists {
		t.Error("function field should not be present when EnableCaller is false")
	}

	// Other required fields should still be present
	requiredFields := []string{"timestamp", "level", "message", "service", "env", "trace_id", "metadata"}
	for _, field := range requiredFields {
		if _, exists := logEntry[field]; !exists {
			t.Errorf("missing required field: %s", field)
		}
	}
}

func TestLogger_CallerEnabled(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test_caller_enabled.log")

	cfg := log.Config{
		Service:      "test-service",
		Env:          "dev",
		Level:        log.InfoLevel,
		Output:       log.OutputFile,
		FilePath:     tmpFile,
		EnableCaller: true, // Explicitly enable
	}

	logger, err := log.New(cfg)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	logger.Info("req-123", "test message", nil, log.String("user_id", "user-456"))
	logger.Sync()

	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	var logEntry map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(content), &logEntry); err != nil {
		t.Errorf("log output is not valid JSON: %v", err)
	}

	// Caller and function SHOULD be present
	caller, callerExists := logEntry["caller"].(string)
	if !callerExists {
		t.Fatal("caller field should be present when EnableCaller is true")
	}
	if !strings.Contains(caller, "logger_test.go") {
		t.Errorf("caller should contain logger_test.go, got %s", caller)
	}

	function, functionExists := logEntry["function"].(string)
	if !functionExists {
		t.Fatal("function field should be present when EnableCaller is true")
	}
	if !strings.Contains(function, "TestLogger_CallerEnabled") {
		t.Errorf("function should contain TestLogger_CallerEnabled, got %s", function)
	}
}

func TestLogger_With_PreservesCallerSetting(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test_with_caller_setting.log")

	// Test with EnableCaller: true
	cfg := log.Config{
		Service:      "test-service",
		Env:          "dev",
		Level:        log.InfoLevel,
		Output:       log.OutputFile,
		FilePath:     tmpFile,
		EnableCaller: true,
	}

	logger, err := log.New(cfg)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	childLogger := logger.With(log.String("user_id", "user-456"))
	childLogger.Info("req-123", "child message", nil)
	logger.Sync()

	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	var logEntry map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(content), &logEntry); err != nil {
		t.Errorf("log output is not valid JSON: %v", err)
	}

	// Child should have caller/function since parent had EnableCaller: true
	if _, exists := logEntry["caller"]; !exists {
		t.Error("child logger should preserve parent's EnableCaller setting")
	}
	if _, exists := logEntry["function"]; !exists {
		t.Error("child logger should preserve parent's EnableCaller setting")
	}
}

// TestNew_DefaultOutput verifies the documented default: a Config without an
// explicit Output must construct successfully and behave as stdout output.
func TestNew_DefaultOutput(t *testing.T) {
	cfg := log.Config{
		Service: "test-service",
		Env:     "dev",
		Level:   log.InfoLevel,
		// Output deliberately omitted.
	}

	logger, err := log.New(cfg)
	if err != nil {
		t.Fatalf("expected default output to construct successfully, got %v", err)
	}
	if logger == nil {
		t.Fatal("expected logger to be non-nil")
	}
	logger.Info("req-123", "default output", nil) // must not panic
}

// TestLogger_ReservedField verifies that user fields using library-owned keys
// panic with the exact contract message, from both With and entry methods.
func TestLogger_ReservedField(t *testing.T) {
	cfg := log.Config{
		Service: "test-service",
		Env:     "dev",
		Level:   log.InfoLevel,
	}

	logger, err := log.New(cfg)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	t.Run("With", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("expected panic for reserved field key, got none")
			}
			msg, ok := r.(string)
			if !ok || msg != `log: field key "service" is reserved` {
				t.Errorf("unexpected panic: %v", r)
			}
		}()
		logger.With(log.String("service", "forged"))
	})

	t.Run("Info", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("expected panic for reserved field key, got none")
			}
			msg, ok := r.(string)
			if !ok || msg != `log: field key "message" is reserved` {
				t.Errorf("unexpected panic: %v", r)
			}
		}()
		logger.Info("req-123", "forged message", nil, log.String("message", "forged"))
	})
}

// TestNew_FileOutputPreflight verifies that an unusable file path fails at
// construction, before any log call can drop a record.
func TestNew_FileOutputPreflight(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("failed to create blocker file: %v", err)
	}

	cfg := log.Config{
		Service:  "test-service",
		Env:      "dev",
		Level:    log.InfoLevel,
		Output:   log.OutputFile,
		FilePath: filepath.Join(blocker, "sub", "app.log"), // parent exists but is a file
	}

	logger, err := log.New(cfg)
	if err == nil {
		logger.Sync()
		t.Fatal("expected construction error for unusable file path, got nil")
	}
	if !strings.Contains(err.Error(), "prepare log file") {
		t.Errorf("expected preflight error, got: %v", err)
	}
}

// TestLogger_ConcurrentFileEntries verifies that concurrent logging to a file
// sink produces exactly one intact JSON object per entry, with required keys.
func TestLogger_ConcurrentFileEntries(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "concurrent.log")

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

	const goroutines = 8
	const perGoroutine = 25
	var wg sync.WaitGroup
	for g := range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range perGoroutine {
				logger.Info(fmt.Sprintf("trace-%d-%d", g, i), "concurrent entry", nil)
			}
		}()
	}
	wg.Wait()
	if err := logger.Sync(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	lines := bytes.Split(bytes.TrimSpace(content), []byte("\n"))
	if len(lines) != goroutines*perGoroutine {
		t.Fatalf("expected %d log entries, got %d", goroutines*perGoroutine, len(lines))
	}
	for i, line := range lines {
		var logEntry map[string]any
		if err := json.Unmarshal(line, &logEntry); err != nil {
			t.Fatalf("line %d is not valid JSON (interleaved write): %v", i, err)
		}
		for _, key := range []string{"service", "env", "trace_id", "metadata"} {
			if _, exists := logEntry[key]; !exists {
				t.Errorf("line %d missing required key %q", i, key)
			}
		}
	}
}

// TestLogger_FilteredDebugDoesNotWrite verifies that a debug entry below the
// configured level produces no output, even with caller capture enabled.
func TestLogger_FilteredDebugDoesNotWrite(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "filtered.log")

	cfg := log.Config{
		Service:      "test-service",
		Env:          "dev",
		Level:        log.WarnLevel,
		Output:       log.OutputFile,
		FilePath:     tmpFile,
		EnableCaller: true,
	}

	logger, err := log.New(cfg)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	logger.Debug("req-123", "should not appear", nil)
	if err := logger.Sync(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	if trimmed := bytes.TrimSpace(content); len(trimmed) != 0 {
		t.Errorf("expected no output for filtered debug entry, got: %s", trimmed)
	}
}

// TestLogger_CloseFileSink verifies shared parent/child lifecycle: entries
// through both loggers persist, Close on the child closes the shared sink,
// Close is idempotent, post-close Sync returns nil, and post-close entries
// are discarded without corrupting the file.
func TestLogger_CloseFileSink(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "close.log")

	cfg := log.Config{
		Service:  "test-service",
		Env:      "dev",
		Level:    log.InfoLevel,
		Output:   log.OutputFile,
		FilePath: tmpFile,
	}

	parent, err := log.New(cfg)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	child := parent.With(log.String("component", "child"))

	parent.Info("trace-1", "before close parent", nil)
	child.Info("trace-2", "before close child", nil)
	if err := child.Sync(); err != nil {
		t.Fatalf("sync before close failed: %v", err)
	}

	if err := child.Close(); err != nil {
		t.Fatalf("child Close failed: %v", err)
	}
	if err := parent.Close(); err != nil {
		t.Fatalf("repeated parent Close must be idempotent, got: %v", err)
	}
	if err := parent.Sync(); err != nil {
		t.Fatalf("post-close Sync must return nil, got: %v", err)
	}
	parent.Info("trace-3", "after close parent", nil)
	child.Info("trace-4", "after close child", nil)

	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	lines := bytes.Split(bytes.TrimSpace(content), []byte("\n"))
	if len(lines) != 2 {
		t.Fatalf("expected exactly 2 entries after close, got %d: %s", len(lines), content)
	}
	var first map[string]any
	if err := json.Unmarshal(lines[0], &first); err != nil {
		t.Fatalf("first line is not valid JSON: %v", err)
	}
	if first["message"] != "before close parent" {
		t.Errorf("first entry message = %v, want before close parent", first["message"])
	}
	var second map[string]any
	if err := json.Unmarshal(lines[1], &second); err != nil {
		t.Fatalf("second line is not valid JSON: %v", err)
	}
	if second["message"] != "before close child" {
		t.Errorf("second entry message = %v, want before close child", second["message"])
	}
}

// TestLogger_SyncStdoutUnderPipe verifies Sync returns nil for stdout output
// even when stdout is a pipe: v0.2 fsynced os.Stdout, which fails with EBADF
// on pipes and ttys; stdout writes are unbuffered, so there is nothing to
// flush. The test binary's stdout is redirected to a pipe by go test.
func TestLogger_SyncStdoutUnderPipe(t *testing.T) {
	logger, err := log.New(log.Config{
		Service: "test-service",
		Env:     "dev",
		Level:   log.InfoLevel,
		Output:  log.OutputStdout,
	})
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	logger.Info("req-123", "sync stdout", nil)
	if err := logger.Sync(); err != nil {
		t.Fatalf("stdout Sync must return nil (unbuffered writes), got: %v", err)
	}
}

// TestLogger_CloseStdout verifies Close is a no-op returning nil for stdout.
func TestLogger_CloseStdout(t *testing.T) {
	logger, err := log.New(log.Config{
		Service: "test-service",
		Env:     "dev",
		Level:   log.InfoLevel,
		Output:  log.OutputStdout,
	})
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	if err := logger.Close(); err != nil {
		t.Fatalf("stdout Close must return nil, got: %v", err)
	}
	// Stdout logging must keep working after Close.
	logger.Info("req-123", "still logging", nil)
}

// TestLogger_CloseConcurrent verifies that Close contending with parent/child
// Info+Sync goroutines never interleaves a write: every persisted nonempty
// line must decode as one JSON object, and a final Close returns nil.
func TestLogger_CloseConcurrent(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "close-concurrent.log")

	cfg := log.Config{
		Service:  "test-service",
		Env:      "dev",
		Level:    log.InfoLevel,
		Output:   log.OutputFile,
		FilePath: tmpFile,
	}

	parent, err := log.New(cfg)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	child := parent.With(log.String("component", "child"))

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			parent.Info("trace-parent", "concurrent parent", nil)
		}
		if err := parent.Sync(); err != nil {
			t.Errorf("parent sync: %v", err)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			child.Info("trace-child", "concurrent child", nil)
		}
		if err := child.Sync(); err != nil {
			t.Errorf("child sync: %v", err)
		}
	}()

	go func() {
		if err := parent.Close(); err != nil {
			t.Errorf("Close under contention: %v", err)
		}
	}()
	wg.Wait()

	// Wait for Close goroutine to finish before final assertions.
	time.Sleep(10 * time.Millisecond)
	if err := parent.Close(); err != nil {
		t.Fatalf("final Close must return nil, got: %v", err)
	}

	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	for i, line := range bytes.Split(bytes.TrimSpace(content), []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		var entry map[string]any
		if err := json.Unmarshal(line, &entry); err != nil {
			t.Fatalf("line %d is not valid JSON (interleaved write): %v", i, err)
		}
	}
}

// TestNop verifies Nop emits nothing, stays composable via With, returns nil
// from Sync and Close, and preserves the input-validation panic contracts.
func TestNop(t *testing.T) {
	nop := log.Nop()
	if nop == nil {
		t.Fatal("expected Nop to return a non-nil logger")
	}

	// Every nonfatal method must be callable and silent.
	nop.Debug("trace", "debug", nil)
	nop.Info("trace", "info", nil)
	nop.Warn("trace", "warn", nil)
	nop.Error("trace", "error", nil)

	child := nop.With(log.String("key", "value"))
	child.Info("trace", "child info", nil)

	if err := nop.Sync(); err != nil {
		t.Errorf("Nop Sync must return nil, got: %v", err)
	}
	if err := nop.Close(); err != nil {
		t.Errorf("Nop Close must return nil, got: %v", err)
	}

	// Validation policy preserved: empty trace ID panics.
	assertPanic(t, "log: traceId cannot be empty", func() {
		nop.Info("", "empty trace", nil)
	})

	// Validation policy preserved: reserved key panics.
	assertPanic(t, `log: field key "trace_id" is reserved`, func() {
		nop.Info("trace", "reserved key", nil, log.String("trace_id", "forged"))
	})
}

// assertPanic asserts fn panics with exactly wantMsg.
func assertPanic(t *testing.T, wantMsg string, fn func()) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected panic %q, got none", wantMsg)
		}
		msg, ok := r.(string)
		if !ok {
			t.Fatalf("expected string panic %q, got %T: %v", wantMsg, r, r)
		}
		if msg != wantMsg {
			t.Fatalf("panic = %q, want %q", msg, wantMsg)
		}
	}()
	fn()
}

// TestNopFatalHelperProcess is only run as a helper process by
// TestNopFatalExits; it must terminate the process via Nop().Fatal.
func TestNopFatalHelperProcess(t *testing.T) {
	if os.Getenv("LOG_NOP_FATAL_HELPER") != "1" {
		t.Skip("helper process for TestNopFatalExits")
	}
	log.Nop().Fatal("trace", "fatal", nil)
}

// TestNopFatalExits verifies Nop preserves the fatal exit-1 contract by
// re-executing the test binary as a helper process.
func TestNopFatalExits(t *testing.T) {
	if os.Getenv("LOG_NOP_FATAL_HELPER") == "1" {
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestNopFatalHelperProcess$")
	cmd.Env = append(os.Environ(), "LOG_NOP_FATAL_HELPER=1")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected non-zero exit from Nop().Fatal, got success; output:\n%s", out)
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected *exec.ExitError, got %T: %v; output:\n%s", err, err, out)
	}
	if code := exitErr.ExitCode(); code != 1 {
		t.Fatalf("exit code = %d, want 1; output:\n%s", code, out)
	}
}
