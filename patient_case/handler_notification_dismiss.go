package patient_case

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
)

type dismissNotificationHandler struct {
	dataAPI api.DataAPI
}

type dismissNotificationRequestData struct {
	NotificationId int64 `json:"notification_id,string"`
}

func NewDismissNotificationHandler(dataAPI api.DataAPI) *dismissNotificationHandler {
	return &dismissNotificationHandler{
		dataAPI: dataAPI,
	}
}

func (d *dismissNotificationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != apiservice.HTTP_PUT {
		http.NotFound(w, r)
		return
	}

	requestData := &dismissNotificationRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		apiservice.WriteError(err, w, r)
		return
	} else if requestData.NotificationId == 0 {
		apiservice.WriteValidationError("notification_id not specified", w, r)
		return
	}

	if apiservice.GetContext(r).Role != api.PATIENT_ROLE {
		apiservice.WriteValidationError("only patient can dismiss case notification", w, r)
		return
	}

	if err := d.dataAPI.DeleteCaseNotificationBasedOnId(requestData.NotificationId); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSONSuccess(w)
}
