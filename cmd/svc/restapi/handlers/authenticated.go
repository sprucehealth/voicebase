package handlers

import (
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/apiservice"
	"github.com/sprucehealth/backend/cmd/svc/restapi/internal/httputil"
	"github.com/sprucehealth/backend/libs/golog"
)

type isAuthenticatedHandler struct {
	authAPI api.AuthAPI
}

// NewIsAuthenticatedHandler returns an initialized instance of isAuthenticatedHandler
func NewIsAuthenticatedHandler(authAPI api.AuthAPI) http.Handler {
	return httputil.SupportedMethods(apiservice.NoAuthorizationRequired(
		&isAuthenticatedHandler{
			authAPI: authAPI,
		}), httputil.Get)
}

func (i *isAuthenticatedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	account := apiservice.MustCtxAccount(r.Context())
	go func() {
		// asyncrhonously update the last opened date for this account
		if err := i.authAPI.UpdateLastOpenedDate(account.ID); err != nil {
			golog.Errorf("Unable to update last opened date for account: %s", err)
		}
	}()
	apiservice.WriteJSONSuccess(w)
}
