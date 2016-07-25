package auth

import (
	"encoding/json"
	"net/http"
	"time"

	"context"

	"github.com/sprucehealth/backend/cmd/svc/admin/internal/auth"
	"github.com/sprucehealth/backend/cmd/svc/admin/internal/common"
	"github.com/sprucehealth/backend/cmd/svc/restapi/apiservice"
	"github.com/sprucehealth/backend/environment"
	lauth "github.com/sprucehealth/backend/libs/auth"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/sig"
)

type authenticationRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type authenticationResponse struct {
	Token string `json:"token"`
	UID   string `json:"uid"`
}

type authenticationHandler struct {
	signer *sig.Signer
	ap     lauth.AuthenticationProvider
}

func (h *authenticationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case httputil.Post:
		var req authenticationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apiservice.WriteBadRequestError(err, w, r)
			return
		}
		id, err := h.ap.Authenticate(r.Context(), req.Username, req.Password)
		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		token, exp, err := auth.NewToken(r.Context(), id, h.signer)
		if err != nil {
			http.Error(w, "Internal Error", http.StatusInternalServerError)
			return
		}
		setAuthCookie(r.Context(), w, r.Host, token, exp)
		httputil.JSONResponse(w, http.StatusOK, &authenticationResponse{
			Token: token,
			UID:   id,
		})
	default:
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
}

// NewAuthentication returns an authentication handler
func NewAuthentication(ap lauth.AuthenticationProvider, signer *sig.Signer) http.Handler {
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
