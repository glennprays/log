// Command stdout demonstrates the default stdout output mode: one structured
// JSON entry printed to standard output with zero files created.
package main

import (
	"github.com/glennprays/log"
)

func main() {
	// Output omitted defaults to stdout; no file is written.
	logger, err := log.New(log.Config{
		Service: "example-service",
		Env:     "dev",
		Level:   log.InfoLevel,
	})
	if err != nil {
		panic(err)
	}

	logger.Info("example-trace", "stdout example", nil, log.String("component", "examples"))

	if err := logger.Close(); err != nil {
		panic(err)
	}
}
