package dronboard

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/www"
)

const maxMemory = 5 * 1024 * 1024

type uploadCVHandler struct {
	router  *mux.Router
	dataAPI api.DataAPI
	authAPI api.AuthAPI
}

func NewUploadCVHandler(router *mux.Router, dataAPI api.DataAPI) http.Handler {
	return www.SupportedMethodsHandler(&uploadCVHandler{
		router:  router,
		dataAPI: dataAPI,
	}, []string{"GET", "POST"})
}

func (h *uploadCVHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if err := r.ParseMultipartForm(maxMemory); err != nil {
			www.InternalServerError(w, r, err)
			return
		}

		file, header, err := r.FormFile("File")
		if err != nil {
			www.InternalServerError(w, r, err)
			return
		}

		_ = file
		_ = header
		// TODO

		if u, err := h.router.Get("doctor-register-upload-license").URLPath(); err != nil {
			www.InternalServerError(w, r, err)
		} else {
			http.Redirect(w, r, u.String(), http.StatusSeeOther)
		}
		return
	}

	www.TemplateResponse(w, http.StatusOK, uploadTemplate, &www.BaseTemplateContext{
		Title: "Upload CV | Doctor Registration | Spruce",
		SubContext: &uploadTemplateContext{
			Title: "Upload CV / Résumé",
		},
	})
}
