package test_integration

import (
	"testing"

	"github.com/sprucehealth/backend/appevent"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/messages"
	"github.com/sprucehealth/backend/test"
)

func TestPersonCreation(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	// Make sure a person row is inserted when creating a patient

	pr := SignupRandomTestPatientWithPharmacyAndAddress(t, testData)
	patientID := pr.Patient.ID.Int64()
	if pid, err := testData.DataAPI.GetPersonIDByRole(api.RolePatient, patientID); err != nil {
		t.Fatalf("Failed to get person for role %s/%d: %s", api.RolePatient, patientID, err.Error())
	} else if pid <= 0 {
		t.Fatalf("Invalid patient ID %d", pid)
	}

	// Make sure a person row is inserted when creating a doctor

	dr, _, _ := SignupRandomTestDoctor(t, testData)
	doctorID := dr.DoctorID
	if pid, err := testData.DataAPI.GetPersonIDByRole(api.RoleDoctor, doctorID); err != nil {
		t.Fatalf("Failed to get person for role %s/%d: %s", api.RoleDoctor, doctorID, err.Error())
	} else if pid <= 0 {
		t.Fatalf("Invalid patient ID %d", pid)
	}
}

func TestCaseMessages(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	cc, _, _ := SignupRandomTestCC(t, testData, true)
	doctorID := GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	test.OK(t, err)
	doctorPersonID, err := testData.DataAPI.GetPersonIDByRole(api.RoleDoctor, doctorID)
	test.OK(t, err)

	visit, treatmentPlan := CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	patient, err := testData.DataAPI.GetPatientFromPatientVisitID(visit.PatientVisitID)
	test.OK(t, err)
	patientPersonID, err := testData.DataAPI.GetPersonIDByRole(api.RolePatient, patient.ID.Int64())
	test.OK(t, err)

	doctorCli := DoctorClient(testData, t, doctorID)
	patientCli := PatientClient(testData, t, patient.ID)
	ccCli := DoctorClient(testData, t, cc.DoctorID)

	test.OK(t, doctorCli.UpdateTreatmentPlanNote(treatmentPlan.ID.Int64(), "foo"))
	test.OK(t, doctorCli.SubmitTreatmentPlan(treatmentPlan.ID.Int64()))

	caseID, err := testData.DataAPI.GetPatientCaseIDFromPatientVisitID(visit.PatientVisitID)
	test.OK(t, err)

	photoID, _ := UploadPhoto(t, testData, doctor.AccountID.Int64())

	audioID, _ := uploadMedia(t, testData, doctor.AccountID.Int64())
	attachments := []*messages.Attachment{
		&messages.Attachment{
			Type: common.AttachmentTypePhoto,
			ID:   photoID,
		},
		&messages.Attachment{
			Type: common.AttachmentTypeAudio,
			ID:   audioID,
		},
	}

	_, err = doctorCli.PostCaseMessage(caseID, "foo", attachments)
	test.OK(t, err)

	msgs, err := testData.DataAPI.ListCaseMessages(caseID, api.LCMOIncludePrivate)
	test.OK(t, err)
	test.Equals(t, 2, len(msgs)) // one we just posted and one for the treatment plan submission

	m := msgs[len(msgs)-1]
	test.Equals(t, 0, len(m.ReadReceipts))
	test.Equals(t, 2, len(m.Attachments))
	a := m.Attachments[0]
	if a.ItemType != common.AttachmentTypePhoto || a.ItemID != photoID {
		t.Fatalf("Wrong attachment type or ID")

	}
	photo, err := testData.DataAPI.GetMedia(photoID)
	test.OK(t, err)
	ok, err := testData.DataAPI.MediaHasClaim(photo.ID, common.ClaimerTypeConversationMessage, m.ID)
	test.OK(t, err)
	test.Equals(t, true, ok)

	b := m.Attachments[1]
	if b.ItemType != common.AttachmentTypeAudio || b.ItemID != audioID {
		t.Fatalf("Wrong attachment type or ID")
	}
	media, err := testData.DataAPI.GetMedia(audioID)
	test.OK(t, err)
	ok, err = testData.DataAPI.MediaHasClaim(media.ID, common.ClaimerTypeConversationMessage, m.ID)
	test.OK(t, err)
	test.Equals(t, true, ok)

	participants, err := testData.DataAPI.CaseMessageParticipants(caseID, false)
	test.OK(t, err)
	test.Equals(t, 1, len(participants))
	test.Equals(t, false, participants[doctorPersonID] == nil)

	// Reply from patient
	_, err = patientCli.PostCaseMessage(caseID, "bar", nil)
	test.OK(t, err)

	msgs, err = testData.DataAPI.ListCaseMessages(caseID, 0)
	test.OK(t, err)
	test.Equals(t, 3, len(msgs))

	participants, err = testData.DataAPI.CaseMessageParticipants(caseID, false)
	test.OK(t, err)
	test.Equals(t, 2, len(participants))
	test.Equals(t, false, participants[doctorPersonID] == nil)
	test.Equals(t, false, participants[patientPersonID] == nil)

	// Test read receipts
	{
		// Patient reading messages should record a read receipt
		msgs, _, err := patientCli.ListCaseMessages(caseID)
		test.OK(t, err)
		test.Equals(t, 3, len(msgs))

		// CC should see read receipts
		msgs, pars, err := ccCli.ListCaseMessages(caseID)
		test.OK(t, err)
		test.Equals(t, 3, len(msgs))
		for _, m := range msgs {
			test.Equals(t, 1, len(m.ReadReceipts))
			for _, rr := range m.ReadReceipts {
				var found bool
				for _, p := range pars {
					if p.ID == rr.ParticipantID {
						found = true
						break
					}
				}
				test.Equals(t, true, found)
			}
		}

		// Doctor SHOULD NOT see read receipts
		msgs, _, err = doctorCli.ListCaseMessages(caseID)
		test.OK(t, err)
		test.Equals(t, 3, len(msgs))
		for _, m := range msgs {
			test.Equals(t, 0, len(m.ReadReceipts))
		}

		// Patient MUST NOT see read receipts
		msgs, _, err = patientCli.ListCaseMessages(caseID)
		test.OK(t, err)
		test.Equals(t, 3, len(msgs))
		for _, m := range msgs {
			test.Equals(t, 0, len(m.ReadReceipts))
		}
	}

	// Test unread count
	{
		test.OK(t, patientCli.AppEvent(appevent.ViewedAction, "all_case_messages", caseID))
		test.OK(t, doctorCli.AppEvent(appevent.ViewedAction, "all_case_messages", caseID))

		// Initial unread counts should be 0
		count, err := testData.DataAPI.UnreadMessageCount(caseID, doctor.PersonID)
		test.OK(t, err)
		test.Equals(t, 0, count)
		count, err = testData.DataAPI.UnreadMessageCount(caseID, patient.PersonID)
		test.OK(t, err)
		test.Equals(t, 0, count)

		_, err = patientCli.PostCaseMessage(caseID, "bar", nil)
		test.OK(t, err)

		// Doctor who hasn't seen the message yet should see an unread count of 1
		count, err = testData.DataAPI.UnreadMessageCount(caseID, doctor.PersonID)
		test.OK(t, err)
		test.Equals(t, 1, count)
		// Patient who posted the message should see an unread count of 0
		count, err = testData.DataAPI.UnreadMessageCount(caseID, patient.PersonID)
		test.OK(t, err)
		test.Equals(t, 0, count)

		test.OK(t, doctorCli.AppEvent(appevent.ViewedAction, "all_case_messages", caseID))

		// Now that the doctor read the message the unread count should be 0
		count, err = testData.DataAPI.UnreadMessageCount(caseID, doctor.PersonID)
		test.OK(t, err)
		test.Equals(t, 0, count)
	}
}
