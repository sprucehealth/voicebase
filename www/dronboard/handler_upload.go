package dronboard

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/context"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/www"
)

const maxMemory = 5 * 1024 * 1024

type uploadHandler struct {
	router   *mux.Router
	dataAPI  api.DataAPI
	authAPI  api.AuthAPI
	store    storage.Store
	attrName string
	fileTag  string
	title    string
	nextURL  string
}

func NewUploadCVHandler(router *mux.Router, dataAPI api.DataAPI, store storage.Store) http.Handler {
	return www.SupportedMethodsHandler(&uploadHandler{
		router:   router,
		dataAPI:  dataAPI,
		store:    store,
		attrName: api.AttrCVFile,
		fileTag:  "cv",
		title:    "Upload CV / Résumé",
		nextURL:  "doctor-register-upload-license",
	}, []string{"GET", "POST"})
}

func NewUploadLicenseHandler(router *mux.Router, dataAPI api.DataAPI, store storage.Store) http.Handler {
	return www.SupportedMethodsHandler(&uploadHandler{
		router:   router,
		dataAPI:  dataAPI,
		store:    store,
		attrName: api.AttrDriversLicenseFile,
		fileTag:  "dl",
		title:    "Upload Driver's License",
		nextURL:  "doctor-register-upload-claims-history",
	}, []string{"GET", "POST"})
}

func NewUploadClaimsHistory(router *mux.Router, dataAPI api.DataAPI, store storage.Store) http.Handler {
	return www.SupportedMethodsHandler(&uploadHandler{
		router:   router,
		dataAPI:  dataAPI,
		store:    store,
		attrName: api.AttrClaimsHistory,
		fileTag:  "claimshistory",
		title:    "Upload Claims History",
		nextURL:  "doctor-register-engagement",
	}, []string{"GET", "POST"})
}

func (h *uploadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		account := context.Get(r, www.CKAccount).(*common.Account)
		doctorID, err := h.dataAPI.GetDoctorIdFromAccountId(account.ID)
		if err != nil {
			www.InternalServerError(w, r, err)
			return
		}

		if err := r.ParseMultipartForm(maxMemory); err != nil {
			www.InternalServerError(w, r, err)
			return
		}

		file, fileHandler, err := r.FormFile("File")
		if err != nil {
			www.InternalServerError(w, r, err)
			return
		}
		defer file.Close()

		headers := http.Header{
			"Content-Type":  []string{fileHandler.Header.Get("Content-Type")},
			"Original-Name": []string{fileHandler.Filename},
		}

		size, err := common.SeekerSize(file)
		if err != nil {
			www.InternalServerError(w, r, err)
			return
		}

		fileID, err := h.store.PutReader(fmt.Sprintf("doctor-%d-%s", doctorID, h.fileTag), file, size, headers)
		if err != nil {
			www.InternalServerError(w, r, err)
		}

		if err := h.dataAPI.UpdateDoctorAttributes(doctorID, map[string]string{h.attrName: fileID}); err != nil {
			www.InternalServerError(w, r, err)
			return
		}

		h.redirectToNextStep(w, r)
		return
	}

	www.TemplateResponse(w, http.StatusOK, uploadTemplate, &www.BaseTemplateContext{
		Title: template.HTML(template.HTMLEscapeString(h.title) + " | Doctor Registration | Spruce"),
		SubContext: &uploadTemplateContext{
			Title: h.title,
		},
	})
}

func (h *uploadHandler) redirectToNextStep(w http.ResponseWriter, r *http.Request) {
	if u, err := h.router.Get(h.nextURL).URLPath(); err != nil {
		www.InternalServerError(w, r, err)
	} else {
		http.Redirect(w, r, u.String(), http.StatusSeeOther)
	}
}
