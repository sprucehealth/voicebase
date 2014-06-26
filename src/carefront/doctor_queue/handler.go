package doctor_queue

import (
	"carefront/api"
	"carefront/apiservice"
	"carefront/app_url"
	"net/http"
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

func NewQueueHandler(dataAPI api.DataAPI) *queueHandler {
	return &queueHandler{
		dataAPI: dataAPI,
	}
}

type DoctorQueueRequestData struct {
	State string `schema:"state"`
}

func (d *queueHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != apiservice.HTTP_GET {
		http.NotFound(w, r)
		return
	}

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
		addAuthUrl = true
		queueItems, err = d.dataAPI.GetElligibleItemsInUnclaimedQueue(doctorId)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	case stateCompleted:
		queueItems, err = d.dataAPI.GetCompletedItemsInDoctorQueue(doctorId)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	default:
		apiservice.WriteValidationError("Unexpected state value. Can only be local, global or completed", w, r)
		return
	}

	feedItems := make([]*DisplayFeedItem, len(queueItems))
	for i, doctorQueueItem := range queueItems {
		doctorQueueItem.PositionInQueue = i
		feedItems[i], err = converQueueItemToDisplayFeedItem(d.dataAPI, doctorQueueItem)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		if addAuthUrl {
			feedItems[i].AuthUrl = app_url.ClaimPatientCaseAction(doctorQueueItem.PatientCaseId)
		}
	}
	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorQueueItemsResponseData{Items: feedItems})
}
