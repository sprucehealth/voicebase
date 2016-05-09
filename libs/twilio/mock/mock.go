package mock

import (
	"testing"

	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/libs/twilio"
)

var _ twilio.IncomingPhoneNumberIFace = &MockIncomingPhoneNumberService{}

type MockIncomingPhoneNumberService struct {
	*mock.Expector
}

func NewIncomingPhoneNumber(t *testing.T) *MockIncomingPhoneNumberService {
	return &MockIncomingPhoneNumberService{
		Expector: &mock.Expector{
			T: t,
		},
	}
}

func (m *MockIncomingPhoneNumberService) PurchaseLocal(params twilio.PurchasePhoneNumberParams) (*twilio.IncomingPhoneNumber, *twilio.Response, error) {
	rets := m.Record(params)
	if len(rets) == 0 {
		return nil, nil, nil
	}

	return rets[0].(*twilio.IncomingPhoneNumber), rets[1].(*twilio.Response), mock.SafeError(rets[2])
}

func (m *MockIncomingPhoneNumberService) List(params twilio.ListPurchasedPhoneNumberParams) (*twilio.ListPurchasedPhoneNumbersResponse, *twilio.Response, error) {
	rets := m.Record(params)
	if len(rets) == 0 {
		return nil, nil, nil
	}

	return rets[0].(*twilio.ListPurchasedPhoneNumbersResponse), rets[1].(*twilio.Response), mock.SafeError(rets[2])
}

func (m *MockIncomingPhoneNumberService) Delete(sid string) (*twilio.Response, error) {
	rets := m.Record(sid)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*twilio.Response), mock.SafeError(rets[1])
}

func (m *MockIncomingPhoneNumberService) Update(sid string, update twilio.UpdatePurchasedPhoneNumberParams) (*twilio.IncomingPhoneNumber, *twilio.Response, error) {
	rets := m.Record(sid, update)
	if len(rets) == 0 {
		return nil, nil, nil
	}

	return rets[0].(*twilio.IncomingPhoneNumber), rets[1].(*twilio.Response), mock.SafeError(rets[2])
}
