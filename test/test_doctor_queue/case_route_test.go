package test_doctor_queue

import (
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

// This test is to ensure that if a patient has a doctor assigned to their care team,
// any new case created for the condition supported by the doctor gets directly routed
// to the doctor and permanently assigned to them
func TestCaseRoute_DoctorInCareTeam(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	doctorID := test_integration.GetDoctorIdOfCurrentDoctor(testData, t)

	pr := test_integration.SignupRandomTestPatientInState("CA", t, testData)

	// assign the doctor to the patient file
	if err := testData.DataApi.AddDoctorToCareTeamForPatient(pr.Patient.PatientId.Int64(), apiservice.HEALTH_CONDITION_ACNE_ID, doctorID); err != nil {
		t.Fatal(err)
	}

	pv := test_integration.CreatePatientVisitForPatient(pr.Patient.PatientId.Int64(), testData, t)
	answerIntakeRequestBody := test_integration.PrepareAnswersForQuestionsInPatientVisit(pv, t)
	test_integration.SubmitAnswersIntakeForPatient(pr.Patient.PatientId.Int64(), pr.Patient.AccountId.Int64(),
		answerIntakeRequestBody, testData, t)
	test_integration.SubmitPatientVisitForPatient(pr.Patient.PatientId.Int64(), pv.PatientVisitId, testData, t)

	// the patient case should now be in the assigned state
	patientCase, err := testData.DataApi.GetPatientCaseFromPatientVisitId(pv.PatientVisitId)
	if err != nil {
		t.Fatal(err)
	} else if patientCase.Status != common.PCStatusClaimed {
		t.Fatalf("Expected patient case to be %s but it was %s", common.PCStatusClaimed, patientCase.Status)
	}

	// there should exist an item in the local queue of the doctor
	pendingItems, err := testData.DataApi.GetPendingItemsInDoctorQueue(doctorID)
	if err != nil {
		t.Fatal(err)
	} else if len(pendingItems) != 1 {
		t.Fatalf("Expected 1 item in doctor's local queue but instead got %d", len(pendingItems))
	}

	// there should be a permanent access of the doctor to the patient case
	doctorAssignments, err := testData.DataApi.GetDoctorsAssignedToPatientCase(patientCase.Id.Int64())
	if err != nil {
		t.Fatal(err)
	} else if len(doctorAssignments) != 1 {
		t.Fatalf("Expected 1 doctor assigned to case instead got %d", len(doctorAssignments))
	} else if doctorAssignments[0].Status != api.STATUS_ACTIVE {
		t.Fatalf("Expected permanent assignment of doctor to patient case instead got %s", doctorAssignments[0].Status)
	} else if doctorAssignments[0].ProviderRole != api.DOCTOR_ROLE {
		t.Fatalf("Expected a doctor to be assigned to the patient case instead it was %s", doctorAssignments[0].ProviderRole)
	} else if doctorAssignments[0].ProviderID != doctorID {
		t.Fatalf("Expected doctor %d to be assigned to patient case instead got %d", doctorID, doctorAssignments[0].ProviderID)
	}

}

// This test is to ensure that we are notifying doctors that are configured to receive SMS notifications
// of unclaimed cases submitted in the states they are activated in
func TestCaseRoute_NotifyingDoctors(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	// lets create three doctors in three different state
	dr1 := test_integration.SignupRandomTestDoctorInState("CA", t, testData)
	dr2 := test_integration.SignupRandomTestDoctorInState("NY", t, testData)
	dr3 := test_integration.SignupRandomTestDoctorInState("WA", t, testData)
	test_integration.SignupRandomTestDoctorInState("PA", t, testData)

	careProvidingStateIDCA, err := testData.DataApi.GetCareProvidingStateId("CA", apiservice.HEALTH_CONDITION_ACNE_ID)
	test.OK(t, err)

	careProvidingStateIDWA, err := testData.DataApi.GetCareProvidingStateId("WA", apiservice.HEALTH_CONDITION_ACNE_ID)
	test.OK(t, err)

	careProvidingStateIDNY, err := testData.DataApi.GetCareProvidingStateId("NY", apiservice.HEALTH_CONDITION_ACNE_ID)
	test.OK(t, err)

	// lets register doctor1 to get notified for visits in CA and NY
	_, err = testData.DB.Exec(`UPDATE care_provider_state_elligibility SET notify = 1 WHERE provider_id = ? and care_providing_state_id = ?`, dr1.DoctorId, careProvidingStateIDCA)
	test.OK(t, err)

	err = testData.DataApi.MakeDoctorElligibleinCareProvidingState(careProvidingStateIDNY, dr1.DoctorId)
	test.OK(t, err)
	_, err = testData.DB.Exec(`UPDATE care_provider_state_elligibility SET notify = 1 WHERE provider_id = ? and care_providing_state_id = ?`, dr1.DoctorId, careProvidingStateIDNY)
	test.OK(t, err)

	// lets update doctor1's phone number to make it something that is distinguishable
	doctor1, err := testData.DataApi.GetDoctorFromId(dr1.DoctorId)
	test.OK(t, err)
	err = testData.AuthApi.ReplacePhoneNumbersForAccount(doctor1.AccountId.Int64(), []*common.PhoneNumber{
		&common.PhoneNumber{
			Phone:  common.Phone("734-846-5520"),
			Type:   api.PHONE_CELL,
			Status: api.STATUS_ACTIVE,
		},
	})
	test.OK(t, err)

	// now lets create and submit a visit in the state of CA
	test_integration.CreateRandomPatientVisitInState("CA", t, testData)

	// at this point doctor1 should have received an SMS about the visit
	test.Equals(t, 1, testData.SMSAPI.Len())
	test.Equals(t, "734-846-5520", testData.SMSAPI.Sent[0].To)

	// now lets create and submit a visit in the state of NY
	test_integration.CreateRandomPatientVisitInState("NY", t, testData)

	// at this point doctor1 should have received an SMS about the visit in NY
	test.Equals(t, 2, testData.SMSAPI.Len())
	test.Equals(t, "734-846-5520", testData.SMSAPI.Sent[1].To)

	// lets change doctor2's phone number to something unique
	doctor2, err := testData.DataApi.GetDoctorFromId(dr2.DoctorId)
	test.OK(t, err)
	err = testData.AuthApi.ReplacePhoneNumbersForAccount(doctor2.AccountId.Int64(), []*common.PhoneNumber{
		&common.PhoneNumber{
			Phone:  common.Phone("734-846-5521"),
			Type:   api.PHONE_CELL,
			Status: api.STATUS_ACTIVE,
		},
	})
	test.OK(t, err)

	// lets register doctor2 to get notified for visits in NY
	_, err = testData.DB.Exec(`UPDATE care_provider_state_elligibility SET notify = 1 WHERE provider_id = ? and care_providing_state_id = ?`, dr2.DoctorId, careProvidingStateIDNY)
	test.OK(t, err)

	// lets register doctor3 to get notified for visits in WA
	_, err = testData.DB.Exec(`UPDATE care_provider_state_elligibility SET notify = 1 WHERE provider_id = ? and care_providing_state_id = ?`, dr3.DoctorId, careProvidingStateIDWA)
	test.OK(t, err)

	// now lets submit another visit in NY
	// both doctors should be notified
	test_integration.CreateRandomPatientVisitInState("NY", t, testData)

	test.Equals(t, 4, testData.SMSAPI.Len())
	test.Equals(t, "734-846-5521", testData.SMSAPI.Sent[2].To)
	test.Equals(t, "734-846-5520", testData.SMSAPI.Sent[3].To)

	// lets change doctor3's phone number to something unique
	doctor3, err := testData.DataApi.GetDoctorFromId(dr3.DoctorId)
	test.OK(t, err)
	err = testData.AuthApi.ReplacePhoneNumbersForAccount(doctor3.AccountId.Int64(), []*common.PhoneNumber{
		&common.PhoneNumber{
			Phone:  common.Phone("734-846-5525"),
			Type:   api.PHONE_CELL,
			Status: api.STATUS_ACTIVE,
		},
	})
	test.OK(t, err)

	// now lets submit a visit in WA and only doctor3 should be notified
	test_integration.CreateRandomPatientVisitInState("WA", t, testData)

	test.Equals(t, 5, testData.SMSAPI.Len())
	test.Equals(t, "734-846-5525", testData.SMSAPI.Sent[4].To)

	// now submit a visit in PA and no one should be notified
	test_integration.CreateRandomPatientVisitInState("PA", t, testData)
	test.Equals(t, 5, testData.SMSAPI.Len())
}
