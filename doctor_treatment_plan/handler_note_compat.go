package doctor_treatment_plan

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/httputil"
)

type savedNoteCompatibilityHandler struct {
	dataAPI api.DataAPI
}

func NewSavedNoteCompatibilityHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.NoAuthorizationRequired(
			apiservice.SupportedRoles(
				&savedNoteCompatibilityHandler{
					dataAPI: dataAPI,
				}, []string{api.DOCTOR_ROLE})),
		[]string{"GET", "PUT"})
}

func (h *savedNoteCompatibilityHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		// Ignore PUT
		apiservice.WriteJSONSuccess(w)
		return
	}

	ctx := apiservice.GetContext(r)
	doctorID, err := h.dataAPI.GetDoctorIdFromAccountId(ctx.AccountId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	msg, err := h.dataAPI.GetSavedDoctorNote(doctorID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSON(w, &struct {
		Message string `json:"message"`
	}{
		Message: msg,
	})
}
