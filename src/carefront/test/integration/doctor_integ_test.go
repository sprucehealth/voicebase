package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"carefront/api"
	"carefront/apiservice"
	"carefront/common"
)

func TestDoctorRegistration(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}

	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	SignupRandomTestDoctor(t, testData.DataApi, testData.AuthApi)
}

func TestDoctorAuthentication(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}

	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	_, email, password := SignupRandomTestDoctor(t, testData.DataApi, testData.AuthApi)

	doctorAuthHandler := &apiservice.DoctorAuthenticationHandler{AuthApi: testData.AuthApi, DataApi: testData.DataApi}
	ts := httptest.NewServer(doctorAuthHandler)
	defer ts.Close()
	requestBody := bytes.NewBufferString("email=")
	requestBody.WriteString(email)
	requestBody.WriteString("&password=")
	requestBody.WriteString(password)
	res, err := authPost(ts.URL, "application/x-www-form-urlencoded", requestBody, 0)
	if err != nil {
		t.Fatal("Unable to authenticate doctor " + err.Error())
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal("Unable to read body of response: " + err.Error())
	}
	CheckSuccessfulStatusCode(res, fmt.Sprintf("Unable to make success request to authenticate doctor. Here's the code returned %d and here's the body of the request %s", res.StatusCode, body), t)

	authenticatedDoctorResponse := &apiservice.DoctorAuthenticationResponse{}
	err = json.Unmarshal(body, authenticatedDoctorResponse)
	if err != nil {
		t.Fatal("Unable to parse response from patient authenticated")
	}

	if authenticatedDoctorResponse.Token == "" || authenticatedDoctorResponse.DoctorId == 0 {
		t.Fatal("Doctor not authenticated as expected")
	}
}

func TestDoctorDrugSearch(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}

	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	erx := setupErxAPI(t)

	// ensure that the autcoomplete api returns results
	autocompleteHandler := &apiservice.AutocompleteHandler{ERxApi: erx, Role: api.DOCTOR_ROLE}
	ts := httptest.NewServer(autocompleteHandler)
	defer ts.Close()

	resp, err := authGet(ts.URL+"?query=pro", 0)
	if err != nil {
		t.Fatal("Unable to make a successful query to the autocomplete API")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to parse the body of the response: " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unable to make a successful query to the autocomplete api for the doctor: "+string(body), t)
	autocompleteResponse := &apiservice.AutocompleteResponse{}
	err = json.Unmarshal(body, autocompleteResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal the response from the autocomplete call into a json object as expected: " + err.Error())
	}

	if autocompleteResponse.Suggestions == nil || len(autocompleteResponse.Suggestions) == 0 {
		t.Fatal("Expected suggestions from the autocomplete api but got none")
	}

	for _, suggestion := range autocompleteResponse.Suggestions {
		if suggestion.Title == "" || suggestion.Subtitle == "" || suggestion.DrugInternalName == "" {
			t.Fatalf("Suggestion structure not filled in with data as expected. %q", suggestion)
		}
	}
}

func TestDoctorDiagnosisOfPatientVisit(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		t.Log("Skipping test since there is no database to run test on")
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

	// get patient to submit the visit
	SubmitPatientVisitForPatient(patientSignedupResponse.Patient.PatientId, patientVisitResponse.PatientVisitId, testData, t)

	// doctor now attempts to diagnose patient visit
	diagnosePatientHandler := apiservice.NewDiagnosePatientHandler(testData.DataApi, testData.AuthApi, testData.CloudStorageService)
	ts := httptest.NewServer(diagnosePatientHandler)
	defer ts.Close()

	requestParams := bytes.NewBufferString("?patient_visit_id=")
	requestParams.WriteString(strconv.FormatInt(patientVisitResponse.PatientVisitId, 10))

	resp, err := authGet(ts.URL+requestParams.String(), doctor.AccountId)
	if err != nil {
		t.Fatal("Something went wrong when trying to get diagnoses layout for doctor to diagnose patient visit: " + err.Error())
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to read body of response for getting diagnosis layout for doctor to diagnose patient: " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unable to make successful request for doctor to get layout to diagnose  Reason: "+string(data), t)

	diagnosisResponse := apiservice.GetDiagnosisResponse{}
	err = json.Unmarshal(data, &diagnosisResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal response for diagnosis of patient visit: " + err.Error())
	}

	if diagnosisResponse.DiagnosisLayout == nil || diagnosisResponse.DiagnosisLayout.PatientVisitId != patientVisitResponse.PatientVisitId {
		t.Fatal("Diagnosis response not as expected")
	}

	// Now, actually diagnose the patient visit and check the response to ensure that the doctor diagnosis was returned in the response
	// prepapre a response for the doctor
	diagnosisQuestionId, severityQuestionId, acneTypeQuestionId := submitPatientVisitDiagnosis(patientVisitResponse.PatientVisitId, doctor, testData, t)

	// now, get diagnosis layout again and check to ensure that the doctor successfully diagnosed the patient with the expected answers
	resp, err = authGet(ts.URL+requestParams.String(), doctor.AccountId)
	if err != nil {
		t.Fatal(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to read body of request to get diagnosis layout after submitting diagnosis: " + err.Error())
	}

	err = json.Unmarshal(body, &diagnosisResponse)
	if err != nil {
		t.Fatal("Unable to marshal response for diagnosis of patient visit after doctor submitted diagnosis: " + err.Error())
	}

	if diagnosisResponse.DiagnosisLayout == nil || diagnosisResponse.DiagnosisLayout.PatientVisitId != patientVisitResponse.PatientVisitId {
		t.Fatal("Diagnosis response not as expected after doctor submitted diagnosis")
	}

	for _, section := range diagnosisResponse.DiagnosisLayout.InfoIntakeLayout.Sections {
		for _, question := range section.Questions {

			for _, doctorResponse := range question.DoctorAnswers {
				switch doctorResponse.QuestionId {
				case diagnosisQuestionId:
					if doctorResponse.PotentialAnswerId != 102 {
						t.Fatalf("Doctor response to question id %d expectd to have id %d but has id %d", doctorResponse.QuestionId, 102, doctorResponse.PotentialAnswerId)
					}
				case severityQuestionId:
					if doctorResponse.PotentialAnswerId != 107 {
						t.Fatalf("Doctor response to question id %d expectd to have id %d but has id %d", doctorResponse.QuestionId, 107, doctorResponse.PotentialAnswerId)
					}

				case acneTypeQuestionId:
					if doctorResponse.PotentialAnswerId != 109 && doctorResponse.PotentialAnswerId != 114 && doctorResponse.PotentialAnswerId != 113 {
						t.Fatalf("Doctor response to question id %d expectd to have any of ids %s but instead has id %d", doctorResponse.QuestionId, "(109,114,113)", doctorResponse.PotentialAnswerId)
					}

				}
			}
		}
	}

	// check if the diagnosis summary exists for the patient visit
	diagnosisSummaryHandler := &apiservice.DiagnosisSummaryHandler{DataApi: testData.DataApi}
	ts = httptest.NewServer(diagnosisSummaryHandler)
	defer ts.Close()

	resp, err = authGet(ts.URL+"?patient_visit_id="+strconv.FormatInt(patientVisitResponse.PatientVisitId, 10), doctor.AccountId)
	if err != nil {
		t.Fatal("Unable to make call to get diagnosis summary for patient visit: " + err.Error())
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to parse body of response when trying to get diagnosis summary for patient visit")
	}

	CheckSuccessfulStatusCode(resp, "Unable to make successful call to get diagnosis summary for patient visit "+string(respBody), t)

	getDiagnosisSummaryResponse := &common.DiagnosisSummary{}
	err = json.Unmarshal(respBody, getDiagnosisSummaryResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal response into json object : " + err.Error())
	}

	if getDiagnosisSummaryResponse.Summary == "" {
		t.Fatal("Expected summary for patient visit to exist but instead got nothing")
	}
}

func TestDoctorSubmissionOfPatientVisitReview(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		t.Log("Skipping test since there is no database to run test on")
		return
	}
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	patientSignedupResponse := SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)

	doctorId := getDoctorIdOfCurrentPrimaryDoctor(testData, t)

	// get patient to start a visit
	patientVisitResponse := CreatePatientVisitForPatient(patientSignedupResponse.Patient.PatientId, testData, t)

	// get patient to submit the visit
	SubmitPatientVisitForPatient(patientSignedupResponse.Patient.PatientId, patientVisitResponse.PatientVisitId, testData, t)

	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal("Unable to get doctor object from id: " + err.Error())
	}

	// attempt to submit the patient visit review here. It should fail
	doctorSubmitPatientVisitReviewHandler := &apiservice.DoctorSubmitPatientVisitReviewHandler{DataApi: testData.DataApi}
	ts := httptest.NewServer(doctorSubmitPatientVisitReviewHandler)
	defer ts.Close()

	resp, err := authPost(ts.URL, "application/x-www-form-urlencoded", bytes.NewBufferString("patient_visit_id="+strconv.FormatInt(patientVisitResponse.PatientVisitId, 10)), doctor.AccountId)
	if err != nil {
		t.Fatal("Unable to make a call to submit the patient visit review : " + err.Error())
	}

	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to parse the response body for the call to submit patient visit review: " + err.Error())
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status code to be %d but got %d instead. The call should have failed because the patient visit is not being REVIEWED by the doctor yet. ", http.StatusBadRequest, resp.StatusCode)
	}

	// get the doctor to start reviewing the patient visit
	doctorPatientVisitReviewHandler := &apiservice.DoctorPatientVisitReviewHandler{DataApi: testData.DataApi, LayoutStorageService: testData.CloudStorageService, PatientPhotoStorageService: testData.CloudStorageService}
	ts2 := httptest.NewServer(doctorPatientVisitReviewHandler)
	defer ts2.Close()

	resp, err = authGet(ts2.URL+"?patient_visit_id="+strconv.FormatInt(patientVisitResponse.PatientVisitId, 10), doctor.AccountId)
	if err != nil {
		t.Fatal("Unable to get the doctor to start reviewing the patient visit: " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unable to make a successful call for doctor to start reviewing patient visti", t)

	// attempt to submit the patient visit review here. It should work
	resp, err = authPost(ts.URL, "application/x-www-form-urlencoded", bytes.NewBufferString("patient_visit_id="+strconv.FormatInt(patientVisitResponse.PatientVisitId, 10)), doctor.AccountId)
	if err != nil {
		t.Fatal("Unable to make successful call to submit patient visit review")
	}

	CheckSuccessfulStatusCode(resp, "Unable to make successful call to submit patient visit review", t)

	patientVisit, err := testData.DataApi.GetPatientVisitFromId(patientVisitResponse.PatientVisitId)
	if err != nil {
		t.Fatal("Unable to get patient visit given id: " + err.Error())
	}

	if patientVisit.Status != api.CASE_STATUS_TREATED {
		t.Fatalf("Expected the status to be %s but status is %s", api.CASE_STATUS_CLOSED, patientVisit.Status)
	}
}

func TestDoctorAddingOfFollowUpForPatientVisit(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		t.Log("Skipping test since there is no database to run test on")
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

	// get patient to submit the visit
	SubmitPatientVisitForPatient(patientSignedupResponse.Patient.PatientId, patientVisitResponse.PatientVisitId, testData, t)

	// lets add a follow up time for 1 week from now
	doctorFollowupHandler := apiservice.NewPatientVisitFollowUpHandler(testData.DataApi)
	ts := httptest.NewServer(doctorFollowupHandler)
	defer ts.Close()

	requestBody := fmt.Sprintf("patient_visit_id=%d&follow_up_unit=week&follow_up_value=1", patientVisitResponse.PatientVisitId)
	resp, err := authPost(ts.URL, "application/x-www-form-urlencoded", bytes.NewBufferString(requestBody), doctor.AccountId)
	if err != nil {
		t.Fatal("Unable to make successful call to add follow up time for patient visit: " + err.Error())
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to read the response body: " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unable to make successful call to add follow up for patient visit: "+string(body), t)

	// lets get the follow up time back
	resp, err = authGet(ts.URL+"?patient_visit_id="+strconv.FormatInt(patientVisitResponse.PatientVisitId, 10), doctor.AccountId)
	if err != nil {
		t.Fatal("Unable to make successful call to get follow up time for patient visit: " + err.Error())
	}

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to parse body of the response to get follow up time for patient visit: " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unable to make successful call to get follow up time for patient visit: "+string(body), t)

	patientVisitFollowupResponse := &apiservice.PatientVisitFollowupResponse{}
	err = json.Unmarshal(body, patientVisitFollowupResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal the response into a json object: " + err.Error())
	}

	oneWeekFromNow := time.Now().Add(7 * 24 * 60 * time.Minute)
	year, month, day := oneWeekFromNow.Date()
	year1, month1, day1 := patientVisitFollowupResponse.FollowUpTime.Date()

	if year != year1 || month1 != month || math.Abs(float64(day1-day)) > 2 {
		t.Fatalf("Expected date to follow up time returned to be around %d/%d/%d, but got %d/%d/%d instead", year, month, day, year1, month1, day1)
	}
}
