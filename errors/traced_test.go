package errors

import (
	"errors"
	"testing"
)

func TestTrace(t *testing.T) {
	err := testFunc1()
	if ex := "failed [errors/traced_test.go:16]"; err.Error() != ex {
		t.Fatalf("Expected '%s' got '%s'", ex, err.Error())
	}
}

func testFunc1() error {
	return Trace(testFunc2())
}

func testFunc2() error {
	return errors.New("failed")
}
