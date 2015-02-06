package test_patient

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"testing"

	"github.com/sprucehealth/backend/apiservice/apipaths"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/patient_file"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestGetCareTeamsForPatientDataAccess(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	patientCase, doctor := createPatientCaseAndAssignToDoctor(t, testData)

	// Verify that the doctor belongs to our care team, and there is only one care team.
	// Note: Why isn't this complex assertion in a helper?
	//    Inorder for this to be a localized test for the GetCareTeamsForPatient we must
	//    call it directly and assert on it's granular information
	careTeam, err := testData.DataAPI.GetCareTeamForPatient(patientCase.PatientID.Int64())
	test.OK(t, err)
	if careTeam == nil {
		t.Fatal("Expected a set of care teams to exist but they don't")
	} else if len(careTeam.Assignments) != 1 {
		t.Fatalf("Expected 1 doctor to exist in care team instead got %d", len(careTeam.Assignments))
	} else if careTeam.Assignments[0].ProviderID != doctor.DoctorID.Int64() {
		t.Fatalf("Expected the doctor in the care team to be %v instead found %v", doctor.DoctorID.Int64Value, careTeam.Assignments[0].ProviderID)
	}
}

func TestGetCareTeamsForPatientByCaseDataAccess(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	patientCase, doctor := createPatientCaseAndAssignToDoctor(t, testData)

	// Verify that the doctor belongs to our care team, and there is only one care team.
	// Note: Why isn't this complex assertion in a helper?
	//    Inorder for this to be a localized test for the GetCareTeamsForPatientByHealthCondition we must
	//    call it directly and assert on it's granular information
	careTeams, err := testData.DataAPI.GetCareTeamsForPatientByCase(patientCase.PatientID.Int64())
	test.OK(t, err)
	if careTeams == nil {
		t.Fatal("Expected a set of care teams to exist but they don't")
	} else if len(careTeams) != 1 {
		t.Fatalf("Expected 1 care team to exist for the patient instead got %d", len(careTeams))
	} else if _, ok := careTeams[patientCase.ID.Int64Value]; !ok {
		t.Fatalf("Expected care team to map to case id %d", patientCase.ID.Int64Value)
	} else if careTeams[patientCase.ID.Int64Value].Assignments[0].ProviderID != doctor.DoctorID.Int64Value {
		t.Fatalf("Expected the doctor in the care team to be %v instead found %v", doctor.DoctorID.Int64(), careTeams[patientCase.ID.Int64Value].Assignments[0].ProviderID)
	}
}

// TestGetCareTeamsForPatient tests the /v1/patient/care_teams endpoint
func TestGetCareTeamsForPatient(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	patientCase, doctor := createPatientCaseAndAssignToDoctor(t, testData)

	// Verify that the doctor belongs to our care team, and there is only one care team.
	res, err := testData.AuthGet(testData.APIServer.URL+apipaths.PatientCareTeamsURLPath+"?patient_id="+strconv.FormatInt(patientCase.PatientID.Int64Value, 10), doctor.AccountID.Int64())
	test.Equals(t, http.StatusOK, res.StatusCode)
	test.OK(t, err)
	body, err := ioutil.ReadAll(res.Body)
	test.OK(t, err)
	var response patient_file.PatientCareTeamResponse
	err = json.Unmarshal(body, &response)
	test.OK(t, err)
	if len(response.CareTeams) != 1 {
		t.Fatalf("Expected 1 care team to exist but found %v", len(response.CareTeams))
	} else if len(response.CareTeams[0].Members) != 1 {
		t.Fatalf("Expected 1 member to be assigned to the patients care team but found %v", len(response.CareTeams[patientCase.ID.Int64Value].Members))
	} else if response.CareTeams[0].Members[0].ProviderID != doctor.DoctorID.Int64Value {
		t.Fatalf("Expected the doctor assigned to the care team to be %v but found %v", doctor.DoctorID.Int64Value, response.CareTeams[patientCase.ID.Int64Value].Members[0].ProviderID)
	}
}

func createPatientCaseAndAssignToDoctor(t *testing.T, testData *test_integration.TestData) (*common.PatientCase, *common.Doctor) {
	doctorID := test_integration.GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	test.OK(t, err)

	// Create a random patient
	vp, _ := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	// Create a visit/case for the patient visit
	patientCase, err := testData.DataAPI.GetPatientCaseFromPatientVisitID(vp.PatientVisitID)
	test.OK(t, err)

	// ensure a doctor is assigned to case
	doctorAssignments, err := testData.DataAPI.GetDoctorsAssignedToPatientCase(patientCase.ID.Int64())
	if len(doctorAssignments) != 1 {
		t.Fatal("Expected there to be only 1 doctor assigned to patient's case")
	}

	return patientCase, doctor
}
