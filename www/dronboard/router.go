package dronboard

import (
	"log"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/libs/stripe"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/www"
)

func SetupRoutes(r *mux.Router, dataAPI api.DataAPI, authAPI api.AuthAPI, supportEmail string, stripeCli *stripe.StripeService, signer *common.Signer, stores map[string]storage.Store, templateLoader *www.TemplateLoader, metricsRegistry metrics.Registry) {
	if stores["onboarding"] == nil {
		log.Fatal("onboarding storage not configured")
	}

	templateLoader.MustLoadTemplate("dronboard/base.html", "base.html", nil)

	doctorRoles := []string{api.DOCTOR_ROLE}

	// If logged in as the doctor then jump to first step rather than registration
	registerHandler := www.AuthRequiredHandler(authAPI, doctorRoles,
		http.RedirectHandler("/doctor-register/credentials", http.StatusSeeOther),
		NewIntroHandler(r, signer, templateLoader))
	r.Handle("/doctor-register", registerHandler).Name("doctor-register-intro")

	// If logged in as the doctor then jump to first step rather than registration
	registerHandler = www.AuthRequiredHandler(authAPI, doctorRoles,
		http.RedirectHandler("/doctor-register/credentials", http.StatusSeeOther),
		NewRegisterHandler(r, dataAPI, authAPI, signer, templateLoader))
	r.Handle("/doctor-register/account", registerHandler).Name("doctor-register-account")

	authFilter := www.AuthRequiredFilter(authAPI, doctorRoles, nil)

	r.Handle("/doctor-register/credentials", authFilter(NewCredentialsHandler(r, dataAPI, templateLoader))).Name("doctor-register-credentials")
	r.Handle("/doctor-register/upload-cv", authFilter(NewUploadCVHandler(r, dataAPI, stores["onboarding"], templateLoader))).Name("doctor-register-upload-cv")
	r.Handle("/doctor-register/upload-license", authFilter(NewUploadLicenseHandler(r, dataAPI, stores["onboarding"], templateLoader))).Name("doctor-register-upload-license")
	r.Handle("/doctor-register/upload-claims-history", authFilter(NewUploadClaimsHistoryHandler(r, dataAPI, stores["onboarding"], templateLoader))).Name("doctor-register-upload-claims-history")
	r.Handle("/doctor-register/claims-history", authFilter(NewClaimsHistoryHandler(r, dataAPI, stores["onboarding"], templateLoader))).Name("doctor-register-claims-history")
	r.Handle("/doctor-register/insurance", authFilter(NewInsuranceHandler(r, dataAPI, templateLoader))).Name("doctor-register-insurance")
	r.Handle("/doctor-register/financials", authFilter(NewFinancialsHandler(r, dataAPI, stripeCli, templateLoader))).Name("doctor-register-financials")
	r.Handle("/doctor-register/success", authFilter(NewSuccessHandler(r, dataAPI, templateLoader))).Name("doctor-register-success")
	r.Handle("/doctor-register/financials-verify", authFilter(NewFinancialVerifyHandler(r, dataAPI, supportEmail, stripeCli, templateLoader))).Name("doctor-register-financials-verify")
	r.Handle("/doctor-register/malpractice-faq", authFilter(NewStaticTemplateHandler(
		templateLoader.MustLoadTemplate("dronboard/malpracticefaq.html", "dronboard/base.html", nil),
		&www.BaseTemplateContext{Title: "Malpractice FAQ | Spruce"}))).Name("doctor-register-malpractice-faq")
	r.Handle("/doctor-register/background-check", authFilter(NewBackgroundCheckHandler(r, dataAPI, templateLoader))).Name("doctor-register-background-check")
}
