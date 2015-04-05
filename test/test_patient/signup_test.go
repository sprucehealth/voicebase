package test_patient

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/sprucehealth/backend/address"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice/apipaths"
	patientpkg "github.com/sprucehealth/backend/patient"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestPatientSignup_WithStateCode(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	stubAddressValidationAPI := testData.Config.AddressValidator.(*address.StubAddressValidationService)
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

	req, err := http.NewRequest("POST", testData.APIServer.URL+apipaths.PatientSignupURLPath, strings.NewReader(params.Encode()))
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
	stubAddressValidationAPI := testData.Config.AddressValidator.(*address.StubAddressValidationService)
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

	req, err := http.NewRequest("POST", testData.APIServer.URL+apipaths.PatientSignupURLPath, strings.NewReader(params.Encode()))
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

	patientVisit, err := testData.DataAPI.GetPatientVisitForSKU(respData.Patient.PatientID.Int64(), test_integration.SKUAcneVisit)
	test.OK(t, err)
	test.Equals(t, patientVisit.PatientVisitID.Int64(), respData.PatientVisitData.PatientVisitID)

	// ensure that there are no members assigned to the care team of the case yet
	members, err := testData.DataAPI.GetActiveMembersOfCareTeamForCase(patientVisit.PatientCaseID.Int64(), false)
	test.OK(t, err)
	test.Equals(t, 0, len(members))
}

func TestPatientSignup_Idempotent(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	stubAddressValidationAPI := testData.Config.AddressValidator.(*address.StubAddressValidationService)
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

	req, err := http.NewRequest("POST", testData.APIServer.URL+apipaths.PatientSignupURLPath, strings.NewReader(params.Encode()))
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
	patientID := respData.Patient.PatientID.Int64()

	// ensure that a signup with the same credentials goes through if made within a small window
	// and make sure that any patient information is updated as well
	params.Set("last_name", "test_again")
	req, err = http.NewRequest("POST", testData.APIServer.URL+apipaths.PatientSignupURLPath, strings.NewReader(params.Encode()))
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
	test.Equals(t, patientID, respData.Patient.PatientID.Int64())
	// ensure that the patient information was indeed updated
	test.Equals(t, "test_again", respData.Patient.LastName)

	// ensure that a signup call with the same email address but different password does not succeed
	params.Set("password", "2323")
	req, err = http.NewRequest("POST", testData.APIServer.URL+apipaths.PatientSignupURLPath, strings.NewReader(params.Encode()))
	test.OK(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("S-Version", "Patient;Dev;0.9.5")
	req.Header.Set("S-OS", "iOS;")
	resp, err = http.DefaultClient.Do(req)
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusBadRequest, resp.StatusCode)

	// now simulate the case where the patient registered 30 minutes ago
	_, err = testData.DB.Exec(`update account set registration_date = ? where id = ?`, time.Now().Add(-30*time.Minute), respData.Patient.AccountID.Int64())
	test.OK(t, err)

	// now try signing the patient up again and it should fail
	req, err = http.NewRequest("POST", testData.APIServer.URL+apipaths.PatientSignupURLPath, strings.NewReader(params.Encode()))
	test.OK(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("S-Version", "Patient;Dev;0.9.5")
	req.Header.Set("S-OS", "iOS;")
	resp, err = http.DefaultClient.Do(req)
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusBadRequest, resp.StatusCode)
}

func TestPatientSignup_WithDoctorPicked(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	stubAddressValidationAPI := testData.Config.AddressValidator.(*address.StubAddressValidationService)
	// dont return any city state info so as to ensure that the call to sign patient up
	// still doesnt fail
	stubAddressValidationAPI.CityStateToReturn = nil

	// create the doctor to be picked
	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)

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
	params.Set("care_provider_id", strconv.FormatInt(dr.DoctorID, 10))

	req, err := http.NewRequest("POST", testData.APIServer.URL+apipaths.PatientSignupURLPath, strings.NewReader(params.Encode()))
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
	patientID := respData.Patient.PatientID.Int64()

	// there should be a single case for the patient
	cases, err := testData.DataAPI.GetCasesForPatient(patientID, nil)
	test.OK(t, err)
	test.Equals(t, 1, len(cases))

	// the doctor should be assigned to this case
	members, err := testData.DataAPI.GetActiveMembersOfCareTeamForCase(cases[0].ID.Int64(), false)
	test.OK(t, err)
	test.Equals(t, 1, len(members))
	test.Equals(t, api.RoleDoctor, members[0].ProviderRole)
	test.Equals(t, dr.DoctorID, members[0].ProviderID)

}
