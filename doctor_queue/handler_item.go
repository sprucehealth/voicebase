package doctor_queue

import (
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
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
				}), []string{api.RoleMA}), httputil.Put)
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
	default:
		apiservice.WriteAccessNotAllowedError(w, r)
		return
	}
	if status != api.DQItemStatusPending {
		apiservice.WriteAccessNotAllowedError(w, r)
		return
	}

	if err := h.dataAPI.UpdateDoctorQueue([]*api.DoctorQueueUpdate{
		{
			Action: api.DQActionRemove,
			QueueItem: &api.DoctorQueueItem{
				EventType: eventType,
				Status:    status,
				DoctorID:  doctorID,
				ItemID:    itemID,
			},
		},
	}); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSONSuccess(w)
}
