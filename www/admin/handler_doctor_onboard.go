package admin

import (
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/www"
)

type doctorOnboardHandler struct {
	router  *mux.Router
	dataAPI api.DataAPI
}

func NewDoctorOnboardHandler(router *mux.Router, dataAPI api.DataAPI) http.Handler {
	return www.SupportedMethodsHandler(&doctorOnboardHandler{
		router:  router,
		dataAPI: dataAPI,
	}, []string{"GET"})
}

func (h *doctorOnboardHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	doctorID, err := strconv.ParseInt(r.FormValue("doctor_id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	doctor, err := h.dataAPI.GetDoctorFromId(doctorID)
	if err == api.NoRowsError {
		http.NotFound(w, r)
	} else if err != nil {
		www.InternalServerError(w, r, err)
		return
	}

	attributes, err := h.dataAPI.DoctorAttributes(doctorID, nil)
	if err != nil {
		www.InternalServerError(w, r, err)
		return
	}

	www.TemplateResponse(w, http.StatusOK, drOnboardTemplate, &www.BaseTemplateContext{
		Title: "Doctor Onboarding",
		SubContext: &drOnboardTemplateContext{
			Doctor:     doctor,
			Attributes: attributes,
		},
	})
}
