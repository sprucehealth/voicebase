package integration

import (
	"carefront/api"
	"carefront/apiservice"
	// "carefront/config"
	"encoding/json"
	// _ "github.com/go-sql-driver/mysql"
	"bytes"
	"fmt"
	// "io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

type AnswerIntakeHandler struct {
	DataApi   api.DataAPI
	accountId int64
}

func getQuestionWithTagAndExpectedType(questionTag, questionType string, t *testing.T, testData TestData) int64 {
	questionId, _, questionType, _, _, err := testData.DataApi.GetQuestionInfo(questionTag, 1)
	if err != nil {
		t.Fatal("Unable to query for question q_reason_visit from database: " + err.Error())
	}

	// need to ensure that the question we are trying to get the information for is a single select
	// question type
	if questionType != questionType {
		t.Fatal("Expected q_reason_visit to be q_type_single_select but it's not.")
	}

	return questionId
}

func getAnswerWithTagAndExpectedType(answerTag, answerType string, questionId int64, testData TestData, t *testing.T) int64 {

	potentialAnswers, err := testData.DataApi.GetAnswerInfo(questionId, 1)
	if err != nil {
		t.Fatal("Unable to get answers for question with id " + strconv.FormatInt(questionId, 10))
	}

	expectedAnswerTag := answerTag
	var potentialAnswerId int64
	var potentialAnswerType string
	for _, potentialAnswer := range potentialAnswers {
		if potentialAnswer.AnswerTag == expectedAnswerTag {
			potentialAnswerId = potentialAnswer.PotentialAnswerId
			potentialAnswerType = potentialAnswer.AnswerType
		}
	}

	if potentialAnswerId == 0 {
		t.Fatal("Unable to find the answer for the question with intended answer tag " + expectedAnswerTag)
	}

	if potentialAnswerType != answerType {
		t.Fatalf("Potential answer found does not have matching type. Expected %s, Found %s ", answerType, potentialAnswerType)
	}

	return potentialAnswerId
}

func submitPatientAnswerForVisit(PatientId int64, testData TestData, patientIntakeRequestData string, t *testing.T) {
	answerIntakeHandler := apiservice.NewAnswerIntakeHandler(testData.DataApi)
	patient, err := testData.DataApi.GetPatientFromId(PatientId)
	if err != nil {
		t.Fatal("Unable to get patient information given the patient id when trying to enter patient intake: " + err.Error())
	}

	answerIntakeHandler.AccountIdFromAuthToken(patient.AccountId)
	ts := httptest.NewServer(answerIntakeHandler)
	defer ts.Close()

	client := &http.Client{}
	req, _ := http.NewRequest("POST", ts.URL, bytes.NewBufferString(patientIntakeRequestData))
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal("Unable to get the patient visit id")
	}
	CheckSuccessfulStatusCode(resp, "Unable to submit a single select answer for patient", t)
}

func TestSingleSelectIntake(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}

	testData := SetupIntegrationTest(t)
	defer testData.DB.Close()

	// signup a random test patient for which to answer questions
	patientSignedUpResponse := SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patientVisitResponse := GetPatientVisitForPatient(patientSignedUpResponse.PatientId, testData, t)

	// now lets go ahead and try and answer the question about the reason for visit given that it is
	// single select
	questionId := getQuestionWithTagAndExpectedType("q_reason_visit", "q_type_single_select", t, testData)
	potentialAnswerId := getAnswerWithTagAndExpectedType("a_acne", "a_type_multiple_choice", questionId, testData, t)

	// lets go ahead and populate a response for the question
	patientIntakeRequestData := fmt.Sprintf(`{"patient_visit_id": %d, "potential_answers": [{"potential_answer_id": %d } ], "question_id": %d }`, patientVisitResponse.PatientVisitId, potentialAnswerId, questionId)

	// now, lets go ahead and answer the question for the patient
	submitPatientAnswerForVisit(patientSignedUpResponse.PatientId, testData, patientIntakeRequestData, t)

	// now, get the patient visit again to ensure that a patient answer was registered for the intended question
	patientVisitResponse = GetPatientVisitForPatient(patientSignedUpResponse.PatientId, testData, t)

	// lets go through the questions to find the one for which the patient answer should be present
	for _, section := range patientVisitResponse.ClientLayout.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {
				if question.QuestionId == questionId {
					if question.PatientAnswers == nil || len(question.PatientAnswers) == 0 {
						t.Fatalf("Expected patient answer for question with id %d, but got none", questionId)
					}
					for _, patientAnswer := range question.PatientAnswers {
						if patientAnswer.PotentialAnswerId == potentialAnswerId {
							return
						}
					}
				}
			}
		}
	}

	t.Fatalf("While a patient answer exists for question with id %d, unable to find the expected potential answer with id %d", questionId, potentialAnswerId)
}

func TestMultipleChoiceIntake(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}
	testData := SetupIntegrationTest(t)
	defer testData.DB.Close()

	// signup a random test patient for which to answer questions
	patientSignedUpResponse := SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patientVisitResponse := GetPatientVisitForPatient(patientSignedUpResponse.PatientId, testData, t)

	// now lets go ahead and try and answer the question about the reason for visit given that it is
	// single select
	questionId := getQuestionWithTagAndExpectedType("q_acne_prev_treatment_types", "q_type_multiple_choice", t, testData)
	potentialAnswers, err := testData.DataApi.GetAnswerInfo(questionId, 1)
	if err != nil {
		t.Fatal("Unable to get answers for question with id " + strconv.FormatInt(questionId, 10))
	}

	answerIntakeRequestBody := apiservice.AnswerIntakeRequestBody{}
	answerIntakeRequestBody.PatientVisitId = patientVisitResponse.PatientVisitId
	answerIntakeRequestBody.QuestionId = questionId
	for _, potentialAnswer := range potentialAnswers {
		if potentialAnswer.AnswerTag == "a_otc_prev_treatment_type" || potentialAnswer.AnswerTag == "a_prescription_prev_treatment_type" {
			answerIntakeRequestBody.AnswerIntakes = append(answerIntakeRequestBody.AnswerIntakes, &apiservice.AnswerIntake{PotentialAnswerId: potentialAnswer.PotentialAnswerId})
		}
	}

	requestData, err := json.Marshal(&answerIntakeRequestBody)
	if err != nil {
		t.Fatal("Unable to marshal request body")
	}
	submitPatientAnswerForVisit(patientSignedUpResponse.PatientId, testData, string(requestData), t)
	// now, get the patient visit again to ensure that a patient answer was registered for the intended question
	patientVisitResponse = GetPatientVisitForPatient(patientSignedUpResponse.PatientId, testData, t)

	// lets go through the questions to find the one for which the patient answer should be present
	for _, section := range patientVisitResponse.ClientLayout.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {
				if question.QuestionId == questionId {
					if question.PatientAnswers == nil || len(question.PatientAnswers) == 0 {
						t.Fatalf("Expected patient answer for question with id %d, but got none", questionId)
					}
					for _, patientAnswer := range question.PatientAnswers {
						answerNotFound := true
						for _, answerIntake := range answerIntakeRequestBody.AnswerIntakes {
							if answerIntake.PotentialAnswerId == patientAnswer.PotentialAnswerId {
								answerNotFound = false
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
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}
	testData := SetupIntegrationTest(t)
	defer testData.DB.Close()

	// signup a random test patient for which to answer questions
	patientSignedUpResponse := SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patientVisitResponse := GetPatientVisitForPatient(patientSignedUpResponse.PatientId, testData, t)

	// now lets go ahead and try and answer the question about the reason for visit given that it is
	// single select
	questionId := getQuestionWithTagAndExpectedType("q_condition_for_diagnosis", "q_type_single_entry", t, testData)
	potentialAnswerId := getAnswerWithTagAndExpectedType("a_condition_entry", "a_type_single_entry", questionId, testData, t)
	answerIntakeRequestBody := apiservice.AnswerIntakeRequestBody{}
	answerIntakeRequestBody.PatientVisitId = patientVisitResponse.PatientVisitId
	answerIntakeRequestBody.QuestionId = questionId
	answerIntakeRequestBody.AnswerIntakes = []*apiservice.AnswerIntake{&apiservice.AnswerIntake{PotentialAnswerId: potentialAnswerId, AnswerText: "testAnswer"}}
	requestData, err := json.Marshal(&answerIntakeRequestBody)
	if err != nil {
		t.Fatal("Unable to marshal request body")
	}
	submitPatientAnswerForVisit(patientSignedUpResponse.PatientId, testData, string(requestData), t)
	// now, get the patient visit again to ensure that a patient answer was registered for the intended question
	patientVisitResponse = GetPatientVisitForPatient(patientSignedUpResponse.PatientId, testData, t)

	// lets go through the questions to find the one for which the patient answer should be present
	for _, section := range patientVisitResponse.ClientLayout.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {
				if question.QuestionId == questionId {
					if question.PatientAnswers == nil || len(question.PatientAnswers) == 0 {
						t.Fatalf("Expected patient answer for question with id %d, but got none", questionId)
					}
					for _, patientAnswer := range question.PatientAnswers {
						if patientAnswer.PotentialAnswerId == potentialAnswerId && patientAnswer.AnswerText == "testAnswer" {
							return
						}
					}
				}
			}
		}
	}
	t.Fatalf("While an answer for the expected question exists, unable to find the expected answer with id %d for single entry intake test", potentialAnswerId)
}

func TestFreeTextEntryIntake(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}

}

func TestSubQuestionEntryIntake(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}

}

func TestMultipleAnswersForSamePotentialAnswerIntake(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}

}

func TestPhotoAnswerIntake(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}

}
