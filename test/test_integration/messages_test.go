package test_integration

import (
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/messages"
	"testing"
)

func TestPersonCreation(t *testing.T) {
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	// Make sure a person row is inserted when creating a patient

	pr := SignupRandomTestPatient(t, testData)
	patientId := pr.Patient.PatientId.Int64()
	if pid, err := testData.DataApi.GetPersonIdByRole(api.PATIENT_ROLE, patientId); err != nil {
		t.Fatalf("Failed to get person for role %s/%d: %s", api.PATIENT_ROLE, patientId, err.Error())
	} else if pid <= 0 {
		t.Fatalf("Invalid patient ID %d", pid)
	}

	// Make sure a person row is inserted when creating a doctor

	dr, _, _ := SignupRandomTestDoctor(t, testData)
	doctorId := dr.DoctorId
	if pid, err := testData.DataApi.GetPersonIdByRole(api.DOCTOR_ROLE, doctorId); err != nil {
		t.Fatalf("Failed to get person for role %s/%d: %s", api.DOCTOR_ROLE, doctorId, err.Error())
	} else if pid <= 0 {
		t.Fatalf("Invalid patient ID %d", pid)
	}
}

func TestCaseMessages(t *testing.T) {
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	doctorID := GetDoctorIdOfCurrentDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorID)
	if err != nil {
		t.Fatal(err)
	}
	doctorPersonID, err := testData.DataApi.GetPersonIdByRole(api.DOCTOR_ROLE, doctorID)
	if err != nil {
		t.Fatal(err)
	}

	visit, treatmentPlan := CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	patient, err := testData.DataApi.GetPatientFromPatientVisitId(visit.PatientVisitId)
	if err != nil {
		t.Fatal(err)
	}
	patientPersonID, err := testData.DataApi.GetPersonIdByRole(api.PATIENT_ROLE, patient.PatientId.Int64())
	if err != nil {
		t.Fatal(err)
	}
	SubmitPatientVisitBackToPatient(treatmentPlan.Id.Int64(), doctor, testData, t)

	caseID, err := testData.DataApi.GetPatientCaseIdFromPatientVisitId(visit.PatientVisitId)
	if err != nil {
		t.Fatal(err)
	}

	photoID := uploadPhoto(t, testData, doctor.AccountId.Int64())
	attachments := []*messages.Attachment{
		&messages.Attachment{
			Type: common.AttachmentTypePhoto,
			ID:   photoID,
		},
	}

	PostCaseMessage(t, testData, doctor.AccountId.Int64(), &messages.PostMessageRequest{
		CaseID:      caseID,
		Message:     "foo",
		Attachments: attachments,
	})

	msgs, err := testData.DataApi.ListCaseMessages(caseID)
	if err != nil {
		t.Fatal(err)
	} else if len(msgs) != 2 { // one we just posted and one for the treatment plan submission
		t.Fatalf("Expected 2 message. Got %d", len(msgs))
	}

	m := msgs[len(msgs)-1]
	if len(m.Attachments) != 1 {
		t.Fatalf("Expected 1 attachment. Got %d", len(m.Attachments))
	}
	a := m.Attachments[0]
	if a.ItemType != common.AttachmentTypePhoto || a.ItemID != photoID {
		t.Fatalf("Wrong attachment type or ID")

	}
	photo, err := testData.DataApi.GetPhoto(photoID)
	if err != nil {
		t.Fatal(err)
	}
	if photo.ClaimerType != common.ClaimerTypeConversationMessage {
		t.Fatalf("Expected claimer type of '%s'. Got '%s'", common.ClaimerTypeConversationMessage, photo.ClaimerType)
	}
	if photo.ClaimerId != m.ID {
		t.Fatalf("Expected claimer id to be %d. Got %d", m.ID, photo.ClaimerId)
	}

	if participants, err := testData.DataApi.CaseMessageParticipants(caseID, false); err != nil {
		t.Fatal(err)
	} else if len(participants) != 1 {
		t.Fatalf("Expected 1 participant. Got %d", len(participants))
	} else if participants[doctorPersonID] == nil {
		t.Fatalf("Participant does not match")
	} else if participants[doctorPersonID].Unread {
		t.Fatalf("Expected conversation to be read")
	}

	// Reply from patient
	PostCaseMessage(t, testData, patient.AccountId.Int64(), &messages.PostMessageRequest{
		CaseID:  caseID,
		Message: "bar",
	})

	if msgs, err = testData.DataApi.ListCaseMessages(caseID); err != nil {
		t.Fatal(err)
	} else if len(msgs) != 3 {
		t.Fatalf("Expected 3 messages. Got %d", len(msgs))
	}

	if participants, err := testData.DataApi.CaseMessageParticipants(caseID, false); err != nil {
		t.Fatal(err)
	} else if len(participants) != 2 {
		t.Fatalf("Expected 2 participants. Got %d", len(participants))
	} else if participants[doctorPersonID] == nil {
		t.Fatalf("Participant does not exist")
	} else if !participants[doctorPersonID].Unread {
		t.Fatalf("Expected doctor's conversation to be unread")
	} else if participants[patientPersonID] == nil {
		t.Fatalf("Participant does not exist")
	} else if participants[patientPersonID].Unread {
		t.Fatalf("Expected patient's conversation to be read")
	}

	if err := testData.DataApi.MarkCaseMessagesAsRead(caseID, doctorPersonID); err != nil {
		t.Fatal(err)
	}

	if participants, err := testData.DataApi.CaseMessageParticipants(caseID, false); err != nil {
		t.Fatal(err)
	} else if participants[doctorPersonID].Unread {
		t.Fatalf("Expected doctor's conversation to be read")
	}
}
