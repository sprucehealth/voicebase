package main

import (
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/libs/golog"
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
	switch pm.Type {
	case payments.PAYMENT_METHOD_TYPE_CARD:
		rpm := &models.PaymentCard{
			ID:   pm.ID,
			Type: paymentMethodTypeCard,
		}
		switch pm.StorageType {
		case payments.PAYMENT_METHOD_STORAGE_TYPE_STRIPE:
			sc := pm.GetStripeCard()
			rpm.PaymentProcessor = paymentProcessorStripe
			rpm.TokenizationMethod = sc.TokenizationMethod
			rpm.Brand = sc.Brand
			rpm.Last4 = sc.Last4
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
