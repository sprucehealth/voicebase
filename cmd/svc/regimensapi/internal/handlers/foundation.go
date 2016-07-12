package handlers

import (
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/regimensapi/responses"
	"github.com/sprucehealth/backend/cmd/svc/restapi/apiservice"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/svc/regimens"
	"github.com/sprucehealth/schema"
)

type foundationProvider interface {
	FoundationOf(id string, maxResults int) ([]*regimens.Regimen, error)
}

type foundationHandler struct {
	svc foundationProvider
}

// NewFoundation returns an initialized instance of foundationHandler
func NewFoundation(svc foundationProvider) http.Handler {
	return httputil.SupportedMethods(&foundationHandler{svc: svc}, httputil.Get)
}

func (h *foundationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, ok := mux.Vars(r.Context())["id"]
	if !ok {
		apiservice.WriteResourceNotFoundError("an id must be provided", w, r)
		return
	}

	switch r.Method {
	case httputil.Get:
		rd, err := h.parseGETRequest(r)
		if err != nil {
			apiservice.WriteBadRequestError(err, w, r)
			return
		}
		h.serveGET(w, r, rd, id)
	}
}

func (h *foundationHandler) parseGETRequest(r *http.Request) (*responses.FoundationGETRequest, error) {
	rd := &responses.FoundationGETRequest{}
	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	if err := schema.NewDecoder().Decode(rd, r.Form); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}

	return rd, nil
}

func (h *foundationHandler) serveGET(w http.ResponseWriter, r *http.Request, rd *responses.FoundationGETRequest, id string) {
	foundations, err := h.svc.FoundationOf(id, rd.MaxResults)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	httputil.JSONResponse(w, http.StatusOK, &responses.FoundationGETResponse{FoundationOf: foundations})
}
