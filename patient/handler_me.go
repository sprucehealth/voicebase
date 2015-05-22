package patient

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/auth"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
)

type meHandler struct {
	dataAPI    api.DataAPI
	dispatcher *dispatch.Dispatcher
}

type meResponse struct {
	Patient       *common.Patient `json:"patient"`
	Token         string          `json:"token"`
	ActionsNeeded []*ActionNeeded `json:"actions_needed,omitempty"`
}

func NewMeHandler(dataAPI api.DataAPI, dispatcher *dispatch.Dispatcher) http.Handler {
	return httputil.SupportedMethods(
		apiservice.AuthorizationRequired(
			&meHandler{
				dataAPI:    dataAPI,
				dispatcher: dispatcher,
			}), httputil.Get)
}

func (m *meHandler) IsAuthorized(r *http.Request) (bool, error) {
	if apiservice.GetContext(r).Role != api.RolePatient {
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

	token, err := apiservice.GetAuthTokenFromHeader(r)
	if err != nil {
		// Should never fail but if it does it's a very bad thing since it
		// should have been checked before we even got this far.
		golog.Errorf("Failed to get auth token when already authenticated: %s", err)
	}
	res := &meResponse{
		Patient: patient,
		Token:   token,
	}

	if showFeedback(m.dataAPI, patient.PatientID.Int64()) {
		res.ActionsNeeded = append(res.ActionsNeeded, &ActionNeeded{Type: actionNeededSimpleFeedbackPrompt})
	}

	httputil.JSONResponse(w, http.StatusOK, res)

	headers := apiservice.ExtractSpruceHeaders(r)
	m.dispatcher.PublishAsync(&auth.AuthenticatedEvent{
		AccountID:     patient.AccountID.Int64(),
		SpruceHeaders: headers,
	})
}
