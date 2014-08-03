package test_doctor_queue

import (
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
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
	doctorID := test_integration.GetDoctorIdOfCurrentDoctor(testData, t)

	pr := test_integration.SignupRandomTestPatientInState("CA", t, testData)

	// assign the doctor to the patient file
	if err := testData.DataApi.AddDoctorToCareTeamForPatient(pr.Patient.PatientId.Int64(), apiservice.HEALTH_CONDITION_ACNE_ID, doctorID); err != nil {
		t.Fatal(err)
	}

	pv := test_integration.CreatePatientVisitForPatient(pr.Patient.PatientId.Int64(), testData, t)
	answerIntakeRequestBody := test_integration.PrepareAnswersForQuestionsInPatientVisit(pv, t)
	test_integration.SubmitAnswersIntakeForPatient(pr.Patient.PatientId.Int64(), pr.Patient.AccountId.Int64(),
		answerIntakeRequestBody, testData, t)
	test_integration.SubmitPatientVisitForPatient(pr.Patient.PatientId.Int64(), pv.PatientVisitId, testData, t)

	// the patient case should now be in the assigned state
	patientCase, err := testData.DataApi.GetPatientCaseFromPatientVisitId(pv.PatientVisitId)
	if err != nil {
		t.Fatal(err)
	} else if patientCase.Status != common.PCStatusClaimed {
		t.Fatalf("Expected patient case to be %s but it was %s", common.PCStatusClaimed, patientCase.Status)
	}

	// there should exist an item in the local queue of the doctor
	pendingItems, err := testData.DataApi.GetPendingItemsInDoctorQueue(doctorID)
	if err != nil {
		t.Fatal(err)
	} else if len(pendingItems) != 1 {
		t.Fatalf("Expected 1 item in doctor's local queue but instead got %d", len(pendingItems))
	}

	// there should be a permanent access of the doctor to the patient case
	doctorAssignments, err := testData.DataApi.GetDoctorsAssignedToPatientCase(patientCase.Id.Int64())
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
