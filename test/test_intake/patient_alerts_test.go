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
	patientVisitResponse := test_integration.CreatePatientVisitForPatient(patientSignedupResponse.Patient.PatientID.Int64(), testData, t)

	patient, err := testData.DataAPI.GetPatientFromID(patientSignedupResponse.Patient.PatientID.Int64())
	if err != nil {
		t.Fatal("Unable to get patient from id: " + err.Error())
	}

	intakeData := test_integration.PrepareAnswersForQuestionsInPatientVisit(patientVisitResponse.PatientVisitID, patientVisitResponse.ClientLayout, t)

	questionInfo, err := testData.DataAPI.GetQuestionInfo("q_allergic_medication_entry", api.EN_LANGUAGE_ID)
	test.OK(t, err)

	// lets update the answer intake to capture a medication the patient is allergic to
	// and ensure that gets populated on visit submission
	answerText := "Sulfa Drugs (Testing)"
	aItem := &apiservice.QuestionAnswerItem{
		QuestionID: questionInfo.QuestionID,
		AnswerIntakes: []*apiservice.AnswerItem{
			&apiservice.AnswerItem{
				AnswerText: answerText,
			},
		},
	}

	for i, item := range intakeData.Questions {
		if item.QuestionID == aItem.QuestionID {
			intakeData.Questions[i] = aItem
		}
	}

	test_integration.SubmitAnswersIntakeForPatient(patient.PatientID.Int64(), patient.AccountID.Int64(), intakeData, testData, t)
	test_integration.SubmitPatientVisitForPatient(patientSignedupResponse.Patient.PatientID.Int64(), patientVisitResponse.PatientVisitID, testData, t)

	// wait for a second so that the goroutine runs to capture the patient alerts
	time.Sleep(time.Second)

	// now there should be atlest 1 alert for the patient
	alerts, err := testData.DataAPI.GetAlertsForPatient(patient.PatientID.Int64())
	if err != nil {
		t.Fatal(err)
	} else if len(alerts) == 0 {
		t.Fatalf("Expected atleast %d alerts instead got %d", 1, len(alerts))
	}

	// lets go through the alerts and ensure that our response was inserted
	alertFound := false
	for _, alert := range alerts {
		if alert.Source == common.AlertSourcePatientVisitIntake && alert.SourceID == aItem.QuestionID {
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
	patientVisitResponse := test_integration.CreatePatientVisitForPatient(patientSignedupResponse.Patient.PatientID.Int64(), testData, t)

	patient, err := testData.DataAPI.GetPatientFromID(patientSignedupResponse.Patient.PatientID.Int64())
	if err != nil {
		t.Fatal("Unable to get patient from id: " + err.Error())
	}

	intakeData := test_integration.PrepareAnswersForQuestionsInPatientVisitWithoutAlerts(patientVisitResponse, t)
	test_integration.SubmitAnswersIntakeForPatient(patient.PatientID.Int64(), patient.AccountID.Int64(), intakeData, testData, t)
	test_integration.SubmitPatientVisitForPatient(patientSignedupResponse.Patient.PatientID.Int64(), patientVisitResponse.PatientVisitID, testData, t)

	// wait for a second so that the goroutine runs to capture the patient alerts
	time.Sleep(time.Second)

	// at this point, no alerts should exist for the patient since we chose not to answer questions that would result in patient alerts
	alerts, err := testData.DataAPI.GetAlertsForPatient(patient.PatientID.Int64())
	if err != nil {
		t.Fatal(err)
	} else if len(alerts) != 0 {
		t.Fatalf("Expected atleast %d alerts instead got %d", 0, len(alerts))
	}
}
