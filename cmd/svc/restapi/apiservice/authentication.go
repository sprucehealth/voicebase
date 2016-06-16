package apiservice

import (
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"golang.org/x/net/context"
)

type tokenValidator interface {
	ValidateToken(token string, platform api.Platform) (*common.Account, error)
}

type authenticatedHandler struct {
	h        httputil.ContextHandler
	authAPI  api.AuthAPI
	optional bool
}

// NoAuthenticationRequiredHandler wraps the provided handler is a layer that performs no authentication
func NoAuthenticationRequiredHandler(h httputil.ContextHandler, authAPI api.AuthAPI) httputil.ContextHandler {
	return &authenticatedHandler{
		h:        h,
		authAPI:  authAPI,
		optional: true,
	}
}

// AuthenticationRequiredHandler wraps the provided handler is a layer that performs authentication
func AuthenticationRequiredHandler(h httputil.ContextHandler, authAPI api.AuthAPI) httputil.ContextHandler {
	return &authenticatedHandler{
		h:        h,
		authAPI:  authAPI,
		optional: false,
	}
}

func (a *authenticatedHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if verifyAuthSetupInTest(ctx, w, r, a.h, authentication, VerifyAuthCode) {
		return
	}

	account, err := a.checkAuth(r)
	if err == nil {
		ctx = CtxWithAccount(ctx, account)
	} else if !a.optional {
		HandleAuthError(ctx, err, w, r)
		return
	}

	if account != nil {
		httputil.CtxLogMap(ctx).Set("AccountID", account.ID)
	}

	a.h.ServeHTTP(ctx, w, r)
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
