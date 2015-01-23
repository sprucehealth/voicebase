package test_doctor_queue

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice/apipaths"
	"github.com/sprucehealth/backend/doctor_queue"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

// This test is to ensure that the auth url is included in the
// doctor queue information for the doctor app to know what to do
// to claim access to the patient case
func TestJBCQRouting_AuthUrlInDoctorQueue(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	d1 := test_integration.SignupRandomTestDoctorInState("CA", t, testData)
	doctor, err := testData.DataAPI.GetDoctorFromID(d1.DoctorID)
	test.OK(t, err)

	test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	responseData := &doctor_queue.DoctorQueueItemsResponseData{}
	res, err := testData.AuthGet(testData.APIServer.URL+apipaths.DoctorQueueURLPath+"?state=global", doctor.AccountID.Int64())
	if err != nil {
		t.Fatal(err)
	} else if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected %d instead got %d", http.StatusOK, res.StatusCode)
	} else if err := json.NewDecoder(res.Body).Decode(responseData); err != nil {
		t.Fatal(err)
	} else if len(responseData.Items) != 1 {
		t.Fatalf("Expected 1 items instead got %d", len(responseData.Items))
	} else if responseData.Items[0].AuthURL == nil {
		t.Fatal("Expected auth url instead got nothing")
	}
}

func TestJBCQRouting_ItemDescription(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	d1 := test_integration.SignupRandomTestDoctorInState("CA", t, testData)
	doctor, err := testData.DataAPI.GetDoctorFromID(d1.DoctorID)
	test.OK(t, err)

	test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	unassignedItems, err := testData.DataAPI.GetElligibleItemsInUnclaimedQueue(doctor.DoctorID.Int64())
	test.OK(t, err)
	test.Equals(t, 1, len(unassignedItems))
	test.Equals(t, "New visit", unassignedItems[0].ShortDescription)
	test.Equals(t, true, strings.Contains(unassignedItems[0].Description, "New visit"))
}

// This test ensures that all doctors in the same state see
// an elligible item
func TestJBCQRouting_MultipleDocsInSameState(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	// lets go ahead and register 4 doctors in the state of CA
	doctorId1 := test_integration.GetDoctorIDOfCurrentDoctor(testData, t)
	d2 := test_integration.SignupRandomTestDoctorInState("CA", t, testData)
	d3 := test_integration.SignupRandomTestDoctorInState("CA", t, testData)
	d4 := test_integration.SignupRandomTestDoctorInState("CA", t, testData)

	// lets simulate an incoming visit that gets routed to the JBCQ
	test_integration.CreateRandomPatientVisitInState("CA", t, testData)

	// all 4 doctors should see the unclaimed case
	doctorIDs := []int64{doctorId1, d2.DoctorID, d3.DoctorID, d4.DoctorID}
	for _, doctorID := range doctorIDs {
		unclaimedItems, err := testData.DataAPI.GetElligibleItemsInUnclaimedQueue(doctorID)
		if err != nil {
			t.Fatal(err)
		} else if len(unclaimedItems) != 1 {
			t.Fatalf("Expected 1 unclaimed item for doctor instead got %d", len(unclaimedItems))
		}
	}
}

func TestJBCQRouting_MultipleDocsDifferentStates(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	// lets add the care providing states that we are testing the scenarios in
	_, err := testData.DataAPI.AddCareProvidingState("WA", "Washington", api.AcnePathwayTag)
	test.OK(t, err)

	orProvidingStateId, err := testData.DataAPI.AddCareProvidingState("OR", "Oregon", api.AcnePathwayTag)
	test.OK(t, err)

	// lets sign up a doc in CA and a doc in WA
	d1 := test_integration.SignupRandomTestDoctorInState("CA", t, testData)
	d2 := test_integration.SignupRandomTestDoctorInState("WA", t, testData)

	// lets submit a patient visit in WA
	test_integration.CreateRandomPatientVisitInState("WA", t, testData)

	// doctor in CA should not see the case in the global queue
	// doctor in WA should see the case in the global queue
	if unclaimedItems := getUnclaimedItemsForDoctor(d1.DoctorID, t, testData); len(unclaimedItems) != 0 {
		t.Fatalf("Expected 0 unclaimed items for doctor instead got %d", len(unclaimedItems))
	} else if unclaimedItems := getUnclaimedItemsForDoctor(d2.DoctorID, t, testData); len(unclaimedItems) != 1 {
		t.Fatalf("Expected 1 unclaimed item for doctor instead got %d", len(unclaimedItems))
	}

	// now lets submit a case in "OR"
	test_integration.CreateRandomPatientVisitInState("OR", t, testData)

	// neither doctor should be able to see the case
	if unclaimedItems := getUnclaimedItemsForDoctor(d1.DoctorID, t, testData); len(unclaimedItems) != 0 {
		t.Fatalf("Expected 0 unclaimed items for doctor instead got %d", len(unclaimedItems))
	} else if unclaimedItems := getUnclaimedItemsForDoctor(d2.DoctorID, t, testData); len(unclaimedItems) != 1 {
		t.Fatalf("Expected 1 unclaimed item for doctor instead got %d", len(unclaimedItems))
	}

	// now make doctor1 and doctor2 elligible in OR
	if err := testData.DataAPI.MakeDoctorElligibleinCareProvidingState(orProvidingStateId, d1.DoctorID); err != nil {
		t.Fatal(err)
	}
	if err := testData.DataAPI.MakeDoctorElligibleinCareProvidingState(orProvidingStateId, d2.DoctorID); err != nil {
		t.Fatal(err)
	}

	// both doctors should now see case registeres in OR
	if unclaimedItems := getUnclaimedItemsForDoctor(d1.DoctorID, t, testData); len(unclaimedItems) != 1 {
		t.Fatalf("Expected 1 unclaimed item for doctor instead got %d", len(unclaimedItems))
	} else if unclaimedItems := getUnclaimedItemsForDoctor(d2.DoctorID, t, testData); len(unclaimedItems) != 2 {
		t.Fatalf("Expected 2 unclaimed items for doctor instead got %d", len(unclaimedItems))
	}
}
