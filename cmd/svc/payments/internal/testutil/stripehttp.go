package testutil

import (
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/payments/internal/oauth"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
)

var _ oauth.StripeOAuth = &MockStripeOAuth{}

type MockStripeOAuth struct {
	*mock.Expector
}

// New returns an initialized instance of MockStripeOAuth
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
