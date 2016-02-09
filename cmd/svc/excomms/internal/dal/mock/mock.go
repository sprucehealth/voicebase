package mock

import (
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/rawmsg"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
)

var _ dal.DAL = &mockDAL{}

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

func (m *mockDAL) Transact(f func(dl dal.DAL) error) error {
	if err := f(m); err != nil {
		return err
	}
	return nil
}

func (m *mockDAL) LookupProvisionedEndpoint(endpoint string, endpiontType models.EndpointType) (*models.ProvisionedEndpoint, error) {
	rets := m.Record(endpoint, endpiontType)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*models.ProvisionedEndpoint), mock.SafeError(rets[1])
}

func (m *mockDAL) ProvisionEndpoint(ppn *models.ProvisionedEndpoint) error {
	rets := m.Record(ppn)
	if len(rets) == 0 {
		return nil
	}

	return mock.SafeError(rets[0])
}

func (m *mockDAL) LogCallEvent(e *models.CallEvent) error {
	rets := m.Record(e)
	if len(rets) == 0 {
		return nil
	}

	return mock.SafeError(rets[0])
}

func (m *mockDAL) CreateSentMessage(sm *models.SentMessage) error {
	rets := m.Record(sm)
	if len(rets) == 0 {
		return nil
	}

	return mock.SafeError(rets[0])
}

func (m *mockDAL) LookupSentMessageByUUID(uuid, destination string) (*models.SentMessage, error) {
	rets := m.Record(uuid, destination)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*models.SentMessage), mock.SafeError(rets[1])
}

func (m *mockDAL) CreateCallRequest(cr *models.CallRequest) error {
	rets := m.Record(cr)
	if len(rets) == 0 {
		return nil
	}

	return mock.SafeError(rets[0])
}

func (m *mockDAL) LookupCallRequest(callSID string) (*models.CallRequest, error) {
	rets := m.Record(callSID)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*models.CallRequest), mock.SafeError(rets[1])
}

func (m *mockDAL) AvailableProxyPhoneNumbers(originatingPhoneNumber phone.Number) ([]*models.ProxyPhoneNumber, error) {
	rets := m.Record(originatingPhoneNumber)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].([]*models.ProxyPhoneNumber), mock.SafeError(rets[1])
}

func (m *mockDAL) CreateProxyPhoneNumberReservation(model *models.ProxyPhoneNumberReservation) error {
	rets := m.Record(model)
	if len(rets) == 0 {
		return nil
	}

	return mock.SafeError(rets[0])
}

func (m *mockDAL) UpdateActiveProxyPhoneNumberReservation(originatingPhoneNumber phone.Number, destinationPhoneNumber, proxyPhoneNumber *phone.Number, update *dal.ProxyPhoneNumberReservationUpdate) (int64, error) {
	rets := m.Record(originatingPhoneNumber, destinationPhoneNumber, proxyPhoneNumber, update)
	if len(rets) == 0 {
		return 0, nil
	}

	return rets[0].(int64), mock.SafeError(rets[1])
}

func (m *mockDAL) ActiveProxyPhoneNumberReservation(originatingPhoneNumber phone.Number, destinationPhoneNumber, proxyPhoneNumber *phone.Number) (*models.ProxyPhoneNumberReservation, error) {
	rets := m.Record(originatingPhoneNumber, destinationPhoneNumber, proxyPhoneNumber)
	if len(rets) == 0 {
		return nil, nil
	}

	if rets[0] == nil {
		return nil, mock.SafeError(rets[1])
	}

	return rets[0].(*models.ProxyPhoneNumberReservation), mock.SafeError(rets[1])
}

func (m *mockDAL) SetCurrentOriginatingNumber(phoneNumber phone.Number, entityID string) error {
	rets := m.Record(phoneNumber, entityID)
	if len(rets) == 0 {
		return nil
	}

	return mock.SafeError(rets[0])
}

func (m *mockDAL) CurrentOriginatingNumber(entityID string) (phone.Number, error) {
	rets := m.Record(entityID)
	if len(rets) == 0 {
		return phone.Number(""), nil
	}

	return rets[0].(phone.Number), mock.SafeError(rets[1])
}

func (m *mockDAL) StoreIncomingRawMessage(rm *rawmsg.Incoming) (uint64, error) {
	rets := m.Record(rm)
	if len(rets) == 0 {
		return 0, nil
	}

	return rets[0].(uint64), mock.SafeError(rets[1])
}

func (m *mockDAL) IncomingRawMessage(id uint64) (*rawmsg.Incoming, error) {
	rets := m.Record(id)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*rawmsg.Incoming), mock.SafeError(rets[1])
}

func (m *mockDAL) StoreMedia(media []*models.Media) error {
	rets := m.Record(media)
	if len(rets) == 0 {
		return nil
	}

	return mock.SafeError(rets[0])
}

func (m *mockDAL) LookupMedia(ids []uint64) (map[uint64]*models.Media, error) {
	rets := m.Record(ids)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(map[uint64]*models.Media), mock.SafeError(rets[1])
}