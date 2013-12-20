package integration

import (
	"bytes"
	"carefront/apiservice"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
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
}
