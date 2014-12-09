package test_doctor_queue

import (
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestConversationItemsInDoctorQueue(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	doctorID := test_integration.GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	test.OK(t, err)

	visit, treatmentPlan := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	patient, err := testData.DataAPI.GetPatientFromPatientVisitID(visit.PatientVisitID)
	test.OK(t, err)
	test_integration.SubmitPatientVisitBackToPatient(treatmentPlan.ID.Int64(), doctor, testData, t)

	caseID, err := testData.DataAPI.GetPatientCaseIDFromPatientVisitID(visit.PatientVisitID)
	test.OK(t, err)

	doctorCli := test_integration.DoctorClient(testData, t, doctorID)
	patientCli := test_integration.PatientClient(testData, t, patient.PatientID.Int64())

	_, err = patientCli.PostCaseMessage(caseID, "foo", nil)
	test.OK(t, err)

	// ensure that an item is inserted into the doctor queue
	pendingItems, err := testData.DataAPI.GetPendingItemsInDoctorQueue(doctorID)
	test.OK(t, err)
	test.Equals(t, 1, len(pendingItems))
	test.Equals(t, api.DQEventTypeCaseMessage, pendingItems[0].EventType)
	test.Equals(t, api.DQItemStatusPending, pendingItems[0].Status)

	// Reply
	_, err = doctorCli.PostCaseMessage(caseID, "bar", nil)
	test.OK(t, err)

	// ensure that the item is marked as completed for the doctor
	pendingItems, err = testData.DataAPI.GetPendingItemsInDoctorQueue(doctorID)
	test.OK(t, err)
	test.Equals(t, 0, len(pendingItems))

	completedItems, err := testData.DataAPI.GetCompletedItemsInDoctorQueue(doctorID)
	test.OK(t, err)
	test.Equals(t, 2, len(completedItems))

	if !(completedItems[0].EventType == api.DQEventTypeCaseMessage || completedItems[1].EventType == api.DQEventTypeCaseMessage) {
		t.Fatal("Expected a case message item in the completed queue for the doctor but found none")
	}
}
