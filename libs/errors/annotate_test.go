package errors

import "testing"

// Annotations returns all annotations attached to an error.
func Annotations(err error) []string {
	if e, ok := err.(aerr); ok {
		return e.annotations
	}
	return nil
}

func TestWrap(t *testing.T) {
	if e := Wrap(nil, "XXX"); e != nil {
		t.Error("Wrap should return nil on a nil error")
	}
	if a := Annotations(nil); a != nil {
		t.Error("Annotations should return nil on a nil error")
	}
	e := New("test")
	if a := Annotations(e); a != nil {
		t.Error("Expected no annotations for a non aerr")
	}
	e = Wrap(e, "foo")
	if a := Annotations(e); len(a) != 1 || a[0] != "foo" {
		t.Errorf("Expected ['foo'] got %+v", a)
	}
	e = Wrap(e, "bar")
	if a := Annotations(e); len(a) != 2 || a[0] != "foo" || a[1] != "bar" {
		t.Errorf("Expected ['foo', 'bar'] got %+v", a)
	}
	exp := `test (foo, bar) [backend/libs/errors/annotate_test.go:24, backend/libs/errors/annotate_test.go:28]`
	if es := e.Error(); es != exp {
		t.Errorf("Expected %q, got %q", exp, es)
	}
}

func TestWrapf(t *testing.T) {
	if e := Wrapf(nil, "XXX"); e != nil {
		t.Errorf("Expected nil on a nil error")
	}
	if a := Annotations(nil); a != nil {
		t.Error("Annotations should return nil on a nil error")
	}
	e := New("test")
	if a := Annotations(e); a != nil {
		t.Error("Expected no annotations for a non aerr")
	}
	e = Wrapf(e, "foo%d", 111)
	if a := Annotations(e); len(a) != 1 || a[0] != "foo111" {
		t.Errorf("Expected ['foo111'] got %+v", a)
	}
	e = Wrapf(e, "bar%d%d", 2, 3)
	if a := Annotations(e); len(a) != 2 || a[0] != "foo111" || a[1] != "bar23" {
		t.Errorf("Expected ['foo111', 'bar23'] got %+v", a)
	}
}
