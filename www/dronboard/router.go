package dronboard

import (
	"net/http"

	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/libs/sig"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/libs/stripe"
	"github.com/sprucehealth/backend/www"
)

type Config struct {
	DataAPI         api.DataAPI
	AuthAPI         api.AuthAPI
	SMSAPI          api.SMSAPI
	Dispatcher      *dispatch.Dispatcher
	SMSFromNumber   string
	SupportEmail    string
	StripeClient    *stripe.Client
	Signer          *sig.Signer
	Stores          storage.StoreMap
	TemplateLoader  *www.TemplateLoader
	MetricsRegistry metrics.Registry
}

// This has been made into a function pointer so that it can be overriden in testing primarily to a no-op
var SetupRoutes = func(r *mux.Router, config *Config) {
	config.TemplateLoader.MustLoadTemplate("dronboard/base.html", "base.html", nil)

	savedMessageRedirect := httputil.RedirectHandler("/doctor-register/saved-message", http.StatusSeeOther)

	// If logged in as the doctor then jump to first step rather than registration
	h := newIntroHandler(r, config.Signer, config.TemplateLoader)
	registerHandler := www.AuthRequiredHandler(
		www.RoleRequiredHandler(savedMessageRedirect, h, api.RoleDoctor), h, config.AuthAPI)
	r.Handle("/doctor-register", registerHandler).Name("doctor-register-intro")

	// If logged in as the doctor then jump to first step rather than registration
	h = newRegisterHandler(r, config.DataAPI, config.AuthAPI, config.Dispatcher, config.Signer, config.TemplateLoader)
	registerHandler = www.AuthRequiredHandler(
		www.RoleRequiredHandler(savedMessageRedirect, h, api.RoleDoctor), h, config.AuthAPI)
	r.Handle("/doctor-register/account", registerHandler).Name("doctor-register-account")

	authFilter := func(h httputil.ContextHandler) httputil.ContextHandler {
		return www.AuthRequiredHandler(www.RoleRequiredHandler(h, nil, api.RoleDoctor), nil, config.AuthAPI)
	}

	r.Handle("/doctor-register/cell-verify", authFilter(newCellVerifyHandler(r, config.DataAPI, config.AuthAPI, config.SMSAPI, config.SMSFromNumber, config.TemplateLoader))).Name("doctor-register-cell-verify")
	r.Handle("/doctor-register/credentials", authFilter(newCredentialsHandler(r, config.DataAPI, config.TemplateLoader))).Name("doctor-register-credentials")
	r.Handle("/doctor-register/upload-cv", authFilter(newUploadCVHandler(r, config.DataAPI, config.Stores.MustGet("onboarding"), config.TemplateLoader))).Name("doctor-register-upload-cv")
	r.Handle("/doctor-register/upload-license", authFilter(newUploadLicenseHandler(r, config.DataAPI, config.Stores["onboarding"], config.TemplateLoader))).Name("doctor-register-upload-license")
	r.Handle("/doctor-register/upload-claims-history", authFilter(newUploadClaimsHistoryHandler(r, config.DataAPI, config.Stores.MustGet("onboarding"), config.TemplateLoader))).Name("doctor-register-upload-claims-history")
	r.Handle("/doctor-register/claims-history", authFilter(newClaimsHistoryHandler(r, config.DataAPI, config.Stores.MustGet("onboarding"), config.TemplateLoader))).Name("doctor-register-claims-history")
	r.Handle("/doctor-register/insurance", authFilter(newInsuranceHandler(r, config.DataAPI, config.TemplateLoader))).Name("doctor-register-insurance")
	r.Handle("/doctor-register/financials", authFilter(newFinancialsHandler(r, config.DataAPI, config.StripeClient, config.TemplateLoader))).Name("doctor-register-financials")
	r.Handle("/doctor-register/success", authFilter(newSuccessHandler(r, config.DataAPI, config.TemplateLoader))).Name("doctor-register-success")
	r.Handle("/doctor-register/financials-verify", authFilter(newFinancialVerifyHandler(r, config.DataAPI, config.SupportEmail, config.StripeClient, config.TemplateLoader))).Name("doctor-register-financials-verify")
	r.Handle("/doctor-register/malpractice-faq", authFilter(newStaticTemplateHandler(
		config.TemplateLoader.MustLoadTemplate("dronboard/malpracticefaq.html", "dronboard/base.html", nil),
		&www.BaseTemplateContext{Title: "Malpractice FAQ | Spruce"}))).Name("doctor-register-malpractice-faq")
	r.Handle("/doctor-register/background-check", authFilter(newBackgroundCheckHandler(r, config.DataAPI, config.TemplateLoader))).Name("doctor-register-background-check")
}
