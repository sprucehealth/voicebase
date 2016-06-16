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
	"golang.org/x/net/context"
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
func NewRXGuide(dal rxGuideDAL) httputil.ContextHandler {
	return httputil.SupportedMethods(&rxGuideHandler{dal: dal}, httputil.Post, httputil.Get)
}

func (h *rxGuideHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case httputil.Get:
		drugName, ok := mux.Vars(ctx)["drug_name"]
		if !ok {
			apiservice.WriteResourceNotFoundError(ctx, "a drug name must be provided", w, r)
			return
		}
		h.serveGET(ctx, w, r, drugName)
	// TODO: Figure out a clever way to protect this endpoint from non internal use
	case httputil.Post:
		rxGuide, err := h.parsePOSTRequest(ctx, r)
		if err != nil {
			apiservice.WriteBadRequestError(ctx, err, w, r)
			return
		}
		h.servePOST(ctx, w, r, rxGuide)
	}
}

func (h *rxGuideHandler) serveGET(ctx context.Context, w http.ResponseWriter, r *http.Request, drugName string) {
	guide, err := h.dal.RXGuide(drugName)
	if err == rxguide.ErrNoGuidesFound {
		apiservice.WriteResourceNotFoundError(ctx, err.Error(), w, r)
		return
	} else if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}
	// Cache RX guides forever since they should be static
	httputil.FarFutureCacheHeaders(w.Header(), time.Time{})
	httputil.JSONResponse(w, http.StatusOK, &rxGuideHandlerGETResponse{RXGuide: guide})
}

func (h *rxGuideHandler) parsePOSTRequest(ctx context.Context, r *http.Request) (*responses.RXGuidePOSTRequest, error) {
	rd := &responses.RXGuidePOSTRequest{}
	if err := json.NewDecoder(r.Body).Decode(rd); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}

	if rd.RXGuide == nil {
		return nil, fmt.Errorf("rx_guide required")
	}
	return rd, nil
}

func (h *rxGuideHandler) servePOST(ctx context.Context, w http.ResponseWriter, r *http.Request, rd *responses.RXGuidePOSTRequest) {
	if err := h.dal.PutRXGuide(rd.RXGuide); err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}
	apiservice.WriteJSONSuccess(w)
}
