package log

// OutputType specifies the destination for log output.
type OutputType string

const (
	// OutputStdout writes logs to standard output.
	// This is the default and recommended for containerized applications.
	// Logs are written as JSON, one entry per line.
	OutputStdout OutputType = "stdout"

	// OutputFile writes logs to a file with automatic rotation.
	// Rotation is handled by lumberjack based on MaxSizeMB, MaxBackups, and MaxAgeDays settings.
	OutputFile OutputType = "file"
)

// String returns the string representation of the OutputType.
func (o OutputType) String() string {
	return string(o)
}
