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

// Description of the request to be used for querying this endpoint
type versionedQuestionGETRequest struct {
	LayoutVersion string `schema:"layout_version"`
	LayoutType    string `schema:"layout_type"`
	ID            int64  `schema:"id"`
	Tag           string `schema:"tag"`
	Version       int64  `schema:"version"`
	LanguageID    int64  `schema:"language_id,required"`
}

type versionedQuestionGETResponse struct {
	VersionedQuestions []*responses.VersionedQuestion `json:"versioned_questions"`
}

// NewPatientCareTeamsHandler returns a new handler to access the care teams associated with a given patient.
// Authorization Required: true
// Supported Roles: ADMIN_ROLE
// Supported Method: GET, POST
func NewVersionedQuestionHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.AuthorizationRequired(
				&versionedQuestionHandler{
					dataAPI: dataAPI,
				}), []string{api.ADMIN_ROLE}), []string{"GET"})
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
	default:
		return false, apiservice.NewValidationError("unable to match/parse request data")
	}

	ctxt.RequestCache[apiservice.RequestData] = rd
	return true, nil
}

// parseGETRequest parses and verifies that a valid combination of GET parameters were supplied to the API
// Valid combinations inclde
// 	LayoutVersion & LayoutType - Returns all questions that apply to a layout
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
	if (rd.LayoutVersion == "" && rd.ID == 0 && rd.Tag == "") ||
		(rd.LayoutVersion != "" && rd.LayoutType == "") ||
		(rd.LayoutVersion == "" && rd.LayoutType != "") ||
		(rd.Tag != "" && rd.Version == 0) ||
		(rd.Tag == "" && rd.Version != 0) ||
		(rd.LanguageID == 0) {
		return nil, apiservice.NewValidationError("insufficent parameters supplied to form complete query")
	}
	return rd, nil
}

// Utilizes dataAPI.VersionedQuestion to fetch versioned questions
func (h *versionedQuestionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	switch r.Method {
	case "GET":
		h.serveGET(w, r, ctxt.RequestCache[apiservice.RequestData].(*versionedQuestionGETRequest))
	}
}

func (h *versionedQuestionHandler) serveGET(w http.ResponseWriter, r *http.Request, rd *versionedQuestionGETRequest) {
	if rd.LayoutVersion != "" {
		h.serveLayoutGET(w, r, rd.LayoutType, rd.LayoutVersion)
	} else if rd.ID != 0 {
		h.serveQuestionIDGET(w, r, rd.ID)
	} else {
		h.serveQuestionTagGET(w, r, rd.Tag, rd.Version, rd.LanguageID)
	}
}

func (h *versionedQuestionHandler) serveLayoutGET(w http.ResponseWriter, r *http.Request, layoutType, layoutVersion string) {

}

func (h *versionedQuestionHandler) serveQuestionIDGET(w http.ResponseWriter, r *http.Request, ID int64) {
	vq, err := h.dataAPI.VersionedQuestionFromID(ID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	apiservice.WriteJSON(w, versionedQuestionGETResponse{
		VersionedQuestions: []*responses.VersionedQuestion{h.dbmodelToResponse(vq)},
	})
}

func (h *versionedQuestionHandler) serveQuestionTagGET(w http.ResponseWriter, r *http.Request, tag string, version, language_id int64) {
	vq, err := h.dataAPI.VersionedQuestion(tag, version, language_id)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	apiservice.WriteJSON(w, versionedQuestionGETResponse{
		VersionedQuestions: []*responses.VersionedQuestion{h.dbmodelToResponse(vq)},
	})
}

func (h *versionedQuestionHandler) dbmodelToResponse(dbmodel *common.VersionedQuestion) *responses.VersionedQuestion {
	return &responses.VersionedQuestion{
		AlertText:     dbmodel.AlertText.String,
		ID:            dbmodel.ID,
		LanguageID:    dbmodel.LanguageID,
		ParentID:      dbmodel.ParentQuestionID.Int64,
		Subtext:       dbmodel.SubtextText.String,
		SummaryText:   dbmodel.SummaryText.String,
		Tag:           dbmodel.QuestionTag,
		Text:          dbmodel.QuestionText.String,
		TextHasTokens: dbmodel.TextHasTokens.Bool,
		ToAlert:       dbmodel.ToAlert.Bool,
		Type:          dbmodel.QuestionType,
		Version:       dbmodel.Version,
	}
}
