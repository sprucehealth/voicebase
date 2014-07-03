package dronboard

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/third_party/github.com/SpruceHealth/schema"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/www"
)

type financialsHandler struct {
	router  *mux.Router
	dataAPI api.DataAPI
}

type financialsForm struct {
	AccountNumber string
	RoutingNumber string
}

func (r *financialsForm) Validate() map[string]string {
	errors := map[string]string{}
	return errors
}

func NewFinancialsHandler(router *mux.Router, dataAPI api.DataAPI) http.Handler {
	return www.SupportedMethodsHandler(&financialsHandler{
		router:  router,
		dataAPI: dataAPI,
	}, []string{"GET", "POST"})
}

func (h *financialsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	req := &financialsForm{}
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

			if u, err := h.router.Get("TODO").URLPath(); err != nil {
				www.InternalServerError(w, r, err)
			} else {
				http.Redirect(w, r, u.String(), http.StatusSeeOther)
			}
			return
		}
	}

	www.TemplateResponse(w, http.StatusOK, financialsTemplate, &www.BaseTemplateContext{
		Title: "Financials | Doctor Registration | Spruce",
		SubContext: &financialsTemplateContext{
			Form:       req,
			FormErrors: errors,
		},
	})
}
