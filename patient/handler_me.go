package patient

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/auth"
	"github.com/sprucehealth/backend/libs/dispatch"
)

type meHandler struct {
	dataAPI    api.DataAPI
	dispatcher *dispatch.Dispatcher
}

func NewMeHandler(dataAPI api.DataAPI, dispatcher *dispatch.Dispatcher) http.Handler {
	return &meHandler{
		dataAPI:    dataAPI,
		dispatcher: dispatcher,
	}
}

func (m *meHandler) IsAuthorized(r *http.Request) (bool, error) {
	if apiservice.GetContext(r).Role != api.PATIENT_ROLE {
		return false, nil
	}

	return true, nil
}

func (m *meHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != apiservice.HTTP_GET {
		http.NotFound(w, r)
		return
	}

	patient, err := m.dataAPI.GetPatientFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// ignoring the error because
	token, _ := apiservice.GetAuthTokenFromHeader(r)
	apiservice.WriteJSON(w, map[string]interface{}{
		"patient": patient,
		"token":   token,
	})

	headers := apiservice.ExtractSpruceHeaders(r)
	m.dispatcher.PublishAsync(&auth.AuthenticatedEvent{
		AccountID:     patient.AccountId.Int64(),
		SpruceHeaders: headers,
	})
}
