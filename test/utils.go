// Test utils inspired from https://github.com/benbjohnson/testing
package test

import (
	"reflect"
	"testing"

	"github.com/sprucehealth/backend/libs/golog"
)

// assert fails the test if the condition is false.
func Assert(t testing.TB, condition bool, msg string, v ...interface{}) {
	if !condition {
		golog.LogDepthf(1, golog.ERR, msg+"\n\n", v...)
		t.FailNow()
	}
}

// ok fails the test if an err is not nil.
func OK(t testing.TB, err error) {
	if err != nil {
		golog.LogDepthf(1, golog.ERR, "unexpected error: %s\n\n", err.Error())
		t.FailNow()
	}
}

// equals fails the test if exp is not equal to act.
func Equals(t testing.TB, exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		golog.LogDepthf(1, golog.ERR, "\n\n\texp: %#v\n\n\tgot: %#v\n\n", exp, act)
		t.FailNow()
	}
}
