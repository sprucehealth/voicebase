package auth

import (
	"net/http"
	"strings"

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
	golog.Debugf("Cookies: %d", len(r.Cookies()))
	for _, c := range r.Cookies() {
		golog.Debugf("%s: %s", c.Name, c.Value)
	}
	if c, err := r.Cookie(common.AuthCookieName); err == nil && c.Value != "" {
		valid, _ := auth.IsTokenValid(c.Value, h.signer)
		if !valid {
			removeAuthCookie(w, r.Host)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
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
