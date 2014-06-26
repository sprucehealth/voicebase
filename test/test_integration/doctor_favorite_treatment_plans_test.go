package test_integration

import (
	"bytes"
	"carefront/api"
	"carefront/common"
	"carefront/doctor_treatment_plan"
	"carefront/encoding"
	"carefront/libs/erx"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
)

func TestFavoriteTreatmentPlan(t *testing.T) {
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	doctorId := GetDoctorIdOfCurrentDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatalf("Unable to get doctor from id: %s", err)
	}

	patientVisitResponse, treatmentPlan := CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	favoriteTreatmentPlan := CreateFavoriteTreatmentPlan(patientVisitResponse.PatientVisitId, treatmentPlan.Id.Int64(), testData, doctor, t)

	originalRegimenPlan := favoriteTreatmentPlan.RegimenPlan
	originalAdvice := favoriteTreatmentPlan.Advice

	// now lets go ahead and update the favorite treatment plan

	updatedName := "Updating name"
	favoriteTreatmentPlan.Name = updatedName
	favoriteTreatmentPlan.RegimenPlan.RegimenSections = favoriteTreatmentPlan.RegimenPlan.RegimenSections[1:]
	favoriteTreatmentPlan.Advice.SelectedAdvicePoints = favoriteTreatmentPlan.Advice.SelectedAdvicePoints[1:]

	requestData := &doctor_treatment_plan.DoctorFavoriteTreatmentPlansRequestData{}
	requestData.FavoriteTreatmentPlan = favoriteTreatmentPlan
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		t.Fatalf("Unable to marshal json data: %s", err)
	}

	ts := httptest.NewServer(doctor_treatment_plan.NewDoctorFavoriteTreatmentPlansHandler(testData.DataApi))
	defer ts.Close()

	responseData := &doctor_treatment_plan.DoctorFavoriteTreatmentPlansResponseData{}
	resp, err := testData.AuthPut(ts.URL, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	if err != nil {
		t.Fatalf("Unable to make call to update favorite treatment plan %s", err)
	} else if err := json.NewDecoder(resp.Body).Decode(responseData); err != nil {
		t.Fatalf("Unable to decode response body into json object %s", err)
	} else if responseData.FavoriteTreatmentPlan == nil {
		t.Fatalf("Expected 1 favorite treatment plan to be returned instead got back %d", len(responseData.FavoriteTreatmentPlans))
	} else if len(responseData.FavoriteTreatmentPlan.RegimenPlan.RegimenSections) != 1 {
		t.Fatalf("Expected 1 section in the regimen plan instead got %d", len(responseData.FavoriteTreatmentPlan.RegimenPlan.RegimenSections))
	} else if len(responseData.FavoriteTreatmentPlan.Advice.SelectedAdvicePoints) != 1 {
		t.Fatalf("Expected 1 section in the advice instead got %d", len(responseData.FavoriteTreatmentPlan.Advice.SelectedAdvicePoints))
	} else if responseData.FavoriteTreatmentPlan.Name != updatedName {
		t.Fatalf("Expected name of favorite treatment plan to be %s instead got %s", updatedName, responseData.FavoriteTreatmentPlan.Name)
	}

	CheckSuccessfulStatusCode(resp, "unable to make call to update favorite treatment plan", t)

	// lets go ahead and add another favorited treatment
	favoriteTreatmentPlan2 := &common.FavoriteTreatmentPlan{
		Name: "Test Treatment Plan #2",
		TreatmentList: &common.TreatmentList{
			Treatments: []*common.Treatment{&common.Treatment{
				DrugDBIds: map[string]string{
					erx.LexiDrugSynId:     "1234",
					erx.LexiGenProductId:  "12345",
					erx.LexiSynonymTypeId: "123556",
					erx.NDC:               "2415",
				},
				DrugName:                "Teting (This - Drug)",
				DosageStrength:          "10 mg",
				DispenseValue:           5,
				DispenseUnitDescription: "Tablet",
				DispenseUnitId:          encoding.NewObjectId(19),
				NumberRefills: encoding.NullInt64{
					IsValid:    true,
					Int64Value: 5,
				},
				SubstitutionsAllowed: false,
				DaysSupply: encoding.NullInt64{
					IsValid:    true,
					Int64Value: 5,
				},
				PatientInstructions: "Take once daily",
				OTC:                 false,
			},
			},
		},
		RegimenPlan: originalRegimenPlan,
		Advice:      originalAdvice,
	}

	requestData.FavoriteTreatmentPlan = favoriteTreatmentPlan2
	jsonData, err = json.Marshal(requestData)
	if err != nil {
		t.Fatalf("Unable to marshal favorited treatment plan %s", err)
	}

	resp, err = testData.AuthPost(ts.URL, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	if err != nil {
		t.Fatalf("Unable to add another favorite treatment plan %s", err)
	}

	CheckSuccessfulStatusCode(resp, "unable to add another favorite treatment plan", t)

	resp, err = testData.AuthGet(ts.URL, doctor.AccountId.Int64())
	if err != nil {
		t.Fatalf("Unabke to get list of favorite treatment plans %s", err)
	} else if err := json.NewDecoder(resp.Body).Decode(responseData); err != nil {
		t.Fatalf("Unable to unmarshal response into a list of favorite treatment plans %s", err)
	} else if len(responseData.FavoriteTreatmentPlans) != 2 {
		t.Fatalf("Expected 2 favorite treatment plans instead got %d", len(responseData.FavoriteTreatmentPlans))
	} else if len(responseData.FavoriteTreatmentPlans[0].RegimenPlan.RegimenSections) != 1 {
		t.Fatalf("Expected favorite treatment plan to have 1 regimen section")
	} else if len(responseData.FavoriteTreatmentPlans[0].Advice.SelectedAdvicePoints) != 1 {
		t.Fatalf("Expected favorite treatment plan to have 1 advice point")
	} else if len(responseData.FavoriteTreatmentPlans[1].RegimenPlan.RegimenSections) != 1 {
		t.Fatalf("Expected favorite treatment plan to have 2 regimen sections")
	} else if len(responseData.FavoriteTreatmentPlans[1].Advice.SelectedAdvicePoints) != 1 {
		t.Fatalf("Expected favorite treatment plan to have 2 advice points")
	}

	CheckSuccessfulStatusCode(resp, "Unable to get list of favorite treatment plans for doctor", t)

	// lets go ahead and delete favorite treatment plan
	params := url.Values{}
	params.Set("favorite_treatment_plan_id", strconv.FormatInt(responseData.FavoriteTreatmentPlans[0].Id.Int64(), 10))
	resp, err = testData.AuthDelete(ts.URL+"?"+params.Encode(), "application/x-www-form-urlencoded", nil, doctor.AccountId.Int64())
	if err != nil {
		t.Fatalf("Unable to delete favorite treatment plan %s", err)
	}

	CheckSuccessfulStatusCode(resp, "Unable to delete favorite treatment plan", t)
}

// This test ensures to check that after deleting a FTP, the TP that was created
// from the FTP has its content source deleted and getting the TP still works
func TestFavoriteTreatmentPlan_DeletingFTP(t *testing.T) {
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	doctorId := GetDoctorIdOfCurrentDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatalf("Unable to get doctor from id: %s", err)
	}

	patientVisitResponse, treatmentPlan := CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	favoriteTreatmentPlan := CreateFavoriteTreatmentPlan(patientVisitResponse.PatientVisitId, treatmentPlan.Id.Int64(), testData, doctor, t)

	// lets start a new TP based on FTP
	responseData := PickATreatmentPlan(&common.TreatmentPlanParent{
		ParentType: common.TPParentTypePatientVisit,
		ParentId:   encoding.NewObjectId(patientVisitResponse.PatientVisitId),
	}, &common.TreatmentPlanContentSource{
		ContentSourceType: common.TPContentSourceTypeFTP,
		ContentSourceId:   favoriteTreatmentPlan.Id,
	}, doctor, testData, t)

	// ensure that this TP has the FTP as its content source
	if responseData.TreatmentPlan.ContentSource == nil ||
		responseData.TreatmentPlan.ContentSource.ContentSourceType != common.TPContentSourceTypeFTP ||
		responseData.TreatmentPlan.ContentSource.ContentSourceId.Int64() != favoriteTreatmentPlan.Id.Int64() {
		t.Fatal("Expected the newly created Treatment plan to have the FTP as its source")
	}

	// now lets go ahead and delete the FTP
	ts := httptest.NewServer(doctor_treatment_plan.NewDoctorFavoriteTreatmentPlansHandler(testData.DataApi))
	defer ts.Close()

	params := url.Values{}
	params.Set("favorite_treatment_plan_id", strconv.FormatInt(favoriteTreatmentPlan.Id.Int64(), 10))
	resp, err := testData.AuthDelete(ts.URL+"?"+params.Encode(), "application/x-www-form-urlencoded", nil, doctor.AccountId.Int64())
	if err != nil {
		t.Fatalf("Unable to delete favorite treatment plan %s", err)
	}

	// now if we try to get the TP initially created from the FTP, the content source should not exist
	doctorTreatmentPlanHandler := doctor_treatment_plan.NewDoctorTreatmentPlanHandler(testData.DataApi, nil, nil, false)
	doctorTPServer := httptest.NewServer(doctorTreatmentPlanHandler)
	defer doctorTPServer.Close()

	doctorTreatmentPlanResponse := doctor_treatment_plan.DoctorTreatmentPlanResponse{}
	resp, err = testData.AuthGet(doctorTPServer.URL+"?treatment_plan_id="+strconv.FormatInt(responseData.TreatmentPlan.Id.Int64(), 10), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal(err)
	} else if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code %d  but got %d", http.StatusOK, resp.StatusCode)
	} else if err := json.NewDecoder(resp.Body).Decode(&doctorTreatmentPlanResponse); err != nil {
		t.Fatal(err)
	} else if doctorTreatmentPlanResponse.TreatmentPlan.ContentSource != nil {
		t.Fatal("Expected nil content source for treatment plan after deleting FTP from which the TP was started")
	}
}

// This test ensures that even if an FTP is deleted that was picked as content source for a TP that has been activated for a patient,
// the content source gets deleted while TP remains unaltered
func TestFavoriteTreatmentPlan_DeletingFTP_ActiveTP(t *testing.T) {
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	doctorId := GetDoctorIdOfCurrentDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatalf("Unable to get doctor from id: %s", err)
	}

	patientVisitResponse, treatmentPlan := CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	favoriteTreatmentPlan := CreateFavoriteTreatmentPlan(patientVisitResponse.PatientVisitId, treatmentPlan.Id.Int64(), testData, doctor, t)

	// lets start a new TP based on FTP
	responseData := PickATreatmentPlan(&common.TreatmentPlanParent{
		ParentType: common.TPParentTypePatientVisit,
		ParentId:   encoding.NewObjectId(patientVisitResponse.PatientVisitId),
	}, &common.TreatmentPlanContentSource{
		ContentSourceType: common.TPContentSourceTypeFTP,
		ContentSourceId:   favoriteTreatmentPlan.Id,
	}, doctor, testData, t)

	// ensure that this TP has the FTP as its content source
	if responseData.TreatmentPlan.ContentSource == nil ||
		responseData.TreatmentPlan.ContentSource.ContentSourceType != common.TPContentSourceTypeFTP ||
		responseData.TreatmentPlan.ContentSource.ContentSourceId.Int64() != favoriteTreatmentPlan.Id.Int64() {
		t.Fatal("Expected the newly created Treatment plan to have the FTP as its source")
	}

	// submit the treatments for the TP
	AddAndGetTreatmentsForPatientVisit(testData, favoriteTreatmentPlan.TreatmentList.Treatments, doctor.AccountId.Int64(), responseData.TreatmentPlan.Id.Int64(), t)

	// submit regimen for TP
	regimenPlanRequest := &common.RegimenPlan{}
	regimenPlanRequest.TreatmentPlanId = responseData.TreatmentPlan.Id
	regimenPlanRequest.RegimenSections = favoriteTreatmentPlan.RegimenPlan.RegimenSections
	CreateRegimenPlanForTreatmentPlan(regimenPlanRequest, testData, doctor, t)

	// submit advice for TP
	doctorAdviceRequest := &common.Advice{}
	doctorAdviceRequest.SelectedAdvicePoints = favoriteTreatmentPlan.Advice.SelectedAdvicePoints
	doctorAdviceRequest.TreatmentPlanId = responseData.TreatmentPlan.Id
	UpdateAdvicePointsForPatientVisit(doctorAdviceRequest, testData, doctor, t)

	// submit treatment plan to patient
	SubmitPatientVisitBackToPatient(responseData.TreatmentPlan.Id.Int64(), doctor, testData, t)

	// now lets go ahead and delete the FTP
	ts := httptest.NewServer(doctor_treatment_plan.NewDoctorFavoriteTreatmentPlansHandler(testData.DataApi))
	defer ts.Close()

	params := url.Values{}
	params.Set("favorite_treatment_plan_id", strconv.FormatInt(favoriteTreatmentPlan.Id.Int64(), 10))
	resp, err := testData.AuthDelete(ts.URL+"?"+params.Encode(), "application/x-www-form-urlencoded", nil, doctor.AccountId.Int64())
	if err != nil {
		t.Fatalf("Unable to delete favorite treatment plan %s", err)
	}

	// now if we try to get the TP initially created from the FTP, the content source should not exist
	doctorTreatmentPlanHandler := doctor_treatment_plan.NewDoctorTreatmentPlanHandler(testData.DataApi, nil, nil, false)
	doctorTPServer := httptest.NewServer(doctorTreatmentPlanHandler)
	defer doctorTPServer.Close()

	doctorTreatmentPlanResponse := doctor_treatment_plan.DoctorTreatmentPlanResponse{}
	resp, err = testData.AuthGet(doctorTPServer.URL+"?treatment_plan_id="+strconv.FormatInt(responseData.TreatmentPlan.Id.Int64(), 10), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal(err)
	} else if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code %d  but got %d", http.StatusOK, resp.StatusCode)
	} else if err := json.NewDecoder(resp.Body).Decode(&doctorTreatmentPlanResponse); err != nil {
		t.Fatal(err)
	} else if doctorTreatmentPlanResponse.TreatmentPlan.ContentSource != nil {
		t.Fatal("Expected nil content source for treatment plan after deleting FTP from which the TP was started")
	} else if doctorTreatmentPlanResponse.TreatmentPlan.Status != api.STATUS_ACTIVE {
		t.Fatalf("Expected the treatment plan to be %s instead it was %s", api.STATUS_ACTIVE, doctorTreatmentPlanResponse.TreatmentPlan.Status)
	} else if !favoriteTreatmentPlan.EqualsDoctorTreatmentPlan(doctorTreatmentPlanResponse.TreatmentPlan) {
		t.Fatal("Even though the FTP was deleted, the contents of the TP and FTP should still match")
	}
}

func TestFavoriteTreatmentPlan_PickingAFavoriteTreatmentPlan(t *testing.T) {
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	doctorId := GetDoctorIdOfCurrentDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatalf("Unable to get doctor from id: %s", err)
	}

	patientVisitResponse, treatmentPlan := CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	// create a favorite treatment plan
	favoriteTreamentPlan := CreateFavoriteTreatmentPlan(patientVisitResponse.PatientVisitId, treatmentPlan.Id.Int64(), testData, doctor, t)

	// lets attempt to get the treatment plan for the patient visit
	// and ensure that its empty
	ts := httptest.NewServer(doctor_treatment_plan.NewDoctorTreatmentPlanHandler(testData.DataApi, nil, nil, false))
	defer ts.Close()

	responseData := &doctor_treatment_plan.DoctorTreatmentPlanResponse{}
	if resp, err := testData.AuthGet(ts.URL+"?treatment_plan_id="+strconv.FormatInt(treatmentPlan.Id.Int64(), 10), doctor.AccountId.Int64()); err != nil {
		t.Fatalf("Unable to make call to get treatment plan for patient visit")
	} else if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected %d response for getting treatment plan instead got %d", http.StatusOK, resp.StatusCode)
	} else if json.NewDecoder(resp.Body).Decode(responseData); err != nil {
		t.Fatalf("Unable to unmarshal response into struct %s", err)
	} else if responseData.TreatmentPlan == nil {
		t.Fatalf("Expected treatment plan to exist")
	} else if responseData.TreatmentPlan.TreatmentList != nil && len(responseData.TreatmentPlan.TreatmentList.Treatments) != 0 {
		t.Fatalf("Expected there to exist no treatments in treatment plan")
	} else if responseData.TreatmentPlan.RegimenPlan != nil && len(responseData.TreatmentPlan.RegimenPlan.RegimenSections) != 0 {
		t.Fatalf("Expected regimen to not exist for treatment plan instead we have %d regimen sections", len(responseData.TreatmentPlan.RegimenPlan.RegimenSections))
	} else if len(responseData.TreatmentPlan.RegimenPlan.AllRegimenSteps) == 0 {
		t.Fatalf("Expected regimen steps to exist given that they were created to create the treatment plan")
	} else if responseData.TreatmentPlan.Advice != nil && len(responseData.TreatmentPlan.Advice.SelectedAdvicePoints) != 0 {
		t.Fatalf("Expected there to exist no advice points for treatment plan")
	} else if len(responseData.TreatmentPlan.Advice.AllAdvicePoints) == 0 {
		t.Fatalf("Expected there to exist advice points given that some were created when creating the favorite treatment plan")
	}

	// now lets attempt to pick the added favorite treatment plan and compare the two again
	// this time the treatment plan should be populated with data from the favorite treatment plan
	responseData = PickATreatmentPlanForPatientVisit(patientVisitResponse.PatientVisitId, doctor, favoriteTreamentPlan, testData, t)
	if responseData.TreatmentPlan == nil {
		t.Fatalf("Expected treatment plan to exist")
	} else if responseData.TreatmentPlan.TreatmentList != nil && len(responseData.TreatmentPlan.TreatmentList.Treatments) != 1 {
		t.Fatalf("Expected there to exist no treatments in treatment plan")
	} else if responseData.TreatmentPlan.TreatmentList.Status != api.STATUS_UNCOMMITTED {
		t.Fatalf("Status should indicate UNCOMMITTED for treatment section when the doctor has not committed the section")
	} else if responseData.TreatmentPlan.RegimenPlan != nil && len(responseData.TreatmentPlan.RegimenPlan.RegimenSections) != 2 {
		t.Fatalf("Expected regimen to not exist for treatment plan instead we have %d regimen sections", len(responseData.TreatmentPlan.RegimenPlan.RegimenSections))
	} else if len(responseData.TreatmentPlan.RegimenPlan.AllRegimenSteps) != 2 {
		t.Fatalf("Expected there to exist 2 regimen steps in the master list instead got %d", len(responseData.TreatmentPlan.RegimenPlan.AllRegimenSteps))
	} else if responseData.TreatmentPlan.RegimenPlan.Status != api.STATUS_UNCOMMITTED {
		t.Fatalf("Status should indicate UNCOMMITTED for regimen plan when the doctor has not committed the section")
	} else if responseData.TreatmentPlan.Advice != nil && len(responseData.TreatmentPlan.Advice.SelectedAdvicePoints) != 2 {
		t.Fatalf("Expected there to exist no advice points for treatment plan")
	} else if len(responseData.TreatmentPlan.Advice.AllAdvicePoints) != 2 {
		t.Fatalf("Expected there to exist 2 advice points in the master list instead got %d", len(responseData.TreatmentPlan.Advice.AllAdvicePoints))
	} else if responseData.TreatmentPlan.Advice.Status != api.STATUS_UNCOMMITTED {
		t.Fatalf("Status should indicate UNCOMMITTED for advice when the doctor has not committed the section")
	} else if !favoriteTreamentPlan.EqualsDoctorTreatmentPlan(responseData.TreatmentPlan) {
		t.Fatal("Expected the contents of the favorite treatment plan to be the same as that of the treatment plan but its not")
	}

	var count int64
	if err := testData.DB.QueryRow(`select count(*) from treatment_plan inner join treatment_plan_patient_visit_mapping on treatment_plan_id = treatment_plan.id where patient_visit_id = ?`, patientVisitResponse.PatientVisitId).Scan(&count); err != nil {
		t.Fatalf("Unable to query database to get number of treatment plans for patient visit: %s", err)
	} else if count != 1 {
		t.Fatalf("Expected 1 treatment plan for patient visit instead got %d", count)
	}
}

func TestFavoriteTreatmentPlan_CommittedStateForTreatmentPlan(t *testing.T) {
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	doctorId := GetDoctorIdOfCurrentDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatalf("Unable to get doctor from id: %s", err)
	}

	patientVisitResponse, treatmentPlan := CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	// create a favorite treatment plan
	favoriteTreamentPlan := CreateFavoriteTreatmentPlan(patientVisitResponse.PatientVisitId, treatmentPlan.Id.Int64(), testData, doctor, t)

	// pick this favorite treatment plan for the visit
	responseData := PickATreatmentPlanForPatientVisit(patientVisitResponse.PatientVisitId, doctor, favoriteTreamentPlan, testData, t)
	treatmentPlanId := responseData.TreatmentPlan.Id.Int64()
	// lets attempt to submit regimen section for patient visit
	regimenPlanRequest := &common.RegimenPlan{
		TreatmentPlanId: responseData.TreatmentPlan.Id,
		AllRegimenSteps: favoriteTreamentPlan.RegimenPlan.AllRegimenSteps,
		RegimenSections: favoriteTreamentPlan.RegimenPlan.RegimenSections,
	}
	CreateRegimenPlanForTreatmentPlan(regimenPlanRequest, testData, doctor, t)

	// now lets attempt to get the treatment plan for the patient visit
	ts := httptest.NewServer(doctor_treatment_plan.NewDoctorTreatmentPlanHandler(testData.DataApi, nil, nil, false))
	defer ts.Close()

	// the regimen plan should indicate that it was committed while the rest of the sections
	// should continue to be in the UNCOMMITTED state
	responseData = &doctor_treatment_plan.DoctorTreatmentPlanResponse{}
	if resp, err := testData.AuthGet(ts.URL+"?treatment_plan_id="+strconv.FormatInt(treatmentPlanId, 10), doctor.AccountId.Int64()); err != nil {
		t.Fatalf("Unable to make call to get treatment plan for patient visit")
	} else if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected %d response for getting treatment plan instead got %d", http.StatusOK, resp.StatusCode)
	} else if json.NewDecoder(resp.Body).Decode(responseData); err != nil {
		t.Fatalf("Unable to unmarshal response into struct %s", err)
	} else if responseData.TreatmentPlan.TreatmentList.Status != api.STATUS_UNCOMMITTED {
		t.Fatalf("Expected the status to be UNCOMMITTED for treatments")
	} else if responseData.TreatmentPlan.RegimenPlan.Status != api.STATUS_COMMITTED {
		t.Fatalf("Expected regimen status to not be COMMITTED")
	} else if responseData.TreatmentPlan.Advice.Status != api.STATUS_UNCOMMITTED {
		t.Fatalf("Expected the advice status to be UNCOMMITTED")
	}

	// now lets go ahead and submit the advice section
	doctorAdviceRequest := &common.Advice{
		TreatmentPlanId:      encoding.NewObjectId(treatmentPlanId),
		AllAdvicePoints:      favoriteTreamentPlan.Advice.AllAdvicePoints,
		SelectedAdvicePoints: favoriteTreamentPlan.Advice.SelectedAdvicePoints,
	}
	UpdateAdvicePointsForPatientVisit(doctorAdviceRequest, testData, doctor, t)

	// now if we were to get the treatment plan again it should indicate that the
	// advice and regimen sections are committed but not the treatment section
	responseData = &doctor_treatment_plan.DoctorTreatmentPlanResponse{}
	if resp, err := testData.AuthGet(ts.URL+"?treatment_plan_id="+strconv.FormatInt(treatmentPlanId, 10), doctor.AccountId.Int64()); err != nil {
		t.Fatalf("Unable to make call to get treatment plan for patient visit")
	} else if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected %d response for getting treatment plan instead got %d", http.StatusOK, resp.StatusCode)
	} else if json.NewDecoder(resp.Body).Decode(responseData); err != nil {
		t.Fatalf("Unable to unmarshal response into struct %s", err)
	} else if responseData.TreatmentPlan.TreatmentList.Status != api.STATUS_UNCOMMITTED {
		t.Fatalf("Expected the status to be UNCOMMITTED for treatments")
	} else if responseData.TreatmentPlan.RegimenPlan.Status != api.STATUS_COMMITTED {
		t.Fatalf("Expected regimen status to be COMMITTED")
	} else if responseData.TreatmentPlan.Advice.Status != api.STATUS_COMMITTED {
		t.Fatalf("Expected the advice status to be COMMITTED")
	}

	// now lets go ahead and add a treatment to the treatment plan
	AddAndGetTreatmentsForPatientVisit(testData, favoriteTreamentPlan.TreatmentList.Treatments, doctor.AccountId.Int64(), treatmentPlanId, t)

	// now the treatment section should also indicate that it has been committed
	responseData = &doctor_treatment_plan.DoctorTreatmentPlanResponse{}
	if resp, err := testData.AuthGet(ts.URL+"?treatment_plan_id="+strconv.FormatInt(treatmentPlanId, 10), doctor.AccountId.Int64()); err != nil {
		t.Fatalf("Unable to make call to get treatment plan for patient visit")
	} else if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected %d response for getting treatment plan instead got %d", http.StatusOK, resp.StatusCode)
	} else if json.NewDecoder(resp.Body).Decode(responseData); err != nil {
		t.Fatalf("Unable to unmarshal response into struct %s", err)
	} else if responseData.TreatmentPlan.TreatmentList.Status != api.STATUS_COMMITTED {
		t.Fatalf("Expected the status to be in the committed state")
	} else if responseData.TreatmentPlan.RegimenPlan.Status != api.STATUS_COMMITTED {
		t.Fatalf("Expected regimen status to be in the committed state")
	} else if responseData.TreatmentPlan.Advice.Status != api.STATUS_COMMITTED {
		t.Fatalf("Expected the advice status to be in the committed")
	}

}

func TestFavoriteTreatmentPlan_BreakingMappingOnModify(t *testing.T) {
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	doctorId := GetDoctorIdOfCurrentDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatalf("Unable to get doctor from id: %s", err)
	}

	patientVisitResponse, treatmentPlan := CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	// create a favorite treatment plan
	favoriteTreamentPlan := CreateFavoriteTreatmentPlan(patientVisitResponse.PatientVisitId, treatmentPlan.Id.Int64(), testData, doctor, t)

	// pick this favorite treatment plan for the visit
	responseData := PickATreatmentPlanForPatientVisit(patientVisitResponse.PatientVisitId, doctor, favoriteTreamentPlan, testData, t)

	// lets attempt to modify and submit regimen section for patient visit
	regimenPlanRequest := &common.RegimenPlan{
		TreatmentPlanId: responseData.TreatmentPlan.Id,
		AllRegimenSteps: favoriteTreamentPlan.RegimenPlan.AllRegimenSteps,
		RegimenSections: favoriteTreamentPlan.RegimenPlan.RegimenSections[:1],
	}
	CreateRegimenPlanForTreatmentPlan(regimenPlanRequest, testData, doctor, t)

	// now lets attempt to get the abbreviated version of the treatment plan
	ts := httptest.NewServer(doctor_treatment_plan.NewDoctorTreatmentPlanHandler(testData.DataApi, nil, nil, false))
	defer ts.Close()

	// the regimen plan should indicate that it was committed while the rest of the sections
	// should continue to be in the UNCOMMITTED state
	params := url.Values{}
	params.Set("treatment_plan_id", strconv.FormatInt(responseData.TreatmentPlan.Id.Int64(), 10))
	params.Set("abridged", "true")
	responseData = &doctor_treatment_plan.DoctorTreatmentPlanResponse{}
	if resp, err := testData.AuthGet(ts.URL+"?"+params.Encode(), doctor.AccountId.Int64()); err != nil {
		t.Fatalf("Unable to make call to get treatment plan for patient visit")
	} else if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected %d response for getting treatment plan instead got %d", http.StatusOK, resp.StatusCode)
	} else if json.NewDecoder(resp.Body).Decode(responseData); err != nil {
		t.Fatalf("Unable to unmarshal response into struct %s", err)
	} else if responseData.TreatmentPlan.ContentSource == nil || responseData.TreatmentPlan.ContentSource.ContentSourceType != common.TPContentSourceTypeFTP ||
		responseData.TreatmentPlan.ContentSource.ContentSourceId.Int64() == 0 || !responseData.TreatmentPlan.ContentSource.HasDeviated {
		t.Fatalf("Expected the treatment plan to indicate that it has deviated from the original content source (ftp) but it doesnt do so")
	}

	// lets try modfying treatments on a new treatment plan picked from favorites
	responseData = PickATreatmentPlanForPatientVisit(patientVisitResponse.PatientVisitId, doctor, favoriteTreamentPlan, testData, t)

	// lets make sure linkage exists
	if responseData.TreatmentPlan.ContentSource == nil || responseData.TreatmentPlan.ContentSource.ContentSourceType != common.TPContentSourceTypeFTP ||
		responseData.TreatmentPlan.ContentSource.ContentSourceId.Int64() == 0 {
		t.Fatalf("Expected the treatment plan to come from a favorite treatment plan")
	} else if responseData.TreatmentPlan.ContentSource.ContentSourceId.Int64() != favoriteTreamentPlan.Id.Int64() {
		t.Fatalf("Got a different favorite treatment plan linking to the treatment plan. Expected %d got %d", favoriteTreamentPlan.Id.Int64(), responseData.TreatmentPlan.ContentSource.ContentSourceId.Int64())
	}

	// modify treatment
	favoriteTreamentPlan.TreatmentList.Treatments[0].DispenseValue = encoding.HighPrecisionFloat64(123.12345)
	AddAndGetTreatmentsForPatientVisit(testData, favoriteTreamentPlan.TreatmentList.Treatments, doctor.AccountId.Int64(), responseData.TreatmentPlan.Id.Int64(), t)

	// linkage should now be broken
	params.Set("treatment_plan_id", strconv.FormatInt(responseData.TreatmentPlan.Id.Int64(), 10))
	if resp, err := testData.AuthGet(ts.URL+"?"+params.Encode(), doctor.AccountId.Int64()); err != nil {
		t.Fatalf("Unable to make call to get treatment plan for patient visit")
	} else if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected %d response for getting treatment plan instead got %d", http.StatusOK, resp.StatusCode)
	} else if json.NewDecoder(resp.Body).Decode(responseData); err != nil {
		t.Fatalf("Unable to unmarshal response into struct %s", err)
	} else if responseData.TreatmentPlan.ContentSource == nil || responseData.TreatmentPlan.ContentSource.ContentSourceType != common.TPContentSourceTypeFTP ||
		responseData.TreatmentPlan.ContentSource.ContentSourceId.Int64() == 0 || !responseData.TreatmentPlan.ContentSource.HasDeviated {
		t.Fatalf("Expected the treatment plan to indicate that it has deviated from the original content source (ftp) but it doesnt do so")
	}

	// lets try modifying advice
	responseData = PickATreatmentPlanForPatientVisit(patientVisitResponse.PatientVisitId, doctor, favoriteTreamentPlan, testData, t)

	// lets make sure linkage exists
	if responseData.TreatmentPlan.ContentSource == nil || responseData.TreatmentPlan.ContentSource.ContentSourceId.Int64() == 0 || responseData.TreatmentPlan.ContentSource.ContentSourceType != common.TPContentSourceTypeFTP {
		t.Fatalf("Expected the treatment plan to come from a favorite treatment plan")
	}

	// modify advice
	doctorAdviceRequest := &common.Advice{
		TreatmentPlanId:      responseData.TreatmentPlan.Id,
		AllAdvicePoints:      favoriteTreamentPlan.Advice.AllAdvicePoints,
		SelectedAdvicePoints: favoriteTreamentPlan.Advice.SelectedAdvicePoints[1:],
	}
	UpdateAdvicePointsForPatientVisit(doctorAdviceRequest, testData, doctor, t)

	// linkage should now be broken
	params.Set("treatment_plan_id", strconv.FormatInt(responseData.TreatmentPlan.Id.Int64(), 10))
	if resp, err := testData.AuthGet(ts.URL+"?"+params.Encode(), doctor.AccountId.Int64()); err != nil {
		t.Fatalf("Unable to make call to get treatment plan for patient visit")
	} else if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected %d response for getting treatment plan instead got %d", http.StatusOK, resp.StatusCode)
	} else if json.NewDecoder(resp.Body).Decode(responseData); err != nil {
		t.Fatalf("Unable to unmarshal response into struct %s", err)
	} else if responseData.TreatmentPlan.ContentSource == nil || responseData.TreatmentPlan.ContentSource.ContentSourceType != common.TPContentSourceTypeFTP ||
		responseData.TreatmentPlan.ContentSource.ContentSourceId.Int64() == 0 || !responseData.TreatmentPlan.ContentSource.HasDeviated {
		t.Fatalf("Expected the treatment plan to indicate that it has deviated from the original content source (ftp) but it doesnt do so")
	}
}

// This test is to cover the scenario where if a doctor modifies,say, the treatment section after
// starting from a favorite treatment plan, we ensure that the rest of the sections are still prefilled
// with the contents of the favorite treatment plan
func TestFavoriteTreatmentPlan_BreakingMappingOnModify_PrefillRestOfData(t *testing.T) {
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	doctorId := GetDoctorIdOfCurrentDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatalf("Unable to get doctor from id: %s", err)
	}

	patientVisitResponse, treatmentPlan := CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	// create a favorite treatment plan
	favoriteTreamentPlan := CreateFavoriteTreatmentPlan(patientVisitResponse.PatientVisitId, treatmentPlan.Id.Int64(), testData, doctor, t)

	// pick this favorite treatment plan for the visit
	responseData := PickATreatmentPlanForPatientVisit(patientVisitResponse.PatientVisitId, doctor, favoriteTreamentPlan, testData, t)

	// modify treatment
	favoriteTreamentPlan.TreatmentList.Treatments[0].DispenseValue = encoding.HighPrecisionFloat64(123.12345)
	AddAndGetTreatmentsForPatientVisit(testData, favoriteTreamentPlan.TreatmentList.Treatments, doctor.AccountId.Int64(), responseData.TreatmentPlan.Id.Int64(), t)

	// now lets attempt to get the abbreviated version of the treatment plan
	ts := httptest.NewServer(doctor_treatment_plan.NewDoctorTreatmentPlanHandler(testData.DataApi, nil, nil, false))
	defer ts.Close()

	params := url.Values{}
	params.Set("treatment_plan_id", strconv.FormatInt(responseData.TreatmentPlan.Id.Int64(), 10))
	responseData = &doctor_treatment_plan.DoctorTreatmentPlanResponse{}
	if resp, err := testData.AuthGet(ts.URL+"?"+params.Encode(), doctor.AccountId.Int64()); err != nil {
		t.Fatalf("Unable to make call to get treatment plan for patient visit")
	} else if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected %d response for getting treatment plan instead got %d", http.StatusOK, resp.StatusCode)
	} else if json.NewDecoder(resp.Body).Decode(responseData); err != nil {
		t.Fatalf("Unable to unmarshal response into struct %s", err)
	}

	// the treatments should be in the committed state while the regimen and advice should still be prefilled in the uncommitted state
	if responseData.TreatmentPlan.TreatmentList == nil || len(responseData.TreatmentPlan.TreatmentList.Treatments) == 0 || responseData.TreatmentPlan.TreatmentList.Status != api.STATUS_COMMITTED {
		t.Fatal("Expected treatments to exist and be in COMMITTED state")
	} else if responseData.TreatmentPlan.RegimenPlan == nil || len(responseData.TreatmentPlan.RegimenPlan.RegimenSections) == 0 || responseData.TreatmentPlan.RegimenPlan.Status != api.STATUS_UNCOMMITTED {
		t.Fatal("Expected regimen plan to be prefilled with FTP and be in UNCOMMITTED state")
	} else if responseData.TreatmentPlan.Advice == nil || len(responseData.TreatmentPlan.Advice.SelectedAdvicePoints) == 0 || responseData.TreatmentPlan.Advice.Status != api.STATUS_UNCOMMITTED {
		t.Fatal("Expected advice section to be prefilled with FTP and be in UNCOMMITTED state")
	}
}

// This test ensures that the user can create a favorite treatment plan
// in the context of treatment plan by specifying the treatment plan to associate the
// favorite treatment plan with
func TestFavoriteTreatmentPlan_InContextOfTreatmentPlan(t *testing.T) {
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	doctorId := GetDoctorIdOfCurrentDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatalf("Unable to get doctor from id: %s", err)
	}

	_, treatmentPlan := CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	regimenPlanRequest := &common.RegimenPlan{}
	regimenPlanRequest.TreatmentPlanId = treatmentPlan.Id

	regimenStep1 := &common.DoctorInstructionItem{}
	regimenStep1.Text = "Regimen Step 1"
	regimenStep1.State = common.STATE_ADDED

	regimenStep2 := &common.DoctorInstructionItem{}
	regimenStep2.Text = "Regimen Step 2"
	regimenStep2.State = common.STATE_ADDED

	regimenSection := &common.RegimenSection{}
	regimenSection.RegimenName = "morning"
	regimenSection.RegimenSteps = []*common.DoctorInstructionItem{&common.DoctorInstructionItem{
		Text:  regimenStep1.Text,
		State: common.STATE_ADDED,
	},
	}

	regimenSection2 := &common.RegimenSection{}
	regimenSection2.RegimenName = "night"
	regimenSection2.RegimenSteps = []*common.DoctorInstructionItem{&common.DoctorInstructionItem{
		Text:  regimenStep2.Text,
		State: common.STATE_ADDED,
	},
	}

	regimenPlanRequest.AllRegimenSteps = []*common.DoctorInstructionItem{regimenStep1, regimenStep2}
	regimenPlanRequest.RegimenSections = []*common.RegimenSection{regimenSection, regimenSection2}
	regimenPlanResponse := CreateRegimenPlanForTreatmentPlan(regimenPlanRequest, testData, doctor, t)
	ValidateRegimenRequestAgainstResponse(regimenPlanRequest, regimenPlanResponse, t)

	// lets submit advice for this patient
	// lets go ahead and add a couple of advice points
	// reason we do this is because the advice steps have to exist before treatment plan can be favorited,
	// and the only way we can create advice steps today is in the context of a patient visit
	advicePoint1 := &common.DoctorInstructionItem{Text: "Advice point 1", State: common.STATE_ADDED}
	advicePoint2 := &common.DoctorInstructionItem{Text: "Advice point 2", State: common.STATE_ADDED}

	// lets go ahead and create a request for this patient visit
	doctorAdviceRequest := &common.Advice{}
	doctorAdviceRequest.AllAdvicePoints = []*common.DoctorInstructionItem{advicePoint1, advicePoint2}
	doctorAdviceRequest.SelectedAdvicePoints = doctorAdviceRequest.AllAdvicePoints
	doctorAdviceRequest.TreatmentPlanId = treatmentPlan.Id

	doctorAdviceResponse := UpdateAdvicePointsForPatientVisit(doctorAdviceRequest, testData, doctor, t)
	ValidateAdviceRequestAgainstResponse(doctorAdviceRequest, doctorAdviceResponse, t)

	// prepare the regimen steps and the advice points to be added into the sections
	// after the global list for each has been updated to include items.
	// the reason this is important is because favorite treatment plans require items to exist that are linked
	// from the master list
	regimenSection.RegimenSteps[0].ParentId = regimenPlanResponse.AllRegimenSteps[0].Id
	regimenSection2.RegimenSteps[0].ParentId = regimenPlanResponse.AllRegimenSteps[1].Id
	advicePoint1 = &common.DoctorInstructionItem{
		Text:     advicePoint1.Text,
		ParentId: doctorAdviceResponse.AllAdvicePoints[0].Id,
	}
	advicePoint2 = &common.DoctorInstructionItem{
		Text:     advicePoint2.Text,
		ParentId: doctorAdviceResponse.AllAdvicePoints[1].Id,
	}

	treatment1 := &common.Treatment{
		DrugDBIds: map[string]string{
			erx.LexiDrugSynId:     "1234",
			erx.LexiGenProductId:  "12345",
			erx.LexiSynonymTypeId: "123556",
			erx.NDC:               "2415",
		},
		DrugInternalName:        "Teting (This - Drug)",
		DosageStrength:          "10 mg",
		DispenseValue:           5,
		DispenseUnitDescription: "Tablet",
		DispenseUnitId:          encoding.NewObjectId(19),
		NumberRefills: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 5,
		},
		SubstitutionsAllowed: false,
		DaysSupply: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 5,
		},
		PatientInstructions: "Take once daily",
		OTC:                 false,
	}

	AddAndGetTreatmentsForPatientVisit(testData, []*common.Treatment{treatment1}, doctor.AccountId.Int64(), treatmentPlan.Id.Int64(), t)

	// lets add a favorite treatment plan for doctor
	favoriteTreatmentPlan := &common.FavoriteTreatmentPlan{
		Name: "Test Treatment Plan",
		TreatmentList: &common.TreatmentList{
			Treatments: []*common.Treatment{treatment1},
		},
		RegimenPlan: &common.RegimenPlan{
			AllRegimenSteps: regimenPlanResponse.AllRegimenSteps,
			RegimenSections: []*common.RegimenSection{regimenSection, regimenSection2},
		},
		Advice: &common.Advice{
			AllAdvicePoints:      doctorAdviceResponse.AllAdvicePoints,
			SelectedAdvicePoints: []*common.DoctorInstructionItem{advicePoint1, advicePoint2},
		},
	}

	ts := httptest.NewServer(doctor_treatment_plan.NewDoctorFavoriteTreatmentPlansHandler(testData.DataApi))
	defer ts.Close()

	requestData := &doctor_treatment_plan.DoctorFavoriteTreatmentPlansRequestData{
		FavoriteTreatmentPlan: favoriteTreatmentPlan,
		TreatmentPlanId:       treatmentPlan.Id.Int64(),
	}
	jsonData, err := json.Marshal(&requestData)
	if err != nil {
		t.Fatalf("Unable to marshal json %s", err)
	}

	resp, err := testData.AuthPost(ts.URL, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	if err != nil {
		t.Fatalf("Unable to add favorite treatment plan: %s", err)
	}

	responseData := &doctor_treatment_plan.DoctorFavoriteTreatmentPlansResponseData{}
	if err := json.NewDecoder(resp.Body).Decode(responseData); err != nil {
		t.Fatalf("Unable to unmarshal response into json %s", err)
	} else if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 response for adding a favorite treatment plan but got %d instead", resp.StatusCode)
	} else if responseData.FavoriteTreatmentPlan == nil {
		t.Fatalf("Expected to get back the treatment plan added but got none")
	} else if responseData.FavoriteTreatmentPlan.RegimenPlan == nil || len(responseData.FavoriteTreatmentPlan.RegimenPlan.RegimenSections) != 2 {
		t.Fatalf("Expected to have a regimen plan or 2 items in the regimen section")
	} else if len(responseData.FavoriteTreatmentPlan.Advice.SelectedAdvicePoints) != 2 {
		t.Fatalf("Expected 2 items in the advice list")
	}

	abbreviatedTreatmentPlan, err := testData.DataApi.GetAbridgedTreatmentPlan(treatmentPlan.Id.Int64(), doctorId)
	if err != nil {
		t.Fatalf("Unable to get abbreviated favorite treatment plan: %s", err)
	} else if abbreviatedTreatmentPlan.ContentSource == nil || abbreviatedTreatmentPlan.ContentSource.ContentSourceType != common.TPContentSourceTypeFTP ||
		abbreviatedTreatmentPlan.ContentSource.ContentSourceId.Int64() != responseData.FavoriteTreatmentPlan.Id.Int64() {
		t.Fatalf("Expected the link between treatmenet plan and favorite treatment plan to exist but it doesnt")
	}
}

func TestFavoriteTreatmentPlan_InContextOfTreatmentPlan_EmptyRegimenAndAdvice(t *testing.T) {
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	doctorId := GetDoctorIdOfCurrentDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatalf("Unable to get doctor from id: %s", err)
	}

	_, treatmentPlan := CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	regimenPlanRequest := &common.RegimenPlan{}
	regimenPlanRequest.TreatmentPlanId = treatmentPlan.Id

	regimenStep1 := &common.DoctorInstructionItem{}
	regimenStep1.Text = "Regimen Step 1"
	regimenStep1.State = common.STATE_ADDED

	regimenStep2 := &common.DoctorInstructionItem{}
	regimenStep2.Text = "Regimen Step 2"
	regimenStep2.State = common.STATE_ADDED

	regimenPlanRequest.AllRegimenSteps = []*common.DoctorInstructionItem{regimenStep1, regimenStep2}
	regimenPlanResponse := CreateRegimenPlanForTreatmentPlan(regimenPlanRequest, testData, doctor, t)
	ValidateRegimenRequestAgainstResponse(regimenPlanRequest, regimenPlanResponse, t)

	// lets submit advice for this patient
	// lets go ahead and add a couple of advice points
	// reason we do this is because the advice steps have to exist before treatment plan can be favorited,
	// and the only way we can create advice steps today is in the context of a patient visit
	advicePoint1 := &common.DoctorInstructionItem{Text: "Advice point 1", State: common.STATE_ADDED}
	advicePoint2 := &common.DoctorInstructionItem{Text: "Advice point 2", State: common.STATE_ADDED}

	// lets go ahead and create a request for this patient visit
	doctorAdviceRequest := &common.Advice{}
	doctorAdviceRequest.AllAdvicePoints = []*common.DoctorInstructionItem{advicePoint1, advicePoint2}
	doctorAdviceRequest.TreatmentPlanId = treatmentPlan.Id

	doctorAdviceResponse := UpdateAdvicePointsForPatientVisit(doctorAdviceRequest, testData, doctor, t)
	ValidateAdviceRequestAgainstResponse(doctorAdviceRequest, doctorAdviceResponse, t)

	advicePoint1 = &common.DoctorInstructionItem{
		Text:     advicePoint1.Text,
		ParentId: doctorAdviceResponse.AllAdvicePoints[0].Id,
	}
	advicePoint2 = &common.DoctorInstructionItem{
		Text:     advicePoint2.Text,
		ParentId: doctorAdviceResponse.AllAdvicePoints[1].Id,
	}

	treatment1 := &common.Treatment{
		DrugDBIds: map[string]string{
			erx.LexiDrugSynId:     "1234",
			erx.LexiGenProductId:  "12345",
			erx.LexiSynonymTypeId: "123556",
			erx.NDC:               "2415",
		},
		DrugInternalName:        "Teting (This - Drug)",
		DosageStrength:          "10 mg",
		DispenseValue:           5,
		DispenseUnitDescription: "Tablet",
		DispenseUnitId:          encoding.NewObjectId(19),
		NumberRefills: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 5,
		},
		SubstitutionsAllowed: false,
		DaysSupply: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 5,
		},
		PatientInstructions: "Take once daily",
		OTC:                 false,
	}

	AddAndGetTreatmentsForPatientVisit(testData, []*common.Treatment{treatment1}, doctor.AccountId.Int64(), treatmentPlan.Id.Int64(), t)

	// lets add a favorite treatment plan for doctor
	favoriteTreatmentPlan := &common.FavoriteTreatmentPlan{
		Name: "Test Treatment Plan",
		TreatmentList: &common.TreatmentList{
			Treatments: []*common.Treatment{treatment1},
		},
		RegimenPlan: &common.RegimenPlan{
			AllRegimenSteps: regimenPlanResponse.AllRegimenSteps,
		},
		Advice: &common.Advice{
			AllAdvicePoints: doctorAdviceResponse.AllAdvicePoints,
		},
	}

	ts := httptest.NewServer(doctor_treatment_plan.NewDoctorFavoriteTreatmentPlansHandler(testData.DataApi))
	defer ts.Close()

	requestData := &doctor_treatment_plan.DoctorFavoriteTreatmentPlansRequestData{
		FavoriteTreatmentPlan: favoriteTreatmentPlan,
		TreatmentPlanId:       treatmentPlan.Id.Int64(),
	}
	jsonData, err := json.Marshal(&requestData)
	if err != nil {
		t.Fatalf("Unable to marshal json %s", err)
	}

	resp, err := testData.AuthPost(ts.URL, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	if err != nil {
		t.Fatalf("Unable to add favorite treatment plan: %s", err)
	}

	responseData := &doctor_treatment_plan.DoctorFavoriteTreatmentPlansResponseData{}
	if err := json.NewDecoder(resp.Body).Decode(responseData); err != nil {
		t.Fatalf("Unable to unmarshal response into json %s", err)
	} else if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 response for adding a favorite treatment plan but got %d instead", resp.StatusCode)
	} else if responseData.FavoriteTreatmentPlan == nil {
		t.Fatalf("Expected to get back the treatment plan added but got none")
	}

	abbreviatedTreatmentPlan, err := testData.DataApi.GetAbridgedTreatmentPlan(treatmentPlan.Id.Int64(), doctorId)
	if err != nil {
		t.Fatalf("Unable to get abbreviated favorite treatment plan: %s", err)
	} else if abbreviatedTreatmentPlan.ContentSource == nil || abbreviatedTreatmentPlan.ContentSource.ContentSourceType != common.TPContentSourceTypeFTP ||
		abbreviatedTreatmentPlan.ContentSource.ContentSourceId.Int64() != responseData.FavoriteTreatmentPlan.Id.Int64() {
		t.Fatalf("Expected the link between treatmenet plan and favorite treatment plan to exist but it doesnt")
	}

}

func TestFavoriteTreatmentPlan_InContextOfTreatmentPlan_TwoDontMatch(t *testing.T) {
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	doctorId := GetDoctorIdOfCurrentDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatalf("Unable to get doctor from id: %s", err)
	}

	_, treatmentPlan := CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	regimenPlanRequest := &common.RegimenPlan{}
	regimenPlanRequest.TreatmentPlanId = treatmentPlan.Id

	regimenStep1 := &common.DoctorInstructionItem{}
	regimenStep1.Text = "Regimen Step 1"
	regimenStep1.State = common.STATE_ADDED

	regimenStep2 := &common.DoctorInstructionItem{}
	regimenStep2.Text = "Regimen Step 2"
	regimenStep2.State = common.STATE_ADDED

	regimenPlanRequest.AllRegimenSteps = []*common.DoctorInstructionItem{regimenStep1, regimenStep2}
	regimenPlanResponse := CreateRegimenPlanForTreatmentPlan(regimenPlanRequest, testData, doctor, t)
	ValidateRegimenRequestAgainstResponse(regimenPlanRequest, regimenPlanResponse, t)

	// lets submit advice for this patient
	// lets go ahead and add a couple of advice points
	// reason we do this is because the advice steps have to exist before treatment plan can be favorited,
	// and the only way we can create advice steps today is in the context of a patient visit
	advicePoint1 := &common.DoctorInstructionItem{Text: "Advice point 1", State: common.STATE_ADDED}
	advicePoint2 := &common.DoctorInstructionItem{Text: "Advice point 2", State: common.STATE_ADDED}

	// lets go ahead and create a request for this patient visit
	doctorAdviceRequest := &common.Advice{}
	doctorAdviceRequest.AllAdvicePoints = []*common.DoctorInstructionItem{advicePoint1, advicePoint2}
	doctorAdviceRequest.SelectedAdvicePoints = doctorAdviceRequest.AllAdvicePoints
	doctorAdviceRequest.TreatmentPlanId = treatmentPlan.Id

	doctorAdviceResponse := UpdateAdvicePointsForPatientVisit(doctorAdviceRequest, testData, doctor, t)
	ValidateAdviceRequestAgainstResponse(doctorAdviceRequest, doctorAdviceResponse, t)

	advicePoint1 = &common.DoctorInstructionItem{
		Text:     advicePoint1.Text,
		ParentId: doctorAdviceResponse.AllAdvicePoints[0].Id,
	}
	advicePoint2 = &common.DoctorInstructionItem{
		Text:     advicePoint2.Text,
		ParentId: doctorAdviceResponse.AllAdvicePoints[1].Id,
	}

	treatment1 := &common.Treatment{
		DrugDBIds: map[string]string{
			erx.LexiDrugSynId:     "1234",
			erx.LexiGenProductId:  "12345",
			erx.LexiSynonymTypeId: "123556",
			erx.NDC:               "2415",
		},
		DrugInternalName:        "Teting (This - Drug)",
		DosageStrength:          "10 mg",
		DispenseValue:           5,
		DispenseUnitDescription: "Tablet",
		DispenseUnitId:          encoding.NewObjectId(19),
		NumberRefills: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 5,
		},
		SubstitutionsAllowed: false,
		DaysSupply: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 5,
		},
		PatientInstructions: "Take once daily",
		OTC:                 false,
	}

	AddAndGetTreatmentsForPatientVisit(testData, []*common.Treatment{treatment1}, doctor.AccountId.Int64(), treatmentPlan.Id.Int64(), t)

	// lets add a favorite treatment plan for doctor
	favoriteTreatmentPlan := &common.FavoriteTreatmentPlan{
		Name: "Test Treatment Plan",
		TreatmentList: &common.TreatmentList{
			Treatments: []*common.Treatment{treatment1},
		},
		RegimenPlan: &common.RegimenPlan{
			AllRegimenSteps: regimenPlanResponse.AllRegimenSteps,
		},
		Advice: &common.Advice{
			AllAdvicePoints: doctorAdviceResponse.AllAdvicePoints,
		},
	}

	ts := httptest.NewServer(doctor_treatment_plan.NewDoctorFavoriteTreatmentPlansHandler(testData.DataApi))
	defer ts.Close()

	requestData := &doctor_treatment_plan.DoctorFavoriteTreatmentPlansRequestData{
		FavoriteTreatmentPlan: favoriteTreatmentPlan,
		TreatmentPlanId:       treatmentPlan.Id.Int64(),
	}
	jsonData, err := json.Marshal(&requestData)
	if err != nil {
		t.Fatalf("Unable to marshal json %s", err)
	}

	resp, err := testData.AuthPost(ts.URL, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	if err != nil {
		t.Fatalf("Unable to add favorite treatment plan: %s", err)
	}

	responseData := &doctor_treatment_plan.DoctorFavoriteTreatmentPlansResponseData{}
	if err := json.NewDecoder(resp.Body).Decode(responseData); err != nil {
		t.Fatalf("Unable to unmarshal response into json %s", err)
	} else if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected 400 response for adding a favorite treatment plan but got %d instead", resp.StatusCode)
	}

	abbreviatedTreatmentPlan, err := testData.DataApi.GetAbridgedTreatmentPlan(treatmentPlan.Id.Int64(), doctorId)
	if err != nil {
		t.Fatalf("Unable to get abbreviated favorite treatment plan: %s", err)
	} else if abbreviatedTreatmentPlan.ContentSource != nil {
		t.Fatalf("Expected the treatment plan to not indicate that it was linked to another doctor's favorite treatment plan")
	}

}
