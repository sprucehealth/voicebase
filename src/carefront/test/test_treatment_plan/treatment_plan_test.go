package test_treatment_plan

import (
	"carefront/api"
	"carefront/test/test_integration"
	"testing"
)

func TestTreatmentPlanStatus(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)
	doctorId := test_integration.GetDoctorIdOfCurrentPrimaryDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal("Unable to get doctor from doctor id " + err.Error())
	}

	// get patient to start a visit
	patientVisitResponse, treatmentPlan := test_integration.SignupAndSubmitPatientVisitForRandomPatient(t, testData, doctor)

	// this treatment plan should be in draft mode
	if treatmentPlan.Status != api.STATUS_DRAFT {
		t.Fatalf("Expected treatmentPlan status to be %s but it was %s instead", api.STATUS_DRAFT, treatmentPlan.Status)
	}

	// once the doctor submits it it should become ACTIVE
	test_integration.SubmitPatientVisitBackToPatient(patientVisitResponse.PatientVisitId, doctor, testData, t)

	drTreatmentPlan, err := testData.DataApi.GetAbridgedTreatmentPlan(treatmentPlan.Id.Int64(), doctorId)
	if err != nil {
		t.Fatal(err.Error())
	}

	if drTreatmentPlan.Status != api.STATUS_ACTIVE {
		t.Fatalf("Expected status to be %s instead it was %s", api.STATUS_ACTIVE, drTreatmentPlan.Status)
	}
}
