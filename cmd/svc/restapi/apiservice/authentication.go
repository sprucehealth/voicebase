package apiservice

import (
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
	"github.com/sprucehealth/backend/cmd/svc/restapi/internal/httputil"
)

type tokenValidator interface {
	ValidateToken(token string, platform api.Platform) (*common.Account, error)
}

type authenticatedHandler struct {
	h        http.Handler
	authAPI  api.AuthAPI
	optional bool
}

// NoAuthenticationRequiredHandler wraps the provided handler is a layer that performs no authentication
func NoAuthenticationRequiredHandler(h http.Handler, authAPI api.AuthAPI) http.Handler {
	return &authenticatedHandler{
		h:        h,
		authAPI:  authAPI,
		optional: true,
	}
}

// AuthenticationRequiredHandler wraps the provided handler is a layer that performs authentication
func AuthenticationRequiredHandler(h http.Handler, authAPI api.AuthAPI) http.Handler {
	return &authenticatedHandler{
		h:        h,
		authAPI:  authAPI,
		optional: false,
	}
}

func (a *authenticatedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if verifyAuthSetupInTest(w, r, a.h, authentication, VerifyAuthCode) {
		return
	}

	account, err := a.checkAuth(r)
	if err == nil {
		ctx = CtxWithAccount(ctx, account)
	} else if !a.optional {
		HandleAuthError(err, w, r)
		return
	}

	if account != nil {
		httputil.CtxLogMap(ctx).Set("AccountID", account.ID)
	}

	a.h.ServeHTTP(w, r.WithContext(ctx))
}

// checkAuth parses the "Authorization: token xxx" header and check the token for validity
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

	return AccountFromAuthHeader(r, a.authAPI)
}

// AccountFromAuthHeader inspects the header of the provided requests and returns the associated account if one exists
func AccountFromAuthHeader(r *http.Request, tokenValidator tokenValidator) (*common.Account, error) {
	token, err := GetAuthTokenFromHeader(r)
	if err != nil {
		return nil, err
	}
	return tokenValidator.ValidateToken(token, api.Mobile)
}
