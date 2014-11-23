package errors

import (
	"fmt"
	"runtime"
	"strings"
)

type Traced struct {
	Err   error // actual error
	Trace []string
}

func (t Traced) Error() string {
	return fmt.Sprintf("%s [%s]", t.Err.Error(), strings.Join(t.Trace, ", "))
}

// Trace returns an error wrapped in a struct to track where the error is generated.
// This error should not be returned from a package (as it masks the actual error),
// but it can be used to give better feedback about the source of a generic error
// inside of a package.
func Trace(err error) error {
	// Just incase we get a nil make sure it doesn't turn into an error.
	if err == nil {
		return nil
	}

	trace := "unknown"
	_, file, line, ok := runtime.Caller(1)
	if ok {
		short := file
		depth := 0
		for i := len(file) - 1; i > 0; i-- {
			if file[i] == '/' {
				short = file[i+1:]
				depth++
				if depth == 2 {
					break
				}
			}
		}
		trace = fmt.Sprintf("%s:%d", short, line)
	}

	if t, ok := err.(Traced); ok {
		t.Trace = append(t.Trace, trace)
		return t
	}

	return Traced{Err: err, Trace: []string{trace}}
}
