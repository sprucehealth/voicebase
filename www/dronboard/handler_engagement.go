package dronboard

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/third_party/github.com/SpruceHealth/schema"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/www"
)

type engagementHandler struct {
	router  *mux.Router
	dataAPI api.DataAPI
}

type engagementRequest struct {
	HoursPerWeek string
	TimesActive  string
	JacketSize   string
	Excitement   string
}

func (r *engagementRequest) Validate() map[string]string {
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
	req := &engagementRequest{}
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

			if u, err := h.router.Get("doctor-register-malpractice").URLPath(); err != nil {
				www.InternalServerError(w, r, err)
			} else {
				http.Redirect(w, r, u.String(), http.StatusSeeOther)
			}
			return
		}
	}

	www.TemplateResponse(w, http.StatusOK, engagementTemplate, &www.BaseTemplateContext{
		Title: "Identity & Credentials | Doctor Registration | Spruce",
		SubContext: &engagementTemplateContext{
			Form:       req,
			FormErrors: errors,
		},
	})
}
