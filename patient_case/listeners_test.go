package patient_case

import (
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/notify"
	"github.com/sprucehealth/backend/patient"
	"github.com/sprucehealth/backend/test"
)

func init() {
	dispatch.Testing = true
}

type mockDataAPI_listeners struct {
	api.DataAPI
	patient *common.Patient
}

func (m *mockDataAPI_listeners) LocalizedText(langID int64, tags []string) (map[string]string, error) {
	text := make(map[string]string, len(tags))
	for _, t := range tags {
		text[t] = t
	}
	return text, nil
}

func (m *mockDataAPI_listeners) Patient(id int64, basicOnly bool) (*common.Patient, error) {
	if m.patient == nil || id != m.patient.ID.Int64() {
		return nil, api.ErrNotFound("patient")
	}
	return m.patient, nil
}

type mockNotificationManager_listeners struct {
	patient *common.Patient
	msg     *notify.Message
}

func (m *mockNotificationManager_listeners) NotifyPatient(patient *common.Patient, msg *notify.Message) error {
	m.patient = patient
	m.msg = msg
	return nil
}

func TestListeners(t *testing.T) {
	dataAPI := &mockDataAPI_listeners{
		patient: &common.Patient{
			ID: encoding.NewObjectID(1),
		},
	}
	dispatcher := dispatch.New()
	nm := &mockNotificationManager_listeners{}
	InitListeners(dataAPI, dispatcher, nm)

	test.OK(t, dispatcher.Publish(&patient.ParentalConsentCompletedEvent{
		ChildPatientID:  1,
		ParentPatientID: 2,
	}))
	test.Assert(t, nm.patient != nil, "Notification not sent or patient is nil")
	test.Assert(t, nm.msg != nil, "Notification not sent or message is nil")
	test.Equals(t, int64(1), nm.patient.ID.Int64())
	test.Assert(t, nm.msg.ShortMessage != "", "ShortMessage is empty")
}
