package dronboard

import (
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/dispatch"
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

	doctorRoles := []string{api.RoleDoctor}

	// If logged in as the doctor then jump to first step rather than registration
	registerHandler := www.AuthRequiredHandler(config.AuthAPI, doctorRoles,
		http.RedirectHandler("/doctor-register/saved-message", http.StatusSeeOther),
		NewIntroHandler(r, config.Signer, config.TemplateLoader))
	r.Handle("/doctor-register", registerHandler).Name("doctor-register-intro")

	// If logged in as the doctor then jump to first step rather than registration
	registerHandler = www.AuthRequiredHandler(config.AuthAPI, doctorRoles,
		http.RedirectHandler("/doctor-register/saved-message", http.StatusSeeOther),
		NewRegisterHandler(r, config.DataAPI, config.AuthAPI, config.Dispatcher, config.Signer, config.TemplateLoader))
	r.Handle("/doctor-register/account", registerHandler).Name("doctor-register-account")

	authFilter := www.AuthRequiredFilter(config.AuthAPI, doctorRoles, nil)

	r.Handle("/doctor-register/cell-verify", authFilter(NewCellVerifyHandler(r, config.DataAPI, config.AuthAPI, config.SMSAPI, config.SMSFromNumber, config.TemplateLoader))).Name("doctor-register-cell-verify")
	r.Handle("/doctor-register/credentials", authFilter(NewCredentialsHandler(r, config.DataAPI, config.TemplateLoader))).Name("doctor-register-credentials")
	r.Handle("/doctor-register/upload-cv", authFilter(NewUploadCVHandler(r, config.DataAPI, config.Stores.MustGet("onboarding"), config.TemplateLoader))).Name("doctor-register-upload-cv")
	r.Handle("/doctor-register/upload-license", authFilter(NewUploadLicenseHandler(r, config.DataAPI, config.Stores["onboarding"], config.TemplateLoader))).Name("doctor-register-upload-license")
	r.Handle("/doctor-register/upload-claims-history", authFilter(NewUploadClaimsHistoryHandler(r, config.DataAPI, config.Stores.MustGet("onboarding"), config.TemplateLoader))).Name("doctor-register-upload-claims-history")
	r.Handle("/doctor-register/claims-history", authFilter(NewClaimsHistoryHandler(r, config.DataAPI, config.Stores.MustGet("onboarding"), config.TemplateLoader))).Name("doctor-register-claims-history")
	r.Handle("/doctor-register/insurance", authFilter(NewInsuranceHandler(r, config.DataAPI, config.TemplateLoader))).Name("doctor-register-insurance")
	r.Handle("/doctor-register/financials", authFilter(NewFinancialsHandler(r, config.DataAPI, config.StripeClient, config.TemplateLoader))).Name("doctor-register-financials")
	r.Handle("/doctor-register/success", authFilter(NewSuccessHandler(r, config.DataAPI, config.TemplateLoader))).Name("doctor-register-success")
	r.Handle("/doctor-register/financials-verify", authFilter(NewFinancialVerifyHandler(r, config.DataAPI, config.SupportEmail, config.StripeClient, config.TemplateLoader))).Name("doctor-register-financials-verify")
	r.Handle("/doctor-register/malpractice-faq", authFilter(NewStaticTemplateHandler(
		config.TemplateLoader.MustLoadTemplate("dronboard/malpracticefaq.html", "dronboard/base.html", nil),
		&www.BaseTemplateContext{Title: "Malpractice FAQ | Spruce"}))).Name("doctor-register-malpractice-faq")
	r.Handle("/doctor-register/background-check", authFilter(NewBackgroundCheckHandler(r, config.DataAPI, config.TemplateLoader))).Name("doctor-register-background-check")
}
