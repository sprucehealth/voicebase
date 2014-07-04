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
	licenseStatuses = []common.MedicalLicenseStatus{
		common.Active,
		common.Inactive,
		common.Temporary,
		common.Pending,
	}
)

type credentialsHandler struct {
	router  *mux.Router
	dataAPI api.DataAPI
}

type stateLicense struct {
	State  string
	Number string
	Status string
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
			if l.Number == "" || l.Status == "" {
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

			// TODO: SSN

			licenses := make([]*common.MedicalLicense, 0, len(req.StateLicenses))
			for _, l := range req.StateLicenses {
				status, err := common.GetMedicalLicenseStatus(l.Status)
				if err != nil {
					// TODO: this should just show an error on the form but should
					// only ever happen if someone tries a POST without using the form
					// so an internal error is fine for now.
					www.InternalServerError(w, r, err)
					return
				}
				if l.State != "" {
					licenses = append(licenses, &common.MedicalLicense{
						DoctorID: doctorID,
						State:    l.State,
						Status:   status,
						Number:   l.Number,
					})
				}
			}
			if err := h.dataAPI.AddMedicalLicenses(licenses); err != nil {
				www.InternalServerError(w, r, err)
				return
			}

			attributes := map[string]string{
				api.AttrAmericanBoardCertified: api.BoolToString(req.AmericanBoardCertified),
				api.AttrContinuedEducation:     api.BoolToString(req.ContinuedEducation),
				api.AttrRiskManagementCourse:   api.BoolToString(req.RiskManagementCourse),
			}
			if req.AmericanBoardCertified {
				attributes[api.AttrSpecialtyBoard] = req.SpecialtyBoard
				attributes[api.AttrMostRecentCertificationDate] = req.RecentCertDate
			}
			if req.ContinuedEducation {
				attributes[api.AttrContinuedEducationCreditHours] = req.CreditHours
			}
			if err := h.dataAPI.UpdateDoctorAttributes(doctorID, attributes); err != nil {
				www.InternalServerError(w, r, err)
				return
			}

			if u, err := h.router.Get("doctor-register-upload-cv").URLPath(); err != nil {
				www.InternalServerError(w, r, err)
			} else {
				http.Redirect(w, r, u.String(), http.StatusSeeOther)
			}
			return
		}
	}

	// Pad with empty entries so that they render
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
