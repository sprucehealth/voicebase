package main

import (
	"context"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/device/devicectx"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/payments"
)

func transformPaymentMethodsToResponse(pms []*payments.PaymentMethod) []models.PaymentMethod {
	rpms := make([]models.PaymentMethod, len(pms))
	for i, pm := range pms {
		rpms[i] = transformPaymentMethodToResponse(pm)
	}
	return rpms
}

func transformPaymentMethodToResponse(pm *payments.PaymentMethod) models.PaymentMethod {
	if pm == nil {
		return nil
	}
	switch pm.Type {
	case payments.PAYMENT_METHOD_TYPE_CARD:
		rpm := &models.PaymentCard{
			ID:      pm.ID,
			Default: pm.Default,
			Type:    paymentMethodTypeCard,
		}
		switch pm.StorageType {
		case payments.PAYMENT_METHOD_STORAGE_TYPE_STRIPE:
			// https://stripe.com/docs/api#card_object
			sc := pm.GetStripeCard()
			rpm.PaymentProcessor = paymentProcessorStripe
			rpm.TokenizationMethod = sc.TokenizationMethod
			rpm.Brand = sc.Brand
			rpm.Last4 = sc.Last4
			rpm.IsApplePay = sc.TokenizationMethod == `apple_pay`
			rpm.IsAndroidPay = sc.TokenizationMethod == `android_pay`
		default:
			golog.Errorf("Unhandled payment method storage type %s in %s, unable to transform", pm.StorageType, pm.ID)
			return nil
		}
		return rpm
	default:
		golog.Errorf("Unhandled payment method type %s in %s, unable to transform", pm.Type, pm.ID)
		return nil
	}
}

func transformPaymentsToResponse(ctx context.Context, ps []*payments.Payment, ram raccess.ResourceAccessor, staticURLPrefix string) ([]*models.PaymentRequest, error) {
	rps := make([]*models.PaymentRequest, len(ps))
	for i, p := range ps {
		rp, err := transformPaymentToResponse(ctx, p, ram, staticURLPrefix)
		if err != nil {
			return nil, errors.Trace(err)
		}
		rps[i] = rp
	}
	return rps, nil
}

func transformPaymentToResponse(ctx context.Context, p *payments.Payment, ram raccess.ResourceAccessor, staticURLPrefix string) (*models.PaymentRequest, error) {
	headers := devicectx.SpruceHeaders(ctx)
	account := gqlctx.Account(ctx)
	// TODO: Where some of this info comes from will change in time, this is just to get something working
	requestingEntity, err := raccess.Entity(ctx, ram, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: p.RequestingEntityID,
		},
	})
	if err != nil {
		return nil, errors.Trace(err)
	}
	rRequestingEntity, err := transformEntityToResponse(ctx, staticURLPrefix, requestingEntity, headers, account)
	if err != nil {
		return nil, errors.Trace(err)
	}
	var completedTimestamp uint64
	if p.Lifecycle != payments.PAYMENT_LIFECYCLE_SUBMITTED {
		// TODO: This is a stop gap solution until we get payment history enabled
		completedTimestamp = p.Modified
	}
	return &models.PaymentRequest{
		ID:               p.ID,
		RequestingEntity: rRequestingEntity,
		PaymentMethod:    transformPaymentMethodToResponse(p.PaymentMethod),
		Currency:         p.Currency,
		AmountInCents:    p.Amount,
		// TODO: Figure out what we want this text to be
		Status:          p.ProcessorStatusMessage,
		ProcessingError: p.Lifecycle == payments.PAYMENT_LIFECYCLE_ERROR_PROCESSING,
		// TODO: The source of these two timestamps will change
		RequestedTimestamp: p.Created,
		CompletedTimestamp: completedTimestamp,
		AllowPay:           account.Type == auth.AccountType_PATIENT && p.Lifecycle != payments.PAYMENT_LIFECYCLE_ERROR_PROCESSING,
	}, nil
}
