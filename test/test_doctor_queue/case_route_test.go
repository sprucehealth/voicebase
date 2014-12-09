package test_doctor_queue

import (
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/test/test_integration"
)

// This test is to ensure that if a patient has a doctor assigned to their care team,
// any new case created for the condition supported by the doctor gets directly routed
// to the doctor and permanently assigned to them
func TestCaseRoute_DoctorInCareTeam(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	doctorID := test_integration.GetDoctorIDOfCurrentDoctor(testData, t)

	pr := test_integration.SignupRandomTestPatientWithPharmacyAndAddress(t, testData)

	// assign the doctor to the patient file
	if err := testData.DataAPI.AddDoctorToCareTeamForPatient(pr.Patient.PatientID.Int64(), api.HEALTH_CONDITION_ACNE_ID, doctorID); err != nil {
		t.Fatal(err)
	}

	pv := test_integration.CreatePatientVisitForPatient(pr.Patient.PatientID.Int64(), testData, t)
	answerIntakeRequestBody := test_integration.PrepareAnswersForQuestionsInPatientVisit(pv.PatientVisitID, pv.ClientLayout, t)
	test_integration.SubmitAnswersIntakeForPatient(pr.Patient.PatientID.Int64(), pr.Patient.AccountID.Int64(),
		answerIntakeRequestBody, testData, t)
	test_integration.SubmitPatientVisitForPatient(pr.Patient.PatientID.Int64(), pv.PatientVisitID, testData, t)

	// the patient case should now be in the assigned state
	patientCase, err := testData.DataAPI.GetPatientCaseFromPatientVisitID(pv.PatientVisitID)
	if err != nil {
		t.Fatal(err)
	} else if patientCase.Status != common.PCStatusClaimed {
		t.Fatalf("Expected patient case to be %s but it was %s", common.PCStatusClaimed, patientCase.Status)
	}

	// there should exist an item in the local queue of the doctor
	pendingItems, err := testData.DataAPI.GetPendingItemsInDoctorQueue(doctorID)
	if err != nil {
		t.Fatal(err)
	} else if len(pendingItems) != 1 {
		t.Fatalf("Expected 1 item in doctor's local queue but instead got %d", len(pendingItems))
	}

	// there should be a permanent access of the doctor to the patient case
	doctorAssignments, err := testData.DataAPI.GetDoctorsAssignedToPatientCase(patientCase.ID.Int64())
	if err != nil {
		t.Fatal(err)
	} else if len(doctorAssignments) != 1 {
		t.Fatalf("Expected 1 doctor assigned to case instead got %d", len(doctorAssignments))
	} else if doctorAssignments[0].Status != api.STATUS_ACTIVE {
		t.Fatalf("Expected permanent assignment of doctor to patient case instead got %s", doctorAssignments[0].Status)
	} else if doctorAssignments[0].ProviderRole != api.DOCTOR_ROLE {
		t.Fatalf("Expected a doctor to be assigned to the patient case instead it was %s", doctorAssignments[0].ProviderRole)
	} else if doctorAssignments[0].ProviderID != doctorID {
		t.Fatalf("Expected doctor %d to be assigned to patient case instead got %d", doctorID, doctorAssignments[0].ProviderID)
	}

}
