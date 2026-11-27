// Command log-v0-2-consumer is a frozen external-consumer fixture written
// against only the v0.2 API of github.com/glennprays/log. CI compiles and
// runs it unchanged against the local module (via replace) and, on version
// tags, against the published module. It must never import a v0.3 symbol
// (Close, Nop).
package main

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/glennprays/log"
)

func main() {
	dir, err := os.MkdirTemp("", "log-v0-2-consumer-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	fileConfig := log.Config{
		Service: "legacy-consumer", Env: "dev", Level: log.InfoLevel,
		Output: log.OutputFile, FilePath: filepath.Join(dir, "legacy.log"),
		MaxSizeMB: 1, MaxBackups: 1, MaxAgeDays: 1, EnableCaller: false,
	}
	if err := fileConfig.Validate(); err != nil {
		panic(err)
	}
	for _, level := range []log.Level{log.DebugLevel, log.InfoLevel, log.WarnLevel, log.ErrorLevel, log.FatalLevel} {
		if level.String() == "" {
			panic("v0.2 level string contract changed")
		}
	}
	if log.OutputStdout.String() != "stdout" || log.OutputFile.String() != "file" {
		panic("v0.2 output string contract changed")
	}

	fileLogger, err := log.New(fileConfig)
	if err != nil {
		panic(err)
	}
	fileLogger.Info("legacy-trace", "file", nil)
	if err := fileLogger.Sync(); err != nil {
		panic(err)
	}

	logger, err := log.New(log.Config{
		Service: "legacy-consumer", Env: "dev", Level: log.DebugLevel,
		Output: log.OutputStdout, EnableCaller: false,
	})
	if err != nil {
		panic(err)
	}
	child := logger.With(
		log.String("string", "value"), log.Int("int", 1), log.Int64("int64", 2),
		log.Float64("float", 3.5), log.Bool("bool", true),
		log.Any("any", map[string]string{"key": "value"}), log.Error(errors.New("legacy error")),
	)
	child.Debug("legacy-trace", "debug", nil)
	child.Info("legacy-trace", "info", nil)
	child.Warn("legacy-trace", "warn", nil)
	child.Error("legacy-trace", "error", nil)
	if false {
		child.Fatal("legacy-trace", "compile-only", nil)
	}
	if err := logger.Sync(); err != nil {
		panic(err)
	}
}
