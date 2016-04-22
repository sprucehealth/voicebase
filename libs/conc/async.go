// Package conc includes helpers for concurrency patterns that avoid some of the most common pitfalls.
package conc

import "time"

// Testing should be set to true when running tests for code that use this package.
// This makes all functionality synchronous and makes tests deterministic.
var Testing bool

// Go runs the provided function in a go routine if Testing is not set,
// and synchronously if it is
func Go(f func()) {
	if !Testing {
		go f()
	} else {
		f()
	}
}

// AfterFunc runs the provided function in a go routine after the provided duration if Testing is not set,
// and synchronously if it is
func AfterFunc(t time.Duration, f func()) *time.Timer {
	if !Testing {
		return time.AfterFunc(t, f)
	}
	f()
	// TODO: Figure out what to do with this timer we're returning in tests
	return nil
}
