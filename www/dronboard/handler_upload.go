package dronboard

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/www"
)

const maxMemory = 5 * 1024 * 1024

type uploadHandler struct {
	router   *mux.Router
	dataAPI  api.DataAPI
	authAPI  api.AuthAPI
	store    storage.Store
	template *template.Template
	attrName string
	fileTag  string
	title    string
	subtitle string
	nextURL  string
	required bool
}

func newUploadCVHandler(router *mux.Router, dataAPI api.DataAPI, store storage.Store, templateLoader *www.TemplateLoader) httputil.ContextHandler {
	return httputil.SupportedMethods(&uploadHandler{
		router:   router,
		dataAPI:  dataAPI,
		store:    store,
		attrName: api.AttrCVFile,
		fileTag:  "cv",
		title:    "Upload CV / Résumé",
		nextURL:  "doctor-register-upload-license",
		required: true,
		template: templateLoader.MustLoadTemplate("dronboard/upload.html", "dronboard/base.html", nil),
	}, httputil.Get, httputil.Post)
}

func newUploadLicenseHandler(router *mux.Router, dataAPI api.DataAPI, store storage.Store, templateLoader *www.TemplateLoader) httputil.ContextHandler {
	return httputil.SupportedMethods(&uploadHandler{
		router:   router,
		dataAPI:  dataAPI,
		store:    store,
		attrName: api.AttrDriversLicenseFile,
		fileTag:  "dl",
		title:    "Upload Image of Driver's License",
		subtitle: "Used as part of identity verification",
		nextURL:  "doctor-register-insurance",
		required: true,
		template: templateLoader.MustLoadTemplate("dronboard/upload.html", "dronboard/base.html", nil),
	}, httputil.Get, httputil.Post)
}

func newUploadClaimsHistoryHandler(router *mux.Router, dataAPI api.DataAPI, store storage.Store, templateLoader *www.TemplateLoader) httputil.ContextHandler {
	return httputil.SupportedMethods(&uploadHandler{
		router:   router,
		dataAPI:  dataAPI,
		store:    store,
		attrName: api.AttrClaimsHistoryFile,
		fileTag:  "claimshistory",
		title:    "Upload Claims History",
		subtitle: "You may skip this step and instead permit us to obtain this information on your behalf from your previous malpractice insurance carriers.",
		nextURL:  "doctor-register-claims-history",
		required: false,
		template: templateLoader.MustLoadTemplate("dronboard/upload.html", "dronboard/base.html", nil),
	}, httputil.Get, httputil.Post)
}

func (h *uploadHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	u, err := h.router.Get(h.nextURL).URLPath()
	if err != nil {
		www.InternalServerError(w, r, err)
		return
	}
	nextURL := u.String()

	account := www.MustCtxAccount(ctx)
	doctorID, err := h.dataAPI.GetDoctorIDFromAccountID(account.ID)
	if err != nil {
		www.InternalServerError(w, r, err)
		return
	}

	// See if the doctor already uploaded the file. If so then skip this step
	attr, err := h.dataAPI.DoctorAttributes(doctorID, []string{h.attrName})
	if err != nil {
		www.InternalServerError(w, r, err)
		return
	}
	if attr[h.attrName] != "" {
		http.Redirect(w, r, nextURL, http.StatusSeeOther)
		return
	}

	var errorMsg string
	if r.Method == "POST" {
		if err := r.ParseMultipartForm(maxMemory); err != nil {
			www.InternalServerError(w, r, err)
			return
		}

		file, fileHandler, err := r.FormFile("File")
		switch err {
		default:
			www.InternalServerError(w, r, err)
			return
		case http.ErrMissingFile:
			if h.required {
				errorMsg = "File is required"
			}
		case nil:
			defer file.Close()

			size, err := common.SeekerSize(file)
			if err != nil {
				www.InternalServerError(w, r, err)
				return
			}

			meta := map[string]string{
				"X-Amz-Meta-Original-Name": fileHandler.Filename,
			}
			fileID, err := h.store.PutReader(fmt.Sprintf("doctor-%d-%s", doctorID, h.fileTag), file, size, "", meta)
			if err != nil {
				www.InternalServerError(w, r, err)
				return
			}

			if err := h.dataAPI.UpdateDoctorAttributes(doctorID, map[string]string{h.attrName: fileID}); err != nil {
				www.InternalServerError(w, r, err)
				return
			}
		}

		if errorMsg == "" {
			http.Redirect(w, r, nextURL, http.StatusSeeOther)
			return
		}
	}

	www.TemplateResponse(w, http.StatusOK, h.template, &www.BaseTemplateContext{
		Title: template.HTML(template.HTMLEscapeString(h.title) + " | Doctor Registration | Spruce"),
		SubContext: &struct {
			Title    string
			Subtitle string
			Required bool
			Error    string
			NextURL  string
		}{
			Title:    h.title,
			Subtitle: h.subtitle,
			Required: h.required,
			Error:    errorMsg,
			NextURL:  nextURL,
		},
	})
}
