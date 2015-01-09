package test_doctor_queue

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/apiservice/apipaths"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/doctor_queue"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/messages"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

// This test is to ensure that the a case is correctly
// temporarily claimed by a doctor
func TestJBCQ_TempCaseClaim(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	iassert := test_integration.NewAssertion(testData, t)

	doctorID := test_integration.GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	test.OK(t, err)

	vp, _ := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	// ensure that the test is temporarily claimed
	patientCase, err := testData.DataAPI.GetPatientCaseFromPatientVisitID(vp.PatientVisitID)
	if err != nil {
		t.Fatal(err)
	} else if patientCase.Status != common.PCStatusTempClaimed {
		t.Fatalf("Expected the patientCase status to be %s but it was %s", common.PCStatusTempClaimed, patientCase.Status)
	}

	// Assert that our doctor is assigned to our case
	iassert.ProviderIsAssignedToCase(patientCase.ID.Int64(), doctorID, api.STATUS_TEMP)

	// ensure that doctor is temporarily assigned to patient file
	iassert.ProviderIsMemberOfCareTeam(patientCase.PatientID.Int64Value, doctorID, patientCase.ID.Int64Value, api.STATUS_TEMP)

	// ensure that item is still returned in the global case queue for this doctor
	// given that it is currently claimed by this doctor
	if unclaimedItems := getUnclaimedItemsForDoctor(doctorID, t, testData); len(unclaimedItems) != 1 {
		t.Fatalf("Expected 1 item in the global queue for doctor instead got %d", len(unclaimedItems))
	}

	// for any other doctor also registered in CA, there should be no elligible item
	doctor2 := test_integration.SignupRandomTestDoctorInState("CA", t, testData)
	if unclaimedItems := getUnclaimedItemsForDoctor(doctor2.DoctorID, t, testData); len(unclaimedItems) != 0 {
		t.Fatalf("Expected no elligible items in the queue given that it is currently claimed by other doctor instead got %d", len(unclaimedItems))
	}
}

// This test is to ensure that if a test is claimed by a doctor,
// then any attempt by a second doctor to claim the case is forbidden
func TestJBCQ_ForbiddenClaimAttempt(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	doctorID := test_integration.GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	test.OK(t, err)

	vp, _ := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	// now lets sign up a second doctor in CA and get the doctor to attempt to claim the case
	d2 := test_integration.SignupRandomTestDoctorInState("CA", t, testData)
	doctor2, err := testData.DataAPI.GetDoctorFromID(d2.DoctorID)
	test.OK(t, err)
	doctor2Cli := test_integration.DoctorClient(testData, t, d2.DoctorID)

	// attempt for doctor2 to review the visit information
	// ensure that doctor2 is forbidden access to the visit
	var errorResponse map[string]interface{}
	resp, err := testData.AuthGet(testData.APIServer.URL+apipaths.DoctorVisitReviewURLPath+"?patient_visit_id="+strconv.FormatInt(vp.PatientVisitID, 10), doctor2.AccountID.Int64())
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
		t.Fatalf("Expected developer code to be %d but it was %s instead", apiservice.DEVELOPER_JBCQ_FORBIDDEN, developerErrorCode)
	}
	resp.Body.Close()

	// attempt for doctor2 to diagnose the visit
	answerIntakeRequest := test_integration.PrepareAnswersForDiagnosis(testData, t, vp.PatientVisitID)
	jsonData, err := json.Marshal(&answerIntakeRequest)
	test.OK(t, err)

	// ensure that doctor2 is forbidden from diagnosing the visit for the same reason
	resp, err = testData.AuthPost(testData.APIServer.URL+apipaths.DoctorVisitDiagnosisURLPath, "application/json", bytes.NewReader(jsonData), doctor2.AccountID.Int64())
	test.OK(t, err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("Expected response code %d but got %d", http.StatusForbidden, resp.StatusCode)
	} else if err := json.NewDecoder(resp.Body).Decode(&errorResponse); err != nil {
		t.Fatal(err)
	} else if errorResponse["developer_code"] == nil {
		t.Fatal("Expected developer code but got none")
	} else if developerErrorCode, ok := errorResponse["developer_code"].(string); !ok {
		t.Fatal("Expected developer code to be an string but it wasnt")
	} else if developerErrorCode != strconv.FormatInt(apiservice.DEVELOPER_JBCQ_FORBIDDEN, 10) {
		t.Fatalf("Expected developer code to be %d but it was %s instead", apiservice.DEVELOPER_JBCQ_FORBIDDEN, developerErrorCode)
	}

	// ensure that doctor2 is forbiddden from picking a treatment plan for the same reason
	if _, err := doctor2Cli.PickTreatmentPlanForVisit(vp.PatientVisitID, nil); err == nil {
		t.Fatal("Expected StatusForbidden but got no error")
	} else if e, ok := err.(*apiservice.SpruceError); !ok {
		t.Fatalf("Expected a SpruceError. Got %T: %s", err, err.Error())
	} else if e.HTTPStatusCode != http.StatusForbidden {
		t.Fatalf("Expectes status StatusForbidden got %d", e.HTTPStatusCode)
	} else if e.DeveloperErrorCode != apiservice.DEVELOPER_JBCQ_FORBIDDEN {
		t.Fatalf("Expected developer code to be %d but it was %d instead", apiservice.DEVELOPER_JBCQ_FORBIDDEN, e.DeveloperErrorCode)
	}
}

// This test is to ensure that the claim works as expected where it doesn't exist at the time of visit/case creation
// and then once a doctor temporarily claims the case, the claim can be extended as expected
func TestJBCQ_Claim(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	iassert := test_integration.NewAssertion(testData, t)

	doctor, err := testData.DataAPI.GetDoctorFromID(test_integration.GetDoctorIDOfCurrentDoctor(testData, t))
	test.OK(t, err)
	cli := test_integration.DoctorClient(testData, t, doctor.DoctorID.Int64())

	pv := test_integration.CreateRandomPatientVisitInState("CA", t, testData)

	// at this point check to ensure that the patient case is the unclaimed state
	patientCase, err := testData.DataAPI.GetPatientCaseFromPatientVisitID(pv.PatientVisitID)
	if err != nil {
		t.Fatal(err)
	} else if patientCase.Status != common.PCStatusUnclaimed {
		t.Fatalf("Expected patient case to be %s but it waas %s", common.PCStatusUnclaimed, patientCase.Status)
	}

	test_integration.StartReviewingPatientVisit(pv.PatientVisitID, doctor, testData, t)

	// at this point check to ensure that the patient case has been claimed
	patientCase, err = testData.DataAPI.GetPatientCaseFromPatientVisitID(pv.PatientVisitID)
	if err != nil {
		t.Fatal(err)
	} else if patientCase.Status != common.PCStatusTempClaimed {
		t.Fatalf("Expected patient case to be %s but it waas %s", common.PCStatusTempClaimed, patientCase.Status)
	}

	// at this point the claim should exist
	claimExpirationTime := getExpiresTimeFromDoctorForCase(testData, t, patientCase.ID.Int64())
	if claimExpirationTime == nil {
		t.Fatal("Expected claim expiration time to exist")
	}

	// CHECK CLAIM EXTENSION AFTER DIAGNOSING PATIENT
	test_integration.SubmitPatientVisitDiagnosis(pv.PatientVisitID, doctor, testData, t)
	claimExpirationTime2 := getExpiresTimeFromDoctorForCase(testData, t, patientCase.ID.Int64())
	if claimExpirationTime2 == nil || !claimExpirationTime.Before(*claimExpirationTime2) {
		t.Fatal("Expected the claim to have been extended but it wasn't")
	}
	claimExpirationTime = claimExpirationTime2

	// CHECK CLAIM EXTENSION AFTER PICKING TREATMENT PLAN
	tp := test_integration.PickATreatmentPlanForPatientVisit(pv.PatientVisitID, doctor, nil, testData, t).TreatmentPlan
	claimExpirationTime2 = getExpiresTimeFromDoctorForCase(testData, t, tp.PatientCaseID.Int64())
	// ensure that the time is not null
	if claimExpirationTime == nil {
		t.Fatal("Expected to have a claim expiration time")
	}
	claimExpirationTime = claimExpirationTime2

	// CHECK CLAIM EXTENSION AFTER ADDING TREATMENTS
	test_integration.AddAndGetTreatmentsForPatientVisit(testData, []*common.Treatment{}, doctor.AccountID.Int64(), tp.ID.Int64(), t)
	claimExpirationTime2 = getExpiresTimeFromDoctorForCase(testData, t, tp.PatientCaseID.Int64())
	if claimExpirationTime2 == nil || !claimExpirationTime.Before(*claimExpirationTime2) {
		t.Fatal("Expected the claim to have been extended but it wasn't")
	}
	claimExpirationTime = claimExpirationTime2

	// CHECK CLAIM EXTENSION AFTER CREATING REGIMEN PLAN
	if _, err := cli.CreateRegimenPlan(&common.RegimenPlan{TreatmentPlanID: tp.ID}); err != nil {
		t.Fatal(err)
	}
	claimExpirationTime2 = getExpiresTimeFromDoctorForCase(testData, t, tp.PatientCaseID.Int64())
	if claimExpirationTime2 == nil || !claimExpirationTime.Before(*claimExpirationTime2) {
		t.Fatal("Expected the claim to have been extended but it wasn't")
	}
	claimExpirationTime = claimExpirationTime2

	// CHECK CLAIM EXTENSION AFTER UPDATING NOTE
	err = cli.UpdateTreatmentPlanNote(tp.ID.Int64(), "foo ")
	test.OK(t, err)
	claimExpirationTime2 = getExpiresTimeFromDoctorForCase(testData, t, tp.PatientCaseID.Int64())
	if claimExpirationTime2 == nil || !claimExpirationTime.Before(*claimExpirationTime2) {
		t.Fatal("Expected the claim to have been extended but it wasn't")
	}
	claimExpirationTime = claimExpirationTime2

	// CHECK CLAIM EXTENSION AFTER UPDATING SCHEDULED MESSAGE
	_, err = cli.CreateTreatmentPlanScheduledMessage(tp.ID.Int64(), &doctor_treatment_plan.ScheduledMessage{
		ScheduledDays: 7*4 + 1,
		Message:       "Hello, welcome",
		Attachments: []*messages.Attachment{
			{
				Type: common.AttachmentTypeFollowupVisit,
			},
		},
	})
	test.OK(t, err)
	claimExpirationTime2 = getExpiresTimeFromDoctorForCase(testData, t, tp.PatientCaseID.Int64())
	if claimExpirationTime2 == nil || !claimExpirationTime.Before(*claimExpirationTime2) {
		t.Fatal("Expected the claim to have been extended but it wasn't")
	}
	claimExpirationTime = claimExpirationTime2

	// CHECK CLAIM EXTENSION AFTER UPDATING RESOURCE GUIDES
	_, guideIDs := test_integration.CreateTestResourceGuides(t, testData)
	test.OK(t, cli.AddResourceGuidesToTreatmentPlan(tp.ID.Int64(), guideIDs))
	claimExpirationTime2 = getExpiresTimeFromDoctorForCase(testData, t, tp.PatientCaseID.Int64())
	if claimExpirationTime2 == nil || !claimExpirationTime.Before(*claimExpirationTime2) {
		t.Fatal("Expected the claim to have been extended but it wasn't")
	}
	claimExpirationTime = claimExpirationTime2

	// CHECK CLAIM COMPLETION ON SUBMISSION OF TREATMENT PLAN
	// Now, the doctor should've permenantly claimed the case
	test_integration.SubmitPatientVisitBackToPatient(tp.ID.Int64(), doctor, testData, t)

	// patient case should be in claimed state
	patientCase, err = testData.DataAPI.GetPatientCaseFromID(tp.PatientCaseID.Int64())
	if err != nil {
		t.Fatal(err)
	} else if patientCase.Status != common.PCStatusClaimed {
		t.Fatalf("Expected patietn case to be in %s state instead it was in %s state", common.PCStatusClaimed, patientCase.Status)
	}

	// doctor should be permenantly assigned to the case
	// Assert that our doctor is assigned to our
	iassert.ProviderIsAssignedToCase(patientCase.ID.Int64(), doctor.DoctorID.Int64(), api.STATUS_ACTIVE)

	// The doctor should also be permenanently assigned to the careteam of the patient
	careTeam, err := testData.DataAPI.GetCareTeamForPatient(patientCase.PatientID.Int64())
	if err != nil {
		t.Fatal(err)
	} else if careTeam.Assignments[0].Status != api.STATUS_ACTIVE {
		t.Fatal("Expected the doctor to be permanently assigned to the care team but it wasn't")
	} else if careTeam.Assignments[0].Expires != nil {
		t.Fatal("Expected there to be no expiration time on the assignment but there was")
	}

	// There should no longer be an unclaimed item in the doctor queue
	if unclaimedItems := getUnclaimedItemsForDoctor(doctor.DoctorID.Int64(), t, testData); len(unclaimedItems) != 0 {
		t.Fatalf("Expected 0 items in the global queue but got %d", len(unclaimedItems))
	}

	// There should be 1 completed item in the doctor's queue
	completedItems, err := testData.DataAPI.GetCompletedItemsInDoctorQueue(doctor.DoctorID.Int64())
	if err != nil {
		t.Fatal(err)
	} else if len(completedItems) != 1 {
		t.Fatalf("Expected 1 completed item instead got %d", len(completedItems))
	}
}

// This test is to ensure that the doctor is permanently assigned to the
// case in the event that the visit is marked as unsuitable for spruce
func TestJBCQ_AssignOnMarkingUnsuitableForSpruce(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	iassert := test_integration.NewAssertion(testData, t)

	doctor, err := testData.DataAPI.GetDoctorFromID(test_integration.GetDoctorIDOfCurrentDoctor(testData, t))
	test.OK(t, err)

	pv := test_integration.CreateRandomPatientVisitInState("CA", t, testData)
	test_integration.StartReviewingPatientVisit(pv.PatientVisitID, doctor, testData, t)

	intakeData := test_integration.PrepareAnswersForDiagnosingAsUnsuitableForSpruce(testData, t, pv.PatientVisitID)
	test_integration.SubmitPatientVisitDiagnosisWithIntake(pv.PatientVisitID, doctor.AccountID.Int64(), intakeData, testData, t)

	// at this point the patient case should be considered claimed
	iassert.CaseStatusFromVisitIs(pv.PatientVisitID, common.PCStatusClaimed)
}

// This test is to ensure that the case gets permanently assigned to the doctor
// if a doctor sends a message to the patient while the case is unclaimed.
func TestJBCQ_PermanentlyAssigningCaseOnMessagePost(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	doctor, err := testData.DataAPI.GetDoctorFromID(test_integration.GetDoctorIDOfCurrentDoctor(testData, t))
	test.OK(t, err)
	doctorCli := test_integration.DoctorClient(testData, t, doctor.DoctorID.Int64())

	pv := test_integration.CreateRandomPatientVisitInState("CA", t, testData)
	test_integration.StartReviewingPatientVisit(pv.PatientVisitID, doctor, testData, t)

	patientCaseID, err := testData.DataAPI.GetPatientCaseIDFromPatientVisitID(pv.PatientVisitID)
	test.OK(t, err)

	// Grant the doctor access to the case
	test_integration.GrantDoctorAccessToPatientCase(t, testData, doctor, patientCaseID)

	// Send a message from the doctor to the patient
	_, err = doctorCli.PostCaseMessage(patientCaseID, "Foo", nil)
	test.OK(t, err)

	// the case should now be permanently assigned to the doctor
	patientCase, err := testData.DataAPI.GetPatientCaseFromID(patientCaseID)
	if err != nil {
		t.Fatal(err)
	} else if patientCase.Status != common.PCStatusClaimed {
		t.Fatalf("Expected case to have status %s instead it had status %s", common.PCStatusClaimed, patientCase.Status)
	}

	// there should be a pending item in the doctor's queue to represnt the visit
	pendingItems, err := testData.DataAPI.GetPendingItemsInDoctorQueue(doctor.DoctorID.Int64())
	if err != nil {
		t.Fatal(err)
	} else if len(pendingItems) != 1 {
		t.Fatalf("Expected %d items in the doctor queue instead got %d", 1, len(pendingItems))
	}

	// there should be no unclaimed items in the case queue
	unclaimedItems, err := testData.DataAPI.GetElligibleItemsInUnclaimedQueue(doctor.DoctorID.Int64())
	if err != nil {
		t.Fatal(err)
	} else if len(unclaimedItems) != 0 {
		t.Fatalf("Expected %d items but got %d items instad", 0, len(unclaimedItems))
	}
}

// This test is to ensure that the doctor's claim on a case is revoked after a
// period of inactivity has elapsed
func TestJBCQ_RevokingAccessOnClaimExpiration(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	doctor, err := testData.DataAPI.GetDoctorFromID(test_integration.GetDoctorIDOfCurrentDoctor(testData, t))
	test.OK(t, err)

	pv, _ := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	doctor_queue.CheckForExpiredClaimedItems(testData.DataAPI, testData.Config.AnalyticsLogger, metrics.NewCounter(), metrics.NewCounter())

	// because of the grace period, the doctor's claim should not have been revoked
	patientCase, err := testData.DataAPI.GetPatientCaseFromPatientVisitID(pv.PatientVisitID)
	if err != nil {
		t.Fatal(err)
	} else if patientCase.Status != common.PCStatusTempClaimed {
		t.Fatalf("Expected the status to be %s but it was %s", common.PCStatusTempClaimed, patientCase.Status)
	}

	// now lets update the expired time on the unclaimed_case_queue beyond the (expiration + grace period)
	_, err = testData.DB.Exec(`update unclaimed_case_queue set expires = ? where patient_case_id = ?`, time.Now().Add(-(doctor_queue.ExpireDuration + doctor_queue.GracePeriod + time.Minute)), patientCase.ID.Int64())
	test.OK(t, err)
	doctor_queue.CheckForExpiredClaimedItems(testData.DataAPI, testData.Config.AnalyticsLogger, metrics.NewCounter(), metrics.NewCounter())

	// at this point the access should have been revoked
	patientCase, err = testData.DataAPI.GetPatientCaseFromPatientVisitID(pv.PatientVisitID)
	if err != nil {
		t.Fatal(err)
	} else if patientCase.Status != common.PCStatusUnclaimed {
		t.Fatalf("Expected the status to be %s but it was %s", common.PCStatusUnclaimed, patientCase.Status)
	}

	// now that the access is revoked, the patient case or file should not have a doctor assigned to it
	doctorAssignments, err := testData.DataAPI.GetDoctorsAssignedToPatientCase(patientCase.ID.Int64())
	if err != nil {
		t.Fatal(err)
	} else if len(doctorAssignments) != 0 {
		t.Fatalf("Expected 0 doctors assigned to case instead got %d", len(doctorAssignments))
	}
	careTeam, err := testData.DataAPI.GetCareTeamForPatient(patientCase.PatientID.Int64())
	if err != nil {
		t.Fatal(err)
	} else if len(careTeam.Assignments) != 0 {
		t.Fatalf("Expected 0 doctors as part of the patient's care team instead got %d", len(careTeam.Assignments))
	}

	// now let's try and get another doctor to claim the item
	d2 := test_integration.SignupRandomTestDoctorInState("CA", t, testData)
	doctor2, err := testData.DataAPI.GetDoctorFromID(d2.DoctorID)
	test.OK(t, err)
	test_integration.StartReviewingPatientVisit(pv.PatientVisitID, doctor2, testData, t)

	// the patient case should now be claimed by this doctor
	patientCase, err = testData.DataAPI.GetPatientCaseFromPatientVisitID(pv.PatientVisitID)
	if err != nil {
		t.Fatal(err)
	} else if patientCase.Status != common.PCStatusTempClaimed {
		t.Fatalf("Expected the status to be %s but it was %s", common.PCStatusTempClaimed, patientCase.Status)
	}
}
