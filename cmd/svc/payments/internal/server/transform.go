package server

import (
	"github.com/sprucehealth/backend/cmd/svc/payments/internal/dal"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/payments"
)

func transformVendorAccountsToResponse(vas []*dal.VendorAccount) []*payments.VendorAccount {
	rvas := make([]*payments.VendorAccount, len(vas))
	for i, va := range vas {
		rvas[i] = transformVendorAccountToResponse(va)
	}
	return rvas
}

func transformVendorAccountToResponse(va *dal.VendorAccount) *payments.VendorAccount {
	rva := &payments.VendorAccount{
		ID:          va.ID.String(),
		EntityID:    va.EntityID,
		Lifecycle:   transformVendorAccountLifecycleToResponse(va.Lifecycle),
		ChangeState: transformVendorAccountChangeStateToResponse(va.ChangeState),
		Live:        va.Live,
	}
	switch va.AccountType {
	case dal.VendorAccountAccountTypeStripe:
		rva.VendorAccountType = payments.VENDOR_ACCOUNT_TYPE_STRIPE
		rva.VendorAccountOneof = &payments.VendorAccount_StripeAccount{
			StripeAccount: &payments.StripeAccount{
				UserID: va.ConnectedAccountID,
				Scope:  va.Scope,
			},
		}
	default:
		golog.Errorf("Unknown vendor account type %s - id %s - cannot transform fully", va.AccountType, va.ID)
	}

	return rva
}

func transformVendorAccountLifecycleToResponse(vl dal.VendorAccountLifecycle) payments.VendorAccountLifecycle {
	switch vl {
	case dal.VendorAccountLifecycleConnected:
		return payments.VENDOR_ACCOUNT_LIFECYCLE_CONNECTED
	case dal.VendorAccountLifecycleDisconnected:
		return payments.VENDOR_ACCOUNT_LIFECYCLE_DISCONNECTED
	}
	golog.Errorf("Unknown VendorAccountLifecycle %s", vl)
	return payments.VENDOR_ACCOUNT_LIFECYCLE_UNKNOWN
}

func transformVendorAccountChangeStateToResponse(vl dal.VendorAccountChangeState) payments.VendorAccountChangeState {
	switch vl {
	case dal.VendorAccountChangeStateNone:
		return payments.VENDOR_ACCOUNT_CHANGE_STATE_NONE
	case dal.VendorAccountChangeStatePending:
		return payments.VENDOR_ACCOUNT_CHANGE_STATE_PENDING
	}
	golog.Errorf("Unknown VendorAccountChangeState %s", vl)
	return payments.VENDOR_ACCOUNT_CHANGE_STATE_UNKNOWN
}
