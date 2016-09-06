package testutil

import (
	"testing"

	"context"

	"github.com/sprucehealth/backend/cmd/svc/payments/internal/oauth"
	istripe "github.com/sprucehealth/backend/cmd/svc/payments/internal/stripe"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/stripe/stripe-go"
)

// StripeOAuth
var _ oauth.StripeOAuth = &MockStripeOAuth{}

type MockStripeOAuth struct {
	*mock.Expector
}

// NewMockStripeOAuth returns an initialized instance of MockStripeOAuth
func NewMockStripeOAuth(t *testing.T) *MockStripeOAuth {
	return &MockStripeOAuth{&mock.Expector{T: t}}
}

func (m *MockStripeOAuth) DisconnectStripeAccount(userID string) error {
	rets := m.Record(userID)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}

func (m *MockStripeOAuth) RequestStripeAccessToken(code string) (*oauth.StripeAccessTokenResponse, error) {
	rets := m.Record(code)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*oauth.StripeAccessTokenResponse), mock.SafeError(rets[1])
}

// IdempotentStripeClient
var _ istripe.IdempotentStripeClient = &MockIdempotentStripeClient{}

type MockIdempotentStripeClient struct {
	*mock.Expector
}

//NewMockIdempotentStripeClient returns an initialized instance of MockIdempotentStripeClient
func NewMockIdempotentStripeClient(t *testing.T) *MockIdempotentStripeClient {
	return &MockIdempotentStripeClient{&mock.Expector{T: t}}
}

func (m *MockIdempotentStripeClient) Account(ctx context.Context) (*stripe.Account, error) {
	rets := m.Record()
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*stripe.Account), mock.SafeError(rets[1])
}

func (m *MockIdempotentStripeClient) Card(ctx context.Context, id string, cParams *stripe.CardParams) (*stripe.Card, error) {
	rets := m.Record(id, cParams)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*stripe.Card), mock.SafeError(rets[1])
}

func (m *MockIdempotentStripeClient) CreateCard(ctx context.Context, cParams *stripe.CardParams, opts ...istripe.CallOption) (*stripe.Card, error) {
	rets := m.Record(cParams)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*stripe.Card), mock.SafeError(rets[1])
}

func (m *MockIdempotentStripeClient) CreateCharge(ctx context.Context, cParams *stripe.ChargeParams, opts ...istripe.CallOption) (*stripe.Charge, error) {
	rets := m.Record(cParams)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*stripe.Charge), mock.SafeError(rets[1])
}

func (m *MockIdempotentStripeClient) CreateCustomer(ctx context.Context, cParams *stripe.CustomerParams, opts ...istripe.CallOption) (*stripe.Customer, error) {
	rets := m.Record(cParams)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*stripe.Customer), mock.SafeError(rets[1])
}

func (m *MockIdempotentStripeClient) DeleteCard(ctx context.Context, id string, cParams *stripe.CardParams, opts ...istripe.CallOption) error {
	rets := m.Record(id, cParams)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}

func (m *MockIdempotentStripeClient) Token(ctx context.Context, tParams *stripe.TokenParams, opts ...istripe.CallOption) (*stripe.Token, error) {
	rets := m.Record(tParams)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*stripe.Token), mock.SafeError(rets[1])
}
