package apiservice

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/SpruceHealth/schema"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/sqs"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
)

var (
	ErrBadAuthHeader = errors.New("bad authorization header")
	ErrNoAuthHeader  = errors.New("no authorization header")
)

var Testing = false

const (
	DeveloperErrorNoVisitExists                    = 10001
	DeveloperErrorAuthTokenExpired                 = 10002
	DeveloperErrorTreatmentMissingDNTF             = 10003
	DeveloperErrorNoTreatmentPlan                  = 10004
	DeveloperErrorJBCQForbidden                    = 10005
	DeveloperErrorControlledSubstanceRefillRequest = 10006
)

const (
	genericUserErrorMessage   = "Something went wrong on our end. Apologies for the inconvenience and please try again later!"
	authTokenExpiredMessage   = "Authentication expired. Log in to continue."
	StatusUnprocessableEntity = 422
	TimeFormatLayout          = "January 2 at 3:04pm"
)

type GenericJSONResponse struct {
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
	case ErrBadAuthHeader, ErrNoAuthHeader, api.ErrTokenExpired, api.ErrTokenDoesNotExist:
		golog.Context("AuthEvent", AuthEventInvalidToken).Infof(err.Error())
		WriteError(NewAuthTimeoutError(), w, r)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func EnsureTreatmentPlanOrPatientVisitIDPresent(dataAPI api.DataAPI, treatmentPlanID int64, patientVisitID *int64) error {
	if patientVisitID == nil {
		return fmt.Errorf("PatientVisitId should not be nil!")
	}

	if *patientVisitID == 0 && treatmentPlanID == 0 {
		return errors.New("Either patientVisitId or treatmentPlanId should be specified")
	}

	if *patientVisitID == 0 {
		patientVisitIDFromTreatmentPlanID, err := dataAPI.GetPatientVisitIDFromTreatmentPlanID(treatmentPlanID)
		if err != nil {
			return errors.New("Unable to get patient visit id from treatmentPlanId: " + err.Error())
		}
		*patientVisitID = patientVisitIDFromTreatmentPlanID
	}

	return nil
}

var SuccessfulGenericJSONResponse = &GenericJSONResponse{Result: "success"}

type ErrorResponse struct {
	DeveloperError string `json:"developer_error,omitempty"`
	DeveloperCode  int64  `json:"developer_code,string,omitempty"`
	UserError      string `json:"user_error,omitempty"`
}

func WriteJSONSuccess(w http.ResponseWriter) {
	httputil.JSONResponse(w, http.StatusOK, SuccessfulGenericJSONResponse)
}

func DecodeRequestData(requestData interface{}, r *http.Request) error {
	switch r.Header.Get("Content-Type") {
	case "application/json", "application/json; charset=UTF-8", "text/json":
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

func PopulateAnswersToStoreForQuestion(role string, item *QuestionAnswerItem, contextID, roleID, layoutVersionID int64) []*common.AnswerIntake {
	// get a list of top level answers to store for each of the quetions
	answersToStore := createAnswersToStoreForQuestion(role, roleID, item.QuestionID,
		contextID, layoutVersionID, item.AnswerIntakes)

	// go through all the answers of each question intake to identify responses that have responses to subquestions
	// embedded in them, and add that to the list of answers to store in the database
	for i, answerIntake := range item.AnswerIntakes {
		if answerIntake.SubQuestions != nil {
			var subAnswers []*common.AnswerIntake
			for _, subAnswer := range answerIntake.SubQuestions {
				subAnswers = append(subAnswers, createAnswersToStoreForQuestion(role, roleID, subAnswer.QuestionID, contextID, layoutVersionID, subAnswer.AnswerIntakes)...)
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
	jsonDataString := string(jsonData)

	for i := 0; i < numRetries; i++ {
		_, err := queue.QueueService.SendMessage(&sqs.SendMessageInput{
			QueueURL:    &queue.QueueURL,
			MessageBody: &jsonDataString,
		})
		if err != nil {
			golog.Errorf("Unable to queue job: %s. Retrying after %d seconds", err, retryIntervalSeconds)
			time.Sleep(time.Duration(retryIntervalSeconds) * time.Second)
			continue
		}
		return nil
	}

	// queue up a job
	return fmt.Errorf("Unable to enqueue job after retrying %d times", numRetries)
}

func createAnswersToStoreForQuestion(role string, roleID, questionID, contextID, layoutVersionID int64, answerIntakes []*AnswerItem) []*common.AnswerIntake {
	answersToStore := make([]*common.AnswerIntake, len(answerIntakes))
	for i, answerIntake := range answerIntakes {
		answersToStore[i] = &common.AnswerIntake{
			RoleID:            encoding.NewObjectID(roleID),
			Role:              role,
			QuestionID:        encoding.NewObjectID(questionID),
			ContextID:         encoding.NewObjectID(contextID),
			LayoutVersionID:   encoding.NewObjectID(layoutVersionID),
			PotentialAnswerID: encoding.NewObjectID(answerIntake.PotentialAnswerID),
			AnswerText:        answerIntake.AnswerText,
		}
	}
	return answersToStore
}
