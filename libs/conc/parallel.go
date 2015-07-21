package conc

import "fmt"

// Errors is a slice of multiple errors
type Errors []error

// Error implements the error interface
func (e Errors) Error() string {
	return fmt.Sprintf("%+v", []error(e))
}

// Parallel helps with the pattern of starting multiple goroutines to do work in parallel
// and waiting for them all to complete immediately after (the normal use case for WaitGroup).
// It helps with avoiding some common problems such as catching panics and making sure to
// not block on the error response channel.
type Parallel struct {
	errCh []chan error
}

// NewParallel returns a new instance of Parallel.
func NewParallel() *Parallel {
	return &Parallel{}
}

// Go runs the provided function in the background and handled panic recovery and error capture.
// It should not be called after Wait.
func (p *Parallel) Go(fn func() error) {
	ch := make(chan error, 1)
	p.errCh = append(p.errCh, ch)
	go func() {
		defer func() {
			if e := recover(); e != nil {
				if err, ok := e.(error); ok {
					ch <- err
				} else {
					ch <- fmt.Errorf("runtime error: %v", e)
				}
			}
			close(ch)
		}()
		if err := fn(); err != nil {
			ch <- err
		}
	}()
}

// Wait waits for all goroutines started by Go to complete and returns all errors if any.
func (p *Parallel) Wait() error {
	// Collect errors from goroutines.
	var errors []error
	for _, ch := range p.errCh {
		if err, ok := <-ch; ok && err != nil {
			errors = append(errors, err)
		}
	}
	if len(errors) != 0 {
		return Errors(errors)
	}
	return nil
}
