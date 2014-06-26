package apiservice

import (
	"carefront/api"
	"carefront/libs/golog"
	"net/http"
)

type isAuthenticatedHandler struct {
	AuthApi api.AuthAPI
}

func NewIsAuthenticatedHandler(authApi api.AuthAPI) *isAuthenticatedHandler {
	return &isAuthenticatedHandler{
		AuthApi: authApi,
	}
}

func (i *isAuthenticatedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// asyncrhonously update the last opened date for this account
	accountId := GetContext(r).AccountId
	go func() {
		if err := i.AuthApi.UpdateLastOpenedDate(accountId); err != nil {
			golog.Errorf("Unable to update last opened date for account: %s", err)
		}
	}()

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, SuccessfulGenericJSONResponse())
}
