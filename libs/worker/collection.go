package worker

import (
	"time"

	"github.com/sprucehealth/backend/libs/conc"
)

// Collection is a collection of workers
type Collection struct {
	workers []Worker
}

// Start starts the workers
func (c *Collection) Start() {
	for _, wk := range c.workers {
		// Capture worker
		wk := wk
		conc.Go(wk.Start)
	}
}

// Stop stops the workers
func (c *Collection) Stop(wait time.Duration) {
	parallel := conc.NewParallel()
	for _, wk := range c.workers {
		// Capture worker
		wk := wk
		parallel.Go(func() error {
			wk.Stop(wait)
			return nil
		})
	}
	parallel.Wait()
}

// AddWorker adds a worker to the collection of managed workers
func (c *Collection) AddWorker(w Worker) {
	c.workers = append(c.workers, w)
}
