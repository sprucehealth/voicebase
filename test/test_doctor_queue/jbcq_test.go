package test_doctor_queue

import (
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
	"github.com/sprucehealth/backend/diagnosis/handlers"
	"github.com/sprucehealth/backend/doctor_queue"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

// This test is to ensure that the a case is correctly
// temporarily claimed by a doctor
func TestJBCQ_TempCaseClaim(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)
	iassert := test_integration.NewAssertion(testData, t)

	doctorID := test_integration.GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	test.OK(t, err)

	vp := test_integration.CreateRandomPatientVisitInState("CA", t, testData)
	test_integration.StartReviewingPatientVisit(vp.PatientVisitID, doctor, testData, t)

	// ensure that the test is temporarily claimed
	patientCase, err := testData.DataAPI.GetPatientCaseFromPatientVisitID(vp.PatientVisitID)
	test.OK(t, err)
	test.Equals(t, common.PCStatusActive, patientCase.Status)
	test.Equals(t, false, patientCase.Claimed)

	// Assert that our doctor is assigned to our case
	iassert.ProviderIsAssignedToCase(patientCase.ID.Int64(), doctorID, api.StatusTemp)

	// ensure that doctor is temporarily assigned to patient file
	iassert.ProviderIsMemberOfCareTeam(doctorID, patientCase.ID.Int64(), api.StatusTemp)

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

// This test is to ensure that if a visit is claimed by a doctor,
// then any attempt by a second doctor to claim the case is forbidden
func TestJBCQ_ForbiddenClaimAttempt(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	doctorID := test_integration.GetDoctorIDOfCurrentDoctor(testData, t)
	dc := test_integration.DoctorClient(testData, t, doctorID)

	vp := test_integration.CreateRandomPatientVisitInState("CA", t, testData)

	// claim the case by the first doctor
	patientCase, err := testData.DataAPI.GetPatientCaseFromPatientVisitID(vp.PatientVisitID)
	test.OK(t, err)
	test.OK(t, dc.ClaimCase(patientCase.ID.Int64()))

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
	} else if developerErrorCode != strconv.FormatInt(apiservice.DeveloperErrorJBCQForbidden, 10) {
		t.Fatalf("Expected developer code to be %d but it was %s instead", apiservice.DeveloperErrorJBCQForbidden, developerErrorCode)
	}
	resp.Body.Close()

	// attempt for doctor2 to diagnose the visit
	// ensure that doctor2 is forbidden from diagnosing the visit for the same reason
	err = doctor2Cli.CreateDiagnosisSet(&handlers.DiagnosisListRequestData{
		VisitID: vp.PatientVisitID,
		CaseManagement: handlers.CaseManagementItem{
			Unsuitable: true,
		},
	})
	test.Equals(t, true, err != nil)
	sErr := err.(*apiservice.SpruceError)
	test.Equals(t, int64(apiservice.DeveloperErrorJBCQForbidden), sErr.DeveloperErrorCode)
	test.Equals(t, http.StatusForbidden, sErr.HTTPStatusCode)

	// ensure that doctor2 is forbiddden from picking a treatment plan for the same reason
	if _, err := doctor2Cli.PickTreatmentPlanForVisit(vp.PatientVisitID, nil); err == nil {
		t.Fatal("Expected StatusForbidden but got no error")
	} else if e, ok := err.(*apiservice.SpruceError); !ok {
		t.Fatalf("Expected a SpruceError. Got %T: %s", err, err.Error())
	} else if e.HTTPStatusCode != http.StatusForbidden {
		t.Fatalf("Expectes status StatusForbidden got %d", e.HTTPStatusCode)
	} else if e.DeveloperErrorCode != apiservice.DeveloperErrorJBCQForbidden {
		t.Fatalf("Expected developer code to be %d but it was %d instead", apiservice.DeveloperErrorJBCQForbidden, e.DeveloperErrorCode)
	}
}

// This test is to ensure that the claim works as expected where it doesn't exist at the time of visit/case creation
// and then once a doctor temporarily claims the case, the claim can be extended as expected
func TestJBCQ_Claim(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)
	iassert := test_integration.NewAssertion(testData, t)

	doctor, err := testData.DataAPI.GetDoctorFromID(test_integration.GetDoctorIDOfCurrentDoctor(testData, t))
	test.OK(t, err)
	cli := test_integration.DoctorClient(testData, t, doctor.ID.Int64())

	pv := test_integration.CreateRandomPatientVisitInState("CA", t, testData)

	// at this point check to ensure that the patient case is the unclaimed state
	patientCase, err := testData.DataAPI.GetPatientCaseFromPatientVisitID(pv.PatientVisitID)
	test.OK(t, err)
	test.Equals(t, common.PCStatusActive, patientCase.Status)
	test.Equals(t, false, patientCase.Claimed)

	test_integration.StartReviewingPatientVisit(pv.PatientVisitID, doctor, testData, t)

	// at this point check to ensure that the case has been claimed
	members, err := testData.DataAPI.GetDoctorsAssignedToPatientCase(patientCase.ID.Int64())
	test.OK(t, err)

	var tempDoctorIDFound int64
	for _, member := range members {
		if member.Status == api.StatusTemp {
			tempDoctorIDFound = member.ProviderID
			break
		}
	}
	test.Equals(t, doctor.ID.Int64(), tempDoctorIDFound)

	// at this point the claim should exist
	claimExpirationTime := getExpiresTimeFromDoctorForCase(testData, t, patientCase.ID.Int64())
	if claimExpirationTime == nil {
		t.Fatal("Expected claim expiration time to exist")
	}

	// CHECK CLAIM EXTENSION AFTER DIAGNOSING PATIENT
	if err := cli.CreateDiagnosisSet(&handlers.DiagnosisListRequestData{
		VisitID: pv.PatientVisitID,
		Diagnoses: []*handlers.DiagnosisInputItem{
			{
				CodeID: "diag_l780",
			},
		},
	}); err != nil {
		t.Fatal(err)
	}

	claimExpirationTime2 := getExpiresTimeFromDoctorForCase(testData, t, patientCase.ID.Int64())
	if claimExpirationTime2 == nil || !claimExpirationTime.Before(*claimExpirationTime2) {
		t.Fatal("Expected the claim to have been extended but it wasn't")
	}
	claimExpirationTime = claimExpirationTime2

	// CHECK CLAIM COMPLETION ON STARTING OF TREATMENT PLAN
	tp := test_integration.PickATreatmentPlanForPatientVisit(pv.PatientVisitID, doctor, nil, testData, t).TreatmentPlan
	// patient case should be in active, claimed state
	patientCase, err = testData.DataAPI.GetPatientCaseFromID(tp.PatientCaseID.Int64())
	if err != nil {
		t.Fatal(err)
	} else if !patientCase.Claimed {
		t.Fatalf("Expected case to be claimed")
	} else if patientCase.Status != common.PCStatusActive {
		t.Fatalf("Expected the case to be in %s state but it was in %s state", common.PCStatusActive, patientCase.Status)
	}

	// doctor should be permenantly assigned to the case
	// Assert that our doctor is assigned to our
	iassert.ProviderIsAssignedToCase(patientCase.ID.Int64(), doctor.ID.Int64(), api.StatusActive)

	// The doctor should also be permenanently assigned to the careteam of the patient
	careTeams, err := testData.DataAPI.CaseCareTeams([]int64{patientCase.ID.Int64()})
	if err != nil {
		t.Fatal(err)
	} else if len(careTeams) != 1 {
		t.Fatalf("Expected single care team for case but got %d", len(careTeams))
	} else if careTeams[patientCase.ID.Int64()] == nil {
		t.Fatalf("Expected patient case to exist for caseID %d but it doesnt", patientCase.ID.Int64())
	}

	careTeam := careTeams[patientCase.ID.Int64()]
	if careTeam.Assignments[0].Status != api.StatusActive {
		t.Fatal("Expected the doctor to be permanently assigned to the care team but it wasn't")
	} else if careTeam.Assignments[0].Expires != nil {
		t.Fatal("Expected there to be no expiration time on the assignment but there was")
	}

	// There should no longer be an unclaimed item in the doctor queue
	if unclaimedItems := getUnclaimedItemsForDoctor(doctor.ID.Int64(), t, testData); len(unclaimedItems) != 0 {
		t.Fatalf("Expected 0 items in the global queue but got %d", len(unclaimedItems))
	}

	// There should be 1 completed item in the doctor's queue
	test.OK(t, cli.UpdateTreatmentPlanNote(tp.ID.Int64(), "foo"))
	test.OK(t, cli.SubmitTreatmentPlan(tp.ID.Int64()))
	completedItems, err := testData.DataAPI.GetCompletedItemsInDoctorQueue(doctor.ID.Int64())
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
	defer testData.Close(t)
	testData.StartAPIServer(t)

	doctor, err := testData.DataAPI.GetDoctorFromID(test_integration.GetDoctorIDOfCurrentDoctor(testData, t))
	test.OK(t, err)
	doctorCLI := test_integration.DoctorClient(testData, t, doctor.ID.Int64())

	pv := test_integration.CreateRandomPatientVisitInState("CA", t, testData)
	test_integration.StartReviewingPatientVisit(pv.PatientVisitID, doctor, testData, t)
	test.OK(t, doctorCLI.CreateDiagnosisSet(&handlers.DiagnosisListRequestData{
		VisitID: pv.PatientVisitID,
		CaseManagement: handlers.CaseManagementItem{
			Unsuitable: true,
		},
	}))

	// at this point the patient case should be considered claimed
	patientCase, err := testData.DataAPI.GetPatientCaseFromPatientVisitID(pv.PatientVisitID)
	test.OK(t, err)
	test.Equals(t, patientCase.Status, common.PCStatusActive)
	test.Equals(t, true, patientCase.Claimed)
}

// This test is to ensure that the case gets permanently assigned to the doctor
// if a doctor sends a message to the patient while the case is unclaimed.
func TestJBCQ_PermanentlyAssigningCaseOnMessagePost(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	doctor, err := testData.DataAPI.GetDoctorFromID(test_integration.GetDoctorIDOfCurrentDoctor(testData, t))
	test.OK(t, err)
	doctorCli := test_integration.DoctorClient(testData, t, doctor.ID.Int64())

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
	test.OK(t, err)
	test.Equals(t, common.PCStatusActive, patientCase.Status)
	test.Equals(t, true, patientCase.Claimed)

	// there should be a pending item in the doctor's queue to represnt the visit
	pendingItems, err := testData.DataAPI.GetPendingItemsInDoctorQueue(doctor.ID.Int64())
	if err != nil {
		t.Fatal(err)
	} else if len(pendingItems) != 1 {
		t.Fatalf("Expected %d items in the doctor queue instead got %d", 1, len(pendingItems))
	}

	// there should be no unclaimed items in the case queue
	unclaimedItems, err := testData.DataAPI.GetElligibleItemsInUnclaimedQueue(doctor.ID.Int64())
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
	defer testData.Close(t)
	testData.StartAPIServer(t)
	doctor, err := testData.DataAPI.GetDoctorFromID(test_integration.GetDoctorIDOfCurrentDoctor(testData, t))
	test.OK(t, err)

	dc := test_integration.DoctorClient(testData, t, doctor.ID.Int64())

	pv := test_integration.CreateRandomPatientVisitInState("CA", t, testData)
	patientCase, err := testData.DataAPI.GetPatientCaseFromPatientVisitID(pv.PatientVisitID)
	test.OK(t, dc.ClaimCase(patientCase.ID.Int64()))
	doctor_queue.CheckForExpiredClaimedItems(testData.DataAPI, testData.Config.AnalyticsLogger, metrics.NewCounter(), metrics.NewCounter())

	// because of the grace period, the doctor's claim should not have been revoked
	patientCase, err = testData.DataAPI.GetPatientCaseFromPatientVisitID(pv.PatientVisitID)
	test.OK(t, err)
	test.Equals(t, common.PCStatusActive, patientCase.Status)
	test.Equals(t, false, patientCase.Claimed)

	// now lets update the expired time on the unclaimed_case_queue beyond the (expiration + grace period)
	_, err = testData.DB.Exec(`update unclaimed_case_queue set expires = ? where patient_case_id = ?`, time.Now().Add(-(doctor_queue.ExpireDuration + doctor_queue.GracePeriod + time.Minute)), patientCase.ID.Int64())
	test.OK(t, err)
	doctor_queue.CheckForExpiredClaimedItems(testData.DataAPI, testData.Config.AnalyticsLogger, metrics.NewCounter(), metrics.NewCounter())

	// at this point the access should have been revoked
	patientCase, err = testData.DataAPI.GetPatientCaseFromPatientVisitID(pv.PatientVisitID)
	test.OK(t, err)
	test.Equals(t, common.PCStatusActive, patientCase.Status)
	test.Equals(t, false, patientCase.Claimed)

	// now that the access is revoked, the patient case or file should not have a doctor assigned to it
	doctorAssignments, err := testData.DataAPI.GetDoctorsAssignedToPatientCase(patientCase.ID.Int64())
	if err != nil {
		t.Fatal(err)
	} else if len(doctorAssignments) != 0 {
		t.Fatalf("Expected 0 doctors assigned to case instead got %d", len(doctorAssignments))
	}
	careTeams, err := testData.DataAPI.CaseCareTeams([]int64{patientCase.ID.Int64()})
	if err != nil {
		t.Fatal(err)
	} else if len(careTeams) != 0 {
		t.Fatalf("Expected 1 care team instead got %d", len(careTeams))
	}

	// now let's try and get another doctor to claim the item
	d2 := test_integration.SignupRandomTestDoctorInState("CA", t, testData)
	doctor2, err := testData.DataAPI.GetDoctorFromID(d2.DoctorID)
	test.OK(t, err)
	dc2 := test_integration.DoctorClient(testData, t, doctor2.ID.Int64())
	dc2.ClaimCase(patientCase.ID.Int64())

	// the patient case should now be claimed by this doctor
	patientCase, err = testData.DataAPI.GetPatientCaseFromPatientVisitID(pv.PatientVisitID)
	test.OK(t, err)
	test.Equals(t, common.PCStatusActive, patientCase.Status)

	members, err := testData.DataAPI.GetDoctorsAssignedToPatientCase(patientCase.ID.Int64())
	test.OK(t, err)
	var tempStatusFound bool
	for _, member := range members {
		if member.Status == api.StatusTemp {
			tempStatusFound = true
			break
		}
	}
	test.Equals(t, true, tempStatusFound)
}
