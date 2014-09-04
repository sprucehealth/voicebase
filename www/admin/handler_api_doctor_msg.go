package admin

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/context"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/www"
)

type doctorSavedMessageHandler struct {
	dataAPI api.DataAPI
}

type doctorSavedMessage struct {
	Message string `json:"message"`
}

func NewDoctorSavedMessageHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(&doctorSavedMessageHandler{
		dataAPI: dataAPI,
	}, []string{"GET", "PUT"})
}

func (h *doctorSavedMessageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	account := context.Get(r, www.CKAccount).(*common.Account)

	doctorID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	if r.Method == "PUT" {
		audit.LogAction(account.ID, "AdminAPI", "UpdateDoctorSavedMessage", map[string]interface{}{"doctor_id": doctorID})

		var req doctorSavedMessage
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			www.APIInternalError(w, r, err)
			return
		}

		if err := h.dataAPI.SetSavedMessageForDoctor(doctorID, req.Message); err != nil {
			www.APIInternalError(w, r, err)
			return
		}

		www.JSONResponse(w, r, http.StatusOK, true)
		return
	}

	audit.LogAction(account.ID, "AdminAPI", "GetDoctorSavedMessage", map[string]interface{}{"doctor_id": doctorID})
	msg, err := h.dataAPI.GetSavedMessageForDoctor(doctorID)
	if err == api.NoRowsError {
		msg = ""
	} else if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	www.JSONResponse(w, r, http.StatusOK, &doctorSavedMessage{
		Message: msg,
	})
}
