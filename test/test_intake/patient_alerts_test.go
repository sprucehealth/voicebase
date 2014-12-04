package test_intake

import (
	"strings"
	"testing"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestPatientAlerts(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	test_integration.SignupRandomTestMA(t, testData)

	patientSignedupResponse := test_integration.SignupRandomTestPatientWithPharmacyAndAddress(t, testData)
	patientVisitResponse := test_integration.CreatePatientVisitForPatient(patientSignedupResponse.Patient.PatientId.Int64(), testData, t)

	patient, err := testData.DataApi.GetPatientFromId(patientSignedupResponse.Patient.PatientId.Int64())
	if err != nil {
		t.Fatal("Unable to get patient from id: " + err.Error())
	}

	answerIntakeRequestBody := test_integration.PrepareAnswersForQuestionsInPatientVisit(patientVisitResponse.PatientVisitId, patientVisitResponse.ClientLayout, t)

	questionInfo, err := testData.DataApi.GetQuestionInfo("q_allergic_medication_entry", api.EN_LANGUAGE_ID)
	test.OK(t, err)

	// lets update the answer intake to capture a medication the patient is allergic to
	// and ensure that gets populated on visit submission
	answerText := "Sulfa Drugs (Testing)"
	aItem := &apiservice.AnswerToQuestionItem{
		QuestionId: questionInfo.QuestionId,
		AnswerIntakes: []*apiservice.AnswerItem{
			&apiservice.AnswerItem{
				AnswerText: answerText,
			},
		},
	}

	for i, item := range answerIntakeRequestBody.Questions {
		if item.QuestionId == aItem.QuestionId {
			answerIntakeRequestBody.Questions[i] = aItem
		}
	}

	test_integration.SubmitAnswersIntakeForPatient(patient.PatientId.Int64(), patient.AccountId.Int64(), answerIntakeRequestBody, testData, t)
	test_integration.SubmitPatientVisitForPatient(patientSignedupResponse.Patient.PatientId.Int64(), patientVisitResponse.PatientVisitId, testData, t)

	// wait for a second so that the goroutine runs to capture the patient alerts
	time.Sleep(time.Second)

	// now there should be atlest 1 alert for the patient
	alerts, err := testData.DataApi.GetAlertsForPatient(patient.PatientId.Int64())
	if err != nil {
		t.Fatal(err)
	} else if len(alerts) == 0 {
		t.Fatalf("Expected atleast %d alerts instead got %d", 1, len(alerts))
	}

	// lets go through the alerts and ensure that our response was inserted
	alertFound := false
	for _, alert := range alerts {
		if alert.Source == common.AlertSourcePatientVisitIntake && alert.SourceId == aItem.QuestionId {
			alertFound = true
			if !strings.Contains(alert.Message, answerText) {
				t.Fatal("Alert message different than what was expected")
			}
		}
	}

	if !alertFound {
		t.Fatal("Expected alert not found")
	}
}

func TestPatientAlerts_NoAlerts(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	patientSignedupResponse := test_integration.SignupRandomTestPatientWithPharmacyAndAddress(t, testData)
	patientVisitResponse := test_integration.CreatePatientVisitForPatient(patientSignedupResponse.Patient.PatientId.Int64(), testData, t)

	patient, err := testData.DataApi.GetPatientFromId(patientSignedupResponse.Patient.PatientId.Int64())
	if err != nil {
		t.Fatal("Unable to get patient from id: " + err.Error())
	}

	answerIntakeRequestBody := test_integration.PrepareAnswersForQuestionsInPatientVisitWithoutAlerts(patientVisitResponse, t)
	test_integration.SubmitAnswersIntakeForPatient(patient.PatientId.Int64(), patient.AccountId.Int64(), answerIntakeRequestBody, testData, t)
	test_integration.SubmitPatientVisitForPatient(patientSignedupResponse.Patient.PatientId.Int64(), patientVisitResponse.PatientVisitId, testData, t)

	// wait for a second so that the goroutine runs to capture the patient alerts
	time.Sleep(time.Second)

	// at this point, no alerts should exist for the patient since we chose not to answer questions that would result in patient alerts
	alerts, err := testData.DataApi.GetAlertsForPatient(patient.PatientId.Int64())
	if err != nil {
		t.Fatal(err)
	} else if len(alerts) != 0 {
		t.Fatalf("Expected atleast %d alerts instead got %d", 0, len(alerts))
	}
}
