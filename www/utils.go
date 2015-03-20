package www

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/golog"
)

const (
	authCookieName     = "at"
	deviceIDCookieName = "did"
)

// validateRedirectURL makes sure that a user provided URL that will be
// used for a redirect (such as 'next' during login) is valid and safe.
// It also optionally rewrites the URL when appropriate.
func validateRedirectURL(urlString string) (string, bool) {
	u, err := url.Parse(urlString)
	if err != nil {
		println(err.Error())
		return "", false
	}
	path := u.Path
	if len(path) == 0 || path[0] != '/' {
		return "", false
	}
	// TODO: what else needs to be checked?
	return path, true
}

func NewAuthCookie(token string, r *http.Request) *http.Cookie {
	return NewCookie(authCookieName, token, r)
}

// NewCookie returns a cookie that has a path of '/' and domain equal to
// the HOST of the request.
func NewCookie(name, value string, r *http.Request) *http.Cookie {
	domain := r.Host
	if i := strings.IndexByte(domain, ':'); i > 0 {
		domain = domain[:i]
	}
	return &http.Cookie{
		Name:   name,
		Value:  value,
		Path:   "/",
		Domain: domain,
		Secure: true,
		// Expires: time.Time
		// MaxAge : int
	}
}

func TomestoneAuthCookie(r *http.Request) *http.Cookie {
	c := NewAuthCookie("", r)
	c.MaxAge = -1
	c.Value = ""
	return c
}

// ValidateAuth validates the authentication token in the request. If there
// is no cookie then it returns http.ErrNoCookie. Otherwise, the response is
// api.(*AuthAPI).ValidateToken(...)
func ValidateAuth(authAPI api.AuthAPI, r *http.Request) (*common.Account, error) {
	c, err := r.Cookie(authCookieName)
	if err != nil {
		return nil, err
	} else if c.Value == "" {
		return nil, http.ErrNoCookie
	}
	return authAPI.ValidateToken(c.Value, api.Web)
}

type authRequiredFilter struct {
	authAPI       api.AuthAPI
	roles         []string
	okHandler     http.Handler
	failedHandler http.Handler
}

func AuthRequiredFilter(authAPI api.AuthAPI, roles []string, failed http.Handler) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return AuthRequiredHandler(authAPI, roles, h, failed)
	}
}

func AuthRequiredHandler(authAPI api.AuthAPI, roles []string, ok, failed http.Handler) http.Handler {
	if failed == nil {
		failed = loginRedirectHandler
	}
	return &authRequiredFilter{
		authAPI:       authAPI,
		roles:         roles,
		okHandler:     ok,
		failedHandler: failed,
	}
}

func (h *authRequiredFilter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	account, err := ValidateAuth(h.authAPI, r)
	switch err {
	case nil:
		for _, role := range h.roles {
			if role == account.Role {
				context.Set(r, CKAccount, account)
				h.okHandler.ServeHTTP(w, r)
				return
			}
		}
	case http.ErrNoCookie, api.TokenDoesNotExist, api.TokenExpired:
	default:
		// Log any other error
		golog.Errorf("Failed to validate auth: %s", err.Error())
	}
	h.failedHandler.ServeHTTP(w, r)
}

var loginRedirectHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/login?next="+url.QueryEscape(r.URL.Path), http.StatusSeeOther)
})
