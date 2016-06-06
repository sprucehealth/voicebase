package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/media/internal/mediactx"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/svc/auth"
	"golang.org/x/net/context"
)

type authHandler struct {
	auth auth.AuthClient
	h    httputil.ContextHandler
}

func authenticationRequired(h httputil.ContextHandler, auth auth.AuthClient) httputil.ContextHandler {
	return &authHandler{
		auth: auth,
		h:    h,
	}
}

func (h *authHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(authTokenCookieName)
	if err == http.ErrNoCookie {
		forbidden(w, err, golog.WARN)
		return
	} else if err != nil {
		internalError(w, fmt.Errorf("Error getting cookie: %s", err))
		return
	}
	if c.Value == "" {
		forbidden(w, errors.New("Empty cookie value. Temporary log to weed out any issues with cookie handling between subdomains"), golog.WARN)
		return
	}

	res, err := h.auth.CheckAuthentication(ctx,
		&auth.CheckAuthenticationRequest{
			Token: c.Value,
		},
	)
	if err != nil {
		forbidden(w, fmt.Errorf("Failed to check auth token: %s", err), golog.ERR)
		return
	}
	if !res.IsAuthenticated {
		forbidden(w, errors.New("User is unauthenticated. Temporary log to weed out any issues with cookie handling between subdomains"), golog.WARN)
		return
	}

	ctx = mediactx.WithAuthToken(ctx, c.Value)
	ctx = mediactx.WithAccount(ctx, res.Account)
	h.h.ServeHTTP(ctx, w, r)
}
