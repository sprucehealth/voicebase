package test_integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"testing"

	_ "github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/go-sql-driver/mysql"
	"github.com/sprucehealth/backend/address"
	"github.com/sprucehealth/backend/apiservice/apipaths"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/misc/handlers"
	"github.com/sprucehealth/backend/patient"
	"github.com/sprucehealth/backend/test"
)

func TestPatientRegistration(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	SignupRandomTestPatientWithPharmacyAndAddress(t, testData)
}

func TestPatientCareProvidingEllgibility(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	resp, err := http.Get(testData.APIServer.URL + apipaths.CheckEligibilityURLPath + "?zip_code=94115")
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	// should be marked as available
	var j map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&j); err != nil {
		t.Fatal(err)
	} else if !j["available"].(bool) {
		t.Fatal("Expected this state to be eligible but it wasnt")
	}

	// when the state code is provided, should skip resolving of zipcode to state
	stubAddressValidationService := testData.Config.AddressValidationAPI.(*address.StubAddressValidationService)
	stubAddressValidationService.CityStateToReturn = nil
	resp, err = http.Get(testData.APIServer.URL + apipaths.CheckEligibilityURLPath + "?state_code=CA")
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)
	j = nil
	if err := json.NewDecoder(resp.Body).Decode(&j); err != nil {
		t.Fatal(err)
	} else if !j["available"].(bool) {
		t.Fatal("Expected this state to be eligible but it wasnt")
	}

	// when state and zipcode is provided, should still skip resolving of zipcode to state
	resp, err = http.Get(testData.APIServer.URL + apipaths.CheckEligibilityURLPath + "?state_code=CA&zip_code=94115")
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	// should be marked as unavailable
	stubAddressValidationService.CityStateToReturn = &address.CityState{
		City:              "Aventura",
		State:             "Florida",
		StateAbbreviation: "FL",
	}
	resp, err = testData.AuthGet(testData.APIServer.URL+apipaths.CheckEligibilityURLPath+"?zip_code=33180", 0)
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	if err := json.NewDecoder(resp.Body).Decode(&j); err != nil {
		t.Fatal(err)
	} else if j["available"].(bool) {
		t.Fatal("Expected this state to be ineligible but it wasnt")
	}

}

func TestPatientVisitCreation(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	signedupPatientResponse := SignupRandomTestPatientWithPharmacyAndAddress(t, testData)
	patientVisitResponse := CreatePatientVisitForPatient(signedupPatientResponse.Patient.PatientID.Int64(), testData, t)

	if patientVisitResponse.PatientVisitID == 0 {
		t.Fatal("Patient Visit Id not set when it should be.")
	}

	if patientVisitResponse.ClientLayout == nil {
		t.Fatal("The questions for patient intake should be returned as part of the patient visit")
	}

	// getting the patient visit again as we should get back the same patient visit id
	// since this patient visit has not been completed
	anotherPatientVisitResponse := GetPatientVisitForPatient(signedupPatientResponse.Patient.PatientID.Int64(), testData, t)
	if anotherPatientVisitResponse.PatientVisitID != patientVisitResponse.PatientVisitID {
		t.Fatal("The patient visit id for subsequent calls should be the same so long as we have not closed/submitted the case")
	}

	// ensure that the patient visit is created in the unclaimed state
	patientCase, err := testData.DataAPI.GetPatientCaseFromPatientVisitID(patientVisitResponse.PatientVisitID)
	if err != nil {
		t.Fatal(err)
	} else if patientCase.Status != common.PCStatusUnclaimed {
		t.Fatalf("Expected the patient case to be created in the %s state but it was %s state", common.PCStatusUnclaimed, patientCase.Status)
	}

	// ensure that no doctor are assigned to the patient case yet
	doctorAssignments, err := testData.DataAPI.GetDoctorsAssignedToPatientCase(patientCase.ID.Int64())
	if err != nil {
		t.Fatal(err)
	} else if len(doctorAssignments) != 0 {
		t.Fatalf("Expected 0 doctors assigned to patient case instead got %d", len(doctorAssignments))
	}
}

func TestPatientVisitSubmission(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	signedupPatientResponse := SignupRandomTestPatientWithPharmacyAndAddress(t, testData)
	patientVisitResponse := CreatePatientVisitForPatient(signedupPatientResponse.Patient.PatientID.Int64(), testData, t)

	SubmitPatientVisitForPatient(signedupPatientResponse.Patient.PatientID.Int64(), patientVisitResponse.PatientVisitID, testData, t)

	// try submitting the exact same patient visit again, and it should come back with a 200 to be idempotent
	patient, err := testData.DataAPI.GetPatientFromID(signedupPatientResponse.Patient.PatientID.Int64())
	if err != nil {
		t.Fatal("Unable to get patient information given the patient id: " + err.Error())
	}

	buffer := bytes.NewBufferString("patient_visit_id=")
	buffer.WriteString(strconv.FormatInt(patientVisitResponse.PatientVisitID, 10))

	resp, err := testData.AuthPut(testData.APIServer.URL+apipaths.PatientVisitURLPath, "application/x-www-form-urlencoded", buffer, patient.AccountID.Int64())
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)
}

func TestPatientAutocompleteForDrugs(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	// use a real dosespot service before instantiating the server
	testData.Config.ERxAPI = testData.ERxAPI
	testData.StartAPIServer(t)

	signedupPatientResponse := SignupRandomTestPatientWithPharmacyAndAddress(t, testData)

	params := url.Values{}
	params.Set("query", "Lipi")

	resp, err := testData.AuthGet(testData.APIServer.URL+apipaths.AutocompleteURLPath+"?"+params.Encode(), signedupPatientResponse.Patient.AccountID.Int64())
	if err != nil {
		t.Fatalf("Unsuccessful get request to autocomplete api: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Unable to successfully do a drug search from patient side: %s", err)
	}

	autoCompleteResponse := handlers.AutocompleteResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&autoCompleteResponse); err != nil {
		t.Fatalf("Unable to decode response body into json: %s", err)
	}

	if len(autoCompleteResponse.Suggestions) == 0 {
		t.Fatalf("Expected suggestions to be returned from the autocomplete api instead got 0")
	}
}

func TestPatientUpdate(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	signedupPatientResponse := SignupRandomTestPatientWithPharmacyAndAddress(t, testData)

	pat, err := testData.DataAPI.GetPatientFromID(signedupPatientResponse.Patient.PatientID.Int64())
	test.OK(t, err)
	test.Equals(t, 1, len(pat.PhoneNumbers))
	test.Equals(t, &common.PhoneNumber{Phone: "734-846-5522", Type: "Cell", Status: "", Verified: false}, pat.PhoneNumbers[0])

	patientCli := PatientClient(testData, t, signedupPatientResponse.Patient.PatientID.Int64())

	test.OK(t, patientCli.UpdatePatient(&patient.UpdateRequest{
		PhoneNumbers: []patient.PhoneNumber{
			{
				Number: "415-555-5555",
				Type:   "Home",
			},
		},
	}))

	pat, err = testData.DataAPI.GetPatientFromID(signedupPatientResponse.Patient.PatientID.Int64())
	test.OK(t, err)
	test.Equals(t, 1, len(pat.PhoneNumbers))
	test.Equals(t, &common.PhoneNumber{Phone: "415-555-5555", Type: "Home", Status: "", Verified: false}, pat.PhoneNumbers[0])
}
