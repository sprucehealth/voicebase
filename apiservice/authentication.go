package apiservice

import (
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
)

type authenticatedHandler struct {
	h       http.Handler
	authAPI api.AuthAPI
}

func NoAuthenticationRequiredHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if verifyAuthSetupInTest(w, r, h, authentication, VerifyAuthCode) {
			return
		}

		h.ServeHTTP(w, r)
	})
}

func AuthenticationRequiredHandler(h http.Handler, authAPI api.AuthAPI) http.Handler {
	return &authenticatedHandler{
		h:       h,
		authAPI: authAPI,
	}
}

func (a *authenticatedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if verifyAuthSetupInTest(w, r, a.h, authentication, VerifyAuthCode) {
		return
	}

	ctx := GetContext(r)
	account, err := a.checkAuth(r)

	if err == nil {
		ctx.AccountID = account.ID
		ctx.Role = account.Role
	} else {
		HandleAuthError(err, w, r)
		return
	}

	a.h.ServeHTTP(w, r)
}

// Parse the "Authorization: token xxx" header and check the token for validity
func (a *authenticatedHandler) checkAuth(r *http.Request) (*common.Account, error) {
	if Testing {
		if idStr := r.Header.Get("AccountID"); idStr != "" {
			id, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil {
				return nil, err
			}
			return a.authAPI.GetAccount(id)
		}
	}

	token, err := GetAuthTokenFromHeader(r)
	if err != nil {
		return nil, err
	}
	return a.authAPI.ValidateToken(token, api.Mobile)
}
