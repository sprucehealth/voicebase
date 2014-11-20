package dronboard

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/SpruceHealth/schema"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/context"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

const none = "NONE"

type insuranceHandler struct {
	router   *mux.Router
	dataAPI  api.DataAPI
	template *template.Template
	nextStep string
}

type explanation struct {
	Date        string
	Explanation string
}

// Needed because strangly 'schema' can't decode into []string
type str struct {
	Str string
}

func strSlice(strs []string) []str {
	x := make([]str, len(strs))
	for i, s := range strs {
		x[i] = str{s}
	}
	return x
}

type insuranceForm struct {
	CurrentInsurer      string
	PreviousInsurers    []str
	Violations          string
	InsuranceDeclines   string
	SexualMisconduct    string
	Impairments         string
	Claims              string
	Incidents           string
	NoViolations        bool
	NoInsuranceDeclines bool
	NoSexualMisconduct  bool
	NoImpairments       bool
	NoClaims            bool
	NoIncidents         bool
	Signature           string
	SignatureDate       string
	ESigAgree           bool
}

func (f *insuranceForm) Validate() map[string]string {
	errors := map[string]string{}
	if f.Signature == "" {
		errors["Signature"] = "Signature is required"
	}
	if f.SignatureDate == "" {
		errors["SignatureDate"] = "Signature date is required"
	}
	if !f.NoViolations && f.Violations == "" {
		errors["Violations"] = "Must either respond 'no' or provide more information."
	}
	if !f.NoInsuranceDeclines && f.InsuranceDeclines == "" {
		errors["InsuranceDeclines"] = "Must either respond 'no' or provide more information."
	}
	if !f.NoSexualMisconduct && f.SexualMisconduct == "" {
		errors["SexualMisconduct"] = "Must either respond 'no' or provide more information."
	}
	if !f.NoImpairments && f.Impairments == "" {
		errors["Impairments"] = "Must either respond 'no' or provide more information."
	}
	if !f.NoClaims && f.Claims == "" {
		errors["Claims"] = "Must either respond 'no' or provide more information."
	}
	if !f.NoIncidents && f.Incidents == "" {
		errors["Incidents"] = "Must either respond 'no' or provide more information."
	}
	if !f.ESigAgree {
		errors["ESigAgree"] = "Must agree to sign electronically"
	}
	return errors
}

func NewInsuranceHandler(router *mux.Router, dataAPI api.DataAPI, templateLoader *www.TemplateLoader) http.Handler {
	return httputil.SupportedMethods(&insuranceHandler{
		router:   router,
		dataAPI:  dataAPI,
		template: templateLoader.MustLoadTemplate("dronboard/insurance.html", "dronboard/base.html", nil),
		nextStep: "doctor-register-upload-claims-history",
	}, []string{"GET", "POST"})
}

func (h *insuranceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	form := &insuranceForm{}
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
			account := context.Get(r, www.CKAccount).(*common.Account)
			doctorID, err := h.dataAPI.GetDoctorIdFromAccountId(account.ID)
			if err != nil {
				www.InternalServerError(w, r, err)
				return
			}

			var p []string
			for _, in := range form.PreviousInsurers {
				if in.Str != "" {
					p = append(p, in.Str)
				}
			}
			previousInsurers := strings.Join(p, "\n")

			attributes := map[string]string{
				api.AttrCurrentLiabilityInsurer:   form.CurrentInsurer,
				api.AttrPreviousLiabilityInsurers: previousInsurers,
				api.AttrDoctorViolations:          ors(form.NoViolations, none, form.Violations),
				api.AttrInsuranceDeclines:         ors(form.NoInsuranceDeclines, none, form.InsuranceDeclines),
				api.AttrSexualMisconduct:          ors(form.NoSexualMisconduct, none, form.SexualMisconduct),
				api.AttrDoctorImpairments:         ors(form.NoImpairments, none, form.Impairments),
				api.AttrDoctorClaims:              ors(form.NoClaims, none, form.Claims),
				api.AttrDoctorIncidents:           ors(form.NoIncidents, none, form.Incidents),
				api.AttrInsuranceSignature:        fmt.Sprintf("%s|%s", form.Signature, form.SignatureDate),
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
		account := context.Get(r, www.CKAccount).(*common.Account)
		doctorID, err := h.dataAPI.GetDoctorIdFromAccountId(account.ID)
		if err != nil {
			www.InternalServerError(w, r, err)
			return
		}
		attr, err := h.dataAPI.DoctorAttributes(doctorID, []string{
			api.AttrCurrentLiabilityInsurer,
			api.AttrPreviousLiabilityInsurers,
			api.AttrDoctorViolations,
			api.AttrInsuranceDeclines,
			api.AttrSexualMisconduct,
			api.AttrDoctorImpairments,
			api.AttrDoctorClaims,
			api.AttrDoctorIncidents,
		})
		if err != nil {
			www.InternalServerError(w, r, err)
			return
		}

		form.CurrentInsurer = attr[api.AttrCurrentLiabilityInsurer]
		form.PreviousInsurers = strSlice(strings.Split(attr[api.AttrPreviousLiabilityInsurers], "\n"))
		form.NoViolations = attr[api.AttrDoctorViolations] == none
		form.Violations = ors(form.NoViolations, "", attr[api.AttrDoctorViolations])
		form.NoInsuranceDeclines = attr[api.AttrInsuranceDeclines] == none
		form.InsuranceDeclines = ors(form.NoInsuranceDeclines, "", attr[api.AttrInsuranceDeclines])
		form.NoSexualMisconduct = attr[api.AttrSexualMisconduct] == none
		form.SexualMisconduct = ors(form.NoSexualMisconduct, "", attr[api.AttrSexualMisconduct])
		form.NoImpairments = attr[api.AttrDoctorImpairments] == none
		form.Impairments = ors(form.NoImpairments, "", attr[api.AttrDoctorImpairments])
		form.NoClaims = attr[api.AttrDoctorClaims] == none
		form.Claims = ors(form.NoClaims, "", attr[api.AttrDoctorClaims])
		form.NoInsuranceDeclines = attr[api.AttrDoctorIncidents] == none
		form.Incidents = ors(form.NoInsuranceDeclines, "", attr[api.AttrDoctorIncidents])
	}

	for len(form.PreviousInsurers) < 5 {
		form.PreviousInsurers = append(form.PreviousInsurers, str{""})
	}

	www.TemplateResponse(w, http.StatusOK, h.template, &www.BaseTemplateContext{
		Title: "Malpractice Coverage | Doctor Registration | Spruce",
		SubContext: &struct {
			Form       *insuranceForm
			FormErrors map[string]string
		}{
			Form:       form,
			FormErrors: errors,
		},
	})
}

func ors(b bool, t, s string) string {
	if b {
		return t
	}
	return s
}
