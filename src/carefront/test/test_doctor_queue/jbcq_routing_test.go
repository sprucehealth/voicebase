package test_doctor_queue

import (
	"carefront/test/test_integration"
	"testing"
)

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

	// lets sign up a doc in CA and a doc in WA
	d1 := test_integration.SignupRandomTestDoctorInState("CA", t, testData)
	d2 := test_integration.SignupRandomTestDoctorInState("WA", t, testData)

	// lets submit a patient visit in WA
	test_integration.CreateRandomPatientVisitInState("WA", t, testData)

	// doctor in CA should not see the case in the global queue
	unclaimedItems, err := testData.DataApi.GetElligibleItemsInUnclaimedQueue(d1.DoctorId)
	if err != nil {
		t.Fatal(err)
	} else if len(unclaimedItems) != 0 {
		t.Fatalf("Expected 0 unclaimed item for doctor instead got %d", len(unclaimedItems))
	}

	// doctor in WA should see the case in the global queue
	unclaimedItems, err = testData.DataApi.GetElligibleItemsInUnclaimedQueue(d2.DoctorId)
	if err != nil {
		t.Fatal(err)
	} else if len(unclaimedItems) != 1 {
		t.Fatalf("Expected 1 unclaimed item for doctor instead got %d", len(unclaimedItems))
	}

	// now lets submit a case in "OR"
	test_integration.CreateRandomPatientVisitInState("OR", t, testData)

	// neither doctor should be able to see the case
	unclaimedItems, err = testData.DataApi.GetElligibleItemsInUnclaimedQueue(d1.DoctorId)
	if err != nil {
		t.Fatal(err)
	} else if len(unclaimedItems) != 0 {
		t.Fatalf("Expected 0 unclaimed items for doctor instead got %d", len(unclaimedItems))
	}
	unclaimedItems, err = testData.DataApi.GetElligibleItemsInUnclaimedQueue(d2.DoctorId)
	if err != nil {
		t.Fatal(err)
	} else if len(unclaimedItems) != 1 {
		t.Fatalf("Expected 1 unclaimed item for doctor instead got %d", len(unclaimedItems))
	}

}
