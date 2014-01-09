package integration

import (
	"testing"
)

func TestNoPotentialAnswerForQuestionTypes(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}

	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	// no free text question type should have potential answers associated with it
	rows, err := testData.DB.Query(`select question.id from question inner join question_type on question_type.id = question.qtype_id where question_type.qtype in ('q_type_free_text', 'q_type_autocomplete')`)
	if err != nil {
		t.Fatal("Unable to query database for a list of question ids : " + err.Error())
	}

	questionIds := make([]int64, 0)
	for rows.Next() {
		var id int64
		rows.Scan(&id)
		questionIds = append(questionIds, id)
	}

	// for each of these question ids, there should be no potential responses
	for _, questionId := range questionIds {
		answerInfos, err := testData.DataApi.GetAnswerInfo(questionId, 1)
		if err != nil {
			t.Fatal("Error when trying to get answer for question (which should return no answers) : " + err.Error())
		}
		if !(answerInfos == nil || len(answerInfos) == 0) {
			t.Fatal("No potential answers should be returned for these questions")
		}
	}
}

// This test is to ensure that additional fields are set for the
// autocomplete question type, as they should be for the client to
// be able to show additional pieces of content in the question
func TestAdditionalFieldsInAutocompleteQuestion(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	// signup a random test patient for which to answer questions
	patientSignedUpResponse := SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patientVisitResponse := CreatePatientVisitForPatient(patientSignedUpResponse.PatientId, testData, t)

	// lets go through the questions to find the one for which the patient answer should be present
	for _, section := range patientVisitResponse.ClientLayout.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {
				if question.QuestionTypes[0] == "q_type_autocomplete" {
					if question.AdditionalFields == nil || len(question.AdditionalFields) == 0 {
						t.Fatal("Expected additional fields to be set for the autocomplete question type")
					}
				}
			}
		}
	}
}
