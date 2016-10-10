package auth

import (
	"encoding/json"
	"net/http"
	"time"

	"context"

	"github.com/sprucehealth/backend/cmd/svc/admin/internal/auth"
	"github.com/sprucehealth/backend/cmd/svc/admin/internal/common"
	"github.com/sprucehealth/backend/cmd/svc/admin/internal/google"
	"github.com/sprucehealth/backend/cmd/svc/restapi/apiservice"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/sig"
)

type authenticationRequest struct {
	IDToken string `json:"idToken"`
}

type authenticationResponse struct {
	Token string `json:"token"`
	Name  string `json:"name"`
}

type authenticationHandler struct {
	signer *sig.Signer
	ap     *google.AuthProvider
}

func (h *authenticationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	golog.ContextLogger(r.Context()).Debugf("Handling %v", r)
	switch r.Method {
	case httputil.Post:
		var req authenticationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apiservice.WriteBadRequestError(err, w, r)
			return
		}
		id, err := h.ap.Authenticate(r.Context(), req.IDToken)
		if errors.Cause(err) == google.ErrForbidden {
			http.Error(w, "Bad Request", http.StatusForbidden)
			return
		} else if err != nil {
			golog.ContextLogger(r.Context()).Errorf("Error authenticating with token %s: %s", req.IDToken, err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		// TODO: If we ever want to interact with the google account, we need to encode more token info here
		token, exp, err := auth.NewToken(r.Context(), id, h.signer)
		if err != nil {
			golog.ContextLogger(r.Context()).Errorf("Error creating token for user %s: %s", id, err)
			http.Error(w, "Internal Error", http.StatusInternalServerError)
			return
		}
		setAuthCookie(r.Context(), w, r.Host, token, exp)
		httputil.JSONResponse(w, http.StatusOK, &authenticationResponse{
			Token: token,
			Name:  id,
		})
	default:
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
}

// NewAuthentication returns an authentication handler
func NewAuthentication(ap *google.AuthProvider, signer *sig.Signer) http.Handler {
	return &authenticationHandler{
		signer: signer,
		ap:     ap,
	}
}

func setAuthCookie(ctx context.Context, w http.ResponseWriter, domain, token string, expires time.Time) {
	domain = getDomain(domain)
	golog.ContextLogger(ctx).Debugf("Setting auth cookie: %s, %s, expires - %v", domain, token, expires)
	http.SetCookie(w, &http.Cookie{
		Name:     common.AuthCookieName,
		Domain:   domain,
		Value:    token,
		Path:     "/",
		MaxAge:   int(expires.Sub(time.Now()).Nanoseconds() / 1e9),
		Secure:   !environment.IsDev(),
		HttpOnly: true,
	})
}

type unAuthenticationHandler struct{}

func (h *unAuthenticationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	golog.ContextLogger(r.Context()).Debugf("Handling %v", r)
	switch r.Method {
	case httputil.Post:
		tombstoneAuthCookie(r.Context(), w, r.Host)
	default:
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
}

// NewUnauthentication returns an unauthentication handler
func NewUnauthentication() http.Handler {
	return &unAuthenticationHandler{}
}

// TombstoneAuthCookie set an empty valued auth cookie. It has a max age
// set to tell the browser to delete the cookie immediately.
func tombstoneAuthCookie(ctx context.Context, w http.ResponseWriter, domain string) {
	http.SetCookie(w, &http.Cookie{
		Name:     common.AuthCookieName,
		Domain:   domain,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Now().Add(-time.Hour),
		Secure:   !environment.IsDev(),
		HttpOnly: true,
	})
}
