package mock

import (
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/proxynumber"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
)

var _ proxynumber.Manager = &mockManager{}

type mockManager struct {
	*mock.Expector
}

func NewMockManager(t *testing.T) *mockManager {
	return &mockManager{
		Expector: &mock.Expector{
			T: t,
		},
	}
}

func (m *mockManager) ReserveNumber(originatingNumber, destinationNumber, provisionedNumber phone.Number, destinationEntityID, sourceEntityID, organizationID string) (phone.Number, error) {
	rets := m.Record(originatingNumber, destinationNumber, provisionedNumber, destinationEntityID, sourceEntityID, organizationID)
	if len(rets) == 0 {
		return phone.Number(""), nil
	}

	return rets[0].(phone.Number), mock.SafeError(rets[1])
}

func (m *mockManager) ActiveReservation(originatingNumber, proxyNumber phone.Number) (*models.ProxyPhoneNumberReservation, error) {
	rets := m.Record(originatingNumber, proxyNumber)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*models.ProxyPhoneNumberReservation), mock.SafeError(rets[1])
}

func (m *mockManager) CallStarted(originatingNumber, proxyNumber phone.Number) error {
	rets := m.Record(originatingNumber, proxyNumber)
	if len(rets) == 0 {
		return nil
	}

	return mock.SafeError(rets[0])
}

func (m *mockManager) CallEnded(originatingNumber, proxyNumber phone.Number) error {
	rets := m.Record(originatingNumber, proxyNumber)
	if len(rets) == 0 {
		return nil
	}

	return mock.SafeError(rets[0])
}
