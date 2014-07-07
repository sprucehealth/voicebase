package dronboard

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/third_party/github.com/SpruceHealth/schema"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/context"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/www"
)

type engagementHandler struct {
	router  *mux.Router
	dataAPI api.DataAPI
}

type engagementForm struct {
	HoursPerWeek string
	TimesActive  string
	JacketSize   string
	Excitement   string
}

func (r *engagementForm) Validate() map[string]string {
	errors := map[string]string{}
	return errors
}

func NewEngagementHandler(router *mux.Router, dataAPI api.DataAPI) http.Handler {
	return www.SupportedMethodsHandler(&engagementHandler{
		router:  router,
		dataAPI: dataAPI,
	}, []string{"GET", "POST"})
}

func (h *engagementHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	form := &engagementForm{}
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

			attributes := map[string]string{
				api.AttrHoursUsingSprucePerWeek: form.HoursPerWeek,
				api.AttrTimesActiveOnSpruce:     form.TimesActive,
				api.AttrJacketSize:              form.JacketSize,
				api.AttrExcitedAboutSpruce:      form.Excitement,
			}
			if err := h.dataAPI.UpdateDoctorAttributes(doctorID, attributes); err != nil {
				www.InternalServerError(w, r, err)
				return
			}

			if u, err := h.router.Get("doctor-register-insurance").URLPath(); err != nil {
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
			api.AttrHoursUsingSprucePerWeek,
			api.AttrTimesActiveOnSpruce,
			api.AttrJacketSize,
			api.AttrExcitedAboutSpruce,
		})
		if err != nil {
			www.InternalServerError(w, r, err)
			return
		}

		form.HoursPerWeek = attr[api.AttrHoursUsingSprucePerWeek]
		form.TimesActive = attr[api.AttrTimesActiveOnSpruce]
		form.JacketSize = attr[api.AttrJacketSize]
		form.Excitement = attr[api.AttrExcitedAboutSpruce]
	}

	www.TemplateResponse(w, http.StatusOK, engagementTemplate, &www.BaseTemplateContext{
		Title: "Identity & Credentials | Doctor Registration | Spruce",
		SubContext: &engagementTemplateContext{
			Form:       form,
			FormErrors: errors,
		},
	})
}
