package dronboard

import (
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/third_party/github.com/gorilla/context"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
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

type credentialsForm struct {
	AmericanBoardCertified bool
	SpecialtyBoard         string
	RecentCertDate         string
	ContinuedEducation     bool
	CreditHours            string
	RiskManagementCourse   bool
	NPI                    string
	SSN                    string
	StateLicenses          []*stateLicense
}

func (r *credentialsForm) Validate() map[string]string {
	errors := map[string]string{}
	if r.AmericanBoardCertified == true {
		if r.SpecialtyBoard == "" {
			errors["SpecialtyBoard"] = "Special board is required"
		}
		if r.RecentCertDate == "" {
			errors["RecentCertDate"] = "Recent certification date is required"
		}
	}
	if r.ContinuedEducation {
		if r.CreditHours == "" {
			errors["CreditHours"] = "Credit hours are required"
		}
	}
	if r.NPI == "" {
		errors["NPI"] = "NPI is required"
	}
	n := 0
	for i, l := range r.StateLicenses {
		if l.State != "" {
			// if l.License == "" {
			// 	errors[fmt.Sprintf("StateLicenses.%d.License", i)] = "License is required"
			// }
			// if l.Status == "" {
			// 	errors[fmt.Sprintf("StateLicenses.%d.Status", i)] = "Status is required"
			// }
			if l.License == "" || l.Status == "" {
				errors[fmt.Sprintf("StateLicenses.%d", i)] = "Missing value"
			}
			n++
		}
	}
	if n == 0 {
		errors["StateLicenses"] = "At least one state license is required"
	}
	return errors
}

func NewCredentialsHandler(router *mux.Router, dataAPI api.DataAPI) http.Handler {
	return www.SupportedMethodsHandler(&credentialsHandler{
		router:  router,
		dataAPI: dataAPI,
	}, []string{"GET", "POST"})
}

func (h *credentialsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	req := &credentialsForm{}
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
			// Allowed to panic since it should never ever happen
			account := context.Get(r, www.CKAccount).(*common.Account)
			doctorID, err := h.dataAPI.GetDoctorIdFromAccountId(account.ID)
			if err != nil {
				www.InternalServerError(w, r, err)
				return
			}
			if err := h.dataAPI.SetDoctorNPI(doctorID, req.NPI); err != nil {
				www.InternalServerError(w, r, err)
				return
			}

			// TODO

			if u, err := h.router.Get("doctor-register-upload-cv").URLPath(); err != nil {
				www.InternalServerError(w, r, err)
			} else {
				http.Redirect(w, r, u.String(), http.StatusSeeOther)
			}
			return
		}
	}

	// Padd with empty entries so that they render
	for len(req.StateLicenses) < 6 {
		req.StateLicenses = append(req.StateLicenses, &stateLicense{})
	}

	states, err := h.dataAPI.ListStates()
	if err != nil {
		www.InternalServerError(w, r, err)
	}
	states = append([]*common.State{
		&common.State{
			Name:         "Select state",
			Abbreviation: "",
		}}, states...)
	www.TemplateResponse(w, http.StatusOK, credsTemplate, &www.BaseTemplateContext{
		Title: "Identity & Credentials | Doctor Registration | Spruce",
		SubContext: &credsTemplateContext{
			Form:            req,
			FormErrors:      errors,
			States:          states,
			LicenseStatuses: licenseStatuses,
		},
	})
}
