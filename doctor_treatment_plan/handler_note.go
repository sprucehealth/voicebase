package doctor_treatment_plan

import (
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/httputil"
)

type savedNoteHandler struct {
	dataAPI    api.DataAPI
	dispatcher *dispatch.Dispatcher
}

type DoctorSavedNoteRequestData struct {
	TreatmentPlanID int64  `json:"treatment_plan_id,string" schema:"treatment_plan_id"`
	Message         string `json:"message"`
}

func NewSavedNoteHandler(dataAPI api.DataAPI, dispatcher *dispatch.Dispatcher) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.NoAuthorizationRequired(
			apiservice.SupportedRoles(
				&savedNoteHandler{
					dataAPI:    dataAPI,
					dispatcher: dispatcher,
				}, api.RoleDoctor)),
		httputil.Put)
}

func (h *savedNoteHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case httputil.Put:
		h.put(ctx, w, r)
	default:
		httputil.SupportedMethodsResponse(w, r, []string{"PUT"})
	}
}

func (h *savedNoteHandler) put(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	account := apiservice.MustCtxAccount(ctx)
	doctorID, err := h.dataAPI.GetDoctorIDFromAccountID(account.ID)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	var req DoctorSavedNoteRequestData
	if err := apiservice.DecodeRequestData(&req, r); err != nil {
		apiservice.WriteValidationError(ctx, err.Error(), w, r)
		return
	}

	// Update message for a treatment plan
	if err := h.dataAPI.SetTreatmentPlanNote(doctorID, req.TreatmentPlanID, req.Message); err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	h.dispatcher.Publish(&TreatmentPlanUpdatedEvent{
		TreatmentPlanID: req.TreatmentPlanID,
		DoctorID:        doctorID,
		SectionUpdated:  NoteSection,
	})

	apiservice.WriteJSONSuccess(w)
}
