package apiservice

import (
	"carefront/api"
	"net/http"
	"strconv"
)

type AnswerIntakeHandler struct {
	DataApi   api.DataAPI
	accountId int64
}

type AnswerIntakeErrorResponse struct {
	ErrorString string `json:"error"`
}

type AnswerIntakeResponse struct {
	AnswerId int64 `json:answer_id,string"`
}

func NewAnswerIntakeHandler(dataApi api.DataAPI) *AnswerIntakeHandler {
	return &AnswerIntakeHandler{dataApi, 0}
}

func (a *AnswerIntakeHandler) AccountIdFromAuthToken(accountId int64) {
	a.accountId = accountId
}

func (a *AnswerIntakeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	patientVisitId := r.FormValue("patient_visit_id")
	questionId := r.FormValue("question_id")
	sectionId := r.FormValue("section_id")
	answerId := r.FormValue("potential_answer_id")
	answerText := r.FormValue("answer_text")
	// toUpdate := r.FormValue("to_update")

	if patientVisitId == "" || questionId == "" || sectionId == "" || answerId == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	patientVisitIdInt, err := strconv.ParseInt(patientVisitId, 0, 64)
	questionIdInt, err := strconv.ParseInt(questionId, 0, 64)
	sectionIdInt, err := strconv.ParseInt(sectionId, 0, 64)
	answerIdInt, err := strconv.ParseInt(answerId, 0, 64)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// get the answer type to determine the structure of the answer that is to be stored
	answerType, err := a.DataApi.GetAnswerType(answerIdInt)
	freeTextRequired := false
	// photoRequired := false
	switch answerType {
	case "a_type_free_text":
	case "a_type_single_entry":
		freeTextRequired = true
		// case "a_type_photo_entry_back":
		// case "a_type_photo_entry_chest":
		// case "a_type_photo_entry_face_left":
		// case "a_type_photo_entry_face_middle":
		// case "a_type_photo_entry_face_right":
		// case "a_type_photo_entry_other":
		// 	photoRequired = true
	}

	if freeTextRequired && answerText == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// get layout version id
	layoutVersionId, err := a.DataApi.GetLayoutVersionIdForPatientVisit(patientVisitIdInt)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	patientId, err := a.DataApi.GetPatientIdFromAccountId(a.accountId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var patientInfoIntakeId int64
	if freeTextRequired {
		patientInfoIntakeId, err = a.DataApi.StoreFreeTextAnswerForQuestion(patientId, questionIdInt, answerIdInt, sectionIdInt, patientVisitIdInt, layoutVersionId, answerText)
	} else {
		patientInfoIntakeId, err = a.DataApi.StoreChoiceAnswerForQuestion(patientId, questionIdInt, answerIdInt, sectionIdInt, patientVisitIdInt, layoutVersionId)
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	WriteJSONToHTTPResponseWriter(w, AnswerIntakeResponse{patientInfoIntakeId})
}
