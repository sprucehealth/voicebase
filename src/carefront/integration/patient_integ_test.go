package integration

import (
	"carefront/apiservice"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPatientRegistration(t *testing.T) {
	CheckIfRunningLocally(t)
	testData := SetupIntegrationTest(t)
	defer testData.DB.Close()
	SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
}

func TestPatientVisitCreation(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}
	testData := SetupIntegrationTest(t)
	defer testData.DB.Close()

	signedupPatientResponse := SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patientVisitHandler := apiservice.NewPatientVisitHandler(testData.DataApi, testData.AuthApi,
		testData.CloudStorageService, testData.CloudStorageService)
	patientVisitHandler.AccountIdFromAuthToken(signedupPatientResponse.PatientId)
	ts := httptest.NewServer(patientVisitHandler)
	defer ts.Close()

	client := &http.Client{}
	req, _ := http.NewRequest("GET", ts.URL, nil)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal("Error making request to create patient visit")
	}

	body, err := ioutil.ReadAll(resp.Body)

	CheckSuccessfulStatusCode(resp, fmt.Sprintf("Unable to make success request to signup patient. Here's the code returned %d and here's the body of the request %s", resp.StatusCode, body), t)

	patientVisitResponse := &apiservice.PatientVisitResponse{}
	err = json.Unmarshal(body, patientVisitResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal response from call to patient visit into the response object: " + err.Error())
	}

	if patientVisitResponse.PatientVisitId == 0 {
		t.Fatal("Patient Visit Id not set when it should be.")
	}

	if patientVisitResponse.ClientLayout == nil {
		t.Fatal("The questions for patient intake should be returned as part of the patient visit")
	}

	// getting the patient visit again as we should get back the same patient visit id
	// since this patient visit has not been completed
	client = &http.Client{}
	req, _ = http.NewRequest("GET", ts.URL, nil)

	resp, err = client.Do(req)
	if err != nil {
		t.Fatal("Error making subsequent patient visit request : " + err.Error())
	}
	CheckSuccessfulStatusCode(resp, "Unable to make successful subsequent patient visit request", t)

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to read the body of the response on the subsequent patient visit call: " + err.Error())
	}

	anotherPatientVisitResponse := &apiservice.PatientVisitResponse{}
	err = json.Unmarshal(body, anotherPatientVisitResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal response body into json response : " + err.Error())
	}

	if anotherPatientVisitResponse.PatientVisitId != patientVisitResponse.PatientVisitId {
		t.Fatal("The patient visit id for subsequent calls should be the same so long as we have not closed/submitted the case")
	}
}
