package errors

import (
	"fmt"
	"runtime"
)

// Trace returns an error wrapped in a struct to track where the error is generated.
func Trace(err error) error {
	return trace(err, 1)
}

// Traces returns the stack trace for an error.
func Traces(e error) []string {
	if e, ok := e.(aerr); ok {
		return e.trace
	}
	return nil
}

func trace(err error, n int) error {
	// Just incase we get a nil make sure it doesn't turn into an error.
	if err == nil {
		return nil
	}

	trace := "unknown"
	_, file, line, ok := runtime.Caller(n + 1)
	if ok {
		short := file
		depth := 0
		for i := len(file) - 1; i > 0; i-- {
			if file[i] == '/' {
				short = file[i+1:]
				depth++
				if depth == 4 {
					break
				}
			}
		}
		trace = fmt.Sprintf("%s:%d", short, line)
	}

	e := wrap(err)
	e.trace = append(e.trace, trace)
	return e
}
