package workers

import (
	"context"

	"github.com/sprucehealth/backend/cmd/svc/payments/internal/dal"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
)

func (w *Workers) processVendorAccountPendingDisconnected() {
	ctx := context.Background()
	// TODO: figure out how to mitigate partial failure here
	if err := w.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
		vendorAccountsPendingDisconnect, err := dl.VendorAccountsInState(ctx, dal.VendorAccountLifecycleDisconnected, dal.VendorAccountChangeStatePending, 10, dal.ForUpdate)
		if err != nil {
			return errors.Trace(err)
		}
		if len(vendorAccountsPendingDisconnect) == 0 {
			return nil
		}
		golog.Debugf("Found %d vendor accounts PENDING/DISCONNECTED", len(vendorAccountsPendingDisconnect))
		for _, va := range vendorAccountsPendingDisconnect {
			golog.Debugf("Attempting disconnect of vendor account: %s", va.ID)
			switch va.AccountType {
			case dal.VendorAccountAccountTypeStripe:
				if err := w.stripeOAuth.DisconnectStripeAccount(va.ConnectedAccountID); err != nil {
					golog.Errorf("Error while disconnecting vendor STRIPE account %s: %s", va.ID, err)
					continue
				}
			default:
				golog.Errorf("Unable to disconnect vendor account %s because type %s is not understood", va.ID, va.AccountType)
				continue
			}
			golog.Debugf("Deleting vendor account record %s", va.ID)
			if _, err := dl.DeleteVendorAccount(ctx, va.ID); err != nil {
				return errors.Trace(err)
			}
			// TODO: Write down the information about the disconnection somewhere for auditing
			golog.Debugf("Vendor account %s disconnected", va.ID)
		}
		return nil
	}); err != nil {
		golog.Errorf("Encountered error while processing PENDING/DISCONNECTED vendor accounts: %s", err)
	}
}
