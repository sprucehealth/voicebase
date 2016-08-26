package workers

import (
	"time"

	"github.com/sprucehealth/backend/cmd/svc/payments/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/payments/internal/oauth"
	"github.com/sprucehealth/backend/cmd/svc/payments/internal/stripe"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/worker"
	"github.com/sprucehealth/backend/svc/directory"
)

// Workers collection of all workers used by the Payments system
type Workers struct {
	dal             dal.DAL
	stripeOAuth     oauth.StripeOAuth
	stripeClient    stripe.IdempotentStripeClient
	directoryClient directory.DirectoryClient
	workers         []worker.Worker
}

// New initializes a collection of all workers used by the Payments system
func New(dl dal.DAL, directoryClient directory.DirectoryClient, stripeSecretKey, stripeClientID string) *Workers {
	w := &Workers{
		dal:             dl,
		stripeOAuth:     oauth.NewStripe(stripeSecretKey, stripeClientID),
		stripeClient:    stripe.NewClient(stripeSecretKey),
		directoryClient: directoryClient,
	}
	w.workers = append(w.workers, worker.NewRepeat(time.Second*10, w.processVendorAccountPendingDisconnected))
	// TODO: We should stagger this query relative to the number of processes. V1 just poll every 3 seconds
	w.workers = append(w.workers, worker.NewRepeat(time.Second*3, w.processPaymentNoneAccepted))
	w.workers = append(w.workers, worker.NewRepeat(time.Second*3, w.processPaymentPendingProcessing))
	return w
}

// Start starts the service workers
func (w *Workers) Start() {
	for _, wk := range w.workers {
		// Capture worker
		wk := wk
		conc.Go(wk.Start)
	}
}

// Stop stops the service workers
func (w *Workers) Stop(wait time.Duration) {
	parallel := conc.NewParallel()
	for _, wk := range w.workers {
		// Capture worker
		wk := wk
		parallel.Go(func() error {
			wk.Stop(wait)
			return nil
		})
	}
	parallel.Wait()
}
