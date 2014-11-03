package home

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/www"
)

type formsAPIHandler struct {
	dataAPI api.DataAPI
}

func NewFormsAPIHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(&formsAPIHandler{
		dataAPI: dataAPI,
	}, []string{"POST"})
}

func (h *formsAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	formName := mux.Vars(r)["form"]
	formType := common.Forms[formName]
	if formType == nil {
		golog.Warningf("Form %s not found", formName)
		www.APINotFound(w, r)
		return
	}
	form, ok := reflect.New(formType).Interface().(api.Form)
	if !ok {
		www.APIInternalError(w, r, fmt.Errorf("Form type %s does not conform to the Form interface", formName))
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&form); err != nil {
		www.APIInternalError(w, r, err)
		return
	}
	requestID := httputil.RequestID(r)
	// if err := h.dataAPI.RecordNotifyMe(req.Email, req.State, req.Platform, requestID); err != nil {
	if err := h.dataAPI.RecordForm(form, "home", requestID); err != nil {
		www.APIInternalError(w, r, err)
		return
	}
	www.JSONResponse(w, r, http.StatusOK, true)
}
