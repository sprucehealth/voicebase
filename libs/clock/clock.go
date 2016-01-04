package clock

import "time"

// Clock represents an struct used for individual time management
type Clock interface {
	Now() time.Time
}

// Clock represents a wrapper for time that allows granular management for testing
type clock struct{}

// New returns an initialized instance of clock
func New() Clock {
	return &clock{}
}

// Now returns the current time
func (c *clock) Now() time.Time {
	return time.Now()
}

// ManagedClock is a struct used to hand manage time. Intended for tests
type ManagedClock struct {
	startTime time.Time
	offset    time.Duration
}

// NewManaged returns an initialized instance of managedClock for use in tests
func NewManaged(startTime time.Time) *ManagedClock {
	return &ManagedClock{startTime: startTime}
}

// Now returns the current managed time
func (c *ManagedClock) Now() time.Time {
	return c.startTime.Add(c.offset)
}

// WarpForward moves time forward by the provided offset within the clock and returns the new time
func (c *ManagedClock) WarpForward(offset time.Duration) time.Time {
	c.offset = c.offset + offset
	return c.startTime.Add(c.offset)
}

// Why is there no WarpBackward? Time should never go backwards, especially in your tests
