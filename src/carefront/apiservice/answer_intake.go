package apiservice

import (
	"carefront/api"
	"fmt"
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

	if patientVisitId == "" || questionId == "" || sectionId == "" || answerId == "" ||
		answerText == "" {
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

	// get layout version id
	layoutVersionId, err := a.DataApi.GetLayoutVersionIdForPatientVisit(patientVisitIdInt)
	if err != nil {
		fmt.Println("unable to get layout version id from database")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	patientId, err := a.DataApi.GetPatientIdFromAccountId(a.accountId)
	if err != nil {
		fmt.Println("unable to get patient id from account id")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	patientInfoIntakeId, err := a.DataApi.StorePatientAnswerForQuestion(patientId, questionIdInt, answerIdInt, sectionIdInt, patientVisitIdInt, layoutVersionId, answerText)
	if err != nil {
		fmt.Println("unable to store response to question in database", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	WriteJSONToHTTPResponseWriter(w, AnswerIntakeResponse{patientInfoIntakeId})
}
