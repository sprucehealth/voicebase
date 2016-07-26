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

func TestErrorf(t *testing.T) {
	err := testFuncf1()
	if ex := "Test 123 [backend/libs/errors/errors_test.go:27]"; err.Error() != ex {
		t.Fatalf("Expected '%s' got '%s'", ex, err.Error())
	}
}

func testFuncf1() error {
	return Errorf("Test %d", 123)
}
