package admin

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/SpruceHealth/schema"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type layoutTemplateHandler struct {
	dataAPI api.DataAPI
}

type layoutTemplateGETRequest struct {
	PathwayTag string `schema:"pathway_tag,required"`
	SKUType    string `schema:"sku,required"`
	Purpose    string `schema:"purpose,required"`
	Major      int    `schema:"major,required"`
	Minor      int    `schema:"minor,required"`
	Patch      int    `schema:"patch,required"`
}

type layoutTemplateGETResponse map[string]interface{}

func newLayoutTemplateHandler(dataAPI api.DataAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(&layoutTemplateHandler{dataAPI: dataAPI}, httputil.Get)
}

func (h *layoutTemplateHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		requestData, err := h.parseGETRequest(ctx, r)
		if err != nil {
			www.APIBadRequestError(w, r, err.Error())
			return
		}
		h.serveGET(w, r, requestData)
	}
}

func (h *layoutTemplateHandler) parseGETRequest(ctx context.Context, r *http.Request) (*layoutTemplateGETRequest, error) {
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
	layoutTemplate, err := h.dataAPI.LayoutTemplate(req.PathwayTag, req.SKUType, req.Purpose, &common.Version{Major: req.Major, Minor: req.Minor, Patch: req.Patch})
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	var response layoutTemplateGETResponse
	if err := json.Unmarshal(layoutTemplate, &response); err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, response)
}
