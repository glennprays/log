package log

import (
	"errors"
	"fmt"
	"strings"
)

// Config holds logger configuration.
// All fields except file rotation settings (MaxSizeMB, MaxBackups, MaxAgeDays) are required.
// File rotation settings have defaults and are only used when Output is OutputFile.
type Config struct {
	// Service is the name of the service (required).
	Service string

	// Env is the environment: dev, development, staging, prod, or production (required).
	Env string

	// Level is the minimum log level (required).
	// Use log.DebugLevel, log.InfoLevel, log.WarnLevel, log.ErrorLevel, or log.FatalLevel.
	Level Level

	// Output specifies where to write logs: OutputStdout or OutputFile (required).
	Output OutputType

	// FilePath is the path to the log file (required if Output is OutputFile).
	FilePath string

	// MaxSizeMB is the maximum size in megabytes before log rotation (default: 100).
	// Only used when Output is OutputFile.
	MaxSizeMB int

	// MaxBackups is the maximum number of old log files to retain (default: 3).
	// Only used when Output is OutputFile.
	MaxBackups int

	// MaxAgeDays is the maximum number of days to retain old log files (default: 28).
	// Only used when Output is OutputFile.
	MaxAgeDays int
}

// Validate checks if the Config is valid. Returns an error containing all validation failures.
// It also sets default values for file rotation settings if they are not provided.
func (c *Config) Validate() error {
	var errs []error

	if strings.TrimSpace(c.Service) == "" {
		errs = append(errs, errors.New("service name is required"))
	}

	if strings.TrimSpace(c.Env) == "" {
		errs = append(errs, errors.New("environment is required"))
	} else {
		env := strings.ToLower(strings.TrimSpace(c.Env))
		if env != "dev" && env != "development" && env != "staging" && env != "prod" && env != "production" {
			errs = append(errs, fmt.Errorf("environment must be one of: dev, development, staging, prod, production (got: %s)", c.Env))
		}
	}

	if c.Level == "" {
		errs = append(errs, errors.New("log level is required"))
	} else {
		if _, err := c.Level.toZapLevel(); err != nil {
			errs = append(errs, err)
		}
	}

	if c.Output == "" {
		errs = append(errs, errors.New("output type is required"))
	} else if c.Output != OutputStdout && c.Output != OutputFile {
		errs = append(errs, fmt.Errorf("output must be stdout or file (got: %s)", c.Output))
	}

	if c.Output == OutputFile && strings.TrimSpace(c.FilePath) == "" {
		errs = append(errs, errors.New("file path is required when output is file"))
	}

	if c.MaxSizeMB <= 0 {
		c.MaxSizeMB = 100
	}
	if c.MaxBackups <= 0 {
		c.MaxBackups = 3
	}
	if c.MaxAgeDays <= 0 {
		c.MaxAgeDays = 28
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}
