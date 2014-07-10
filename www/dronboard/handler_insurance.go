package dronboard

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/third_party/github.com/SpruceHealth/schema"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/context"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/www"
)

type insuranceHandler struct {
	router   *mux.Router
	dataAPI  api.DataAPI
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
	CurrentInsurer    string
	PreviousInsurers  []str
	Violations        string
	InsuranceDeclines string
	SexualMisconduct  string
	Impairments       string
	Claims            string
	Incidents         string
	Signature         string
	SignatureDate     string
	ESigAgree         bool
}

func (f *insuranceForm) Validate() map[string]string {
	errors := map[string]string{}
	if f.Signature == "" {
		errors["Signature"] = "Signature is required"
	}
	if f.SignatureDate == "" {
		errors["SignatureDate"] = "Signature date is required"
	}
	if !f.ESigAgree {
		errors["ESigAgree"] = "Must agree to sign electronically"
	}
	return errors
}

func NewInsuranceHandler(router *mux.Router, dataAPI api.DataAPI) http.Handler {
	return www.SupportedMethodsHandler(&insuranceHandler{
		router:   router,
		dataAPI:  dataAPI,
		nextStep: "doctor-register-engagement",
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
				api.AttrDoctorViolations:          form.Violations,
				api.AttrInsuranceDeclines:         form.InsuranceDeclines,
				api.AttrSexualMisconduct:          form.SexualMisconduct,
				api.AttrDoctorImpairments:         form.Impairments,
				api.AttrDoctorClaims:              form.Claims,
				api.AttrDoctorIncidents:           form.Incidents,
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
		form.Violations = attr[api.AttrDoctorViolations]
		form.InsuranceDeclines = attr[api.AttrInsuranceDeclines]
		form.SexualMisconduct = attr[api.AttrSexualMisconduct]
		form.Impairments = attr[api.AttrDoctorImpairments]
		form.Claims = attr[api.AttrDoctorClaims]
		form.Incidents = attr[api.AttrDoctorIncidents]
	}

	for len(form.PreviousInsurers) < 5 {
		form.PreviousInsurers = append(form.PreviousInsurers, str{""})
	}

	www.TemplateResponse(w, http.StatusOK, insuranceTemplate, &www.BaseTemplateContext{
		Title: "Malpractice Coverage | Doctor Registration | Spruce",
		SubContext: &insuranceTemplateContext{
			Form:       form,
			FormErrors: errors,
		},
	})
}
