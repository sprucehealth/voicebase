// Package test utils inspired from https://github.com/benbjohnson/testing
package test

import (
	"fmt"
	"net/http/httptest"
	"reflect"
	"runtime"
	"testing"

	"github.com/kr/pretty"
)

// Assert fails the test if the condition is false.
func Assert(t testing.TB, condition bool, msg string, v ...interface{}) {
	if !condition {
		t.Fatalf("["+CallerString(1)+"] "+msg, v...)
	}
}

// OK fails the test if an err is not nil.
func OK(t testing.TB, err error) {
	if err != nil {
		t.Fatalf("unexpected error [%s]: %s", CallerString(1), err.Error())
	}
}

// Equals fails the test if exp is not equal to act.
func Equals(t testing.TB, exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		t.Fatalf(pretty.Sprintf("[%s] difference: %s", CallerString(1), pretty.Diff(exp, act)))
	}
}

// EqualsCase is a utility equals function that reports the test case name on failure
func EqualsCase(t testing.TB, caseName string, exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		t.Fatalf(pretty.Sprintf("[%s]:[%s] difference: %s", caseName, CallerString(1), pretty.Diff(exp, act)))
	}
}

// AssertNil fails the test if the provided value is not nil
func AssertNil(t testing.TB, e interface{}) {
	if !reflect.ValueOf(e).IsNil() {
		t.Fatalf("[%s] Expected a nil value but got %+v", CallerString(1), e)
	}
}

// AssertNotNil fails the test if the provided value is nil
func AssertNotNil(t testing.TB, e interface{}) {
	if reflect.ValueOf(e).IsNil() {
		t.Fatalf("[%s] Expected a non nil value but got %+v", CallerString(1), e)
	}
}

// HTTPResponseCode fails the test if the response code does not match. Upon failure it
// will output the response body for easier debugging.
func HTTPResponseCode(t testing.TB, exp int, res *httptest.ResponseRecorder) {
	if res.Code != exp {
		t.Fatalf("[%s]\nexp status code: %d\ngot status code: %d\nbody: %s", CallerString(1), exp, res.Code, res.Body.String())
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
