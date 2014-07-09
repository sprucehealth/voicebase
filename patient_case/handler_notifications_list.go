package patient_case

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
)

type notificationsListHandler struct {
	dataAPI api.DataAPI
}

type notificationsListRequestData struct {
	PatientCaseId int64 `schema:"case_id"`
}

type notificationsListResponseData struct {
	Items []common.ClientView `json:"items"`
}

func NewNotificationsListHandler(dataAPI api.DataAPI) http.Handler {
	return &notificationsListHandler{
		dataAPI: dataAPI,
	}
}

func (n *notificationsListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != apiservice.HTTP_GET {
		http.NotFound(w, r)
		return
	}

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
		nViewItems[i], err = notificationItem.Data.(notification).makeCaseNotificationView(n.dataAPI, notificationItem.Id)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	apiservice.WriteJSON(w, &notificationsListResponseData{Items: nViewItems})
}
