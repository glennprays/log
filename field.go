package log

import "go.uber.org/zap"

// Field is an opaque type that represents a log field.
// It wraps zap.Field but does not expose it publicly.
type Field struct {
	zapField zap.Field
}

// String creates a string field.
func String(key, value string) Field {
	return Field{zapField: zap.String(key, value)}
}

// Int creates an integer field.
func Int(key string, value int) Field {
	return Field{zapField: zap.Int(key, value)}
}

// Int64 creates an int64 field.
func Int64(key string, value int64) Field {
	return Field{zapField: zap.Int64(key, value)}
}

// Float64 creates a float64 field.
func Float64(key string, value float64) Field {
	return Field{zapField: zap.Float64(key, value)}
}

// Bool creates a boolean field.
func Bool(key string, value bool) Field {
	return Field{zapField: zap.Bool(key, value)}
}

// Any creates a field with any type. The value will be JSON-marshaled.
func Any(key string, value any) Field {
	return Field{zapField: zap.Any(key, value)}
}

// Error creates an error field with key "error".
func Error(err error) Field {
	return Field{zapField: zap.Error(err)}
}

// toZapFields converts a slice of Field to a slice of zap.Field.
func toZapFields(fields []Field) []zap.Field {
	zapFields := make([]zap.Field, len(fields))
	for i, f := range fields {
		zapFields[i] = f.zapField
	}
	return zapFields
}
