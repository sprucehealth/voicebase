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
	genericUserErrorMessage         = "Something went wrong on our end. Apologies for the inconvenience and please try again later!"
	DEVELOPER_ERROR_NO_VISIT_EXISTS = 10001
)

type GenericJsonResponse struct {
	Result string `json:"result"`
}

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

func SuccessfulGenericJSONResponse() *GenericJsonResponse {
	return &GenericJsonResponse{Result: "success"}
}

type ErrorResponse struct {
	DeveloperError string `json:"developer_error,omitempty"`
	DeveloperCode  int64  `json:"developer_code,string,omitempty"`
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

func WriteDeveloperErrorWithCode(w http.ResponseWriter, developerStatusCode int64, httpStatusCode int, errorString string) {
	developerError := new(ErrorResponse)
	developerError.DeveloperError = errorString
	developerError.DeveloperCode = developerStatusCode
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
	QuestionId    int64         `json:"question_id"`
	AnswerIntakes []*AnswerItem `json:"potential_answers,omitempty"`
}

type AnswerItem struct {
	PotentialAnswerId        int64                      `json:"potential_answer_id"`
	AnswerText               string                     `json:"answer_text"`
	SubQuestionAnswerIntakes []*SubQuestionAnswerIntake `json:"answers,omitempty"`
}

type AnswerToQuestionItem struct {
	QuestionId    int64         `json:"question_id"`
	AnswerIntakes []*AnswerItem `json:"potential_answers"`
}

type AnswerIntakeRequestBody struct {
	PatientVisitId int64                   `json:"patient_visit_id"`
	Questions      []*AnswerToQuestionItem `json:"questions"`
}

type AnswerIntakeResponse struct {
	Result string `json:"result"`
}

func validateRequestBody(answerIntakeRequestBody *AnswerIntakeRequestBody, w http.ResponseWriter) error {
	if answerIntakeRequestBody.PatientVisitId == 0 {
		return errors.New("patient_visit_id missing")
	}

	if answerIntakeRequestBody.Questions == nil || len(answerIntakeRequestBody.Questions) == 0 {
		return errors.New("missing patient information to save for patient visit.")
	}

	for _, questionItem := range answerIntakeRequestBody.Questions {
		if questionItem.QuestionId == 0 {
			return errors.New("question_id missing")
		}

		if questionItem.AnswerIntakes == nil {
			return errors.New("potential_answers missing")
		}
	}

	return nil
}

func populateAnswersToStoreForQuestion(role string, answerToQuestionItem *AnswerToQuestionItem, patientVisitId, roleId, layoutVersionId int64) (answersToStore []*common.AnswerIntake) {
	// get a list of top level answers to store for each of the quetions
	answersToStore = createAnswersToStoreForQuestion(role, roleId, answerToQuestionItem.QuestionId,
		patientVisitId, layoutVersionId, answerToQuestionItem.AnswerIntakes)

	// go through all the answers of each question intake to identify responses that have responses to subquestions
	// embedded in them, and add that to the list of answers to store in the database
	for i, answerIntake := range answerToQuestionItem.AnswerIntakes {
		if answerIntake.SubQuestionAnswerIntakes != nil {
			subAnswers := make([]*common.AnswerIntake, 0)
			for _, subAnswer := range answerIntake.SubQuestionAnswerIntakes {
				subAnswers = append(subAnswers, createAnswersToStoreForQuestion(role, roleId, subAnswer.QuestionId, patientVisitId, layoutVersionId, subAnswer.AnswerIntakes)...)
			}
			answersToStore[i].SubAnswers = subAnswers
		}
	}
	return answersToStore
}

func createAnswersToStoreForQuestion(role string, roleId, questionId, patientVisitId, layoutVersionId int64, answerIntakes []*AnswerItem) []*common.AnswerIntake {
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
