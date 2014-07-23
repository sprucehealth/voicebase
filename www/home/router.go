package home

import (
	"net/http"
	"strings"

	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/www"
)

const passCookieName = "hp"

func SetupRoutes(r *mux.Router, password string, metricsRegistry metrics.Registry) {
	r.Handle("/", PasswordProtect(password, NewHomeHandler(r)))
	r.Handle("/about", PasswordProtect(password, NewAboutHandler(r)))
}

func PasswordProtect(pass string, h http.Handler) http.Handler {
	return &passwordProtectHandler{
		h: h,
		p: pass,
	}
}

type passwordProtectHandler struct {
	h http.Handler
	p string
}

func (h *passwordProtectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(passCookieName)
	if err == nil {
		if c.Value == h.p {
			h.h.ServeHTTP(w, r)
			return
		}
	}

	var errorMsg string
	if r.Method == "POST" {
		if pass := r.FormValue("Password"); pass == h.p {
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
			http.Redirect(w, r, "", http.StatusSeeOther)
			return
		} else {
			errorMsg = "Invalid password."
		}
	}
	www.TemplateResponse(w, http.StatusOK, passTemplate, &www.BaseTemplateContext{
		Title: "Spruce",
		SubContext: &passTemplateContext{
			Error: errorMsg,
		},
	})
}
