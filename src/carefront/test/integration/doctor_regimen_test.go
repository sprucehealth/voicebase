package integration

import (
	"bytes"
	"carefront/apiservice"
	"carefront/common"
	"carefront/encoding"
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestRegimenForPatientVisit(t *testing.T) {

	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	patientVisitResponse, doctor := setupTestForRegimenCreation(t, testData)

	// attempt to get the regimen plan or a patient visit
	regimenPlan := getRegimenPlanForPatientVisit(testData, doctor, patientVisitResponse.PatientVisitId, t)

	if len(regimenPlan.AllRegimenSteps) > 0 {
		t.Fatal("There should be no regimen steps given that none have been created yet")
	}

	if len(regimenPlan.RegimenSections) > 0 {
		t.Fatal("There should be no regimen sections for the patient visit given that none have been created yet")
	}

	// adding new regimen steps to the doctor but not to the patient visit
	regimenPlanRequest := &common.RegimenPlan{}
	regimenPlanRequest.PatientVisitId = encoding.NewObjectId(patientVisitResponse.PatientVisitId)

	regimenStep1 := &common.DoctorInstructionItem{}
	regimenStep1.Text = "Regimen Step 1"
	regimenStep1.State = common.STATE_ADDED

	regimenStep2 := &common.DoctorInstructionItem{}
	regimenStep2.Text = "Regimen Step 2"
	regimenStep2.State = common.STATE_ADDED

	regimenPlanRequest.AllRegimenSteps = []*common.DoctorInstructionItem{regimenStep1, regimenStep2}
	regimenPlanResponse := createRegimenPlanForPatientVisit(regimenPlanRequest, testData, doctor, t)
	validateRegimenRequestAgainstResponse(regimenPlanRequest, regimenPlanResponse, t)

	if len(regimenPlanResponse.RegimenSections) > 0 {
		t.Fatal("Regimen section should not exist even though regimen steps were created by doctor")
	}

	// make the response the request since the response always returns the updated view of the system
	regimenPlanRequest = regimenPlanResponse

	// now lets add a couple regimen steps to a regimen section
	regimenSection := &common.RegimenSection{}
	regimenSection.RegimenName = "morning"
	regimenSection.RegimenSteps = []*common.DoctorInstructionItem{&common.DoctorInstructionItem{
		ParentId: regimenPlanRequest.AllRegimenSteps[0].Id,
		Text:     regimenPlanRequest.AllRegimenSteps[0].Text,
	},
	}

	regimenSection2 := &common.RegimenSection{}
	regimenSection2.RegimenName = "night"
	regimenSection2.RegimenSteps = []*common.DoctorInstructionItem{&common.DoctorInstructionItem{
		ParentId: regimenPlanRequest.AllRegimenSteps[1].Id,
		Text:     regimenPlanRequest.AllRegimenSteps[1].Text,
	},
	}

	regimenPlanRequest.RegimenSections = []*common.RegimenSection{regimenSection, regimenSection2}
	regimenPlanResponse = createRegimenPlanForPatientVisit(regimenPlanRequest, testData, doctor, t)
	validateRegimenRequestAgainstResponse(regimenPlanResponse, regimenPlanResponse, t)

	if len(regimenPlanResponse.RegimenSections) != 2 {
		t.Fatalf("Expected the number of regimen sections to be 2 but there are %d instead", len(regimenPlanResponse.RegimenSections))
	}

	// now remove a section from the request
	regimenPlanRequest = regimenPlanResponse
	regimenPlanRequest.RegimenSections = []*common.RegimenSection{regimenPlanRequest.RegimenSections[0]}

	regimenPlanResponse = createRegimenPlanForPatientVisit(regimenPlanRequest, testData, doctor, t)
	validateRegimenRequestAgainstResponse(regimenPlanRequest, regimenPlanResponse, t)

	if len(regimenPlanResponse.RegimenSections) != 1 {
		t.Fatalf("Expected the number of regimen sections to be 2 but there are %d instead", len(regimenPlanResponse.RegimenSections))
	}

	// lets update a regimen step in the section
	regimenPlanRequest = regimenPlanResponse
	regimenPlanRequest.AllRegimenSteps[0].Text = "UPDATED 1"
	regimenPlanRequest.AllRegimenSteps[0].State = common.STATE_MODIFIED
	regimenPlanRequest.RegimenSections[0].RegimenSteps[0].Text = "UPDATED 1"
	regimenPlanResponse = createRegimenPlanForPatientVisit(regimenPlanRequest, testData, doctor, t)
	validateRegimenRequestAgainstResponse(regimenPlanRequest, regimenPlanResponse, t)

	// lets delete a regimen step
	regimenPlanRequest = regimenPlanResponse
	regimenPlanRequest.AllRegimenSteps = []*common.DoctorInstructionItem{regimenPlanRequest.AllRegimenSteps[0]}
	regimenPlanRequest.RegimenSections = []*common.RegimenSection{}
	regimenPlanResponse = createRegimenPlanForPatientVisit(regimenPlanRequest, testData, doctor, t)
	validateRegimenRequestAgainstResponse(regimenPlanRequest, regimenPlanResponse, t)
	if len(regimenPlanResponse.AllRegimenSteps) != 1 {
		t.Fatal("Should only have 1 regimen step given that we just deleted one from the list")
	}

	// lets attempt to remove the regimen step, but keep it in the regimen section. This should fail
	// since the regimen step in the section does not exist in the global steps
	regimenPlanRequest = regimenPlanResponse
	regimenPlanRequest.AllRegimenSteps = []*common.DoctorInstructionItem{}
	regimenPlanRequest.RegimenSections = []*common.RegimenSection{regimenSection}
	doctorRegimenHandler := apiservice.NewDoctorRegimenHandler(testData.DataApi)
	ts := httptest.NewServer(doctorRegimenHandler)
	defer ts.Close()

	requestBody, err := json.Marshal(regimenPlanRequest)
	if err != nil {
		t.Fatal("Unable to marshal request body for adding regimen steps: " + err.Error())
	}

	resp, err := authPost(ts.URL, "application/json", bytes.NewBuffer(requestBody), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make successful request to create regimen for patient visit")
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatal("Expected to get a bad request for when the regimen step does not exist in the regimen sections")
	}

	// get patient to start a visit
	patientSignedupResponse := signupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patientVisitResponse = createPatientVisitForPatient(patientSignedupResponse.Patient.PatientId.Int64(), testData, t)

	regimenPlan = getRegimenPlanForPatientVisit(testData, doctor, patientVisitResponse.PatientVisitId, t)
	if len(regimenPlan.RegimenSections) > 0 {
		t.Fatal("There should not be any regimen sections for a new patient visit")
	}

	if len(regimenPlan.AllRegimenSteps) != 1 {
		t.Fatal("There should be 1 regimen step existing globally for this doctor")
	}
}

func TestRegimenForPatientVisit_AddingMultipleItemsWithSameText(t *testing.T) {

	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	patientVisitResponse, doctor := setupTestForRegimenCreation(t, testData)

	// add multiple items with the exact same text and ensure that they all get assigned new ids
	regimenPlanRequest := &common.RegimenPlan{}
	regimenPlanRequest.PatientVisitId = encoding.NewObjectId(patientVisitResponse.PatientVisitId)
	regimenPlanRequest.AllRegimenSteps = make([]*common.DoctorInstructionItem, 0)

	for i := 0; i < 5; i++ {
		regimenPlanRequest.AllRegimenSteps = append(regimenPlanRequest.AllRegimenSteps, &common.DoctorInstructionItem{
			Text:  "Regimen Step",
			State: common.STATE_ADDED,
		})

		regimenPlanRequest.RegimenSections = append(regimenPlanRequest.RegimenSections, &common.RegimenSection{
			RegimenName: "test " + strconv.Itoa(i),
			RegimenSteps: []*common.DoctorInstructionItem{&common.DoctorInstructionItem{
				Text:  "Regimen Step",
				State: common.STATE_ADDED,
			},
			},
		})
	}

	regimenPlanResponse := createRegimenPlanForPatientVisit(regimenPlanRequest, testData, doctor, t)
	validateRegimenRequestAgainstResponse(regimenPlanRequest, regimenPlanResponse, t)

}

// The purpose of this test is to ensure that we do not let the client specify text for
// items in the regimen sections that does not match up to what is indicated in the global list, if the
// linkage exists in the global list.
func TestRegimenForPatientVisit_ErrorTextDifferentForLinkedItem(t *testing.T) {

	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	patientVisitResponse, doctor := setupTestForRegimenCreation(t, testData)

	// add multiple items with the exact same text and ensure that they all get assigned new ids
	regimenPlanRequest := &common.RegimenPlan{}
	regimenPlanRequest.PatientVisitId = encoding.NewObjectId(patientVisitResponse.PatientVisitId)
	regimenPlanRequest.AllRegimenSteps = make([]*common.DoctorInstructionItem, 0)

	for i := 0; i < 5; i++ {
		regimenPlanRequest.AllRegimenSteps = append(regimenPlanRequest.AllRegimenSteps, &common.DoctorInstructionItem{
			Text:  "Regimen Step",
			State: common.STATE_ADDED,
		})

		regimenPlanRequest.RegimenSections = append(regimenPlanRequest.RegimenSections, &common.RegimenSection{
			RegimenName: "test " + strconv.Itoa(i),
			RegimenSteps: []*common.DoctorInstructionItem{&common.DoctorInstructionItem{
				Text:  "Regimen Step",
				State: common.STATE_ADDED,
			},
			},
		})
	}

	regimenPlanResponse := createRegimenPlanForPatientVisit(regimenPlanRequest, testData, doctor, t)
	validateRegimenRequestAgainstResponse(regimenPlanRequest, regimenPlanResponse, t)

	regimenPlanRequest = regimenPlanResponse

	// lets go ahead and update each item in the list
	for i := 0; i < 5; i++ {
		regimenPlanRequest.AllRegimenSteps[i].Text = "Updated Regimen Step"
		regimenPlanRequest.AllRegimenSteps[i].State = common.STATE_MODIFIED

		// text cannot be different given that the parent id maps to an item in the global list so this should error out
		regimenPlanRequest.RegimenSections[i].RegimenSteps[0].Text = "Updated Regimen Step " + strconv.Itoa(i)
		regimenPlanRequest.RegimenSections[i].RegimenSteps[0].State = common.STATE_MODIFIED
	}

	doctorRegimenHandler := apiservice.NewDoctorRegimenHandler(testData.DataApi)
	ts := httptest.NewServer(doctorRegimenHandler)
	defer ts.Close()

	requestBody, err := json.Marshal(regimenPlanRequest)
	if err != nil {
		t.Fatal("Unable to marshal request body for adding regimen steps: " + err.Error())
	}

	resp, err := authPost(ts.URL, "application/json", bytes.NewBuffer(requestBody), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make successful request to create regimen for patient visit")
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatal("Expected to get a bad request for when the regimen step's text is different than what its linked to")
	}

}

func TestRegimenForPatientVisit_UpdatingMultipleItemsWithSameText(t *testing.T) {

	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	patientVisitResponse, doctor := setupTestForRegimenCreation(t, testData)

	// add multiple items with the exact same text and ensure that they all get assigned new ids
	regimenPlanRequest := &common.RegimenPlan{}
	regimenPlanRequest.PatientVisitId = encoding.NewObjectId(patientVisitResponse.PatientVisitId)
	regimenPlanRequest.AllRegimenSteps = make([]*common.DoctorInstructionItem, 0)

	for i := 0; i < 5; i++ {
		regimenPlanRequest.AllRegimenSteps = append(regimenPlanRequest.AllRegimenSteps, &common.DoctorInstructionItem{
			Text:  "Regimen Step",
			State: common.STATE_ADDED,
		})

		regimenPlanRequest.RegimenSections = append(regimenPlanRequest.RegimenSections, &common.RegimenSection{
			RegimenName: "test " + strconv.Itoa(i),
			RegimenSteps: []*common.DoctorInstructionItem{&common.DoctorInstructionItem{
				Text:  "Regimen Step",
				State: common.STATE_ADDED,
			},
			},
		})
	}

	regimenPlanResponse := createRegimenPlanForPatientVisit(regimenPlanRequest, testData, doctor, t)
	validateRegimenRequestAgainstResponse(regimenPlanRequest, regimenPlanResponse, t)

	regimenPlanRequest = regimenPlanResponse

	// lets go ahead and update each item in the list
	for i := 0; i < 5; i++ {
		regimenPlanRequest.AllRegimenSteps[i].Text = "Updated Regimen Step"
		regimenPlanRequest.AllRegimenSteps[i].State = common.STATE_MODIFIED

		regimenPlanRequest.RegimenSections[i].RegimenSteps[0].Text = "Updated Regimen Step"
		regimenPlanRequest.RegimenSections[i].RegimenSteps[0].State = common.STATE_MODIFIED
	}

	regimenPlanResponse = createRegimenPlanForPatientVisit(regimenPlanRequest, testData, doctor, t)
	validateRegimenRequestAgainstResponse(regimenPlanRequest, regimenPlanResponse, t)
}

func TestRegimenForPatientVisit_UpdatingItemLinkedToDeletedItem(t *testing.T) {

	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	patientVisitResponse, doctor := setupTestForRegimenCreation(t, testData)

	// add multiple items with the exact same text and ensure that they all get assigned new ids
	regimenPlanRequest := &common.RegimenPlan{}
	regimenPlanRequest.PatientVisitId = encoding.NewObjectId(patientVisitResponse.PatientVisitId)
	regimenPlanRequest.AllRegimenSteps = make([]*common.DoctorInstructionItem, 0)

	for i := 0; i < 5; i++ {
		regimenPlanRequest.AllRegimenSteps = append(regimenPlanRequest.AllRegimenSteps, &common.DoctorInstructionItem{
			Text:  "Regimen Step",
			State: common.STATE_ADDED,
		})

		regimenPlanRequest.RegimenSections = append(regimenPlanRequest.RegimenSections, &common.RegimenSection{
			RegimenName: "test " + strconv.Itoa(i),
			RegimenSteps: []*common.DoctorInstructionItem{&common.DoctorInstructionItem{
				Text:  "Regimen Step",
				State: common.STATE_ADDED,
			},
			},
		})
	}

	regimenPlanResponse := createRegimenPlanForPatientVisit(regimenPlanRequest, testData, doctor, t)
	validateRegimenRequestAgainstResponse(regimenPlanRequest, regimenPlanResponse, t)

	// now lets update the global set of regimen steps in the context of another patient's visit
	patientVisitResponse2, _ := signupAndSubmitPatientVisitForRandomPatient(t, testData, doctor)
	regimenPlanResponse = getRegimenPlanForPatientVisit(testData, doctor, patientVisitResponse2.PatientVisitId, t)

	// lets go ahead and delete one of the items from the regimen step
	regimenPlanRequest = regimenPlanResponse
	regimenPlanRequest.PatientVisitId = encoding.NewObjectId(patientVisitResponse2.PatientVisitId)
	regimenPlanRequest.AllRegimenSteps = regimenPlanRequest.AllRegimenSteps[0:4]

	regimenPlanResponse = createRegimenPlanForPatientVisit(regimenPlanRequest, testData, doctor, t)
	if len(regimenPlanResponse.AllRegimenSteps) != 4 {
		t.Fatalf("Expected there to exist 4 items in the global regimen steps after deleting one of them instead got %d items ", len(regimenPlanResponse.AllRegimenSteps))
	}

	// now, lets go back to the previous patient and attempt to get the regimen plan
	regimenPlanResponse = getRegimenPlanForPatientVisit(testData, doctor, patientVisitResponse.PatientVisitId, t)
	if len(regimenPlanResponse.AllRegimenSteps) != 4 && len(regimenPlanResponse.RegimenSections) != 5 {
		t.Fatalf("Expected 4 items in the global regimen steps and 5 items in the regimen sections instead got %d in global regimen list and %d items in the regimen sections", len(regimenPlanRequest.AllRegimenSteps), len(regimenPlanRequest.RegimenSections))
	}

	// now lets go ahead and try and modify the item in the regimen section
	regimenPlanRequest = regimenPlanResponse
	regimenPlanRequest.RegimenSections[4].RegimenSteps[0].State = common.STATE_MODIFIED
	regimenPlanRequest.PatientVisitId = encoding.NewObjectId(patientVisitResponse.PatientVisitId)
	updatedText := "Updating text for an item linked to deleted item"
	regimenPlanRequest.RegimenSections[4].RegimenSteps[0].Text = updatedText

	regimenPlanResponse = createRegimenPlanForPatientVisit(regimenPlanRequest, testData, doctor, t)
	if len(regimenPlanResponse.AllRegimenSteps) != 4 && len(regimenPlanResponse.RegimenSections) != 5 {
		t.Fatalf("Expected 4 items in the global regimen steps and 5 items in the regimen sections instead got %d in global regimen list and %d items in the regimen sections", len(regimenPlanRequest.AllRegimenSteps), len(regimenPlanRequest.RegimenSections))
	}

	if regimenPlanResponse.RegimenSections[4].RegimenSteps[0].Text != updatedText {
		t.Fatalf("Exepcted text to have updated for item linked to deleted item but it didn't")
	}

	// now lets go ahead and remove the item from the regimen section
	regimenPlanRequest = regimenPlanResponse
	regimenPlanRequest.RegimenSections = regimenPlanRequest.RegimenSections[:4]
	regimenPlanResponse = createRegimenPlanForPatientVisit(regimenPlanRequest, testData, doctor, t)
	validateRegimenRequestAgainstResponse(regimenPlanRequest, regimenPlanResponse, t)
}

// The purpose of this test is to ensure that when regimen steps are updated,
// we are keeping track of the original step that has been modified via a source_id
func TestRegimenForPatientVisit_TrackingSourceId(t *testing.T) {

	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	patientVisitResponse, doctor := setupTestForRegimenCreation(t, testData)

	// adding new regimen steps to the doctor but not to the patient visit
	regimenPlanRequest := &common.RegimenPlan{}
	regimenPlanRequest.PatientVisitId = encoding.NewObjectId(patientVisitResponse.PatientVisitId)

	regimenStep1 := &common.DoctorInstructionItem{}
	regimenStep1.Text = "Regimen Step 1"
	regimenStep1.State = common.STATE_ADDED

	regimenStep2 := &common.DoctorInstructionItem{}
	regimenStep2.Text = "Regimen Step 2"
	regimenStep2.State = common.STATE_ADDED

	regimenPlanRequest.AllRegimenSteps = []*common.DoctorInstructionItem{regimenStep1, regimenStep2}
	regimenPlanResponse := createRegimenPlanForPatientVisit(regimenPlanRequest, testData, doctor, t)
	validateRegimenRequestAgainstResponse(regimenPlanRequest, regimenPlanResponse, t)

	if len(regimenPlanResponse.RegimenSections) > 0 {
		t.Fatal("Regimen section should not exist even though regimen steps were created by doctor")
	}

	// keep track of the source ids of both steps
	sourceId1 := regimenPlanResponse.AllRegimenSteps[0].Id.Int64()
	sourceId2 := regimenPlanResponse.AllRegimenSteps[1].Id.Int64()

	// lets update both steps
	regimenPlanRequest = regimenPlanResponse
	regimenPlanRequest.AllRegimenSteps[0].State = common.STATE_MODIFIED
	regimenPlanRequest.AllRegimenSteps[0].Text = "Updated step 1"
	regimenPlanRequest.AllRegimenSteps[1].State = common.STATE_MODIFIED
	regimenPlanRequest.AllRegimenSteps[1].Text = "Updated step 2"
	regimenPlanResponse = createRegimenPlanForPatientVisit(regimenPlanRequest, testData, doctor, t)
	validateRegimenRequestAgainstResponse(regimenPlanRequest, regimenPlanResponse, t)

	// the source id of the two returned steps should match the source id of the original steps
	var updatedItemSourceId1, updatedItemSourceId2 sql.NullInt64
	if err := testData.DB.QueryRow(`select source_id from dr_regimen_step where id=?`, regimenPlanResponse.AllRegimenSteps[0].Id.Int64()).Scan(&updatedItemSourceId1); err != nil {
		t.Fatalf("Expected the query to get source_id to succeed instead it failed: %s", err)
	}

	if updatedItemSourceId1.Int64 != sourceId1 {
		t.Fatalf("Expected the sourceId retrieved from the updated item (%d) to match the id of the original item (%d)", updatedItemSourceId1.Int64, sourceId1)
	}

	if err := testData.DB.QueryRow(`select source_id from dr_regimen_step where id=?`, regimenPlanResponse.AllRegimenSteps[1].Id.Int64()).Scan(&updatedItemSourceId2); err != nil {
		t.Fatalf("Expected the query to get source_id to succeed instead it failed: %s", err)
	}

	if updatedItemSourceId2.Int64 != sourceId2 {
		t.Fatalf("Expected the sourceId retrieved from the updated item (%d) to match the id of the original item (%d)", updatedItemSourceId2.Int64, sourceId2)
	}

	// lets update again and the source id should still match
	regimenPlanRequest = regimenPlanResponse
	regimenPlanRequest.AllRegimenSteps[0].State = common.STATE_MODIFIED
	regimenPlanRequest.AllRegimenSteps[0].Text = "Updated again step 1"
	regimenPlanRequest.AllRegimenSteps[1].State = common.STATE_MODIFIED
	regimenPlanRequest.AllRegimenSteps[1].Text = "Updated again step 2"
	regimenPlanResponse = createRegimenPlanForPatientVisit(regimenPlanRequest, testData, doctor, t)
	validateRegimenRequestAgainstResponse(regimenPlanRequest, regimenPlanResponse, t)

	// the source id of the two returned steps should match the source id of the original steps
	if err := testData.DB.QueryRow(`select source_id from dr_regimen_step where id=?`, regimenPlanResponse.AllRegimenSteps[0].Id.Int64()).Scan(&updatedItemSourceId1); err != nil {
		t.Fatalf("Expected the query to get source_id to succeed instead it failed: %s", err)
	}

	if updatedItemSourceId1.Int64 != sourceId1 {
		t.Fatalf("Expected the sourceId retrieved from the updated item (%d) to match the id of the original item (%d)", updatedItemSourceId1.Int64, sourceId1)
	}

	if err := testData.DB.QueryRow(`select source_id from dr_regimen_step where id=?`, regimenPlanResponse.AllRegimenSteps[1].Id.Int64()).Scan(&updatedItemSourceId2); err != nil {
		t.Fatalf("Expected the query to get source_id to succeed instead it failed: %s", err)
	}

	if updatedItemSourceId2.Int64 != sourceId2 {
		t.Fatalf("Expected the sourceId retrieved from the updated item (%d) to match the id of the original item (%d)", updatedItemSourceId2.Int64, sourceId2)
	}

}

func setupTestForRegimenCreation(t *testing.T, testData TestData) (*apiservice.PatientVisitResponse, *common.Doctor) {

	// get the current primary doctor
	doctorId := getDoctorIdOfCurrentPrimaryDoctor(testData, t)

	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal("Unable to get doctor from doctor id " + err.Error())
	}
	patientVisitResponse, _ := signupAndSubmitPatientVisitForRandomPatient(t, testData, doctor)
	return patientVisitResponse, doctor
}

func getRegimenPlanForPatientVisit(testData TestData, doctor *common.Doctor, patientVisitId int64, t *testing.T) *common.RegimenPlan {
	doctorRegimenHandler := apiservice.NewDoctorRegimenHandler(testData.DataApi)
	ts := httptest.NewServer(doctorRegimenHandler)
	defer ts.Close()

	resp, err := authGet(ts.URL+"?patient_visit_id="+strconv.FormatInt(patientVisitId, 10), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to get regimen for patient visit: " + err.Error())
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to parse the body of the response for getting the regimen plan: " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unable to make successful call to get regimen plan for patient visit: "+string(body), t)

	doctorRegimenResponse := &common.RegimenPlan{}
	err = json.Unmarshal(body, doctorRegimenResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal body into json object: " + err.Error())
	}

	return doctorRegimenResponse
}

func createRegimenPlanForPatientVisit(doctorRegimenRequest *common.RegimenPlan, testData TestData, doctor *common.Doctor, t *testing.T) *common.RegimenPlan {
	doctorRegimenHandler := apiservice.NewDoctorRegimenHandler(testData.DataApi)
	ts := httptest.NewServer(doctorRegimenHandler)
	defer ts.Close()

	requestBody, err := json.Marshal(doctorRegimenRequest)
	if err != nil {
		t.Fatal("Unable to marshal request body for adding regimen steps: " + err.Error())
	}

	resp, err := authPost(ts.URL, "application/json", bytes.NewBuffer(requestBody), doctor.AccountId.Int64())
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

func validateRegimenRequestAgainstResponse(doctorRegimenRequest, doctorRegimenResponse *common.RegimenPlan, t *testing.T) {

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
	// all regimen steps in the sections should also be present in the global list
	for _, regimenSection := range doctorRegimenResponse.RegimenSections {
		for _, regimenStep := range regimenSection.RegimenSteps {
			if regimenStep.Id.Int64() == 0 {
				t.Fatal("Regimen steps in each section are expected to have an id")
			}
			if regimenStepsMapping[regimenStep.ParentId.Int64()] == false {
				t.Fatalf("There exists a regimen step in a section that is not present in the global list. Id of regimen step %d", regimenStep.Id.Int64Value)
			}
			if regimenStep.ParentId.Int64() == 0 {
				t.Fatal("Regimen steps in each section are expected to link to an item in the global regimen list")
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

	// no two items should have the same parent id in the regimen section
	idsFound = make(map[int64]bool)
	for _, regimenSection := range doctorRegimenResponse.RegimenSections {
		for _, regimenStep := range regimenSection.RegimenSteps {
			if _, ok := idsFound[regimenStep.ParentId.Int64()]; ok {
				t.Fatalf("No two items can have the same parent id")
			}
			idsFound[regimenStep.ParentId.Int64()] = true
		}
	}

	// deleted regimen steps should not show up in the response
	deletedRegimenStepIds := make(map[int64]bool)
	// updated regimen steps should have a different id in the response
	updatedRegimenSteps := make(map[string][]int64)

	for _, regimenStep := range doctorRegimenRequest.AllRegimenSteps {
		switch regimenStep.State {
		case common.STATE_MODIFIED:
			updatedRegimenSteps[regimenStep.Text] = append(updatedRegimenSteps[regimenStep.Text], regimenStep.Id.Int64())
		case common.STATE_DELETED:
			deletedRegimenStepIds[regimenStep.Id.Int64()] = true
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
