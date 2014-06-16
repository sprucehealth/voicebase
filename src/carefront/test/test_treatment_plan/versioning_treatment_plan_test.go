package test_treatment_plan

import (
	"carefront/common"
	"carefront/test/test_integration"
	"testing"
)

func TestVersionTreatmentPlan_NewTP(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)
	doctorId := test_integration.GetDoctorIdOfCurrentPrimaryDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal("Unable to get doctor from doctor id " + err.Error())
	}

	// get patient to start a visit and doctor to pick treatment plan
	patientVisitResponse, treatmentPlan := test_integration.SignupAndSubmitPatientVisitForRandomPatient(t, testData, doctor)

	patientId, err := testData.DataApi.GetPatientIdFromPatientVisitId(patientVisitResponse.PatientVisitId)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// submit treatment plan
	test_integration.SubmitPatientVisitBackToPatient(treatmentPlan.Id.Int64(), doctor, testData, t)

	// now try to start a new treatment plan that is a version of the previous one
	tpResponse := test_integration.PickATreatmentPlan(&common.TreatmentPlanParent{
		ParentId:   treatmentPlan.Id,
		ParentType: common.TPParentTypeTreatmentPlan,
	}, nil, doctor, testData, t)

	if tpResponse.TreatmentPlan.Id.Int64() == treatmentPlan.Id.Int64() {
		t.Fatal("Expected treatment plan to be different given that it was just versioned")
	}

	currentTreatmentPlan, err := testData.DataApi.GetTreatmentPlan(tpResponse.TreatmentPlan.Id.Int64(), doctorId)
	if err != nil {
		t.Fatal(err)
	}

	// the first treatment plan should be the parent of this treatment plan
	if currentTreatmentPlan.Parent.ParentType != common.TPParentTypeTreatmentPlan ||
		currentTreatmentPlan.Parent.ParentId.Int64() != treatmentPlan.Id.Int64() {
		t.Fatalf("expected treatment plan id %d to be the parent of treatment plan id %d but it wasnt", treatmentPlan.Id.Int64(), currentTreatmentPlan.Id.Int64())
	}

	// there should be no content source for this treatment plan
	if currentTreatmentPlan.ContentSource != nil {
		t.Fatal("Expected no content source for this treatment plan")
	}

	// there should be no treatments, regimen or advice
	if len(currentTreatmentPlan.TreatmentList.Treatments) > 0 {
		t.Fatalf("Expected no treatments isntead got %d", len(currentTreatmentPlan.TreatmentList.Treatments))
	} else if len(currentTreatmentPlan.RegimenPlan.RegimenSections) > 0 {
		t.Fatalf("Expected no regimen sections instead got %d", len(currentTreatmentPlan.RegimenPlan.RegimenSections))
	} else if len(currentTreatmentPlan.Advice.SelectedAdvicePoints) > 0 {
		t.Fatalf("Expected no advice points instead got %d", len(currentTreatmentPlan.Advice.SelectedAdvicePoints))
	}

	// should get back 1 treatment plan in draft and the other one active
	treatmentPlanResponse := test_integration.GetListOfTreatmentPlansForPatient(patientId, doctor.AccountId.Int64(), testData, t)
	if len(treatmentPlanResponse.DraftTreatmentPlans) != 1 {
		t.Fatalf("Expected 1 treamtent plan in draft instead got %d", len(treatmentPlanResponse.DraftTreatmentPlans))
	} else if len(treatmentPlanResponse.ActiveTreatmentPlans) != 1 {
		t.Fatalf("Expected 1 treatment plan in active mode instead got %d", len(treatmentPlanResponse.ActiveTreatmentPlans))
	}

	// now go ahead and submit the treatment plan
	test_integration.SubmitPatientVisitBackToPatient(currentTreatmentPlan.Id.Int64(), doctor, testData, t)

	// the new versioned treatment plan should be active and the previous one inactice
	treatmentPlanResponse = test_integration.GetListOfTreatmentPlansForPatient(patientId, doctor.AccountId.Int64(), testData, t)
	if len(treatmentPlanResponse.DraftTreatmentPlans) != 0 {
		t.Fatalf("Expected 0 treamtent plans in draft instead got %d", len(treatmentPlanResponse.DraftTreatmentPlans))
	} else if len(treatmentPlanResponse.ActiveTreatmentPlans) != 1 {
		t.Fatalf("Expected 1 treatment plan in active mode instead got %d", len(treatmentPlanResponse.ActiveTreatmentPlans))
	} else if treatmentPlanResponse.ActiveTreatmentPlans[0].Id.Int64() != currentTreatmentPlan.Id.Int64() {
		t.Fatalf("Expected treatment plan id %d instead got %d", currentTreatmentPlan.Id.Int64(), treatmentPlanResponse.ActiveTreatmentPlans[0].Id.Int64())
	} else if len(treatmentPlanResponse.InactiveTreatmentPlans) != 1 {
		t.Fatalf("Expected 1 inactive treatment plan instead got %d", len(treatmentPlanResponse.InactiveTreatmentPlans))
	} else if treatmentPlanResponse.InactiveTreatmentPlans[0].Id.Int64() != treatmentPlan.Id.Int64() {
		t.Fatalf("Expected inactive treatment plan to be %d instead it was %d", treatmentPlan.Id.Int64(), treatmentPlanResponse.InactiveTreatmentPlans[0].Id.Int64())
	}

}
