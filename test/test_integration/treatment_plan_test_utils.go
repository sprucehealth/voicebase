package test_integration

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"testing"

	"github.com/sprucehealth/backend/apiservice/router"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/test"
)

func GetRegimenPlanForTreatmentPlan(testData *TestData, doctor *common.Doctor, treatmentPlanId int64, t *testing.T) *common.RegimenPlan {

	resp, err := testData.AuthGet(testData.APIServer.URL+router.DoctorTreatmentPlansURLPath+"?treatment_plan_id="+strconv.FormatInt(treatmentPlanId, 10), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to get regimen for patient visit: " + err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 instead got %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to parse the body of the response for getting the regimen plan: " + err.Error())
	}

	doctorTreatmentPlanResponse := &doctor_treatment_plan.DoctorTreatmentPlanResponse{}
	err = json.Unmarshal(body, doctorTreatmentPlanResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal body into json object: " + err.Error())
	}

	return doctorTreatmentPlanResponse.TreatmentPlan.RegimenPlan
}

func CreateRegimenPlanForTreatmentPlan(doctorRegimenRequest *common.RegimenPlan, testData *TestData, doctor *common.Doctor, t *testing.T) *common.RegimenPlan {
	requestBody, err := json.Marshal(doctorRegimenRequest)
	if err != nil {
		t.Fatal("Unable to marshal request body for adding regimen steps: " + err.Error())
	}

	resp, err := testData.AuthPost(testData.APIServer.URL+router.DoctorRegimenURLPath, "application/json", bytes.NewBuffer(requestBody), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make successful request to create regimen for patient visit")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 instead got %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to read body of response after making call to create regimen plan")
	}

	regimenPlanResponse := &common.RegimenPlan{}
	err = json.Unmarshal(body, regimenPlanResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal response into json object : " + err.Error())
	}

	return regimenPlanResponse
}

func GetAdvicePointsInTreatmentPlan(testData *TestData, doctor *common.Doctor, treatmentPlanId int64, t *testing.T) *common.Advice {
	resp, err := testData.AuthGet(testData.APIServer.URL+router.DoctorTreatmentPlansURLPath+"?treatment_plan_id="+strconv.FormatInt(treatmentPlanId, 10), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to get advice points for patient visit: " + err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 instead got %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to parse the body of the response for getting the advice points: " + err.Error())
	}

	doctorTreatmentPlanResponse := &doctor_treatment_plan.DoctorTreatmentPlanResponse{}
	err = json.Unmarshal(body, doctorTreatmentPlanResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal the response body into the advice repsonse object: " + err.Error())
	}

	return doctorTreatmentPlanResponse.TreatmentPlan.Advice
}

func UpdateAdvicePointsForPatientVisit(doctorAdviceRequest *common.Advice, testData *TestData, doctor *common.Doctor, t *testing.T) *common.Advice {
	requestBody, err := json.Marshal(doctorAdviceRequest)
	if err != nil {
		t.Fatal("Unable to marshal request body for adding advice points: " + err.Error())
	}

	resp, err := testData.AuthPost(testData.APIServer.URL+router.DoctorAdviceURLPath, "application/json", bytes.NewBuffer(requestBody), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make successful request to add advice points to patient visit " + err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 instead got %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable tp read body of the response after adding advice points to patient visit: " + err.Error())
	}

	doctorAdviceResponse := &common.Advice{}
	err = json.Unmarshal(body, doctorAdviceResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal response body into json object : " + err.Error())
	}

	return doctorAdviceResponse
}

func GetListOfTreatmentPlansForPatient(patientId, doctorAccountId int64, testData *TestData, t *testing.T) *doctor_treatment_plan.TreatmentPlansResponse {

	response := &doctor_treatment_plan.TreatmentPlansResponse{}
	res, err := testData.AuthGet(testData.APIServer.URL+router.DoctorTreatmentPlansListURLPath+"?patient_id="+strconv.FormatInt(patientId, 10), doctorAccountId)
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code %d but got %d instead", http.StatusOK, res.StatusCode)
	} else if err := json.NewDecoder(res.Body).Decode(response); err != nil {
		t.Fatalf(err.Error())
	}

	return response
}

func DeleteTreatmentPlanForDoctor(treatmentPlanId, doctorAccountId int64, testData *TestData, t *testing.T) {
	jsonData, err := json.Marshal(&doctor_treatment_plan.TreatmentPlanRequestData{
		TreatmentPlanID: treatmentPlanId,
	})

	res, err := testData.AuthDelete(testData.APIServer.URL+router.DoctorTreatmentPlansURLPath, "application/json", bytes.NewReader(jsonData), doctorAccountId)
	test.OK(t, err)
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected %d but got %d instead", http.StatusOK, res.StatusCode)
	}
}

func GetDoctorTreatmentPlanById(treatmentPlanId, doctorAccountId int64, testData *TestData, t *testing.T) *common.DoctorTreatmentPlan {
	response := &doctor_treatment_plan.DoctorTreatmentPlanResponse{}
	res, err := testData.AuthGet(testData.APIServer.URL+router.DoctorTreatmentPlansURLPath+"?treatment_plan_id="+strconv.FormatInt(treatmentPlanId, 10), doctorAccountId)
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code %d but got %d instead", http.StatusOK, res.StatusCode)
	} else if err := json.NewDecoder(res.Body).Decode(response); err != nil {
		t.Fatalf(err.Error())
	}
	return response.TreatmentPlan
}

func AddAndGetTreatmentsForPatientVisit(testData *TestData, treatments []*common.Treatment, doctorAccountId, treatmentPlanId int64, t *testing.T) *doctor_treatment_plan.GetTreatmentsResponse {
	testData.Config.ERxAPI = &erx.StubErxService{
		SelectedMedicationToReturn: &common.Treatment{},
	}

	treatmentRequestBody := doctor_treatment_plan.AddTreatmentsRequestBody{
		TreatmentPlanID: encoding.NewObjectId(treatmentPlanId),
		Treatments:      treatments,
	}

	data, err := json.Marshal(&treatmentRequestBody)
	if err != nil {
		t.Fatal("Unable to marshal request body for adding treatments to patient visit")
	}

	resp, err := testData.AuthPost(testData.APIServer.URL+router.DoctorVisitTreatmentsURLPath, "application/json", bytes.NewBuffer(data), doctorAccountId)
	if err != nil {
		t.Fatal("Unable to make POST request to add treatments to patient visit " + err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 instead got %d", resp.StatusCode)
	}

	addTreatmentsResponse := &doctor_treatment_plan.GetTreatmentsResponse{}
	err = json.NewDecoder(resp.Body).Decode(addTreatmentsResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal response into object : " + err.Error())
	}

	treatmentList := &common.TreatmentList{Treatments: treatments}
	if !treatmentList.Equals(addTreatmentsResponse.TreatmentList) {
		t.Fatal("Expected treatments added to match treatments returned but they dont")
	}

	return addTreatmentsResponse
}

func ValidateRegimenRequestAgainstResponse(doctorRegimenRequest, doctorRegimenResponse *common.RegimenPlan, t *testing.T) {

	// there should be the same number of sections in the request and the response
	if len(doctorRegimenRequest.Sections) != len(doctorRegimenResponse.Sections) {
		t.Fatalf("Number of regimen sections should be the same in the request and the response. Request = %d, response = %d", len(doctorRegimenRequest.Sections), len(doctorRegimenResponse.Sections))
	}

	// there should be the same number of steps in each section in the request and the response
	if doctorRegimenRequest.Sections != nil {
		for i, regimenSection := range doctorRegimenRequest.Sections {
			if len(regimenSection.Steps) != len(doctorRegimenResponse.Sections[i].Steps) {
				t.Fatalf(`the number of regimen steps in the regimen section of the request and the response should be the same, 
				regimen section = %s, request = %d, response = %d`, regimenSection.Name, len(regimenSection.Steps), len(doctorRegimenResponse.Sections[i].Steps))
			}
		}
	}

	// the number of steps in each regimen section should be the same across the request and response
	for i, regimenSection := range doctorRegimenRequest.Sections {
		if len(regimenSection.Steps) != len(doctorRegimenResponse.Sections[i].Steps) {
			t.Fatalf("Expected have the same number of regimen steps for each section. Section %s has %d steps but expected %d steps", regimenSection.Name, len(regimenSection.Steps), len(doctorRegimenResponse.Sections[i].Steps))
		}
	}

	// all regimen steps should have an id in the response
	regimenStepsMapping := make(map[int64]bool)
	for _, regimenStep := range doctorRegimenResponse.AllSteps {
		if regimenStep.ID.Int64() == 0 {
			t.Fatal("Regimen steps in the response are expected to have an id")
		}
		regimenStepsMapping[regimenStep.ID.Int64()] = true
	}

	// all regimen steps in the regimen sections should have an id in the response
	// all regimen steps in the sections that have a parentId should also be present in the global list
	for _, regimenSection := range doctorRegimenResponse.Sections {
		for _, regimenStep := range regimenSection.Steps {
			if regimenStep.ID.Int64() == 0 {
				t.Fatal("Regimen steps in each section are expected to have an id")
			}
			if regimenStep.ParentID.IsValid && regimenStepsMapping[regimenStep.ParentID.Int64()] == false {
				t.Fatalf("There exists a regimen step in a section that is not present in the global list. Id of regimen step %d", regimenStep.ID.Int64Value)
			}
		}
	}

	// no two items should have the same id
	idsFound := make(map[int64]bool)
	for _, regimenStep := range doctorRegimenResponse.AllSteps {
		if _, ok := idsFound[regimenStep.ID.Int64()]; ok {
			t.Fatal("No two items can have the same id in the global list")
		}
		idsFound[regimenStep.ID.Int64()] = true
	}

	// deleted regimen steps should not show up in the response
	deletedRegimenStepIds := make(map[int64]bool)
	// updated regimen steps should have a different id in the response
	updatedRegimenSteps := make(map[string][]int64)

	for _, regimenStep := range doctorRegimenRequest.AllSteps {
		switch regimenStep.State {
		case common.STATE_MODIFIED:
			updatedRegimenSteps[regimenStep.Text] = append(updatedRegimenSteps[regimenStep.Text], regimenStep.ID.Int64())
		}
	}

	for _, regimenStep := range doctorRegimenResponse.AllSteps {
		if updatedIds, ok := updatedRegimenSteps[regimenStep.Text]; ok {
			for _, updatedId := range updatedIds {
				if regimenStep.ID.Int64() == updatedId {
					t.Fatalf("Expected an updated regimen step to have a different id in the response. Id = %d", regimenStep.ID.Int64())
				}
			}
		}

		if deletedRegimenStepIds[regimenStep.ID.Int64()] == true {
			t.Fatalf("Expected regimen step %d to have been deleted and not in the response", regimenStep.ID.Int64())
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
		if advicePoint.ID.Int64() == 0 {
			t.Fatal("Advice point expected to have an id but it doesnt")
		}
		if advicePoint.Text == "" {
			t.Fatal("Advice point text is empty when not expected to be")
		}

		if _, ok := idsFound[advicePoint.ID.Int64()]; ok {
			t.Fatal("No two ids should be the same in the global list")
		}
		idsFound[advicePoint.ID.Int64()] = true

	}

	// now two ids should be the same in the selected list
	idsFound = make(map[int64]bool)
	parentIdsFound := make(map[int64]bool)
	// all advice points in the selected list should have ids
	for _, advicePoint := range doctorAdviceResponse.SelectedAdvicePoints {
		if advicePoint.ID.Int64() == 0 {
			t.Fatal("Selected Advice point expected to have an id but it doesnt")
		}
		if advicePoint.Text == "" {
			t.Fatal("Selectd advice point text is empty when not expected to be")
		}
		if _, ok := idsFound[advicePoint.ID.Int64()]; ok {
			t.Fatal("No two ids should be the same in the global list")
		}
		idsFound[advicePoint.ID.Int64()] = true

		if _, ok := parentIdsFound[advicePoint.ParentID.Int64()]; advicePoint.ParentID.IsValid && ok {
			t.Fatal("No two ids should be the same in the global list")
		}
		parentIdsFound[advicePoint.ParentID.Int64()] = true
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
			textToIdMapping[advicePoint.Text] = append(textToIdMapping[advicePoint.Text], advicePoint.ID.Int64())

		case common.STATE_ADDED:
			newAdvicePoints[advicePoint.Text] = true
		}
	}

	for _, advicePoint := range doctorAdviceResponse.AllAdvicePoints {
		if updatedIds, ok := textToIdMapping[advicePoint.Text]; ok {
			for _, updatedId := range updatedIds {
				if updatedId == advicePoint.ID.Int64() {
					t.Fatal("Updated advice points should have different ids")
				}
			}
		}

		if deletedAdvicePointIds[advicePoint.ID.Int64()] == true {
			t.Fatal("Deleted advice point should not exist in the response")
		}

		if newAdvicePoints[advicePoint.Text] == true {
			if advicePoint.ID.Int64() == 0 {
				t.Fatal("Newly added advice point should have an id")
			}
		}
	}
}

func CreateFavoriteTreatmentPlan(patientVisitId, treatmentPlanId int64, testData *TestData, doctor *common.Doctor, t *testing.T) *common.FavoriteTreatmentPlan {

	// lets submit a regimen plan for this patient
	// reason we do this is because the regimen steps have to exist before treatment plan can be favorited,
	// and the only way we can create regimen steps today is in the context of a patient visit
	regimenPlanRequest := &common.RegimenPlan{
		TreatmentPlanID: encoding.NewObjectId(treatmentPlanId),
	}

	regimenStep1 := &common.DoctorInstructionItem{
		Text:  "Regimen Step 1",
		State: common.STATE_ADDED,
	}

	regimenStep2 := &common.DoctorInstructionItem{
		Text:  "Regimen Step 2",
		State: common.STATE_ADDED,
	}

	regimenSection := &common.RegimenSection{
		Name: "morning",
		Steps: []*common.DoctorInstructionItem{{
			Text:  regimenStep1.Text,
			State: common.STATE_ADDED,
		}},
	}

	regimenSection2 := &common.RegimenSection{
		Name: "night",
		Steps: []*common.DoctorInstructionItem{{
			Text:  regimenStep2.Text,
			State: common.STATE_ADDED,
		}},
	}

	regimenPlanRequest.AllSteps = []*common.DoctorInstructionItem{regimenStep1, regimenStep2}
	regimenPlanResponse := CreateRegimenPlanForTreatmentPlan(regimenPlanRequest, testData, doctor, t)
	ValidateRegimenRequestAgainstResponse(regimenPlanRequest, regimenPlanResponse, t)

	// lets submit advice for this patient
	// lets go ahead and add a couple of advice points
	// reason we do this is because the advice steps have to exist before treatment plan can be favorited,
	// and the only way we can create advice steps today is in the context of a patient visit
	advicePoint1 := &common.DoctorInstructionItem{Text: "Advice point 1", State: common.STATE_ADDED}
	advicePoint2 := &common.DoctorInstructionItem{Text: "Advice point 2", State: common.STATE_ADDED}

	// lets go ahead and create a request for this patient visit
	doctorAdviceRequest := &common.Advice{
		AllAdvicePoints: []*common.DoctorInstructionItem{advicePoint1, advicePoint2},
		TreatmentPlanID: encoding.NewObjectId(treatmentPlanId),
	}

	doctorAdviceResponse := UpdateAdvicePointsForPatientVisit(doctorAdviceRequest, testData, doctor, t)
	ValidateAdviceRequestAgainstResponse(doctorAdviceRequest, doctorAdviceResponse, t)

	// prepare the regimen steps and the advice points to be added into the sections
	// after the global list for each has been updated to include items.
	// the reason this is important is because favorite treatment plans require items to exist that are linked
	// from the master list
	regimenSection.Steps[0].ParentID = regimenPlanResponse.AllSteps[0].ID
	regimenSection2.Steps[0].ParentID = regimenPlanResponse.AllSteps[1].ID
	advicePoint1 = &common.DoctorInstructionItem{
		Text:     advicePoint1.Text,
		ParentID: doctorAdviceResponse.AllAdvicePoints[0].ID,
	}
	advicePoint2 = &common.DoctorInstructionItem{
		Text:     advicePoint2.Text,
		ParentID: doctorAdviceResponse.AllAdvicePoints[1].ID,
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
			AllSteps: regimenPlanResponse.AllSteps,
			Sections: []*common.RegimenSection{regimenSection, regimenSection2},
		},
		Advice: &common.Advice{
			AllAdvicePoints:      doctorAdviceResponse.AllAdvicePoints,
			SelectedAdvicePoints: []*common.DoctorInstructionItem{advicePoint1, advicePoint2},
		},
	}

	requestData := &doctor_treatment_plan.DoctorFavoriteTreatmentPlansRequestData{
		FavoriteTreatmentPlan: favoriteTreatmentPlan,
	}
	jsonData, err := json.Marshal(&requestData)
	if err != nil {
		t.Fatalf("Unable to marshal json %s", err)
	}

	resp, err := testData.AuthPost(testData.APIServer.URL+router.DoctorFTPURLPath, "application/json", bytes.NewReader(jsonData), doctor.AccountId.Int64())
	if err != nil {
		t.Fatalf("Unable to add favorite treatment plan: %s", err)
	}
	defer resp.Body.Close()

	responseData := &doctor_treatment_plan.DoctorFavoriteTreatmentPlansResponseData{}
	if err := json.NewDecoder(resp.Body).Decode(responseData); err != nil {
		t.Fatalf("Unable to unmarshal response into json %s", err)
	} else if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 response for adding a favorite treatment plan but got %d instead", resp.StatusCode)
	} else if responseData.FavoriteTreatmentPlan == nil {
		t.Fatalf("Expected to get back the treatment plan added but got none")
	} else if responseData.FavoriteTreatmentPlan.RegimenPlan == nil || len(responseData.FavoriteTreatmentPlan.RegimenPlan.Sections) != 2 {
		t.Fatalf("Expected to have a regimen plan or 2 items in the regimen section")
	} else if len(responseData.FavoriteTreatmentPlan.Advice.SelectedAdvicePoints) != 2 {
		t.Fatalf("Expected 2 items in the advice list")
	}

	return responseData.FavoriteTreatmentPlan
}
