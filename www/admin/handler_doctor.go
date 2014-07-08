package admin

import (
	"html/template"
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/www"
)

type doctorHandler struct {
	router  *mux.Router
	dataAPI api.DataAPI
}

func NewDoctorHandler(router *mux.Router, dataAPI api.DataAPI) http.Handler {
	return www.SupportedMethodsHandler(&doctorHandler{
		router:  router,
		dataAPI: dataAPI,
	}, []string{"GET"})
}

func (h *doctorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	doctorID, err := strconv.ParseInt(vars["id"], 10, 64)
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

	www.TemplateResponse(w, http.StatusOK, doctorTemplate, &www.BaseTemplateContext{
		Title: template.HTML("Dr. " + template.HTMLEscapeString(doctor.FirstName) + " " + template.HTMLEscapeString(doctor.LastName)),
		SubContext: &doctorTemplateContext{
			Doctor:     doctor,
			Attributes: attributes,
		},
	})
}
