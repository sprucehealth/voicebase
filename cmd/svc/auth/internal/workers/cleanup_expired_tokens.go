package workers

import (
	"context"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/auth/internal/dal"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/smet"
)

// Cleanup tokens that are a week expired
var tokenCleanupDelay = time.Hour * 24 * 7

// processPaymentNoneAccepted asserts the existance of the customer and payment method in the context of the vendor
func (w *Workers) cleanupExpiredTokens() {
	ctx := context.Background()
	if err := w.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
		deletedTokenCount, err := dl.DeleteExpiredAuthTokens(ctx, w.clk.Now().Add(-tokenCleanupDelay))
		if err != nil {
			return errors.Trace(err)
		}
		golog.Infof("Cleaned up %d tokens that expired %v ago", deletedTokenCount, tokenCleanupDelay)
		return nil
	}); err != nil {
		smet.Errorf(workerErrMetricName, "Encountered error while cleaning up expired auth tokens: %s", err)
	}
}
