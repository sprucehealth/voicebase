package mock

import (
	"github.com/sprucehealth/backend/libs/testhelpers/mock"

	"testing"
)

type mockDAL struct {
	*mock.Expector
}

func New(t *testing.T) *mockDAL {
	return &mockDAL{
		Expector: &mock.Expector{
			T: t,
		},
	}
}

func (m *mockDAL) MarkAccountAsBlocked(email string) error {
	rets := m.Record(email)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}
