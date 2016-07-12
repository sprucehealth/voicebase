package conc

import (
	"testing"
	"time"

	"context"
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

func TestGoCtx(t *testing.T) {
	var success bool
	done := make(chan bool, 1)

	ctx := context.Background()
	ctx = context.WithValue(ctx, "ami", "thesame")
	GoCtx(ctx, func(ctx context.Context) {
		if ctx.Value("ami") != "thesame" {
			t.Fatal("Context expected to be the same as arg")
		}
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

	ctx = context.Background()
	ctx = context.WithValue(ctx, "ami", "thesame")
	GoCtx(ctx, func(ctx context.Context) {
		if ctx.Value("ami") != "thesame" {
			t.Fatal("Context expected to be the same as arg")
		}
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

func TestAfterFunc(t *testing.T) {
	var success bool
	done := make(chan bool, 1)

	AfterFunc(time.Second*1, func() {
		success = true
		done <- true
	})

	select {
	case <-done:
		if !success {
			t.Fatal("Success should be true but was false")
		}
	case <-time.After(time.Second * 5):
		t.Fatal("Expected function never ran")
	}

	Testing = true
	defer func() { Testing = false }()

	success = false
	done = make(chan bool, 1)
	AfterFunc(time.Second*1, func() {
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
