package apiservice

import (
	"carefront/api"
	"carefront/common"
	"carefront/libs/golog"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type AnswerIntakeHandler struct {
	DataApi api.DataAPI
}

func NewAnswerIntakeHandler(dataApi api.DataAPI) *AnswerIntakeHandler {
	return &AnswerIntakeHandler{dataApi}
}

const (
	// Error we get from mysql is: "Error 1213: Deadlock found when trying to get lock; try restarting transaction"
	mysqlDeadlockError    = "Error 1213"
	waitTimeBeforeTxRetry = 100
)

func (a *AnswerIntakeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	jsonDecoder := json.NewDecoder(r.Body)
	answerIntakeRequestBody := &AnswerIntakeRequestBody{}

	err := jsonDecoder.Decode(answerIntakeRequestBody)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	}

	err = validateRequestBody(answerIntakeRequestBody, w)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Bad request parameters for answer intake: "+err.Error())
		return
	}

	patientId, err := a.DataApi.GetPatientIdFromAccountId(GetContext(r).AccountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patientId from the auth token provided")
		return
	}

	patientIdFromPatientVisitId, err := a.DataApi.GetPatientIdFromPatientVisitId(answerIntakeRequestBody.PatientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient_id from patient_visit_id: "+err.Error())
		return
	}

	if patientIdFromPatientVisitId != patientId {
		WriteDeveloperError(w, http.StatusBadRequest, "Patient Id from auth token does not match patient id from the patient visit entry")
		return
	}

	// get layout version id
	layoutVersionId, err := a.DataApi.GetLayoutVersionIdForPatientVisit(answerIntakeRequestBody.PatientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get the layout version to use for the client layout based on the patient_visit_id")
		return
	}

	answersToStorePerQuestion := make(map[int64][]*common.AnswerIntake)
	for _, questionItem := range answerIntakeRequestBody.Questions {

		questionType, err := a.DataApi.GetQuestionType(questionItem.QuestionId)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get the question_type from the question_id provided")
			return
		}

		// only one response allowed for these type of questions
		if questionType == "q_type_single_select" || questionType == "q_type_photo" || questionType == "q_type_free_text" || questionType == "q_type_segmented_control" {
			if len(questionItem.AnswerIntakes) > 1 {
				WriteDeveloperError(w, http.StatusBadRequest, "You cannot have more than 1 response for this question type")
				return
			}
		}

		// enumerate the answers to store from the top level questions as well as the sub questions
		answersToStorePerQuestion[questionItem.QuestionId] = populateAnswersToStoreForQuestion(api.PATIENT_ROLE, questionItem, answerIntakeRequestBody.PatientVisitId, patientId, layoutVersionId)
	}

	err = a.DataApi.StoreAnswersForQuestion(api.PATIENT_ROLE, patientId, answerIntakeRequestBody.PatientVisitId, layoutVersionId, answersToStorePerQuestion)
	if err != nil {
		if strings.Contains(err.Error(), mysqlDeadlockError) {
			golog.Warningf("MYSQL Deadlock found when trying to get lock. Retrying transaction after waiting for %d milliseconds...", waitTimeBeforeTxRetry)
			time.Sleep(waitTimeBeforeTxRetry * time.Millisecond)
			err = a.DataApi.StoreAnswersForQuestion(api.PATIENT_ROLE, patientId, answerIntakeRequestBody.PatientVisitId, layoutVersionId, answersToStorePerQuestion)
			if err != nil {
				WriteDeveloperError(w, http.StatusInternalServerError, "Second try: Unable to store the multiple choice answer to the question for the patient based on the parameters provided and the internal state of the system: "+err.Error())
				return
			}
		} else {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to store the multiple choice answer to the question for the patient based on the parameters provided and the internal state of the system: "+err.Error())
			return
		}
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, AnswerIntakeResponse{Result: "success"})
}
