package app_event

import (
	"net/http"

	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/dispatch"
)

type eventHandler struct {
}

type EventRequestData struct {
	Action     string `json:"action"`
	Resource   string `json:"resource"`
	ResourceId int64  `json:"resource_id,string"`
}

// NewHandler returns a handler that dispatches events
// received from the client for anyone interested in ClientEvents. The idea is to create a generic
// way for the client to send events of what the user id doing
// ("viewing", "updating", "deleting", etc. a resource of a particular id) for the server to appropriately
// act on the event
func NewHandler() *eventHandler {
	return &eventHandler{}
}

func (e *eventHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != apiservice.HTTP_POST {
		http.NotFound(w, r)
		return
	}

	requestData := &EventRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	dispatch.Default.Publish(&AppEvent{
		AccountId:  apiservice.GetContext(r).AccountId,
		Role:       apiservice.GetContext(r).Role,
		Resource:   requestData.Resource,
		ResourceId: requestData.ResourceId,
		Action:     requestData.Action,
	})

	apiservice.WriteJSONSuccess(w)
}
