package dronboard

import (
	"encoding/json"
	"html/template"
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/SpruceHealth/schema"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/context"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type bgCheckHandler struct {
	router   *mux.Router
	dataAPI  api.DataAPI
	template *template.Template
	nextStep string
}

type bgCheckForm struct {
	FirstName   string
	LastName    string
	MiddleName  string
	Signature   string
	Date        string
	CopyRequest bool
	ESigAgree   bool
}

func (r *bgCheckForm) Validate() map[string]string {
	errors := map[string]string{}
	if r.FirstName == "" {
		errors["FirstName"] = "First name is required"
	}
	if r.LastName == "" {
		errors["LastName"] = "Last name is required"
	}
	if r.Signature == "" {
		errors["Signature"] = "Signature is required"
	}
	if r.Date == "" {
		errors["Date"] = "Date is required"
	}
	if !r.ESigAgree {
		errors["ESigAgree"] = "You must agree to sign electronically"
	}
	return errors
}

func NewBackgroundCheckHandler(router *mux.Router, dataAPI api.DataAPI, templateLoader *www.TemplateLoader) http.Handler {
	return httputil.SupportedMethods(&bgCheckHandler{
		router:   router,
		dataAPI:  dataAPI,
		template: templateLoader.MustLoadTemplate("dronboard/backgroundcheck.html", "dronboard/base.html", nil),
		nextStep: "doctor-register-financials",
	}, httputil.Get, httputil.Post)
}

func (h *bgCheckHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	account := context.Get(r, www.CKAccount).(*common.Account)
	doctorID, err := h.dataAPI.GetDoctorIDFromAccountID(account.ID)
	if err != nil {
		www.InternalServerError(w, r, err)
		return
	}

	// See if the doctor already agreed. If so then skip this step
	attr, err := h.dataAPI.DoctorAttributes(doctorID, []string{api.AttrBackgroundCheckAgreement})
	if err != nil {
		www.InternalServerError(w, r, err)
		return
	}
	if attr[api.AttrBackgroundCheckAgreement] != "" {
		h.redirectToNextStep(w, r)
		return
	}

	form := &bgCheckForm{}
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
			// Store all the content json serialized. This avoids having to create a bunch of attributes.
			js, err := json.Marshal(form)
			if err != nil {
				www.InternalServerError(w, r, err)
				return
			}
			attributes := map[string]string{
				api.AttrBackgroundCheckAgreement: string(js),
			}
			if err := h.dataAPI.UpdateDoctorAttributes(doctorID, attributes); err != nil {
				www.InternalServerError(w, r, err)
				return
			}

			h.redirectToNextStep(w, r)
			return
		}
	} else {
		doctor, err := h.dataAPI.GetDoctorFromID(doctorID)
		if err != nil {
			www.InternalServerError(w, r, err)
			return
		}
		form.FirstName = doctor.FirstName
		form.LastName = doctor.LastName
	}

	www.TemplateResponse(w, http.StatusOK, h.template, &www.BaseTemplateContext{
		Title: "Background Check Agreement | Doctor Registration | Spruce",
		SubContext: &struct {
			Form       *bgCheckForm
			FormErrors map[string]string
		}{
			Form:       form,
			FormErrors: errors,
		},
	})
}

func (h *bgCheckHandler) redirectToNextStep(w http.ResponseWriter, r *http.Request) {
	if u, err := h.router.Get(h.nextStep).URLPath(); err != nil {
		www.InternalServerError(w, r, err)
	} else {
		http.Redirect(w, r, u.String(), http.StatusSeeOther)
	}
}
