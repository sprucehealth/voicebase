package patient

import (
	"carefront/test/integration"
	_ "github.com/go-sql-driver/mysql"
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
	careTeam, err := testData.DataApi.GetCareTeamForPatientVisit(patientVisitResponse.PatientVisitId)
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

	SubmitPatientVisitForPatient(signedupPatientResponse.PatientId, patientVisitResponse.PatientVisitId, testData, t)

	// now, the patient_visit returned should be diffeent than the previous one
	anotherPatientVisitResponse := GetPatientVisitForPatient(signedupPatientResponse.PatientId, testData, t)
	if anotherPatientVisitResponse.PatientVisitId == patientVisitResponse.PatientVisitId {
		t.Fatal("The patient visit id should be different as a new visit should start after the patient has submitted a patient visit")
	}
}
