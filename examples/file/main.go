// Command file demonstrates the file output mode: one structured JSON entry
// written to a temporary log file, printed to stdout, and no artifact left
// behind on exit.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/glennprays/log"
)

func main() {
	dir, err := os.MkdirTemp("", "log-file-example-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	logPath := filepath.Join(dir, "app.log")
	logger, err := log.New(log.Config{
		Service:  "example-service",
		Env:      "dev",
		Level:    log.InfoLevel,
		Output:   log.OutputFile,
		FilePath: logPath,
	})
	if err != nil {
		panic(err)
	}

	logger.Info("example-trace", "file example", nil, log.String("component", "examples"))

	if err := logger.Close(); err != nil {
		panic(err)
	}

	content, err := os.ReadFile(logPath)
	if err != nil {
		panic(err)
	}
	fmt.Print(string(content))
}
