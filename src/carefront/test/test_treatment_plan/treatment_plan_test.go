package test_treatment_plan

import (
	"bytes"
	"carefront/api"
	"carefront/apiservice"
	"carefront/doctor_treatment_plan"
	"carefront/test/test_integration"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTreatmentPlanStatus(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)
	doctorId := test_integration.GetDoctorIdOfCurrentDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal("Unable to get doctor from doctor id " + err.Error())
	}

	// get patient to start a visit
	_, treatmentPlan := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	// this treatment plan should be in draft mode
	if treatmentPlan.Status != api.STATUS_DRAFT {
		t.Fatalf("Expected treatmentPlan status to be %s but it was %s instead", api.STATUS_DRAFT, treatmentPlan.Status)
	}

	// once the doctor submits it it should become ACTIVE
	test_integration.SubmitPatientVisitBackToPatient(treatmentPlan.Id.Int64(), doctor, testData, t)

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
	doctorId := test_integration.GetDoctorIdOfCurrentDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal("Unable to get doctor from doctor id " + err.Error())
	}

	patientVisitResponse, treatmentPlan := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

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

	test_integration.SubmitPatientVisitBackToPatient(treatmentPlan.Id.Int64(), doctor, testData, t)

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
	doctorId := test_integration.GetDoctorIdOfCurrentDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal("Unable to get doctor from doctor id " + err.Error())
	}

	patientVisitResponse, treatmentPlan := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	patientId, err := testData.DataApi.GetPatientIdFromPatientVisitId(patientVisitResponse.PatientVisitId)
	if err != nil {
		t.Fatalf(err.Error())
	}

	signedUpDoctorResponse, _, _ := test_integration.SignupRandomTestDoctor(t, testData)

	doctor2, err := testData.DataApi.GetDoctorFromId(signedUpDoctorResponse.DoctorId)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// add doctor2 to the care team of the patient
	if err := testData.DataApi.AddDoctorToCareTeamForPatient(patientId, apiservice.HEALTH_CONDITION_ACNE_ID, doctor2.DoctorId.Int64()); err != nil {
		t.Fatal(err)
	}

	// doctor2 should not be able to see previous doctor's draft
	treatmentPlanResponse := test_integration.GetListOfTreatmentPlansForPatient(patientId, doctor2.AccountId.Int64(), testData, t)
	if len(treatmentPlanResponse.DraftTreatmentPlans) != 0 {
		t.Fatalf("Expected no treatment plans instead got %d", len(treatmentPlanResponse.DraftTreatmentPlans))
	} else if len(treatmentPlanResponse.ActiveTreatmentPlans) != 0 {
		t.Fatalf("Expected no treatment plans instead got %d", len(treatmentPlanResponse.ActiveTreatmentPlans))
	}

	test_integration.SubmitPatientVisitBackToPatient(treatmentPlan.Id.Int64(), doctor, testData, t)

	// now doctor2 should be able to see the treatment plan that doctor1 just submitted
	treatmentPlanResponse = test_integration.GetListOfTreatmentPlansForPatient(patientId, doctor2.AccountId.Int64(), testData, t)
	if len(treatmentPlanResponse.DraftTreatmentPlans) != 0 {
		t.Fatalf("Expected no treatment plans instead got %d", len(treatmentPlanResponse.DraftTreatmentPlans))
	} else if len(treatmentPlanResponse.ActiveTreatmentPlans) != 1 {
		t.Fatalf("Expected 1 treatment plan instead got %d", len(treatmentPlanResponse.ActiveTreatmentPlans))
	}
}

func TestTreatmentPlanList_FavTP(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)
	doctorId := test_integration.GetDoctorIdOfCurrentDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal("Unable to get doctor from doctor id " + err.Error())
	}

	patientVisitResponse, treatmentPlan := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	patientId, err := testData.DataApi.GetPatientIdFromPatientVisitId(patientVisitResponse.PatientVisitId)
	if err != nil {
		t.Fatalf(err.Error())
	}

	favoriteTreatmentPlan := test_integration.CreateFavoriteTreatmentPlan(patientVisitResponse.PatientVisitId, treatmentPlan.Id.Int64(), testData, doctor, t)
	responseData := test_integration.PickATreatmentPlanForPatientVisit(patientVisitResponse.PatientVisitId, doctor, favoriteTreatmentPlan, testData, t)

	// favorite treatment plan information should be included in the list of treatment plans
	treatmentPlanResponse := test_integration.GetListOfTreatmentPlansForPatient(patientId, doctor.AccountId.Int64(), testData, t)
	if len(treatmentPlanResponse.DraftTreatmentPlans) != 1 {
		t.Fatalf("Expected 1 treatment plan instead got %d", len(treatmentPlanResponse.DraftTreatmentPlans))
	} else if len(treatmentPlanResponse.ActiveTreatmentPlans) != 0 {
		t.Fatalf("Expected no treatment plans instead got %d", len(treatmentPlanResponse.ActiveTreatmentPlans))
	}

	// now lets attempt to get this treatment plan by id to ensure that its linked to favorite treatment plan
	drTreatmentPlan := test_integration.GetDoctorTreatmentPlanById(treatmentPlanResponse.DraftTreatmentPlans[0].Id.Int64(), doctor.AccountId.Int64(), testData, t)
	if drTreatmentPlan.ContentSource == nil || drTreatmentPlan.ContentSource.ContentSourceId.Int64() == 0 {
		t.Fatalf("Expected link to favorite treatment plan to exist but it doesnt")
	} else if drTreatmentPlan.ContentSource.ContentSourceId.Int64() != favoriteTreatmentPlan.Id.Int64() {
		t.Fatalf("Expected treatment plan to be linked to fav treatment plan id %d but instead it ewas linked to id %d", favoriteTreatmentPlan.Id.Int64(), drTreatmentPlan.ContentSource.ContentSourceId.Int64())
	}

	// lets submit the treatment plan back to patient so that we can test whether or not favorite tretment plan information is shown to another doctor
	// it shouldn't be
	test_integration.SubmitPatientVisitBackToPatient(responseData.TreatmentPlan.Id.Int64(), doctor, testData, t)

	signedUpDoctorResponse, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor2, err := testData.DataApi.GetDoctorFromId(signedUpDoctorResponse.DoctorId)
	if err != nil {
		t.Fatalf(err.Error())
	}

	patientCase, err := testData.DataApi.GetPatientCaseFromPatientVisitId(patientVisitResponse.PatientVisitId)
	if err != nil {
		t.Fatal(err)
	}

	// assign the doctor to the patient case
	if err := testData.DataApi.AssignDoctorToPatientFileAndCase(doctor2.DoctorId.Int64(),
		patientCase); err != nil {
		t.Fatal(err)
	}

	drTreatmentPlan = test_integration.GetDoctorTreatmentPlanById(treatmentPlanResponse.DraftTreatmentPlans[0].Id.Int64(), doctor2.AccountId.Int64(), testData, t)
	if drTreatmentPlan.ContentSource != nil && drTreatmentPlan.ContentSource.ContentSourceId.Int64() != 0 {
		t.Fatalf("Expected content source to indicate that treatment plan deviated from original content source but it doesnt")
	}
}

func TestTreatmentPlanDelete(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)
	doctorId := test_integration.GetDoctorIdOfCurrentDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal("Unable to get doctor from doctor id " + err.Error())
	}

	patientVisitResponse, treatmentPlan := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	patientId, err := testData.DataApi.GetPatientIdFromPatientVisitId(patientVisitResponse.PatientVisitId)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// should be able to delete this treatment plan owned by doctor
	test_integration.DeleteTreatmentPlanForDoctor(treatmentPlan.Id.Int64(), doctor.AccountId.Int64(), testData, t)

	// there should be no drafts left given that we just deleted it
	treatmentPlanResponse := test_integration.GetListOfTreatmentPlansForPatient(patientId, doctor.AccountId.Int64(), testData, t)
	if len(treatmentPlanResponse.DraftTreatmentPlans) != 0 {
		t.Fatalf("Expected no treatment plans instead got %d", len(treatmentPlanResponse.DraftTreatmentPlans))
	} else if len(treatmentPlanResponse.ActiveTreatmentPlans) != 0 {
		t.Fatalf("Expected no treatment plans instead got %d", len(treatmentPlanResponse.ActiveTreatmentPlans))
	}
}

func TestTreatmentPlanDelete_ActiveTP(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)
	doctorId := test_integration.GetDoctorIdOfCurrentDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal("Unable to get doctor from doctor id " + err.Error())
	}

	patientVisitResponse, treatmentPlan := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	patientId, err := testData.DataApi.GetPatientIdFromPatientVisitId(patientVisitResponse.PatientVisitId)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// submit treatment plan to patient to make it active
	test_integration.SubmitPatientVisitBackToPatient(treatmentPlan.Id.Int64(), doctor, testData, t)

	// attempting to delete the treatment plan should fail given that the treatment plan is active
	doctorTreatmentPlanHandler := doctor_treatment_plan.NewDoctorTreatmentPlanHandler(testData.DataApi, nil, nil, false)
	doctorServer := httptest.NewServer(doctorTreatmentPlanHandler)
	defer doctorServer.Close()

	jsonData, err := json.Marshal(&doctor_treatment_plan.TreatmentPlanRequestData{
		TreatmentPlanId: treatmentPlan.Id,
	})

	res, err := testData.AuthDelete(doctorServer.URL, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal(err)
	} else if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected %d instead got %d", http.StatusBadRequest, res.StatusCode)
	}

	// there should still exist an active treatment plan
	treatmentPlanResponse := test_integration.GetListOfTreatmentPlansForPatient(patientId, doctor.AccountId.Int64(), testData, t)
	if len(treatmentPlanResponse.DraftTreatmentPlans) != 0 {
		t.Fatalf("Expected no treatment plans instead got %d", len(treatmentPlanResponse.DraftTreatmentPlans))
	} else if len(treatmentPlanResponse.ActiveTreatmentPlans) != 1 {
		t.Fatalf("Expected 1 treatment plan instead got %d", len(treatmentPlanResponse.ActiveTreatmentPlans))
	}
}

func TestTreatmentPlanDelete_DifferentDoctor(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)
	doctorId := test_integration.GetDoctorIdOfCurrentDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal("Unable to get doctor from doctor id " + err.Error())
	}

	patientVisitResponse, treatmentPlan := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	patientId, err := testData.DataApi.GetPatientIdFromPatientVisitId(patientVisitResponse.PatientVisitId)
	if err != nil {
		t.Fatalf(err.Error())
	}

	signedUpDoctorResponse, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor2, err := testData.DataApi.GetDoctorFromId(signedUpDoctorResponse.DoctorId)
	if err != nil {
		t.Fatal(err)
	}

	// attempting to delete the treatment plan should fail given that the treatment plan is active
	doctorTreatmentPlanHandler := doctor_treatment_plan.NewDoctorTreatmentPlanHandler(testData.DataApi, nil, nil, false)
	doctorServer := httptest.NewServer(doctorTreatmentPlanHandler)
	defer doctorServer.Close()

	jsonData, err := json.Marshal(&doctor_treatment_plan.TreatmentPlanRequestData{
		TreatmentPlanId: treatmentPlan.Id,
	})

	res, err := testData.AuthDelete(doctorServer.URL, "application/json", bytes.NewReader(jsonData), doctor2.AccountId.Int64())
	if err != nil {
		t.Fatal(err)
	} else if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected %d instead got %d", http.StatusBadRequest, res.StatusCode)
	}

	// there should still exist an active treatment plan
	treatmentPlanResponse := test_integration.GetListOfTreatmentPlansForPatient(patientId, doctor.AccountId.Int64(), testData, t)
	if len(treatmentPlanResponse.DraftTreatmentPlans) != 1 {
		t.Fatalf("Expected 1 treatment plan instead got %d", len(treatmentPlanResponse.DraftTreatmentPlans))
	} else if len(treatmentPlanResponse.ActiveTreatmentPlans) != 0 {
		t.Fatalf("Expected no treatment plans instead got %d", len(treatmentPlanResponse.ActiveTreatmentPlans))
	}
}
