package integration

import (
	"bytes"
	"carefront/api"
	"carefront/apiservice"
	"carefront/libs/erx"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
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
	requestBody := bytes.NewBufferString("email=")
	requestBody.WriteString(email)
	requestBody.WriteString("&password=")
	requestBody.WriteString(password)
	res, err := http.Post(ts.URL, "application/x-www-form-urlencoded", requestBody)
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

	SignupRandomTestDoctor(t, testData.DataApi, testData.AuthApi)

	clinicKey := os.Getenv("DOSESPOT_CLINIC_KEY")
	userId := os.Getenv("DOSESPOT_USER_ID")
	clinicId := os.Getenv("DOSESPOT_CLINIC_ID")

	if clinicKey == "" {
		t.Log("WARNING: skipping doctor drug search test since the dosespot ids are not present as environment variables")
		t.SkipNow()
	}

	erx := erx.NewDoseSpotService(clinicId, clinicKey, userId)

	// ensure that the autcoomplete api returns results
	autocompleteHandler := &apiservice.AutocompleteHandler{ERxApi: erx, Role: api.DOCTOR_ROLE}
	ts := httptest.NewServer(autocompleteHandler)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "?query=pro")
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
		if suggestion.Title == "" || suggestion.Subtitle == "" || suggestion.InternalName == "" {
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
	var doctorId int64
	err := testData.DB.QueryRow(`select provider_id from care_provider_state_elligibility 
							inner join provider_role on provider_role_id = provider_role.id 
							inner join care_providing_state on care_providing_state_id = care_providing_state.id
							where provider_tag='DOCTOR' and care_providing_state.state = 'CA'`).Scan(&doctorId)
	if err != nil {
		t.Fatal("Unable to query for doctor that is elligible to diagnose in CA: " + err.Error())
	}

	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal("Unable to get doctor from doctor id " + err.Error())
	}

	// get patient to start a visit
	patientVisitResponse := GetPatientVisitForPatient(patientSignedupResponse.PatientId, testData, t)

	// get patient to submit the visit
	SubmitPatientVisitForPatient(patientSignedupResponse.PatientId, patientVisitResponse.PatientVisitId, testData, t)

	// doctor now attempts to diagnose patient visit
	diagnosePatientHandler := apiservice.NewDiagnosePatientHandler(testData.DataApi, testData.AuthApi, testData.CloudStorageService)
	diagnosePatientHandler.AccountIdFromAuthToken(doctor.AccountId)
	ts := httptest.NewServer(diagnosePatientHandler)
	defer ts.Close()

	requestParams := bytes.NewBufferString("?patient_visit_id=")
	requestParams.WriteString(strconv.FormatInt(patientVisitResponse.PatientVisitId, 10))
	request, err := http.NewRequest("GET", ts.URL+requestParams.String(), nil)

	if err != nil {
		t.Fatal("Something went wrong when trying to setup the GET request for diagnosis layout :" + err.Error())
	}

	client := http.Client{}
	resp, err := client.Do(request)
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
	answerIntakeRequestBody := &apiservice.AnswerIntakeRequestBody{}
	answerIntakeRequestBody.PatientVisitId = diagnosisResponse.DiagnosisLayout.PatientVisitId

	diagnosisQuestionId, _, _, _, _, _, _, err := testData.DataApi.GetQuestionInfo("q_acne_diagnosis", 1)
	severityQuestionId, _, _, _, _, _, _, err := testData.DataApi.GetQuestionInfo("q_acne_severity", 1)
	acneTypeQuestionId, _, _, _, _, _, _, err := testData.DataApi.GetQuestionInfo("q_acne_type", 1)

	if err != nil {
		t.Fatal("Unable to get the questionIds for the question tags requested for the doctor to diagnose patient visit")
	}

	answerToQuestionItem := &apiservice.AnswerToQuestionItem{}
	answerToQuestionItem.QuestionId = diagnosisQuestionId
	answerToQuestionItem.AnswerIntakes = []*apiservice.AnswerItem{&apiservice.AnswerItem{PotentialAnswerId: 102}}

	answerToQuestionItem2 := &apiservice.AnswerToQuestionItem{}
	answerToQuestionItem2.QuestionId = severityQuestionId
	answerToQuestionItem2.AnswerIntakes = []*apiservice.AnswerItem{&apiservice.AnswerItem{PotentialAnswerId: 107}}

	answerToQuestionItem3 := &apiservice.AnswerToQuestionItem{}
	answerToQuestionItem3.QuestionId = acneTypeQuestionId
	answerToQuestionItem3.AnswerIntakes = []*apiservice.AnswerItem{&apiservice.AnswerItem{PotentialAnswerId: 109}, &apiservice.AnswerItem{PotentialAnswerId: 114}, &apiservice.AnswerItem{PotentialAnswerId: 113}}

	answerIntakeRequestBody.Questions = []*apiservice.AnswerToQuestionItem{answerToQuestionItem, answerToQuestionItem2, answerToQuestionItem3}

	requestData, err := json.Marshal(answerIntakeRequestBody)
	if err != nil {
		t.Fatal("Unable to marshal request body")
	}

	req, _ := http.NewRequest("POST", ts.URL, bytes.NewBuffer(requestData))
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	if err != nil {
		t.Fatal("Unable to successfully submit the diagnosis of a patient visit: " + err.Error())
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to read body of response on submitting diagnosis of patient visit : " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unable to submit diagnosis of patient visit "+string(body), t)

	// now, get diagnosis layout again and check to ensure that the doctor successfully diagnosed the patient with the expected answers
	resp, err = client.Do(request)
	body, err = ioutil.ReadAll(resp.Body)
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
			// each question should be answered
			if question.DoctorAnswers == nil || len(question.DoctorAnswers) == 0 {
				t.Fatalf("Expected a response from the doctor to question %d but not present", question.QuestionId)
			}
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

}
