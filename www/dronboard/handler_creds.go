package dronboard

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/SpruceHealth/schema"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/www"
)

var (
	licenseStatuses = []common.MedicalLicenseStatus{
		common.MLActive,
		common.MLInactive,
		common.MLTemporary,
		common.MLPending,
	}
)

type credentialsHandler struct {
	router   *mux.Router
	dataAPI  api.DataAPI
	template *template.Template
	nextStep string
}

type stateLicense struct {
	State      string
	Number     string
	Status     string
	Expiration string

	status     common.MedicalLicenseStatus
	expiration *encoding.Date
}

type credentialsForm struct {
	AmericanBoardCertified bool
	SpecialtyBoard         string
	RecentCertDate         string
	ContinuedEducation     bool
	CreditHours            string
	RiskManagementCourse   bool
	NPI                    string
	DEA                    string
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
	if r.DEA == "" {
		errors["DEA"] = "DEA is required"
	}

	n := 0
	for i, l := range r.StateLicenses {
		if l.State != "" {
			if l.Number == "" || l.Status == "" {
				errors[fmt.Sprintf("StateLicenses.%d", i)] = "Missing value"
				continue
			} else if l.Status != "" {
				status, err := common.GetMedicalLicenseStatus(l.Status)
				if err == nil {
					l.status = status
				} else {
					errors[fmt.Sprintf("StateLicenses.%d", i)] = "Bad status value"
					continue
				}
			}
			if l.Expiration != "" {
				cutoffYear := time.Now().UTC().Year() + 50
				date, err := encoding.ParseDate(l.Expiration, "YMD", []rune{'/', '-'}, cutoffYear)
				if err != nil {
					date, err = encoding.ParseDate(l.Expiration, "MDY", []rune{'/', '-'}, cutoffYear)
				}
				if err == nil {
					l.expiration = &date
				} else {
					errors[fmt.Sprintf("StateLicenses.%d", i)] = "Bad expiration date format (mm/dd/yyyy)"
				}
			} else if l.status == common.MLActive {
				errors[fmt.Sprintf("StateLicenses.%d", i)] = "Expiration date is required"
			}
			n++
		}
	}
	if n == 0 {
		errors["StateLicenses"] = "At least one state license is required"
	}
	return errors
}

func newCredentialsHandler(router *mux.Router, dataAPI api.DataAPI, templateLoader *www.TemplateLoader) httputil.ContextHandler {
	return httputil.ContextSupportedMethods(&credentialsHandler{
		router:   router,
		dataAPI:  dataAPI,
		template: templateLoader.MustLoadTemplate("dronboard/creds.html", "dronboard/base.html", nil),
		nextStep: "doctor-register-upload-cv",
	}, httputil.Get, httputil.Post)
}

func (h *credentialsHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	account := www.MustCtxAccount(ctx)

	form := &credentialsForm{}
	var errors map[string]string

	if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			www.InternalServerError(w, r, err)
			return
		}

		if err := schema.NewDecoder().Decode(form, r.PostForm); err != nil {
			www.InternalServerError(w, r, err)
			return
		}

		errors = form.Validate()
		if len(errors) == 0 {
			doctorID, err := h.dataAPI.GetDoctorIDFromAccountID(account.ID)
			if err != nil {
				www.InternalServerError(w, r, err)
				return
			}

			if err := h.dataAPI.UpdateDoctor(doctorID, &api.DoctorUpdate{NPI: &form.NPI, DEA: &form.DEA}); err != nil {
				www.InternalServerError(w, r, err)
				return
			}

			licenses := make([]*common.MedicalLicense, 0, len(form.StateLicenses))
			for _, l := range form.StateLicenses {
				if l.State != "" {
					licenses = append(licenses, &common.MedicalLicense{
						DoctorID:   doctorID,
						State:      l.State,
						Status:     l.status,
						Number:     l.Number,
						Expiration: l.expiration,
					})
				}
			}
			if err := h.dataAPI.AddMedicalLicenses(licenses); err != nil {
				www.InternalServerError(w, r, err)
				return
			}

			attributes := map[string]string{
				api.AttrSocialSecurityNumber:   form.SSN,
				api.AttrAmericanBoardCertified: strconv.FormatBool(form.AmericanBoardCertified),
				api.AttrContinuedEducation:     strconv.FormatBool(form.ContinuedEducation),
				api.AttrRiskManagementCourse:   strconv.FormatBool(form.RiskManagementCourse),
			}
			if form.AmericanBoardCertified {
				attributes[api.AttrSpecialtyBoard] = form.SpecialtyBoard
				attributes[api.AttrMostRecentCertificationDate] = form.RecentCertDate
			}
			if form.ContinuedEducation {
				attributes[api.AttrContinuedEducationCreditHours] = form.CreditHours
			}
			if err := h.dataAPI.UpdateDoctorAttributes(doctorID, attributes); err != nil {
				www.InternalServerError(w, r, err)
				return
			}

			if u, err := h.router.Get(h.nextStep).URLPath(); err != nil {
				www.InternalServerError(w, r, err)
			} else {
				http.Redirect(w, r, u.String(), http.StatusSeeOther)
			}
			return
		}
	} else {
		// Pull up old information if available
		doctor, err := h.dataAPI.GetDoctorFromAccountID(account.ID)
		if err != nil {
			www.InternalServerError(w, r, err)
			return
		}
		form.NPI = doctor.NPI
		form.DEA = doctor.DEA
		attr, err := h.dataAPI.DoctorAttributes(doctor.ID.Int64(), []string{
			api.AttrSocialSecurityNumber,
			api.AttrAmericanBoardCertified,
			api.AttrContinuedEducation,
			api.AttrRiskManagementCourse,
			api.AttrSpecialtyBoard,
			api.AttrMostRecentCertificationDate,
			api.AttrContinuedEducationCreditHours,
		})
		if err != nil {
			www.InternalServerError(w, r, err)
			return
		}
		form.SSN = attr[api.AttrSocialSecurityNumber]
		form.AmericanBoardCertified, _ = strconv.ParseBool(attr[api.AttrAmericanBoardCertified])
		form.SpecialtyBoard = attr[api.AttrSpecialtyBoard]
		form.RecentCertDate = attr[api.AttrMostRecentCertificationDate]
		form.ContinuedEducation, _ = strconv.ParseBool(attr[api.AttrContinuedEducation])
		form.CreditHours = attr[api.AttrContinuedEducationCreditHours]
		form.RiskManagementCourse, _ = strconv.ParseBool(attr[api.AttrRiskManagementCourse])

		licenses, err := h.dataAPI.MedicalLicenses(doctor.ID.Int64())
		if err != nil {
			www.InternalServerError(w, r, err)
			return
		}
		for _, l := range licenses {
			var exp string
			if l.Expiration != nil {
				// Using YYYY-MM-DD since that's what Chrome expects for a date field (otherwise
				// it doesn't show the value). It would be better to format it properly based on
				// browser and support for HTML5 input fields, but that requires some mangling
				// of the value in javascript. Hopefully people will understand if they revisit
				// the page what the parts mean.
				exp = l.Expiration.String()
			}
			form.StateLicenses = append(form.StateLicenses, &stateLicense{
				State:      l.State,
				Number:     l.Number,
				Status:     l.Status.String(),
				Expiration: exp,
			})
		}
	}

	// Pad with empty entries so that they render
	for len(form.StateLicenses) < 8 {
		form.StateLicenses = append(form.StateLicenses, &stateLicense{})
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
	www.TemplateResponse(w, http.StatusOK, h.template, &www.BaseTemplateContext{
		Title: "Identity & Credentials | Doctor Registration | Spruce",
		SubContext: &struct {
			Form            *credentialsForm
			FormErrors      map[string]string
			LicenseStatuses []common.MedicalLicenseStatus
			States          []*common.State
		}{
			Form:            form,
			FormErrors:      errors,
			States:          states,
			LicenseStatuses: licenseStatuses,
		},
	})
}
