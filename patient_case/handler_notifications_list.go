package patient_case

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
)

type notificationsListHandler struct {
	dataAPI   api.DataAPI
	apiDomain string
}

type notificationsListRequestData struct {
	PatientCaseId int64 `schema:"case_id"`
}

type notificationsListResponseData struct {
	Items []common.ClientView `json:"items"`
}

func NewNotificationsListHandler(dataAPI api.DataAPI, apiDomain string) http.Handler {
	return &notificationsListHandler{
		dataAPI:   dataAPI,
		apiDomain: apiDomain,
	}
}

func (n *notificationsListHandler) IsAuthorized(r *http.Request) (bool, error) {
	if r.Method != apiservice.HTTP_GET {
		return false, apiservice.NewResourceNotFoundError("", r)
	}

	if apiservice.GetContext(r).Role != api.PATIENT_ROLE {
		return false, apiservice.NewAccessForbiddenError()
	}

	return true, nil
}

func (n *notificationsListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestData := &notificationsListRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		apiservice.WriteError(err, w, r)
		return
	} else if requestData.PatientCaseId == 0 {
		apiservice.WriteValidationError("case_id must be specified", w, r)
		return
	}

	notificationItems, err := n.dataAPI.GetNotificationsForCase(requestData.PatientCaseId, notifyTypes)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	nViewItems := make([]common.ClientView, len(notificationItems))
	for i, notificationItem := range notificationItems {
		nViewItems[i], err = notificationItem.Data.(notification).makeCaseNotificationView(n.dataAPI, n.apiDomain, notificationItem)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	apiservice.WriteJSON(w, &notificationsListResponseData{Items: nViewItems})
}
