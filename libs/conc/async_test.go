package conc

import (
	"testing"
	"time"
)

func TestGo(t *testing.T) {
	var success bool
	done := make(chan bool, 1)

	Go(func() {
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

	Testing = true
	defer func() { Testing = false }()

	success = false
	done = make(chan bool, 1)
	Go(func() {
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
