package workers

import (
	"time"

	"github.com/sprucehealth/backend/cmd/svc/auth/internal/dal"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/worker"
)

const workerErrMetricName = "AuthWorkerError"

// Workers collection of all workers used by the Auth system
type Workers struct {
	worker.Collection
	dal dal.DAL
	clk clock.Clock
}

// New initializes a collection of all workers used by the Payments system
func New(dl dal.DAL) *Workers {
	w := &Workers{
		dal: dl,
		clk: clock.New(),
	}
	w.AddWorker(worker.NewRepeat(time.Hour*24, w.cleanupExpiredTokens))
	return w
}
