package workers

import (
	"time"

	"github.com/sprucehealth/backend/cmd/svc/payments/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/payments/internal/oauth"
	"github.com/sprucehealth/backend/cmd/svc/payments/internal/stripe"
	"github.com/sprucehealth/backend/libs/worker"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/threading"
)

const workerErrMetricName = "PaymentsWorkerError"

// Workers collection of all workers used by the Payments system
type Workers struct {
	worker.Collection
	dal             dal.DAL
	webDomain       string
	stripeOAuth     oauth.StripeOAuth
	stripeClient    stripe.IdempotentStripeClient
	directoryClient directory.DirectoryClient
	threadingClient threading.ThreadsClient
}

// New initializes a collection of all workers used by the Payments system
func New(
	dl dal.DAL,
	directoryClient directory.DirectoryClient,
	threadingClient threading.ThreadsClient,
	stripeClient stripe.IdempotentStripeClient,
	stripeSecretKey, stripeClientID, webDomain string) *Workers {
	w := &Workers{
		dal:             dl,
		stripeOAuth:     oauth.NewStripe(stripeSecretKey, stripeClientID),
		stripeClient:    stripeClient,
		webDomain:       webDomain,
		directoryClient: directoryClient,
		threadingClient: threadingClient,
	}
	w.AddWorker(worker.NewRepeat(time.Second*10, w.processVendorAccountPendingDisconnected))
	w.AddWorker(worker.NewRepeat(time.Second*3, w.processPaymentNoneAccepted))
	w.AddWorker(worker.NewRepeat(time.Second*3, w.processPaymentPendingProcessing))
	return w
}
