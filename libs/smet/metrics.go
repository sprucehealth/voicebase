package smet

import (
	"sync"

	"github.com/samuel/go-metrics/metrics"
)

// Note: The current dependency on gometrics is an implementation detail that should be hidden from the consumer

// Hide the gometrics dependency by keeping a global cache of the metrics currently in use
var (
	registry = metrics.NewRegistry()
	mux      sync.RWMutex
)

// Init sets the root registry
func Init(r metrics.Registry) {
	mux.Lock()
	defer mux.Unlock()
	registry = r
}

// Scope changes the scope for all metrics added after this point, this is an artifact of the gometrics dependency
func Scope(scope string) {
	mux.Lock()
	defer mux.Unlock()
	registry = registry.Scope(scope)
}
