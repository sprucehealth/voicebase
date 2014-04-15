package apiservice

import (
	"carefront/api"
	"carefront/common"
	"carefront/info_intake"
	"carefront/libs/golog"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

var ErrBadAuthToken = errors.New("BadAuthToken")

var Testing = false

const (
	genericUserErrorMessage         = "Something went wrong on our end. Apologies for the inconvenience and please try again later!"
	authTokenExpiredMessage         = "Authentication expired. Log in to continue."
	DEVELOPER_ERROR_NO_VISIT_EXISTS = 10001
	DEVELOPER_AUTH_TOKEN_EXPIRED    = 10002
	HTTP_GET                        = "GET"
	HTTP_POST                       = "POST"
	HTTP_PUT                        = "PUT"
	HTTP_DELETE                     = "DELETE"
	HTTP_UNPROCESSABLE_ENTITY       = 422
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

func ensureTreatmentPlanOrPatientVisitIdPresent(dataApi api.DataAPI, treatmentPlanId int64, patientVisitId *int64) error {
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

func verifyDoctorPatientRelationship(dataApi api.DataAPI, doctor *common.Doctor, patient *common.Patient) error {
	// nothing to verify for an unlinked patient since they dont have a care team
	if patient.IsUnlinked {
		return nil
	}

	careTeam, err := dataApi.GetCareTeamForPatient(patient.PatientId.Int64())
	if err != nil {
		return fmt.Errorf("Unable to get care team based on patient id: %+v", err)
	}

	primaryDoctorId := getPrimaryDoctorIdFromCareTeam(careTeam)
	if doctor.DoctorId.Int64() != primaryDoctorId {
		return fmt.Errorf("Unable to get the patient information by doctor when this doctor is not the primary doctor for patient")
	}
	return nil
}

func GetSignedUrlsForAnswersInQuestion(question *info_intake.Question, photoStorageService api.CloudStorageAPI) {
	// go through each answer to get signed urls
	for _, patientAnswer := range question.Answers {
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

func GetPrimaryDoctorInfoBasedOnPatient(dataApi api.DataAPI, patient *common.Patient, staticBaseContentUrl string) (*common.Doctor, error) {
	careTeam, err := dataApi.GetCareTeamForPatient(patient.PatientId.Int64())
	if err != nil {
		return nil, err
	}

	primaryDoctorId := getPrimaryDoctorIdFromCareTeam(careTeam)
	if primaryDoctorId == 0 {
		return nil, errors.New("Unable to get primary doctor based on patient")
	}

	doctor, err := GetDoctorInfo(dataApi, primaryDoctorId, staticBaseContentUrl)
	return doctor, err
}

func GetDoctorInfo(dataApi api.DataAPI, doctorId int64, staticBaseContentUrl string) (*common.Doctor, error) {

	doctor, err := dataApi.GetDoctorFromId(doctorId)
	if err != nil {
		return nil, err
	}

	doctor.ThumbnailUrl = strings.ToLower(fmt.Sprintf("%sdoctor_photo_%s_%s", staticBaseContentUrl, doctor.FirstName, doctor.LastName))
	return doctor, err
}

func getPrimaryDoctorIdFromCareTeam(careTeam *common.PatientCareProviderGroup) int64 {
	for _, assignment := range careTeam.Assignments {
		if assignment.ProviderRole == api.DOCTOR_ROLE && assignment.Status == api.PRIMARY_DOCTOR_STATUS {
			return assignment.ProviderId
		}
	}
	return 0
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

func WriteDeveloperError(w http.ResponseWriter, httpStatusCode int, errorString string) {
	golog.Logf(2, golog.ERR, errorString)
	developerError := &ErrorResponse{
		DeveloperError: errorString,
		UserError:      genericUserErrorMessage,
	}
	WriteJSONToHTTPResponseWriter(w, httpStatusCode, developerError)
}

func WriteDeveloperErrorWithCode(w http.ResponseWriter, developerStatusCode int64, httpStatusCode int, errorString string) {
	golog.Logf(2, golog.ERR, errorString)
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

func populateAnswersToStoreForQuestion(role string, answerToQuestionItem *AnswerToQuestionItem, contextId, roleId, layoutVersionId int64) []*common.AnswerIntake {
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

func createAnswersToStoreForQuestion(role string, roleId, questionId, contextId, layoutVersionId int64, answerIntakes []*AnswerItem) []*common.AnswerIntake {
	answersToStore := make([]*common.AnswerIntake, len(answerIntakes))
	for i, answerIntake := range answerIntakes {
		answersToStore[i] = &common.AnswerIntake{
			RoleId:            common.NewObjectId(roleId),
			Role:              role,
			QuestionId:        common.NewObjectId(questionId),
			ContextId:         common.NewObjectId(contextId),
			LayoutVersionId:   common.NewObjectId(layoutVersionId),
			PotentialAnswerId: common.NewObjectId(answerIntake.PotentialAnswerId),
			AnswerText:        answerIntake.AnswerText,
		}
	}
	return answersToStore
}
