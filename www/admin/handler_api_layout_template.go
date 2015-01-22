package admin

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/SpruceHealth/schema"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type layoutTemplateHandler struct {
	dataAPI api.DataAPI
}

type layoutTemplateGETRequest struct {
	PathwayTag string `schema:"pathway_tag,required"`
	Purpose    string `schema:"purpose,required"`
	Major      int64  `schema:"major,required"`
	Minor      int64  `schema:"minor,required"`
	Patch      int64  `schema:"patch,required"`
}

type layoutTemplateGETResponse map[string]interface{}

func NewLayoutTemplateHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			&layoutTemplateHandler{
				dataAPI: dataAPI,
			}, []string{api.ADMIN_ROLE}), []string{"GET"})
}

func (h *layoutTemplateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		requestData, err := h.parseGETRequest(r)
		if err != nil {
			www.APIBadRequestError(w, r, err.Error())
			return
		}
		h.serveGET(w, r, requestData)
	}
}

func (h *layoutTemplateHandler) parseGETRequest(r *http.Request) (*layoutTemplateGETRequest, error) {
	rd := &layoutTemplateGETRequest{}
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	if err := schema.NewDecoder().Decode(rd, r.Form); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}

	return rd, nil
}

func (h *layoutTemplateHandler) serveGET(w http.ResponseWriter, r *http.Request, req *layoutTemplateGETRequest) {
	// get a map of layout versions and info
	layoutTemplate, err := h.dataAPI.LayoutTemplate(req.PathwayTag, req.Purpose, req.Major, req.Minor, req.Patch)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	var response layoutTemplateGETResponse
	if err := json.Unmarshal(layoutTemplate, &response); err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	www.JSONResponse(w, r, http.StatusOK, response)
}
