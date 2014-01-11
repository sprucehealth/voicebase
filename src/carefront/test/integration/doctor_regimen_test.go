package integration

import (
	"bytes"
	"carefront/apiservice"
	"carefront/common"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestRegimenForPatientVisit(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}

	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	patientSignedupResponse := SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)

	// get the current primary doctor
	doctorId := getDoctorIdOfCurrentPrimaryDoctor(testData, t)

	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal("Unable to get doctor from doctor id " + err.Error())
	}

	// get patient to start a visit
	patientVisitResponse := CreatePatientVisitForPatient(patientSignedupResponse.Patient.PatientId, testData, t)

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
	regimenPlanRequest.PatientVisitId = patientVisitResponse.PatientVisitId

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
	regimenSection.RegimenSteps = []*common.DoctorInstructionItem{regimenPlanRequest.AllRegimenSteps[0]}

	regimenSection2 := &common.RegimenSection{}
	regimenSection2.RegimenName = "night"
	regimenSection2.RegimenSteps = []*common.DoctorInstructionItem{regimenPlanRequest.AllRegimenSteps[1]}

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
	regimenPlanRequest.AllRegimenSteps[1].State = common.STATE_DELETED
	regimenPlanResponse = createRegimenPlanForPatientVisit(regimenPlanRequest, testData, doctor, t)
	validateRegimenRequestAgainstResponse(regimenPlanRequest, regimenPlanResponse, t)
	if len(regimenPlanResponse.AllRegimenSteps) != 1 {
		t.Fatal("Should only have 1 regimen step given that we just deleted one from the list")
	}

	SubmitPatientVisitForPatient(patientSignedupResponse.Patient.PatientId, patientVisitResponse.PatientVisitId, testData, t)

	// get patient to start a visit
	patientSignedupResponse = SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patientVisitResponse = CreatePatientVisitForPatient(patientSignedupResponse.Patient.PatientId, testData, t)

	regimenPlan = getRegimenPlanForPatientVisit(testData, doctor, patientVisitResponse.PatientVisitId, t)
	if len(regimenPlan.RegimenSections) > 0 {
		t.Fatal("There should not be any regimen sections for a new patient visit")
	}

	if len(regimenPlan.AllRegimenSteps) != 1 {
		t.Fatal("There should be 1 regimen step existing globally for this doctor")
	}
}

func getRegimenPlanForPatientVisit(testData TestData, doctor *common.Doctor, patientVisitId int64, t *testing.T) *common.RegimenPlan {
	doctorRegimenHandler := apiservice.NewDoctorRegimenHandler(testData.DataApi)
	doctorRegimenHandler.AccountIdFromAuthToken(doctor.AccountId)
	ts := httptest.NewServer(doctorRegimenHandler)

	resp, err := http.Get(ts.URL + "?patient_visit_id=" + strconv.FormatInt(patientVisitId, 10))
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
	doctorRegimenHandler.AccountIdFromAuthToken(doctor.AccountId)
	ts := httptest.NewServer(doctorRegimenHandler)

	requestBody, err := json.Marshal(doctorRegimenRequest)
	if err != nil {
		t.Fatal("Unable to marshal request body for adding regimen steps: " + err.Error())
	}

	resp, err := http.Post(ts.URL, "application/json", bytes.NewBuffer(requestBody))
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
		if regimenStep.Id == 0 {
			t.Fatal("Regimen steps in the response are expected to have an id")
		}
		regimenStepsMapping[regimenStep.Id] = true
	}

	// all regimen steps in the regimen sections should have an id in the response
	// all regimen steps in the sections should also be present in the global list
	for _, regimenSection := range doctorRegimenResponse.RegimenSections {
		for _, regimenStep := range regimenSection.RegimenSteps {
			if regimenStep.Id == 0 {
				t.Fatal("Regimen steps in each section are expected to have an id")
			}
			if regimenStepsMapping[regimenStep.Id] == false {
				t.Fatalf("There exists a regimen step in a section that is not present in the global list. Id of regimen step %d", regimenStep.Id)
			}
		}
	}

	// deleted regimen steps should not show up in the response
	deletedRegimenStepIds := make(map[int64]bool)
	// updated regimen steps should have a different id in the response
	updatedRegimenSteps := make(map[string]int64)

	for _, regimenStep := range doctorRegimenRequest.AllRegimenSteps {
		switch regimenStep.State {
		case common.STATE_MODIFIED:
			updatedRegimenSteps[regimenStep.Text] = regimenStep.Id
		case common.STATE_DELETED:
			deletedRegimenStepIds[regimenStep.Id] = true
		}
	}

	for _, regimenStep := range doctorRegimenResponse.AllRegimenSteps {
		if updatedRegimenSteps[regimenStep.Text] != 0 {
			if regimenStep.Id == updatedRegimenSteps[regimenStep.Text] {
				t.Fatalf("Expected an updated regimen step to have a different id in the response. Id = %d", regimenStep.Id)
			}
		}

		if deletedRegimenStepIds[regimenStep.Id] == true {
			t.Fatalf("Expected regimen step %d to have been deleted and not in the response", regimenStep.Id)
		}
	}
}
