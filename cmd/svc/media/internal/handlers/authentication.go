package handlers

import (
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/media/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/media/internal/mediactx"
	"github.com/sprucehealth/backend/cmd/svc/media/internal/service"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/libs/urlutil"
	"github.com/sprucehealth/backend/svc/auth"
	"golang.org/x/net/context"
)

type authHandler struct {
	auth   auth.AuthClient
	signer *urlutil.Signer
	svc    service.Service
	h      httputil.ContextHandler
}

func authenticationRequired(h httputil.ContextHandler, auth auth.AuthClient, signer *urlutil.Signer, svc service.Service) httputil.ContextHandler {
	return &authHandler{
		auth:   auth,
		signer: signer,
		svc:    svc,
		h:      h,
	}
}

func (h *authHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Check to see if this is public media. If so se tthe appropriate flags and pass through
	mediaID, err := dal.ParseMediaID(mux.Vars(ctx)[idParamName])
	if err != nil {
		badRequest(w, errors.New("Cannot parse media id"), http.StatusBadRequest)
		return
	}
	public, err := h.svc.IsPublic(ctx, mediaID)
	if errors.Cause(err) == dal.ErrNotFound {
		http.Error(w, "Not Found", http.StatusNotFound)
	} else if err != nil {
		internalError(w, err)
	}
	if public {
		ctx = mediactx.WithRequiresAuthorization(ctx, false)
		h.h.ServeHTTP(ctx, w, r)
		return
	}

	// Check to see if the url is signed. If it is use that as auth and flag it as not requiring authorization, else follow the normal flow
	sig := r.URL.Query().Get(urlutil.SigParamName)
	if sig != "" && (r.Method == httputil.Get || r.Method == httputil.Head) {
		if err := h.signer.ValidateSignature(r.URL.Path, r.URL.Query()); err == nil {
			ctx = mediactx.WithRequiresAuthorization(ctx, false)
		} else if errors.Cause(err) == urlutil.ErrExpiredURL {
			forbidden(w, errors.New("URL Expired"), golog.WARN)
			return
		} else if errors.Cause(err) == urlutil.ErrSignatureMismatch {
			forbidden(w, errors.New("Incorrect Signature"), golog.WARN)
			return
		} else if err != nil {
			internalError(w, err)
			return
		}
	} else {
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
	}
	h.h.ServeHTTP(ctx, w, r)
}
