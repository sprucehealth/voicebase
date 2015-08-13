package apiservice

import (
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
)

type authenticatedHandler struct {
	h        httputil.ContextHandler
	authAPI  api.AuthAPI
	optional bool
}

func NoAuthenticationRequiredHandler(h httputil.ContextHandler, authAPI api.AuthAPI) httputil.ContextHandler {
	return &authenticatedHandler{
		h:        h,
		authAPI:  authAPI,
		optional: true,
	}
}

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

	token, err := GetAuthTokenFromHeader(r)
	if err != nil {
		return nil, err
	}
	return a.authAPI.ValidateToken(token, api.Mobile)
}
