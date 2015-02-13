package treatment_plan

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/httputil"
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

func NewRXGuideHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.NoAuthorizationRequired(
			&rxGuideHandler{
				dataAPI: dataAPI,
			}), []string{"GET"})
}

func (h *rxGuideHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		rd, err := h.parseGETRequest(r)
		if err != nil {
			apiservice.WriteValidationError(err.Error(), w, r)
			return
		}
		treatmentGuideResponse(h.dataAPI, rd.GenericName, rd.Route, rd.Form, rd.Dosage, "", nil, nil, w, r)
	}
}

func (h *rxGuideHandler) parseGETRequest(r *http.Request) (*RXGuideGETRequest, error) {
	rd := &RXGuideGETRequest{}
	if err := apiservice.DecodeRequestData(rd, r); err != nil {
		return nil, apiservice.NewValidationError(err.Error())
	}
	return rd, nil
}
