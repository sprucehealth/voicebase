package handlers

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"golang.org/x/net/context"
)

type isAuthenticatedHandler struct {
	authAPI api.AuthAPI
}

func NewIsAuthenticatedHandler(authAPI api.AuthAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(apiservice.NoAuthorizationRequired(
		&isAuthenticatedHandler{
			authAPI: authAPI,
		}), httputil.Get)
}

func (i *isAuthenticatedHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	account := apiservice.MustCtxAccount(ctx)
	go func() {
		// asyncrhonously update the last opened date for this account
		if err := i.authAPI.UpdateLastOpenedDate(account.ID); err != nil {
			golog.Errorf("Unable to update last opened date for account: %s", err)
		}
	}()
	apiservice.WriteJSONSuccess(w)
}
