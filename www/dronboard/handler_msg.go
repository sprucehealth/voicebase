package dronboard

import (
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

const defaultSavedMessage = `I've taken a look at your pictures, and have put together a treatment regimen for you that will take roughly 3 months to take full effect. Please stick with it as best as you can, unless you are having a concerning complication. Often times, acne gets slightly worse before it gets better.

Please keep in mind finding the right "recipe" to treat your acne may take some tweaking. As always, feel free to communicate any questions or issues you have along the way.

Sincerely,
`

type savedMessageHandler struct {
	router   *mux.Router
	dataAPI  api.DataAPI
	template *template.Template
	nextStep string
}

type savedMessageForm struct {
	Message string
}

func (r *savedMessageForm) Validate() map[string]string {
	errors := map[string]string{}
	if r.Message == "" {
		errors["Message"] = "Message is required"
	}
	return errors
}

func NewSavedMessageHandler(router *mux.Router, dataAPI api.DataAPI, templateLoader *www.TemplateLoader) http.Handler {
	return httputil.SupportedMethods(&savedMessageHandler{
		router:   router,
		dataAPI:  dataAPI,
		template: templateLoader.MustLoadTemplate("dronboard/saved_message.html", "dronboard/base.html", nil),
		nextStep: "doctor-register-credentials",
	}, []string{"GET", "POST"})
}

func (h *savedMessageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	account := context.Get(r, www.CKAccount).(*common.Account)

	form := &savedMessageForm{}
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
			doctorID, err := h.dataAPI.GetDoctorIdFromAccountId(account.ID)
			if err != nil {
				www.InternalServerError(w, r, err)
				return
			}

			if err := h.dataAPI.SetSavedMessageForDoctor(doctorID, form.Message); err != nil {
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
		doctor, err := h.dataAPI.GetDoctorFromAccountId(account.ID)
		if err != nil {
			www.InternalServerError(w, r, err)
			return
		}

		msg, err := h.dataAPI.GetSavedMessageForDoctor(doctor.DoctorId.Int64())
		if err == api.NoRowsError {
			msg = ""
		} else if err != nil {
			www.InternalServerError(w, r, err)
			return
		}

		if msg == "" {
			msg = defaultSavedMessage + "Dr. " + doctor.LastName
		}

		form.Message = msg
	}

	www.TemplateResponse(w, http.StatusOK, h.template, &www.BaseTemplateContext{
		Title: "Saved Message | Doctor Registration | Spruce",
		SubContext: &struct {
			Form       *savedMessageForm
			FormErrors map[string]string
		}{
			Form:       form,
			FormErrors: errors,
		},
	})
}
