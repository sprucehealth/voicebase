package integration

import (
	"bytes"
	"carefront/libs/maps"
	thriftapi "carefront/thrift/api"
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"fmt"

	"carefront/api"
	"carefront/apiservice"
)

func SignupRandomTestPatient(t *testing.T, dataApi api.DataAPI, authApi thriftapi.Auth) *apiservice.PatientSignedupResponse {
	authHandler := &apiservice.SignupPatientHandler{AuthApi: authApi, DataApi: dataApi, MapsApi: maps.GoogleMapsService(0)}
	ts := httptest.NewServer(authHandler)
	defer ts.Close()

	requestBody := bytes.NewBufferString("first_name=Test&last_name=Test&email=")
	requestBody.WriteString(strconv.FormatInt(rand.Int63(), 10))
	requestBody.WriteString("@example.com&password=12345&dob=11/08/1987&zip_code=94115&phone=123455115&gender=male")
	res, err := http.Post(ts.URL, "application/x-www-form-urlencoded", requestBody)
	if err != nil {
		t.Fatal("Unable to make post request for registering patient: " + err.Error())
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal("Unable to read body of response: " + err.Error())
	}
	CheckSuccessfulStatusCode(res, fmt.Sprintf("Unable to make success request to signup patient. Here's the code returned %d and here's the body of the request %s", res.StatusCode, body), t)

	signedupPatientResponse := &apiservice.PatientSignedupResponse{}
	err = json.Unmarshal(body, signedupPatientResponse)
	if err != nil {
		t.Fatal("Unable to parse response from patient signed up")
	}
	return signedupPatientResponse
}

func GetPatientVisitForPatient(PatientId int64, testData TestData, t *testing.T) *apiservice.PatientVisitResponse {
	patientVisitHandler := apiservice.NewPatientVisitHandler(testData.DataApi, testData.AuthApi,
		testData.CloudStorageService, testData.CloudStorageService, nil, "")
	patient, err := testData.DataApi.GetPatientFromId(PatientId)
	if err != nil {
		t.Fatal("Unable to get patient information given the patient id: " + err.Error())
	}

	patientVisitHandler.AccountIdFromAuthToken(patient.AccountId)
	ts := httptest.NewServer(patientVisitHandler)
	defer ts.Close()

	// register a patient visit for this patient
	client := &http.Client{}
	req, _ := http.NewRequest("GET", ts.URL, nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal("Unable to get the patient visit id")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to read body of the response for the new patient visit call: " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unsuccessful call to register new patient visit: "+string(body), t)

	patientVisitResponse := &apiservice.PatientVisitResponse{}
	err = json.Unmarshal(body, patientVisitResponse)
	if err != nil {
		t.Fatal("Unable to unmarshall response body into patient visit response: " + err.Error())
	}

	return patientVisitResponse
}

func CreatePatientVisitForPatient(PatientId int64, testData TestData, t *testing.T) *apiservice.PatientVisitResponse {
	patientVisitHandler := apiservice.NewPatientVisitHandler(testData.DataApi, testData.AuthApi,
		testData.CloudStorageService, testData.CloudStorageService, nil, "")
	patient, err := testData.DataApi.GetPatientFromId(PatientId)
	if err != nil {
		t.Fatal("Unable to get patient information given the patient id: " + err.Error())
	}

	patientVisitHandler.AccountIdFromAuthToken(patient.AccountId)
	ts := httptest.NewServer(patientVisitHandler)
	defer ts.Close()

	// register a patient visit for this patient
	client := &http.Client{}
	req, _ := http.NewRequest("POST", ts.URL, nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal("Unable to get the patient visit id")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to read body of the response for the new patient visit call: " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unsuccessful call to register new patient visit: "+string(body), t)

	patientVisitResponse := &apiservice.PatientVisitResponse{}
	err = json.Unmarshal(body, patientVisitResponse)
	if err != nil {
		t.Fatal("Unable to unmarshall response body into patient visit response: " + err.Error())
	}

	return patientVisitResponse
}

func SubmitPatientVisitForPatient(PatientId, PatientVisitId int64, testData TestData, t *testing.T) {
	patientVisitHandler := apiservice.NewPatientVisitHandler(testData.DataApi, testData.AuthApi,
		testData.CloudStorageService, testData.CloudStorageService, nil, "")
	patient, err := testData.DataApi.GetPatientFromId(PatientId)
	if err != nil {
		t.Fatal("Unable to get patient information given the patient id: " + err.Error())
	}

	patientVisitHandler.AccountIdFromAuthToken(patient.AccountId)
	ts := httptest.NewServer(patientVisitHandler)
	defer ts.Close()
	buffer := bytes.NewBufferString("patient_visit_id=")
	buffer.WriteString(strconv.FormatInt(PatientVisitId, 10))

	client := &http.Client{}
	req, err := http.NewRequest("PUT", ts.URL, buffer)
	if err != nil {
		t.Fatal("Unable to create request to submit patient visit")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)

	if err != nil {
		t.Fatal("Unable to get the patient visit id")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to read body of the response for the new patient visit call: " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unsuccessful call to register new patient visit: "+string(body), t)

	// get the patient visit information to ensure that the case has been submitted
	patientVisit, err := testData.DataApi.GetPatientVisitFromId(PatientVisitId)
	if patientVisit.Status != "SUBMITTED" {
		t.Fatalf("Case status should be submitted after the case was submitted to the doctor, but its not. It is %s instead.", patientVisit.Status)
	}
}
