package apiservice

import (
	"carefront/libs/golog"
	thriftapi "carefront/thrift/api"
	"net/http"
)

type isAuthenticatedHandler struct {
	AuthApi thriftapi.Auth
}

func NewIsAuthenticatedHandler(authApi thriftapi.Auth) *isAuthenticatedHandler {
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
