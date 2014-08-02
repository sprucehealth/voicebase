package doctor_queue

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/libs/golog"
)

const (
	stateCompleted = "completed"
	stateLocal     = "local"
	stateGlobal    = "global"
)

type queueHandler struct {
	dataAPI api.DataAPI
}

type DoctorQueueItemsResponseData struct {
	Items []*DisplayFeedItem `json:"items"`
}

func NewQueueHandler(dataAPI api.DataAPI) http.Handler {
	return apiservice.SupportedMethods(&queueHandler{
		dataAPI: dataAPI,
	}, []string{apiservice.HTTP_GET})
}

type DoctorQueueRequestData struct {
	State string `schema:"state"`
}

func (d *queueHandler) IsAuthorized(r *http.Request) (bool, error) {
	switch apiservice.GetContext(r).Role {
	case api.DOCTOR_ROLE, api.MA_ROLE:
	default:
		return false, apiservice.NewAccessForbiddenError()
	}

	return true, nil
}

func (d *queueHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestData := &DoctorQueueRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		apiservice.WriteValidationError("Unable to parse input parameters", w, r)
		return
	} else if requestData.State == "" {
		apiservice.WriteValidationError("State (local,global,completed) required", w, r)
		return
	}

	doctorId, err := d.dataAPI.GetDoctorIdFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor id from account id:"+err.Error())
		return
	}

	// only add auth url for items in global queue so that
	// the doctor can first be granted acess to the case before opening the case
	var addAuthUrl bool
	var queueItems []*api.DoctorQueueItem
	switch requestData.State {
	case stateLocal:
		queueItems, err = d.dataAPI.GetPendingItemsInDoctorQueue(doctorId)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	case stateGlobal:
		if apiservice.GetContext(r).Role == api.MA_ROLE {
			queueItems, err = d.dataAPI.GetPendingItemsForClinic()
			if err != nil {
				apiservice.WriteError(err, w, r)
				return
			}
		} else {
			addAuthUrl = true
			queueItems, err = d.dataAPI.GetElligibleItemsInUnclaimedQueue(doctorId)
			if err != nil {
				apiservice.WriteError(err, w, r)
				return
			}
		}
	case stateCompleted:
		if apiservice.GetContext(r).Role == api.MA_ROLE {
			queueItems, err = d.dataAPI.GetCompletedItemsForClinic()
			if err != nil {
				apiservice.WriteError(err, w, r)
				return
			}
		} else {
			queueItems, err = d.dataAPI.GetCompletedItemsInDoctorQueue(doctorId)
			if err != nil {
				apiservice.WriteError(err, w, r)
				return
			}
		}
	default:
		apiservice.WriteValidationError("Unexpected state value. Can only be local, global or completed", w, r)
		return
	}

	feedItems := make([]*DisplayFeedItem, 0, len(queueItems))
	for i, doctorQueueItem := range queueItems {
		doctorQueueItem.PositionInQueue = i
		feedItem, err := converQueueItemToDisplayFeedItem(d.dataAPI, doctorQueueItem)
		if err != nil {
			golog.Errorf("Unable to convert item (Id: %d, EventType: %s, Status: %s, ItemId: %d) into display item", doctorQueueItem.Id,
				doctorQueueItem.EventType, doctorQueueItem.Status, doctorQueueItem.ItemId)
			continue
		}
		if addAuthUrl {
			feedItem.AuthUrl = app_url.ClaimPatientCaseAction(doctorQueueItem.PatientCaseId)
		}

		feedItems = append(feedItems, feedItem)
	}
	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorQueueItemsResponseData{Items: feedItems})
}
