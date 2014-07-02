package dronboard

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/third_party/github.com/SpruceHealth/schema"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/www"
)

var (
	licenseStatuses = []string{
		"Active",
		"Inactive",
		"Temporary",
		"Pending",
	}
)

type credentialsHandler struct {
	router  *mux.Router
	dataAPI api.DataAPI
}

type stateLicense struct {
	State   string
	License string
	Status  string
}

type credentialsRequest struct {
	AmericanBoardCertified bool
	SpecialtyBoard         string
	RecentCertDate         string
	StateLicenses          []*stateLicense
}

func (r *credentialsRequest) Validate() map[string]string {
	errors := map[string]string{}
	if r.AmericanBoardCertified == true {
		if r.SpecialtyBoard == "" {
			errors["SpecialtyBoard"] = "Special board is required"
		}
		if r.RecentCertDate == "" {
			errors["RecentCertDate"] = "Recent certification date is required"
		}
	}
	if len(r.StateLicenses) == 0 {
		errors["StateLicenses"] = "At least one state license is required"
	}
	return errors
}

func NewCredentialsHandler(router *mux.Router, dataAPI api.DataAPI) http.Handler {
	return www.SupportedMethodsFilter(&credentialsHandler{
		router:  router,
		dataAPI: dataAPI,
	}, []string{"GET", "POST"})
}

func (h *credentialsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	req := &credentialsRequest{}
	var errors map[string]string

	if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			www.InternalServerError(w, r, err)
			return
		}

		if err := schema.NewDecoder().Decode(req, r.PostForm); err != nil {
			www.InternalServerError(w, r, err)
			return
		}

		errors = req.Validate()
		if len(errors) == 0 {
			// TODO
		}
	}

	// Add an empty entry so that it renders
	if len(req.StateLicenses) == 0 {
		req.StateLicenses = append(req.StateLicenses, &stateLicense{})
	}

	states, err := h.dataAPI.ListStates()
	if err != nil {
		www.InternalServerError(w, r, err)
	}
	www.TemplateResponse(w, http.StatusOK, credsTemplate, &www.BaseTemplateContext{
		Title: "Doctor Registration | Credentials | Spruce",
		SubContext: &credsTemplateContext{
			Form:            req,
			FormErrors:      errors,
			States:          states,
			LicenseStatuses: licenseStatuses,
		},
	})
}
