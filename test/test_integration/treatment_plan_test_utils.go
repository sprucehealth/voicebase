package test_integration

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/erx"
)

func GetRegimenPlanForTreatmentPlan(testData *TestData, doctor *common.Doctor, treatmentPlanId int64, t *testing.T) *common.RegimenPlan {
	doctorTreatmentPlanHandler := doctor_treatment_plan.NewDoctorTreatmentPlanHandler(testData.DataApi, nil, nil, false)
	ts := httptest.NewServer(doctorTreatmentPlanHandler)
	defer ts.Close()

	resp, err := testData.AuthGet(ts.URL+"?treatment_plan_id="+strconv.FormatInt(treatmentPlanId, 10), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to get regimen for patient visit: " + err.Error())
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to parse the body of the response for getting the regimen plan: " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unable to make successful call to get regimen plan for patient visit: "+string(body), t)

	doctorTreatmentPlanResponse := &doctor_treatment_plan.DoctorTreatmentPlanResponse{}
	err = json.Unmarshal(body, doctorTreatmentPlanResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal body into json object: " + err.Error())
	}

	return doctorTreatmentPlanResponse.TreatmentPlan.RegimenPlan
}

func CreateRegimenPlanForTreatmentPlan(doctorRegimenRequest *common.RegimenPlan, testData *TestData, doctor *common.Doctor, t *testing.T) *common.RegimenPlan {
	doctorRegimenHandler := doctor_treatment_plan.NewRegimenHandler(testData.DataApi)
	ts := httptest.NewServer(doctorRegimenHandler)
	defer ts.Close()

	requestBody, err := json.Marshal(doctorRegimenRequest)
	if err != nil {
		t.Fatal("Unable to marshal request body for adding regimen steps: " + err.Error())
	}

	resp, err := testData.AuthPost(ts.URL, "application/json", bytes.NewBuffer(requestBody), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make successful request to create regimen for patient visit")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to read body of response after making call to create regimen plan")
	}

	CheckSuccessfulStatusCode(resp, "Unable to make successful call to create regimen plan for patient: "+string(body), t)

	regimenPlanResponse := &common.RegimenPlan{}
	err = json.Unmarshal(body, regimenPlanResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal response into json object : " + err.Error())
	}

	return regimenPlanResponse
}

func GetAdvicePointsInTreatmentPlan(testData *TestData, doctor *common.Doctor, treatmentPlanId int64, t *testing.T) *common.Advice {
	doctorTreatmentPlanHandler := doctor_treatment_plan.NewDoctorTreatmentPlanHandler(testData.DataApi, nil, nil, false)
	ts := httptest.NewServer(doctorTreatmentPlanHandler)
	defer ts.Close()

	resp, err := testData.AuthGet(ts.URL+"?treatment_plan_id="+strconv.FormatInt(treatmentPlanId, 10), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to get advice points for patient visit: " + err.Error())
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to parse the body of the response for getting the advice points: " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unable to make a successful call to get advice points for patient visit : "+string(body), t)

	doctorTreatmentPlanResponse := &doctor_treatment_plan.DoctorTreatmentPlanResponse{}
	err = json.Unmarshal(body, doctorTreatmentPlanResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal the response body into the advice repsonse object: " + err.Error())
	}

	return doctorTreatmentPlanResponse.TreatmentPlan.Advice
}

func UpdateAdvicePointsForPatientVisit(doctorAdviceRequest *common.Advice, testData *TestData, doctor *common.Doctor, t *testing.T) *common.Advice {
	doctorAdviceHandler := doctor_treatment_plan.NewAdviceHandler(testData.DataApi)
	ts := httptest.NewServer(doctorAdviceHandler)
	defer ts.Close()

	requestBody, err := json.Marshal(doctorAdviceRequest)
	if err != nil {
		t.Fatal("Unable to marshal request body for adding advice points: " + err.Error())
	}

	resp, err := testData.AuthPost(ts.URL, "application/json", bytes.NewBuffer(requestBody), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make successful request to add advice points to patient visit " + err.Error())
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable tp read body of the response after adding advice points to patient visit: " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unable to make successful call to add advice points : "+string(body), t)

	doctorAdviceResponse := &common.Advice{}
	err = json.Unmarshal(body, doctorAdviceResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal response body into json object : " + err.Error())
	}

	return doctorAdviceResponse
}

func GetListOfTreatmentPlansForPatient(patientId, doctorAccountId int64, testData *TestData, t *testing.T) *doctor_treatment_plan.TreatmentPlansResponse {
	listHandler := doctor_treatment_plan.NewListHandler(testData.DataApi)
	doctorServer := httptest.NewServer(listHandler)
	defer doctorServer.Close()

	response := &doctor_treatment_plan.TreatmentPlansResponse{}
	res, err := testData.AuthGet(doctorServer.URL+"?patient_id="+strconv.FormatInt(patientId, 10), doctorAccountId)
	if err != nil {
		t.Fatalf(err.Error())
	} else if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code %d but got %d instead", http.StatusOK, res.StatusCode)
	} else if err := json.NewDecoder(res.Body).Decode(response); err != nil {
		t.Fatalf(err.Error())
	}

	return response
}

func DeleteTreatmentPlanForDoctor(treatmentPlanId, doctorAccountId int64, testData *TestData, t *testing.T) {
	doctorTreatmentPlanHandler := doctor_treatment_plan.NewDoctorTreatmentPlanHandler(testData.DataApi, nil, nil, false)
	doctorServer := httptest.NewServer(doctorTreatmentPlanHandler)
	defer doctorServer.Close()

	jsonData, err := json.Marshal(&doctor_treatment_plan.TreatmentPlanRequestData{
		TreatmentPlanId: treatmentPlanId,
	})

	res, err := testData.AuthDelete(doctorServer.URL, "application/json", bytes.NewReader(jsonData), doctorAccountId)
	if err != nil {
		t.Fatal(err)
	} else if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected %d but got %d instead", http.StatusOK, res.StatusCode)
	}
}

func GetDoctorTreatmentPlanById(treatmentPlanId, doctorAccountId int64, testData *TestData, t *testing.T) *common.DoctorTreatmentPlan {
	drTreatmentPlanHandler := doctor_treatment_plan.NewDoctorTreatmentPlanHandler(testData.DataApi, nil, nil, false)
	doctorServer := httptest.NewServer(drTreatmentPlanHandler)
	defer doctorServer.Close()

	response := &doctor_treatment_plan.DoctorTreatmentPlanResponse{}
	res, err := testData.AuthGet(doctorServer.URL+"?treatment_plan_id="+strconv.FormatInt(treatmentPlanId, 10), doctorAccountId)
	if err != nil {
		t.Fatalf(err.Error())
	} else if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code %d but got %d instead", http.StatusOK, res.StatusCode)
	} else if err := json.NewDecoder(res.Body).Decode(response); err != nil {
		t.Fatalf(err.Error())
	}
	return response.TreatmentPlan
}

func AddAndGetTreatmentsForPatientVisit(testData *TestData, treatments []*common.Treatment, doctorAccountId, treatmentPlanId int64, t *testing.T) *doctor_treatment_plan.GetTreatmentsResponse {
	stubErxApi := &erx.StubErxService{
		SelectedMedicationToReturn: &common.Treatment{},
	}

	treatmentRequestBody := doctor_treatment_plan.AddTreatmentsRequestBody{TreatmentPlanId: encoding.NewObjectId(treatmentPlanId), Treatments: treatments}
	treatmentsHandler := doctor_treatment_plan.NewTreatmentsHandler(testData.DataApi, stubErxApi)

	ts := httptest.NewServer(treatmentsHandler)
	defer ts.Close()

	data, err := json.Marshal(&treatmentRequestBody)
	if err != nil {
		t.Fatal("Unable to marshal request body for adding treatments to patient visit")
	}

	resp, err := testData.AuthPost(ts.URL, "application/json", bytes.NewBuffer(data), doctorAccountId)
	if err != nil {
		t.Fatal("Unable to make POST request to add treatments to patient visit " + err.Error())
	}

	addTreatmentsResponse := &doctor_treatment_plan.GetTreatmentsResponse{}
	err = json.NewDecoder(resp.Body).Decode(addTreatmentsResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal response into object : " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unsuccessful call made to add treatments for patient visit: ", t)

	treatmentList := &common.TreatmentList{Treatments: treatments}
	if !treatmentList.Equals(addTreatmentsResponse.TreatmentList) {
		t.Fatal("Expected treatments added to match treatments returned but they dont")
	}

	return addTreatmentsResponse
}

func ValidateRegimenRequestAgainstResponse(doctorRegimenRequest, doctorRegimenResponse *common.RegimenPlan, t *testing.T) {

	// there should be the same number of sections in the request and the response
	if len(doctorRegimenRequest.RegimenSections) != len(doctorRegimenResponse.RegimenSections) {
		t.Fatalf("Number of regimen sections should be the same in the request and the response. Request = %d, response = %d", len(doctorRegimenRequest.RegimenSections), len(doctorRegimenResponse.RegimenSections))
	}

	// there should be the same number of steps in each section in the request and the response
	if doctorRegimenRequest.RegimenSections != nil {
		for i, regimenSection := range doctorRegimenRequest.RegimenSections {
			if len(regimenSection.RegimenSteps) != len(doctorRegimenResponse.RegimenSections[i].RegimenSteps) {
				t.Fatalf(`the number of regimen steps in the regimen section of the request and the response should be the same, 
				regimen section = %s, request = %d, response = %d`, regimenSection.RegimenName, len(regimenSection.RegimenSteps), len(doctorRegimenResponse.RegimenSections[i].RegimenSteps))
			}
		}
	}

	// the number of steps in each regimen section should be the same across the request and response
	for i, regimenSection := range doctorRegimenRequest.RegimenSections {
		if len(regimenSection.RegimenSteps) != len(doctorRegimenResponse.RegimenSections[i].RegimenSteps) {
			t.Fatalf("Expected have the same number of regimen steps for each section. Section %s has %d steps but expected %d steps", regimenSection.RegimenName, len(regimenSection.RegimenSteps), len(doctorRegimenResponse.RegimenSections[i].RegimenSteps))
		}
	}

	// all regimen steps should have an id in the response
	regimenStepsMapping := make(map[int64]bool)
	for _, regimenStep := range doctorRegimenResponse.AllRegimenSteps {
		if regimenStep.Id.Int64() == 0 {
			t.Fatal("Regimen steps in the response are expected to have an id")
		}
		regimenStepsMapping[regimenStep.Id.Int64()] = true
	}

	// all regimen steps in the regimen sections should have an id in the response
	// all regimen steps in the sections that have a parentId should also be present in the global list
	for _, regimenSection := range doctorRegimenResponse.RegimenSections {
		for _, regimenStep := range regimenSection.RegimenSteps {
			if regimenStep.Id.Int64() == 0 {
				t.Fatal("Regimen steps in each section are expected to have an id")
			}
			if regimenStep.ParentId.IsValid && regimenStepsMapping[regimenStep.ParentId.Int64()] == false {
				t.Fatalf("There exists a regimen step in a section that is not present in the global list. Id of regimen step %d", regimenStep.Id.Int64Value)
			}
		}
	}

	// no two items should have the same id
	idsFound := make(map[int64]bool)
	for _, regimenStep := range doctorRegimenResponse.AllRegimenSteps {
		if _, ok := idsFound[regimenStep.Id.Int64()]; ok {
			t.Fatal("No two items can have the same id in the global list")
		}
		idsFound[regimenStep.Id.Int64()] = true
	}

	// deleted regimen steps should not show up in the response
	deletedRegimenStepIds := make(map[int64]bool)
	// updated regimen steps should have a different id in the response
	updatedRegimenSteps := make(map[string][]int64)

	for _, regimenStep := range doctorRegimenRequest.AllRegimenSteps {
		switch regimenStep.State {
		case common.STATE_MODIFIED:
			updatedRegimenSteps[regimenStep.Text] = append(updatedRegimenSteps[regimenStep.Text], regimenStep.Id.Int64())
		}
	}

	for _, regimenStep := range doctorRegimenResponse.AllRegimenSteps {
		if updatedIds, ok := updatedRegimenSteps[regimenStep.Text]; ok {
			for _, updatedId := range updatedIds {
				if regimenStep.Id.Int64() == updatedId {
					t.Fatalf("Expected an updated regimen step to have a different id in the response. Id = %d", regimenStep.Id.Int64())
				}
			}
		}

		if deletedRegimenStepIds[regimenStep.Id.Int64()] == true {
			t.Fatalf("Expected regimen step %d to have been deleted and not in the response", regimenStep.Id.Int64())
		}
	}
}
func ValidateAdviceRequestAgainstResponse(doctorAdviceRequest, doctorAdviceResponse *common.Advice, t *testing.T) {
	if len(doctorAdviceRequest.SelectedAdvicePoints) != len(doctorAdviceResponse.SelectedAdvicePoints) {
		t.Fatalf("Expected the same number of selected advice points in request and response. Instead request has %d while response has %d", len(doctorAdviceRequest.SelectedAdvicePoints), len(doctorAdviceResponse.SelectedAdvicePoints))
	}

	// now two ids in the global list should be the same
	idsFound := make(map[int64]bool)

	// all advice points in the global list should have ids
	for _, advicePoint := range doctorAdviceResponse.AllAdvicePoints {
		if advicePoint.Id.Int64() == 0 {
			t.Fatal("Advice point expected to have an id but it doesnt")
		}
		if advicePoint.Text == "" {
			t.Fatal("Advice point text is empty when not expected to be")
		}

		if _, ok := idsFound[advicePoint.Id.Int64()]; ok {
			t.Fatal("No two ids should be the same in the global list")
		}
		idsFound[advicePoint.Id.Int64()] = true

	}

	// now two ids should be the same in the selected list
	idsFound = make(map[int64]bool)
	parentIdsFound := make(map[int64]bool)
	// all advice points in the selected list should have ids
	for _, advicePoint := range doctorAdviceResponse.SelectedAdvicePoints {
		if advicePoint.Id.Int64() == 0 {
			t.Fatal("Selected Advice point expected to have an id but it doesnt")
		}
		if advicePoint.Text == "" {
			t.Fatal("Selectd advice point text is empty when not expected to be")
		}
		if _, ok := idsFound[advicePoint.Id.Int64()]; ok {
			t.Fatal("No two ids should be the same in the global list")
		}
		idsFound[advicePoint.Id.Int64()] = true

		if _, ok := parentIdsFound[advicePoint.ParentId.Int64()]; advicePoint.ParentId.IsValid && ok {
			t.Fatal("No two ids should be the same in the global list")
		}
		parentIdsFound[advicePoint.ParentId.Int64()] = true
	}

	// all updated texts should have different ids than the requests
	// all deleted advice points should not exist in the response
	// all newly added advice points should have ids
	textToIdMapping := make(map[string][]int64)
	deletedAdvicePointIds := make(map[int64]bool)
	newAdvicePoints := make(map[string]bool)
	for _, advicePoint := range doctorAdviceRequest.AllAdvicePoints {
		switch advicePoint.State {
		case common.STATE_MODIFIED:
			textToIdMapping[advicePoint.Text] = append(textToIdMapping[advicePoint.Text], advicePoint.Id.Int64())

		case common.STATE_ADDED:
			newAdvicePoints[advicePoint.Text] = true
		}
	}

	for _, advicePoint := range doctorAdviceResponse.AllAdvicePoints {
		if updatedIds, ok := textToIdMapping[advicePoint.Text]; ok {
			for _, updatedId := range updatedIds {
				if updatedId == advicePoint.Id.Int64() {
					t.Fatal("Updated advice points should have different ids")
				}
			}
		}

		if deletedAdvicePointIds[advicePoint.Id.Int64()] == true {
			t.Fatal("Deleted advice point should not exist in the response")
		}

		if newAdvicePoints[advicePoint.Text] == true {
			if advicePoint.Id.Int64() == 0 {
				t.Fatal("Newly added advice point should have an id")
			}
		}
	}
}

func CreateFavoriteTreatmentPlan(patientVisitId, treatmentPlanId int64, testData *TestData, doctor *common.Doctor, t *testing.T) *common.FavoriteTreatmentPlan {

	// lets submit a regimen plan for this patient
	// reason we do this is because the regimen steps have to exist before treatment plan can be favorited,
	// and the only way we can create regimen steps today is in the context of a patient visit
	regimenPlanRequest := &common.RegimenPlan{}
	regimenPlanRequest.TreatmentPlanId = encoding.NewObjectId(treatmentPlanId)

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
	doctorAdviceRequest.TreatmentPlanId = encoding.NewObjectId(treatmentPlanId)

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

	// lets add a favorite treatment plan for doctor
	favoriteTreatmentPlan := &common.FavoriteTreatmentPlan{
		Name: "Test Treatment Plan",
		TreatmentList: &common.TreatmentList{
			Treatments: []*common.Treatment{&common.Treatment{
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
			},
			},
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

	return responseData.FavoriteTreatmentPlan
}
