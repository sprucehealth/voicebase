package test_doctor_queue

import (
	"carefront/api"
	"carefront/common"
	"carefront/test/test_integration"
	"testing"
)

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
