package handlers

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/golog"
)

type isAuthenticatedHandler struct {
	authAPI api.AuthAPI
}

func NewIsAuthenticatedHandler(authAPI api.AuthAPI) http.Handler {
	return &isAuthenticatedHandler{
		authAPI: authAPI,
	}
}

func (i *isAuthenticatedHandler) IsAuthorized(r *http.Request) (bool, error) {
	return true, nil
}

func (i *isAuthenticatedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	accountId := apiservice.GetContext(r).AccountId
	go func() {
		// asyncrhonously update the last opened date for this account
		if err := i.authAPI.UpdateLastOpenedDate(accountId); err != nil {
			golog.Errorf("Unable to update last opened date for account: %s", err)
		}
	}()

	apiservice.WriteJSONSuccess(w)
}
