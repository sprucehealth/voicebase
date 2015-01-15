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

// Description of the request to be used for querying this endpoint
type versionedAnswerGETRequest struct {
	ID         int64  `schema:"id"`
	LanguageID int64  `schema:"language_id"`
	QuestionID int64  `schema:"question_id"`
	Tag        string `schema:"tag"`
}

// Description of the response given to the
type versionedAnswerGETResponse struct {
	VersionedAnswer *responses.VersionedAnswer `json:"versioned_answer"`
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
	ID         int64  `json:"id,string"`
	QuestionID int64  `json:"question_id,string"`
	Tag        string `json:"tag"`
	LanguageID int64  `json:"language_id,string"`
}

// Description of the request to be used for removing an answer from an answer set
type versionedAnswerDELETERequest struct {
	Tag        string `schema:"tag,required"`
	QuestionID int64  `schema:"question_id,required"`
	LanguageID int64  `schema:"language_id,required"`
}

// Description of the response from a sucessful DELTE request
type versionedAnswerDELETEResponse struct {
	QuestionID int64 `json:"question_id,string"`
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
				}), []string{api.ADMIN_ROLE}), []string{"GET", "POST", "DELETE"})
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
	case "GET":
		rd, err = h.parseGETRequest(r)
		if err != nil {
			return false, err
		}
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
	case "GET":
		h.serveGET(w, r, ctxt.RequestCache[apiservice.RequestData].(*versionedAnswerGETRequest))
	case "POST":
		h.servePOST(w, r, ctxt.RequestCache[apiservice.RequestData].(*versionedAnswerPOSTRequest))
	case "DELETE":
		h.serveDELETE(w, r, ctxt.RequestCache[apiservice.RequestData].(*versionedAnswerDELETERequest))
	}
}

// parseGETRequest parses and verifies that a valid combination of GET parameters were supplied to the API
// Valid combinations inclde
//  ID - Returns a question that maps to a specific ID
//  Tag & Version - Returns a question that maps to a specific tag and version
func (h *versionedAnswerHandler) parseGETRequest(r *http.Request) (*versionedAnswerGETRequest, error) {
	rd := &versionedAnswerGETRequest{}
	if err := apiservice.DecodeRequestData(rd, r); err != nil {
		return nil, apiservice.NewValidationError(err.Error())
	}

	// If none of the critical params are provided then we have an invalid request
	// If we have partially completed sets then we have an invalid request
	// If no language ID is present then we have an invalid request
	if rd.ID == 0 && (rd.Tag == "" || rd.QuestionID == 0 || rd.LanguageID == 0) {
		return nil, apiservice.NewValidationError("insufficent parameters supplied to form complete query")
	}
	return rd, nil
}

func (h *versionedAnswerHandler) serveGET(w http.ResponseWriter, r *http.Request, rd *versionedAnswerGETRequest) {
	if rd.ID != 0 {
		h.serveAnswerIDGET(w, r, rd.ID)
	} else {
		h.serveAnswerTagGET(w, r, rd.Tag, rd.QuestionID, rd.LanguageID)
	}
}

func (h *versionedAnswerHandler) serveAnswerIDGET(w http.ResponseWriter, r *http.Request, ID int64) {
	vq, err := h.dataAPI.VersionedAnswerFromID(ID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	apiservice.WriteJSON(w, versionedAnswerGETResponse{
		VersionedAnswer: responses.NewVersionedAnswerFromDBModel(vq),
	})
}

func (h *versionedAnswerHandler) serveAnswerTagGET(w http.ResponseWriter, r *http.Request, tag string, questionID, languageID int64) {
	vq, err := h.dataAPI.VersionedAnswer(tag, questionID, languageID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	apiservice.WriteJSON(w, versionedAnswerGETResponse{
		VersionedAnswer: responses.NewVersionedAnswerFromDBModel(vq),
	})
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

	id, err := h.dataAPI.VersionAnswer(va)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// Note: Why waste a read here and look this back up? We want to assert we're giving the user an honest view of the DB state
	va, err = h.dataAPI.VersionedAnswerFromID(id)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSON(w, versionedAnswerPOSTResponse{
		ID:         va.ID,
		QuestionID: va.QuestionID,
		Tag:        va.AnswerTag,
		LanguageID: va.LanguageID,
	})
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

	apiservice.WriteJSON(w, versionedAnswerDELETEResponse{
		QuestionID: id,
	})
}
