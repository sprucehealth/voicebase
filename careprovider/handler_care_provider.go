package careprovider

import (
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/responses"
	"golang.org/x/net/context"
)

type careProviderGETRequest struct {
	ProviderID int64 `schema:"provider_id,required"`
}

type careProviderHandler struct {
	dataAPI   api.DataAPI
	apiDomain string
}

func NewCareProviderHandler(dataAPI api.DataAPI, apiDomain string) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.NoAuthorizationRequired(
			&careProviderHandler{
				dataAPI:   dataAPI,
				apiDomain: apiDomain,
			}), httputil.Get)
}

func (h *careProviderHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		requestData, err := h.parseGETRequest(ctx, r)
		if err != nil {
			apiservice.WriteValidationError(ctx, err.Error(), w, r)
			return
		}
		h.serveGET(ctx, w, r, requestData)
	}
}

func (h *careProviderHandler) parseGETRequest(ctx context.Context, r *http.Request) (*careProviderGETRequest, error) {
	rd := &careProviderGETRequest{}
	if err := apiservice.DecodeRequestData(rd, r); err != nil {
		return nil, apiservice.NewValidationError(err.Error())
	}
	return rd, nil
}

func (h *careProviderHandler) serveGET(ctx context.Context, w http.ResponseWriter, r *http.Request, rd *careProviderGETRequest) {
	careProvider, err := h.dataAPI.Doctor(rd.ProviderID, false)
	if api.IsErrNotFound(err) {
		apiservice.WriteResourceNotFoundError(ctx, fmt.Sprintf("No care provider exists for ID %d", rd.ProviderID), w, r)
		return
	} else if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}
	response := responses.NewCareProviderFromDoctorDBModel(careProvider, h.apiDomain)
	httputil.JSONResponse(w, http.StatusOK, response)
}
