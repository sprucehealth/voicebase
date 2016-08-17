package raccess

import (
	"context"

	"github.com/sprucehealth/backend/svc/payments"
)

func (m *resourceAccessor) ConnectVendorAccount(ctx context.Context, req *payments.ConnectVendorAccountRequest) (*payments.ConnectVendorAccountResponse, error) {
	if err := m.canAccessResource(ctx, req.EntityID, m.orgsForEntity); err != nil {
		return nil, err
	}

	resp, err := m.payments.ConnectVendorAccount(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (m *resourceAccessor) CreatePaymentMethod(ctx context.Context, req *payments.CreatePaymentMethodRequest) (*payments.CreatePaymentMethodResponse, error) {
	if err := m.assertIsEntity(ctx, req.EntityID); err != nil {
		return nil, err
	}

	resp, err := m.payments.CreatePaymentMethod(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (m *resourceAccessor) PaymentMethods(ctx context.Context, req *payments.PaymentMethodsRequest) (*payments.PaymentMethodsResponse, error) {
	if err := m.assertIsEntity(ctx, req.EntityID); err != nil {
		return nil, err
	}

	resp, err := m.payments.PaymentMethods(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (m *resourceAccessor) DeletePaymentMethod(ctx context.Context, req *payments.DeletePaymentMethodRequest) (*payments.DeletePaymentMethodResponse, error) {
	// TODO: Add ability to look up payment method by ID and assert entity identity
	resp, err := m.payments.DeletePaymentMethod(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
