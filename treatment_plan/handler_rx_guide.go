package treatment_plan

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/httputil"
	"golang.org/x/net/context"
)

type RXGuideGETRequest struct {
	GenericName string `schema:"generic_name,required"`
	Route       string `schema:"route,required"`
	Form        string `schema:"form"`
	Dosage      string `schema:"dosage"`
}

type rxGuideHandler struct {
	dataAPI api.DataAPI
}

func NewRXGuideHandler(dataAPI api.DataAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.NoAuthorizationRequired(
			&rxGuideHandler{
				dataAPI: dataAPI,
			}), httputil.Get)
}

func (h *rxGuideHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		rd, err := h.parseGETRequest(ctx, r)
		if err != nil {
			apiservice.WriteValidationError(ctx, err.Error(), w, r)
			return
		}
		treatmentGuideResponse(ctx, h.dataAPI, rd.GenericName, rd.Route, rd.Form, rd.Dosage, "", nil, nil, w, r)
	}
}

func (h *rxGuideHandler) parseGETRequest(ctx context.Context, r *http.Request) (*RXGuideGETRequest, error) {
	rd := &RXGuideGETRequest{}
	if err := apiservice.DecodeRequestData(rd, r); err != nil {
		return nil, apiservice.NewValidationError(err.Error())
	}
	return rd, nil
}
