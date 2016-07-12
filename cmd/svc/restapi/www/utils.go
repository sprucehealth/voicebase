package www

import (
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
	"github.com/sprucehealth/backend/libs/golog"
)

const (
	authCookieName     = "d_at"
	deviceIDCookieName = "d_did"
	passCookieName     = "d_hp"
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
	okHandler     http.Handler
	failedHandler http.Handler
}

type roleRequiredHandler struct {
	roles         []string
	okHandler     http.Handler
	failedHandler http.Handler
}

type apiAuthRequiredFilter struct {
	authAPI api.AuthAPI
	h       http.Handler
}

// APIRoleRequiredHandler returns a handler that can be used to restrict access to
// an API handler to only requests from a user matching a set of roles. The request must
// already have passed through the authentication handler.
func APIRoleRequiredHandler(h http.Handler, roles ...string) http.Handler {
	return RoleRequiredHandler(h, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		APIForbidden(w, r)
	}), roles...)
}

// RoleRequiredHandler returns a handler that can be used to restrict access to
// a handler to only requests from a user matching a set of roles. The request must
// already have passed through the authentication handler.
func RoleRequiredHandler(ok, failed http.Handler, roles ...string) http.Handler {
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
func APIAuthRequiredHandler(h http.Handler, authAPI api.AuthAPI) http.Handler {
	return AuthRequiredHandler(h, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		APIForbidden(w, r)
	}), authAPI)
}

// AuthRequiredHandler returns a filter that can be used to restrict access to
// a handler only requests that are authenticated.
func AuthRequiredHandler(ok, failed http.Handler, authAPI api.AuthAPI) http.Handler {
	if failed == nil {
		failed = loginRedirectHandler
	}
	return &authRequiredHandler{
		authAPI:       authAPI,
		okHandler:     ok,
		failedHandler: failed,
	}
}

func (h roleRequiredHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	account := MustCtxAccount(r.Context())
	for _, role := range h.roles {
		if account.Role == role {
			h.okHandler.ServeHTTP(w, r)
			return
		}
	}
	h.failedHandler.ServeHTTP(w, r)
}

func (h *authRequiredHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	account, err := ValidateAuth(h.authAPI, r)
	switch err {
	case nil:
		h.okHandler.ServeHTTP(w, r.WithContext(CtxWithAccount(r.Context(), account)))
		return
	case http.ErrNoCookie, api.ErrTokenDoesNotExist, api.ErrTokenExpired:
	default:
		// Log any other error
		golog.Errorf("Failed to validate auth: %s", err.Error())
	}
	h.failedHandler.ServeHTTP(w, r)
}

var loginRedirectHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	RedirectToSignIn(w, r)
})

// PasswordProtectFilter returns a function wrapper for an http handler to check if a specified
// password is set before proceeding to the page requested.
func PasswordProtectFilter(pass string, templateLoader *TemplateLoader) func(h http.Handler) http.Handler {
	tmpl := templateLoader.MustLoadTemplate("home/pass.html", "base.html", nil)
	return func(h http.Handler) http.Handler {
		if pass == "" {
			return h
		}
		return &passwordProtectHandler{
			h:    h,
			pass: pass,
			tmpl: tmpl,
		}
	}
}

type passwordProtectHandler struct {
	h    http.Handler
	pass string
	tmpl *template.Template
}

func (h *passwordProtectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(passCookieName)
	if err == nil {
		if c.Value == h.pass {
			h.h.ServeHTTP(w, r)
			return
		}
	}

	var errorMsg string
	if r.Method == "POST" {
		if pass := r.FormValue("Password"); pass == h.pass {
			domain := r.Host
			if i := strings.IndexByte(domain, ':'); i > 0 {
				domain = domain[:i]
			}
			http.SetCookie(w, &http.Cookie{
				Name:   passCookieName,
				Value:  pass,
				Path:   "/",
				Domain: domain,
				Secure: true,
			})
			// Redirect back to the same URL to get rid of the POST. On the next request
			// this handler should just pass through to the real handler since the cookie
			// will be set.
			http.Redirect(w, r, r.RequestURI, http.StatusSeeOther)
			return
		}
		errorMsg = "Invalid password."
	}
	TemplateResponse(w, http.StatusOK, h.tmpl, &BaseTemplateContext{
		Title: "Spruce",
		SubContext: &struct {
			Error string
		}{
			Error: errorMsg,
		},
	})
}
