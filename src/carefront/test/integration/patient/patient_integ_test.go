package patient

import (
	// "encoding/json"
	// "fmt"
	"bytes"
	"carefront/apiservice"
	"carefront/test/integration"
	_ "github.com/go-sql-driver/mysql"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestPatientRegistration(t *testing.T) {
	testData := integration.SetupIntegrationTest(t)
	defer testData.DB.Close()
	SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
}

func TestPatientVisitCreation(t *testing.T) {
	if err := integration.CheckIfRunningLocally(t); err == integration.CannotRunTestLocally {
		return
	}
	testData := integration.SetupIntegrationTest(t)
	defer testData.DB.Close()

	signedupPatientResponse := SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patientVisitResponse := GetPatientVisitForPatient(signedupPatientResponse.PatientId, testData, t)

	if patientVisitResponse.PatientVisitId == 0 {
		t.Fatal("Patient Visit Id not set when it should be.")
	}

	if patientVisitResponse.ClientLayout == nil {
		t.Fatal("The questions for patient intake should be returned as part of the patient visit")
	}

	// checking to ensure that the care team was created
	careTeam, err := testData.DataApi.GetCareTeamForPatientVisitId(patientVisitResponse.PatientVisitId)
	if err != nil {
		t.Fatal("Unable to get care team for patient visit: " + err.Error())
	}

	if !(careTeam == nil || careTeam.PatientVisitId == patientVisitResponse.PatientVisitId) {
		t.Fatal("Unable to get patient visit id for care team")
	}

	// ensuring that we have a primary doctor assigned to the case
	primaryDoctorFound := false
	for _, assignment := range careTeam.Assignments {
		if assignment.ProviderRole == "PRIMARY_DOCTOR" {
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
	if err := integration.CheckIfRunningLocally(t); err == integration.CannotRunTestLocally {
		return
	}
	testData := integration.SetupIntegrationTest(t)
	defer testData.DB.Close()

	signedupPatientResponse := SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patientVisitResponse := GetPatientVisitForPatient(signedupPatientResponse.PatientId, testData, t)

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
	resp, err := http.Post(ts.URL, "application/x-www-form-urlencoded", buffer)

	if err != nil {
		t.Fatal("Unable to get the patient visit id")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to read body of the response for the new patient visit call: " + err.Error())
	}

	integration.CheckSuccessfulStatusCode(resp, "Unsuccessful call to register new patient visit: "+string(body), t)

	// get the patient visit information to ensure that the case has been submitted
	patientVisit, err := testData.DataApi.GetPatientVisitFromId(patientVisitResponse.PatientVisitId)
	if patientVisit.Status != "SUBMITTED" {
		t.Fatalf("Case status should be submitted after the case was submitted to the doctor, but its not. It is %s instead.", patientVisit.Status)
	}

	// now, the patient_visit returned should be diffeent than the previous one
	anotherPatientVisitResponse := GetPatientVisitForPatient(signedupPatientResponse.PatientId, testData, t)
	if anotherPatientVisitResponse.PatientVisitId == patientVisitResponse.PatientVisitId {
		t.Fatal("The patient visit id should be different as a new visit should start after the patient has submitted a patient visit")
	}
}
