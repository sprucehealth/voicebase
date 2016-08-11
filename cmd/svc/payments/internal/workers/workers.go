package workers

import (
	"time"

	"github.com/sprucehealth/backend/cmd/svc/payments/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/payments/internal/oauth"
	"github.com/sprucehealth/backend/libs/worker"
)

// Workers collection of all workers used by the Payments system
type Workers struct {
	dal                                    dal.DAL
	stripeOAuth                            oauth.StripeOAuth
	vendorAccountPendingDisconnectedWorker worker.Worker
}

// New initializes a collection of all workers used by the Payments system
func New(dl dal.DAL, stripeSecretKey, stripeClientID string) *Workers {
	w := &Workers{
		dal:         dl,
		stripeOAuth: oauth.NewStripe(stripeSecretKey, stripeClientID),
	}
	w.vendorAccountPendingDisconnectedWorker = worker.NewRepeat(time.Second*15, w.processVendorAccountPendingDisconnected)
	return w
}

// Start starts the service workers
func (m *Workers) Start() {
	m.vendorAccountPendingDisconnectedWorker.Start()
}

// Stop stops the service workers
func (m *Workers) Stop(wait time.Duration) {
	m.vendorAccountPendingDisconnectedWorker.Stop(wait)
}
