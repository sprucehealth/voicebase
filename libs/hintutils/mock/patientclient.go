package mock

import (
	"testing"

	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/go-hint"
)

var _ hint.PatientClient = &mockPatientClient{}

type mockPatientClient struct {
	*mock.Expector
}

func New(t *testing.T) *mockPatientClient {
	return &mockPatientClient{
		&mock.Expector{
			T: t,
		},
	}
}

func (m *mockPatientClient) New(practiceKey string, params *hint.PatientParams) (*hint.Patient, error) {
	rets := m.Record(practiceKey, params)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*hint.Patient), mock.SafeError(rets[1])
}

func (m *mockPatientClient) Get(practiceKey, id string) (*hint.Patient, error) {
	rets := m.Record(practiceKey, id)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*hint.Patient), mock.SafeError(rets[1])
}

func (m *mockPatientClient) Update(practiceKey, id string, params *hint.PatientParams) (*hint.Patient, error) {
	rets := m.Record(practiceKey, id, params)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*hint.Patient), mock.SafeError(rets[1])
}

func (m *mockPatientClient) Delete(practiceKey, id string) error {
	rets := m.Record(practiceKey, id)
	if len(rets) == 0 {
		return nil
	}

	return mock.SafeError(rets[0])

}

func (m *mockPatientClient) List(practiceKey string, params *hint.ListParams) *hint.Iter {
	rets := m.Record(practiceKey, params)
	if len(rets) == 0 {
		return nil
	}

	return rets[0].(*hint.Iter)
}
