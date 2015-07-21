// Package conc includes helpers for concurrency patterns that avoid some of the most common pitfalls.
package conc

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
