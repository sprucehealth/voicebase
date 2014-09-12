package test_ma

import (
	"strings"
	"testing"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/app_event"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/schedmsg"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-metrics/metrics"
)

func TestScheduledMessage_InsuredPatient(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	admin := test_integration.CreateRandomAdmin(t, testData)

	// UNINSURED PATIENT SCENARIO

	// Now lets go ahead and add a message template for visit charged
	if err := testData.DataApi.CreateScheduledMessageTemplate(&common.ScheduledMessageTemplate{
		Message: `Hi {{.PatientFirstName}},
		Why are you uninsured?,
		Thanks,
		{{.ProviderShortDisplayName}}`,
		Event:            "uninsured_patient",
		CreatorAccountID: admin.AccountId.Int64(),
		SchedulePeriod:   1,
		Name:             "This is a test",
	}); err != nil {
		t.Fatal(err)
	}

	insuranceCoverageQuestionID := test_integration.DetermineQuestionIDForTag("q_insurance_coverage", testData, t)
	noInsurancePotentialAnswerID := test_integration.DeterminePotentialAnswerIDForTag("a_no_insurance", testData, t)
	genericOnlyAnswerID := test_integration.DeterminePotentialAnswerIDForTag("a_insurance_generic_only", testData, t)

	// signup ma
	mr, _, _ := test_integration.SignupRandomTestMA(t, testData)
	_, err := testData.DataApi.GetDoctorFromId(mr.DoctorId)
	test.OK(t, err)

	// create a random patient and simulate uninsured patient coming through visit
	pv := test_integration.CreateAndSubmitPatientVisitWithSpecifiedAnswers(map[int64]*apiservice.AnswerToQuestionItem{
		insuranceCoverageQuestionID: &apiservice.AnswerToQuestionItem{
			QuestionId: insuranceCoverageQuestionID,
			AnswerIntakes: []*apiservice.AnswerItem{
				&apiservice.AnswerItem{
					PotentialAnswerId: noInsurancePotentialAnswerID,
				},
			},
		},
	}, testData, t)

	time.Sleep(time.Second)

	// at this point there should be a scheduled message
	var count int64
	err = testData.DB.QueryRow(`select count(*) from scheduled_message`).Scan(&count)
	test.OK(t, err)
	test.Equals(t, int64(1), count)

	// lets start the worker to check for scheduled jobs
	schedmsg.StartWorker(testData.DataApi, nil, metrics.NewRegistry(), 24*60)

	time.Sleep(time.Second)

	// at this point the message should be processed
	err = testData.DB.QueryRow(`select count(*) from scheduled_message where status = ?`, common.SMSent.String()).Scan(&count)
	test.OK(t, err)
	test.Equals(t, int64(1), count)

	// at this point there should be a message for the patient from the MA
	patientCase, err := testData.DataApi.GetPatientCaseFromPatientVisitId(pv.PatientVisitId)
	test.OK(t, err)
	caseMessages, err := testData.DataApi.ListCaseMessages(patientCase.Id.Int64(), api.PATIENT_ROLE)
	test.OK(t, err)
	test.Equals(t, 1, len(caseMessages))
	test.Equals(t, false, strings.Contains(caseMessages[0].Body, "{{.PatientFirstName}}"))
	test.Equals(t, true, strings.Contains(caseMessages[0].Body, "Why are you uninsured?"))

	// INSURED PATIENT SCENARIO

	// now lets enter a template for an insured patient
	if err := testData.DataApi.CreateScheduledMessageTemplate(&common.ScheduledMessageTemplate{
		Message: `Hi {{.PatientFirstName}},
		You're insured! Yay!,
		Thanks,
		{{.ProviderShortDisplayName}}`,
		Event:            "insured_patient",
		CreatorAccountID: admin.AccountId.Int64(),
		SchedulePeriod:   1,
		Name:             "This is a test",
	}); err != nil {
		t.Fatal(err)
	}

	pv = test_integration.CreateAndSubmitPatientVisitWithSpecifiedAnswers(map[int64]*apiservice.AnswerToQuestionItem{
		insuranceCoverageQuestionID: &apiservice.AnswerToQuestionItem{
			QuestionId: insuranceCoverageQuestionID,
			AnswerIntakes: []*apiservice.AnswerItem{
				&apiservice.AnswerItem{
					PotentialAnswerId: genericOnlyAnswerID,
				},
			},
		},
	}, testData, t)

	time.Sleep(time.Second)

	// at this point there should be a scheduled message
	err = testData.DB.QueryRow(`select count(*) from scheduled_message where status = ?`, common.SMScheduled.String()).Scan(&count)
	test.OK(t, err)
	test.Equals(t, int64(1), count)

	// lets start the worker to check for scheduled jobs
	schedmsg.StartWorker(testData.DataApi, nil, metrics.NewRegistry(), 24*60)

	time.Sleep(time.Second)

	// at this point both messages should be processed
	err = testData.DB.QueryRow(`select count(*) from scheduled_message where status = ?`, common.SMSent.String()).Scan(&count)
	test.OK(t, err)
	test.Equals(t, int64(2), count)

	// at this point there should be a message for the patient from the MA
	patientCase, err = testData.DataApi.GetPatientCaseFromPatientVisitId(pv.PatientVisitId)
	test.OK(t, err)
	caseMessages, err = testData.DataApi.ListCaseMessages(patientCase.Id.Int64(), api.PATIENT_ROLE)
	test.OK(t, err)
	test.Equals(t, 1, len(caseMessages))
	test.Equals(t, false, strings.Contains(caseMessages[0].Body, "{{.PatientFirstName}}"))
	test.Equals(t, true, strings.Contains(caseMessages[0].Body, "You're insured! Yay!"))
}

func TestScheduledMessage_TreatmentPlanViewed(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	admin := test_integration.CreateRandomAdmin(t, testData)

	// Now lets go ahead and add a message template for visit charged
	if err := testData.DataApi.CreateScheduledMessageTemplate(&common.ScheduledMessageTemplate{
		Message: `Hi {{.PatientFirstName}},
		Did you pick up your prescriptions?,
		Thanks,
		{{.ProviderShortDisplayName}}`,
		Event:            "treatment_plan_viewed",
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
	test.Equals(t, false, strings.Contains(caseMessages[1].Body, "{{.PatientFirstName}}"))
	test.Equals(t, true, strings.Contains(caseMessages[1].Body, "Did you pick up your prescriptions?"))
}
