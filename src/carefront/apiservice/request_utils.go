package apiservice

import (
	"carefront/api"
	"carefront/common"
	"carefront/info_intake"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"
)

var ErrBadAuthToken = errors.New("BadAuthToken")

const (
	genericUserErrorMessage = "Something went wrong on our end. Apologies for the inconvenience and please try again later!"
)

func GetAuthTokenFromHeader(r *http.Request) (string, error) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return "", ErrBadAuthToken
	}

	parts := strings.Split(auth, " ")
	if len(parts) != 2 || parts[0] != "token" {
		return "", ErrBadAuthToken
	}
	return parts[1], nil
}

func GetSignedUrlsForAnswersInQuestion(question *info_intake.Question, photoStorageService api.CloudStorageAPI) {
	// go through each answer to get signed urls
	for _, patientAnswer := range question.PatientAnswers {
		if patientAnswer.StorageKey != "" {
			objectUrl, err := photoStorageService.GetSignedUrlForObjectAtLocation(patientAnswer.StorageBucket,
				patientAnswer.StorageKey, patientAnswer.StorageRegion, time.Now().Add(10*time.Minute))
			if err != nil {
				log.Fatal("Unable to get signed url for photo object: " + err.Error())
			} else {
				patientAnswer.ObjectUrl = objectUrl
			}
		}
	}
}

type ErrorResponse struct {
	DeveloperError string `json:"developer_error,omitempty"`
	UserError      string `json:"user_error,omitempty"`
}

func WriteJSONToHTTPResponseWriter(w http.ResponseWriter, httpStatusCode int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatusCode)
	enc := json.NewEncoder(w)
	if err := enc.Encode(v); err != nil {
		log.Printf("apiservice: failed to json encode: %+v", err)
	}
}

func WriteDeveloperError(w http.ResponseWriter, httpStatusCode int, errorString string) {
	log.Println(errorString)
	developerError := new(ErrorResponse)
	developerError.DeveloperError = errorString
	developerError.UserError = genericUserErrorMessage
	WriteJSONToHTTPResponseWriter(w, httpStatusCode, developerError)
}

func WriteUserError(w http.ResponseWriter, httpStatusCode int, errorString string) {
	userError := new(ErrorResponse)
	userError.UserError = errorString
	WriteJSONToHTTPResponseWriter(w, httpStatusCode, userError)
}

// this structure is present only if we are taking in answers to subquestions
// linked to a root question.
// Note that the structure has been created to be flexible enough to have any kind of
// question type as a subquestion; although we won't have subquestions to subquestions
type SubQuestionAnswerIntake struct {
	QuestionId    int64                      `json:"question_id"`
	AnswerIntakes []*AnswerIntakeRequestItem `json:"potential_answers,omitempty"`
}

type AnswerIntakeRequestItem struct {
	PotentialAnswerId        int64                      `json:"potential_answer_id"`
	AnswerText               string                     `json:"answer_text"`
	SubQuestionAnswerIntakes []*SubQuestionAnswerIntake `json:"answers,omitempty"`
}

type AnswerIntakeRequestBody struct {
	PatientVisitId int64                      `json:"patient_visit_id"`
	QuestionId     int64                      `json:"question_id"`
	AnswerIntakes  []*AnswerIntakeRequestItem `json:"potential_answers"`
}

type AnswerIntakeResponse struct {
	AnswerIds []int64 `json:"answer_ids"`
}

func validateRequestBody(answerIntakeRequestBody *AnswerIntakeRequestBody, w http.ResponseWriter) error {
	if answerIntakeRequestBody.PatientVisitId == 0 {
		return errors.New("patient_visit_id missing")
	}

	if answerIntakeRequestBody.QuestionId == 0 {
		return errors.New("question_id missing")
	}

	if answerIntakeRequestBody.AnswerIntakes == nil {
		return errors.New("potential_answers missing")
	}
	return nil
}

func populateAnswersToStore(role string, answerIntakeRequestBody *AnswerIntakeRequestBody, roleId, layoutVersionId int64) (answersToStore []*common.AnswerIntake) {
	// get a list of top level answers to store for each of the quetions
	answersToStore = createAnswersToStore(role, roleId, answerIntakeRequestBody.QuestionId,
		answerIntakeRequestBody.PatientVisitId, layoutVersionId, answerIntakeRequestBody.AnswerIntakes)

	// go through all the answers of each question intake to identify responses that have responses to subquestions
	// embedded in them, and add that to the list of answers to store in the database
	for i, answerIntake := range answerIntakeRequestBody.AnswerIntakes {
		if answerIntake.SubQuestionAnswerIntakes != nil {
			subAnswers := make([]*common.AnswerIntake, 0)
			for _, subAnswer := range answerIntake.SubQuestionAnswerIntakes {
				subAnswers = append(subAnswers, createAnswersToStore(role, roleId, subAnswer.QuestionId, answerIntakeRequestBody.PatientVisitId, layoutVersionId, subAnswer.AnswerIntakes)...)
			}
			answersToStore[i].SubAnswers = subAnswers
		}
	}
	return answersToStore
}

func createAnswersToStore(role string, roleId, questionId, patientVisitId, layoutVersionId int64, answerIntakes []*AnswerIntakeRequestItem) []*common.AnswerIntake {
	answersToStore := make([]*common.AnswerIntake, 0)
	for _, answerIntake := range answerIntakes {
		answerToStore := new(common.AnswerIntake)
		answerToStore.RoleId = roleId
		answerToStore.Role = role
		answerToStore.QuestionId = questionId
		answerToStore.PatientVisitId = patientVisitId
		answerToStore.LayoutVersionId = layoutVersionId
		answerToStore.PotentialAnswerId = answerIntake.PotentialAnswerId
		answerToStore.AnswerText = answerIntake.AnswerText
		answersToStore = append(answersToStore, answerToStore)
	}
	return answersToStore
}
