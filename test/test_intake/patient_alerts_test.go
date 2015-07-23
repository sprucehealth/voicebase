package test_intake

import (
	"strings"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestPatientAlerts(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	test_integration.SignupRandomTestCC(t, testData, true)

	patientSignedupResponse := test_integration.SignupRandomTestPatientWithPharmacyAndAddress(t, testData)
	patientVisitResponse := test_integration.CreatePatientVisitForPatient(patientSignedupResponse.Patient.ID.Int64(), testData, t)

	patient, err := testData.DataAPI.GetPatientFromID(patientSignedupResponse.Patient.ID.Int64())
	if err != nil {
		t.Fatal("Unable to get patient from id: " + err.Error())
	}

	intakeData := test_integration.PrepareAnswersForQuestionsInPatientVisit(patientVisitResponse.PatientVisitID, patientVisitResponse.ClientLayout.InfoIntakeLayout, t)

	questionInfo, err := testData.DataAPI.GetQuestionInfo("q_allergic_medication_entry", api.LanguageIDEnglish, 1)
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

	test_integration.SubmitAnswersIntakeForPatient(patient.ID.Int64(), patient.AccountID.Int64(), intakeData, testData, t)
	test_integration.SubmitPatientVisitForPatient(patientSignedupResponse.Patient.ID.Int64(), patientVisitResponse.PatientVisitID, testData, t)

	// now there should be atlest 1 alert for the patient
	alerts, err := testData.DataAPI.AlertsForVisit(patientVisitResponse.PatientVisitID)
	if err != nil {
		t.Fatal(err)
	} else if len(alerts) == 0 {
		t.Fatalf("Expected atleast %d alerts instead got %d", 1, len(alerts))
	}

	// lets go through the alerts and ensure that our response was inserted
	alertFound := false
	for _, alert := range alerts {
		if alert.QuestionID == aItem.QuestionID {
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
	patientVisitResponse := test_integration.CreatePatientVisitForPatient(patientSignedupResponse.Patient.ID.Int64(), testData, t)

	patient, err := testData.DataAPI.GetPatientFromID(patientSignedupResponse.Patient.ID.Int64())
	if err != nil {
		t.Fatal("Unable to get patient from id: " + err.Error())
	}

	intakeData := test_integration.PrepareAnswersForQuestionsInPatientVisitWithoutAlerts(patientVisitResponse, t)
	test_integration.SubmitAnswersIntakeForPatient(patient.ID.Int64(), patient.AccountID.Int64(), intakeData, testData, t)
	test_integration.SubmitPatientVisitForPatient(patientSignedupResponse.Patient.ID.Int64(), patientVisitResponse.PatientVisitID, testData, t)

	// at this point, no alerts should exist for the patient since we chose not to answer questions that would result in patient alerts
	alerts, err := testData.DataAPI.AlertsForVisit(patientVisitResponse.PatientVisitID)
	if err != nil {
		t.Fatal(err)
	} else if len(alerts) != 0 {
		t.Fatalf("Expected atleast %d alerts instead got %d", 0, len(alerts))
	}
}
