package dronboard

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/www"
)

func NewUploadLicenseHandler(router *mux.Router, dataAPI api.DataAPI, store storage.Store) http.Handler {
	return www.SupportedMethodsHandler(&uploadHandler{
		router:   router,
		dataAPI:  dataAPI,
		store:    store,
		attrName: api.AttrDriversLicenseFile,
		fileTag:  "dl",
		title:    "Upload Driver's License",
		nextURL:  "doctor-register-engagement",
	}, []string{"GET", "POST"})
}
