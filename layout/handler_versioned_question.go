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
type versionedQuestionHandler struct {
	dataAPI api.DataAPI
}

// Description of the request to be used for querying this endpoint with GET
type versionedQuestionGETRequest struct {
	ID         int64  `schema:"id"`
	Tag        string `schema:"tag"`
	Version    int64  `schema:"version"`
	LanguageID int64  `schema:"language_id"`
}

// Description of the respone object for a GET request
type versionedQuestionGETResponse struct {
	VersionedQuestion *responses.VersionedQuestion `json:"versioned_question"`
}

// Description of the request to be used for inserting new questions via POST
type versionedQuestionPOSTRequest struct {
	Tag                string `schema:"tag,required"`
	LanguageID         int64  `schema:"language_id,required"`
	Type               string `schema:"type,required"`
	Text               string `schema:"text"`
	ParentQuestionID   int64  `schema:"parent_question_id"`
	Required           bool   `schema:"required"`
	FormattedFieldTags string `schema:"formatted_field_tags"`
	ToAlert            bool   `schema:"to_alert"`
	TextHasTokens      bool   `schema:"text_has_tokens"`
	Subtext            string `schema:"subtext"`
	SummaryText        string `schema:"summary_text"`
	AlertText          string `schema:"alert_text"`
	Version            int64  `schema:"version"`
}

// Description of the request to be used for inserting new questions via POST
type versionedQuestionPOSTResponse struct {
	ID         int64  `json:"id,string"`
	Tag        string `json:"tag"`
	Version    int64  `json:"version,string"`
	LanguageID int64  `json:"language_id,string"`
}

// NewPatientCareTeamsHandler returns a new handler to access the question bank
// Authorization Required: true
// Supported Roles: ADMIN_ROLE
// Supported Method: GET, POST
func NewVersionedQuestionHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.AuthorizationRequired(
				&versionedQuestionHandler{
					dataAPI: dataAPI,
				}), []string{api.ADMIN_ROLE}), []string{"GET", "POST"})
}

// IsAuthorized when given a http.Request object, determines if the caller is an ADMIN user or not
func (h *versionedQuestionHandler) IsAuthorized(r *http.Request) (bool, error) {
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
	default:
		return false, apiservice.NewValidationError("unable to match/parse request data")
	}

	ctxt.RequestCache[apiservice.RequestData] = rd
	return true, nil
}

// Utilizes dataAPI.VersionedQuestion to fetch versioned questions
func (h *versionedQuestionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	switch r.Method {
	case "GET":
		h.serveGET(w, r, ctxt.RequestCache[apiservice.RequestData].(*versionedQuestionGETRequest))
	case "POST":
		h.servePOST(w, r, ctxt.RequestCache[apiservice.RequestData].(*versionedQuestionPOSTRequest))
	}
}

// parseGETRequest parses and verifies that a valid combination of GET parameters were supplied to the API
// Valid combinations inclde
//	ID - Returns a question that maps to a specific ID
//	Tag & Version - Returns a question that maps to a specific tag and version
func (h *versionedQuestionHandler) parseGETRequest(r *http.Request) (*versionedQuestionGETRequest, error) {
	rd := &versionedQuestionGETRequest{}
	if err := apiservice.DecodeRequestData(rd, r); err != nil {
		return nil, apiservice.NewValidationError(err.Error())
	}

	// If none of the critical params are provided then we have an invalid request
	// If we have partially completed sets then we have an invalid request
	// If no language ID is present then we have an invalid request
	if rd.ID == 0 && (rd.Version == 0 || rd.Tag == "" || rd.LanguageID == 0) {
		return nil, apiservice.NewValidationError("insufficent parameters supplied to form complete query")
	}
	return rd, nil
}

func (h *versionedQuestionHandler) serveGET(w http.ResponseWriter, r *http.Request, rd *versionedQuestionGETRequest) {
	if rd.ID != 0 {
		h.serveQuestionIDGET(w, r, rd.ID)
	} else {
		h.serveQuestionTagGET(w, r, rd.Tag, rd.Version, rd.LanguageID)
	}
}

func (h *versionedQuestionHandler) serveQuestionIDGET(w http.ResponseWriter, r *http.Request, ID int64) {
	vq, err := h.dataAPI.VersionedQuestionFromID(ID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	apiservice.WriteJSON(w, versionedQuestionGETResponse{
		VersionedQuestion: responses.NewVersionedQuestionFromDBModel(vq),
	})
}

func (h *versionedQuestionHandler) serveQuestionTagGET(w http.ResponseWriter, r *http.Request, tag string, version, language_id int64) {
	vq, err := h.dataAPI.VersionedQuestion(tag, version, language_id)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	apiservice.WriteJSON(w, versionedQuestionGETResponse{
		VersionedQuestion: responses.NewVersionedQuestionFromDBModel(vq),
	})
}

// parsePOSTRequest parses the requested question to insert into the question bank
func (h *versionedQuestionHandler) parsePOSTRequest(r *http.Request) (*versionedQuestionPOSTRequest, error) {
	rd := &versionedQuestionPOSTRequest{}
	if err := apiservice.DecodeRequestData(rd, r); err != nil {
		return nil, apiservice.NewValidationError(err.Error())
	}

	return rd, nil
}

func (h *versionedQuestionHandler) servePOST(w http.ResponseWriter, r *http.Request, rd *versionedQuestionPOSTRequest) {
	vq := &common.VersionedQuestion{
		AlertText:        newNullString(rd.AlertText, rd.AlertText != ""),
		LanguageID:       rd.LanguageID,
		ParentQuestionID: newNullInt64(rd.ParentQuestionID, rd.ParentQuestionID != 0),
		SubtextText:      newNullString(rd.Subtext, rd.Subtext != ""),
		SummaryText:      newNullString(rd.SummaryText, rd.SummaryText != ""),
		QuestionTag:      rd.Tag,
		QuestionText:     newNullString(rd.Text, rd.Text != ""),
		TextHasTokens:    newNullBool(rd.TextHasTokens),
		ToAlert:          newNullBool(rd.ToAlert),
		QuestionType:     rd.Type,
		Version:          rd.Version,
	}

	id, err := h.dataAPI.VersionQuestion(vq)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// Note: Why waste a read here and look this back up? We want to assert we're giving the user an honest view of the DB state
	vq, err = h.dataAPI.VersionedQuestionFromID(id)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSON(w, versionedQuestionPOSTResponse{
		ID:         vq.ID,
		Tag:        vq.QuestionTag,
		Version:    vq.Version,
		LanguageID: vq.LanguageID,
	})
}
