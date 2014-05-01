package integration

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"carefront/api"
	"carefront/apiservice"
	"carefront/libs/address_validation"

	_ "github.com/go-sql-driver/mysql"
)

func TestPatientRegistration(t *testing.T) {
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)
	signupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
}

func TestPatientCareProvidingEllgibility(t *testing.T) {
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	stubAddressValidationService := address_validation.StubAddressValidationService{
		CityStateToReturn: address_validation.CityState{
			City:              "San Francisco",
			State:             "California",
			StateAbbreviation: "CA",
		},
	}

	checkElligibilityHandler := &apiservice.CheckCareProvidingElligibilityHandler{DataApi: testData.DataApi, AddressValidationApi: stubAddressValidationService}
	ts := httptest.NewServer(checkElligibilityHandler)
	defer ts.Close()
	resp, err := http.Get(ts.URL + "?zip_code=94115")

	if err != nil {
		t.Fatal("Unable to successfuly check care providing elligiblity for patient " + err.Error())
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to successfuly read the body of the response")
	}

	CheckSuccessfulStatusCode(resp, "Unable to make a successful call to check for care providing elligibility: "+string(body), t)

	stubAddressValidationService.CityStateToReturn = address_validation.CityState{
		City:              "Aventura",
		State:             "Florida",
		StateAbbreviation: "FL",
	}

	checkElligibilityHandler.AddressValidationApi = stubAddressValidationService

	resp, err = authGet(ts.URL+"?zip_code=33180", 0)
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
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	signedupPatientResponse := signupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patientVisitResponse := createPatientVisitForPatient(signedupPatientResponse.Patient.PatientId.Int64(), testData, t)

	if patientVisitResponse.PatientVisitId == 0 {
		t.Fatal("Patient Visit Id not set when it should be.")
	}

	if patientVisitResponse.ClientLayout == nil {
		t.Fatal("The questions for patient intake should be returned as part of the patient visit")
	}

	// checking to ensure that the care team was created
	careTeam, err := testData.DataApi.GetCareTeamForPatient(signedupPatientResponse.Patient.PatientId.Int64())
	if err != nil {
		t.Fatal("Unable to get care team for patient visit: " + err.Error())
	}

	if !(careTeam == nil || careTeam.PatientId == signedupPatientResponse.Patient.PatientId.Int64()) {
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
	anotherPatientVisitResponse := getPatientVisitForPatient(signedupPatientResponse.Patient.PatientId.Int64(), testData, t)
	if anotherPatientVisitResponse.PatientVisitId != patientVisitResponse.PatientVisitId {
		t.Fatal("The patient visit id for subsequent calls should be the same so long as we have not closed/submitted the case")
	}
}

func TestPatientVisitSubmission(t *testing.T) {
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	signedupPatientResponse := signupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patientVisitResponse := createPatientVisitForPatient(signedupPatientResponse.Patient.PatientId.Int64(), testData, t)

	submitPatientVisitForPatient(signedupPatientResponse.Patient.PatientId.Int64(), patientVisitResponse.PatientVisitId, testData, t)

	// try submitting the exact same patient visit again, and it should come back with a 403 given that the case has already been submitted

	patientVisitHandler := apiservice.NewPatientVisitHandler(testData.DataApi, testData.AuthApi,
		testData.CloudStorageService, testData.CloudStorageService)
	patient, err := testData.DataApi.GetPatientFromId(signedupPatientResponse.Patient.PatientId.Int64())
	if err != nil {
		t.Fatal("Unable to get patient information given the patient id: " + err.Error())
	}

	ts := httptest.NewServer(patientVisitHandler)
	defer ts.Close()
	buffer := bytes.NewBufferString("patient_visit_id=")
	buffer.WriteString(strconv.FormatInt(patientVisitResponse.PatientVisitId, 10))

	resp, err := authPut(ts.URL, "application/x-www-form-urlencoded", buffer, patient.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to get the patient visit id")
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected a bad request 403 to be returned when attempting to submit an already submitted patient visit, but instead got %d", resp.StatusCode)
	}
}

func TestPatientAutocompleteForDrugs(t *testing.T) {
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	signedupPatientResponse := signupRandomTestPatient(t, testData.DataApi, testData.AuthApi)

	autocompleteHandler := apiservice.AutocompleteHandler{
		DataApi: testData.DataApi,
		ERxApi:  setupErxAPI(t),
		Role:    api.PATIENT_ROLE,
	}

	params := url.Values{}
	params.Set("query", "Lipi")

	ts := httptest.NewServer(&autocompleteHandler)
	defer ts.Close()

	resp, err := authGet(ts.URL+"?"+params.Encode(), signedupPatientResponse.Patient.AccountId.Int64())
	if err != nil {
		t.Fatalf("Unsuccessful get request to autocomplete api: %s", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Unable to successfully do a drug search from patient side: %s", err)
	}

	autoCompleteResponse := apiservice.AutocompleteResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&autoCompleteResponse); err != nil {
		t.Fatalf("Unable to decode response body into json: %s", err)
	}

	if len(autoCompleteResponse.Suggestions) == 0 {
		t.Fatalf("Expected suggestions to be returned from the autocomplete api instead got 0")
	}
}
