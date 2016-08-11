package workers

import (
	"errors"
	"flag"
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/payments/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/payments/internal/testutil"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
)

func init() {
	flagDebug := flag.Bool("debug", false, "enable debug logging")
	if *flagDebug {
		golog.Default().SetLevel(golog.DEBUG)
	}
}

type tProcessVendorAccountPendingDisconnected struct {
	works     *Workers
	finishers []mock.Finisher
}

func TestProcessVendorAccountPendingDisconnected(t *testing.T) {
	connectedAccountID1 := "connectedAccountID1"
	connectedAccountID2 := "connectedAccountID2"
	id1, err := dal.NewVendorAccountID()
	test.OK(t, err)
	id2, err := dal.NewVendorAccountID()
	test.OK(t, err)
	cases := map[string]struct {
		tDisconnected *tProcessVendorAccountPendingDisconnected
	}{
		"Success-NothingToDisconnect": {
			tDisconnected: func() *tProcessVendorAccountPendingDisconnected {
				mdal := testutil.NewMockDAL(t)
				mdal.Expect(mock.NewExpectation(mdal.VendorAccountsInState, dal.VendorAccountLifecycleDisconnected, dal.VendorAccountChangeStatePending, int64(10), []dal.QueryOption{dal.ForUpdate}))

				mstripeOAuth := testutil.NewMockStripeOAuth(t)
				return &tProcessVendorAccountPendingDisconnected{
					works: &Workers{
						dal:         mdal,
						stripeOAuth: mstripeOAuth,
					},
					finishers: []mock.Finisher{mdal, mstripeOAuth},
				}
			}(),
		},
		"PartialSuccess-Stripe-ErrorWhileDisconnecting-Only1Delete": {
			tDisconnected: func() *tProcessVendorAccountPendingDisconnected {
				mdal := testutil.NewMockDAL(t)
				mdal.Expect(mock.NewExpectation(mdal.VendorAccountsInState, dal.VendorAccountLifecycleDisconnected, dal.VendorAccountChangeStatePending, int64(10), []dal.QueryOption{dal.ForUpdate}).WithReturns(
					[]*dal.VendorAccount{
						{ID: id1, ConnectedAccountID: connectedAccountID1, AccountType: dal.VendorAccountAccountTypeStripe},
						{ID: id2, ConnectedAccountID: connectedAccountID2, AccountType: dal.VendorAccountAccountTypeStripe},
					}, nil))

				mstripeOAuth := testutil.NewMockStripeOAuth(t)
				mstripeOAuth.Expect(mock.NewExpectation(mstripeOAuth.DisconnectStripeAccount, connectedAccountID1).WithReturns(errors.New("Stripe Error:")))
				// should be no delete

				mstripeOAuth.Expect(mock.NewExpectation(mstripeOAuth.DisconnectStripeAccount, connectedAccountID2))
				mdal.Expect(mock.NewExpectation(mdal.DeleteVendorAccount, id2))
				return &tProcessVendorAccountPendingDisconnected{
					works: &Workers{
						dal:         mdal,
						stripeOAuth: mstripeOAuth,
					},
					finishers: []mock.Finisher{mdal, mstripeOAuth},
				}
			}(),
		},
		"Success": {
			tDisconnected: func() *tProcessVendorAccountPendingDisconnected {
				mdal := testutil.NewMockDAL(t)
				mdal.Expect(mock.NewExpectation(mdal.VendorAccountsInState, dal.VendorAccountLifecycleDisconnected, dal.VendorAccountChangeStatePending, int64(10), []dal.QueryOption{dal.ForUpdate}).WithReturns(
					[]*dal.VendorAccount{
						{ID: id1, ConnectedAccountID: connectedAccountID1, AccountType: dal.VendorAccountAccountTypeStripe},
						{ID: id2, ConnectedAccountID: connectedAccountID2, AccountType: dal.VendorAccountAccountTypeStripe},
					}, nil))

				mstripeOAuth := testutil.NewMockStripeOAuth(t)
				mstripeOAuth.Expect(mock.NewExpectation(mstripeOAuth.DisconnectStripeAccount, connectedAccountID1))
				mdal.Expect(mock.NewExpectation(mdal.DeleteVendorAccount, id1))

				mstripeOAuth.Expect(mock.NewExpectation(mstripeOAuth.DisconnectStripeAccount, connectedAccountID2))
				mdal.Expect(mock.NewExpectation(mdal.DeleteVendorAccount, id2))
				return &tProcessVendorAccountPendingDisconnected{
					works: &Workers{
						dal:         mdal,
						stripeOAuth: mstripeOAuth,
					},
					finishers: []mock.Finisher{mdal, mstripeOAuth},
				}
			}(),
		},
	}
	for _, c := range cases {
		c.tDisconnected.works.processVendorAccountPendingDisconnected()
		mock.FinishAll(c.tDisconnected.finishers...)
	}
}
