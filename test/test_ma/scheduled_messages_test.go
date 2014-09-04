package test_ma

import (
	"strings"
	"testing"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/app_event"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/schedmsg"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-metrics/metrics"
)

func TestScheduledMessage_VisitCharged(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	admin := test_integration.CreateRandomAdmin(t, testData)

	// Now lets go ahead and add a message template for visit charged
	if err := testData.DataApi.CreateScheduledMessageTemplate(&common.ScheduledMessageTemplate{
		Message: `Hi [Patient.FirstName],
		Send me your insurance info,
		Thanks,
		[Provider.ShortDisplayName]`,
		Event:            common.SMVisitChargedEvent,
		CreatorAccountID: admin.AccountId.Int64(),
		SchedulePeriod:   1,
		Name:             "This is a test",
	}); err != nil {
		t.Fatal(err)
	}

	// create doctor
	dr := test_integration.SignupRandomTestDoctorInState("CA", t, testData)
	doctor, err := testData.DataApi.GetDoctorFromId(dr.DoctorId)
	test.OK(t, err)

	// signup ma
	mr, _, _ := test_integration.SignupRandomTestMA(t, testData)
	_, err = testData.DataApi.GetDoctorFromId(mr.DoctorId)
	test.OK(t, err)

	// now lets go ahead and submit a visit
	_, tp := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	time.Sleep(time.Second)

	// at this point there should be a scheduled message
	var count int64
	err = testData.DB.QueryRow(`select count(*) from scheduled_message`).Scan(&count)
	test.OK(t, err)

	// lets start the worker to check for scheduled jobs
	schedmsg.StartWorker(testData.DataApi, nil, metrics.NewRegistry(), 24*60)

	time.Sleep(time.Second)

	// at this point there should be a message for the patient from the MA
	caseMessages, err := testData.DataApi.ListCaseMessages(tp.PatientCaseId.Int64(), api.PATIENT_ROLE)
	test.OK(t, err)
	test.Equals(t, 1, len(caseMessages))
	test.Equals(t, false, strings.Contains(caseMessages[0].Body, "[Patient.FirstName]"))
	test.Equals(t, true, strings.Contains(caseMessages[0].Body, "Send me your insurance info"))
}

func TestScheduledMessage_TreatmentPlanViewed(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	admin := test_integration.CreateRandomAdmin(t, testData)

	// Now lets go ahead and add a message template for visit charged
	if err := testData.DataApi.CreateScheduledMessageTemplate(&common.ScheduledMessageTemplate{
		Message: `Hi [Patient.FirstName],
		Did you pick up your prescriptions?,
		Thanks,
		[Provider.ShortDisplayName]`,
		Event:            common.SMTreatmentPlanViewedEvent,
		CreatorAccountID: admin.AccountId.Int64(),
		SchedulePeriod:   1,
		Name:             "This is a test",
	}); err != nil {
		t.Fatal(err)
	}

	// create doctor
	dr := test_integration.SignupRandomTestDoctorInState("CA", t, testData)
	doctor, err := testData.DataApi.GetDoctorFromId(dr.DoctorId)
	test.OK(t, err)

	// signup ma
	mr, _, _ := test_integration.SignupRandomTestMA(t, testData)
	_, err = testData.DataApi.GetDoctorFromId(mr.DoctorId)
	test.OK(t, err)

	// now lets go ahead and submit a visit
	_, tp := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	patient, err := testData.DataApi.GetPatientFromId(tp.PatientId)

	// lets get the doctor to submit the treatment plan back to the patietn
	test_integration.SubmitPatientVisitBackToPatient(tp.Id.Int64(), doctor, testData, t)

	// now lets get the patient to view the treatment plan
	test_integration.GenerateAppEvent(app_event.ViewedAction,
		"treatment_plan", tp.Id.Int64(), patient.AccountId.Int64(), testData, t)

	time.Sleep(time.Second)

	// at this point there should be a scheduled message
	var count int64
	err = testData.DB.QueryRow(`select count(*) from scheduled_message`).Scan(&count)
	test.OK(t, err)

	// lets start the worker to check for scheduled jobs
	schedmsg.StartWorker(testData.DataApi, nil, metrics.NewRegistry(), 24*60)

	time.Sleep(time.Second)

	// at this point there should be a message for the patient from the MA
	caseMessages, err := testData.DataApi.ListCaseMessages(tp.PatientCaseId.Int64(), api.PATIENT_ROLE)
	test.OK(t, err)
	test.Equals(t, 2, len(caseMessages))
	test.Equals(t, false, strings.Contains(caseMessages[1].Body, "[Patient.FirstName]"))
	test.Equals(t, true, strings.Contains(caseMessages[1].Body, "Did you pick up your prescriptions?"))
}
