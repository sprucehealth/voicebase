package apiservice

import (
	"carefront/api"
	"carefront/common"
	"encoding/json"
	"errors"
	"net/http"
)

type AnswerIntakeHandler struct {
	DataApi   api.DataAPI
	accountId int64
}

type AnswerIntakeResponse struct {
	AnswerIds []int64 `json:answer_ids"`
}

// this structure is present only if we are taking in answers to subquestions
// linked to a root question.
// Note that the structure has been created to be flexible enough to have any kind of
// question type as a subquestion; although we won't have subquestions to subquestions
type SubQuestionAnswerIntake struct {
	QuestionId    int64           `json:"question_id"`
	AnswerIntakes []*AnswerIntake `json:"potential_answers,omitempty"`
}

type AnswerIntake struct {
	PotentialAnswerId        int64                      `json:"potential_answer_id"`
	AnswerText               string                     `json:"answer_text"`
	SubQuestionAnswerIntakes []*SubQuestionAnswerIntake `json:"answers,omitempty"`
}

type AnswerIntakeRequestBody struct {
	PatientVisitId int64           `json:"patient_visit_id"`
	QuestionId     int64           `json:"question_id"`
	AnswerIntakes  []*AnswerIntake `json:"potential_answers"`
}

func NewAnswerIntakeHandler(dataApi api.DataAPI) *AnswerIntakeHandler {
	return &AnswerIntakeHandler{dataApi, 0}
}

func (a *AnswerIntakeHandler) AccountIdFromAuthToken(accountId int64) {
	a.accountId = accountId
}

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
		return
	}

	patientId, err := a.DataApi.GetPatientIdFromAccountId(a.accountId)
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

	questionType, err := a.DataApi.GetQuestionType(answerIntakeRequestBody.QuestionId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get the question_type from the question_id provided")
		return
	}

	// only one response allowed for these type of questions
	if questionType == "q_type_single_select" || questionType == "q_type_photo" || questionType == "q_type_free_text" || questionType == "q_type_segmented_control" {
		if len(answerIntakeRequestBody.AnswerIntakes) > 1 {
			WriteDeveloperError(w, http.StatusBadRequest, "You cannot have more than 1 response for this question type")
			return
		}
	}

	// enumerate the answers to store from the top level questions as well as the sub questions
	answersToStore := populateAnswersToStore(answerIntakeRequestBody, patientId, layoutVersionId)

	err = a.DataApi.StoreAnswersForQuestion(answerIntakeRequestBody.QuestionId, patientId, answerIntakeRequestBody.PatientVisitId, layoutVersionId, answersToStore)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to store the multiple choice answer to the question for the patient based on the parameters provided and the internal state of the system: "+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, AnswerIntakeResponse{})
}

func populateAnswersToStore(answerIntakeRequestBody *AnswerIntakeRequestBody, patientId, layoutVersionId int64) (answersToStore []*common.PatientAnswer) {
	// get a list of top level answers to store for each of the quetions
	answersToStore = createAnswersToStore(patientId, answerIntakeRequestBody.QuestionId,
		answerIntakeRequestBody.PatientVisitId, layoutVersionId, answerIntakeRequestBody.AnswerIntakes)

	// go through all the answers of each question intake to identify responses that have responses to subquestions
	// embedded in them, and add that to the list of answers to store in the database
	for i, answerIntake := range answerIntakeRequestBody.AnswerIntakes {
		if answerIntake.SubQuestionAnswerIntakes != nil {
			subAnswers := make([]*common.PatientAnswer, 0)
			for _, subAnswer := range answerIntake.SubQuestionAnswerIntakes {
				subAnswers = append(subAnswers, createAnswersToStore(patientId, subAnswer.QuestionId, answerIntakeRequestBody.PatientVisitId, layoutVersionId, subAnswer.AnswerIntakes)...)
			}
			answersToStore[i].SubAnswers = subAnswers
		}
	}
	return answersToStore
}

func createAnswersToStore(patientId, questionId, patientVisitId, layoutVersionId int64, answerIntakes []*AnswerIntake) []*common.PatientAnswer {
	answersToStore := make([]*common.PatientAnswer, 0)
	for _, answerIntake := range answerIntakes {
		answerToStore := new(common.PatientAnswer)
		answerToStore.PatientId = patientId
		answerToStore.QuestionId = questionId
		answerToStore.PatientVisitId = patientVisitId
		answerToStore.LayoutVersionId = layoutVersionId
		answerToStore.PotentialAnswerId = answerIntake.PotentialAnswerId
		answerToStore.AnswerText = answerIntake.AnswerText
		answersToStore = append(answersToStore, answerToStore)
	}
	return answersToStore
}

func validateRequestBody(answerIntakeRequestBody *AnswerIntakeRequestBody, w http.ResponseWriter) error {
	if answerIntakeRequestBody.PatientVisitId == 0 {
		WriteDeveloperError(w, http.StatusBadRequest, "patient_visit_id missing")
		return errors.New("")
	}

	if answerIntakeRequestBody.QuestionId == 0 {
		WriteDeveloperError(w, http.StatusBadRequest, "question_id missing")
		return errors.New("")
	}

	if answerIntakeRequestBody.AnswerIntakes == nil {
		WriteDeveloperError(w, http.StatusBadRequest, "potential_answers missing")
		return errors.New("")
	}
	return nil
}
