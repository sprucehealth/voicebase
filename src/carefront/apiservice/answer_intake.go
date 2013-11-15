package apiservice

import (
	"carefront/api"
	"encoding/json"
	"errors"
	"net/http"
)

type AnswerIntakeHandler struct {
	DataApi   api.DataAPI
	accountId int64
}

type AnswerIntakeErrorResponse struct {
	ErrorString string `json:"error"`
}

type AnswerIntakeResponse struct {
	AnswerIds []int64 `json:answer_ids"`
}

type AnswerIntake struct {
	PotentialAnswerId int64  `json:"potential_answer_id"`
	AnswerText        string `json:"answer_text"`
}

type AnswerIntakeRequestBody struct {
	PatientVisitId int64           `json:"patient_visit_id"`
	QuestionId     int64           `json:"question_id"`
	SectionId      int64           `json:"section_id"`
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

	// get layout version id
	layoutVersionId, err := a.DataApi.GetLayoutVersionIdForPatientVisit(answerIntakeRequestBody.PatientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get the layout version to use for the client layout based on the patient_visit_id")
		return
	}

	patientId, err := a.DataApi.GetPatientIdFromAccountId(a.accountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patientId from the auth token provided")
		return
	}

	questionType, err := a.DataApi.GetQuestionType(answerIntakeRequestBody.QuestionId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get the question_type from the question_id provided")
		return
	}

	freeTextRequired := false
	if questionType == "q_type_free_text" || questionType == "q_type_single_entry" {
		freeTextRequired = true
	}

	// only one response allowed for these type of questions
	if questionType == "q_type_single_select" || questionType == "q_type_photo" || questionType == "q_type_free_text" {
		if len(answerIntakeRequestBody.AnswerIntakes) > 1 {
			WriteDeveloperError(w, http.StatusBadRequest, "You cannot have more than 1 response for this question type")
			return
		}
	}

	potentialAnswerIds := make([]int64, len(answerIntakeRequestBody.AnswerIntakes))
	answerTexts := make([]string, len(answerIntakeRequestBody.AnswerIntakes))

	for i, answerIntake := range answerIntakeRequestBody.AnswerIntakes {

		if freeTextRequired && answerIntake.AnswerText == "" {
			WriteDeveloperError(w, http.StatusBadRequest, "The answer specified is a free text answer, but no answer_text has been specified")
			return
		}
		potentialAnswerIds[i] = answerIntake.PotentialAnswerId
		if freeTextRequired {
			answerTexts[i] = answerIntake.AnswerText
		}
	}

	var potentialInfoIntakeIds []int64
	if freeTextRequired {
		potentialInfoIntakeIds, err = a.DataApi.StoreFreeTextAnswersForQuestion(patientId,
			answerIntakeRequestBody.QuestionId, answerIntakeRequestBody.SectionId,
			answerIntakeRequestBody.PatientVisitId, layoutVersionId, potentialAnswerIds,
			answerTexts)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to store the free text answer to the question based on the parameters provided and the internal state of the system")
			return
		}
	} else {
		potentialInfoIntakeIds, err = a.DataApi.StoreChoiceAnswersForQuestion(patientId,
			answerIntakeRequestBody.QuestionId, answerIntakeRequestBody.SectionId,
			answerIntakeRequestBody.PatientVisitId, layoutVersionId, potentialAnswerIds)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to store the multiple choice answer to the question for the patient based on the parameters provided and the internal state of the system")
			return
		}
	}

	WriteJSONToHTTPResponseWriter(w, AnswerIntakeResponse{potentialInfoIntakeIds})
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

	if answerIntakeRequestBody.SectionId == 0 {
		WriteDeveloperError(w, http.StatusBadRequest, "section_id missing")
		return errors.New("")
	}

	if answerIntakeRequestBody.AnswerIntakes == nil {
		WriteDeveloperError(w, http.StatusBadRequest, "potential_answers missing")
		return errors.New("")
	}
	return nil
}
