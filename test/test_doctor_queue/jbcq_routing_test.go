package test_doctor_queue

import (
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/doctor_queue"
	"github.com/sprucehealth/backend/test/test_integration"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// This test is to ensure that the auth url is included in the
// doctor queue information for the doctor app to know what to do
// to claim access to the patient case
func TestJBCQRouting_AuthUrlInDoctorQueue(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)

	d1 := test_integration.SignupRandomTestDoctorInState("CA", t, testData)
	doctor, err := testData.DataApi.GetDoctorFromId(d1.DoctorId)
	if err != nil {
		t.Fatal(err)
	}

	doctorServer := httptest.NewServer(doctor_queue.NewQueueHandler(testData.DataApi))
	defer doctorServer.Close()

	test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	responseData := &doctor_queue.DoctorQueueItemsResponseData{}
	res, err := testData.AuthGet(doctorServer.URL+"?state=global", doctor.AccountId.Int64())
	if err != nil {
		t.Fatal(err)
	} else if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected %d instead got %d", http.StatusOK, res.StatusCode)
	} else if err := json.NewDecoder(res.Body).Decode(responseData); err != nil {
		t.Fatal(err)
	} else if len(responseData.Items) != 1 {
		t.Fatalf("Expected 1 items instead got %d", len(responseData.Items))
	} else if responseData.Items[0].AuthUrl == nil {
		t.Fatal("Expected auth url instead got nothing")
	}
}

// This test ensures that all doctors in the same state see
// an elligible item
func TestJBCQRouting_MultipleDocsInSameState(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)

	// lets go ahead and register 4 doctors in the state of CA
	doctorId1 := test_integration.GetDoctorIdOfCurrentDoctor(testData, t)
	d2 := test_integration.SignupRandomTestDoctorInState("CA", t, testData)
	d3 := test_integration.SignupRandomTestDoctorInState("CA", t, testData)
	d4 := test_integration.SignupRandomTestDoctorInState("CA", t, testData)

	// lets simulate an incoming visit that gets routed to the JBCQ
	test_integration.CreateRandomPatientVisitInState("CA", t, testData)

	// all 4 doctors should see the unclaimed case
	doctorIds := []int64{doctorId1, d2.DoctorId, d3.DoctorId, d4.DoctorId}
	for _, doctorId := range doctorIds {
		unclaimedItems, err := testData.DataApi.GetElligibleItemsInUnclaimedQueue(doctorId)
		if err != nil {
			t.Fatal(err)
		} else if len(unclaimedItems) != 1 {
			t.Fatalf("Expected 1 unclaimed item for doctor instead got %d", len(unclaimedItems))
		}
	}
}

func TestJBCQRouting_MultipleDocsDifferentStates(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)

	// lets add the care providing states that we are testing the scenarios in
	_, err := testData.DataApi.AddCareProvidingState("WA", "Washington", apiservice.HEALTH_CONDITION_ACNE_ID)
	if err != nil {
		t.Fatal(err)
	}

	orProvidingStateId, err := testData.DataApi.AddCareProvidingState("OR", "Oregon", apiservice.HEALTH_CONDITION_ACNE_ID)
	if err != nil {
		t.Fatal(err)
	}

	// lets sign up a doc in CA and a doc in WA
	d1 := test_integration.SignupRandomTestDoctorInState("CA", t, testData)
	d2 := test_integration.SignupRandomTestDoctorInState("WA", t, testData)

	// lets submit a patient visit in WA
	test_integration.CreateRandomPatientVisitInState("WA", t, testData)

	// doctor in CA should not see the case in the global queue
	// doctor in WA should see the case in the global queue
	if unclaimedItems := getUnclaimedItemsForDoctor(d1.DoctorId, t, testData); len(unclaimedItems) != 0 {
		t.Fatalf("Expected 0 unclaimed items for doctor instead got %d", len(unclaimedItems))
	} else if unclaimedItems := getUnclaimedItemsForDoctor(d2.DoctorId, t, testData); len(unclaimedItems) != 1 {
		t.Fatalf("Expected 1 unclaimed item for doctor instead got %d", len(unclaimedItems))
	}

	// now lets submit a case in "OR"
	test_integration.CreateRandomPatientVisitInState("OR", t, testData)

	// neither doctor should be able to see the case
	if unclaimedItems := getUnclaimedItemsForDoctor(d1.DoctorId, t, testData); len(unclaimedItems) != 0 {
		t.Fatalf("Expected 0 unclaimed items for doctor instead got %d", len(unclaimedItems))
	} else if unclaimedItems := getUnclaimedItemsForDoctor(d2.DoctorId, t, testData); len(unclaimedItems) != 1 {
		t.Fatalf("Expected 1 unclaimed item for doctor instead got %d", len(unclaimedItems))
	}

	// now make doctor1 and doctor2 elligible in OR
	if err := testData.DataApi.MakeDoctorElligibleinCareProvidingState(orProvidingStateId, d1.DoctorId); err != nil {
		t.Fatal(err)
	}
	if err := testData.DataApi.MakeDoctorElligibleinCareProvidingState(orProvidingStateId, d2.DoctorId); err != nil {
		t.Fatal(err)
	}

	// both doctors should now see case registeres in OR
	if unclaimedItems := getUnclaimedItemsForDoctor(d1.DoctorId, t, testData); len(unclaimedItems) != 1 {
		t.Fatalf("Expected 1 unclaimed item for doctor instead got %d", len(unclaimedItems))
	} else if unclaimedItems := getUnclaimedItemsForDoctor(d2.DoctorId, t, testData); len(unclaimedItems) != 2 {
		t.Fatalf("Expected 2 unclaimed items for doctor instead got %d", len(unclaimedItems))
	}
}
