package patient

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/auth"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/httputil"
)

type meHandler struct {
	dataAPI    api.DataAPI
	dispatcher *dispatch.Dispatcher
}

func NewMeHandler(dataAPI api.DataAPI, dispatcher *dispatch.Dispatcher) http.Handler {
	return httputil.SupportedMethods(
		apiservice.AuthorizationRequired(
			&meHandler{
				dataAPI:    dataAPI,
				dispatcher: dispatcher,
			}), []string{"GET"})
}

func (m *meHandler) IsAuthorized(r *http.Request) (bool, error) {
	if apiservice.GetContext(r).Role != api.PATIENT_ROLE {
		return false, nil
	}

	return true, nil
}

func (m *meHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	patient, err := m.dataAPI.GetPatientFromAccountID(apiservice.GetContext(r).AccountID)
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
		AccountID:     patient.AccountID.Int64(),
		SpruceHeaders: headers,
	})
}
