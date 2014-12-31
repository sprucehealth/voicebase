package test_ma

import (
	"strings"
	"testing"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/app_event"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/schedmsg"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestScheduledMessage_InsuredPatient(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	// UNINSURED PATIENT SCENARIO

	// Now lets go ahead and add a message template for visit charged
	if err := testData.DataAPI.CreateScheduledMessageTemplate(&common.ScheduledMessageTemplate{
		Message: `Hi {{.PatientFirstName}},
		Why are you uninsured?,
		Thanks,
		{{.ProviderShortDisplayName}}`,
		Event:          "uninsured_patient",
		SchedulePeriod: 1,
		Name:           "This is a test",
	}); err != nil {
		t.Fatal(err)
	}

	insuranceCoverageQuestionID := test_integration.DetermineQuestionIDForTag("q_insurance_coverage", testData, t)
	noInsurancePotentialAnswerID := test_integration.DeterminePotentialAnswerIDForTag("a_no_insurance", testData, t)
	genericOnlyAnswerID := test_integration.DeterminePotentialAnswerIDForTag("a_insurance_generic_only", testData, t)

	// signup ma
	mr, _, _ := test_integration.SignupRandomTestMA(t, testData)
	_, err := testData.DataAPI.GetDoctorFromID(mr.DoctorID)
	test.OK(t, err)

	// create a random patient and simulate uninsured patient coming through visit
	pv := test_integration.CreateAndSubmitPatientVisitWithSpecifiedAnswers(map[int64]*apiservice.QuestionAnswerItem{
		insuranceCoverageQuestionID: &apiservice.QuestionAnswerItem{
			QuestionID: insuranceCoverageQuestionID,
			AnswerIntakes: []*apiservice.AnswerItem{
				&apiservice.AnswerItem{
					PotentialAnswerID: noInsurancePotentialAnswerID,
				},
			},
		},
	}, testData, t)

	// at this point there should be a scheduled message
	var count int64
	err = testData.DB.QueryRow(`select count(*) from scheduled_message`).Scan(&count)
	test.OK(t, err)
	test.Equals(t, int64(1), count)

	// lets trigger the scheduled message now
	_, err = testData.DB.Exec(`
		UPDATE scheduled_message
		SET scheduled = ?`, time.Now().Add(-5*time.Minute))
	test.OK(t, err)

	// lets start the worker to check for scheduled jobs
	worker := schedmsg.NewWorker(testData.DataAPI, testData.AuthAPI, testData.Config.Dispatcher, nil, metrics.NewRegistry(), 1)
	consumed, err := worker.ConsumeMessage()
	test.OK(t, err)
	test.Equals(t, true, consumed)

	// at this point the message should be processed
	err = testData.DB.QueryRow(`select count(*) from scheduled_message where status = ?`, common.SMSent.String()).Scan(&count)
	test.OK(t, err)
	test.Equals(t, int64(1), count)

	// at this point there should be a message for the patient from the MA
	patientCase, err := testData.DataAPI.GetPatientCaseFromPatientVisitID(pv.PatientVisitID)
	test.OK(t, err)
	caseMessages, err := testData.DataAPI.ListCaseMessages(patientCase.ID.Int64(), api.PATIENT_ROLE)
	test.OK(t, err)
	test.Equals(t, 1, len(caseMessages))
	test.Equals(t, false, strings.Contains(caseMessages[0].Body, "{{.PatientFirstName}}"))
	test.Equals(t, true, strings.Contains(caseMessages[0].Body, "Why are you uninsured?"))

	// INSURED PATIENT SCENARIO

	// now lets enter a template for an insured patient
	if err := testData.DataAPI.CreateScheduledMessageTemplate(&common.ScheduledMessageTemplate{
		Message: `Hi {{.PatientFirstName}},
		You're insured! Yay!,
		Thanks,
		{{.ProviderShortDisplayName}}`,
		Event:          "insured_patient",
		SchedulePeriod: 1,
		Name:           "This is a test",
	}); err != nil {
		t.Fatal(err)
	}

	pv = test_integration.CreateAndSubmitPatientVisitWithSpecifiedAnswers(map[int64]*apiservice.QuestionAnswerItem{
		insuranceCoverageQuestionID: &apiservice.QuestionAnswerItem{
			QuestionID: insuranceCoverageQuestionID,
			AnswerIntakes: []*apiservice.AnswerItem{
				&apiservice.AnswerItem{
					PotentialAnswerID: genericOnlyAnswerID,
				},
			},
		},
	}, testData, t)

	// at this point there should be a scheduled message
	err = testData.DB.QueryRow(`select count(*) from scheduled_message where status = ?`, common.SMScheduled.String()).Scan(&count)
	test.OK(t, err)
	test.Equals(t, int64(1), count)

	// lets trigger the scheduled message now
	_, err = testData.DB.Exec(`
		UPDATE scheduled_message
		SET scheduled = ?`, time.Now().Add(-5*time.Minute))
	test.OK(t, err)

	// lets start the worker to check for scheduled jobs
	_, err = worker.ConsumeMessage()
	test.OK(t, err)

	// at this point both messages should be processed
	err = testData.DB.QueryRow(`select count(*) from scheduled_message where status = ?`, common.SMSent.String()).Scan(&count)
	test.OK(t, err)
	test.Equals(t, int64(2), count)

	// at this point there should be a message for the patient from the MA
	patientCase, err = testData.DataAPI.GetPatientCaseFromPatientVisitID(pv.PatientVisitID)
	test.OK(t, err)
	caseMessages, err = testData.DataAPI.ListCaseMessages(patientCase.ID.Int64(), api.PATIENT_ROLE)
	test.OK(t, err)
	test.Equals(t, 1, len(caseMessages))
	test.Equals(t, false, strings.Contains(caseMessages[0].Body, "{{.PatientFirstName}}"))
	test.Equals(t, true, strings.Contains(caseMessages[0].Body, "You're insured! Yay!"))
}

func TestScheduledMessage_TreatmentPlanViewed(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	// Now lets go ahead and add a message template for visit charged
	if err := testData.DataAPI.CreateScheduledMessageTemplate(&common.ScheduledMessageTemplate{
		Message: `Hi {{.PatientFirstName}},
		Did you pick up your prescriptions?,
		Thanks,
		{{.ProviderShortDisplayName}}`,
		Event:          "treatment_plan_viewed",
		SchedulePeriod: 1,
		Name:           "This is a test",
	}); err != nil {
		t.Fatal(err)
	}

	// create doctor
	dr := test_integration.SignupRandomTestDoctorInState("CA", t, testData)
	doctor, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)

	// signup ma
	mr, _, _ := test_integration.SignupRandomTestMA(t, testData)
	_, err = testData.DataAPI.GetDoctorFromID(mr.DoctorID)
	test.OK(t, err)

	// now lets go ahead and submit a visit
	_, tp := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	patient, err := testData.DataAPI.GetPatientFromID(tp.PatientID)

	// lets get the doctor to submit the treatment plan back to the patietn
	test_integration.SubmitPatientVisitBackToPatient(tp.ID.Int64(), doctor, testData, t)

	// now lets get the patient to view the treatment plan
	test_integration.GenerateAppEvent(app_event.ViewedAction,
		"treatment_plan", tp.ID.Int64(), patient.AccountID.Int64(), testData, t)

	// at this point there should be a scheduled message
	var count int64
	err = testData.DB.QueryRow(`select count(*) from scheduled_message`).Scan(&count)
	test.OK(t, err)

	// lets trigger the scheduled message now
	_, err = testData.DB.Exec(`
		UPDATE scheduled_message
		SET scheduled = ?`, time.Now().Add(-5*time.Minute))
	test.OK(t, err)

	// lets start the worker to check for scheduled jobs
	worker := schedmsg.NewWorker(testData.DataAPI, testData.AuthAPI, testData.Config.Dispatcher, nil, metrics.NewRegistry(), 24*60)
	_, err = worker.ConsumeMessage()
	test.OK(t, err)

	// at this point there should be a message for the patient from the MA
	caseMessages, err := testData.DataAPI.ListCaseMessages(tp.PatientCaseID.Int64(), api.PATIENT_ROLE)
	test.OK(t, err)
	test.Equals(t, 2, len(caseMessages))
	test.Equals(t, false, strings.Contains(caseMessages[1].Body, "{{.PatientFirstName}}"))
	test.Equals(t, true, strings.Contains(caseMessages[1].Body, "Did you pick up your prescriptions?"))
}
