package dispatch

import (
	"errors"
	"testing"
)

type TestEvent struct {
	Foo string
}

func TestDispatcher(t *testing.T) {
	d := New()
	var success bool
	d.Subscribe(func(e *TestEvent) error {
		success = true
		return nil
	})
	if err := d.Publish(&TestEvent{"blah"}); err != nil {
		t.Fatal(err)
	} else if !success {
		t.Fatal("Listener never called")
	}
	d.Subscribe(func(e *TestEvent) error {
		return errors.New("fail")
	})
	success = false
	if err := d.Publish(&TestEvent{"blah"}); err == nil {
		t.Fatal("Expected an error")
	} else if err.Error() != "dispatch: fail" {
		t.Fatalf("Expected error of 'dispatch: fail', got '%s'", err.Error())
	} else if !success {
		t.Fatalf("First listener not called")
	}
}

func TestNonPointer(t *testing.T) {
	d := New()
	var success bool
	d.Subscribe(func(e TestEvent) error {
		success = true
		return nil
	})
	if err := d.Publish(TestEvent{"blah"}); err != nil {
		t.Fatal(err)
	} else if !success {
		t.Fatal("Listener never called")
	}
}
