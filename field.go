package log

import "go.uber.org/zap"

// Field represents a structured log field (key-value pair).
// It is an opaque type that wraps the underlying logging implementation.
// Use the provided helper functions (String, Int, etc.) to create fields.
type Field struct {
	zapField zap.Field
}

// String creates a field with a string value.
func String(key, value string) Field {
	return Field{zapField: zap.String(key, value)}
}

// Int creates a field with an integer value.
func Int(key string, value int) Field {
	return Field{zapField: zap.Int(key, value)}
}

// Int64 creates a field with an int64 value.
func Int64(key string, value int64) Field {
	return Field{zapField: zap.Int64(key, value)}
}

// Float64 creates a field with a float64 value.
func Float64(key string, value float64) Field {
	return Field{zapField: zap.Float64(key, value)}
}

// Bool creates a field with a boolean value.
func Bool(key string, value bool) Field {
	return Field{zapField: zap.Bool(key, value)}
}

// Any creates a field with any type of value.
// The value will be JSON-marshaled in the log output.
// Use this for complex types like maps, structs, and slices.
func Any(key string, value any) Field {
	return Field{zapField: zap.Any(key, value)}
}

// Error creates an error field with the key "error".
// The error message and type will be included in the log output.
func Error(err error) Field {
	return Field{zapField: zap.Error(err)}
}

func toZapFields(fields []Field) []zap.Field {
	zapFields := make([]zap.Field, len(fields))
	for i, f := range fields {
		zapFields[i] = f.zapField
	}
	return zapFields
}
