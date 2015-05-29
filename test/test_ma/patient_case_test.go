package test_ma

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/apiservice/apipaths"
	"github.com/sprucehealth/backend/doctor_queue"
	"github.com/sprucehealth/backend/messages"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

// This test is to ensure that so long as an MA exists, the MA is part
// of every patient care team
func TestMA_PartOfCareTeam(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	mr, _, _ := test_integration.SignupRandomTestCC(t, testData, true)
	ma, err := testData.DataAPI.GetDoctorFromID(mr.DoctorID)
	test.OK(t, err)

	pv := test_integration.CreateRandomPatientVisitInState("CA", t, testData)

	patientVisit, err := testData.DataAPI.GetPatientVisitFromID(pv.PatientVisitID)
	test.OK(t, err)

	assignments, err := testData.DataAPI.GetActiveMembersOfCareTeamForCase(patientVisit.PatientCaseID.Int64(), false)
	test.OK(t, err)
	test.Equals(t, 1, len(assignments))
	test.Equals(t, api.RoleCC, assignments[0].ProviderRole)
	test.Equals(t, ma.ID.Int64(), assignments[0].ProviderID)
}

// This test is to ensure that every patient message is routed to the MA on the patient's care team
func TestMA_RoutePatientMsgsToMA(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	mr, _, _ := test_integration.SignupRandomTestCC(t, testData, true)
	ma, err := testData.DataAPI.GetDoctorFromID(mr.DoctorID)
	test.OK(t, err)

	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)

	_, tp := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	test_integration.SubmitPatientVisitBackToPatient(tp.ID.Int64(), doctor, testData, t)

	patient, err := testData.DataAPI.GetPatientFromID(tp.PatientID)
	test.OK(t, err)

	doctorCli := test_integration.DoctorClient(testData, t, dr.DoctorID)
	patientCli := test_integration.PatientClient(testData, t, patient.PatientID.Int64())

	_, err = patientCli.PostCaseMessage(tp.PatientCaseID.Int64(), "foo", nil)
	test.OK(t, err)

	// this patient message should be in the MA's inbox and not the doctor's
	items, err := testData.DataAPI.GetPendingItemsInDoctorQueue(ma.ID.Int64())
	test.OK(t, err)
	test.Equals(t, 1, len(items))
	test.Equals(t, api.DQEventTypeCaseMessage, items[0].EventType)
	test.Equals(t, tp.PatientCaseID.Int64(), items[0].ItemID)

	// the doctor's queue sould have no pending items
	items, err = testData.DataAPI.GetPendingItemsInDoctorQueue(doctor.ID.Int64())
	test.OK(t, err)
	test.Equals(t, 0, len(items))

	// this should be the case even if the doctor sends a message to the patient; the patient's response should go to the MA
	_, err = doctorCli.PostCaseMessage(tp.PatientCaseID.Int64(), "foo", nil)
	test.OK(t, err)

	_, err = patientCli.PostCaseMessage(tp.PatientCaseID.Int64(), "foo", nil)
	test.OK(t, err)

	items, err = testData.DataAPI.GetPendingItemsInDoctorQueue(ma.ID.Int64())
	test.OK(t, err)
	test.Equals(t, 2, len(items))
	test.Equals(t, api.DQEventTypeCaseAssignment, items[0].EventType)
	test.Equals(t, tp.PatientCaseID.Int64(), items[0].ItemID)
	test.Equals(t, api.DQEventTypeCaseMessage, items[1].EventType)
	test.Equals(t, tp.PatientCaseID.Int64(), items[1].ItemID)

	items, err = testData.DataAPI.GetPendingItemsInDoctorQueue(doctor.ID.Int64())
	test.OK(t, err)
	test.Equals(t, 0, len(items))

}

// This test is to ensure that the MA can assign any case to a doctor
func TestMA_AssignToDoctor(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	mr, _, _ := test_integration.SignupRandomTestCC(t, testData, true)
	ma, err := testData.DataAPI.GetDoctorFromID(mr.DoctorID)
	test.OK(t, err)

	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)

	pv := test_integration.CreateRandomPatientVisitInState("CA", t, testData)

	doctorCli := test_integration.DoctorClient(testData, t, dr.DoctorID)
	maCli := test_integration.DoctorClient(testData, t, ma.ID.Int64())

	pc, err := testData.DataAPI.GetPatientCaseFromPatientVisitID(pv.PatientVisitID)
	if err != nil {
		t.Fatal(err.Error())
	}

	// MA should not be able to assign a case that is not permanently claimed
	if _, err := maCli.AssignCase(pc.ID.Int64(), "testing", nil); err == nil {
		t.Fatal("Expected BadRequest but got no error")
	} else if e, ok := err.(*apiservice.SpruceError); !ok {
		t.Fatalf("Expected SpruceError not %T %+v", err, err)
	} else if e.HTTPStatusCode != 400 {
		t.Fatalf("Expected BadRequest (400) got %d", e.HTTPStatusCode)
	}

	// Once the case is claimed, the MA should be able to assign the case
	test.OK(t, doctorCli.ClaimCase(pc.ID.Int64()))
	tp, err := doctorCli.PickTreatmentPlanForVisit(pv.PatientVisitID, nil)
	test.OK(t, err)

	_, err = maCli.AssignCase(tp.PatientCaseID.Int64(), "testing", nil)
	test.OK(t, err)

	// as a result of the assignment there should be a pending item in the doctor's inbox
	items, err := testData.DataAPI.GetPendingItemsInDoctorQueue(doctor.ID.Int64())
	test.OK(t, err)
	test.Equals(t, 2, len(items))
	test.Equals(t, api.DQEventTypeCaseAssignment, items[1].EventType)

	// MA should be able to assign the same case multiple times
	_, err = maCli.AssignCase(tp.PatientCaseID.Int64(), "testing", nil)
	test.OK(t, err)
	// However, the Doctor should still have a single item in their queue
	items, err = testData.DataAPI.GetPendingItemsInDoctorQueue(doctor.ID.Int64())
	test.OK(t, err)
	test.Equals(t, 2, len(items))
	test.Equals(t, api.DQEventTypeCaseAssignment, items[1].EventType)

	// Lets add another item into the doctor's queue so as to make sure that the assignment moves to the bottom of the queue
	// if the MA assigns the same case multipel times to the doctor
	// To simulate this we will start another case, and have the doctor message the patient so as to cause the case to land up in the doctor's inbox.
	_, tp2 := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	_, err = doctorCli.PostCaseMessage(tp2.PatientCaseID.Int64(), "foo", nil)
	test.OK(t, err)

	// Now lets have the MA assign the case to the doctor again
	_, err = maCli.AssignCase(tp.PatientCaseID.Int64(), "testing", nil)
	test.OK(t, err)
	// At this point the case assignment should maintain its position in the doctor's list
	items, err = testData.DataAPI.GetPendingItemsInDoctorQueue(doctor.ID.Int64())
	test.OK(t, err)
	test.Equals(t, 3, len(items))
	test.Equals(t, api.DQEventTypeCaseAssignment, items[2].EventType)
	test.Equals(t, api.DQEventTypePatientVisit, items[1].EventType)
}

func TestMA_DoctorAssignToMA(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	mr, _, _ := test_integration.SignupRandomTestCC(t, testData, true)
	ma, err := testData.DataAPI.GetDoctorFromID(mr.DoctorID)
	test.OK(t, err)

	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)

	_, tp := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	// A doctor assigning a case to the MA should cause the doctor to permanently be assigned to the case and for the case to
	// move into the doctor's inbox
	req := &messages.PostMessageRequest{
		CaseID:  tp.PatientCaseID.Int64(),
		Message: "foo",
	}
	test_integration.AssignCaseMessage(t, testData, doctor.AccountID.Int64(), req)

	// At this point there should exist an item in he doctor's inbox
	items, err := testData.DataAPI.GetPendingItemsInDoctorQueue(doctor.ID.Int64())
	test.OK(t, err)
	test.Equals(t, 1, len(items))
	test.Equals(t, api.DQEventTypePatientVisit, items[0].EventType)

	// There should also exist 1 item in the MA's inbox
	items, err = testData.DataAPI.GetPendingItemsInDoctorQueue(ma.ID.Int64())
	test.OK(t, err)
	test.Equals(t, 1, len(items))
	test.Equals(t, api.DQEventTypeCaseAssignment, items[0].EventType)

	// The doctor should be able to assign the same case tot he MA multiple times
	test_integration.AssignCaseMessage(t, testData, doctor.AccountID.Int64(), req)

	// And there should still exist just 1 item in tihe doctor queue
	items, err = testData.DataAPI.GetPendingItemsInDoctorQueue(ma.ID.Int64())
	test.OK(t, err)
	test.Equals(t, 1, len(items))
	test.Equals(t, api.DQEventTypeCaseAssignment, items[0].EventType)
}

// This test is to ensure that messages stay private between the MA and the doctor
// and that the patient cannot see these private messages
func TestMA_PrivateMessages(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	mr, _, _ := test_integration.SignupRandomTestCC(t, testData, true)
	ma, err := testData.DataAPI.GetDoctorFromID(mr.DoctorID)
	test.OK(t, err)

	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)

	_, tp := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	patient, err := testData.DataAPI.GetPatientFromID(tp.PatientID)
	test.OK(t, err)

	maCli := test_integration.DoctorClient(testData, t, ma.ID.Int64())

	expectedMessage := "m1"
	req := &messages.PostMessageRequest{
		CaseID:  tp.PatientCaseID.Int64(),
		Message: expectedMessage,
	}

	test_integration.AssignCaseMessage(t, testData, doctor.AccountID.Int64(), req)

	// Doctor should be able to retreive the assigned message in the thread
	listResponse := getCaseMessages(t, testData, doctor.AccountID.Int64(), tp.PatientCaseID.Int64())
	test.OK(t, err)
	test.Equals(t, 1, len(listResponse.Items))
	test.Equals(t, expectedMessage, listResponse.Items[0].Message)

	// MA should be able to retreive the assigned message in the thread
	listResponse = getCaseMessages(t, testData, ma.AccountID.Int64(), tp.PatientCaseID.Int64())
	test.OK(t, err)
	test.Equals(t, 1, len(listResponse.Items))
	test.Equals(t, expectedMessage, listResponse.Items[0].Message)

	// Patient should NOT be able to retrieve the message as it is considered private
	listResponse = getCaseMessages(t, testData, patient.AccountID.Int64(), tp.PatientCaseID.Int64())
	test.OK(t, err)
	test.Equals(t, 0, len(listResponse.Items))

	// MA should be able to message the patient
	msg2 := "foo"
	_, err = maCli.PostCaseMessage(tp.PatientCaseID.Int64(), msg2, nil)
	test.OK(t, err)

	// All three parties should be able to see this message
	// Doctor should be able to retreive the assigned message in the thread
	listResponse = getCaseMessages(t, testData, doctor.AccountID.Int64(), tp.PatientCaseID.Int64())
	test.Equals(t, 2, len(listResponse.Items))
	test.Equals(t, expectedMessage, listResponse.Items[0].Message)
	test.Equals(t, msg2, listResponse.Items[1].Message)

	// MA should be able to retreive the assigned message in the thread
	listResponse = getCaseMessages(t, testData, ma.AccountID.Int64(), tp.PatientCaseID.Int64())
	test.Equals(t, 2, len(listResponse.Items))
	test.Equals(t, expectedMessage, listResponse.Items[0].Message)
	test.Equals(t, msg2, listResponse.Items[1].Message)

	// Patient should be able  to see hte message from the MA
	listResponse = getCaseMessages(t, testData, patient.AccountID.Int64(), tp.PatientCaseID.Int64())
	test.OK(t, err)
	test.Equals(t, 1, len(listResponse.Items))
	test.Equals(t, msg2, listResponse.Items[0].Message)
}

// This test is to ensure that the case is assigned to the MA
// when the doctor marks the case as being unsuitable
func TestMA_AssignOnMarkingCaseAsUnsuitable_NonPublicMessage(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	// Set our public config to enabled
	testData.Config.Cfg.Update(map[string]interface{}{doctor_queue.PublicUnsuitableMessageEnabledDef.Name: false})

	mr, _, _ := test_integration.SignupRandomTestCC(t, testData, true)
	ma, err := testData.DataAPI.GetDoctorFromID(mr.DoctorID)
	test.OK(t, err)

	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)

	pv, _ := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	// lets go ahead and mark this case as being unsuitable
	test_integration.MarkUnsuitableForSpruce(testData, t, pv.PatientVisitID, doctor.AccountID.Int64())

	// now the MA should have an item assigned to them in the queue
	pendingItems, err := testData.DataAPI.GetPendingItemsInDoctorQueue(ma.ID.Int64())
	test.OK(t, err)
	test.Equals(t, 1, len(pendingItems))
	test.Equals(t, api.DQEventTypeCaseAssignment, pendingItems[0].EventType)

	// the doctor should not have any pending items left in their queue
	pendingItems, err = testData.DataAPI.GetPendingItemsInDoctorQueue(doctor.ID.Int64())
	test.OK(t, err)
	test.Equals(t, 0, len(pendingItems))
}

func TestMA_AssignOnMarkingCaseAsUnsuitable_PublicMessage(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	// Set our public config to enabled
	testData.Config.Cfg.Update(map[string]interface{}{doctor_queue.PublicUnsuitableMessageEnabledDef.Name: true})

	mr, _, _ := test_integration.SignupRandomTestCC(t, testData, true)
	ma, err := testData.DataAPI.GetDoctorFromID(mr.DoctorID)
	test.OK(t, err)

	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)

	pv, _ := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	// lets go ahead and mark this case as being unsuitable
	test_integration.MarkUnsuitableForSpruce(testData, t, pv.PatientVisitID, doctor.AccountID.Int64())

	// now the MA should have an item assigned to them in the queue
	pendingItems, err := testData.DataAPI.GetPendingItemsInDoctorQueue(ma.ID.Int64())
	test.OK(t, err)
	test.Equals(t, 2, len(pendingItems))
	test.Equals(t, api.DQEventTypeCaseAssignment, pendingItems[0].EventType)

	// the doctor should not have any pending items left in their queue
	pendingItems, err = testData.DataAPI.GetPendingItemsInDoctorQueue(doctor.ID.Int64())
	test.OK(t, err)
	test.Equals(t, 0, len(pendingItems))
}

func getCaseMessages(t *testing.T, testData *test_integration.TestData, accountID, caseID int64) *messages.ListResponse {
	res, err := testData.AuthGet(testData.APIServer.URL+apipaths.CaseMessagesListURLPath+"?case_id="+strconv.FormatInt(caseID, 10), accountID)
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusOK, res.StatusCode)

	lResponse := &messages.ListResponse{}
	err = json.NewDecoder(res.Body).Decode(lResponse)
	test.OK(t, err)

	return lResponse
}
