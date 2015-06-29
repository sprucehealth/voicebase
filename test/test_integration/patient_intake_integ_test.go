package test_integration

import (
	"strconv"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/patient"
	"github.com/sprucehealth/backend/test"
)

type AnswerIntakeHandler struct {
	DataAPI   api.DataAPI
	accountID int64
}

func getQuestionWithTagAndExpectedType(questionTag, questionType string, t *testing.T, testData *TestData) int64 {
	questionInfo, err := testData.DataAPI.GetQuestionInfo(questionTag, 1, 1)
	if err != nil {
		t.Fatalf("Unable to query for question q_reason_visit from database: %s", err.Error())
	}

	// need to ensure that the question we are trying to get the information for is a single select
	// question type
	if questionInfo.QuestionType != questionType {
		t.Fatalf("Expected q_reason_visit to be '%s' instead of '%s'", questionType, questionInfo.QuestionType)
	}

	return questionInfo.QuestionID
}

func getAnswerWithTagAndExpectedType(answerTag, answerType string, questionID int64, testData *TestData, t *testing.T) int64 {
	potentialAnswers, err := testData.DataAPI.GetAnswerInfo(questionID, 1)
	if err != nil {
		t.Fatal("Unable to get answers for question with id " + strconv.FormatInt(questionID, 10))
	}

	expectedAnswerTag := answerTag
	var potentialAnswerID int64
	var potentialAnswerType string
	for _, potentialAnswer := range potentialAnswers {
		if potentialAnswer.AnswerTag == expectedAnswerTag {
			potentialAnswerID = potentialAnswer.AnswerID
			potentialAnswerType = potentialAnswer.AnswerType
		}
	}

	if potentialAnswerID == 0 {
		t.Fatal("Unable to find the answer for the question with intended answer tag " + expectedAnswerTag)
	}

	if potentialAnswerType != answerType {
		t.Fatalf("Potential answer found does not have matching type. Expected %s, Found %s ", answerType, potentialAnswerType)
	}

	return potentialAnswerID
}

func TestSingleSelectIntake(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	// signup a random test patient for which to answer questions
	pr := SignupRandomTestPatientWithPharmacyAndAddress(t, testData)
	patientVisitResponse := CreatePatientVisitForPatient(pr.Patient.ID.Int64(), testData, t)

	// now lets go ahead and try and answer the question about the reason for visit given that it is
	// single select
	questionID := getQuestionWithTagAndExpectedType("q_onset_acne", "q_type_single_select", t, testData)
	potentialAnswerID := getAnswerWithTagAndExpectedType("a_onset_six_months", "a_type_multiple_choice", questionID, testData, t)

	// lets go ahead and populate a response for the question
	rb := apiservice.IntakeData{
		PatientVisitID: patientVisitResponse.PatientVisitID,
		Questions: []*apiservice.QuestionAnswerItem{
			&apiservice.QuestionAnswerItem{
				QuestionID: questionID,
				AnswerIntakes: []*apiservice.AnswerItem{
					&apiservice.AnswerItem{
						PotentialAnswerID: potentialAnswerID,
					},
				},
			},
		},
	}
	// now, lets go ahead and answer the question for the patient
	SubmitAnswersIntakeForPatient(pr.Patient.ID.Int64(), pr.Patient.AccountID.Int64(), &rb, testData, t)

	// now, get the patient visit again to ensure that a patient answer was registered for the intended question
	patientVisitResponse = GetPatientVisitForPatient(pr.Patient.ID.Int64(), testData, t)

	// lets go through the questions to find the one for which the patient answer should be present
	for _, section := range patientVisitResponse.ClientLayout.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {
				if question.QuestionID == questionID {
					if question.Answers == nil || len(question.Answers) == 0 {
						t.Fatalf("Expected patient answer for question with id %d, but got none", questionID)
					}
					for _, answer := range GetAnswerIntakesFromAnswers(question.Answers, t) {
						if answer.PotentialAnswerID.Int64() == potentialAnswerID {
							return
						}
					}
				}
			}
		}
	}

	t.Fatalf("While a patient answer exists for question with id %d, unable to find the expected potential answer with id %d", questionID, potentialAnswerID)
}

func TestMultipleChoiceIntake(t *testing.T) {

	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	// signup a random test patient for which to answer questions
	pr := SignupRandomTestPatientWithPharmacyAndAddress(t, testData)
	patientVisitResponse := CreatePatientVisitForPatient(pr.Patient.ID.Int64(), testData, t)

	// now lets go ahead and try and answer the question about the reason for visit given that it is
	// single select
	questionID := getQuestionWithTagAndExpectedType("q_acne_prev_treatment_types", "q_type_multiple_choice", t, testData)
	potentialAnswers, err := testData.DataAPI.GetAnswerInfo(questionID, 1)
	if err != nil {
		t.Fatal("Unable to get answers for question with id " + strconv.FormatInt(questionID, 10))
	}

	intakeData := apiservice.IntakeData{
		PatientVisitID: patientVisitResponse.PatientVisitID,
	}

	qaItem := &apiservice.QuestionAnswerItem{
		QuestionID: questionID,
	}
	for _, potentialAnswer := range potentialAnswers {
		if potentialAnswer.AnswerTag == "a_otc_prev_treatment_type" || potentialAnswer.AnswerTag == "a_prescription_prev_treatment_type" {
			qaItem.AnswerIntakes = append(qaItem.AnswerIntakes, &apiservice.AnswerItem{PotentialAnswerID: potentialAnswer.AnswerID})
		}
	}
	intakeData.Questions = []*apiservice.QuestionAnswerItem{qaItem}

	SubmitAnswersIntakeForPatient(pr.Patient.ID.Int64(), pr.Patient.AccountID.Int64(), &intakeData, testData, t)

	// now, get the patient visit again to ensure that a patient answer was registered for the intended question
	patientVisitResponse = GetPatientVisitForPatient(pr.Patient.ID.Int64(), testData, t)

	// lets go through the questions to find the one for which the patient answer should be present
	for _, section := range patientVisitResponse.ClientLayout.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {
				if question.QuestionID == questionID {
					if question.Answers == nil || len(question.Answers) == 0 {
						t.Fatalf("Expected patient answer for question with id %d, but got none", questionID)
					}
					for _, answer := range GetAnswerIntakesFromAnswers(question.Answers, t) {
						answerNotFound := true
						for _, questionItem := range intakeData.Questions {
							for _, answerIntake := range questionItem.AnswerIntakes {
								if answerIntake.PotentialAnswerID == answer.PotentialAnswerID.Int64() {
									answerNotFound = false
								}
							}
						}
						if answerNotFound {
							t.Fatal("Expected answer not found in patient answer for patient visit when testing for answering of multiple choice questions.")
						}
					}
				}
			}
		}
	}
}

func TestSingleEntryIntake(t *testing.T) {

	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	// signup a random test patient for which to answer questions
	pr := SignupRandomTestPatientWithPharmacyAndAddress(t, testData)
	patientVisitResponse := CreatePatientVisitForPatient(pr.Patient.ID.Int64(), testData, t)

	questionID := getQuestionWithTagAndExpectedType("q_other_skin_condition_entry", "q_type_single_entry", t, testData)
	potentialAnswerID := getAnswerWithTagAndExpectedType("a_other_skin_condition_entry", "a_type_single_entry", questionID, testData, t)
	intakeData := apiservice.IntakeData{}
	intakeData.PatientVisitID = patientVisitResponse.PatientVisitID

	qaItem := &apiservice.QuestionAnswerItem{
		QuestionID:    questionID,
		AnswerIntakes: []*apiservice.AnswerItem{&apiservice.AnswerItem{PotentialAnswerID: potentialAnswerID, AnswerText: "testAnswer"}},
	}
	intakeData.Questions = []*apiservice.QuestionAnswerItem{qaItem}
	SubmitAnswersIntakeForPatient(pr.Patient.ID.Int64(), pr.Patient.AccountID.Int64(), &intakeData, testData, t)

	// now, get the patient visit again to ensure that a patient answer was registered for the intended question
	patientVisitResponse = GetPatientVisitForPatient(pr.Patient.ID.Int64(), testData, t)

	// lets go through the questions to find the one for which the patient answer should be present
	for _, section := range patientVisitResponse.ClientLayout.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {
				if question.QuestionID == questionID {
					if question.Answers == nil || len(question.Answers) == 0 {
						t.Fatalf("Expected patient answer for question with id %d, but got none", questionID)
					}
					for _, answer := range GetAnswerIntakesFromAnswers(question.Answers, t) {
						if answer.PotentialAnswerID.Int64() == potentialAnswerID && answer.AnswerText == "testAnswer" {
							return
						}
					}
				}
			}
		}
	}
	t.Fatalf("While an answer for the expected question exists, unable to find the expected answer with id %d for single entry intake test", potentialAnswerID)
}

func TestFreeTextEntryIntake(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	// signup a random test patient for which to answer questions
	pr := SignupRandomTestPatientWithPharmacyAndAddress(t, testData)
	patientVisitResponse := CreatePatientVisitForPatient(pr.Patient.ID.Int64(), testData, t)
	freeTextResponse := "This is a free text response that should be accepted as a response for free text."
	submitFreeTextResponseForPatient(
		patientVisitResponse,
		pr.Patient.ID.Int64(),
		pr.Patient.AccountID.Int64(),
		freeTextResponse,
		testData,
		t)

	// submit another free text response to update teh response to this questiuon to ensure that what is returned is this response
	// for this questions
	updatedFreeTextResponse := "This is an updated free text response"
	submitFreeTextResponseForPatient(
		patientVisitResponse,
		pr.Patient.ID.Int64(),
		pr.Patient.AccountID.Int64(),
		updatedFreeTextResponse,
		testData,
		t)
}

// this test simulates the out of ordering processing
// of a patient response to a question where an older response
// to a question is received after an updated response. The server
// should reject the older response and keep the newer response intact
func TestIntake_ClientOrdering(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	// signup a random test patient for which to answer questions
	pr := SignupRandomTestPatientWithPharmacyAndAddress(t, testData)
	pv := CreatePatientVisitForPatient(
		pr.Patient.ID.Int64(),
		testData,
		t)

	// answer a question with free text input
	questionID := getQuestionWithTagAndExpectedType("q_anything_else_acne", "q_type_free_text", t, testData)
	response1 := "response1"
	rb := apiservice.IntakeData{
		PatientVisitID: pv.PatientVisitID,
		SessionID:      "68753A44-4D6F-1226-9C60-0050E4C00067",
		SessionCounter: 10,
		Questions: []*apiservice.QuestionAnswerItem{
			{
				QuestionID: questionID,
				AnswerIntakes: []*apiservice.AnswerItem{
					{AnswerText: response1},
				},
			},
		},
	}

	SubmitAnswersIntakeForPatient(
		pr.Patient.ID.Int64(),
		pr.Patient.AccountID.Int64(),
		&rb,
		testData, t)

	// attempt to answer again with another response but one that is an older response
	// from the client
	response2 := "response2"
	rb = apiservice.IntakeData{
		PatientVisitID: pv.PatientVisitID,
		SessionID:      "68753A44-4D6F-1226-9C60-0050E4C00067",
		SessionCounter: 9,
		Questions: []*apiservice.QuestionAnswerItem{
			{
				QuestionID: questionID,
				AnswerIntakes: []*apiservice.AnswerItem{
					{AnswerText: response2},
				},
			},
		},
	}

	SubmitAnswersIntakeForPatient(pr.Patient.ID.Int64(), pr.Patient.AccountID.Int64(), &rb, testData, t)

	patientVisit, err := testData.DataAPI.GetPatientVisitFromID(pv.PatientVisitID)
	test.OK(t, err)

	// the second response should be rejected given that it was an older response
	answers, err := testData.DataAPI.AnswersForQuestions([]int64{questionID}, &api.PatientIntake{
		PatientID:      pr.Patient.ID.Int64(),
		PatientVisitID: patientVisit.ID.Int64(),
		LVersionID:     patientVisit.LayoutVersionID.Int64(),
	})
	test.OK(t, err)
	test.Equals(t, response1, answers[questionID][0].(*common.AnswerIntake).AnswerText)
}

func submitFreeTextResponseForPatient(
	patientVisitResponse *patient.PatientVisitResponse,
	patientID, patientAccountID int64,
	freeTextResponse string,
	testData *TestData,
	t *testing.T) {
	// now lets go ahead and try and answer the question about the reason for visit given that it is
	// single select
	questionID := getQuestionWithTagAndExpectedType("q_anything_else_acne", "q_type_free_text", t, testData)
	intakeData := apiservice.IntakeData{
		PatientVisitID: patientVisitResponse.PatientVisitID,
	}

	qaItem := &apiservice.QuestionAnswerItem{
		QuestionID:    questionID,
		AnswerIntakes: []*apiservice.AnswerItem{&apiservice.AnswerItem{AnswerText: freeTextResponse}},
	}

	intakeData.Questions = []*apiservice.QuestionAnswerItem{qaItem}

	SubmitAnswersIntakeForPatient(patientID, patientAccountID, &intakeData, testData, t)

	// now, get the patient visit again to ensure that a patient answer was registered for the intended question
	patientVisitResponse = GetPatientVisitForPatient(patientID, testData, t)

	// lets go through the questions to find the one for which the patient answer should be present
	for _, section := range patientVisitResponse.ClientLayout.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {
				if question.QuestionID == questionID {
					if question.Answers == nil || len(question.Answers) == 0 {
						t.Fatalf("Expected patient answer for question with id %d, but got none", questionID)
					}
					for _, answer := range GetAnswerIntakesFromAnswers(question.Answers, t) {
						if answer.AnswerText == freeTextResponse {
							return
						}
					}
				}
			}
		}
	}

	t.Fatalf("While an answer for the expected question exists, unable to find the expected answer with free text %s for free text intake test", freeTextResponse)
}

func addSubAnswerToAnswerIntake(answerIntake *apiservice.AnswerItem, subAnswerQuestionID, subAnswerPotentialAnswerID int64) {
	qaItem := &apiservice.QuestionAnswerItem{}
	qaItem.QuestionID = subAnswerQuestionID
	qaItem.AnswerIntakes = []*apiservice.AnswerItem{&apiservice.AnswerItem{PotentialAnswerID: subAnswerPotentialAnswerID}}
	if answerIntake.SubQuestions == nil {
		answerIntake.SubQuestions = make([]*apiservice.QuestionAnswerItem, 0)
	}
	answerIntake.SubQuestions = append(answerIntake.SubQuestions, qaItem)
}
