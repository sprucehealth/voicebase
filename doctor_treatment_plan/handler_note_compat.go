package doctor_treatment_plan

import (
	"net/http"
	"strconv"

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
				}, []string{api.RoleDoctor})),
		[]string{"GET", "PUT"})
}

func (h *savedNoteCompatibilityHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		// Ignore PUT
		apiservice.WriteJSONSuccess(w)
		return
	}

	ctx := apiservice.GetContext(r)
	doctorID, err := h.dataAPI.GetDoctorIDFromAccountID(ctx.AccountID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	var treatmentPlanID int64
	if tpIDStr := r.FormValue("treatment_plan_id"); tpIDStr != "" {
		treatmentPlanID, err = strconv.ParseInt(tpIDStr, 10, 64)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	var msg string
	if treatmentPlanID != 0 {
		msg, err = h.dataAPI.GetTreatmentPlanNote(treatmentPlanID)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	if msg == "" {
		msg, err = h.dataAPI.GetSavedDoctorNote(doctorID)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	httputil.JSONResponse(w, http.StatusOK, &struct {
		Message string `json:"message"`
	}{
		Message: msg,
	})
}
