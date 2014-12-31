package dispatch

import (
	"errors"
	"sync"
	"testing"
	"time"
)

type TestEvent struct {
	Foo string
}

func TestRunAsync(t *testing.T) {
	var success bool
	done := make(chan bool, 1)

	RunAsync(func() {
		success = true
		done <- true
	})

	select {
	case <-done:
		if !success {
			t.Fatal("Success should be true but was false")
		}
	case <-time.After(time.Second):
		t.Fatal("Expected function never ran")
	}
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

func TestDispatcher_SyncAsyncSubscribers(t *testing.T) {
	d := New()

	var counter1 int64
	var wg sync.WaitGroup
	wg.Add(1)
	d.SubscribeAsync(func(e *TestEvent) error {
		counter1++
		wg.Done()
		return nil
	})

	var counter2 int64
	wg.Add(1)
	d.Subscribe(func(e *TestEvent) error {
		counter2++
		wg.Done()
		return nil
	})

	if err := d.Publish(&TestEvent{"blak"}); err != nil {
		t.Fatal(err)
	}

	wg.Wait()

	if counter1 != 1 {
		t.Fatalf("Expected counter1=1 but got %d", counter1)
	}
	if counter2 != 1 {
		t.Fatalf("Expeced counter2=1 but got %d", counter2)
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
