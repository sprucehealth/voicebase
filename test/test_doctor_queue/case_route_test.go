package test_doctor_queue

import (
	"strings"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

// This test is to ensure that if a patient has a doctor assigned to their case care team,
// the case is directly routed to the doctor
func TestCaseRoute_DoctorInCareTeam(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	doctorID := test_integration.GetDoctorIDOfCurrentDoctor(testData, t)

	pr := test_integration.SignupRandomTestPatientWithPharmacyAndAddress(t, testData)

	pv := test_integration.CreatePatientVisitForPatient(pr.Patient.ID.Int64(), testData, t)
	intakeData := test_integration.PrepareAnswersForQuestionsInPatientVisit(pv.PatientVisitID, pv.ClientLayout, t)
	test_integration.SubmitAnswersIntakeForPatient(pr.Patient.ID.Int64(), pr.Patient.AccountID.Int64(),
		intakeData, testData, t)

	patientCase, err := testData.DataAPI.GetPatientCaseFromPatientVisitID(pv.PatientVisitID)
	test.OK(t, err)
	test.Equals(t, common.PCStatusOpen, patientCase.Status)

	// add the doctor to the case for the patient
	test.OK(t, testData.DataAPI.AddDoctorToPatientCase(doctorID, patientCase.ID.Int64()))

	test_integration.SubmitPatientVisitForPatient(pr.Patient.ID.Int64(), pv.PatientVisitID, testData, t)

	// there should exist an item in the local queue of the doctor
	pendingItems, err := testData.DataAPI.GetPendingItemsInDoctorQueue(doctorID)
	test.OK(t, err)
	test.Equals(t, 1, len(pendingItems))
	test.Equals(t, "New visit", pendingItems[0].ShortDescription)
	test.Equals(t, true, strings.Contains(pendingItems[0].Description, "New visit"))
	test.Equals(t, 1, testData.SMSAPI.Len())
	test.Equals(t, "You've been selected by a Spruce patient and have a case waiting.", testData.SMSAPI.Sent[0].Text)

	// there should be a permanent access of the doctor to the patient case
	doctorAssignments, err := testData.DataAPI.GetDoctorsAssignedToPatientCase(patientCase.ID.Int64())
	if err != nil {
		t.Fatal(err)
	} else if len(doctorAssignments) != 1 {
		t.Fatalf("Expected 1 doctor assigned to case instead got %d", len(doctorAssignments))
	} else if doctorAssignments[0].Status != api.StatusActive {
		t.Fatalf("Expected permanent assignment of doctor to patient case instead got %s", doctorAssignments[0].Status)
	} else if doctorAssignments[0].ProviderRole != api.RoleDoctor {
		t.Fatalf("Expected a doctor to be assigned to the patient case instead it was %s", doctorAssignments[0].ProviderRole)
	} else if doctorAssignments[0].ProviderID != doctorID {
		t.Fatalf("Expected doctor %d to be assigned to patient case instead got %d", doctorID, doctorAssignments[0].ProviderID)
	}

}
