package router

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/payment/stripe"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/passreset"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/third_party/github.com/subosito/twilio"
	"github.com/sprucehealth/backend/www"
	"github.com/sprucehealth/backend/www/admin"
	"github.com/sprucehealth/backend/www/dronboard"
	"github.com/sprucehealth/backend/www/home"
)

type Config struct {
	DataAPI           api.DataAPI
	AuthAPI           api.AuthAPI
	TwilioCli         *twilio.Client
	FromNumber        string
	EmailService      email.Service
	SupportEmail      string
	WebSubdomain      string
	StaticResourceURL string
	StripeCli         *stripe.StripeService
	Signer            *common.Signer
	Stores            map[string]storage.Store
	MetricsRegistry   metrics.Registry
	WebPassword       string
}

func New(c *Config) http.Handler {
	www.StaticURL = c.StaticResourceURL

	router := mux.NewRouter().StrictSlash(true)
	router.Handle("/login", www.NewLoginHandler(c.AuthAPI))
	router.Handle("/logout", www.NewLogoutHandler(c.AuthAPI))
	router.PathPrefix("/static").Handler(http.StripPrefix("/static", http.FileServer(www.ResourceFileSystem)))

	home.SetupRoutes(router, c.WebPassword, c.MetricsRegistry.Scope("home"))
	passreset.SetupRoutes(router, c.DataAPI, c.AuthAPI, c.TwilioCli, c.FromNumber, c.EmailService, c.SupportEmail, c.WebSubdomain, c.MetricsRegistry.Scope("reset-password"))
	dronboard.SetupRoutes(router, c.DataAPI, c.AuthAPI, c.SupportEmail, c.StripeCli, c.Signer, c.Stores, c.MetricsRegistry.Scope("doctor-onboard"))
	admin.SetupRoutes(router, c.DataAPI, c.AuthAPI, c.StripeCli, c.Signer, c.Stores, c.MetricsRegistry.Scope("admin"))

	secureRedirectHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Forwarded-Proto") != "https" {
			u := r.URL
			u.Scheme = "https"
			u.Host = r.Host
			http.Redirect(w, r, r.URL.String(), http.StatusMovedPermanently)
			return
		}
		router.ServeHTTP(w, r)
	})

	return httputil.CompressResponse(httputil.DecompressRequest(httputil.LoggingHandler(secureRedirectHandler, golog.Default())))
}
