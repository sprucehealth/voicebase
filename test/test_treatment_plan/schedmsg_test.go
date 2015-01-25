package test_treatment_plan

import (
	"sort"
	"testing"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/messages"
	"github.com/sprucehealth/backend/responses"
	"github.com/sprucehealth/backend/schedmsg"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

// For sorting attachments to make tests deterministic
type attachments []*messages.Attachment

func (as attachments) Len() int           { return len(as) }
func (as attachments) Swap(a, b int)      { as[a], as[b] = as[b], as[a] }
func (as attachments) Less(a, b int) bool { return as[a].Type < as[b].Type }

func TestScheduledMessage(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	doctorID := test_integration.GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	test.OK(t, err)
	doctorCli := test_integration.DoctorClient(testData, t, doctorID)

	dr, _, _ := test_integration.SignupRandomTestMA(t, testData)
	ma, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)
	maCli := test_integration.DoctorClient(testData, t, ma.DoctorID.Int64())

	_, tp := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	// List with no scheduled messages
	msgs, err := doctorCli.ListTreatmentPlanScheduledMessages(tp.ID.Int64())
	test.OK(t, err)
	test.Equals(t, len(msgs), 0)

	// Test creating invalid scheduled message
	msg := &responses.ScheduledMessage{}
	_, err = doctorCli.CreateTreatmentPlanScheduledMessage(tp.ID.Int64(), msg)
	test.Equals(t, false, err == nil)

	// Test create with follow-up and photo
	photoID, _ := test_integration.UploadPhoto(t, testData, doctor.AccountID.Int64())

	msg = &responses.ScheduledMessage{
		ScheduledDays: 7*4 + 1,
		Message:       "Hello, welcome",
		Attachments: []*messages.Attachment{
			{
				Type: common.AttachmentTypeFollowupVisit,
			},
			{
				Type: common.AttachmentTypePhoto,
				ID:   photoID,
			},
		},
	}

	// MA should not be able to create messages
	_, err = maCli.CreateTreatmentPlanScheduledMessage(tp.ID.Int64(), msg)
	test.Equals(t, false, err == nil)
	// Doctor should be able to create a message
	msgID, err := doctorCli.CreateTreatmentPlanScheduledMessage(tp.ID.Int64(), msg)
	test.OK(t, err)

	// MA and doctor should be able to list messages
	msgs, err = maCli.ListTreatmentPlanScheduledMessages(tp.ID.Int64())
	test.OK(t, err)
	test.Equals(t, 1, len(msgs))
	msgs, err = doctorCli.ListTreatmentPlanScheduledMessages(tp.ID.Int64())
	test.OK(t, err)
	test.Equals(t, 1, len(msgs))
	test.Equals(t, msgID, msgs[0].ID)
	test.Equals(t, "Message & Follow-Up Visit in 4 weeks", *msgs[0].Title)
	test.Equals(t, msg.ScheduledDays, msgs[0].ScheduledDays)
	test.Equals(t, false, msgs[0].ScheduledFor.IsZero())
	test.Equals(t, msg.Message, msgs[0].Message)
	test.Equals(t, 2, len(msgs[0].Attachments))
	sort.Sort(attachments(msgs[0].Attachments))
	test.Equals(t, messages.AttachmentTypePrefix+common.AttachmentTypeFollowupVisit, msgs[0].Attachments[0].Type)
	test.Equals(t, "Follow-Up Visit", msgs[0].Attachments[0].Title)
	test.Equals(t, messages.AttachmentTypePrefix+common.AttachmentTypePhoto, msgs[0].Attachments[1].Type)
	test.Equals(t, photoID, msgs[0].Attachments[1].ID)

	// Trying to create the exact same message should be a noop and should return the same ID
	msgID2, err := doctorCli.CreateTreatmentPlanScheduledMessage(tp.ID.Int64(), msg)
	test.OK(t, err)
	test.Equals(t, msgID, msgID2)

	// MA should not be able to delete messages
	test.Equals(t, false, nil == maCli.DeleteTreatmentPlanScheduledMessages(tp.ID.Int64(), msgID))
	// Doctor should be able to delete messages
	test.OK(t, doctorCli.DeleteTreatmentPlanScheduledMessages(tp.ID.Int64(), msgID))
	msgs, err = doctorCli.ListTreatmentPlanScheduledMessages(tp.ID.Int64())
	test.OK(t, err)
	test.Equals(t, len(msgs), 0)

	// Media should now be unclaimed so can create another message with same media
	msgID, err = doctorCli.CreateTreatmentPlanScheduledMessage(tp.ID.Int64(), msg)
	test.OK(t, err)

	// Update (replace) the message
	msg.Message = "New message"
	msg.ID = msgID
	msgID2, err = doctorCli.UpdateTreatmentPlanScheduledMessage(tp.ID.Int64(), msg)
	test.OK(t, err)
	test.Equals(t, msgID, msgID2)

	msgs, err = doctorCli.ListTreatmentPlanScheduledMessages(tp.ID.Int64())
	test.OK(t, err)
	test.Equals(t, 1, len(msgs))
	test.Equals(t, msgID, msgs[0].ID)
	test.Equals(t, "Message & Follow-Up Visit in 4 weeks", *msgs[0].Title)
	test.Equals(t, msg.ScheduledDays, msgs[0].ScheduledDays)
	test.Equals(t, false, msgs[0].ScheduledFor.IsZero())
	test.Equals(t, msg.Message, msgs[0].Message)
	test.Equals(t, 2, len(msgs[0].Attachments))
	sort.Sort(attachments(msgs[0].Attachments))
	test.Equals(t, messages.AttachmentTypePrefix+common.AttachmentTypeFollowupVisit, msgs[0].Attachments[0].Type)
	test.Equals(t, "Follow-Up Visit", msgs[0].Attachments[0].Title)
	test.Equals(t, messages.AttachmentTypePrefix+common.AttachmentTypePhoto, msgs[0].Attachments[1].Type)
	test.Equals(t, photoID, msgs[0].Attachments[1].ID)
}

func TestScheduledMessageSend(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	doctorID := test_integration.GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	test.OK(t, err)
	doctorCli := test_integration.DoctorClient(testData, t, doctorID)

	_, tp := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	photoID, _ := test_integration.UploadPhoto(t, testData, doctor.AccountID.Int64())

	msg := &responses.ScheduledMessage{
		ScheduledDays: 7*4 + 1,
		Message:       "Hello, welcome",
		Attachments: []*messages.Attachment{
			{
				Type: common.AttachmentTypeFollowupVisit,
			},
			{
				Type: common.AttachmentTypePhoto,
				ID:   photoID,
			},
		},
	}

	msgID, err := doctorCli.CreateTreatmentPlanScheduledMessage(tp.ID.Int64(), msg)
	test.OK(t, err)

	test.OK(t, doctorCli.UpdateTreatmentPlanNote(tp.ID.Int64(), "foo"))
	test.OK(t, doctorCli.SubmitTreatmentPlan(tp.ID.Int64()))

	// Just to make sure there should be 1 message now for the treatment plan
	msgs, err := testData.DataAPI.ListCaseMessages(tp.PatientCaseID.Int64(), api.DOCTOR_ROLE)
	test.OK(t, err)
	test.Equals(t, 1, len(msgs))

	tpsm, err := testData.DataAPI.TreatmentPlanScheduledMessage(msgID)
	test.OK(t, err)
	test.Equals(t, false, tpsm.ScheduledMessageID == nil)
	sm, err := testData.DataAPI.ScheduledMessage(*tpsm.ScheduledMessageID, schedmsg.ScheduledMsgTypes)
	test.OK(t, err)
	test.Equals(t, common.SMScheduled, sm.Status)
	test.Equals(t, true, sm.Scheduled.Sub(time.Now().UTC()) > (7*4*time.Hour*24))

	// Make sure the job gets properly picked up and processed

	_, err = testData.DB.Exec(`UPDATE scheduled_message SET scheduled = ? WHERE id = ?`, time.Now().UTC().Add(-time.Hour*24), sm.ID)
	test.OK(t, err)

	worker := schedmsg.NewWorker(testData.DataAPI, testData.AuthAPI, testData.Config.Dispatcher, nil, metrics.NewRegistry(), 1)
	consumed, err := worker.ConsumeMessage()
	test.OK(t, err)
	test.Equals(t, true, consumed)

	sm, err = testData.DataAPI.ScheduledMessage(*tpsm.ScheduledMessageID, schedmsg.ScheduledMsgTypes)
	test.OK(t, err)
	test.Equals(t, common.SMSent, sm.Status)

	msgs, err = testData.DataAPI.ListCaseMessages(tp.PatientCaseID.Int64(), api.DOCTOR_ROLE)
	test.OK(t, err)
	test.Equals(t, 2, len(msgs))
	test.Equals(t, "Hello, welcome", msgs[1].Body)
	test.Equals(t, 2, len(msgs[1].Attachments))
}

func TestFavoriteTPScheduledMessage(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	doctorID := test_integration.GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	test.OK(t, err)
	doctorCli := test_integration.DoctorClient(testData, t, doctorID)

	// Create a patient treatment plan, and save a draft message
	visit, tp0 := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	test_integration.AddTreatmentsToTreatmentPlan(tp0.ID.Int64(), doctor, t, testData)
	test_integration.AddRegimenPlanForTreatmentPlan(tp0.ID.Int64(), doctor, t, testData)

	photoID, _ := test_integration.UploadPhoto(t, testData, doctor.AccountID.Int64())

	msg := &responses.ScheduledMessage{
		ScheduledDays: 7*4 + 1,
		Message:       "Hello, welcome",
		Attachments: []*messages.Attachment{
			{
				Type: common.AttachmentTypeFollowupVisit,
			},
			{
				Type: common.AttachmentTypePhoto,
				ID:   photoID,
			},
		},
	}
	_, err = doctorCli.CreateTreatmentPlanScheduledMessage(tp0.ID.Int64(), msg)
	test.OK(t, err)

	// Refetch the treatment plan to fill in with recent updates
	tp, err := doctorCli.TreatmentPlan(tp0.ID.Int64(), false, doctor_treatment_plan.AllSections)
	test.OK(t, err)
	test.Equals(t, 1, len(tp.ScheduledMessages))

	ftp := &responses.FavoriteTreatmentPlan{
		Name:          "Test FTP",
		TreatmentList: tp.TreatmentList,
		RegimenPlan:   tp.RegimenPlan,
	}

	// Test creating ftp when scheduled messages don't match
	_, err = doctorCli.CreateFavoriteTreatmentPlanFromTreatmentPlan(ftp, tp.ID.Int64())
	test.Equals(t, false, err == nil)

	ftp.ScheduledMessages = tp.ScheduledMessages
	_, err = doctorCli.CreateFavoriteTreatmentPlanFromTreatmentPlan(ftp, tp.ID.Int64())
	test.OK(t, err)

	ftps, err := doctorCli.ListFavoriteTreatmentPlans()
	test.OK(t, err)
	test.Equals(t, 1, len(ftps))
	test.Equals(t, len(tp.ScheduledMessages), len(ftps[0].ScheduledMessages))

	// Make sure treatment plan created from an ftp that has scheduled messages also
	// gets the messages.
	tp, err = doctorCli.PickTreatmentPlanForVisit(visit.PatientVisitID, ftps[0])
	test.OK(t, err)
	test.Equals(t, len(ftps[0].ScheduledMessages), len(tp.ScheduledMessages))
	test.Equals(t, true, ftps[0].ScheduledMessages[0].ID != tp.ScheduledMessages[0].ID)

	// update the note so that we can submit the plan
	err = doctorCli.UpdateTreatmentPlanNote(tp.ID.Int64(), "Some note")
	test.OK(t, err)

	// ensure that even after submitting the TP the scheduled messages are still there
	err = doctorCli.SubmitTreatmentPlan(tp.ID.Int64())
	test.OK(t, err)

	tp, err = doctorCli.TreatmentPlan(tp.ID.Int64(), false, doctor_treatment_plan.AllSections)
	test.OK(t, err)
	test.Equals(t, len(ftps[0].ScheduledMessages), len(tp.ScheduledMessages))
	test.Equals(t, true, ftps[0].ScheduledMessages[0].ID != tp.ScheduledMessages[0].ID)

	// lets create yet another TP so that we have a draft that we can then try to delete
	tp, err = doctorCli.PickTreatmentPlanForVisit(visit.PatientVisitID, ftps[0])
	test.OK(t, err)

	// ensure that deleting an FTP or TP with a scheduled msg works
	err = doctorCli.DeleteFavoriteTreatmentPlan(ftps[0].ID.Int64())
	test.OK(t, err)

	err = doctorCli.DeleteTreatmentPlan(tp.ID.Int64())
	test.OK(t, err)
}
