package auth

import (
	"net/http"
	"strings"

	"context"

	"github.com/sprucehealth/backend/cmd/svc/admin/internal/auth"
	"github.com/sprucehealth/backend/cmd/svc/admin/internal/common"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/sig"
)

type authenticatedHandler struct {
	h      http.Handler
	signer *sig.Signer
}

func (h *authenticatedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	golog.ContextLogger(r.Context()).Debugf("Cookies: %d", len(r.Cookies()))
	for _, c := range r.Cookies() {
		golog.ContextLogger(r.Context()).Debugf("%s: %s", c.Name, c.Value)
	}
	if c, err := r.Cookie(common.AuthCookieName); err == nil && c.Value != "" {
		valid, uid := auth.IsTokenValid(r.Context(), c.Value, h.signer)
		if !valid {
			removeAuthCookie(w, r.Host)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		r = r.WithContext(WithUID(r.Context(), uid))
	} else {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	h.h.ServeHTTP(w, r)
}

// NewAuthenticated wraps a handler requiring that all requests be authenticated
func NewAuthenticated(h http.Handler, signer *sig.Signer) http.Handler {
	return &authenticatedHandler{
		h:      h,
		signer: signer,
	}
}

func removeAuthCookie(w http.ResponseWriter, domain string) {
	domain = getDomain(domain)
	http.SetCookie(w, &http.Cookie{
		Name:     common.AuthCookieName,
		Domain:   domain,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Secure:   !environment.IsDev(),
		HttpOnly: true,
	})
}

func getDomain(domain string) string {
	// set the auth cookie for the root domain rather than the specific endpoint.
	idx := strings.IndexByte(domain, '.')
	if idx != -1 {
		domain = domain[idx:]
	}
	idx = strings.IndexByte(domain, ':')
	if idx != -1 {
		domain = domain[:idx]
	}
	return domain
}

type ctxKey int

const (
	ctxUID ctxKey = iota
)

// WithUID attaches the uid to the context
func WithUID(ctx context.Context, uid string) context.Context {
	return context.WithValue(ctx, ctxUID, uid)
}

// UID returns the uid for the requesting user
func UID(ctx context.Context) string {
	uid, _ := ctx.Value(ctxUID).(string)
	return uid
}
