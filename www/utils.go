package www

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"golang.org/x/net/context"
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
		return "", false
	}
	path := u.Path
	if len(path) == 0 || path[0] != '/' {
		return "", false
	}
	// TODO: what else needs to be checked?
	return path, true
}

// NewAuthCookie returns a new auth cookie using the provided token and
// HOST from the request as the domain. By default the cookie has no
// max-age and as such is a session only cookie.
func NewAuthCookie(token string, r *http.Request) *http.Cookie {
	return NewCookie(authCookieName, token, r)
}

// NewCookie returns a cookie that has a path of '/' and domain equal to
// the HOST of the request. By default the cookie is HTTP only (can't access
// it through Javascript), and the cookie has no max-age so exists for
// the current browser session only.
func NewCookie(name, value string, r *http.Request) *http.Cookie {
	domain := r.Host
	if i := strings.IndexByte(domain, ':'); i > 0 {
		domain = domain[:i]
	}
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		Domain:   domain,
		Secure:   true,
		HttpOnly: true,
	}
}

// TombstoneAuthCookie returns an empty valued auth cookie. It has a max age
// set to tell the browser to delete the cookie immediately.
func TombstoneAuthCookie(r *http.Request) *http.Cookie {
	c := NewAuthCookie("", r)
	c.MaxAge = -1
	c.Expires = time.Now().Add(-time.Hour)
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

type authRequiredHandler struct {
	authAPI       api.AuthAPI
	okHandler     httputil.ContextHandler
	failedHandler httputil.ContextHandler
}

type roleRequiredHandler struct {
	roles         []string
	okHandler     httputil.ContextHandler
	failedHandler httputil.ContextHandler
}

type apiAuthRequiredFilter struct {
	authAPI api.AuthAPI
	h       httputil.ContextHandler
}

// APIRoleRequiredHandler returns a handler that can be used to restrict access to
// an API handler to only requests from a user matching a set of roles. The request must
// already have passed through the authentication handler.
func APIRoleRequiredHandler(h httputil.ContextHandler, roles ...string) httputil.ContextHandler {
	return RoleRequiredHandler(h, httputil.ContextHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		APIForbidden(w, r)
	}), roles...)
}

// RoleRequiredHandler returns a handler that can be used to restrict access to
// a handler to only requests from a user matching a set of roles. The request must
// already have passed through the authentication handler.
func RoleRequiredHandler(ok, failed httputil.ContextHandler, roles ...string) httputil.ContextHandler {
	if failed == nil {
		failed = loginRedirectHandler
	}
	return &roleRequiredHandler{
		roles:         roles,
		okHandler:     ok,
		failedHandler: failed,
	}
}

// APIAuthRequiredHandler returns a filter that can be used to restrict access to
// an API handler only requests that are authenticated.
func APIAuthRequiredHandler(h httputil.ContextHandler, authAPI api.AuthAPI) httputil.ContextHandler {
	return AuthRequiredHandler(h, httputil.ContextHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		APIForbidden(w, r)
	}), authAPI)
}

// AuthRequiredHandler returns a filter that can be used to restrict access to
// a handler only requests that are authenticated.
func AuthRequiredHandler(ok, failed httputil.ContextHandler, authAPI api.AuthAPI) httputil.ContextHandler {
	if failed == nil {
		failed = loginRedirectHandler
	}
	return &authRequiredHandler{
		authAPI:       authAPI,
		okHandler:     ok,
		failedHandler: failed,
	}
}

func (h roleRequiredHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	account := MustCtxAccount(ctx)
	for _, role := range h.roles {
		if account.Role == role {
			h.okHandler.ServeHTTP(ctx, w, r)
			return
		}
	}
	h.failedHandler.ServeHTTP(ctx, w, r)
}

func (h *authRequiredHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	account, err := ValidateAuth(h.authAPI, r)
	switch err {
	case nil:
		h.okHandler.ServeHTTP(CtxWithAccount(ctx, account), w, r)
		return
	case http.ErrNoCookie, api.ErrTokenDoesNotExist, api.ErrTokenExpired:
	default:
		// Log any other error
		golog.Errorf("Failed to validate auth: %s", err.Error())
	}
	h.failedHandler.ServeHTTP(ctx, w, r)
}

var loginRedirectHandler = httputil.ContextHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	RedirectToSignIn(w, r)
})
