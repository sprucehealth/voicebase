package patient

import (
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/apiservice"
	"github.com/sprucehealth/backend/cmd/svc/restapi/auth"
	"github.com/sprucehealth/backend/cmd/svc/restapi/feedback"
	"github.com/sprucehealth/backend/cmd/svc/restapi/responses"
	"github.com/sprucehealth/backend/device"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"golang.org/x/net/context"
)

type meHandler struct {
	dataAPI        api.DataAPI
	feedbackClient feedback.DAL
	dispatcher     *dispatch.Dispatcher
}

type meResponse struct {
	Patient       *responses.Patient `json:"patient"`
	Token         string             `json:"token"`
	ActionsNeeded []*ActionNeeded    `json:"actions_needed,omitempty"`
}

// NewMeHandler exposes a handler to get patient information for provided token.
func NewMeHandler(dataAPI api.DataAPI, feedbackClient feedback.DAL, dispatcher *dispatch.Dispatcher) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.NoAuthorizationRequired(
				&meHandler{
					dataAPI:        dataAPI,
					feedbackClient: feedbackClient,
					dispatcher:     dispatcher,
				}),
			api.RolePatient),
		httputil.Get)
}

func (m *meHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	patient, err := m.dataAPI.GetPatientFromAccountID(apiservice.MustCtxAccount(ctx).ID)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	token, err := apiservice.GetAuthTokenFromHeader(r)
	if err != nil {
		// Should never fail but if it does it's a very bad thing since it
		// should have been checked before we even got this far.
		golog.Errorf("Failed to get auth token when already authenticated: %s", err)
	}

	res := &meResponse{
		Patient: responses.TransformPatient(patient),
		Token:   token,
	}

	if showFeedback(m.dataAPI, m.feedbackClient, patient.ID) {
		res.ActionsNeeded = append(res.ActionsNeeded, &ActionNeeded{Type: actionNeededSimpleFeedbackPrompt})
	}

	httputil.JSONResponse(w, http.StatusOK, res)

	headers := device.ExtractSpruceHeaders(w, r)
	m.dispatcher.PublishAsync(&auth.AuthenticatedEvent{
		AccountID:     patient.AccountID.Int64(),
		SpruceHeaders: headers,
	})
}
