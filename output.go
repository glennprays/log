package log

// OutputType defines where logs should be written.
type OutputType string

const (
	// OutputStdout writes logs to standard output (default).
	OutputStdout OutputType = "stdout"
	// OutputFile writes logs to a file with rotation support.
	OutputFile OutputType = "file"
)

// String returns the string representation of the OutputType.
func (o OutputType) String() string {
	return string(o)
}
