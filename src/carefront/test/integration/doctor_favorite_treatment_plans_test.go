package integration

import (
	"bytes"
	"carefront/apiservice"
	"carefront/common"
	"carefront/encoding"
	"carefront/libs/erx"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestAddFavoriteTreatmentPlan(t *testing.T) {
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	doctorId := getDoctorIdOfCurrentPrimaryDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatalf("Unable to get doctor from id: %s", err)
	}

	patientVisitResponse := signupAndSubmitPatientVisitForRandomPatient(t, testData, doctor)

	// lets submit a regimen plan for this patient
	regimenPlanRequest := &common.RegimenPlan{}
	regimenPlanRequest.PatientVisitId = encoding.NewObjectId(patientVisitResponse.PatientVisitId)

	regimenStep1 := &common.DoctorInstructionItem{}
	regimenStep1.Text = "Regimen Step 1"
	regimenStep1.State = common.STATE_ADDED

	regimenStep2 := &common.DoctorInstructionItem{}
	regimenStep2.Text = "Regimen Step 2"
	regimenStep2.State = common.STATE_ADDED

	regimenSection := &common.RegimenSection{}
	regimenSection.RegimenName = "morning"
	regimenSection.RegimenSteps = []*common.DoctorInstructionItem{&common.DoctorInstructionItem{
		ParentId: regimenStep1.Id,
		Text:     regimenStep1.Text,
		State:    common.STATE_ADDED,
	},
	}

	regimenSection2 := &common.RegimenSection{}
	regimenSection2.RegimenName = "night"
	regimenSection2.RegimenSteps = []*common.DoctorInstructionItem{&common.DoctorInstructionItem{
		ParentId: regimenStep2.Id,
		Text:     regimenStep2.Text,
		State:    common.STATE_ADDED,
	},
	}

	regimenPlanRequest.AllRegimenSteps = []*common.DoctorInstructionItem{regimenStep1, regimenStep2}
	regimenPlanRequest.RegimenSections = []*common.RegimenSection{regimenSection, regimenSection2}
	regimenPlanResponse := createRegimenPlanForPatientVisit(regimenPlanRequest, testData, doctor, t)
	validateRegimenRequestAgainstResponse(regimenPlanRequest, regimenPlanResponse, t)

	// lets submit advice for this patient
	// lets go ahead and add a couple of advice points
	advicePoint1 := &common.DoctorInstructionItem{Text: "Advice point 1", State: common.STATE_ADDED}
	advicePoint2 := &common.DoctorInstructionItem{Text: "Advice point 2", State: common.STATE_ADDED}

	// lets go ahead and create a request for this patient visit
	doctorAdviceRequest := &common.Advice{}
	doctorAdviceRequest.AllAdvicePoints = []*common.DoctorInstructionItem{advicePoint1, advicePoint2}
	doctorAdviceRequest.SelectedAdvicePoints = doctorAdviceRequest.AllAdvicePoints
	doctorAdviceRequest.PatientVisitId = encoding.NewObjectId(patientVisitResponse.PatientVisitId)

	doctorAdviceResponse := updateAdvicePointsForPatientVisit(doctorAdviceRequest, testData, doctor, t)
	validateAdviceRequestAgainstResponse(doctorAdviceRequest, doctorAdviceResponse, t)

	// lets add a favorite treatment plan for doctor
	favoriteTreatmentPlan := &common.FavoriteTreatmentPlan{
		Name: "Test Treatment Plan",
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
		RegimenPlan: regimenPlanResponse,
		Advice:      doctorAdviceResponse,
	}

	doctorFavoriteTreatmentPlansHandler := &apiservice.DoctorFavoriteTreatmentPlansHandler{
		DataApi: testData.DataApi,
	}

	ts := httptest.NewServer(doctorFavoriteTreatmentPlansHandler)
	defer ts.Close()

	requestData := &apiservice.DoctorFavoriteTreatmentPlansRequestData{
		FavoriteTreatmentPlan: favoriteTreatmentPlan,
	}
	jsonData, err := json.Marshal(&requestData)
	if err != nil {
		t.Fatalf("Unable to marshal json %s", err)
	}

	resp, err := authPost(ts.URL, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	if err != nil {
		t.Fatalf("Unable to add favorite treatment plan: %s", err)
	}

	responseData := &apiservice.DoctorFavoriteTreatmentPlansResponseData{}
	if err := json.NewDecoder(resp.Body).Decode(responseData); err != nil {
		t.Fatalf("Unable to unmarshal response into json %s", responseData)
	}
	CheckSuccessfulStatusCode(resp, "Unable to make favorite treatment plan", t)

	if responseData.FavoriteTreatmentPlan == nil {
		t.Fatalf("Expected to get back the treatment plan added but got none")
	}

	if responseData.FavoriteTreatmentPlan.RegimenPlan == nil || len(responseData.FavoriteTreatmentPlan.RegimenPlan.RegimenSections) != 2 {
		t.Fatalf("Expected to have a regimen plan or 2 items in the regimen section")
	}

	if len(responseData.FavoriteTreatmentPlan.Advice.SelectedAdvicePoints) != 2 {
		t.Fatalf("Expected 2 items in the advice list")
	}

	// now lets go ahead and update the favorite treatment plan
	favoriteTreatmentPlan = responseData.FavoriteTreatmentPlan
	updatedName := "Updating name"
	favoriteTreatmentPlan.Name = updatedName
	favoriteTreatmentPlan.RegimenPlan.RegimenSections = []*common.RegimenSection{favoriteTreatmentPlan.RegimenPlan.RegimenSections[0]}
	favoriteTreatmentPlan.Advice.SelectedAdvicePoints = []*common.DoctorInstructionItem{favoriteTreatmentPlan.Advice.SelectedAdvicePoints[0]}

	requestData.FavoriteTreatmentPlan = favoriteTreatmentPlan
	jsonData, err = json.Marshal(requestData)
	if err != nil {
		t.Fatalf("Unable to marshal json data: %s", err)
	}

	resp, err = authPut(ts.URL, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	if err != nil {
		t.Fatalf("Unable to make call to update favorite treatment plan %s", err)
	}

	if err := json.NewDecoder(resp.Body).Decode(responseData); err != nil {
		t.Fatalf("Unable to decode response body into json object %s", err)
	}

	if responseData.FavoriteTreatmentPlan == nil {
		t.Fatalf("Expected 1 favorite treatment plan to be returned instead got back %d", len(responseData.FavoriteTreatmentPlans))
	}

	if len(responseData.FavoriteTreatmentPlan.RegimenPlan.RegimenSections) != 1 {
		t.Fatalf("Expected 1 section in the regimen plan instead got %d", len(responseData.FavoriteTreatmentPlan.RegimenPlan.RegimenSections))
	}

	if len(responseData.FavoriteTreatmentPlan.Advice.SelectedAdvicePoints) != 1 {
		t.Fatalf("Expected 1 section in the advice instead got %d", len(responseData.FavoriteTreatmentPlan.Advice.SelectedAdvicePoints))
	}

	if responseData.FavoriteTreatmentPlan.Name != updatedName {
		t.Fatalf("Expected name of favorite treatment plan to be %s instead got %s", updatedName, responseData.FavoriteTreatmentPlan.Name)
	}

	CheckSuccessfulStatusCode(resp, "unable to make call to update favorite treatment plan", t)
}
