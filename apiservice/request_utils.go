package apiservice

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/golog"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/SpruceHealth/schema"
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

func HandleAuthError(err error, w http.ResponseWriter, r *http.Request) {
	switch err {
	case ErrBadAuthHeader, ErrNoAuthHeader, api.TokenExpired, api.TokenDoesNotExist:
		golog.Context("AuthEvent", AuthEventInvalidToken).Infof(err.Error())
		WriteError(NewAuthTimeoutError(), w, r)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func EnsureTreatmentPlanOrPatientVisitIdPresent(dataAPI api.DataAPI, treatmentPlanID int64, patientVisitID *int64) error {
	if patientVisitID == nil {
		return fmt.Errorf("PatientVisitId should not be nil!")
	}

	if *patientVisitID == 0 && treatmentPlanID == 0 {
		return errors.New("Either patientVisitId or treatmentPlanId should be specified")
	}

	if *patientVisitID == 0 {
		patientVisitIdFromTreatmentPlanId, err := dataAPI.GetPatientVisitIDFromTreatmentPlanID(treatmentPlanID)
		if err != nil {
			return errors.New("Unable to get patient visit id from treatmentPlanId: " + err.Error())
		}
		*patientVisitID = patientVisitIdFromTreatmentPlanId
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
	golog.LogDepthf(1, golog.ERR, errorResponse.DeveloperError)
	WriteJSONToHTTPResponseWriter(w, httpStatusCode, &errorResponse)
}

func WriteDeveloperError(w http.ResponseWriter, httpStatusCode int, errorString string) {
	golog.LogDepthf(1, golog.ERR, errorString)
	developerError := &ErrorResponse{
		DeveloperError: errorString,
		UserError:      genericUserErrorMessage,
	}
	WriteJSONToHTTPResponseWriter(w, httpStatusCode, developerError)
}

func WriteDeveloperErrorWithCode(w http.ResponseWriter, developerStatusCode int64, httpStatusCode int, errorString string) {
	golog.LogDepthf(1, golog.ERR, errorString)
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

func PopulateAnswersToStoreForQuestion(role string, item *QuestionAnswerItem, contextId, roleID, layoutVersionID int64) []*common.AnswerIntake {
	// get a list of top level answers to store for each of the quetions
	answersToStore := createAnswersToStoreForQuestion(role, roleID, item.QuestionID,
		contextId, layoutVersionID, item.AnswerIntakes)

	// go through all the answers of each question intake to identify responses that have responses to subquestions
	// embedded in them, and add that to the list of answers to store in the database
	for i, answerIntake := range item.AnswerIntakes {
		if answerIntake.SubQuestions != nil {
			subAnswers := make([]*common.AnswerIntake, 0)
			for _, subAnswer := range answerIntake.SubQuestions {
				subAnswers = append(subAnswers, createAnswersToStoreForQuestion(role, roleID, subAnswer.QuestionID, contextId, layoutVersionID, subAnswer.AnswerIntakes)...)
			}
			answersToStore[i].SubAnswers = subAnswers
		}
	}
	return answersToStore
}

func QueueUpJob(queue *common.SQSQueue, msg interface{}) error {
	retryIntervalSeconds := 5
	numRetries := 3
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	for i := 0; i < numRetries; i++ {
		if err := queue.QueueService.SendMessage(queue.QueueURL, 0, string(jsonData)); err != nil {
			golog.Errorf("Unable to queue job: %s. Retrying after %d seconds", err, retryIntervalSeconds)
			time.Sleep(time.Duration(retryIntervalSeconds) * time.Second)
			continue
		}
		return nil
	}

	// queue up a job
	return fmt.Errorf("Unable to enqueue job after retrying %d times", numRetries)
}

func createAnswersToStoreForQuestion(role string, roleID, questionID, contextId, layoutVersionID int64, answerIntakes []*AnswerItem) []*common.AnswerIntake {
	answersToStore := make([]*common.AnswerIntake, len(answerIntakes))
	for i, answerIntake := range answerIntakes {
		answersToStore[i] = &common.AnswerIntake{
			RoleID:            encoding.NewObjectID(roleID),
			Role:              role,
			QuestionID:        encoding.NewObjectID(questionID),
			ContextId:         encoding.NewObjectID(contextId),
			LayoutVersionID:   encoding.NewObjectID(layoutVersionID),
			PotentialAnswerID: encoding.NewObjectID(answerIntake.PotentialAnswerID),
			AnswerText:        answerIntake.AnswerText,
		}
	}
	return answersToStore
}
