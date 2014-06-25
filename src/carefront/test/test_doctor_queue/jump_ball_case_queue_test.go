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
	"time"
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

// This test is to ensure that the claim works as expected where it doesn't exist at the time of visit/case creation
// and then once a doctor temporarily claims the case, the claim can be extended as expected
func TestJBCQ_Claim(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)
	doctor, err := testData.DataApi.GetDoctorFromId(test_integration.GetDoctorIdOfCurrentDoctor(testData, t))
	if err != nil {
		t.Fatal(err)
	}

	pr := test_integration.SignupRandomTestPatient(t, testData)
	pv := test_integration.CreatePatientVisitForPatient(pr.Patient.PatientId.Int64(), testData, t)
	answerIntakeRequestBody := test_integration.PrepareAnswersForQuestionsInPatientVisit(pv, t)
	test_integration.SubmitAnswersIntakeForPatient(pr.Patient.PatientId.Int64(), pr.Patient.AccountId.Int64(),
		answerIntakeRequestBody, testData, t)
	test_integration.SubmitPatientVisitForPatient(pr.Patient.PatientId.Int64(), pv.PatientVisitId, testData, t)

	// at this point check to ensure that the patient case is the unclaimed state
	patientCase, err := testData.DataApi.GetPatientCaseFromPatientVisitId(pv.PatientVisitId)
	if err != nil {
		t.Fatal(err)
	} else if patientCase.Status != common.PCStatusUnclaimed {
		t.Fatalf("Expected patient case to be %s but it waas %s", common.PCStatusUnclaimed, patientCase.Status)
	}

	test_integration.StartReviewingPatientVisit(pv.PatientVisitId, doctor, testData, t)

	// at this point check to ensure that the patient case has been claimed
	patientCase, err = testData.DataApi.GetPatientCaseFromPatientVisitId(pv.PatientVisitId)
	if err != nil {
		t.Fatal(err)
	} else if patientCase.Status != common.PCStatusTempClaimed {
		t.Fatalf("Expected patient case to be %s but it waas %s", common.PCStatusTempClaimed, patientCase.Status)
	}

	// at this point the claim should exist
	claimExpirationTime := getExpiresTimeFromDoctorForCase(testData, t, patientCase.Id.Int64())
	if claimExpirationTime == nil {
		t.Fatal("Expected claim expiration time to exist")
	}

	// CHECK CLAIM EXTENSION AFTER DIAGNOSING PATIENT
	time.Sleep(time.Second)
	test_integration.SubmitPatientVisitDiagnosis(pv.PatientVisitId, doctor, testData, t)
	claimExpirationTime2 := getExpiresTimeFromDoctorForCase(testData, t, patientCase.Id.Int64())
	if claimExpirationTime2 == nil || !claimExpirationTime.Before(*claimExpirationTime2) {
		t.Fatal("Expected the claim to have been extended but it wasn't")
	}
	claimExpirationTime = claimExpirationTime2

	// CHECK CLAIM EXTENSION AFTER PICKING TREATMENT PLAN
	time.Sleep(time.Second)
	tp := test_integration.PickATreatmentPlanForPatientVisit(pv.PatientVisitId, doctor, nil, testData, t).TreatmentPlan
	claimExpirationTime2 = getExpiresTimeFromDoctorForCase(testData, t, tp.PatientCaseId.Int64())
	// ensure that the time is not null
	if claimExpirationTime == nil {
		t.Fatal("Expected to have a claim expiration time")
	}
	claimExpirationTime = claimExpirationTime2

	// CHECK CLAIM EXTENSION AFTER ADDING TREATMENTS
	time.Sleep(time.Second)
	test_integration.AddAndGetTreatmentsForPatientVisit(testData, []*common.Treatment{}, doctor.AccountId.Int64(), tp.Id.Int64(), t)
	claimExpirationTime2 = getExpiresTimeFromDoctorForCase(testData, t, tp.PatientCaseId.Int64())
	if claimExpirationTime2 == nil || !claimExpirationTime.Before(*claimExpirationTime2) {
		t.Fatal("Expected the claim to have been extended but it wasn't")
	}
	claimExpirationTime = claimExpirationTime2

	// CHECK CLAIM EXTENSION AFTER CREATING REGIMEN PLAN
	time.Sleep(time.Second)
	test_integration.CreateRegimenPlanForTreatmentPlan(&common.RegimenPlan{
		TreatmentPlanId: tp.Id,
	}, testData, doctor, t)

	claimExpirationTime2 = getExpiresTimeFromDoctorForCase(testData, t, tp.PatientCaseId.Int64())
	if claimExpirationTime2 == nil || !claimExpirationTime.Before(*claimExpirationTime2) {
		t.Fatal("Expected the claim to have been extended but it wasn't")
	}
	claimExpirationTime = claimExpirationTime2

	// CHECK CLAIM EXTENSION AFTER ADDING ADVICE
	time.Sleep(time.Second)
	test_integration.UpdateAdvicePointsForPatientVisit(&common.Advice{
		TreatmentPlanId: tp.Id,
	}, testData, doctor, t)

	claimExpirationTime2 = getExpiresTimeFromDoctorForCase(testData, t, tp.PatientCaseId.Int64())
	if claimExpirationTime2 == nil || !claimExpirationTime.Before(*claimExpirationTime2) {
		t.Fatal("Expected the claim to have been extended but it wasn't")
	}

	// CHECK CLAIM COMPLETION ON SUBMISSION OF TREATMENT PLAN
	// Now, the doctor should've permenantly claimed the case
	test_integration.SubmitPatientVisitBackToPatient(tp.Id.Int64(), doctor, testData, t)

	// patient case should be in claimed state
	patientCase, err = testData.DataApi.GetPatientCaseFromId(tp.PatientCaseId.Int64())
	if err != nil {
		t.Fatal(err)
	} else if patientCase.Status != common.PCStatusClaimed {
		t.Fatalf("Expected patietn case to be in %s state instead it was in %s state", common.PCStatusClaimed, patientCase.Status)
	}

	// doctor should be permenantly assigned to the case
	doctorAssignments, err := testData.DataApi.GetDoctorsAssignedToPatientCase(patientCase.Id.Int64())
	if err != nil {
		t.Fatal(err)
	} else if doctorAssignments[0].Status != api.STATUS_ACTIVE {
		t.Fatal("Expected the doctor to be permanently assigned to the patient case but it wasnt")
	} else if doctorAssignments[0].Expires != nil {
		t.Fatal("Expected no expiration date to be set on the assignment but there was one")
	}

	// The doctor should also be permenanently assigned to the careteam of the patient
	careTeam, err := testData.DataApi.GetCareTeamForPatient(patientCase.PatientId.Int64())
	if err != nil {
		t.Fatal(err)
	} else if careTeam.Assignments[0].Status != api.STATUS_ACTIVE {
		t.Fatal("Expected the doctor to be permanently assigned to the care team but it wasn't")
	} else if careTeam.Assignments[0].Expires != nil {
		t.Fatal("Expected there to be no expiration time on the assignment but there was")
	}

	// There should no longer be an unclaimed item in the doctor queue
	unclaimedItems, err := testData.DataApi.GetElligibleItemsInUnclaimedQueue(doctor.DoctorId.Int64())
	if err != nil {
		t.Fatal(err)
	} else if len(unclaimedItems) != 0 {
		t.Fatalf("Expected 0 items in the global queue but got %d", len(unclaimedItems))
	}
}

func getExpiresTimeFromDoctorForCase(testData *test_integration.TestData, t *testing.T, patientCaseId int64) *time.Time {
	doctorAssignments, err := testData.DataApi.GetDoctorsAssignedToPatientCase(patientCaseId)
	if err != nil {
		t.Fatal(err)
	}
	return doctorAssignments[0].Expires
}
