package app_event

import (
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/dispatch"
)

type eventHandler struct {
}

func NewHandler() *eventHandler {
	return &eventHandler{}
}

func (e *eventHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != apiservice.HTTP_POST {
		http.NotFound(w, r)
		return
	}

	if err := r.ParseForm(); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	var resourceId int64
	var err error
	if r.FormValue("resource_id") != "" {
		resourceId, err = strconv.ParseInt(r.FormValue("resource_id"), 10, 64)
		if err != nil {
			apiservice.WriteValidationError(err.Error(), w, r)
			return
		}
	}

	dispatch.Default.Publish(&AppEvent{
		AccountId:  apiservice.GetContext(r).AccountId,
		Role:       apiservice.GetContext(r).Role,
		Resource:   r.FormValue("resource"),
		ResourceId: resourceId,
		Action:     r.FormValue("action"),
	})

	apiservice.WriteJSONSuccess(w)
}
