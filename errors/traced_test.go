package errors

import (
	"errors"
	"testing"
)

func TestTrace(t *testing.T) {
	if tr := Traces(nil); tr != nil {
		t.Fatal("Traces should return nil on a nil error")
	}
	if e := Trace(nil); e != nil {
		t.Fatal("Trace should return nil on a nil error")
	}
	err := testFunc1()
	if ex := "failed [errors/traced_test.go:25]"; err.Error() != ex {
		t.Fatalf("Expected '%s' got '%s'", ex, err.Error())
	}
	if tr := Traces(err); len(tr) != 1 || tr[0] != "errors/traced_test.go:25" {
		t.Fatalf("Expected ['errors/traced_test.go:25'] got %+v", tr)
	}
}

func testFunc1() error {
	return Trace(testFunc2())
}

func testFunc2() error {
	return errors.New("failed")
}
