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

func TestTreatmentPlanList(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)
	doctorId := test_integration.GetDoctorIdOfCurrentPrimaryDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal("Unable to get doctor from doctor id " + err.Error())
	}

	patientVisitResponse, _ := test_integration.SignupAndSubmitPatientVisitForRandomPatient(t, testData, doctor)

	patientId, err := testData.DataApi.GetPatientIdFromPatientVisitId(patientVisitResponse.PatientVisitId)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// before submitting treatment plan if we try to get a list of treatment plans for patient there should be 1 in draft mode
	treatmentPlanResponse := test_integration.GetListOfTreatmentPlansForPatient(patientId, doctor.AccountId.Int64(), testData, t)
	if len(treatmentPlanResponse.DraftTreatmentPlans) != 1 {
		t.Fatalf("Expected 1 treamtent plan in draft instead got %d", len(treatmentPlanResponse.DraftTreatmentPlans))
	} else if len(treatmentPlanResponse.ActiveTreatmentPlans) != 0 {
		t.Fatalf("Expected no treatment plan in active mode instead got %d", len(treatmentPlanResponse.ActiveTreatmentPlans))
	}

	test_integration.SubmitPatientVisitBackToPatient(patientVisitResponse.PatientVisitId, doctor, testData, t)

	// now get a list of treatment plans for a patient
	treatmentPlanResponse = test_integration.GetListOfTreatmentPlansForPatient(patientId, doctor.AccountId.Int64(), testData, t)

	// there should be 1 active treatment plan for this patient
	if len(treatmentPlanResponse.ActiveTreatmentPlans) != 1 {
		t.Fatalf("Expected 1 active treatment plan but got %d", len(treatmentPlanResponse.ActiveTreatmentPlans))
	} else if len(treatmentPlanResponse.DraftTreatmentPlans) != 0 {
		t.Fatalf("Expected no treatment plans in draft mode instead got %d", len(treatmentPlanResponse.DraftTreatmentPlans))
	}
}

func TestTreatmentPlanList_DraftTest(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)
	doctorId := test_integration.GetDoctorIdOfCurrentPrimaryDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal("Unable to get doctor from doctor id " + err.Error())
	}

	patientVisitResponse, _ := test_integration.SignupAndSubmitPatientVisitForRandomPatient(t, testData, doctor)

	patientId, err := testData.DataApi.GetPatientIdFromPatientVisitId(patientVisitResponse.PatientVisitId)
	if err != nil {
		t.Fatalf(err.Error())
	}

	signedUpDoctorResponse, _, _ := test_integration.SignupRandomTestDoctor(t, testData.DataApi, testData.AuthApi)

	doctor2, err := testData.DataApi.GetDoctorFromId(signedUpDoctorResponse.DoctorId)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// doctor2 should not be able to see previous doctor's draft
	treatmentPlanResponse := test_integration.GetListOfTreatmentPlansForPatient(patientId, doctor2.AccountId.Int64(), testData, t)
	if len(treatmentPlanResponse.DraftTreatmentPlans) != 0 {
		t.Fatalf("Expected no treatment plans instead got %d", len(treatmentPlanResponse.DraftTreatmentPlans))
	} else if len(treatmentPlanResponse.ActiveTreatmentPlans) != 0 {
		t.Fatalf("Expected no treatment plans instead got %d", len(treatmentPlanResponse.ActiveTreatmentPlans))
	}

	test_integration.SubmitPatientVisitBackToPatient(patientVisitResponse.PatientVisitId, doctor, testData, t)

	// now doctor2 should be able to see the treatment plan that doctor1 just submitted
	treatmentPlanResponse = test_integration.GetListOfTreatmentPlansForPatient(patientId, doctor2.AccountId.Int64(), testData, t)
	if len(treatmentPlanResponse.DraftTreatmentPlans) != 0 {
		t.Fatalf("Expected no treatment plans instead got %d", len(treatmentPlanResponse.DraftTreatmentPlans))
	} else if len(treatmentPlanResponse.ActiveTreatmentPlans) != 1 {
		t.Fatalf("Expected 1 treatment plan instead got %d", len(treatmentPlanResponse.ActiveTreatmentPlans))
	}
}
