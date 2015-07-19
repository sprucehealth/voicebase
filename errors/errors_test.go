package errors

import "testing"

func TestCause(t *testing.T) {
	if e := Cause(nil); e != nil {
		t.Fatal("Cause should return nil for a nil error")
	}
	err := New("foo")
	if e := Cause(err); e != err {
		t.Fatal("Cause for non-aerr should return the error itself")
	}
	err2 := Trace(err)
	if e := Cause(err2); e != err {
		t.Fatal("Cause for aerr should return the original error")
	}
}
