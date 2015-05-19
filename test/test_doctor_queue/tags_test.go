package test_doctor_queue

import (
	"testing"

	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

// This test is to ensure that the tags for items are being surfaced
// for all the common activities
func TestDoctorQueue_Tags(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)
	dc := test_integration.DoctorClient(testData, t, dr.DoctorID)

	// create ma
	test_integration.SignupRandomTestMA(t, testData)

	pv := test_integration.CreateRandomPatientVisitInState("CA", t, testData)
	unassignedItems, err := dc.UnassignedQueue()
	test.OK(t, err)

	test.Equals(t, 1, len(unassignedItems))
	test.Equals(t, 1, len(unassignedItems[0].Tags))
	test.Equals(t, "Acne", unassignedItems[0].Tags[0])

	// do a case assignment to ensure that the item gets moved over to the inbox
	patientCase, err := testData.DataAPI.GetPatientCaseFromPatientVisitID(pv.PatientVisitID)
	// claim the case before doing the case assignment
	test.OK(t, dc.ClaimCase(patientCase.ID.Int64()))

	_, err = dc.AssignCase(patientCase.ID.Int64(), "SUP", nil)
	test.OK(t, err)

	inboxItems, err := dc.Inbox()
	test.OK(t, err)
	test.Equals(t, 1, len(inboxItems))
	test.Equals(t, 1, len(inboxItems[0].Tags))
	test.Equals(t, "Acne", inboxItems[0].Tags[0])

	// submit the treatment plan to ensure that the tag shows up for a completed treatment plan

	// first start reviewing the visit before picking the treatment plan
	// this is so that the expected state transition takes place from ROUTED->REVIEWING
	_, err = dc.ReviewVisit(pv.PatientVisitID)
	test.OK(t, err)

	tp, err := dc.PickTreatmentPlanForVisit(pv.PatientVisitID, nil)
	test.OK(t, err)
	test_integration.SubmitPatientVisitBackToPatient(tp.ID.Int64(), doctor, testData, t)

	completedItems, err := dc.History()
	test.OK(t, err)
	test.Equals(t, 2, len(completedItems))
	test.Equals(t, 1, len(completedItems[1].Tags))
	test.Equals(t, "Acne", completedItems[1].Tags[0])

	// lets have the doctor send a message to the patient
	_, err = dc.PostCaseMessage(tp.PatientCaseID.Int64(), "SUP", nil)
	test.OK(t, err)

	completedItems, err = dc.History()
	test.OK(t, err)
	// Expected events:
	// CASE_ASSIGNMENT
	// TREATMENT_PLAN
	// CASE_MESSAGE
	// CASE_ASSIGNMENT - Reassign to the CC after messaging
	test.Equals(t, 4, len(completedItems))
	test.Equals(t, 1, len(completedItems[2].Tags))
	test.Equals(t, "Acne", completedItems[2].Tags[0])
}
