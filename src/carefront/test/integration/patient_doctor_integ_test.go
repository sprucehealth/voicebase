package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"carefront/apiservice"
	"carefront/common"
)

func TestPatientVisitReview(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	signedupPatientResponse := SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patientVisitResponse := CreatePatientVisitForPatient(signedupPatientResponse.Patient.PatientId, testData, t)
	SubmitPatientVisitForPatient(signedupPatientResponse.Patient.PatientId, patientVisitResponse.PatientVisitId, testData, t)

	patient, err := testData.DataApi.GetPatientFromId(signedupPatientResponse.Patient.PatientId)
	if err != nil {
		t.Fatal("Unable to get patient from id: " + err.Error())
	}

	// try getting the patient visit review for this patient visit and it should fail
	patientVisitReviewHandler := &apiservice.PatientVisitReviewHandler{DataApi: testData.DataApi}
	ts := httptest.NewServer(patientVisitReviewHandler)
	defer ts.Close()

	resp, err := authGet(ts.URL+"?patient_visit_id="+strconv.FormatInt(patientVisitResponse.PatientVisitId, 10), patient.AccountId)
	if err != nil {
		t.Fatal("Unable to make call to get the patient visit review for patient visit: " + err.Error())
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected to get %d for call to get patient visit review but instead got %d", http.StatusBadRequest, resp.StatusCode)
	}

	doctorId := getDoctorIdOfCurrentPrimaryDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal("Unable to get doctor from id: " + err.Error())
	}

	// now lets go ahead and get doctor to start reviewing the patient visit and then submit the patient visit review
	doctorPatientVisitReviewHandler := &apiservice.DoctorPatientVisitReviewHandler{DataApi: testData.DataApi, LayoutStorageService: testData.CloudStorageService, PatientPhotoStorageService: testData.CloudStorageService}
	ts2 := httptest.NewServer(doctorPatientVisitReviewHandler)
	defer ts2.Close()
	resp, err = authGet(ts2.URL+"?patient_visit_id="+strconv.FormatInt(patientVisitResponse.PatientVisitId, 10), doctor.AccountId)
	if err != nil {
		t.Fatal("Unable to make call to get patient visit review for patient: " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unable to make successful call for doctor to start reviewing patient visit", t)

	// once the doctor has started reviewing the case, lets go ahead and get the doctor to close the case with no diagnosis
	doctorSubmitPatientVisitReviewHandler := &apiservice.DoctorSubmitPatientVisitReviewHandler{DataApi: testData.DataApi}
	ts3 := httptest.NewServer(doctorSubmitPatientVisitReviewHandler)
	defer ts3.Close()

	resp, err = authPost(ts3.URL, "application/x-www-form-urlencoded", bytes.NewBufferString("patient_visit_id="+strconv.FormatInt(patientVisitResponse.PatientVisitId, 10)), doctor.AccountId)
	if err != nil {
		t.Fatal("Unable to make call to close patient visit " + err.Error())
	}
	CheckSuccessfulStatusCode(resp, "Unable to make successful call to close the patient visit", t)

	// now, lets try and get the patient visit review again
	resp, err = authGet(ts2.URL+"?patient_visit_id="+strconv.FormatInt(patientVisitResponse.PatientVisitId, 10), doctor.AccountId)
	if err != nil {
		t.Fatal("Unable to make call to get patient visit review: " + err.Error())
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to parse body of response: " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unable to make successful call to get patient visit review: "+string(respBody), t)

	patientVisitReviewResponse := &apiservice.PatientVisitReviewResponse{}
	err = json.Unmarshal(respBody, patientVisitReviewResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal response body in to json object: " + err.Error())
	}

	if patientVisitReviewResponse.DiagnosisSummary != nil || patientVisitReviewResponse.Treatments != nil || patientVisitReviewResponse.RegimenPlan != nil ||
		patientVisitReviewResponse.Advice != nil || patientVisitReviewResponse.Followup != nil {
		t.Fatal("Expected there to exist no review for this patient visit, but some parts of it do exist")
	}

	patientVisitResponse = GetPatientVisitForPatient(patient.PatientId, testData, t)

	// start a new patient visit
	signedupPatientResponse = SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patientVisitResponse = CreatePatientVisitForPatient(signedupPatientResponse.Patient.PatientId, testData, t)
	SubmitPatientVisitForPatient(signedupPatientResponse.Patient.PatientId, patientVisitResponse.PatientVisitId, testData, t)

	// get doctor to start reviewing it
	StartReviewingPatientVisit(patientVisitResponse.PatientVisitId, doctor, testData, t)

	//
	//
	// SUBMIT DIAGNOSIS
	//
	//

	submitPatientVisitDiagnosis(patientVisitResponse.PatientVisitId, doctor, testData, t)

	//
	//
	// SUBMIT TREATMENT PLAN
	//
	//
	// doctor now attempts to add a couple treatments for patient
	treatment1 := &common.Treatment{}
	treatment1.DrugInternalName = "Advil"
	treatment1.DosageStrength = "10 mg"
	treatment1.DispenseValue = 1
	treatment1.DispenseUnitId = 26
	treatment1.NumberRefills = 1
	treatment1.SubstitutionsAllowed = true
	treatment1.DaysSupply = 1
	treatment1.OTC = true
	treatment1.PharmacyNotes = "testing pharmacy notes"
	treatment1.PatientInstructions = "patient instructions"
	drugDBIds := make(map[string]string)
	drugDBIds["drug_db_id_1"] = "12315"
	drugDBIds["drug_db_id_2"] = "124"
	treatment1.DrugDBIds = drugDBIds

	treatment2 := &common.Treatment{}
	treatment2.DrugInternalName = "Advil 2"
	treatment2.DosageStrength = "100 mg"
	treatment2.DispenseValue = 2
	treatment2.DispenseUnitId = 27
	treatment2.NumberRefills = 3
	treatment2.SubstitutionsAllowed = false
	treatment2.DaysSupply = 12
	treatment2.OTC = false
	treatment2.PharmacyNotes = "testing pharmacy notes 2"
	treatment2.PatientInstructions = "patient instructions 2"
	drugDBIds = make(map[string]string)
	drugDBIds["drug_db_id_3"] = "12414"
	drugDBIds["drug_db_id_4"] = "214"
	treatment2.DrugDBIds = drugDBIds

	treatments := []*common.Treatment{treatment1, treatment2}

	addAndGetTreatmentsForPatientVisit(testData, treatments, doctor.AccountId, patientVisitResponse.PatientVisitId, t)

	//
	//
	// SUBMIT REGIMEN PLAN
	//
	//
	regimenPlanRequest := &common.RegimenPlan{}
	regimenPlanRequest.PatientVisitId = patientVisitResponse.PatientVisitId

	regimenStep1 := &common.DoctorInstructionItem{}
	regimenStep1.Text = "Regimen Step 1"
	regimenStep1.State = common.STATE_ADDED

	regimenStep2 := &common.DoctorInstructionItem{}
	regimenStep2.Text = "Regimen Step 2"
	regimenStep2.State = common.STATE_ADDED
	regimenPlanRequest.AllRegimenSteps = []*common.DoctorInstructionItem{regimenStep1, regimenStep2}

	regimenSection := &common.RegimenSection{}
	regimenSection.RegimenName = "morning"
	regimenSection.RegimenSteps = []*common.DoctorInstructionItem{regimenPlanRequest.AllRegimenSteps[0]}

	regimenSection2 := &common.RegimenSection{}
	regimenSection2.RegimenName = "night"
	regimenSection2.RegimenSteps = []*common.DoctorInstructionItem{regimenPlanRequest.AllRegimenSteps[1]}

	regimenPlanRequest.RegimenSections = []*common.RegimenSection{regimenSection, regimenSection2}
	regimenPlanResponse := createRegimenPlanForPatientVisit(regimenPlanRequest, testData, doctor, t)
	validateRegimenRequestAgainstResponse(regimenPlanRequest, regimenPlanResponse, t)

	//
	//
	// SUBMIT ADVICE
	//
	//
	// lets go ahead and add a couple of advice points
	advicePoint1 := &common.DoctorInstructionItem{Text: "Advice point 1", State: common.STATE_ADDED}
	advicePoint2 := &common.DoctorInstructionItem{Text: "Advice point 2", State: common.STATE_ADDED}

	// lets go ahead and create a request for this patient visit
	doctorAdviceRequest := &common.Advice{}
	doctorAdviceRequest.AllAdvicePoints = []*common.DoctorInstructionItem{advicePoint1, advicePoint2}
	doctorAdviceRequest.SelectedAdvicePoints = doctorAdviceRequest.AllAdvicePoints
	doctorAdviceRequest.PatientVisitId = patientVisitResponse.PatientVisitId

	doctorAdviceResponse := updateAdvicePointsForPatientVisit(doctorAdviceRequest, testData, doctor, t)
	validateAdviceRequestAgainstResponse(doctorAdviceRequest, doctorAdviceResponse, t)

	//
	//
	// SUBMIT FOLLOW UP
	//
	//

	// lets add a follow up time for 1 week from now
	doctorFollowupHandler := apiservice.NewPatientVisitFollowUpHandler(testData.DataApi)
	ts4 := httptest.NewServer(doctorFollowupHandler)
	defer ts4.Close()

	requestBody := fmt.Sprintf("patient_visit_id=%d&follow_up_unit=week&follow_up_value=1", patientVisitResponse.PatientVisitId)
	resp, err = authPost(ts4.URL, "application/x-www-form-urlencoded", bytes.NewBufferString(requestBody), doctor.AccountId)
	if err != nil {
		t.Fatal("Unable to make successful call to add follow up time for patient visit: " + err.Error())
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to read the response body: " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unable to make successful call to add follow up for patient visit: "+string(body), t)

	//
	//
	// SUBMIT VISIT FOR PATIENT VISIT REVIEW
	//
	//

	// get doctor to submit the patient visit review
	resp, err = authPost(ts3.URL, "application/x-www-form-urlencoded", bytes.NewBufferString("patient_visit_id="+strconv.FormatInt(patientVisitResponse.PatientVisitId, 10)), doctor.AccountId)
	if err != nil {
		t.Fatal("Unable to make call to close patient visit " + err.Error())
	}
	CheckSuccessfulStatusCode(resp, "Unable to make successful call to close the patient visit", t)

	//
	//
	// GET PATIENT VISIT REVIEW FOR PATIENT
	//
	//
	patient, err = testData.DataApi.GetPatientFromId(signedupPatientResponse.Patient.PatientId)
	if err != nil {
		t.Fatal("Unable to get the patient object given the id: " + err.Error())
	}
	resp, err = authGet(ts.URL+"?patient_visit_id="+strconv.FormatInt(patientVisitResponse.PatientVisitId, 10), patient.AccountId)
	if err != nil {
		t.Fatal("Unable to make call to get patient visit review: " + err.Error())
	}

	respBody, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to parse body of response: " + err.Error())

	}

	CheckSuccessfulStatusCode(resp, "Unable to make successful call to get patient visit review: "+string(respBody), t)

	patientVisitReviewResponse = &apiservice.PatientVisitReviewResponse{}
	err = json.Unmarshal(respBody, patientVisitReviewResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal response body in to json object: " + err.Error())
	}

	patientVisitResponse = GetPatientVisitForPatient(patient.PatientId, testData, t)

	if patientVisitReviewResponse.DiagnosisSummary == nil || patientVisitReviewResponse.Treatments == nil || patientVisitReviewResponse.RegimenPlan == nil ||
		patientVisitReviewResponse.Advice == nil || patientVisitReviewResponse.Followup == nil {
		t.Fatal("Expected there to exist all sections of the review")
	}
}
