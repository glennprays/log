package log

import (
	"path/filepath"
	"runtime"
	"strings"
)

// callerInfo holds information about the caller of a log function.
type callerInfo struct {
	file     string
	line     int
	function string
}

// getCaller extracts caller information from the call stack.
// skip specifies the number of stack frames to skip (relative to getCaller itself).
func getCaller(skip int) callerInfo {
	// Skip getCaller itself + additional frames requested by caller
	pc, file, line, ok := runtime.Caller(skip + 1)
	if !ok {
		return callerInfo{
			file:     "unknown",
			line:     0,
			function: "unknown",
		}
	}

	// Get the function name
	funcName := "unknown"
	if fn := runtime.FuncForPC(pc); fn != nil {
		funcName = fn.Name()
		// Simplify function name by removing package path
		if idx := strings.LastIndex(funcName, "/"); idx != -1 {
			funcName = funcName[idx+1:]
		}
	}

	// Simplify file path to just the filename
	file = filepath.Base(file)

	return callerInfo{
		file:     file,
		line:     line,
		function: funcName,
	}
}
