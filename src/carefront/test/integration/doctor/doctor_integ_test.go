package doctor

import (
	"bytes"
	"carefront/apiservice"
	"carefront/test/integration"
	"carefront/test/integration/patient"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestDoctorRegistration(t *testing.T) {
	if err := integration.CheckIfRunningLocally(t); err == integration.CannotRunTestLocally {
		return
	}

	testData := integration.SetupIntegrationTest(t)
	defer testData.DB.Close()
	SignupRandomTestDoctor(t, testData.DataApi, testData.AuthApi)
}

func TestDoctorAuthentication(t *testing.T) {
	if err := integration.CheckIfRunningLocally(t); err == integration.CannotRunTestLocally {
		return
	}

	testData := integration.SetupIntegrationTest(t)
	defer testData.DB.Close()
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
	integration.CheckSuccessfulStatusCode(res, fmt.Sprintf("Unable to make success request to authenticate doctor. Here's the code returned %d and here's the body of the request %s", res.StatusCode, body), t)

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
	if err := integration.CheckIfRunningLocally(t); err == integration.CannotRunTestLocally {
		t.Log("Skipping test since there is no database to run test on")
		return
	}
	testData := integration.SetupIntegrationTest(t)
	defer testData.DB.Close()
	signedupDoctorResponse, _, _ := SignupRandomTestDoctor(t, testData.DataApi, testData.AuthApi)
	patientSignedupResponse := patient.SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)

	doctor, err := testData.DataApi.GetDoctorFromId(signedupDoctorResponse.DoctorId)

	// create the role of a primary doctor
	_, err = testData.DB.Exec(`insert into provider_role (provider_tag) values ('PRIMARY_DOCTOR')`)
	if err != nil {
		t.Fatal("Unable to create the provider role of PRIMARY_DOCTOR " + err.Error())
	}

	// clean up of data created after test is run
	defer func() {
		testData.DB.Exec(`delete from patient_visit_provider_assignment where patient_visit_id=?`)
		testData.DB.Exec(`delete form patient_visit_provider_group where patient_visit_id=?`)
		testData.DB.Exec(`delete from care_provider_state_elligibility where provider_id=?`, signedupDoctorResponse.DoctorId)
		testData.DB.Exec(`delete from provider_role where provider_tag='PRIMARY_DOCTOR'`)
	}()

	// make this doctor the primary doctor in the state of CA
	_, err = testData.DB.Exec(`insert into care_provider_state_elligibility (provider_role_id, provider_id, care_providing_state_id) 
					values ((select id from provider_role where provider_tag='PRIMARY_DOCTOR'), ?, (select id from care_providing_state where state='CA'))`, signedupDoctorResponse.DoctorId)
	if err != nil {
		t.Fatal("Unable to make the signed up doctor the primary doctor elligible in CA to diagnose patients: " + err.Error())
	}

	// get patient to start a visit
	patientVisitResponse := patient.GetPatientVisitForPatient(patientSignedupResponse.PatientId, testData, t)

	// clean up of data created after test is run
	defer func() {
		testData.DB.Exec(`delete from patient_visit_provider_assignment where patient_visit_id=?`, patientVisitResponse.PatientVisitId)
		testData.DB.Exec(`delete form patient_visit_provider_group where patient_visit_id=?`, patientVisitResponse.PatientVisitId)
	}()

	// get patient to submit the visit
	patient.SubmitPatientVisitForPatient(patientSignedupResponse.PatientId, patientVisitResponse.PatientVisitId, testData, t)

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

	integration.CheckSuccessfulStatusCode(resp, "Unable to make successful request for doctor to get layout to diagnose patient. Reason: "+string(data), t)

	diagnosisResponse := apiservice.GetDiagnosisResponse{}
	err = json.Unmarshal(data, &diagnosisResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal response for diagnosis of patient visit: " + err.Error())
	}

	if diagnosisResponse.DiagnosisLayout == nil || diagnosisResponse.DiagnosisLayout.PatientVisitId != patientVisitResponse.PatientVisitId {
		t.Fatal("Diagnosis response not as expected")
	}
}
