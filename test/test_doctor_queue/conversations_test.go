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
	doctorID := test_integration.GetDoctorIdOfCurrentDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorID)
	test.OK(t, err)

	visit, treatmentPlan := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	patient, err := testData.DataApi.GetPatientFromPatientVisitId(visit.PatientVisitId)
	test.OK(t, err)
	test_integration.SubmitPatientVisitBackToPatient(treatmentPlan.Id.Int64(), doctor, testData, t)

	caseID, err := testData.DataApi.GetPatientCaseIdFromPatientVisitId(visit.PatientVisitId)
	test.OK(t, err)

	doctorCli := test_integration.DoctorClient(testData, t, doctorID)
	patientCli := test_integration.PatientClient(testData, t, patient.PatientId.Int64())

	_, err = patientCli.PostCaseMessage(caseID, "foo", nil)
	test.OK(t, err)

	// ensure that an item is inserted into the doctor queue
	pendingItems, err := testData.DataApi.GetPendingItemsInDoctorQueue(doctorID)
	test.OK(t, err)
	test.Equals(t, 1, len(pendingItems))
	test.Equals(t, api.DQEventTypeCaseMessage, pendingItems[0].EventType)
	test.Equals(t, api.DQItemStatusPending, pendingItems[0].Status)

	// Reply
	_, err = doctorCli.PostCaseMessage(caseID, "bar", nil)
	test.OK(t, err)

	// ensure that the item is marked as completed for the doctor
	pendingItems, err = testData.DataApi.GetPendingItemsInDoctorQueue(doctorID)
	test.OK(t, err)
	test.Equals(t, 0, len(pendingItems))

	completedItems, err := testData.DataApi.GetCompletedItemsInDoctorQueue(doctorID)
	test.OK(t, err)
	test.Equals(t, 2, len(completedItems))

	if !(completedItems[0].EventType == api.DQEventTypeCaseMessage || completedItems[1].EventType == api.DQEventTypeCaseMessage) {
		t.Fatal("Expected a case message item in the completed queue for the doctor but found none")
	}
}
