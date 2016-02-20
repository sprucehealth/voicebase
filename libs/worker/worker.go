package worker

import (
	"time"
)

// Worker represents the interface that mechanisms performing periodic background tasks should conform to
type Worker interface {
	Start()
	Stop(wait time.Duration)
	Started() bool
}
