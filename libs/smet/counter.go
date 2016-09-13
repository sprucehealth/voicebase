package smet

import (
	"sync"

	"github.com/samuel/go-metrics/metrics"
)

var (
	counters   = map[string]*Counter{}
	counterMux sync.RWMutex
)

// Counter a count oriented metric
type Counter struct {
	id string
	c  *metrics.Counter
}

// Inc icnreases the counter by 1
func (c *Counter) Inc() {
	c.AddN(1)
}

// AddN increases the counter by N
func (c *Counter) AddN(delta uint64) {
	c.c.Inc(delta)
}

// Count returns the current count of the value
func (c *Counter) Count() uint64 {
	return c.c.Count()
}

// GetCounter returns the provided counter and initializes the metric if it doesn't exist
// TODO: There is likely a way to make this fetch more generic and shared accross metric types
func GetCounter(name string) *Counter {
	counterMux.RLock()
	c := counters[name]
	counterMux.RUnlock()
	if c == nil {
		counterMux.Lock()
		defer counterMux.Unlock()
		c = counters[name]
		if c == nil {
			mc := metrics.NewCounter()
			registry.Add(name, mc)
			c = &Counter{
				id: name,
				c:  mc,
			}
			counters[name] = c
			return c
		}
	}
	return c
}
