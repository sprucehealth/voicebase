package server

import (
	"context"

	"github.com/sprucehealth/backend/cmd/svc/payments/internal/dal"
	istripe "github.com/sprucehealth/backend/cmd/svc/payments/internal/stripe"
	"github.com/sprucehealth/backend/libs/errors"
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
		rva.Type = payments.VENDOR_ACCOUNT_TYPE_STRIPE
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

func transformVendorAccountChangeStateToResponse(vc dal.VendorAccountChangeState) payments.VendorAccountChangeState {
	switch vc {
	case dal.VendorAccountChangeStateNone:
		return payments.VENDOR_ACCOUNT_CHANGE_STATE_NONE
	case dal.VendorAccountChangeStatePending:
		return payments.VENDOR_ACCOUNT_CHANGE_STATE_PENDING
	}
	golog.Errorf("Unknown VendorAccountChangeState %s", vc)
	return payments.VENDOR_ACCOUNT_CHANGE_STATE_UNKNOWN
}

func transformVendorAccountLifecycleToDAL(vl payments.VendorAccountLifecycle) (dal.VendorAccountLifecycle, error) {
	switch vl {
	case payments.VENDOR_ACCOUNT_LIFECYCLE_CONNECTED:
		return dal.VendorAccountLifecycleConnected, nil
	case payments.VENDOR_ACCOUNT_LIFECYCLE_DISCONNECTED:
		return dal.VendorAccountLifecycleDisconnected, nil
	}
	return "", errors.Errorf("Unknown VendorAccountLifecycle %s", vl)
}

func transformVendorAccountChangeStateToDAL(vc payments.VendorAccountChangeState) (dal.VendorAccountChangeState, error) {
	switch vc {
	case payments.VENDOR_ACCOUNT_CHANGE_STATE_NONE:
		return dal.VendorAccountChangeStateNone, nil
	case payments.VENDOR_ACCOUNT_CHANGE_STATE_PENDING:
		return dal.VendorAccountChangeStatePending, nil
	}
	return "", errors.Errorf("Unknown VendorAccountChangeState %s", vc)
}

func transformPaymentMethodsToResponse(pms []*dal.PaymentMethod) []*payments.PaymentMethod {
	rpms := make([]*payments.PaymentMethod, len(pms))
	for i, pm := range pms {
		rpm := transformPaymentMethodToResponse(pm)
		// TODO: For now just assume that the first payment method is the default one since we should be sorting by created time desc
		// 	We may track a true default in the future.
		if i == 0 {
			rpm.Default = true
		}
		rpms[i] = rpm
	}
	return rpms
}

func transformPaymentMethodToResponse(pm *dal.PaymentMethod) *payments.PaymentMethod {
	rpm := &payments.PaymentMethod{
		ID:          pm.ID.String(),
		EntityID:    pm.EntityID,
		Lifecycle:   transformPaymentMethodLifecycleToResponse(pm.Lifecycle),
		ChangeState: transformPaymentMethodChangeStateToResponse(pm.ChangeState),
		StorageType: transformPaymentMethodStorageTypeToResponse(pm.StorageType),
		Type:        TransformPaymentMethodTypeToResponse(pm.Type),
	}
	switch pm.StorageType {
	case dal.PaymentMethodStorageTypeStripe:
		switch pm.Type {
		case dal.PaymentMethodTypeCard:
			rpm.PaymentMethodOneof = &payments.PaymentMethod_StripeCard{
				StripeCard: &payments.StripeCard{
					ID:                 pm.StorageID,
					TokenizationMethod: pm.TokenizationMethod,
					Brand:              pm.Brand,
					Last4:              pm.Last4,
				},
			}
		default:
			golog.Errorf("Unknown payment method type %s - id %s - cannot transform fully", pm.Type, pm.ID)
		}
	default:
		golog.Errorf("Unknown payment method storage type %s - id %s - cannot transform fully", pm.StorageType, pm.ID)
	}
	return rpm
}

func transformPaymentMethodLifecycleToResponse(vl dal.PaymentMethodLifecycle) payments.PaymentMethodLifecycle {
	switch vl {
	case dal.PaymentMethodLifecycleActive:
		return payments.PAYMENT_METHOD_LIFECYCLE_ACTIVE
	case dal.PaymentMethodLifecycleDeleted:
		return payments.PAYMENT_METHOD_LIFECYCLE_DELETED
	}
	golog.Errorf("Unknown PaymentMethodLifecycle %s", vl)
	return payments.PAYMENT_METHOD_LIFECYCLE_UNKNOWN
}

func transformPaymentMethodChangeStateToResponse(vc dal.PaymentMethodChangeState) payments.PaymentMethodChangeState {
	switch vc {
	case dal.PaymentMethodChangeStateNone:
		return payments.PAYMENT_METHOD_CHANGE_STATE_NONE
	case dal.PaymentMethodChangeStatePending:
		return payments.PAYMENT_METHOD_CHANGE_STATE_PENDING
	}
	golog.Errorf("Unknown PaymentMethodChangeState %s", vc)
	return payments.PAYMENT_METHOD_CHANGE_STATE_UNKNOWN
}

func transformPaymentMethodStorageTypeToResponse(pst dal.PaymentMethodStorageType) payments.PaymentMethodStorageType {
	switch pst {
	case dal.PaymentMethodStorageTypeStripe:
		return payments.PAYMENT_METHOD_STORAGE_TYPE_STRIPE
	}
	golog.Errorf("Unknown PaymentMethodStorageType %s", pst)
	return payments.PAYMENT_METHOD_STORAGE_TYPE_UNKNOWN
}

func TransformPaymentMethodTypeToResponse(pt dal.PaymentMethodType) payments.PaymentMethodType {
	switch pt {
	case dal.PaymentMethodTypeCard:
		return payments.PAYMENT_METHOD_TYPE_CARD
	}
	golog.Errorf("Unknown PaymentMethodType %s", pt)
	return payments.PAYMENT_METHOD_TYPE_UNKNOWN
}

func transformPaymentsToResponse(ctx context.Context, ps []*dal.Payment, dl dal.DAL, stripeClient istripe.IdempotentStripeClient) ([]*payments.Payment, error) {
	rps := make([]*payments.Payment, len(ps))
	for i, p := range ps {
		rp, err := transformPaymentToResponse(ctx, p, dl, stripeClient)
		if err != nil {
			return nil, errors.Trace(err)
		}
		rps[i] = rp
	}
	return rps, nil
}

func transformPaymentToResponse(ctx context.Context, p *dal.Payment, dl dal.DAL, stripeClient istripe.IdempotentStripeClient) (*payments.Payment, error) {
	// TODO: We could do these lookups in bulk to optimize here
	vendorAccount, err := dl.VendorAccount(ctx, p.VendorAccountID)
	if err != nil {
		return nil, errors.Trace(err)
	}

	var rPaymentMethod *payments.PaymentMethod
	if p.PaymentMethodID.IsValid {
		paymentMethod, err := dl.PaymentMethod(ctx, p.PaymentMethodID)
		if err != nil {
			return nil, errors.Trace(err)
		}
		rPaymentMethod = transformPaymentMethodToResponse(paymentMethod)
	}

	return &payments.Payment{
		ID:                     p.ID.String(),
		RequestingEntityID:     vendorAccount.EntityID,
		PaymentMethod:          rPaymentMethod,
		Amount:                 p.Amount,
		Currency:               p.Currency,
		Lifecycle:              transformPaymentLifecycleToResponse(p.Lifecycle),
		ChangeState:            transformPaymentChangeStateToResponse(p.ChangeState),
		Created:                uint64(p.Created.Unix()),
		Modified:               uint64(p.Modified.Unix()),
		ThreadID:               p.ThreadID,
		ProcessorTransactionID: p.ProcessorTransactionID,
		ProcessorStatusMessage: p.ProcessorStatusMessage,
	}, nil
}

func transformPaymentLifecycleToResponse(pl dal.PaymentLifecycle) payments.PaymentLifecycle {
	switch pl {
	case dal.PaymentLifecycleSubmitted:
		return payments.PAYMENT_LIFECYCLE_SUBMITTED
	case dal.PaymentLifecycleAccepted:
		return payments.PAYMENT_LIFECYCLE_ACCEPTED
	case dal.PaymentLifecycleProcessing:
		return payments.PAYMENT_LIFECYCLE_PROCESSING
	case dal.PaymentLifecycleErrorProcessing:
		return payments.PAYMENT_LIFECYCLE_ERROR_PROCESSING
	case dal.PaymentLifecycleCompleted:
		return payments.PAYMENT_LIFECYCLE_COMPLETED
	}
	golog.Errorf("Unknown PaymentLifecycle %s", pl)
	return payments.PAYMENT_LIFECYCLE_UNKNOWN
}

func transformPaymentChangeStateToResponse(pc dal.PaymentChangeState) payments.PaymentChangeState {
	switch pc {
	case dal.PaymentChangeStateNone:
		return payments.PAYMENT_CHANGE_STATE_NONE
	case dal.PaymentChangeStatePending:
		return payments.PAYMENT_CHANGE_STATE_PENDING
	}
	golog.Errorf("Unknown PaymentChangeState %s", pc)
	return payments.PAYMENT_CHANGE_STATE_UNKNOWN
}
