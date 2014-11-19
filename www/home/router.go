package home

import (
	"html/template"
	"net/http"
	"strings"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/www"
)

const passCookieName = "hp"

func SetupRoutes(r *mux.Router, dataAPI api.DataAPI, authAPI api.AuthAPI, password string, analyticsLogger analytics.Logger, templateLoader *www.TemplateLoader, experimentIDs map[string]string, metricsRegistry metrics.Registry) {
	templateLoader.MustLoadTemplate("home/base.html", "base.html", nil)
	templateLoader.MustLoadTemplate("promotions/base.html", "home/base.html", nil)

	var protect func(http.Handler) http.Handler
	if password != "" {
		protect = PasswordProtectFilter(password, templateLoader)
	} else {
		protect = func(h http.Handler) http.Handler { return h }
	}

	r.Handle("/", protect(newStaticHandler(r, templateLoader, "home/home.html", "Spruce")))
	r.Handle("/about", protect(newStaticHandler(r, templateLoader, "home/about.html", "About | Spruce")))
	r.Handle("/contact", protect(newStaticHandler(r, templateLoader, "home/contact.html", "Contact | Spruce")))
	r.Handle("/meet-the-doctors", protect(newStaticHandler(r, templateLoader, "home/meet-the-doctors.html", "Meet the Doctors | Spruce")))

	// Referrals
	r.Handle("/r/{code}", protect(newPromoClaimHandler(dataAPI, authAPI, analyticsLogger, templateLoader, experimentIDs["promo"])))
	r.Handle("/r/{code}/notify/state", protect(newPromoNotifyStateHandler(dataAPI, analyticsLogger, templateLoader, experimentIDs["promo"])))
	r.Handle("/r/{code}/notify/android", protect(newPromoNotifyAndroidHandler(dataAPI, analyticsLogger, templateLoader, experimentIDs["promo"])))

	// API
	r.Handle("/api/forms/{form:[0-9a-z-]+}", protect(NewFormsAPIHandler(dataAPI)))

	// Analytics
	ah := newAnalyticsHandler(analyticsLogger, metricsRegistry.Scope("analytics"))
	r.Handle("/a/events", ah)   // For javascript originating events
	r.Handle("/a/logo.png", ah) // For remote event tracking "pixels" (e.g. email)
}

func PasswordProtectFilter(pass string, templateLoader *www.TemplateLoader) func(http.Handler) http.Handler {
	tmpl := templateLoader.MustLoadTemplate("home/pass.html", "base.html", nil)
	return func(h http.Handler) http.Handler {
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
			http.Redirect(w, r, "", http.StatusSeeOther)
			return
		} else {
			errorMsg = "Invalid password."
		}
	}
	www.TemplateResponse(w, http.StatusOK, h.tmpl, &www.BaseTemplateContext{
		Title: "Spruce",
		SubContext: &struct {
			Error string
		}{
			Error: errorMsg,
		},
	})
}
