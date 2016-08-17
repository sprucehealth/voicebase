package models

import (
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/payments"
)

// VendorAccount represents the values contained in the payments service
type VendorAccount struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	AccountID   string `json:"accountID"`
	Lifecycle   string `json:"lifecycle"`
	ChangeState string `json:"changeState"`
	Live        bool   `json:"live"`
}

// TransformVendorAccountsToModel transforms the internal payments vendor accounts into something understood by graphql
func TransformVendorAccountsToModel(vas []*payments.VendorAccount) []*VendorAccount {
	mvas := make([]*VendorAccount, len(vas))
	for i, va := range vas {
		mvas[i] = TransformVendorAccountToModel(va)
	}
	return mvas
}

// TransformVendorAccountToModel transforms the internal payments vendor account into something understood by graphql
func TransformVendorAccountToModel(va *payments.VendorAccount) *VendorAccount {
	mva := &VendorAccount{
		ID:          va.ID,
		Type:        friendlyVendorAccountType(va.Type),
		Lifecycle:   friendlyVendorAccountLifecycle(va.Lifecycle),
		ChangeState: friendlyVendorAccountChangeState(va.ChangeState),
		Live:        va.Live,
	}
	switch va.Type {
	case payments.VENDOR_ACCOUNT_TYPE_STRIPE:
		mva.AccountID = va.GetStripeAccount().UserID
	default:
		golog.Errorf("Unknown vendor account type %s", va.Type)
	}
	return mva
}

const (
	// FriendlyVendorAccountTypeUnknown unknown
	FriendlyVendorAccountTypeUnknown = "UNKNOWN"

	// FriendlyVendorAccountTypeStripe a stripe account
	FriendlyVendorAccountTypeStripe = "STRIPE"
)

func friendlyVendorAccountType(at payments.VendorAccountType) string {
	switch at {
	case payments.VENDOR_ACCOUNT_TYPE_UNKNOWN:
		return FriendlyVendorAccountTypeUnknown
	case payments.VENDOR_ACCOUNT_TYPE_STRIPE:
		return FriendlyVendorAccountTypeStripe
	}
	return at.String()
}

const (
	// FriendlyVendorAccountLifecycleUnknown unknown
	FriendlyVendorAccountLifecycleUnknown = "UNKNOWN"

	// FriendlyVendorAccountLifecycleConnected connected
	FriendlyVendorAccountLifecycleConnected = "CONNECTED"

	// FriendlyVendorAccountLifecycleDisconnected disconnected
	FriendlyVendorAccountLifecycleDisconnected = "DISCONNECTED"
)

func friendlyVendorAccountLifecycle(al payments.VendorAccountLifecycle) string {
	switch al {
	case payments.VENDOR_ACCOUNT_LIFECYCLE_UNKNOWN:
		return FriendlyVendorAccountLifecycleUnknown
	case payments.VENDOR_ACCOUNT_LIFECYCLE_CONNECTED:
		return FriendlyVendorAccountLifecycleConnected
	case payments.VENDOR_ACCOUNT_LIFECYCLE_DISCONNECTED:
		return FriendlyVendorAccountLifecycleDisconnected
	}
	return al.String()
}

const (
	// FriendlyVendorAccountChangeStateUnknown unknown
	FriendlyVendorAccountChangeStateUnknown = "UNKNOWN"

	// FriendlyVendorAccountChangeStateNone connected
	FriendlyVendorAccountChangeStateNone = "NONE"

	// FriendlyVendorAccountChangeStatePending disconnected
	FriendlyVendorAccountChangeStatePending = "PENDING"
)

func friendlyVendorAccountChangeState(acs payments.VendorAccountChangeState) string {
	switch acs {
	case payments.VENDOR_ACCOUNT_CHANGE_STATE_UNKNOWN:
		return FriendlyVendorAccountChangeStateUnknown
	case payments.VENDOR_ACCOUNT_CHANGE_STATE_NONE:
		return FriendlyVendorAccountChangeStateNone
	case payments.VENDOR_ACCOUNT_CHANGE_STATE_PENDING:
		return FriendlyVendorAccountChangeStatePending
	}
	return acs.String()
}
