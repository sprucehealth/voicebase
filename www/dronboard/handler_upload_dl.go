package dronboard

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/www"
)

type uploadLicenseHandler struct {
	router  *mux.Router
	dataAPI api.DataAPI
	authAPI api.AuthAPI
}

func NewUploadLicenseHandler(router *mux.Router, dataAPI api.DataAPI) http.Handler {
	return www.SupportedMethodsHandler(&uploadLicenseHandler{
		router:  router,
		dataAPI: dataAPI,
	}, []string{"GET", "POST"})
}

func (h *uploadLicenseHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

		if u, err := h.router.Get("doctor-register-engagement").URLPath(); err != nil {
			www.InternalServerError(w, r, err)
		} else {
			http.Redirect(w, r, u.String(), http.StatusSeeOther)
		}
		return
	}

	www.TemplateResponse(w, http.StatusOK, uploadTemplate, &www.BaseTemplateContext{
		Title: "Upload Driver's License | Doctor Registration | Spruce",
		SubContext: &uploadTemplateContext{
			Title: "Upload Driver's License",
		},
	})
}
