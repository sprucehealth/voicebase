package test_ma

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice/apipaths"
	"github.com/sprucehealth/backend/doctor_queue"
	"github.com/sprucehealth/backend/messages"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

// MA should have all pending items at a clinic level in their queue.
// This includes items in the unclaimed case queue as well as items in a doctor's inbox
func TestMAQueue_UnassignedTab(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	dr, _, _ := test_integration.SignupRandomTestCC(t, testData, true)
	ma, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)

	// lets create a visit in the unassigned state
	test_integration.CreateRandomPatientVisitInState("CA", t, testData)
	test_integration.CreateRandomPatientVisitInState("CA", t, testData)

	// now lets get the doctor queue for the MA
	res, err := testData.AuthGet(testData.APIServer.URL+apipaths.DoctorQueueURLPath+"?state=global", ma.AccountID.Int64())
	test.OK(t, err)
	defer res.Body.Close()

	doctorQueueResponse := &doctor_queue.DoctorQueueItemsResponseData{}
	test.Equals(t, http.StatusOK, res.StatusCode)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 but got %d", res.StatusCode)
	} else if err := json.NewDecoder(res.Body).Decode(doctorQueueResponse); err != nil {
		t.Fatal(err)
	} else if len(doctorQueueResponse.Items) != 2 {
		t.Fatalf("Expected 2 items but got %d", len(doctorQueueResponse.Items))
	}

	// lets simulate an item into a doctor's inbox.
	dr, _, _ = test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)

	// create a random patient and permanently assign patient to doctor
	pr := test_integration.SignupRandomTestPatientWithPharmacyAndAddress(t, testData)
	pv := test_integration.CreatePatientVisitForPatient(pr.Patient.ID.Int64(), testData, t)
	testData.DataAPI.AddDoctorToCareTeamForPatient(pr.Patient.ID.Int64(), doctor.ID.Int64(), api.AcnePathwayTag)

	// submit the visit so that it gets routed directly to the doctor's inbox
	test_integration.SubmitPatientVisitForPatient(pr.Patient.ID.Int64(), pv.PatientVisitID, testData, t)

	// now there should be 3 items in the ma's queue
	res, err = testData.AuthGet(testData.APIServer.URL+apipaths.DoctorQueueURLPath+"?state=global", ma.AccountID.Int64())
	test.OK(t, err)
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 but got %d", res.StatusCode)
	} else if err := json.NewDecoder(res.Body).Decode(doctorQueueResponse); err != nil {
		t.Fatal(err)
	} else if len(doctorQueueResponse.Items) != 3 {
		t.Fatalf("Expected 3 items but got %d", len(doctorQueueResponse.Items))
	}
}

// MA should have the history of events across all doctors
// in their queue, along with their own items
func TestMAQueue_CompletedTab(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor1, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)
	dr, _, _ = test_integration.SignupRandomTestCC(t, testData, true)
	ma, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)

	// have the first doctor complete a treatment plan
	_, tp := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor1)
	test_integration.SubmitPatientVisitBackToPatient(tp.ID.Int64(), doctor1, testData, t)

	dr, _, _ = test_integration.SignupRandomTestDoctor(t, testData)
	doctor2, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)
	// have the second doctor complete a treatment plan
	_, tp2 := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor2)
	test_integration.SubmitPatientVisitBackToPatient(tp2.ID.Int64(), doctor2, testData, t)

	// now lets get the doctor queue for the MA; there should be 2 items in the completed tab
	res, err := testData.AuthGet(testData.APIServer.URL+apipaths.DoctorQueueURLPath+"?state=completed", ma.AccountID.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	doctorQueueResponse := &doctor_queue.DoctorQueueItemsResponseData{}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 but got %d", res.StatusCode)
	} else if err := json.NewDecoder(res.Body).Decode(doctorQueueResponse); err != nil {
		t.Fatal(err)
	} else if len(doctorQueueResponse.Items) != 2 {
		t.Fatalf("Expected 2 items but got %d", len(doctorQueueResponse.Items))
	}

	// lets get the MA to assign the case to the doctor  after which there should be 3 items in the ma's queue
	test_integration.AssignCaseMessage(t, testData, ma.AccountID.Int64(), &messages.PostMessageRequest{
		CaseID:  tp2.PatientCaseID.Int64(),
		Message: "foo",
	})

	res, err = testData.AuthGet(testData.APIServer.URL+apipaths.DoctorQueueURLPath+"?state=completed", ma.AccountID.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 but got %d", res.StatusCode)
	} else if err := json.NewDecoder(res.Body).Decode(doctorQueueResponse); err != nil {
		t.Fatal(err)
	} else if len(doctorQueueResponse.Items) != 3 {
		t.Fatalf("Expected 2 items but got %d", len(doctorQueueResponse.Items))
	}
}
