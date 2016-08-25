package mock

import (
	"testing"

	"github.com/sprucehealth/backend/libs/hint"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
)

type mockOAuthClient struct {
	*mock.Expector
}

func NewOAuthClient(t *testing.T) *mockOAuthClient {
	return &mockOAuthClient{
		&mock.Expector{
			T: t,
		},
	}
}

func (m *mockOAuthClient) GrantAPIKey(code string) (*hint.PracticeGrant, error) {
	rets := m.Record(code)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*hint.PracticeGrant), mock.SafeError(rets[1])
}
