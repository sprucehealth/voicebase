package integration

import (
	// "carefront/apiservice"
	// "encoding/json"
	// "fmt"
	_ "github.com/go-sql-driver/mysql"
	// "io/ioutil"
	// "net/http"
	// "net/http/httptest"
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
	patientVisitResponse := GetPatientVisitForPatient(signedupPatientResponse.PatientId, testData, t)

	if patientVisitResponse.PatientVisitId == 0 {
		t.Fatal("Patient Visit Id not set when it should be.")
	}

	if patientVisitResponse.ClientLayout == nil {
		t.Fatal("The questions for patient intake should be returned as part of the patient visit")
	}

	// getting the patient visit again as we should get back the same patient visit id
	// since this patient visit has not been completed
	anotherPatientVisitResponse := GetPatientVisitForPatient(signedupPatientResponse.PatientId, testData, t)
	if anotherPatientVisitResponse.PatientVisitId != patientVisitResponse.PatientVisitId {
		t.Fatal("The patient visit id for subsequent calls should be the same so long as we have not closed/submitted the case")
	}
}
