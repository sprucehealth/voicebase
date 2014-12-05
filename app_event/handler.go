package app_event

import (
	"net/http"

	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/httputil"
)

type eventHandler struct {
	dispatcher *dispatch.Dispatcher
}

type EventRequestData struct {
	Action     string `json:"action"`
	Resource   string `json:"resource"`
	ResourceId int64  `json:"resource_id,string"`
}

// NewHandler returns a handler that dispatches events
// received from the client for anyone interested in ClientEvents. The idea is to create a generic
// way for the client to send events of what the user is doing
// ("viewing", "updating", "deleting", etc. a resource) for the server to appropriately
// act on the event
func NewHandler(dispatcher *dispatch.Dispatcher) http.Handler {
	return httputil.SupportedMethods(apiservice.NoAuthorizationRequired(&eventHandler{
		dispatcher: dispatcher,
	}), []string{"POST"})
}

func (e *eventHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestData := &EventRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	e.dispatcher.Publish(&AppEvent{
		AccountId:  apiservice.GetContext(r).AccountId,
		Role:       apiservice.GetContext(r).Role,
		Resource:   requestData.Resource,
		ResourceId: requestData.ResourceId,
		Action:     requestData.Action,
	})

	apiservice.WriteJSONSuccess(w)
}
