package errors

import "testing"

func TestAnnotate(t *testing.T) {
	if e := Annotate(nil, "XXX"); e != nil {
		t.Error("Annotate should return nil on a nil error")
	}
	if a := Annotations(nil); a != nil {
		t.Error("Annotations should return nil on a nil error")
	}
	e := New("test")
	if a := Annotations(e); a != nil {
		t.Error("Expected no annotations for a non aerr")
	}
	e = Annotate(e, "foo")
	if a := Annotations(e); len(a) != 1 || a[0] != "foo" {
		t.Errorf("Expected ['foo'] got %+v", a)
	}
	e = Annotate(e, "bar")
	if a := Annotations(e); len(a) != 2 || a[0] != "foo" || a[1] != "bar" {
		t.Errorf("Expected ['foo', 'bar'] got %+v", a)
	}
	if es := e.Error(); es != "test (foo, bar)" {
		t.Errorf("Expected 'test (foo, bar)', got '%s'", es)
	}
}

func TestAnnotatef(t *testing.T) {
	if e := Annotatef(nil, "XXX"); e != nil {
		t.Errorf("Expected nil on a nil error")
	}
	if a := Annotations(nil); a != nil {
		t.Error("Annotations should return nil on a nil error")
	}
	e := New("test")
	if a := Annotations(e); a != nil {
		t.Error("Expected no annotations for a non aerr")
	}
	e = Annotatef(e, "foo%d", 111)
	if a := Annotations(e); len(a) != 1 || a[0] != "foo111" {
		t.Errorf("Expected ['foo111'] got %+v", a)
	}
	e = Annotatef(e, "bar%d%d", 2, 3)
	if a := Annotations(e); len(a) != 2 || a[0] != "foo111" || a[1] != "bar23" {
		t.Errorf("Expected ['foo111', 'bar23'] got %+v", a)
	}
}
