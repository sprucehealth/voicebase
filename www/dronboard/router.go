package dronboard

import (
	"log"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/payment/stripe"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/www"
)

func SetupRoutes(r *mux.Router, dataAPI api.DataAPI, authAPI api.AuthAPI, supportEmail string, stripeCli *stripe.StripeService, signer *common.Signer, stores map[string]storage.Store, metricsRegistry metrics.Registry) {
	if stores["onboarding"] == nil {
		log.Fatal("onboarding storage not configured")
	}

	doctorRoles := []string{api.DOCTOR_ROLE}

	// If logged in as the doctor then jump to first step rather than registration
	registerHandler := www.AuthRequiredHandler(authAPI, doctorRoles,
		http.RedirectHandler("/doctor-register/credentials", http.StatusSeeOther),
		NewRegisterHandler(r, dataAPI, authAPI, signer))
	r.Handle("/doctor-register", registerHandler).Name("doctor-register")

	authFilter := www.AuthRequiredFilter(authAPI, doctorRoles, nil)

	r.Handle("/doctor-register/credentials", authFilter(NewCredentialsHandler(r, dataAPI))).Name("doctor-register-credentials")
	r.Handle("/doctor-register/upload-cv", authFilter(NewUploadCVHandler(r, dataAPI, stores["onboarding"]))).Name("doctor-register-upload-cv")
	r.Handle("/doctor-register/upload-license", authFilter(NewUploadLicenseHandler(r, dataAPI, stores["onboarding"]))).Name("doctor-register-upload-license")
	r.Handle("/doctor-register/upload-claims-history", authFilter(NewUploadClaimsHistory(r, dataAPI, stores["onboarding"]))).Name("doctor-register-upload-claims-history")
	r.Handle("/doctor-register/engagement", authFilter(NewEngagementHandler(r, dataAPI))).Name("doctor-register-engagement")
	r.Handle("/doctor-register/insurance", authFilter(NewInsuranceHandler(r, dataAPI))).Name("doctor-register-insurance")
	r.Handle("/doctor-register/financials", authFilter(NewFinancialsHandler(r, dataAPI, stripeCli))).Name("doctor-register-financials")
	r.Handle("/doctor-register/success", authFilter(NewSuccessHandler(r, dataAPI))).Name("doctor-register-success")
	r.Handle("/doctor-register/financials-verify", authFilter(NewFinancialVerifyHandler(r, dataAPI, supportEmail, stripeCli))).Name("doctor-register-financials-verify")
	r.Handle("/doctor-register/malpractice-faq", authFilter(NewStaticTemplateHandler(malpracticeFAQTemplate, &www.BaseTemplateContext{Title: "Malpractice FAQ | Spruce"}))).Name("doctor-register-malpractice-faq")
}
