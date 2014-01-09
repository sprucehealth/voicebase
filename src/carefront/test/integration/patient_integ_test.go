package integration

import (
	"bytes"
	"carefront/apiservice"
	"carefront/libs/maps"
	_ "github.com/go-sql-driver/mysql"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestPatientRegistration(t *testing.T) {
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)
	SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
}

func TestPatientCareProvidingEllgibility(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	checkElligibilityHandler := &apiservice.CheckCareProvidingElligibilityHandler{DataApi: testData.DataApi, MapsService: maps.GoogleMapsService(0)}
	ts := httptest.NewServer(checkElligibilityHandler)
	resp, err := http.Get(ts.URL + "?zip_code=94115")

	if err != nil {
		t.Fatal("Unable to successfuly check care providing elligiblity for patient " + err.Error())
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to successfuly read the body of the response")
	}

	CheckSuccessfulStatusCode(resp, "Unable to make a successful call to check for care providing elligibility: "+string(body), t)

	resp, err = http.Get(ts.URL + "?zip_code=33180")
	if err != nil {
		t.Fatal("Unable to successfuly check care providing elligibility for patient" + err.Error())
	}

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to read the response from the body for patient care providing elligibility check: " + err.Error())
	}

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("Expected the status code to be 403, but got a %d instead", resp.StatusCode)
	}
}

func TestPatientVisitCreation(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	signedupPatientResponse := SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patientVisitResponse := CreatePatientVisitForPatient(signedupPatientResponse.PatientId, testData, t)

	if patientVisitResponse.PatientVisitId == 0 {
		t.Fatal("Patient Visit Id not set when it should be.")
	}

	if patientVisitResponse.ClientLayout == nil {
		t.Fatal("The questions for patient intake should be returned as part of the patient visit")
	}

	// checking to ensure that the care team was created
	careTeam, err := testData.DataApi.GetCareTeamForPatient(signedupPatientResponse.PatientId)
	if err != nil {
		t.Fatal("Unable to get care team for patient visit: " + err.Error())
	}

	if !(careTeam == nil || careTeam.PatientId == signedupPatientResponse.PatientId) {
		t.Fatal("Unable to get patient visit id for care team")
	}

	// ensuring that we have a primary doctor assigned to the case
	primaryDoctorFound := false
	for _, assignment := range careTeam.Assignments {
		if assignment.ProviderRole == "DOCTOR" {
			primaryDoctorFound = true
		}
	}

	if primaryDoctorFound == false {
		t.Fatal("Primary doctor not found for patient visit")
	}

	// getting the patient visit again as we should get back the same patient visit id
	// since this patient visit has not been completed
	anotherPatientVisitResponse := GetPatientVisitForPatient(signedupPatientResponse.PatientId, testData, t)
	if anotherPatientVisitResponse.PatientVisitId != patientVisitResponse.PatientVisitId {
		t.Fatal("The patient visit id for subsequent calls should be the same so long as we have not closed/submitted the case")
	}
}

func TestPatientVisitSubmission(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	signedupPatientResponse := SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patientVisitResponse := CreatePatientVisitForPatient(signedupPatientResponse.PatientId, testData, t)

	SubmitPatientVisitForPatient(signedupPatientResponse.PatientId, patientVisitResponse.PatientVisitId, testData, t)

	// try submitting the exact same patient visit again, and it should come back with a 403 given that the case has already been submitted

	patientVisitHandler := apiservice.NewPatientVisitHandler(testData.DataApi, testData.AuthApi,
		testData.CloudStorageService, testData.CloudStorageService)
	patient, err := testData.DataApi.GetPatientFromId(signedupPatientResponse.PatientId)
	if err != nil {
		t.Fatal("Unable to get patient information given the patient id: " + err.Error())
	}

	patientVisitHandler.AccountIdFromAuthToken(patient.AccountId)
	ts := httptest.NewServer(patientVisitHandler)
	defer ts.Close()
	buffer := bytes.NewBufferString("patient_visit_id=")
	buffer.WriteString(strconv.FormatInt(patientVisitResponse.PatientVisitId, 10))
	client := &http.Client{}
	req, err := http.NewRequest("PUT", ts.URL, buffer)
	if err != nil {
		t.Fatal("Unable to create new request for submitting patient visit: " + err.Error())
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)

	if err != nil {
		t.Fatal("Unable to get the patient visit id")
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected a bad request 403 to be returned when attempting to submit an already submitted patient visit, but instead got %d", resp.StatusCode)
	}
}
