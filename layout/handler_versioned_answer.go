package layout

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/responses"
)

// The base handler struct to handle requests for interacting with versioned questions
type versionedAnswerHandler struct {
	dataAPI api.DataAPI
}

// Description of the request to be used for inserting new questions via POST
type versionedAnswerPOSTRequest struct {
	Tag         string `schema:"tag,required"`
	QuestionID  int64  `schema:"question_id,required"`
	Type        string `schema:"type,required"`
	LanguageID  int64  `schema:"language_id,required"`
	Ordering    int64  `schema:"ordering,required"`
	Text        string `schema:"text"`
	ToAlert     bool   `schema:"to_alert"`
	SummaryText string `schema:"summary_text"`
	Status      string `schema:"status"`
}

// Description of the response returned from a sucessful POST
type versionedAnswerPOSTResponse struct {
	VersionedQuestion *responses.VersionedQuestion `json:"versioned_question"`
	VersionedAnswerID int64                        `json:"versioned_answer_id"`
}

// Description of the request to be used for removing an answer from an answer set
type versionedAnswerDELETERequest struct {
	Tag        string `schema:"tag,required"`
	QuestionID int64  `schema:"question_id,required"`
	LanguageID int64  `schema:"language_id,required"`
}

// Description of the response from a sucessful DELTE request
type versionedAnswerDELETEResponse struct {
	VersionedQuestion *responses.VersionedQuestion `json:"versioned_question"`
}

// NewPatientCareTeamsHandler returns a new handler to access the answer bank
// Authorization Required: true
// Supported Roles: ADMIN_ROLE
// Supported Method: GET, POST, DELETE
func NewVersionedAnswerHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.AuthorizationRequired(
				&versionedAnswerHandler{
					dataAPI: dataAPI,
				}), []string{api.ADMIN_ROLE}), []string{"POST", "DELETE"})
}

// IsAuthorized when given a http.Request object, determines if the caller is an ADMIN user or not
func (h *versionedAnswerHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)

	if ctxt.Role != api.ADMIN_ROLE {
		return false, nil
	}

	var rd interface{}
	var err error
	switch r.Method {
	case "POST":
		rd, err = h.parsePOSTRequest(r)
		if err != nil {
			return false, err
		}
	case "DELETE":
		rd, err = h.parseDELETERequest(r)
		if err != nil {
			return false, err
		}
	default:
		return false, apiservice.NewValidationError("unable to match/parse request data")
	}

	ctxt.RequestCache[apiservice.RequestData] = rd
	return true, nil
}

// Utilizes dataAPI.VersionedAnswer to fetch versioned questions
func (h *versionedAnswerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	switch r.Method {
	case "POST":
		h.servePOST(w, r, ctxt.RequestCache[apiservice.RequestData].(*versionedAnswerPOSTRequest))
	case "DELETE":
		h.serveDELETE(w, r, ctxt.RequestCache[apiservice.RequestData].(*versionedAnswerDELETERequest))
	}
}

// parsePOSTRequest parses the data needed to perform the record insert
func (h *versionedAnswerHandler) parsePOSTRequest(r *http.Request) (*versionedAnswerPOSTRequest, error) {
	rd := &versionedAnswerPOSTRequest{}
	if err := apiservice.DecodeRequestData(rd, r); err != nil {
		return nil, apiservice.NewValidationError(err.Error())
	}

	return rd, nil
}

func (h *versionedAnswerHandler) servePOST(w http.ResponseWriter, r *http.Request, rd *versionedAnswerPOSTRequest) {
	va := &common.VersionedAnswer{
		AnswerTag:         rd.Tag,
		ToAlert:           newNullBool(rd.ToAlert),
		Ordering:          rd.Ordering,
		QuestionID:        rd.QuestionID,
		LanguageID:        rd.LanguageID,
		AnswerText:        newNullString(rd.Text, rd.Text != ""),
		AnswerSummaryText: newNullString(rd.SummaryText, rd.SummaryText != ""),
		AnswerType:        rd.Type,
	}

	qid, aid, err := h.dataAPI.VersionAnswer(va)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	vq, err := h.dataAPI.VersionedQuestionFromID(qid)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	response := versionedAnswerPOSTResponse{
		VersionedQuestion: responses.NewVersionedQuestionFromDBModel(vq),
		VersionedAnswerID: aid,
	}

	vas, err := answerResponsesForQuestion(h.dataAPI, qid, va.LanguageID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	response.VersionedQuestion.VersionedAnswers = vas

	apiservice.WriteJSON(w, response)
}

// parseDELETERequest parses the data needed to perform the record deletion
func (h *versionedAnswerHandler) parseDELETERequest(r *http.Request) (*versionedAnswerDELETERequest, error) {
	rd := &versionedAnswerDELETERequest{}
	if err := apiservice.DecodeRequestData(rd, r); err != nil {
		return nil, apiservice.NewValidationError(err.Error())
	}

	return rd, nil
}

func (h *versionedAnswerHandler) serveDELETE(w http.ResponseWriter, r *http.Request, rd *versionedAnswerDELETERequest) {
	va := &common.VersionedAnswer{
		AnswerTag:  rd.Tag,
		QuestionID: rd.QuestionID,
		LanguageID: rd.LanguageID,
	}

	id, err := h.dataAPI.DeleteVersionedAnswer(va)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	vq, err := h.dataAPI.VersionedQuestionFromID(id)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	response := &versionedAnswerDELETEResponse{
		VersionedQuestion: responses.NewVersionedQuestionFromDBModel(vq),
	}

	vas, err := answerResponsesForQuestion(h.dataAPI, id, va.LanguageID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	response.VersionedQuestion.VersionedAnswers = vas

	apiservice.WriteJSON(w, response)
}

func answerResponsesForQuestion(dataAPI api.DataAPI, questionID, languageID int64) ([]*responses.VersionedAnswer, error) {
	vas, err := dataAPI.VersionedAnswersForQuestion(questionID, languageID)
	if err != nil {
		return nil, err
	}
	rs := make([]*responses.VersionedAnswer, len(vas))
	for i, va := range vas {
		rs[i] = responses.NewVersionedAnswerFromDBModel(va)
	}
	return rs, nil
}
