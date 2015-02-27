// Test utils inspired from https://github.com/benbjohnson/testing
package test

import (
	"fmt"
	"reflect"
	"runtime"
	"testing"

	"github.com/sprucehealth/backend/libs/golog"
)

// Assert fails the test if the condition is false.
func Assert(t testing.TB, condition bool, msg string, v ...interface{}) {
	if !condition {
		golog.LogDepthf(1, golog.ERR, msg+"\n\n", v...)
		t.FailNow()
	}
}

// OK fails the test if an err is not nil.
func OK(t testing.TB, err error) {
	if err != nil {
		golog.LogDepthf(1, golog.ERR, "unexpected error: %s\n\n", err.Error())
		t.FailNow()
	}
}

// Equals fails the test if exp is not equal to act.
func Equals(t testing.TB, exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		golog.LogDepthf(1, golog.ERR, "\n\n\texp: %#v\n\n\tgot: %#v\n\n", exp, act)
		t.FailNow()
	}
}

// CallerString returns the file:line from the call stack
// at the given position (0 = current file:line, 1 = caller of
// the current function, etc)
func CallerString(skip int) string {
	_, file, line, ok := runtime.Caller(skip + 1)
	if !ok {
		return "unknown"
	}
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
	return fmt.Sprintf("%s:%d", short, line)
}
