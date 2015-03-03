package main

import (
	"database/sql"
	"io"
	"log"
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/cookieo9/resources-go"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-librato/librato"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice/apipaths"
	"github.com/sprucehealth/backend/diagnosis"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/ratelimit"
	"github.com/sprucehealth/backend/libs/sig"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/libs/stripe"
	"github.com/sprucehealth/backend/media"
	"github.com/sprucehealth/backend/www"
	"github.com/sprucehealth/backend/www/router"
)

func buildWWW(
	conf *Config,
	dataAPI api.DataAPI,
	authAPI api.AuthAPI,
	diagnosisAPI diagnosis.API,
	smsAPI api.SMSAPI,
	eRxAPI erx.ERxAPI,
	dispatcher *dispatch.Dispatcher,
	signer *sig.Signer,
	stores storage.StoreMap,
	rateLimiters ratelimit.KeyedRateLimiters,
	alog analytics.Logger,
	compressResponse bool,
	metricsRegistry metrics.Registry,
	onboardingURLExpires int64,
) http.Handler {
	stripeCli := &stripe.StripeService{
		SecretKey:      conf.Stripe.SecretKey,
		PublishableKey: conf.Stripe.PublishableKey,
	}

	templateLoader := www.NewTemplateLoader(func(path string) (io.ReadCloser, error) {
		return resources.DefaultBundle.Open("templates/" + path)
	})

	var err error
	var analyticsDB *sql.DB
	if conf.AnalyticsDB.Host != "" {
		analyticsDB, err = conf.AnalyticsDB.ConnectPostgres()
		if err != nil {
			log.Fatal(err)
		}
	} else {
		golog.Warningf("No analytics database configured")
	}

	var lc *librato.Client
	if conf.Stats.LibratoToken != "" && conf.Stats.LibratoUsername != "" {
		lc = &librato.Client{
			Username: conf.Stats.LibratoUsername,
			Token:    conf.Stats.LibratoToken,
		}
	}

	return router.New(&router.Config{
		DataAPI:              dataAPI,
		AuthAPI:              authAPI,
		DiagnosisAPI:         diagnosisAPI,
		SMSAPI:               smsAPI,
		ERxAPI:               eRxAPI,
		Dispatcher:           dispatcher,
		AnalyticsDB:          analyticsDB,
		AnalyticsLogger:      alog,
		FromNumber:           conf.Twilio.FromNumber,
		EmailService:         email.NewService(dataAPI, conf.Email, metricsRegistry.Scope("email")),
		SupportEmail:         conf.Support.CustomerSupportEmail,
		WebDomain:            conf.WebDomain,
		StaticResourceURL:    conf.StaticResourceURL,
		StripeClient:         stripeCli,
		Signer:               signer,
		Stores:               stores,
		MediaStore:           media.NewStore("https://"+conf.APIDomain+apipaths.MediaURLPath, signer, stores.MustGet("media")),
		RateLimiters:         rateLimiters,
		WebPassword:          conf.WebPassword,
		TemplateLoader:       templateLoader,
		OnboardingURLExpires: onboardingURLExpires,
		TwoFactorExpiration:  conf.TwoFactorExpiration,
		ExperimentIDs:        conf.ExperimentID,
		LibratoClient:        lc,
		CompressResponse:     compressResponse,
		MetricsRegistry:      metricsRegistry.Scope("www"),
	})
}
