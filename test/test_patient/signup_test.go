package test_patient

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/sprucehealth/backend/address"
	"github.com/sprucehealth/backend/apiservice/router"
	patientpkg "github.com/sprucehealth/backend/patient"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestPatientSignup_WithStateCode(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	stubAddressValidationAPI := testData.Config.AddressValidationAPI.(*address.StubAddressValidationService)
	// dont return any city state info so as to ensure that the call to sign patient up
	// still doesnt fail
	stubAddressValidationAPI.CityStateToReturn = nil

	// lets signup a patient with state code provided
	params := url.Values{}
	params.Set("first_name", "test")
	params.Set("last_name", "test1")
	params.Set("email", "test@test.com")
	params.Set("password", "12345")
	params.Set("state_code", "CA")
	params.Set("zip_code", "94115")
	params.Set("dob", "1987-11-08")
	params.Set("gender", "female")
	params.Set("phone", "2068773590")

	req, err := http.NewRequest("POST", testData.APIServer.URL+router.PatientSignupURLPath, strings.NewReader(params.Encode()))
	test.OK(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)
}

func TestPatientSignup_CreateVisit(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	stubAddressValidationAPI := testData.Config.AddressValidationAPI.(*address.StubAddressValidationService)
	// dont return any city state info so as to ensure that the call to sign patient up
	// still doesnt fail
	stubAddressValidationAPI.CityStateToReturn = nil

	// lets signup a patient with state code provided
	params := url.Values{}
	params.Set("first_name", "test")
	params.Set("last_name", "test1")
	params.Set("email", "test@test.com")
	params.Set("password", "12345")
	params.Set("state_code", "CA")
	params.Set("zip_code", "94115")
	params.Set("dob", "1987-11-08")
	params.Set("gender", "female")
	params.Set("phone", "2068773590")
	params.Set("create_visit", "true")

	req, err := http.NewRequest("POST", testData.APIServer.URL+router.PatientSignupURLPath, strings.NewReader(params.Encode()))
	test.OK(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("S-Version", "Patient;Dev;0.9.5")
	req.Header.Set("S-OS", "iOS;")
	resp, err := http.DefaultClient.Do(req)
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	var respData patientpkg.PatientSignedupResponse
	err = json.NewDecoder(resp.Body).Decode(&respData)
	test.OK(t, err)
	test.Equals(t, true, respData.PatientVisitData != nil)

	patientVisit, err := testData.DataApi.GetLastCreatedPatientVisit(respData.Patient.PatientId.Int64())
	test.OK(t, err)
	test.Equals(t, patientVisit.PatientVisitId.Int64(), respData.PatientVisitData.PatientVisitId)
}

func TestPatientSignup_Idempotent(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	stubAddressValidationAPI := testData.Config.AddressValidationAPI.(*address.StubAddressValidationService)
	// dont return any city state info so as to ensure that the call to sign patient up
	// still doesnt fail
	stubAddressValidationAPI.CityStateToReturn = nil

	// lets signup a patient with state code provided
	params := url.Values{}
	params.Set("first_name", "test")
	params.Set("last_name", "test1")
	params.Set("email", "test@test.com")
	params.Set("password", "12345")
	params.Set("state_code", "CA")
	params.Set("zip_code", "94115")
	params.Set("dob", "1987-11-08")
	params.Set("gender", "female")
	params.Set("phone", "2068773590")
	params.Set("create_visit", "true")

	req, err := http.NewRequest("POST", testData.APIServer.URL+router.PatientSignupURLPath, strings.NewReader(params.Encode()))
	test.OK(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("S-Version", "Patient;Dev;0.9.5")
	req.Header.Set("S-OS", "iOS;")
	resp, err := http.DefaultClient.Do(req)
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)
	var respData patientpkg.PatientSignedupResponse
	err = json.NewDecoder(resp.Body).Decode(&respData)
	test.OK(t, err)
	patientId := respData.Patient.PatientId.Int64()

	// ensure that a signup with the same credentials goes through if made within a small window
	// and make sure that any patient information is updated as well
	params.Set("last_name", "test_again")
	req, err = http.NewRequest("POST", testData.APIServer.URL+router.PatientSignupURLPath, strings.NewReader(params.Encode()))
	test.OK(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("S-Version", "Patient;Dev;0.9.5")
	req.Header.Set("S-OS", "iOS;")
	resp, err = http.DefaultClient.Do(req)
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)
	respData = patientpkg.PatientSignedupResponse{}
	err = json.NewDecoder(resp.Body).Decode(&respData)
	test.OK(t, err)
	// ensure that this is the same patient as in the previous call
	test.Equals(t, patientId, respData.Patient.PatientId.Int64())
	// ensure that the patient information was indeed updated
	test.Equals(t, "test_again", respData.Patient.LastName)

	// ensure that a signup call with the same email address but different password does not succeed
	params.Set("password", "2323")
	req, err = http.NewRequest("POST", testData.APIServer.URL+router.PatientSignupURLPath, strings.NewReader(params.Encode()))
	test.OK(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("S-Version", "Patient;Dev;0.9.5")
	req.Header.Set("S-OS", "iOS;")
	resp, err = http.DefaultClient.Do(req)
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusBadRequest, resp.StatusCode)

	// now simulate the case where the patient registered 30 minutes ago
	_, err = testData.DB.Exec(`update account set registration_date = ? where id = ?`, time.Now().Add(-30*time.Minute), respData.Patient.AccountId.Int64())
	test.OK(t, err)

	// now try signing the patient up again and it should fail
	req, err = http.NewRequest("POST", testData.APIServer.URL+router.PatientSignupURLPath, strings.NewReader(params.Encode()))
	test.OK(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("S-Version", "Patient;Dev;0.9.5")
	req.Header.Set("S-OS", "iOS;")
	resp, err = http.DefaultClient.Do(req)
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusBadRequest, resp.StatusCode)
}
