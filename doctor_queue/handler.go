package doctor_queue

import (
	"net/http"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/libs/httputil"
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

type DisplayFeedItem struct {
	ID           int64                 `json:"id,string,omitempty"`
	Title        string                `json:"title"`
	Subtitle     string                `json:"subtitle,omitempty"`
	Timestamp    *time.Time            `json:"timestamp,omitempty"`
	ImageURL     *app_url.SpruceAsset  `json:"image_url,omitempty"`
	ActionURL    *app_url.SpruceAction `json:"action_url,omitempty"`
	AuthURL      *app_url.SpruceAction `json:"auth_url,omitempty"`
	DisplayTypes []string              `json:"display_types,omitempty"`
}

func NewQueueHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.NoAuthorizationRequired(
				&queueHandler{
					dataAPI: dataAPI,
				}), []string{api.RoleDoctor, api.RoleMA}),
		httputil.Get)
}

type DoctorQueueRequestData struct {
	State string `schema:"state"`
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

	doctorID, err := d.dataAPI.GetDoctorIDFromAccountID(apiservice.GetContext(r).AccountID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// only add auth url for items in global queue so that
	// the doctor can first be granted acess to the case before opening the case
	var addAuthURL bool
	var queueItems []*api.DoctorQueueItem
	switch requestData.State {
	case stateLocal:
		queueItems, err = d.dataAPI.GetPendingItemsInDoctorQueue(doctorID)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	case stateGlobal:
		if apiservice.GetContext(r).Role == api.RoleMA {
			queueItems, err = d.dataAPI.GetPendingItemsForClinic()
			if err != nil {
				apiservice.WriteError(err, w, r)
				return
			}
		} else {
			addAuthURL = true
			queueItems, err = d.dataAPI.GetElligibleItemsInUnclaimedQueue(doctorID)
			if err != nil && !api.IsErrNotFound(err) {
				apiservice.WriteError(err, w, r)
				return
			}
		}
	case stateCompleted:
		if apiservice.GetContext(r).Role == api.RoleMA {
			queueItems, err = d.dataAPI.GetCompletedItemsForClinic()
			if err != nil {
				apiservice.WriteError(err, w, r)
				return
			}
		} else {
			queueItems, err = d.dataAPI.GetCompletedItemsInDoctorQueue(doctorID)
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
	for _, doctorQueueItem := range queueItems {

		feedItem := &DisplayFeedItem{
			ID:           doctorQueueItem.ID,
			Title:        doctorQueueItem.Description,
			ActionURL:    doctorQueueItem.ActionURL,
			DisplayTypes: []string{api.DisplayTypeTitleSubtitleActionable},
		}

		if !doctorQueueItem.EnqueueDate.IsZero() {
			feedItem.Timestamp = &doctorQueueItem.EnqueueDate
		}

		if addAuthURL {
			feedItem.AuthURL = app_url.ClaimPatientCaseAction(doctorQueueItem.PatientCaseID)
		}

		feedItems = append(feedItems, feedItem)
	}
	httputil.JSONResponse(w, http.StatusOK, &DoctorQueueItemsResponseData{Items: feedItems})
}
