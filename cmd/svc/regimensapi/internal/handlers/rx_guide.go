package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/regimensapi/internal/rxguide"
	"github.com/sprucehealth/backend/cmd/svc/regimensapi/responses"
	"github.com/sprucehealth/backend/cmd/svc/restapi/apiservice"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
)

type rxGuideDAL interface {
	RXGuide(drugName string) (*responses.RXGuide, error)
	PutRXGuide(r *responses.RXGuide) error
}

type rxGuideHandler struct {
	dal rxGuideDAL
}

type rxGuideHandlerGETResponse struct {
	RXGuide *responses.RXGuide `json:"rx_guide"`
}

// NewRXGuide returns an initialized instance of rxGuideHandler
func NewRXGuide(dal rxGuideDAL) http.Handler {
	return httputil.SupportedMethods(&rxGuideHandler{dal: dal}, httputil.Post, httputil.Get)
}

func (h *rxGuideHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case httputil.Get:
		drugName, ok := mux.Vars(r.Context())["drug_name"]
		if !ok {
			apiservice.WriteResourceNotFoundError("a drug name must be provided", w, r)
			return
		}
		h.serveGET(w, r, drugName)
	// TODO: Figure out a clever way to protect this endpoint from non internal use
	case httputil.Post:
		rxGuide, err := h.parsePOSTRequest(r)
		if err != nil {
			apiservice.WriteBadRequestError(err, w, r)
			return
		}
		h.servePOST(w, r, rxGuide)
	}
}

func (h *rxGuideHandler) serveGET(w http.ResponseWriter, r *http.Request, drugName string) {
	guide, err := h.dal.RXGuide(drugName)
	if err == rxguide.ErrNoGuidesFound {
		apiservice.WriteResourceNotFoundError(err.Error(), w, r)
		return
	} else if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	// Cache RX guides forever since they should be static
	httputil.FarFutureCacheHeaders(w.Header(), time.Time{})
	httputil.JSONResponse(w, http.StatusOK, &rxGuideHandlerGETResponse{RXGuide: guide})
}

func (h *rxGuideHandler) parsePOSTRequest(r *http.Request) (*responses.RXGuidePOSTRequest, error) {
	rd := &responses.RXGuidePOSTRequest{}
	if err := json.NewDecoder(r.Body).Decode(rd); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}

	if rd.RXGuide == nil {
		return nil, fmt.Errorf("rx_guide required")
	}
	return rd, nil
}

func (h *rxGuideHandler) servePOST(w http.ResponseWriter, r *http.Request, rd *responses.RXGuidePOSTRequest) {
	if err := h.dal.PutRXGuide(rd.RXGuide); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	apiservice.WriteJSONSuccess(w)
}
