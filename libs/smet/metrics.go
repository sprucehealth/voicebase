package smet

import "github.com/samuel/go-metrics/metrics"

// Note: The current dependency on gometrics is an implementation detail that should be hidden from the consumer

// Hide the gometrics dependency by keeping a global cache of the metrics currently in use
var (
	registry = metrics.NewRegistry()
)

// Scope changes the scope for all metrics added after this point, this is an artifact of the gometrics dependency
func Scope(scope string) {
	registry = registry.Scope(scope)
}
