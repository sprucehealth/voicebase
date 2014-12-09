package dronboard

import (
	"encoding/json"
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
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/www"
)

type claimsHistoryHandler struct {
	router   *mux.Router
	dataAPI  api.DataAPI
	authAPI  api.AuthAPI
	store    storage.Store
	template *template.Template
	attrName string
	fileTag  string
	nextURL  string
}

type insurancePolicy struct {
	Company string
	Number  string
}

type claimsHistoryForm struct {
	Name      string
	Policies  []insurancePolicy
	Signature string
	ESigAgree bool
}

func (f *claimsHistoryForm) Validate() map[string]string {
	errors := map[string]string{}
	if f.Name == "" {
		errors["Name"] = "Name is required"
	}
	if f.Signature == "" {
		errors["Signature"] = "Signature is required"
	}
	if !f.ESigAgree {
		errors["ESigAgree"] = "You must agree to sign electronically"
	}
	return errors
}

func NewClaimsHistoryHandler(router *mux.Router, dataAPI api.DataAPI, store storage.Store, templateLoader *www.TemplateLoader) http.Handler {
	return httputil.SupportedMethods(&claimsHistoryHandler{
		router:   router,
		dataAPI:  dataAPI,
		store:    store,
		attrName: api.AttrClaimsHistoryFile,
		fileTag:  "claimshistory",
		template: templateLoader.MustLoadTemplate("dronboard/claimshistory.html", "dronboard/base.html", nil),
		nextURL:  "doctor-register-background-check",
	}, []string{"GET", "POST"})
}

func (h *claimsHistoryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	account := context.Get(r, www.CKAccount).(*common.Account)
	doctor, err := h.dataAPI.GetDoctorFromAccountID(account.ID)
	if err != nil {
		www.InternalServerError(w, r, err)
		return
	}

	u, err := h.router.Get(h.nextURL).URLPath()
	if err != nil {
		www.InternalServerError(w, r, err)
		return
	}
	nextURL := u.String()

	// See if the doctor already uploaded the file or agreed. If so then skip this step
	attr, err := h.dataAPI.DoctorAttributes(doctor.DoctorID.Int64(), []string{api.AttrClaimsHistoryFile, api.AttrClaimsHistoryAgreement})
	if err != nil {
		www.InternalServerError(w, r, err)
		return
	}
	if len(attr) != 0 {
		http.Redirect(w, r, nextURL, http.StatusSeeOther)
		return
	}

	form := &claimsHistoryForm{}
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
			js, err := json.Marshal(form)
			if err != nil {
				www.InternalServerError(w, r, err)
				return
			}
			if err := h.dataAPI.UpdateDoctorAttributes(doctor.DoctorID.Int64(), map[string]string{api.AttrClaimsHistoryAgreement: string(js)}); err != nil {
				www.InternalServerError(w, r, err)
				return
			}
			http.Redirect(w, r, nextURL, http.StatusSeeOther)
			return
		}
	} else {
		attr, err := h.dataAPI.DoctorAttributes(doctor.DoctorID.Int64(), []string{
			api.AttrCurrentLiabilityInsurer,
			api.AttrPreviousLiabilityInsurers,
		})
		if err != nil {
			www.InternalServerError(w, r, err)
			return
		}
		policies := []insurancePolicy{
			insurancePolicy{Company: attr[api.AttrCurrentLiabilityInsurer]},
		}
		if in := attr[api.AttrPreviousLiabilityInsurers]; in != "" {
			for _, company := range strings.Split(in, "\n") {
				policies = append(policies, insurancePolicy{Company: company})
			}
		}
		for len(policies) < 5 {
			policies = append(policies, insurancePolicy{})
		}
		form.Policies = policies
	}

	www.TemplateResponse(w, http.StatusOK, h.template, &www.BaseTemplateContext{
		Title: template.HTML("Claims History | Doctor Registration | Spruce"),
		SubContext: &struct {
			Form       *claimsHistoryForm
			FormErrors map[string]string
			Name       string
			NextURL    string
		}{
			Form:       form,
			FormErrors: errors,
			NextURL:    nextURL,
			Name:       fmt.Sprintf("Dr. %s %s", doctor.FirstName, doctor.LastName),
		},
	})
}
