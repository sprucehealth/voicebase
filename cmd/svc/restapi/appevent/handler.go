package appevent

import (
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/apiservice"
	"github.com/sprucehealth/backend/cmd/svc/restapi/internal/httputil"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"
)

type eventHandler struct {
	dataAPI    api.DataAPI
	dispatcher *dispatch.Dispatcher
}

type EventRequestData struct {
	Action     string `json:"action"`
	Resource   string `json:"resource"`
	ResourceID int64  `json:"resource_id,string"`
}

// NewHandler returns a handler that dispatches events received from the
// client for anyone interested in ClientEvents. The idea is to create a
// generic way for the client to send events of what the user is doing
// ("viewing", "updating", "deleting", etc. a resource) for the server to
// appropriately act on the event
func NewHandler(dataAPI api.DataAPI, dispatcher *dispatch.Dispatcher) http.Handler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.NoAuthorizationRequired(&eventHandler{
				dataAPI:    dataAPI,
				dispatcher: dispatcher,
			}),
			api.RolePatient, api.RoleCC, api.RoleDoctor),
		httputil.Post)
}

func (h *eventHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	req := &EventRequestData{}
	if err := apiservice.DecodeRequestData(req, r); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	account := apiservice.MustCtxAccount(r.Context())

	// Make sure the requesting account has access to the resource

	var err error
	var caseID int64

	if account.Role != api.RoleCC {
		allowed := false

		switch req.Resource {
		case "treatment_plan":
			// resource_id is treatment plan ID
			if account.Role == api.RolePatient {
				p, err := h.dataAPI.GetPatientFromTreatmentPlanID(req.ResourceID)
				if api.IsErrNotFound(err) {
					golog.Warningf("appevent action %s from account %d for resource %s:%d: treatment plan not found",
						req.Action, account.ID, req.Resource, req.ResourceID)
					apiservice.WriteResourceNotFoundError("Treatment plan not found", w, r)
					return
				} else if err != nil {
					apiservice.WriteError(err, w, r)
					return
				}
				allowed = p.AccountID.Int64() == account.ID
			} else {
				caseID, err = h.dataAPI.CaseIDForTreatmentPlan(req.ResourceID)
				if err != nil {
					apiservice.WriteError(err, w, r)
					return
				}
			}
		case "case_message":
			// resource_id is message ID
			caseID, err = h.dataAPI.GetCaseIDFromMessageID(req.ResourceID)
			if api.IsErrNotFound(err) {
				golog.Warningf("appevent action %s from account %d for resource %s:%d: message not found",
					req.Action, account.ID, req.Resource, req.ResourceID)
				apiservice.WriteResourceNotFoundError("Message not found", w, r)
				return
			} else if err != nil {
				apiservice.WriteError(err, w, r)
				return
			}
		case "all_case_messages":
			// resource_id is case ID
			caseID = req.ResourceID
		}

		if !allowed && caseID != 0 {
			switch account.Role {
			case api.RolePatient:
				c, err := h.dataAPI.GetPatientCaseFromID(caseID)
				if api.IsErrNotFound(err) {
					golog.Warningf("appevent action %s from account %d for resource %s:%d: case not found",
						req.Action, account.ID, req.Resource, req.ResourceID)
					apiservice.WriteResourceNotFoundError("Case not found", w, r)
					return
				} else if err != nil {
					apiservice.WriteError(err, w, r)
					return
				}
				p, err := h.dataAPI.Patient(c.PatientID, true)
				if err != nil {
					apiservice.WriteError(err, w, r)
					return
				}
				allowed = p.AccountID.Int64() == account.ID
			case api.RoleCC, api.RoleDoctor:
				doctorID, err := h.dataAPI.GetDoctorIDFromAccountID(account.ID)
				if err != nil {
					apiservice.WriteError(err, w, r)
					return
				}
				// Only checking read access since we don't know what the event will be used for at this point.
				allowed, err = apiservice.DoctorHasAccessToCase(r.Context(), doctorID, caseID, account.Role, apiservice.ReadAccessRequired, h.dataAPI)
				if err != nil {
					apiservice.WriteError(err, w, r)
					return
				}
			}
		}

		if !allowed {
			golog.Warningf("app_event action %s from account %d for resource %s:%d not allowed", req.Action, account.ID, req.Resource, req.ResourceID)
			apiservice.WriteAccessNotAllowedError(w, r)
			return
		}
	}

	h.dispatcher.Publish(&AppEvent{
		AccountID:  account.ID,
		Role:       account.Role,
		Resource:   req.Resource,
		ResourceID: req.ResourceID,
		Action:     req.Action,
	})

	apiservice.WriteJSONSuccess(w)
}
