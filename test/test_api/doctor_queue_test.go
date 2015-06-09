package test_api

import (
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestDoctorQueue(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()

	accountID, err := testData.AuthAPI.CreateAccount("test@spruceleaht.com", "abc", api.RolePatient)
	test.OK(t, err)

	patient := &common.Patient{
		AccountID: encoding.NewObjectID(accountID),
	}
	test.OK(t, testData.DataAPI.RegisterPatient(patient))

	_, err = testData.DB.Exec(`
		INSERT INTO patient_case (patient_id, status, name, clinical_pathway_id)
		VALUES (?, ?, ?, ?)`, patient.ID.Int64(), common.PCStatusActive.String(), "Case Name", 1)
	test.OK(t, err)

	test.OK(t, testData.DataAPI.InsertUnclaimedItemIntoQueue(&api.DoctorQueueItem{
		CareProvidingStateID: 1,
		ItemID:               1,
		PatientCaseID:        1,
		PatientID:            1,
		EventType:            api.DQEventTypePatientVisit,
		Status:               api.DQItemStatusOngoing,
		Description:          "Some visit",
		ShortDescription:     "Visit",
		ActionURL:            app_url.ViewPatientVisitAction(1),
		Tags:                 []string{"abc", "123"},
	}))

	items, err := testData.DataAPI.GetAllItemsInUnclaimedQueue()
	test.OK(t, err)
	test.Equals(t, 1, len(items))
	test.Equals(t, api.DQTUnclaimedQueue, items[0].QueueType)
	qi := items[0]

	test.OK(t, testData.DataAPI.UpdateDoctorQueue([]*api.DoctorQueueUpdate{{
		Action:    api.DQActionRemove,
		QueueItem: qi,
	}}))

	items, err = testData.DataAPI.GetAllItemsInUnclaimedQueue()
	test.OK(t, err)
	test.Equals(t, 0, len(items))
}
