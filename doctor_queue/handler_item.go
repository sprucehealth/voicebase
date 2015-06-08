package doctor_queue

import (
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/libs/httputil"
)

type itemHandler struct {
	dataAPI api.DataAPI
}

type itemRequest struct {
	Action string `json:"action"`
	ID     string `json:"id"`
}

func NewItemHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.NoAuthorizationRequired(
				&itemHandler{
					dataAPI: dataAPI,
				}), []string{api.RoleCC}),
		httputil.Put)
}

func (h *itemHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var rd itemRequest
	if err := apiservice.DecodeRequestData(&rd, r); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	} else if rd.Action != "remove" {
		apiservice.WriteValidationError(fmt.Sprintf("%s action not supported", rd.Action), w, r)
		return
	}

	eventType, status, itemID, doctorID, err := queueItemPartsFromID(rd.ID)
	if err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	switch eventType {
	case api.DQEventTypeCaseAssignment, api.DQEventTypeCaseMessage:
		if status != api.DQItemStatusPending {
			apiservice.WriteAccessNotAllowedError(w, r)
			return
		}
	case api.DQEventTypePatientVisit:
		if status != api.DQItemStatusPending && status != api.DQItemStatusOngoing {
			apiservice.WriteAccessNotAllowedError(w, r)
			return
		}
	default:
		apiservice.WriteAccessNotAllowedError(w, r)
		return
	}

	updates := []*api.DoctorQueueUpdate{
		{
			Action: api.DQActionRemove,
			QueueItem: &api.DoctorQueueItem{
				EventType: eventType,
				Status:    status,
				DoctorID:  doctorID,
				ItemID:    itemID,
			},
		},
	}
	if eventType == api.DQEventTypePatientVisit {
		accountID := apiservice.GetContext(r).AccountID
		cc, err := h.dataAPI.GetDoctorFromAccountID(accountID)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		visit, err := h.dataAPI.GetPatientVisitFromID(itemID)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		patient, err := h.dataAPI.Patient(visit.PatientID.Int64(), true)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		updates = append(updates, &api.DoctorQueueUpdate{
			Action: api.DQActionInsert,
			QueueItem: &api.DoctorQueueItem{
				EventType:        eventType,
				Status:           api.DQItemStatusRemoved,
				DoctorID:         cc.ID.Int64(),
				ItemID:           itemID,
				PatientID:        visit.PatientID.Int64(),
				Description:      fmt.Sprintf("%s removed visit for %s %s from queue", cc.ShortDisplayName, patient.FirstName, patient.LastName),
				ShortDescription: "Visit removed from queue",
				ActionURL:        app_url.ViewPatientVisitInfoAction(visit.PatientID.Int64(), itemID, visit.PatientCaseID.Int64()),
			},
		})
	}

	if err := h.dataAPI.UpdateDoctorQueue(updates); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSONSuccess(w)
}
