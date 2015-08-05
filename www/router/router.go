package router

import (
	"database/sql"
	"io"
	"net/http"

	resources "github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/cookieo9/resources-go"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-librato/librato"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/branch"
	"github.com/sprucehealth/backend/diagnosis"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/events"
	"github.com/sprucehealth/backend/libs/cfg"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/libs/ratelimit"
	"github.com/sprucehealth/backend/libs/sig"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/libs/stripe"
	"github.com/sprucehealth/backend/media"
	"github.com/sprucehealth/backend/passreset"
	"github.com/sprucehealth/backend/www"
	"github.com/sprucehealth/backend/www/admin"
	"github.com/sprucehealth/backend/www/dronboard"
	"github.com/sprucehealth/backend/www/home"
)

var robotsTXT = []byte(`Sitemap: https://www.sprucehealth.com/sitemap.xml
User-agent: *
Disallow: /login
`)

var sitemapXML = []byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url>
		<loc>https://www.sprucehealth.com</loc>
		<changefreq>daily</changefreq>
	</url>
	<url>
		<loc>https://www.sprucehealth.com/meet-the-doctors</loc>
		<changefreq>daily</changefreq>
	</url>
	<url>
		<loc>https://www.sprucehealth.com/about</loc>
		<changefreq>daily</changefreq>
	</url>
	<url>
		<loc>https://www.sprucehealth.com/contact</loc>
		<changefreq>daily</changefreq>
	</url>
</urlset>
`)

type Config struct {
	DataAPI             api.DataAPI
	AuthAPI             api.AuthAPI
	ApplicationDB       *sql.DB
	DiagnosisAPI        diagnosis.API
	SMSAPI              api.SMSAPI
	ERxAPI              erx.ERxAPI
	Dispatcher          *dispatch.Dispatcher
	AnalyticsDB         *sql.DB
	AnalyticsLogger     analytics.Logger
	FromNumber          string
	EmailService        email.Service
	SupportEmail        string
	APIDomain           string
	WebDomain           string
	StaticResourceURL   string
	StripeClient        *stripe.Client
	Signer              *sig.Signer
	Stores              map[string]storage.Store
	MediaStore          *media.Store
	RateLimiters        ratelimit.KeyedRateLimiters
	WebPassword         string
	LibratoClient       *librato.Client
	TemplateLoader      *www.TemplateLoader
	MetricsRegistry     metrics.Registry
	TwoFactorExpiration int
	ExperimentIDs       map[string]string
	CompressResponse    bool
	EventsClient        events.Client
	Cfg                 cfg.Store
	BranchClient        branch.Client
}

// New returns the root handler for the www web app
func New(c *Config) httputil.ContextHandler {
	if c.StaticResourceURL == "" {
		c.StaticResourceURL = "/static"
	}

	c.TemplateLoader.RegisterFunctions(map[string]interface{}{
		"staticURL": func(path string) string {
			return c.StaticResourceURL + path
		},
		"isEnv": func(env string) bool {
			return environment.GetCurrent() == env
		},
	})
	c.TemplateLoader.MustLoadTemplate("base.html", "", nil)

	router := mux.NewRouter().StrictSlash(true)
	c.TemplateLoader.MustLoadTemplate("auth/base.html", "base.html", nil)
	router.Handle("/login", www.NewLoginHandler(c.AuthAPI, c.SMSAPI, c.FromNumber, c.TwoFactorExpiration,
		c.TemplateLoader, c.RateLimiters.Get("login"), c.MetricsRegistry.Scope("login")))
	router.Handle("/login/verify", www.NewLoginVerifyHandler(c.AuthAPI, c.TemplateLoader, c.MetricsRegistry.Scope("login-verify")))
	router.Handle("/logout", www.NewLogoutHandler(c.AuthAPI))
	router.Handle("/robots.txt", RobotsTXTHandler())
	router.Handle("/sitemap.xml", SitemapXMLHandler())
	router.Handle("/favicon.ico", httputil.RedirectHandler(c.StaticResourceURL+"/img/_favicon/favicon.ico", http.StatusMovedPermanently))
	router.PathPrefix("/static").Handler(httputil.StripPrefix("/static", httputil.FileServer(www.ResourceFileSystem)))

	router.Handle("/privacy", StaticHTMLHandler("terms.html"))
	router.Handle("/medication-affordability", StaticHTMLHandler("medafford.html"))

	home.SetupRoutes(router, &home.Config{
		DataAPI:         c.DataAPI,
		AuthAPI:         c.AuthAPI,
		SMSAPI:          c.SMSAPI,
		DiagnosisSvc:    c.DiagnosisAPI,
		WebDomain:       c.WebDomain,
		APIDomain:       c.APIDomain,
		FromSMSNumber:   c.FromNumber,
		BranchClient:    c.BranchClient,
		RateLimiters:    c.RateLimiters,
		Signer:          c.Signer,
		Password:        c.WebPassword,
		AnalyticsLogger: c.AnalyticsLogger,
		TemplateLoader:  c.TemplateLoader,
		ExperimentIDs:   c.ExperimentIDs,
		MediaStore:      c.MediaStore,
		Stores:          c.Stores,
		Dispatcher:      c.Dispatcher,
		MetricsRegistry: c.MetricsRegistry.Scope("home"),
	})
	passreset.SetupRoutes(router, c.DataAPI, c.AuthAPI, c.SMSAPI, c.FromNumber, c.EmailService, c.SupportEmail, c.WebDomain, c.TemplateLoader, c.MetricsRegistry.Scope("reset-password"))
	dronboard.SetupRoutes(router, &dronboard.Config{
		DataAPI:         c.DataAPI,
		AuthAPI:         c.AuthAPI,
		SMSAPI:          c.SMSAPI,
		SMSFromNumber:   c.FromNumber,
		SupportEmail:    c.SupportEmail,
		Dispatcher:      c.Dispatcher,
		StripeClient:    c.StripeClient,
		Signer:          c.Signer,
		Stores:          c.Stores,
		TemplateLoader:  c.TemplateLoader,
		MetricsRegistry: c.MetricsRegistry.Scope("doctor-onboard"),
	})
	admin.SetupRoutes(router, &admin.Config{
		DataAPI:         c.DataAPI,
		AuthAPI:         c.AuthAPI,
		ApplicationDB:   c.ApplicationDB,
		DiagnosisAPI:    c.DiagnosisAPI,
		ERxAPI:          c.ERxAPI,
		AnalyticsDB:     c.AnalyticsDB,
		Signer:          c.Signer,
		Stores:          c.Stores,
		TemplateLoader:  c.TemplateLoader,
		EmailService:    c.EmailService,
		LibratoClient:   c.LibratoClient,
		StripeClient:    c.StripeClient,
		WebDomain:       c.WebDomain,
		APIDomain:       c.APIDomain,
		MetricsRegistry: c.MetricsRegistry.Scope("admin"),
		MediaStore:      c.MediaStore,
		EventsClient:    c.EventsClient,
		Cfg:             c.Cfg,
		AnalyticsLogger: c.AnalyticsLogger,
	})

	secureRedirectHandler := httputil.ContextHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		if !environment.IsTest() && r.Header.Get("X-Forwarded-Proto") != "https" {
			u := r.URL
			u.Scheme = "https"
			u.Host = r.Host
			http.Redirect(w, r, r.URL.String(), http.StatusMovedPermanently)
			return
		}
		router.ServeHTTP(ctx, w, r)
	})

	webRequestLogger := func(ctx context.Context, ev *httputil.RequestEvent) {
		av := &analytics.WebRequestEvent{
			Service:      "www",
			RequestID:    httputil.RequestID(ctx),
			Path:         ev.URL.Path,
			Timestamp:    analytics.Time(ev.Timestamp),
			StatusCode:   ev.StatusCode,
			Method:       ev.Request.Method,
			URL:          ev.URL.String(),
			RemoteAddr:   ev.RemoteAddr,
			ContentType:  ev.ResponseHeaders.Get("Content-Type"),
			UserAgent:    ev.Request.UserAgent(),
			Referrer:     ev.Request.Referer(),
			ResponseTime: int(ev.ResponseTime.Nanoseconds() / 1e3),
			Server:       ev.ServerHostname,
		}
		log := golog.Context(
			"Method", av.Method,
			"URL", av.URL,
			"UserAgent", av.UserAgent,
			"RequestID", av.RequestID,
			"RemoteAddr", av.RemoteAddr,
			"StatusCode", av.StatusCode,
		)
		account, ok := www.CtxAccount(ctx)
		if ok {
			log = log.Context("AccountID", account.ID, "Role", account.Role)
			av.AccountID = account.ID
		}
		if ev.Panic != nil {
			log.Criticalf("http: panic: %v\n%s", ev.Panic, ev.StackTrace)
		} else {
			log.Infof("webrequest")
		}
		c.AnalyticsLogger.WriteEvents([]analytics.Event{av})
	}

	h := httputil.SecurityHandler(secureRedirectHandler)
	if !environment.IsTest() {
		h = httputil.LoggingHandler(h, webRequestLogger)
	}
	h = httputil.DecompressRequest(httputil.RequestIDHandler(h))
	if c.CompressResponse {
		h = httputil.CompressResponse(h)
	}
	return httputil.MetricsHandler(h, c.MetricsRegistry)
}

// StaticHTMLHandler serves the named file from templates/static/<name> on GET
func StaticHTMLHandler(name string) httputil.ContextHandler {
	return httputil.ContextHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		f, err := resources.Open("templates/static/" + name)
		if err != nil {
			www.InternalServerError(w, r, err)
		}
		defer f.Close()
		// TODO: set cache headers
		r.Header.Set("Content-Type", "text/html")
		io.Copy(w, f)
	})
}

// RobotsTXTHandler returns a static robots.txt
func RobotsTXTHandler() httputil.ContextHandler {
	return httputil.ContextHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		// TODO: set cache headers
		if _, err := w.Write(robotsTXT); err != nil {
			golog.Errorf(err.Error())
		}
	})
}

// SitemapXMLHandler returns a static sitemap.xml
func SitemapXMLHandler() httputil.ContextHandler {
	return httputil.ContextHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		// TODO: set cache headers
		if _, err := w.Write(sitemapXML); err != nil {
			golog.Errorf(err.Error())
		}
	})
}
