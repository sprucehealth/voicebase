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

	accountID, err := testData.AuthAPI.CreateAccount("cc1@sprucehealth.com", "abc", api.RoleCC)
	test.OK(t, err)
	cc1 := &common.Doctor{
		AccountID: encoding.NewObjectID(accountID),
		Address:   &common.Address{},
	}
	_, err = testData.DataAPI.RegisterDoctor(cc1)
	test.OK(t, err)

	accountID, err = testData.AuthAPI.CreateAccount("cc2@sprucehealth.com", "abc", api.RoleCC)
	test.OK(t, err)
	cc2 := &common.Doctor{
		AccountID: encoding.NewObjectID(accountID),
		Address:   &common.Address{},
	}
	_, err = testData.DataAPI.RegisterDoctor(cc2)
	test.OK(t, err)

	accountID, err = testData.AuthAPI.CreateAccount("test@sprucehealth.com", "abc", api.RolePatient)
	test.OK(t, err)
	patient := &common.Patient{
		AccountID: encoding.NewObjectID(accountID),
	}
	test.OK(t, testData.DataAPI.RegisterPatient(patient))

	_, err = testData.DB.Exec(`
		INSERT INTO patient_case (patient_id, status, name, clinical_pathway_id)
		VALUES (?, ?, ?, ?)`, patient.ID.Int64(), common.PCStatusActive.String(), "Case Name", 1)
	test.OK(t, err)

	// Unclaimed queue

	test.OK(t, testData.DataAPI.InsertUnclaimedItemIntoQueue(&api.DoctorQueueItem{
		CareProvidingStateID: 1,
		ItemID:               1,
		PatientCaseID:        1,
		PatientID:            patient.ID.Int64(),
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

	// Inbox

	items, err = testData.DataAPI.GetPendingItemsInCCQueues()
	test.OK(t, err)
	test.Equals(t, 0, len(items))

	test.OK(t, testData.DataAPI.UpdateDoctorQueue([]*api.DoctorQueueUpdate{{
		Action: api.DQActionInsert,
		QueueItem: &api.DoctorQueueItem{
			DoctorID:         cc1.ID.Int64(),
			ItemID:           1,
			PatientCaseID:    1,
			PatientID:        1,
			EventType:        api.DQEventTypePatientVisit,
			Status:           api.DQItemStatusOngoing,
			Description:      "Some visit",
			ShortDescription: "Visit",
			ActionURL:        app_url.ViewPatientVisitAction(1),
			Tags:             []string{"abc", "123"},
		},
	}}))
	test.OK(t, testData.DataAPI.UpdateDoctorQueue([]*api.DoctorQueueUpdate{{
		Action: api.DQActionInsert,
		QueueItem: &api.DoctorQueueItem{
			DoctorID:         cc2.ID.Int64(),
			ItemID:           1,
			PatientCaseID:    1,
			PatientID:        1,
			EventType:        api.DQEventTypePatientVisit,
			Status:           api.DQItemStatusOngoing,
			Description:      "Some visit",
			ShortDescription: "Visit",
			ActionURL:        app_url.ViewPatientVisitAction(1),
			Tags:             []string{"abc", "123"},
		},
	}}))

	items, err = testData.DataAPI.GetPendingItemsInCCQueues()
	test.OK(t, err)
	test.Equals(t, 2, len(items))

	items, err = testData.DataAPI.GetPendingItemsInDoctorQueue(cc1.ID.Int64())
	test.OK(t, err)
	test.Equals(t, 1, len(items))
}
