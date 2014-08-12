package apiservice

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/golog"

	"github.com/sprucehealth/backend/third_party/github.com/SpruceHealth/schema"
)

var (
	ErrBadAuthHeader = errors.New("bad authorization header")
	ErrNoAuthHeader  = errors.New("no authorization header")
)

var Testing = false

const (
	genericUserErrorMessage                       = "Something went wrong on our end. Apologies for the inconvenience and please try again later!"
	authTokenExpiredMessage                       = "Authentication expired. Log in to continue."
	DEVELOPER_ERROR_NO_VISIT_EXISTS               = 10001
	DEVELOPER_AUTH_TOKEN_EXPIRED                  = 10002
	DEVELOPER_TREATMENT_MISSING_DNTF              = 10003
	DEVELOPER_NO_TREATMENT_PLAN                   = 10004
	DEVELOPER_JBCQ_FORBIDDEN                      = 10005
	DEVELOPER_CONTROLLED_SUBSTANCE_REFILL_REQUEST = 10006
	HTTP_GET                                      = "GET"
	HTTP_POST                                     = "POST"
	HTTP_PUT                                      = "PUT"
	HTTP_DELETE                                   = "DELETE"
	StatusUnprocessableEntity                     = 422
	signedUrlAuthTimeout                          = 10 * time.Minute
	HEALTH_CONDITION_ACNE_ID                      = 1
	TimeFormatLayout                              = "January 2 at 3:04pm"
)

type GenericJsonResponse struct {
	Result string `json:"result"`
}

func GetAuthTokenFromHeader(r *http.Request) (string, error) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return "", ErrNoAuthHeader
	}

	parts := strings.Split(auth, " ")
	if len(parts) != 2 || parts[0] != "token" {
		return "", ErrBadAuthHeader
	}
	return parts[1], nil
}

func HandleAuthError(err error, w http.ResponseWriter) {
	switch err {
	case ErrBadAuthHeader, ErrNoAuthHeader, api.TokenExpired, api.TokenDoesNotExist:
		golog.Context("AuthEvent", AuthEventInvalidToken).Infof(err.Error())
		WriteAuthTimeoutError(w)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func EnsureTreatmentPlanOrPatientVisitIdPresent(dataApi api.DataAPI, treatmentPlanId int64, patientVisitId *int64) error {
	if patientVisitId == nil {
		return fmt.Errorf("PatientVisitId should not be nil!")
	}

	if *patientVisitId == 0 && treatmentPlanId == 0 {
		return errors.New("Either patientVisitId or treatmentPlanId should be specified")
	}

	if *patientVisitId == 0 {
		patientVisitIdFromTreatmentPlanId, err := dataApi.GetPatientVisitIdFromTreatmentPlanId(treatmentPlanId)
		if err != nil {
			return errors.New("Unable to get patient visit id from treatmentPlanId: " + err.Error())
		}
		*patientVisitId = patientVisitIdFromTreatmentPlanId
	}

	return nil
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
	if err := json.NewEncoder(w).Encode(v); err != nil {
		golog.Errorf("apiservice: failed to json encode: %+v", err)
	}
}

func WriteJSON(w http.ResponseWriter, v interface{}) {
	WriteJSONToHTTPResponseWriter(w, http.StatusOK, v)
}

func WriteJSONSuccess(w http.ResponseWriter) {
	WriteJSONToHTTPResponseWriter(w, http.StatusOK, SuccessfulGenericJSONResponse())
}

func WriteErrorResponse(w http.ResponseWriter, httpStatusCode int, errorResponse ErrorResponse) {
	golog.Logf(1, golog.ERR, errorResponse.DeveloperError)
	WriteJSONToHTTPResponseWriter(w, httpStatusCode, &errorResponse)
}

func WriteDeveloperError(w http.ResponseWriter, httpStatusCode int, errorString string) {
	golog.Logf(1, golog.ERR, errorString)
	developerError := &ErrorResponse{
		DeveloperError: errorString,
		UserError:      genericUserErrorMessage,
	}
	WriteJSONToHTTPResponseWriter(w, httpStatusCode, developerError)
}

func WriteDeveloperErrorWithCode(w http.ResponseWriter, developerStatusCode int64, httpStatusCode int, errorString string) {
	golog.Logf(1, golog.ERR, errorString)
	developerError := &ErrorResponse{
		DeveloperError: errorString,
		DeveloperCode:  developerStatusCode,
		UserError:      genericUserErrorMessage,
	}
	WriteJSONToHTTPResponseWriter(w, httpStatusCode, developerError)
}

func WriteUserError(w http.ResponseWriter, httpStatusCode int, errorString string) {
	userError := &ErrorResponse{
		UserError: errorString,
	}
	WriteJSONToHTTPResponseWriter(w, httpStatusCode, userError)
}

func WriteAuthTimeoutError(w http.ResponseWriter) {
	userError := &ErrorResponse{
		UserError:      authTokenExpiredMessage,
		DeveloperCode:  DEVELOPER_AUTH_TOKEN_EXPIRED,
		DeveloperError: authTokenExpiredMessage,
	}
	WriteJSONToHTTPResponseWriter(w, http.StatusForbidden, userError)
}

func DecodeRequestData(requestData interface{}, r *http.Request) error {
	switch r.Header.Get("Content-Type") {
	case "application/json", "text/json":
		if err := json.NewDecoder(r.Body).Decode(requestData); err != nil {
			return fmt.Errorf("Unable to parse input parameters: %s", err)
		}
	default:
		if err := r.ParseForm(); err != nil {
			return fmt.Errorf("Unable to parse input parameters: %s", err)
		}
		if err := schema.NewDecoder().Decode(requestData, r.Form); err != nil {
			return fmt.Errorf("Unable to parse input parameters: %s", err)
		}
	}

	return nil
}

// this structure is present only if we are taking in answers to subquestions
// linked to a root question.
// Note that the structure has been created to be flexible enough to have any kind of
// question type as a subquestion; although we won't have subquestions to subquestions
type SubQuestionAnswerIntake struct {
	QuestionId    int64         `json:"question_id,string"`
	AnswerIntakes []*AnswerItem `json:"potential_answers,omitempty"`
}

type AnswerItem struct {
	PotentialAnswerId        int64                      `json:"potential_answer_id,string"`
	AnswerText               string                     `json:"answer_text"`
	SubQuestionAnswerIntakes []*SubQuestionAnswerIntake `json:"answers,omitempty"`
}

type AnswerToQuestionItem struct {
	QuestionId    int64         `json:"question_id,string"`
	AnswerIntakes []*AnswerItem `json:"potential_answers"`
}

type AnswerIntakeRequestBody struct {
	PatientVisitId int64                   `json:"patient_visit_id,string"`
	Questions      []*AnswerToQuestionItem `json:"questions"`
}

type AnswerIntakeResponse struct {
	Result string `json:"result"`
}

func ValidateRequestBody(answerIntakeRequestBody *AnswerIntakeRequestBody, w http.ResponseWriter) error {
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

func PopulateAnswersToStoreForQuestion(role string, answerToQuestionItem *AnswerToQuestionItem, contextId, roleId, layoutVersionId int64) []*common.AnswerIntake {
	// get a list of top level answers to store for each of the quetions
	answersToStore := createAnswersToStoreForQuestion(role, roleId, answerToQuestionItem.QuestionId,
		contextId, layoutVersionId, answerToQuestionItem.AnswerIntakes)

	// go through all the answers of each question intake to identify responses that have responses to subquestions
	// embedded in them, and add that to the list of answers to store in the database
	for i, answerIntake := range answerToQuestionItem.AnswerIntakes {
		if answerIntake.SubQuestionAnswerIntakes != nil {
			subAnswers := make([]*common.AnswerIntake, 0)
			for _, subAnswer := range answerIntake.SubQuestionAnswerIntakes {
				subAnswers = append(subAnswers, createAnswersToStoreForQuestion(role, roleId, subAnswer.QuestionId, contextId, layoutVersionId, subAnswer.AnswerIntakes)...)
			}
			answersToStore[i].SubAnswers = subAnswers
		}
	}
	return answersToStore
}

func QueueUpJobForErxStatus(erxStatusQueue *common.SQSQueue, prescriptionStatusCheckMessage common.PrescriptionStatusCheckMessage) error {
	jsonData, err := json.Marshal(prescriptionStatusCheckMessage)
	if err != nil {
		return err
	}

	// queue up a job
	return erxStatusQueue.QueueService.SendMessage(erxStatusQueue.QueueUrl, 0, string(jsonData))
}

func createAnswersToStoreForQuestion(role string, roleId, questionId, contextId, layoutVersionId int64, answerIntakes []*AnswerItem) []*common.AnswerIntake {
	answersToStore := make([]*common.AnswerIntake, len(answerIntakes))
	for i, answerIntake := range answerIntakes {
		answersToStore[i] = &common.AnswerIntake{
			RoleId:            encoding.NewObjectId(roleId),
			Role:              role,
			QuestionId:        encoding.NewObjectId(questionId),
			ContextId:         encoding.NewObjectId(contextId),
			LayoutVersionId:   encoding.NewObjectId(layoutVersionId),
			PotentialAnswerId: encoding.NewObjectId(answerIntake.PotentialAnswerId),
			AnswerText:        answerIntake.AnswerText,
		}
	}
	return answersToStore
}

func CreatePhotoUrl(photoId, claimerId int64, claimerType, host string) string {
	params := url.Values{
		"photo_id":     []string{strconv.FormatInt(photoId, 10)},
		"claimer_type": []string{claimerType},
		"claimer_id":   []string{strconv.FormatInt(claimerId, 10)},
	}
	return fmt.Sprintf("https://%s/v1/photo?%s", host, params.Encode())
}

func CreateMediaUrl(mediaId, claimerId int64, claimerType, host string) string {
	params := url.Values{
		"media_id":     []string{strconv.FormatInt(mediaId, 10)},
		"claimer_type": []string{claimerType},
		"claimer_id":   []string{strconv.FormatInt(claimerId, 10)},
	}
	return fmt.Sprintf("https://%s/v1/media?%s", host, params.Encode())
}
