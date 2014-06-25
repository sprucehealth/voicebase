package test_doctor_queue

import (
	"bytes"
	"carefront/api"
	"carefront/apiservice"
	"carefront/common"
	"carefront/doctor_treatment_plan"
	"carefront/encoding"
	"carefront/patient_file"
	"carefront/patient_visit"
	"carefront/test/test_integration"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

// This test is to ensure that the a case is correctly
// temporarily claimed by a doctor
func TestJBCQ_TempCaseClaim(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)

	doctorID := test_integration.GetDoctorIdOfCurrentDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorID)
	if err != nil {
		t.Fatal(err)
	}

	vp, _ := test_integration.SignupAndSubmitPatientVisitForRandomPatient(t, testData, doctor)

	// ensure that the test is temporarily claimed
	patientCase, err := testData.DataApi.GetPatientCaseFromPatientVisitId(vp.PatientVisitId)
	if err != nil {
		t.Fatal(err)
	} else if patientCase.Status != common.PCStatusTempClaimed {
		t.Fatalf("Expected the patientCase status to be %s but it was %s", common.PCStatusTempClaimed, patientCase.Status)
	}

	// ensure that doctor is temporarily assigned to case
	doctorAssignments, err := testData.DataApi.GetDoctorsAssignedToPatientCase(patientCase.Id.Int64())
	if err != nil {
		t.Fatal(err)
	} else if len(doctorAssignments) != 1 {
		t.Fatal("Expected 1 doctor to be assigned to patient case")
	} else if doctorAssignments[0].ProviderId != doctorID {
		t.Fatal("Expected the doctor assigned to be the doctor in the system but it wasnt")
	} else if doctorAssignments[0].Status != api.STATUS_TEMP {
		t.Fatal("Expected the doctor to have temp access but it didn't")
	}

	// ensure that doctor is temporarily assigned to patient file
	careTeam, err := testData.DataApi.GetCareTeamForPatient(patientCase.PatientId.Int64())
	if err != nil {
		t.Fatal(err)
	} else if careTeam == nil {
		t.Fatal("Expected care team to exist but it doesn't")
	} else if len(careTeam.Assignments) != 1 {
		t.Fatalf("Expected 1 doctor to exist in care team instead got %d", len(careTeam.Assignments))
	} else if careTeam.Assignments[0].ProviderId != doctorID {
		t.Fatal("Expected the doctor in the system to be assigned to care team but it was not")
	} else if careTeam.Assignments[0].Status != api.STATUS_TEMP {
		t.Fatal("Expected doctor to be temporarily assigned to patient but it wasnt")
	}

	// ensure that item is still returned in the global case queue for this doctor
	// given that it is currently claimed by this doctor
	unclaimedItems, err := testData.DataApi.GetElligibleItemsInUnclaimedQueue(doctorID)
	if err != nil {
		t.Fatal(err)
	} else if len(unclaimedItems) != 1 {
		t.Fatalf("Expected 1 item in the global queue for doctor instead got %d", len(unclaimedItems))
	}

	// for any other doctor also registered in CA, there should be no elligible item
	doctor2 := test_integration.SignupRandomTestDoctorInState("CA", t, testData)
	unclaimedItems, err = testData.DataApi.GetElligibleItemsInUnclaimedQueue(doctor2.DoctorId)
	if err != nil {
		t.Fatal(err)
	} else if len(unclaimedItems) != 0 {
		t.Fatalf("Expected no elligible items in the queue given that it is currently claimed by other doctor instead got %d", len(unclaimedItems))
	}
}

// This test is to ensure that if a test is claimed by a doctor,
// then any attempt by a second doctor to claim the case is forbidden
func TestJBCQ_ForbiddenClaimAttempt(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)

	doctorID := test_integration.GetDoctorIdOfCurrentDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorID)
	if err != nil {
		t.Fatal(err)
	}

	vp, _ := test_integration.SignupAndSubmitPatientVisitForRandomPatient(t, testData, doctor)

	// now lets sign up a second doctor in CA and get the doctor to attempt to claim the case
	d2 := test_integration.SignupRandomTestDoctorInState("CA", t, testData)
	doctor2, err := testData.DataApi.GetDoctorFromId(d2.DoctorId)
	if err != nil {
		t.Fatal(err)
	}

	// attempt for doctor2 to review the visit information
	visitReviewServer := httptest.NewServer(patient_file.NewDoctorPatientVisitReviewHandler(testData.DataApi))
	defer visitReviewServer.Close()

	// ensure that doctor2 is forbidden access to the visit
	var errorResponse map[string]interface{}
	resp, err := testData.AuthGet(visitReviewServer.URL+"?patient_visit_id="+strconv.FormatInt(vp.PatientVisitId, 10), doctor2.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make call to get patient visit review for patient: " + err.Error())
	} else if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("Expected response code %d but got %d", http.StatusForbidden, resp.StatusCode)
	} else if err := json.NewDecoder(resp.Body).Decode(&errorResponse); err != nil {
		t.Fatal(err)
	} else if errorResponse["developer_code"] == nil {
		t.Fatal("Expected developer code but got none")
	} else if developerErrorCode, ok := errorResponse["developer_code"].(string); !ok {
		t.Fatal("Expected developer code to be an string but it wasnt")
	} else if developerErrorCode != strconv.FormatInt(apiservice.DEVELOPER_JBCQ_FORBIDDEN, 10) {
		t.Fatalf("Expected developer code to be %s but it was %s instead", apiservice.DEVELOPER_JBCQ_FORBIDDEN, developerErrorCode)
	}
	resp.Body.Close()

	// attempt for doctor2 to diagnose the visit
	diagnoseServer := httptest.NewServer(patient_visit.NewDiagnosePatientHandler(testData.DataApi, testData.AuthApi, ""))
	defer diagnoseServer.Close()
	answerIntakeRequest := test_integration.PrepareAnswersForDiagnosis(testData, t, vp.PatientVisitId)
	jsonData, err := json.Marshal(&answerIntakeRequest)
	if err != nil {
		t.Fatal(err)
	}

	// ensure that doctor2 is forbidden from diagnosing the visit for the same reason
	resp, err = testData.AuthPost(diagnoseServer.URL, "application/json", bytes.NewReader(jsonData), doctor2.AccountId.Int64())
	if err != nil {
		t.Fatal(err)
	} else if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("Expected response code %d but got %d", http.StatusForbidden, resp.StatusCode)
	} else if err := json.NewDecoder(resp.Body).Decode(&errorResponse); err != nil {
		t.Fatal(err)
	} else if errorResponse["developer_code"] == nil {
		t.Fatal("Expected developer code but got none")
	} else if developerErrorCode, ok := errorResponse["developer_code"].(string); !ok {
		t.Fatal("Expected developer code to be an string but it wasnt")
	} else if developerErrorCode != strconv.FormatInt(apiservice.DEVELOPER_JBCQ_FORBIDDEN, 10) {
		t.Fatalf("Expected developer code to be %s but it was %s instead", apiservice.DEVELOPER_JBCQ_FORBIDDEN, developerErrorCode)
	}
	resp.Body.Close()

	// attempt for doctor2 to pick a treatment plan
	pickTPServer := httptest.NewServer(doctor_treatment_plan.NewDoctorTreatmentPlanHandler(testData.DataApi, nil, nil, false))
	defer pickTPServer.Close()
	jsonData, err = json.Marshal(&doctor_treatment_plan.PickTreatmentPlanRequestData{
		TPParent: &common.TreatmentPlanParent{
			ParentId:   encoding.NewObjectId(vp.PatientVisitId),
			ParentType: common.TPParentTypePatientVisit,
		},
	})

	// ensure that doctor2 is forbiddden from picking a treatment plan for the same reason
	resp, err = testData.AuthPost(pickTPServer.URL, "application/json", bytes.NewReader(jsonData), doctor2.AccountId.Int64())
	if err != nil {
		t.Fatal(err)
	} else if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("Expected response code %d but got %d", http.StatusForbidden, resp.StatusCode)
	} else if err := json.NewDecoder(resp.Body).Decode(&errorResponse); err != nil {
		t.Fatal(err)
	} else if errorResponse["developer_code"] == nil {
		t.Fatal("Expected developer code but got none")
	} else if developerErrorCode, ok := errorResponse["developer_code"].(string); !ok {
		t.Fatal("Expected developer code to be an string but it wasnt")
	} else if developerErrorCode != strconv.FormatInt(apiservice.DEVELOPER_JBCQ_FORBIDDEN, 10) {
		t.Fatalf("Expected developer code to be %s but it was %s instead", apiservice.DEVELOPER_JBCQ_FORBIDDEN, developerErrorCode)
	}
	resp.Body.Close()

}
