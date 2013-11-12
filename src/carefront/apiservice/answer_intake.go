package apiservice

import (
	"carefront/api"
	"encoding/json"
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
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if answerIntakeRequestBody.PatientVisitId == 0 || answerIntakeRequestBody.QuestionId == 0 ||
		answerIntakeRequestBody.SectionId == 0 || answerIntakeRequestBody.AnswerIntakes == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// get layout version id
	layoutVersionId, err := a.DataApi.GetLayoutVersionIdForPatientVisit(answerIntakeRequestBody.PatientVisitId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	patientId, err := a.DataApi.GetPatientIdFromAccountId(a.accountId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	questionType, err := a.DataApi.GetQuestionType(answerIntakeRequestBody.QuestionId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	freeTextRequired := false
	if questionType == "q_type_free_text" || questionType == "q_type_single_entry" {
		freeTextRequired = true
	}

	// only one response allowed for these type of questions
	if questionType == "q_type_single_select" || questionType == "q_type_photo" || questionType == "q_type_free_text" {
		if len(answerIntakeRequestBody.AnswerIntakes) > 1 {
			w.WriteHeader(http.StatusBadRequest)
			WriteJSONToHTTPResponseWriter(w, AnswerIntakeErrorResponse{"You cannot have more than 1 response for this question type"})
			return
		}
	}

	potentialAnswerIds := make([]int64, len(answerIntakeRequestBody.AnswerIntakes))
	answerTexts := make([]string, len(answerIntakeRequestBody.AnswerIntakes))

	for i, answerIntake := range answerIntakeRequestBody.AnswerIntakes {

		if freeTextRequired && answerIntake.AnswerText == "" {
			w.WriteHeader(http.StatusBadRequest)
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
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	} else {
		potentialInfoIntakeIds, err = a.DataApi.StoreChoiceAnswersForQuestion(patientId,
			answerIntakeRequestBody.QuestionId, answerIntakeRequestBody.SectionId,
			answerIntakeRequestBody.PatientVisitId, layoutVersionId, potentialAnswerIds)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	WriteJSONToHTTPResponseWriter(w, AnswerIntakeResponse{potentialInfoIntakeIds})
}
